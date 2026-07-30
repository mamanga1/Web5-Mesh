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
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mr-tron/base58"
	"web5-mesh/src/crypto"
)

// ========== REGISTRY UDP (con TTL — FIX #2) ==========

type registryEntry struct {
	addr     *net.UDPAddr
	lastSeen time.Time
}

var (
	registry   = make(map[string]*registryEntry)
	lastClient = make(map[string]*net.UDPAddr)
	mu         sync.RWMutex
)

// ========== REGISTRY WSS ==========

var (
	wsRegistry   = make(map[string]*websocket.Conn)
	wsLastClient = make(map[string]*websocket.Conn)
	wsMu         sync.RWMutex
)

// ========== REFERENCIA GLOBAL A UDPConn (para cross-relay WSS→UDP — FIX #1) ==========

var (
	globalUDPConns = make(map[string]*net.UDPConn)
	udpConnsMu     sync.RWMutex
)

// ========== GATE DID ==========

var faroGate = crypto.NewGate(500, 2*time.Hour)

var (
	gateDIDs   = make(map[string]string)
	gateDIDsMu sync.RWMutex
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	buildCommit  string
	buildTime    string
	buildVersion string
)

// ========== UTILIDADES ==========

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

// ← FIX #7: sacar el padding aleatorio que los nodos agregan con addPadding().
// Los nodos mandan "ANNOUNCE did ts sig|paddingRandom". Sin esto, la firma
// llega con el padding pegado, base64 falla, y el ANNOUNCE se descarta en
// silencio → los nodos se conectan (Gate OK) pero NUNCA se registran.
// SOLO se aplica a comandos de control (ANNOUNCE, WHERE_IS, etc.).
// NUNCA al RELAY (el payload se relayea completo con padding al destino).
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

// ========== FIX A : verificar firma del ANNOUNCE ==========

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

// ========== HANDLER UDP ==========

func handleUDPMessage(conn *net.UDPConn, buf []byte, n int, remoteAddr *net.UDPAddr) {
	data := buf[:n]

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
			fmt.Printf("[FARO-UDP] ⚠️ Gate rechazó %s (DID conocido: %s) — IP:puerto cambió, necesita nuevo handshake\n",
				maskAddr(remoteAddr), knownDID[:20]+"...")
		}
		return
	}

	msg := strings.TrimSpace(string(data))

	// Comando JSON
	if strings.HasPrefix(msg, "{") {
		handleJSONCommand(conn, msg, remoteAddr)
		return
	}

	parts := strings.SplitN(msg, " ", 4)
	if len(parts) < 2 {
		return
	}
	cmd := parts[0]

	switch cmd {
	case "ANNOUNCE":
		if len(parts) == 4 {
			did := parts[1]
			ts := parts[2]
			sig := stripPadding(parts[3]) // ← FIX #7: sacar padding antes de verificar firma
			if !verifyAnnounceSig(did, ts, sig) {
				return
			}
			mu.Lock()
			registry[did] = &registryEntry{addr: remoteAddr, lastSeen: time.Now()}
			mu.Unlock()
			fmt.Printf("[FARO-UDP] 📥 ANNOUNCE: %s desde %s\n", did, maskAddr(remoteAddr))
			ack := fmt.Sprintf("ACK_IP %s", remoteAddr.IP.String())
			conn.WriteToUDP([]byte(ack), remoteAddr)
		}

	case "RELAY":
		if len(parts) == 4 {
			targetDID := parts[1]
			senderDID := parts[2]
			payload := parts[3] // NO se le saca padding: se relayea completo al destino

			mu.Lock()
			lastClient[senderDID] = remoteAddr
			mu.Unlock()

			ack := fmt.Sprintf("ACK_IP %s", remoteAddr.IP.String())
			conn.WriteToUDP([]byte(ack), remoteAddr)

			// FIX #1: Cross-relay — buscar en registry UDP primero
			mu.RLock()
			entryUDP, existsUDP := registry[targetDID]
			mu.RUnlock()

			if existsUDP {
				conn.WriteToUDP([]byte(payload), entryUDP.addr)
				ackMsg := fmt.Sprintf("ACK %s %s", senderDID, targetDID)
				conn.WriteToUDP([]byte(ackMsg), remoteAddr)
				return
			}

			// FIX #1: Cross-relay — buscar en registry WSS
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

	case "RESPONSE":
		if len(parts) >= 3 {
			targetDID := parts[1]
			payload := parts[2]
			mu.RLock()
			client, exists := lastClient[targetDID]
			mu.RUnlock()
			if exists {
				conn.WriteToUDP([]byte(payload), client)
			}
		}

	case "WHERE_IS":
		if len(parts) >= 2 {
			did := stripPadding(parts[1]) // ← FIX #7
			mu.RLock()
			_, existsUDP := registry[did]
			mu.RUnlock()
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
	if _, err := io.Copy(h, f); err != nil {
		conn.WriteToUDP([]byte(`{"error":"hash fail"}`), addr)
		return
	}
	hash := hex.EncodeToString(h.Sum(nil))
	info, err := os.Stat(exe)
	if err != nil {
		conn.WriteToUDP([]byte(`{"error":"stat fail"}`), addr)
		return
	}
	resp := fmt.Sprintf(
		`{"hash":"%s","size":%d,"commit":"%s","built":"%s","version":"%s"}`,
		hash, info.Size(), buildCommit, buildTime, buildVersion,
	)
	conn.WriteToUDP([]byte(resp), addr)
}

// ========== HANDLER WEBSOCKET ==========

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
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

	// FIX C : limpiar wsRegistry/wsLastClient al desconectar
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
		parts := strings.SplitN(msg, " ", 4)
		if len(parts) < 2 {
			continue
		}
		cmd := parts[0]

		switch cmd {
		case "ANNOUNCE":
			if len(parts) == 4 {
				did := parts[1]
				ts := parts[2]
				sig := stripPadding(parts[3]) // ← FIX #7: sacar padding antes de verificar firma
				if !verifyAnnounceSig(did, ts, sig) {
					continue
				}
				myDID = did
				wsMu.Lock()
				wsRegistry[did] = conn
				wsMu.Unlock()
				fmt.Printf("[FARO-WS] 📥 ANNOUNCE: %s\n", did)
			}

		case "RELAY":
			if len(parts) == 4 {
				targetDID := parts[1]
				senderDID := parts[2]
				payload := parts[3] // NO se le saca padding

				wsMu.Lock()
				wsLastClient[senderDID] = conn
				wsMu.Unlock()

				// FIX #1: Cross-relay — buscar en wsRegistry WSS primero
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
						conn.WriteMessage(websocket.TextMessage, []byte(errorMsg))
						continue
					}
					ackMsg := fmt.Sprintf("ACK %s %s", senderDID, targetDID)
					conn.WriteMessage(websocket.TextMessage, []byte(ackMsg))
					continue
				}

				// FIX #1: Cross-relay — buscar en registry UDP
				mu.RLock()
				entryUDP, existsUDP := registry[targetDID]
				mu.RUnlock()

				if existsUDP {
					udpConnsMu.RLock()
					udpConn := globalUDPConns["54321"]
					udpConnsMu.RUnlock()
					if udpConn != nil {
						udpConn.WriteToUDP([]byte(payload), entryUDP.addr)
						ackMsg := fmt.Sprintf("ACK %s %s", senderDID, targetDID)
						conn.WriteMessage(websocket.TextMessage, []byte(ackMsg))
						continue
					}
				}

				errorMsg := fmt.Sprintf("ERROR %s: target not found", targetDID)
				conn.WriteMessage(websocket.TextMessage, []byte(errorMsg))
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
				did := stripPadding(parts[1]) // ← FIX #7
				wsMu.RLock()
				_, existsWS := wsRegistry[did]
				wsMu.RUnlock()
				mu.RLock()
				_, existsUDP := registry[did]
				mu.RUnlock()
				if existsWS || existsUDP {
					conn.WriteMessage(websocket.TextMessage, []byte("READY"))
				} else {
					conn.WriteMessage(websocket.TextMessage, []byte("NOT_FOUND"))
				}
			}
		}
	}
}

