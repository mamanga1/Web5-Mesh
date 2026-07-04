package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
)

var (
	registry   = make(map[string]*net.UDPAddr)
	lastClient = make(map[string]*net.UDPAddr)
	mu         sync.RWMutex
)

// maskAddr oculta los últimos octetos de la IP y el puerto por privacidad
func maskAddr(addr *net.UDPAddr) string {
	if addr == nil {
		return "desconocido"
	}
	ip := addr.IP
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%d.%d.*.*", ip4[0], ip4[1])
	}
	// Fallback para IPv6 o formatos no estándar
	return ip.String()[:8] + "..."
}

func main() {
	port := "54321"
	addr, _ := net.ResolveUDPAddr("udp", "0.0.0.0:"+port)
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Error al escuchar: %v", err)
	}
	defer conn.Close()

	fmt.Printf("🛡️ [FARO] Relay Ciego en 0.0.0.0:%s (Logs con IPs truncadas)\n", port)

	buf := make([]byte, 4096)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		msg := strings.TrimSpace(string(buf[:n]))
		parts := strings.SplitN(msg, " ", 4)
		if len(parts) < 2 {
			continue
		}

		cmd := parts[0]

		switch cmd {
		case "ANNOUNCE":
			if len(parts) >= 2 {
				did := parts[1]
				mu.Lock()
				registry[did] = remoteAddr
				mu.Unlock()
				fmt.Printf("[FARO] 📥 ANNOUNCE: %s desde %s\n", did, maskAddr(remoteAddr))
			}

		case "RELAY":
			if len(parts) == 4 {
				targetDID := parts[1]
				senderDID := parts[2]
				payload := parts[3]

				mu.Lock()
				lastClient[senderDID] = remoteAddr
				mu.Unlock()
				fmt.Printf("[FARO] 📥 RELAY: guardado lastClient[%s] = %s\n", senderDID, maskAddr(remoteAddr))

				mu.RLock()
				target, exists := registry[targetDID]
				mu.RUnlock()

				if exists {
					fmt.Printf("[FARO] 📤 RELAY: reenviando a %s (%s)\n", targetDID, maskAddr(target))
					conn.WriteToUDP([]byte(payload), target)
				} else {
					fmt.Printf("[FARO] ❌ RELAY: target %s NO ENCONTRADO\n", targetDID)
				}
			}

		case "RESPONSE":
			if len(parts) >= 3 {
				targetDID := parts[1]
				payload := parts[2]
				fmt.Printf("[FARO] 📥 RESPONSE: recibido para %s\n", targetDID)

				mu.RLock()
				client, exists := lastClient[targetDID]
				mu.RUnlock()

				if exists {
					fmt.Printf("[FARO] 📤 RESPONSE: reenviando a %s\n", maskAddr(client))
					conn.WriteToUDP([]byte(payload), client)
				} else {
					fmt.Printf("[FARO] ❌ RESPONSE: lastClient NO ENCONTRADO para %s\n", targetDID)
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
		}
	}
}
