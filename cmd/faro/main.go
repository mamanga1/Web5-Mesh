package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mr-tron/base58"
	"web5-mesh/src/crypto"
)

// ============================================================================
// REGISTRY CON SHARDING (escalabilidad: reduce contención de mutex 16x)
// ============================================================================

const numShards = 16

type registryEntry struct {
	addr     *net.UDPAddr
	lastSeen time.Time
	endpoint string // "ip:port" público (para signaling)
}

type shard struct {
	mu      sync.RWMutex
	entries map[string]*registryEntry
}

var shards [numShards]*shard

func init() {
	for i := range shards {
		shards[i] = &shard{entries: make(map[string]*registryEntry)}
	}
}

func getShard(did string) *shard {
	h := fnv.New32a()
	h.Write([]byte(did))
	return shards[h.Sum32()%numShards]
}

func registrySet(did string, entry *registryEntry) {
	s := getShard(did)
	s.mu.Lock()
	s.entries[did] = entry
	s.mu.Unlock()
}

func registryGet(did string) (*registryEntry, bool) {
	s := getShard(did)
	s.mu.RLock()
	e, ok := s.entries[did]
	s.mu.RUnlock()
	return e, ok
}

func registryDelete(did string) {
	s := getShard(did)
	s.mu.Lock()
	delete(s.entries, did)
	s.mu.Unlock()
}

// ============================================================================
// REGISTRY WSS
// ============================================================================

var (
	wsRegistry   = make(map[string]*websocket.Conn)
	wsLastClient = make(map[string]*websocket.Conn)
	wsMu         sync.RWMutex
)

// ============================================================================
// SESIONES ACTIVAS (signaling)
// ============================================================================
// Cuando dos nodos establecen una sesión directa, el faro lo registra acá.
// Mientras la sesión está activa, el faro NO relayea entre esos dos nodos
// (se hablan directo). Si la sesión se cierra o expira, el faro vuelve a
// relayear (fallback).

type sessionEntry struct {
	didA     string
	didB     string
	created  time.Time
	lastSeen time.Time
	active   bool // true = sesión directa establecida, faro no relayea
}

var (
	sessionMu sync.RWMutex
	sessions  = make(map[string]*sessionEntry) // clave: "didA|didB" (ordenado)
)

func sessionKey(didA, didB string) string {
	if didA > didB {
		didA, didB = didB, didA
	}
	return didA + "|" + didB
}

// ============================================================================
// REFERENCIA GLOBAL A UDPConn (para cross-relay WSS→UDP)
// ============================================================================

var (
	globalUDPConns = make(map[string]*net.UDPConn)
	udpConnsMu     sync.RWMutex
)

// ============================================================================
// GATE DID
// ============================================================================

var faroGate = crypto.NewGate(500, 2*time.Hour)

var (
	gateDIDs   = make(map[string]string)
	gateDIDsMu sync.RWMutex
)

// ============================================================================
// RATE LIMITER (anti-flood: máximo 20 mensajes por 5s por IP)
// ============================================================================

type rateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

var limiter = &rateLimiter{hits: make(map[string][]time.Time)}

func (rl *rateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-5 * time.Second)
	// Limpiar entradas viejas
	recent := rl.hits[key][:0]
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 20 {
		rl.hits[key] = recent
		return false
	}
	rl.hits[key] = append(recent, now)
	return true
}

// ============================================================================
// MÉTRICAS
// ============================================================================

var (
	statsNodes     int64
	statsRelays    int64
	statsSessions  int64
	statsDropped   int64
	statsStartTime = time.Now()
)

// ============================================================================
// UTILIDADES
// ============================================================================

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	buildCommit  string
	buildTime    string
	buildVersion string
)

func maskAddr(addr *net.UDPAddr) string {
	if addr == nil {
		return "desconocido"
	}
	ip := addr.IP
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%d.%d.*.*", ip4[0], ip4[1])
	}
	return ip.String()[:8] + "..."
}

func maskRemoteAddr(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "***"
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return "***"
	}
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%d.%d.*.*", ip4[0], ip4[1])
	}
	return host[:8] + "..."
}

func stripPadding(data string) string {
	if idx := strings.LastIndex(data, "|"); idx != -1 {
		return data[:idx]
	}
	return data
}

