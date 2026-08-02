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
// REGISTRY CON SHARDING
// ============================================================================

const numShards = 16

type registryEntry struct {
	addr     *net.UDPAddr
	lastSeen time.Time
	endpoint string
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
// SESIONES ACTIVAS
// ============================================================================

// FIX 8: límite de sesiones para prevenir DoS por sesiones fantasmas
const maxSessions = 10000

type sessionEntry struct {
	didA     string
	didB     string
	created  time.Time
	lastSeen time.Time
	active   bool
}

var (
	sessionMu sync.RWMutex
	sessions  = make(map[string]*sessionEntry)
)

func sessionKey(didA, didB string) string {
	if didA > didB {
		didA, didB = didB, didA
	}
	return didA + "|" + didB
}

// ============================================================================
// REFERENCIA GLOBAL A UDPConn
// ============================================================================

var (
	globalUDPConns = make(map[string]*net.UDPConn)
	udpConnsMu     sync.RWMutex
)

// FIX 11: devolver la primera conexión UDP disponible
// en vez de hardcodear globalUDPConns["54321"]
func getPrimaryUDPConn() *net.UDPConn {
	udpConnsMu.RLock()
	defer udpConnsMu.RUnlock()
	for _, conn := range globalUDPConns {
		return conn
	}
	return nil
}

// ============================================================================
// GATE DID
// ============================================================================

var faroGate = crypto.NewGate(500, 2*time.Hour)

var (
	gateDIDs   = make(map[string]string)
	gateDIDsMu sync.RWMutex
)

// FIX  5/6: verificar que el DID reclamado coincide con el DID
// autenticado via Gate para esa dirección remota
func verifyGateDID(remoteAddr string, claimedDID string) bool {
	gateDIDsMu.RLock()
	gateDID := gateDIDs[remoteAddr]
	gateDIDsMu.RUnlock()
	return gateDID != "" && gateDID == claimedDID
}

// ============================================================================
// RATE LIMITER
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

// FIX 2: CheckOrigin restringido.
// Clientes nativos (Go, Android) no envían Origin → se permiten.
// Navegadores envían Origin → se verifica que coincida con el Host.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // cliente nativo, sin Origin
		}
		// Verificar que el host del Origin coincida con el Host de la request
		originHost := origin
		originHost = strings.TrimPrefix(originHost, "https://")
		originHost = strings.TrimPrefix(originHost, "http://")
		originHost = strings.TrimPrefix(originHost, "wss://")
		originHost = strings.TrimPrefix(originHost, "ws://")
		if idx := strings.Index(originHost, "/"); idx != -1 {
			originHost = originHost[:idx]
		}
		return originHost == r.Host
	},
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
	s := ip.String()
	if len(s) > 8 {
		return s[:8] + "..."
	}
	return s + "..."
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
	if len(host) > 8 {
		return host[:8] + "..."
	}
	return host + "..."
}

// FIX 16: stripPadding inteligente.
// Solo trunca si el sufijo después del último '|' es padding válido
// (exclusivamente caracteres alfanuméricos). Si contiene otros caracteres,
// es parte del payload legítimo y NO se trunca.
func truncDID(s string) string {
	if len(s) > 20 {
		return s[:20] + "..."
	}
	return s
}

func stripPadding(data string) string {
	idx := strings.LastIndex(data, "|")
	if idx == -1 {
		return data
	}
	suffix := data[idx+1:]
	if len(suffix) == 0 {
		return data
	}
	for _, c := range suffix {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return data // no es padding, no truncar
		}
	}
	return data[:idx]
}