// ========== SERVIDORES ==========

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

	// FIX #1: guardar referencia global al UDPConn (para cross-relay WSS→UDP)
	udpConnsMu.Lock()
	globalUDPConns[port] = conn
	udpConnsMu.Unlock()

	fmt.Printf("🛡️ [FARO-UDP] Relay Ciego en 0.0.0.0:%s (Gate DID activo)\n", port)

	buf := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		handleUDPMessage(conn, buf, n, remoteAddr)
	}
}

func startWebSocketServer() {
	port := "443"
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		// FIX B : no matar el proceso si faltan certs
		log.Printf("[FARO-WS] ❌ Error cargando certificados: %v", err)
		log.Printf("[FARO-WS] Generá con: openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj \"/CN=faro.local\"")
		log.Printf("[FARO-WS] El Faro sigue sirviendo UDP igual — solo WSS queda caído hasta que arregles esto.")
		return
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
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

// ========== FIX #2: limpieza del registry UDP (TTL) ==========

func startRegistryCleaner() {
	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		mu.Lock()
		expired := 0
		for did, entry := range registry {
			if time.Since(entry.lastSeen) > 90*time.Second {
				delete(registry, did)
				expired++
			}
		}
		mu.Unlock()
		if expired > 0 {
			fmt.Printf("[FARO] 🗑️ Registry: %d entrada(s) expirada(s) (90s sin ANNOUNCE)\n", expired)
		}
	}
}

// ========== MAIN ==========

func main() {
	fmt.Println("🚀 Iniciando Faro Dual (UDP + WebSocket)")
	fmt.Println("   Gate DID: solo nodos con did:maia válido")
	fmt.Println("   Cross-relay: UDP↔WSS activo")
	fmt.Println("   Registry TTL: 90s sin ANNOUNCE → expira")
	fmt.Println("   Padding: stripPadding en ANNOUNCE/WHERE_IS activo")

	// FIX #2: goroutine de limpieza del registry
	go startRegistryCleaner()

	// FIX B : los 3 en goroutines separadas
	go startUDPServer("54321")
	go startUDPServer("443")
	go startWebSocketServer()

	select {} // bloquea para siempre
}