func isHandshake(data []byte) bool {
	return bytes.Contains(data, []byte(`"did"`)) &&
		bytes.Contains(data, []byte(`"sig"`)) &&
		bytes.Contains(data, []byte(`"nonce"`))
}

func verifyAnnounceSig(did, ts, sig string) bool {
	const prefix = "did:maia:"
	if !strings.HasPrefix(did, prefix) {
		return false
	}
	pubBytes, err := base58.Decode(strings.TrimPrefix(did, prefix))
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}
	sigBytes, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(pubBytes), []byte(ts), sigBytes)
}

// ============================================================================
// HANDLER UDP
// ============================================================================

func handleUDPMessage(conn *net.UDPConn, data []byte, remoteAddr *net.UDPAddr) {
	// Rate limiting
	if !limiter.Allow(remoteAddr.String()) {
		atomic.AddInt64(&statsDropped, 1)
		return
	}

	// Handshake del Gate DID
	if isHandshake(data) {
		did, err := faroGate.VerifyHandshake(data, remoteAddr.String())
		if err != nil {
			return
		}
		gateDIDsMu.Lock()
		gateDIDs[remoteAddr.String()] = did
		gateDIDsMu.Unlock()

		ack := fmt.Sprintf(`{"ack":"ok","did":"%s","ts":%d,"nodes":%d}`,
			did, time.Now().Unix(), faroGate.Count())
		conn.WriteToUDP([]byte(ack), remoteAddr)
		fmt.Printf("[FARO-UDP] 🔑 Gate: %s autorizado desde %s\n", did[:20]+"...", maskAddr(remoteAddr))
		return
	}

	// Verificar Gate
	if !faroGate.IsAllowed(remoteAddr.String()) {
		gateDIDsMu.RLock()
		knownDID := gateDIDs[remoteAddr.String()]
		gateDIDsMu.RUnlock()
		if knownDID != "" {
			fmt.Printf("[FARO-UDP] ⚠️ Gate rechazó %s (DID: %s) — IP cambió\n",
				maskAddr(remoteAddr), knownDID[:20]+"...")
		}
		return
	}

	msg := strings.TrimSpace(string(data))

	if strings.HasPrefix(msg, "{") {
		handleJSONCommand(conn, msg, remoteAddr)
		return
	}

	parts := strings.SplitN(msg, " ", 5)
	if len(parts) < 2 {
		return
	}
	cmd := parts[0]

	switch cmd {

	// ====================================================================
	// ANNOUNCE — registro en el registry (con TTL)
	// ====================================================================
	case "ANNOUNCE":
		if len(parts) == 4 {
			did := parts[1]
			ts := parts[2]
			sig := stripPadding(parts[3])
			if !verifyAnnounceSig(did, ts, sig) {
				return
			}
			registrySet(did, &registryEntry{
				addr:     remoteAddr,
				lastSeen: time.Now(),
				endpoint: remoteAddr.String(),
			})
			atomic.StoreInt64(&statsNodes, int64(faroGate.Count()))
			fmt.Printf("[FARO-UDP] 📥 ANNOUNCE: %s desde %s\n", did[:20]+"...", maskAddr(remoteAddr))
			ack := fmt.Sprintf("ACK_IP %s", remoteAddr.IP.String())
			conn.WriteToUDP([]byte(ack), remoteAddr)
		}

	// ====================================================================
	// OPEN_SESSION — signaling: "quiero hablar con B"
	// ====================================================================
	// El faro verifica que ambos están registrados y le devuelve a A
	// los endpoints públicos de B (para hole punching).
	// También le avisa a B que A quiere hablar (para que B empiece a
	// escuchar paquetes de A).
	case "OPEN_SESSION":
		if len(parts) >= 3 {
			targetDID := stripPadding(parts[1])
			senderDID := stripPadding(parts[2])

			// Verificar que ambos están registrados
			senderEntry, senderExists := registryGet(senderDID)
			targetEntry, targetExists := registryGet(targetDID)

			if !senderExists || !targetExists {
				errMsg := fmt.Sprintf("SESSION_ERROR %s: peer not registered", targetDID)
				conn.WriteToUDP([]byte(errMsg), remoteAddr)
				return
			}

			// Crear sesión
			key := sessionKey(senderDID, targetDID)
			sessionMu.Lock()
			sessions[key] = &sessionEntry{
				didA:     senderDID,
				didB:     targetDID,
				created:  time.Now(),
				lastSeen: time.Now(),
				active:   false, // Todavía no se estableció la conexión directa
			}
			sessionMu.Unlock()
			atomic.AddInt64(&statsSessions, 1)

			// Devolver a A los endpoints de B
			sessionInfo := fmt.Sprintf("SESSION_INFO %s %s %s",
				targetDID, targetEntry.endpoint, senderEntry.endpoint)
			conn.WriteToUDP([]byte(sessionInfo), remoteAddr)

			// Avisar a B que A quiere hablar (para que B empiece el punch)
			punchNotify := fmt.Sprintf("SESSION_INCOMING %s %s",
				senderDID, senderEntry.endpoint)
			conn.WriteToUDP([]byte(punchNotify), targetEntry.addr)

			fmt.Printf("[FARO] 🔗 OPEN_SESSION: %s → %s\n",
				senderDID[:15]+"...", targetDID[:15]+"...")
		}

	// ====================================================================
	// PUNCH — hole punching: "A está en ip:port, mandale un paquete"
	// ====================================================================
	// El faro le dice al target que mande un paquete UDP a la dirección
	// pública del sender. Ambos nodos hacen esto simultáneamente, lo que
	// abre el mapeo del NAT en ambos lados.
	case "PUNCH":
		if len(parts) >= 4 {
			targetDID := stripPadding(parts[1])
			senderDID := stripPadding(parts[2])
			senderEndpoint := stripPadding(parts[3])

			targetEntry, exists := registryGet(targetDID)
			if !exists {
				return
			}

			// Decirle al target: "mandale un paquete a sender en endpoint"
			punchCmd := fmt.Sprintf("PUNCH_NOW %s %s", senderDID, senderEndpoint)
			conn.WriteToUDP([]byte(punchCmd), targetEntry.addr)

			fmt.Printf("[FARO] 👊 PUNCH: %s → %s (endpoint: %s)\n",
				senderDID[:15]+"...", targetDID[:15]+"...", maskRemoteAddr(senderEndpoint))
		}

	// ====================================================================
	// SESSION_ACK — "la sesión directa se estableció"
	// ====================================================================
	// Cuando ambos nodos confirman que el hole punching funcionó,
	// el faro marca la sesión como activa y DEJA de relayear entre
	// esos dos nodos (se hablan directo).
	case "SESSION_ACK":
		if len(parts) >= 3 {
			targetDID := stripPadding(parts[1])
			senderDID := stripPadding(parts[2])

			key := sessionKey(senderDID, targetDID)
			sessionMu.Lock()
			if s, ok := sessions[key]; ok {
				s.active = true
				s.lastSeen = time.Now()
			}
			sessionMu.Unlock()

			fmt.Printf("[FARO] ✅ SESSION_ACK: %s ↔ %s (directo activo)\n",
				senderDID[:15]+"...", targetDID[:15]+"...")
		}

	// ====================================================================
	// CLOSE_SESSION — "cerrar sesión directa"
	// ====================================================================
	case "CLOSE_SESSION":
		if len(parts) >= 3 {
			targetDID := stripPadding(parts[1])
			senderDID := stripPadding(parts[2])

			key := sessionKey(senderDID, targetDID)
			sessionMu.Lock()
			delete(sessions, key)
			sessionMu.Unlock()

			fmt.Printf("[FARO] 🔒 CLOSE_SESSION: %s ↔ %s\n",
				senderDID[:15]+"...", targetDID[:15]+"...")
		}

	// ====================================================================
	// RELAY — fallback: relayea si no hay sesión directa activa
	// ====================================================================
	case "RELAY":
		if len(parts) == 4 {
			targetDID := parts[1]
			senderDID := parts[2]
			payload := parts[3]

			// Verificar si hay sesión directa activa
			key := sessionKey(senderDID, targetDID)
			sessionMu.RLock()
			s, hasSession := sessions[key]
			sessionActive := hasSession && s.active
			sessionMu.RUnlock()

			if sessionActive {
				// Sesión directa activa: NO relayear, decirle al sender
				// que use la conexión directa.
				redirect := fmt.Sprintf("SESSION_REDIRECT %s", targetDID)
				conn.WriteToUDP([]byte(redirect), remoteAddr)
				return
			}

			// Fallback: relayear como antes
			atomic.AddInt64(&statsRelays, 1)

			// Buscar en registry UDP
			targetEntry, existsUDP := registryGet(targetDID)
			if existsUDP {
				conn.WriteToUDP([]byte(payload), targetEntry.addr)
				ackMsg := fmt.Sprintf("ACK %s %s", senderDID, targetDID)
				conn.WriteToUDP([]byte(ackMsg), remoteAddr)
				return
			}

			// Buscar en registry WSS
			wsMu.RLock()
			targetWS, existsWS := wsRegistry[targetDID]
			wsMu.RUnlock()
			if existsWS {
				if err := targetWS.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
					wsMu.Lock()
					if wsRegistry[targetDID] == targetWS {
						delete(wsRegistry, targetDID)
					}
					wsMu.Unlock()
					errorMsg := fmt.Sprintf("ERROR %s: delivery failed", targetDID)
					conn.WriteToUDP([]byte(errorMsg), remoteAddr)
					return
				}
				ackMsg := fmt.Sprintf("ACK %s %s", senderDID, targetDID)
				conn.WriteToUDP([]byte(ackMsg), remoteAddr)
				return
			}

			errorMsg := fmt.Sprintf("ERROR %s: target not found", targetDID)
			conn.WriteToUDP([]byte(errorMsg), remoteAddr)
		}

	// ====================================================================
	// WHERE_IS — presencia (busca en ambos registries)
	// ====================================================================
	case "WHERE_IS":
		if len(parts) >= 2 {
			did := stripPadding(parts[1])
			_, existsUDP := registryGet(did)
			wsMu.RLock()
			_, existsWS := wsRegistry[did]
			wsMu.RUnlock()
			if existsUDP || existsWS {
				conn.WriteToUDP([]byte("READY"), remoteAddr)
			} else {
				conn.WriteToUDP([]byte("NOT_FOUND"), remoteAddr)
			}
		}

	// ====================================================================
	// VERIFY_HASH
	// ====================================================================
	case "VERIFY_HASH":
		handleVerifyHash(conn, remoteAddr)

	// ====================================================================
	// STATS — métricas del faro
	// ====================================================================
	case "STATS":
		resp := fmt.Sprintf(
			`{"nodes":%d,"relays":%d,"sessions":%d,"dropped":%d,"uptime_s":%d,"version":"%s","commit":"%s"}`,
			atomic.LoadInt64(&statsNodes),
			atomic.LoadInt64(&statsRelays),
			atomic.LoadInt64(&statsSessions),
			atomic.LoadInt64(&statsDropped),
			int(time.Since(statsStartTime).Seconds()),
			buildVersion, buildCommit)
		conn.WriteToUDP([]byte(resp), remoteAddr)
	}
}