// FIX 12: isHandshake más estricto.
// Verifica que empiece con '{' (JSON) además de contener los campos.
// Previene que paquetes binarios aleatorios consuman CPU en verificación.
func isHandshake(data []byte) bool {
	if len(data) < 10 || data[0] != '{' {
		return false
	}
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
	// FIX 3: rate limit por IP sola (sin puerto).
	// Antes usaba remoteAddr.String() que incluye el puerto,
	// permitiendo evasión cambiando de puerto fuente.
	if !limiter.Allow(remoteAddr.IP.String()) {
		atomic.AddInt64(&statsDropped, 1)
		return
	}

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
		fmt.Printf("[FARO-UDP] 🔑 Gate: %s autorizado desde %s\n", truncDID(did), maskAddr(remoteAddr))
		return
	}

	if !faroGate.IsAllowed(remoteAddr.String()) {
		gateDIDsMu.RLock()
		knownDID := gateDIDs[remoteAddr.String()]
		gateDIDsMu.RUnlock()
		if knownDID != "" {
			fmt.Printf("[FARO-UDP] ⚠️ Gate rechazó %s (DID: %s) — IP cambió\n",
				maskAddr(remoteAddr), truncDID(knownDID))
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
	case "ANNOUNCE":
		if len(parts) == 4 {
			did := parts[1]
			ts := parts[2]
			sig := stripPadding(parts[3])

			if !verifyAnnounceSig(did, ts, sig) {
				return
			}

			// FIX 5: verificar que el DID del ANNOUNCE coincide
			// con el DID autenticado via Gate para esta IP
			if !verifyGateDID(remoteAddr.String(), did) {
				fmt.Printf("[FARO-UDP] ⚠️ ANNOUNCE rechazado: DID %s no coincide con Gate para %s\n",
					truncDID(did), maskAddr(remoteAddr))
				return
			}

			registrySet(did, &registryEntry{
				addr:     remoteAddr,
				lastSeen: time.Now(),
				endpoint: remoteAddr.String(),
			})
			atomic.StoreInt64(&statsNodes, int64(faroGate.Count()))
			fmt.Printf("[FARO-UDP] 📥 ANNOUNCE: %s desde %s\n", truncDID(did), maskAddr(remoteAddr))

			ack := fmt.Sprintf("ACK_IP %s", remoteAddr.IP.String())
			conn.WriteToUDP([]byte(ack), remoteAddr)
		}

	case "OPEN_SESSION":
		if len(parts) >= 3 {
			targetDID := stripPadding(parts[1])
			senderDID := stripPadding(parts[2])

			// FIX 6: verificar senderDID contra Gate
			if !verifyGateDID(remoteAddr.String(), senderDID) {
				fmt.Printf("[FARO-UDP] ⚠️ OPEN_SESSION rechazado: senderDID %s no coincide con Gate\n",
					truncDID(senderDID))
				return
			}

			senderEntry, senderExists := registryGet(senderDID)
			targetEntry, targetExists := registryGet(targetDID)

			if !senderExists || !targetExists {
				errMsg := fmt.Sprintf("SESSION_ERROR %s: peer not registered", targetDID)
				conn.WriteToUDP([]byte(errMsg), remoteAddr)
				return
			}

			key := sessionKey(senderDID, targetDID)

			// FIX 8: límite de sesiones
			sessionMu.Lock()
			if len(sessions) >= maxSessions {
				sessionMu.Unlock()
				errMsg := fmt.Sprintf("SESSION_ERROR %s: too many sessions", targetDID)
				conn.WriteToUDP([]byte(errMsg), remoteAddr)
				return
			}
			sessions[key] = &sessionEntry{
				didA:     senderDID,
				didB:     targetDID,
				created:  time.Now(),
				lastSeen: time.Now(),
				active:   false,
			}
			sessionMu.Unlock()
			atomic.AddInt64(&statsSessions, 1)

			sessionInfo := fmt.Sprintf("SESSION_INFO %s %s %s",
				targetDID, targetEntry.endpoint, senderEntry.endpoint)
			conn.WriteToUDP([]byte(sessionInfo), remoteAddr)

			punchNotify := fmt.Sprintf("SESSION_INCOMING %s %s",
				senderDID, senderEntry.endpoint)
			conn.WriteToUDP([]byte(punchNotify), targetEntry.addr)

			fmt.Printf("[FARO] 🔗 OPEN_SESSION: %s → %s\n",
				truncDID(senderDID), truncDID(targetDID))
		}

	case "PUNCH":
		if len(parts) >= 3 {
			targetDID := stripPadding(parts[1])
			senderDID := stripPadding(parts[2])

			// FIX Kimi 6: verificar senderDID contra Gate
			if !verifyGateDID(remoteAddr.String(), senderDID) {
				fmt.Printf("[FARO-UDP] ⚠️ PUNCH rechazado: senderDID %s no coincide con Gate\n",
					truncDID(senderDID))
				return
			}

			targetEntry, exists := registryGet(targetDID)
			if !exists {
				return
			}
			senderEntry, senderExists := registryGet(senderDID)
			if !senderExists {
				return
			}

			punchCmd := fmt.Sprintf("PUNCH_NOW %s %s", senderDID, senderEntry.endpoint)
			conn.WriteToUDP([]byte(punchCmd), targetEntry.addr)
			fmt.Printf("[FARO] 👊 PUNCH: %s → %s (endpoint: %s)\n",
				truncDID(senderDID), truncDID(targetDID), maskRemoteAddr(senderEntry.endpoint))
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
			fmt.Printf("[FARO] ✅ SESSION_ACK: %s ↔ %s (directo activo)\n",
				truncDID(senderDID), truncDID(targetDID))
		}

	case "CLOSE_SESSION":
		if len(parts) >= 3 {
			targetDID := stripPadding(parts[1])
			senderDID := stripPadding(parts[2])

			key := sessionKey(senderDID, targetDID)
			sessionMu.Lock()
			delete(sessions, key)
			sessionMu.Unlock()
			fmt.Printf("[FARO] 🔒 CLOSE_SESSION: %s ↔ %s\n",
				truncDID(senderDID), truncDID(targetDID))
		}

	case "RELAY":
		if len(parts) == 4 {
			targetDID := parts[1]
			senderDID := parts[2]
			payload := parts[3]

			// FIX 6: verificar senderDID contra Gate
			if !verifyGateDID(remoteAddr.String(), senderDID) {
				fmt.Printf("[FARO-UDP] ⚠️ RELAY rechazado: senderDID %s no coincide con Gate\n",
					truncDID(senderDID))
				return
			}

			key := sessionKey(senderDID, targetDID)
			sessionMu.RLock()
			s, hasSession := sessions[key]
			sessionActive := hasSession && s.active
			sessionMu.RUnlock()

			if sessionActive {
				redirect := fmt.Sprintf("SESSION_REDIRECT %s", targetDID)
				conn.WriteToUDP([]byte(redirect), remoteAddr)
				return
			}

			atomic.AddInt64(&statsRelays, 1)

			targetEntry, existsUDP := registryGet(targetDID)
			if existsUDP {
				conn.WriteToUDP([]byte(payload), targetEntry.addr)
				ackMsg := fmt.Sprintf("ACK %s %s", senderDID, targetDID)
				conn.WriteToUDP([]byte(ackMsg), remoteAddr)
				return
			}

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

	case "VERIFY_HASH":
		handleVerifyHash(conn, remoteAddr)

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
// HANDLER WEBSOCKET
// ============================================================================

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[FARO-WS] ⚠️ Recuperado de panic: %v", r)
		}
	}()

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
	fmt.Printf("[FARO-WS] 🔑 Gate: %s autorizado desde %s\n", truncDID(gateDID), maskRemoteAddr(r.RemoteAddr))

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// FIX 7: límite de lectura para prevenir OOM
	conn.SetReadLimit(65536)

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

				// FIX Kimi 5: verificar DID contra Gate
				if !verifyGateDID(r.RemoteAddr, did) {
					fmt.Printf("[FARO-WS] ⚠️ ANNOUNCE rechazado: DID %s no coincide con Gate\n",
						truncDID(did))
					continue
				}

				myDID = did
				wsMu.Lock()
				wsRegistry[did] = conn
				// FIX Kimi 10: setear wsLastClient para que RESPONSE funcione
				wsLastClient[did] = conn
				wsMu.Unlock()
				fmt.Printf("[FARO-WS] 📥 ANNOUNCE: %s\n", truncDID(did))
			}

		case "OPEN_SESSION":
			if len(parts) >= 3 {
				targetDID := stripPadding(parts[1])
				senderDID := stripPadding(parts[2])

				// FIX Kimi 6: verificar senderDID contra Gate
				if !verifyGateDID(r.RemoteAddr, senderDID) {
					conn.WriteMessage(websocket.TextMessage,
						[]byte(fmt.Sprintf("SESSION_ERROR %s: sender not authenticated", targetDID)))
					continue
				}

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

				// FIX Kimi 8: límite de sesiones
				sessionMu.Lock()
				if len(sessions) >= maxSessions {
					sessionMu.Unlock()
					conn.WriteMessage(websocket.TextMessage,
						[]byte(fmt.Sprintf("SESSION_ERROR %s: too many sessions", targetDID)))
					continue
				}
				sessions[key] = &sessionEntry{
					didA: senderDID, didB: targetDID,
					created: time.Now(), lastSeen: time.Now(), active: false,
				}
				sessionMu.Unlock()

				var targetEndpoint string
				if entry, ok := registryGet(targetDID); ok {
					targetEndpoint = entry.endpoint
				}

				sessionInfo := fmt.Sprintf("SESSION_INFO %s %s", targetDID, targetEndpoint)
				conn.WriteMessage(websocket.TextMessage, []byte(sessionInfo))

				// FIX Kimi 11: usar getPrimaryUDPConn() en vez de hardcodear "54321"
				if targetEntry, ok := registryGet(targetDID); ok {
					udpConn := getPrimaryUDPConn()
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

				if !verifyGateDID(r.RemoteAddr, senderDID) {
					continue
				}

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

				if !verifyGateDID(r.RemoteAddr, senderDID) {
					continue
				}

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

				// FIX Kimi 6: verificar senderDID contra Gate
				if !verifyGateDID(r.RemoteAddr, senderDID) {
					conn.WriteMessage(websocket.TextMessage,
						[]byte(fmt.Sprintf("ERROR %s: sender not authenticated", targetDID)))
					continue
				}

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

				// FIX 11: usar getPrimaryUDPConn()
				if entry, ok := registryGet(targetDID); ok {
					udpConn := getPrimaryUDPConn()
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

type packet struct {
	data []byte
	addr *net.UDPAddr
}

func startUDPServer(port string) {
	addr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:"+port)
	if err != nil {
		log.Fatalf("Error resolviendo UDP %s: %v", port, err)
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		log.Fatalf("Error al escuchar UDP %s: %v", port, err)
	}
	defer conn.Close()

	udpConnsMu.Lock()
	globalUDPConns[port] = conn
	udpConnsMu.Unlock()

	fmt.Printf("🛡️ [FARO-UDP] Relay + Signaling en 0.0.0.0:%s (Gate DID activo)\n", port)

	const numWorkers = 8
	// FIX 9: buffer más grande (8192 vs 2048) para absorber ráfagas
	packetChan := make(chan packet, 8192)

	for i := 0; i < numWorkers; i++ {
		go func() {
			for pkt := range packetChan {
				func() {
					defer func() {
						if r := recover(); r != nil {
							fmt.Printf("[FARO-UDP] ⚠️ Panic recuperado: %v\n", r)
						}
					}()
					handleUDPMessage(conn, pkt.data, pkt.addr)
				}()
			}
		}()
	}

	buf := make([]byte, 65536)
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
			// FIX 9: loguear drops (cada 1000 para no spamear)
			dropped := atomic.AddInt64(&statsDropped, 1)
			if dropped%1000 == 0 {
				fmt.Printf("[FARO-UDP] ⚠️ %d paquetes descartados (buffer lleno)\n", dropped)
			}
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
// LIMPIEZA
// ============================================================================

func startCleaner() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
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

		sessionMu.Lock()
		for key, s := range sessions {
			if time.Since(s.lastSeen) > 5*time.Minute {
				delete(sessions, key)
			}
		}
		sessionMu.Unlock()

		gateDIDsMu.Lock()
		for addr := range gateDIDs {
			if !faroGate.IsAllowed(addr) {
				delete(gateDIDs, addr)
			}
		}
		gateDIDsMu.Unlock()
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	fmt.Println("\n🛑 Apagando faro...")
	fmt.Println("👋 Faro apagado limpiamente")
}
