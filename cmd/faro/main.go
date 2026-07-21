package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	registry   = make(map[string]*net.UDPAddr)
	lastClient = make(map[string]*net.UDPAddr)
	mu         sync.RWMutex

	wsRegistry   = make(map[string]*websocket.Conn)
	wsLastClient = make(map[string]*websocket.Conn)
	wsMu         sync.RWMutex
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Variables inyectadas en build
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

func handleUDPMessage(conn *net.UDPConn, buf []byte, n int, remoteAddr *net.UDPAddr) {
	msg := strings.TrimSpace(string(buf[:n]))

	// === DETECTAR JSON PRIMERO ===
	if strings.HasPrefix(msg, "{") {
		handleJSONCommand(conn, msg, remoteAddr)
		return
	}

	// === FORMATO TEXTO PLANO ===
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
			
			// === ACK_IP: Decirle al nodo su IP pública real ===
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
			fmt.Printf("[FARO-UDP] 📥 RELAY: guardado lastClient[%s] = %s\n", senderDID, maskAddr(remoteAddr))

			// === ACK_IP: Notificar al nodo su IP pública (por si cambió) ===
			ack := fmt.Sprintf("ACK_IP %s", remoteAddr.IP.String())
			conn.WriteToUDP([]byte(ack), remoteAddr)

			mu.RLock()
			target, exists := registry[targetDID]
			mu.RUnlock()

			if exists {
				fmt.Printf("[FARO-UDP] 📤 RELAY: reenviando a %s (%s)\n", targetDID, maskAddr(target))
				conn.WriteToUDP([]byte(payload), target)

				ackMsg := fmt.Sprintf("ACK %s %s", senderDID, targetDID)
				conn.WriteToUDP([]byte(ackMsg), remoteAddr)
				fmt.Printf("[FARO-UDP] ✅ ACK enviado a %s\n", senderDID)
			} else {
				fmt.Printf("[FARO-UDP] ❌ RELAY: target %s NO ENCONTRADO\n", targetDID)
				errorMsg := fmt.Sprintf("ERROR %s: target not found", targetDID)
				conn.WriteToUDP([]byte(errorMsg), remoteAddr)
			}
		}

	case "RESPONSE":
		if len(parts) >= 3 {
			targetDID := parts[1]
			payload := parts[2]
			fmt.Printf("[FARO-UDP] 📥 RESPONSE: recibido para %s\n", targetDID)

			mu.RLock()
			client, exists := lastClient[targetDID]
			mu.RUnlock()

			if exists {
				fmt.Printf("[FARO-UDP] 📤 RESPONSE: reenviando a %s\n", maskAddr(client))
				conn.WriteToUDP([]byte(payload), client)
			} else {
				fmt.Printf("[FARO-UDP] ❌ RESPONSE: lastClient NO ENCONTRADO para %s\n", targetDID)
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

// === NUEVA FUNCIÓN JSON ===
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[FARO-WS] Error upgrade: %v", err)
		return
	}
	defer conn.Close()

	fmt.Printf("[FARO-WS] 📡 Nueva conexión WebSocket desde %s\n", r.RemoteAddr)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("[FARO-WS] ❌ Conexión cerrada: %v\n", err)
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
				fmt.Printf("[FARO-WS] 📥 RELAY: guardado lastClient[%s]\n", senderDID)

				wsMu.RLock()
				target, exists := wsRegistry[targetDID]
				wsMu.RUnlock()

				if exists {
					fmt.Printf("[FARO-WS] 📤 RELAY: reenviando a %s\n", targetDID)
					target.WriteMessage(websocket.TextMessage, []byte(payload))
					ackMsg := fmt.Sprintf("ACK %s %s", senderDID, targetDID)
					conn.WriteMessage(websocket.TextMessage, []byte(ackMsg))
				} else {
					fmt.Printf("[FARO-WS] ❌ RELAY: target %s NO ENCONTRADO\n", targetDID)
					errorMsg := fmt.Sprintf("ERROR %s: target not found", targetDID)
					conn.WriteMessage(websocket.TextMessage, []byte(errorMsg))
				}
			}

		case "RESPONSE":
			if len(parts) >= 3 {
				targetDID := parts[1]
				payload := parts[2]
				fmt.Printf("[FARO-WS] 📥 RESPONSE: recibido para %s\n", targetDID)

				wsMu.RLock()
				client, exists := wsLastClient[targetDID]
				wsMu.RUnlock()

				if exists {
					fmt.Printf("[FARO-WS] 📤 RESPONSE: reenviando\n")
					client.WriteMessage(websocket.TextMessage, []byte(payload))
				} else {
					fmt.Printf("[FARO-WS] ❌ RESPONSE: lastClient NO ENCONTRADO para %s\n", targetDID)
				}
			}
		}
	}
}

func startUDPServer() {
	port := "54321"
	addr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:"+port)
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Error al escuchar UDP: %v", err)
	}
	defer conn.Close()

	fmt.Printf("🛡️ [FARO-UDP] Relay Ciego en 0.0.0.0:%s\n", port)

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
		log.Printf("[FARO-WS] Generá los certificados con: openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj \"/CN=faro.local\"")
		return
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	http.HandleFunc("/ws", handleWebSocket)

	server := &http.Server{
		Addr:      ":" + port,
		TLSConfig: tlsConfig,
	}

	fmt.Printf("🛡️ [FARO-WS] WebSocket TLS en 0.0.0.0:%s\n", port)

	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Printf("[FARO-WS] ❌ Error: %v", err)
	}
}

func main() {
	fmt.Println("🚀 Iniciando Faro Dual (UDP + WebSocket)")

	go startUDPServer()
	startWebSocketServer()
}