func handleJSONCommand(conn *net.UDPConn, msg string, addr *net.UDPAddr) {
	if strings.Contains(msg, `"cmd":"VERIFY_HASH"`) || strings.Contains(msg, `"cmd": "VERIFY_HASH"`) {
		handleVerifyHash(conn, addr)
		return
	}
	conn.WriteToUDP([]byte(`{"error":"comando JSON no reconocido"}`), addr)
}

func handleVerifyHash(conn *net.UDPConn, addr *net.UDPAddr) {
	exe, err := os.Executable()
	if err != nil {
		conn.WriteToUDP([]byte(`{"error":"no puedo leerme"}`), addr)
		return
	}
	f, err := os.Open(exe)
	if err != nil {
		conn.WriteToUDP([]byte(`{"error":"`+err.Error()+`"}`), addr)
		return
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	hash := hex.EncodeToString(h.Sum(nil))
	info, _ := os.Stat(exe)
	resp := fmt.Sprintf(
		`{"hash":"%s","size":%d,"commit":"%s","built":"%s","version":"%s"}`,
		hash, info.Size(), buildCommit, buildTime, buildVersion)
	conn.WriteToUDP([]byte(resp), addr)
}

// ============================================================================
// HANDLER WEBSOCKET (con signaling)
// ============================================================================

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[FARO-WS] ⚠️ Recuperado de panic: %v", r)
		}
	}()

	// Límite de conexiones WSS
	wsMu.RLock()
	wsCount := len(wsRegistry)
	wsMu.RUnlock()
	if wsCount >= 200 {
		http.Error(w, "too many connections", 503)
		return
	}

	hs := crypto.Handshake{
		DID:   r.Header.Get("X-Xionia-DID"),
		Pub:   r.Header.Get("X-Xionia-Pub"),
		Nonce: r.Header.Get("X-Xionia-Nonce"),
		Sig:   r.Header.Get("X-Xionia-Sig"),
	}
	fmt.Sscanf(r.Header.Get("X-Xionia-TS"), "%d", &hs.TS)

	if hs.DID == "" || hs.Sig == "" {
		http.Error(w, "", 403)
		return
	}

	hsJSON, _ := json.Marshal(hs)
	gateDID, err := faroGate.VerifyHandshake(hsJSON, r.RemoteAddr)
	if err != nil {
		http.Error(w, "", 403)
		return
	}

	gateDIDsMu.Lock()
	gateDIDs[r.RemoteAddr] = gateDID
	gateDIDsMu.Unlock()

	fmt.Printf("[FARO-WS] 🔑 Gate: %s autorizado desde %s\n", gateDID[:20]+"...", maskRemoteAddr(r.RemoteAddr))

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var myDID string
	defer func() {
		if myDID == "" {
			return
		}
		wsMu.Lock()
		if wsRegistry[myDID] == conn {
			delete(wsRegistry, myDID)
		}
		if wsLastClient[myDID] == conn {
			delete(wsLastClient, myDID)
		}
		wsMu.Unlock()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		msg := strings.TrimSpace(string(message))
		parts := strings.SplitN(msg, " ", 5)
		if len(parts) < 2 {
			continue
		}
		cmd := parts[0]

		switch cmd {
		case "ANNOUNCE":
			if len(parts) == 4 {
				did := parts[1]
				ts := parts[2]
				sig := stripPadding(parts[3])
				if !verifyAnnounceSig(did, ts, sig) {
					continue
				}
				myDID = did
				wsMu.Lock()
				wsRegistry[did] = conn
				wsMu.Unlock()
				fmt.Printf("[FARO-WS] 📥 ANNOUNCE: %s\n", did[:20]+"...")
			}

		case "OPEN_SESSION":
			if len(parts) >= 3 {
				targetDID := stripPadding(parts[1])
				senderDID := stripPadding(parts[2])

				_, targetExists := registryGet(targetDID)
				wsMu.RLock()
				_, targetWSExists := wsRegistry[targetDID]
				wsMu.RUnlock()

				if !targetExists && !targetWSExists {
					conn.WriteMessage(websocket.TextMessage,
						[]byte(fmt.Sprintf("SESSION_ERROR %s: peer not registered", targetDID)))
					continue
				}

				key := sessionKey(senderDID, targetDID)
				sessionMu.Lock()
				sessions[key] = &sessionEntry{
					didA: senderDID, didB: targetDID,
					created: time.Now(), lastSeen: time.Now(), active: false,
				}
				sessionMu.Unlock()

				// Devolver endpoints del target
				var targetEndpoint string
				if entry, ok := registryGet(targetDID); ok {
					targetEndpoint = entry.endpoint
				}
				sessionInfo := fmt.Sprintf("SESSION_INFO %s %s", targetDID, targetEndpoint)
				conn.WriteMessage(websocket.TextMessage, []byte(sessionInfo))

				// Avisar al target
				if targetEntry, ok := registryGet(targetDID); ok {
					udpConnsMu.RLock()
					udpConn := globalUDPConns["54321"]
					udpConnsMu.RUnlock()
					if udpConn != nil {
						notify := fmt.Sprintf("SESSION_INCOMING %s ws", senderDID)
						udpConn.WriteToUDP([]byte(notify), targetEntry.addr)
					}
				}
				wsMu.RLock()
				if targetWS, ok := wsRegistry[targetDID]; ok {
					notify := fmt.Sprintf("SESSION_INCOMING %s ws", senderDID)
					targetWS.WriteMessage(websocket.TextMessage, []byte(notify))
				}
				wsMu.RUnlock()
			}

		case "SESSION_ACK":
			if len(parts) >= 3 {
				targetDID := stripPadding(parts[1])
				senderDID := stripPadding(parts[2])
				key := sessionKey(senderDID, targetDID)
				sessionMu.Lock()
				if s, ok := sessions[key]; ok {
					s.active = true
					s.lastSeen = time.Now()
				}
				sessionMu.Unlock()
			}

		case "CLOSE_SESSION":
			if len(parts) >= 3 {
				targetDID := stripPadding(parts[1])
				senderDID := stripPadding(parts[2])
				key := sessionKey(senderDID, targetDID)
				sessionMu.Lock()
				delete(sessions, key)
				sessionMu.Unlock()
			}

		case "RELAY":
			if len(parts) == 4 {
				targetDID := parts[1]
				senderDID := parts[2]
				payload := parts[3]

				// Verificar sesión directa
				key := sessionKey(senderDID, targetDID)
				sessionMu.RLock()
				s, hasSession := sessions[key]
				sessionActive := hasSession && s.active
				sessionMu.RUnlock()

				if sessionActive {
					conn.WriteMessage(websocket.TextMessage,
						[]byte(fmt.Sprintf("SESSION_REDIRECT %s", targetDID)))
					continue
				}

				// Fallback relay
				atomic.AddInt64(&statsRelays, 1)

				wsMu.RLock()
				targetWS, existsWS := wsRegistry[targetDID]
				wsMu.RUnlock()
				if existsWS {
					if err := targetWS.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
						wsMu.Lock()
						if wsRegistry[targetDID] == targetWS {
							delete(wsRegistry, targetDID)
						}
						wsMu.Unlock()
						conn.WriteMessage(websocket.TextMessage,
							[]byte(fmt.Sprintf("ERROR %s: delivery failed", targetDID)))
						continue
					}
					conn.WriteMessage(websocket.TextMessage,
						[]byte(fmt.Sprintf("ACK %s %s", senderDID, targetDID)))
					continue
				}

				// Buscar en UDP
				if entry, ok := registryGet(targetDID); ok {
					udpConnsMu.RLock()
					udpConn := globalUDPConns["54321"]
					udpConnsMu.RUnlock()
					if udpConn != nil {
						udpConn.WriteToUDP([]byte(payload), entry.addr)
						conn.WriteMessage(websocket.TextMessage,
							[]byte(fmt.Sprintf("ACK %s %s", senderDID, targetDID)))
						continue
					}
				}

				conn.WriteMessage(websocket.TextMessage,
					[]byte(fmt.Sprintf("ERROR %s: target not found", targetDID)))
			}

		case "RESPONSE":
			if len(parts) >= 3 {
				targetDID := parts[1]
				payload := parts[2]
				wsMu.RLock()
				client, exists := wsLastClient[targetDID]
				wsMu.RUnlock()
				if exists {
					client.WriteMessage(websocket.TextMessage, []byte(payload))
				}
			}

		case "WHERE_IS":
			if len(parts) >= 2 {
				did := stripPadding(parts[1])
				_, existsUDP := registryGet(did)
				wsMu.RLock()
				_, existsWS := wsRegistry[did]
				wsMu.RUnlock()
				if existsUDP || existsWS {
					conn.WriteMessage(websocket.TextMessage, []byte("READY"))
				} else {
					conn.WriteMessage(websocket.TextMessage, []byte("NOT_FOUND"))
				}
			}

		case "VERIFY_HASH":
			exe, err := os.Executable()
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"no puedo leerme"}`))
				continue
			}
			f, err := os.Open(exe)
			if err != nil {
				conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"open fail"}`))
				continue
			}
			h := sha256.New()
			io.Copy(h, f)
			f.Close()
			hash := hex.EncodeToString(h.Sum(nil))
			info, _ := os.Stat(exe)
			resp := fmt.Sprintf(`{"hash":"%s","size":%d,"commit":"%s","built":"%s","version":"%s"}`,
				hash, info.Size(), buildCommit, buildTime, buildVersion)
			conn.WriteMessage(websocket.TextMessage, []byte(resp))

		case "STATS":
			resp := fmt.Sprintf(
				`{"nodes":%d,"relays":%d,"sessions":%d,"dropped":%d,"uptime_s":%d,"version":"%s"}`,
				atomic.LoadInt64(&statsNodes), atomic.LoadInt64(&statsRelays),
				atomic.LoadInt64(&statsSessions), atomic.LoadInt64(&statsDropped),
				int(time.Since(statsStartTime).Seconds()), buildVersion)
			conn.WriteMessage(websocket.TextMessage, []byte(resp))
		}
	}
}

