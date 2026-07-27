package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json" // ← PARCHES: agregado para Gate DID
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
	"web5-mesh/src/crypto" // ← PARCHE: agregado para Gate DID
)

var (
	registry   = make(map[string]*net.UDPAddr)
	lastClient = make(map[string]*net.UDPAddr)
	mu         sync.RWMutex

	wsRegistry   = make(map[string]*websocket.Conn)
	wsLastClient = make(map[string]*websocket.Conn)
	wsMu         sync.RWMutex

	// ← PARCHE: Gate DID
	faroGate = crypto.NewGate(500, 2*time.Hour)
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
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
	return ip.String()[:8] + "..."
}

// ← PARCHE: detectar handshake DID
func isHandshake(data []byte) bool {
	return bytes.Contains(data, []byte(`"did"`)) &&
		bytes.Contains(data, []byte(`"sig"`)) &&
		bytes.Contains(data, []byte(`"nonce"`))
}

func handleUDPMessage(conn *net.UDPConn, buf []byte, n int, remoteAddr *net.UDPAddr) {
	data := buf[:n]

	// ← PARCHE: Gate DID — handshake o IP autorizada
	if isHandshake(data) {
		did, err := faroGate.VerifyHandshake(data, remoteAddr.String())
		if err != nil {
			return // zero logs
		}
		ack := fmt.Sprintf(`{"ack":"ok","did":"%s","ts":%d,"nodes":%d}`,
			did, time.Now().Unix(), faroGate.Count())
		conn.WriteToUDP([]byte(ack), remoteAddr)
		fmt.Printf("[FARO-UDP] 🔑 Gate: %s autorizado desde %s\n", did[:20]+"...", maskAddr(remoteAddr))
		return
	}
	if !faroGate.IsAllowed(remoteAddr.String()) {
		return // zero logs
	}
	// ← FIN PARCHE

	msg := strings.TrimSpace(string(data))

	// === JSON ===
	if strings.HasPrefix(msg, "{") {
		handleJSONCommand(conn, msg, remoteAddr)
		return
	}

	// === TEXTO PLANO ===
	parts := strings.SplitN(msg, " ", 4)
	if len(parts) < 2 {
		return
	}

	cmd := parts[0]

	switch cmd {
	case "ANNOUNCE":
		if len(parts) >= 2 {
			did := parts[1]
			mu.Lock()
			registry[did] = remoteAddr
			mu.Unlock()
			fmt.Printf("[FARO-UDP] 📥 ANNOUNCE: %s desde %s\n", did, maskAddr(remoteAddr))

			ack := fmt.Sprintf("ACK_IP %s", remoteAddr.IP.String())
			conn.WriteToUDP([]byte(ack), remoteAddr)
		}

	case "RELAY":
		if len(parts) == 4 {
			targetDID := parts[1]
			senderDID := parts[2]
			payload := parts[3]

			mu.Lock()
			lastClient[senderDID] = remoteAddr
			mu.Unlock()

			ack := fmt.Sprintf("ACK_IP %s", remoteAddr.IP.String())
			conn.WriteToUDP([]byte(ack), remoteAddr)

			mu.RLock()
			target, exists := registry[targetDID]
			mu.RUnlock()

			if exists {
				conn.WriteToUDP([]byte(payload), target)
				ackMsg := fmt.Sprintf("ACK %s %s", senderDID, targetDID)
				conn.WriteToUDP([]byte(ackMsg), remoteAddr)
			} else {
				errorMsg := fmt.Sprintf("ERROR %s: target not found", targetDID)
				conn.WriteToUDP([]byte(errorMsg), remoteAddr)
			}
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
			did := parts[1]
			mu.RLock()
			_, exists := registry[did]
			mu.RUnlock()

			if exists {
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

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// ← PARCHE: Gate DID en headers HTTP
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
	fmt.Printf("[FARO-WS] 🔑 Gate: %s autorizado desde %s\n", gateDID[:20]+"...", r.RemoteAddr)
	// ← FIN PARCHE

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

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
			if len(parts) >= 2 {
				did := parts[1]
				wsMu.Lock()
				wsRegistry[did] = conn
				wsMu.Unlock()
				fmt.Printf("[FARO-WS] 📥 ANNOUNCE: %s\n", did)
			}

		case "RELAY":
			if len(parts) == 4 {
				targetDID := parts[1]
				senderDID := parts[2]
				payload := parts[3]

				wsMu.Lock()
				wsLastClient[senderDID] = conn
				wsMu.Unlock()

				wsMu.RLock()
				target, exists := wsRegistry[targetDID]
				wsMu.RUnlock()

				if exists {
					target.WriteMessage(websocket.TextMessage, []byte(payload))
					ackMsg := fmt.Sprintf("ACK %s %s", senderDID, targetDID)
					conn.WriteMessage(websocket.TextMessage, []byte(ackMsg))
				} else {
					errorMsg := fmt.Sprintf("ERROR %s: target not found", targetDID)
					conn.WriteMessage(websocket.TextMessage, []byte(errorMsg))
				}
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
		}
	}
}

func startUDPServer() {
	port := "443" // ← PARCHE: era "54321", ahora 443 default
	addr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:"+port)
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Error al escuchar UDP: %v", err)
	}
	defer conn.Close()

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
		log.Printf("[FARO-WS] ❌ Error cargando certificados: %v", err)
		log.Printf("[FARO-WS] Generá con: openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj \"/CN=faro.local\"")
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
		ErrorLog:  log.New(io.Discard, "", 0), // ← PARCHE: zero logs de bots TLS
	}

	fmt.Printf("🛡️ [FARO-WS] WebSocket TLS en 0.0.0.0:%s (Gate DID activo)\n", port)

	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Printf("[FARO-WS] ❌ Error: %v", err)
	}
}

func main() {
	fmt.Println("🚀 Iniciando Faro Dual (UDP + WebSocket)")
	fmt.Println("   Gate DID: solo nodos con did:maia válido")

	go startUDPServer()
	startWebSocketServer()
}