// ============================================================================
// SERVIDORES
// ============================================================================

// Worker pool para UDP (escalabilidad: 8 workers en vez de 1 hilo)
type packet struct {
	data []byte
	addr *net.UDPAddr
}

func startUDPServer(port string) {
	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatalf("Error resolviendo UDP %s: %v", port, err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Error al escuchar UDP %s: %v", port, err)
	}
	defer conn.Close()

	udpConnsMu.Lock()
	globalUDPConns[port] = conn
	udpConnsMu.Unlock()

	fmt.Printf("🛡️ [FARO-UDP] Relay + Signaling en 0.0.0.0:%s (Gate DID activo)\n", port)

	const numWorkers = 8
	packetChan := make(chan packet, 2048)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for pkt := range packetChan {
				handleUDPMessage(conn, pkt.data, pkt.addr)
			}
		}()
	}

	buf := make([]byte, 65536) // Buffer máximo UDP
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		select {
		case packetChan <- packet{data: data, addr: remoteAddr}:
		default:
			atomic.AddInt64(&statsDropped, 1) // Canal lleno, descartar
		}
	}
}

func startWebSocketServer() {
	port := "443"
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Printf("[FARO-WS] ❌ Error cargando certificados: %v", err)
		log.Printf("[FARO-WS] El Faro sigue sirviendo UDP igual.")
		return
	}
	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleWebSocket)
	server := &http.Server{
		Addr:      ":" + port,
		Handler:   mux,
		TLSConfig: tlsConfig,
		ErrorLog:  log.New(io.Discard, "", 0),
	}
	fmt.Printf("🛡️ [FARO-WS] WebSocket TLS en 0.0.0.0:%s (Gate DID activo)\n", port)
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Printf("[FARO-WS] ❌ Error: %v", err)
	}
}

// ============================================================================
// LIMPIEZA: registry TTL + sesiones expiradas
// ============================================================================

func startCleaner() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		// Limpiar registry UDP (90s sin ANNOUNCE)
		expired := 0
		for i := range shards {
			shards[i].mu.Lock()
			for did, entry := range shards[i].entries {
				if time.Since(entry.lastSeen) > 90*time.Second {
					delete(shards[i].entries, did)
					expired++
				}
			}
			shards[i].mu.Unlock()
		}

		// Limpiar sesiones expiradas (5 min sin actividad)
		sessionMu.Lock()
		for key, s := range sessions {
			if time.Since(s.lastSeen) > 5*time.Minute {
				delete(sessions, key)
			}
		}
		sessionMu.Unlock()

		if expired > 0 {
			fmt.Printf("[FARO] 🗑️ Cleaner: %d entrada(s) expirada(s)\n", expired)
		}
	}
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	fmt.Println("🚀 Iniciando Faro XionIA (Signaling + Relay Fallback)")
	fmt.Println("   Gate DID: autenticación Ed25519 obligatoria")
	fmt.Println("   Signaling: OPEN_SESSION / PUNCH / SESSION_ACK")
	fmt.Println("   Relay: fallback si el hole punching falla")
	fmt.Println("   Cross-relay: UDP↔WSS activo")
	fmt.Println("   Registry: sharded (16 shards) + TTL 90s")
	fmt.Println("   Rate limiting: 20 msg/5s por IP")
	fmt.Printf("   Versión: %s (%s)\n", buildVersion, buildCommit)

	go startCleaner()
	go startUDPServer("54321")
	go startUDPServer("443")
	go startWebSocketServer()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan
	fmt.Println("\n🛑 Apagando faro...")
	fmt.Println("👋 Faro apagado limpiamente")
}
