package main

import (
	crypto_rand "crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/gorilla/websocket"
	"github.com/mr-tron/base58"
	"web5-mesh/cmd/mesh/commands"
	"web5-mesh/src/crypto"
	"web5-mesh/src/xtp"
)

func restoreTerminal() {
	fmt.Print("\x1b[?12l")
	fmt.Print("\x1b[?25h")
	fmt.Print("\x1b[?1049l")
	fmt.Print("\x1b[0m")
	fmt.Print("\x1b[2J")
	fmt.Print("\x1b[H")
	cmd := exec.Command("stty", "sane")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
}

func completer(d prompt.Document) []prompt.Suggest {
	text := d.TextBeforeCursor()
	words := strings.Fields(text)

	if len(words) == 0 {
		return []prompt.Suggest{
			{Text: "whoami", Description: "Mostrar tu identidad DID"},
			{Text: "acl", Description: "Gestión de nodos de confianza"},
			{Text: "alias", Description: "Gestión de alias locales"},
			{Text: "group", Description: "Gestión de grupos"},
			{Text: "faro", Description: "Gestión de faro"},
			{Text: "xtp", Description: "Transporte directo XTP (Noise IK)"},
			{Text: "debug", Description: "Debug on/off"},
			{Text: "help", Description: "Mostrar ayuda"},
			{Text: "exit", Description: "Salir de la shell"},
		}
	}

	switch words[0] {
	case "acl":
		if len(words) == 1 {
			return []prompt.Suggest{
				{Text: "add", Description: "Agregar nodo a ACL"},
				{Text: "import", Description: "Importar claves públicas"},
				{Text: "remove", Description: "Eliminar nodo de ACL"},
				{Text: "list", Description: "Listar nodos en ACL"},
				{Text: "clear", Description: "Limpiar ACL completa"},
			}
		}
	case "alias":
		if len(words) == 1 {
			return []prompt.Suggest{
				{Text: "add", Description: "Agregar alias"},
				{Text: "remove", Description: "Eliminar alias"},
				{Text: "list", Description: "Listar alias"},
			}
		}
	case "group":
		if len(words) == 1 {
			return []prompt.Suggest{
				{Text: "create", Description: "Crear nuevo grupo"},
				{Text: "list", Description: "Listar grupos"},
				{Text: "send", Description: "Enviar mensaje a grupo"},
				{Text: "add", Description: "Agregar miembro a grupo"},
				{Text: "remove", Description: "Eliminar miembro de grupo"},
				{Text: "invite", Description: "Invitar a grupo"},
				{Text: "kick", Description: "Expulsar de grupo"},
				{Text: "leave", Description: "Salir de grupo"},
				{Text: "delete", Description: "Eliminar grupo"},
				{Text: "info", Description: "Info de grupo"},
			}
		}
	case "xtp":
		if len(words) == 1 {
			return []prompt.Suggest{
				{Text: "status", Description: "Estado del transporte XTP"},
			}
		}
	}

	s := []prompt.Suggest{}
	allCommands := []string{
		"whoami", "acl", "alias", "group", "faro", "xtp", "help", "exit",
		"acl add", "acl import", "acl remove", "acl list", "acl clear",
		"alias add", "alias remove", "alias list",
		"group create", "group list", "group send", "group add", "group remove",
		"group invite", "group kick", "group leave", "group delete", "group info",
		"faro set", "faro reset",
		"xtp status",
	}
	for _, cmd := range allCommands {
		if strings.HasPrefix(cmd, text) {
			s = append(s, prompt.Suggest{Text: cmd})
		}
	}
	return s
}

var (
	connMu         sync.Mutex
	globalConn     *net.UDPConn
	globalConnWS   *websocket.Conn
	globalUseWS    bool
	globalACLIndex map[[4]byte]peerKeys
	globalID       *crypto.Identity
	globalFaroAddr string
	globalQuit     chan struct{}
	msgChan        = make(chan string, 100)
	cmdHistory     []string
	activeRecipient string
	msgHistory      map[string][]string
	lastPublicIP    string
)

var (
	activityMu   sync.Mutex
	lastActivity time.Time
)

func touchActivity() {
	activityMu.Lock()
	lastActivity = time.Now()
	activityMu.Unlock()
}

func staleSince() time.Duration {
	activityMu.Lock()
	defer activityMu.Unlock()
	if lastActivity.IsZero() {
		return 0
	}
	return time.Since(lastActivity)
}

var globalTM *xtp.TransportManager

type faroSenderShell struct{}

func (f faroSenderShell) SendToFaro(msg string) error {
	return sendToFaroShell(msg)
}

func buildXTPACLIndex() map[[4]byte]xtp.PeerKeys {
	idx := make(map[[4]byte]xtp.PeerKeys, len(globalACLIndex))
	for kid, pk := range globalACLIndex {
		idx[kid] = xtp.PeerKeys{
			DID:       pk.DID,
			PubKeyEd:  pk.PubKeyEd,
			PubKeyX:   pk.PubKeyX,
			SharedKey: pk.SharedKey,
		}
	}
	return idx
}

func isCommand(input string) bool {
	known := []string{
		"whoami", "acl", "alias", "group", "faro", "help", "exit",
		"clear", "import", "export", "sign", "verify", "ls", "cat",
		"mkdir", "rm", "pwd", "touch", "edit", "mv", "cp", "rmdir",
		"chat", "ia", "browse", "host", "sync", "proxy",
		"xtp",
	}
	words := strings.Fields(input)
	if len(words) == 0 {
		return false
	}
	first := strings.ToLower(words[0])
	for _, cmd := range known {
		if first == cmd || first == "/"+cmd {
			return true
		}
	}
	return false
}

func connectToFaroShell() error {
	if globalFaroAddr == "" {
		return fmt.Errorf("faro no configurado")
	}

	host := globalFaroAddr
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	if err := connectUDPShell(host + ":54321"); err == nil {
		return nil
	}
	if err := connectUDPShell(host + ":443"); err == nil {
		return nil
	}
	if err := connectWSShell(); err == nil {
		return nil
	}
	return fmt.Errorf("sin ruta al faro %s", globalFaroAddr)
}

func connectUDPShell(addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return fmt.Errorf("resolviendo UDP: %v", err)
	}
	conn, err := net.DialUDP("udp4", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("conectando UDP: %v", err)
	}

	hs, err := crypto.CreateHandshake(globalID)
	if err != nil {
		conn.Close()
		return err
	}
	conn.Write(hs)

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	ack := make([]byte, 1024)
	n, err := conn.Read(ack)
	if err != nil {
		conn.Close()
		return fmt.Errorf("timeout handshake UDP")
	}
	if !strings.Contains(string(ack[:n]), `"ack":"ok"`) {
		conn.Close()
		return fmt.Errorf("handshake UDP rechazado")
	}
	conn.SetReadDeadline(time.Time{})

	globalConn = conn
	globalUseWS = false
	return nil
}

func connectWSShell() error {
	wsHost := globalFaroAddr
	if !strings.Contains(wsHost, ":") {
		wsHost += ":443"
	}
	if strings.HasSuffix(wsHost, ":54321") {
		wsHost = strings.TrimSuffix(wsHost, ":54321") + ":443"
	}

	nonce := make([]byte, 32)
	crypto_rand.Read(nonce)
	ts := time.Now().Unix()
	nonceB64 := base64.StdEncoding.EncodeToString(nonce)
	msg := fmt.Sprintf("%s|%d|%s", globalID.DID, ts, nonceB64)
	sig := globalID.SignMessage([]byte(msg))

	headers := http.Header{}
	headers.Set("X-Xionia-DID", globalID.DID)
	headers.Set("X-Xionia-Pub", base58.Encode(globalID.PubKeyEd))
	headers.Set("X-Xionia-TS", fmt.Sprintf("%d", ts))
	headers.Set("X-Xionia-Nonce", nonceB64)
	headers.Set("X-Xionia-Sig", base64.StdEncoding.EncodeToString(sig))

	wsURL := fmt.Sprintf("wss://%s/ws", wsHost)

	// FIX 1: InsecureSkipVerify en false por defecto (anti-MITM).
	// Si el faro usa certs self-signed, setear XION_INSECURE_WS=1
	// como variable de entorno (solo para desarrollo/testing).
	insecureWS := os.Getenv("XION_INSECURE_WS") == "1"
	if insecureWS {
		fmt.Println("⚠️ [WS] XION_INSECURE_WS=1 — TLS sin verificar (solo desarrollo)")
	}

	dialer := websocket.Dialer{
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: insecureWS},
		HandshakeTimeout: 5 * time.Second,
	}

	wsConn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return fmt.Errorf("conectando WS: %v", err)
	}

	globalConnWS = wsConn
	globalUseWS = true
	return nil
}

func sendToFaroShell(msg string) error {
	if globalUseWS {
		if globalConnWS == nil {
			return fmt.Errorf("no hay conexión WebSocket al faro")
		}
		return globalConnWS.WriteMessage(websocket.TextMessage, []byte(msg))
	}
	if globalConn == nil {
		return fmt.Errorf("no hay conexión UDP al faro")
	}
	_, err := globalConn.Write([]byte(msg))
	return err
}

func readFromFaroShell() (string, error) {
	if globalUseWS {
		if globalConnWS == nil {
			return "", fmt.Errorf("sin conexión WS")
		}
		globalConnWS.SetReadDeadline(time.Now().Add(15 * time.Second))
		_, message, err := globalConnWS.ReadMessage()
		if err != nil {
			return "", err
		}
		return string(message), nil
	}
	if globalConn == nil {
		return "", fmt.Errorf("sin conexión UDP")
	}
	buf := make([]byte, 65536)
	globalConn.SetReadDeadline(time.Now().Add(15 * time.Second))
	n, _, err := globalConn.ReadFromUDP(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

func startNetworkListener() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("\n⚠️ Listener recuperado: %v — reconectando...\n", r)
			time.Sleep(2 * time.Second)
			go startNetworkListener()
		}
	}()

	if !globalUseWS && globalConn == nil {
		return
	}

	for {
		select {
		case <-globalQuit:
			return
		default:
			raw, err := readFromFaroShell()
			if err != nil {
				if !globalUseWS && globalConn != nil {
					globalConn.Close()
					globalConn = nil
				}
				if globalUseWS && globalConnWS != nil {
					globalConnWS.Close()
					globalConnWS = nil
				}
				time.Sleep(2 * time.Second)
				if err := connectToFaroShell(); err != nil {
					continue
				}
				continue
			}

			touchActivity()
			raw = stripPadding(raw)
			raw = extractPayload(raw)

			if globalTM != nil && globalTM.HandleIncoming(raw) {
				continue
			}

			if strings.HasPrefix(raw, "ACK_IP ") {
				parts := strings.SplitN(raw, " ", 2)
				if len(parts) == 2 {
					currentPublicIP := parts[1]
					if lastPublicIP != "" && lastPublicIP != currentPublicIP {
						fmt.Printf("🔄 IP pública cambió: %s → %s\n", lastPublicIP, currentPublicIP)
						ts := fmt.Sprintf("%d", time.Now().Unix())
						sig := base64.StdEncoding.EncodeToString(globalID.SignMessage([]byte(ts)))
						msg := fmt.Sprintf("ANNOUNCE %s %s %s", globalID.DID, ts, sig)
						sendToFaroShell(addPadding(msg))
					}
					lastPublicIP = currentPublicIP
				}
				continue
			}

			if strings.HasPrefix(raw, "ACK") {
				continue
			}

			parts := strings.SplitN(raw, "|", 2)
			if len(parts) != 2 {
				continue
			}
			kidBytes, err := hex.DecodeString(parts[0])
			if err != nil || len(kidBytes) != 4 {
				continue
			}
			var kid [4]byte
			copy(kid[:], kidBytes)

			peer, exists := globalACLIndex[kid]
			if !exists {
				continue
			}

			ciphertext, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				continue
			}
			plaintext, err := crypto.DecryptPayload(peer.SharedKey, ciphertext)
			if err != nil {
				continue
			}

			var inner InnerPayload
			if json.Unmarshal(plaintext, &inner) != nil {
				continue
			}

			innerForVerify := inner
			innerForVerify.Sig = ""
			verifyJSON, _ := json.Marshal(innerForVerify)
			sigBytes, _ := base64.StdEncoding.DecodeString(inner.Sig)
			if !crypto.VerifyMessage(peer.PubKeyEd, verifyJSON, sigBytes) {
				continue
			}

			if time.Now().Unix()-inner.TS > 60 {
				continue
			}

			displayName := crypto.ResolveDID(peer.DID)

			if strings.HasPrefix(inner.Cmd, "GROUP_SYNC:") {
				parts := strings.SplitN(inner.Cmd, ":", 3)
				if len(parts) == 3 {
					alias := parts[1]
					var group crypto.Group
					if json.Unmarshal([]byte(parts[2]), &group) == nil {
						// FIX Kimi 14: verificar que el sender sea el admin del grupo
						if group.Admin != inner.FromDID {
							msgChan <- fmt.Sprintf("⚠️ [SISTEMA] GROUP_SYNC rechazado: %s no es admin de '%s'", displayName, alias)
						} else {
							crypto.SaveGroupDirect(alias, &group)
							msgChan <- fmt.Sprintf("🔄 [SISTEMA] Grupo '%s' sincronizado (miembros: %d)", alias, len(group.Members))
						}
					}
				}
				continue
			}

			if strings.HasPrefix(inner.Cmd, "GROUP_DELETE:") {
				parts := strings.SplitN(inner.Cmd, ":", 2)
				if len(parts) == 2 {
					alias := parts[1]
					crypto.DeleteGroup(alias)
					msgChan <- fmt.Sprintf("🗑️ [SISTEMA] Grupo '%s' eliminado por el admin", alias)
				}
				continue
			}

			if strings.HasPrefix(inner.Cmd, "GROUP_KICKED:") {
				parts := strings.SplitN(inner.Cmd, ":", 2)
				if len(parts) == 2 {
					alias := parts[1]
					crypto.RemoveMember(alias, globalID.DID)
					msgChan <- fmt.Sprintf("👢 [SISTEMA] Fuiste expulsado del grupo '%s'", alias)
				}
				continue
			}

			if strings.HasPrefix(inner.Cmd, "GROUP_LEAVE:") {
				parts := strings.SplitN(inner.Cmd, ":", 3)
				if len(parts) == 3 {
					alias := parts[1]
					did := parts[2]
					crypto.RemoveMember(alias, did)
					msgChan <- fmt.Sprintf("👋 [SISTEMA] %s salió del grupo '%s'", displayName, alias)
				}
				continue
			}

			if strings.HasPrefix(inner.Cmd, "GROUP:") {
				parts := strings.SplitN(inner.Cmd, ":", 3)
				if len(parts) == 3 {
					msgChan <- fmt.Sprintf("💬 [GRUPO:%s] [%s]: %s", parts[1], displayName, parts[2])
					continue
				}
			}

			msgChan <- fmt.Sprintf("💬 [%s]: %s", displayName, inner.Cmd)
		}
	}
}

func startAnnounceLoop() {
	for globalConn == nil && globalConnWS == nil {
		select {
		case <-globalQuit:
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-globalQuit:
			return
		case <-ticker.C:
			if globalConn == nil && globalConnWS == nil {
				continue
			}
			ts := fmt.Sprintf("%d", time.Now().Unix())
			sig := base64.StdEncoding.EncodeToString(globalID.SignMessage([]byte(ts)))
			msg := fmt.Sprintf("ANNOUNCE %s %s %s", globalID.DID, ts, sig)
			if err := sendToFaroShell(addPadding(msg)); err == nil {
				touchActivity()
			}
		}
	}
}

func startWatchdog() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-globalQuit:
			return
		case <-ticker.C:
			if stale := staleSince(); stale > 20*time.Second {
				fmt.Printf("\n⚠️ [WATCHDOG] Sin actividad hace %v, reconectando...\n", stale)

				if !globalUseWS && globalConn != nil {
					globalConn.Close()
					globalConn = nil
				}
				if globalUseWS && globalConnWS != nil {
					globalConnWS.Close()
					globalConnWS = nil
				}

				if err := connectToFaroShell(); err == nil {
					ts := fmt.Sprintf("%d", time.Now().Unix())
					sig := base64.StdEncoding.EncodeToString(globalID.SignMessage([]byte(ts)))
					msg := fmt.Sprintf("ANNOUNCE %s %s %s", globalID.DID, ts, sig)
					sendToFaroShell(addPadding(msg))
					touchActivity()

					if globalTM != nil {
						globalTM.FSM().Send(xtp.EvFaroConnected, nil)
						globalTM.FSM().Send(xtp.EvAnnounceSent, nil)
					}
					fmt.Println("✅ [WATCHDOG] Reconectado y ANNOUNCE enviado")
				} else {
					fmt.Printf("❌ [WATCHDOG] Reconexión falló: %v\n", err)
				}
			}
		}
	}
}

func runInteractiveShell() {
	var err error
	defer restoreTerminal()

	globalID, err = crypto.LoadOrCreateIdentity()
	if err != nil {
		fmt.Printf("❌ Error de identidad: %v\n", err)
		return
	}
	crypto.SetSelfDID(globalID.DID)

	globalFaroAddr = getFaroAddr()
	globalQuit = make(chan struct{})

	if err := connectToFaroShell(); err != nil {
		fmt.Printf("⚠️ No se pudo conectar al faro: %v\n", err)
	}

	globalACLIndex, _ = buildACLIndex(globalID)

	globalTM = xtp.NewTransportManager(
		globalID,
		faroSenderShell{},
		buildXTPACLIndex(),
		xtp.ManagerCallbacks{
			OnMessage: func(peerDID, displayName, command string) {
				if strings.HasPrefix(command, "GROUP_SYNC:") {
					parts := strings.SplitN(command, ":", 3)
					if len(parts) == 3 {
						alias := parts[1]
						var group crypto.Group
						if json.Unmarshal([]byte(parts[2]), &group) == nil {
							// FIX Kimi 14: verificar que el sender sea el admin del grupo
							if group.Admin != peerDID {
								msgChan <- fmt.Sprintf("⚠️ [SISTEMA] GROUP_SYNC rechazado: %s no es admin de '%s'", displayName, alias)
							} else {
								crypto.SaveGroupDirect(alias, &group)
								msgChan <- fmt.Sprintf("🔄 [SISTEMA] Grupo '%s' sincronizado (miembros: %d)", alias, len(group.Members))
							}
						}
					}
					return
				}

				if strings.HasPrefix(command, "GROUP_DELETE:") {
					parts := strings.SplitN(command, ":", 2)
					if len(parts) == 2 {
						crypto.DeleteGroup(parts[1])
						msgChan <- fmt.Sprintf("🗑️ [SISTEMA] Grupo '%s' eliminado por el admin", parts[1])
					}
					return
				}

				if strings.HasPrefix(command, "GROUP_KICKED:") {
					parts := strings.SplitN(command, ":", 2)
					if len(parts) == 2 {
						crypto.RemoveMember(parts[1], globalID.DID)
						msgChan <- fmt.Sprintf("👢 [SISTEMA] Fuiste expulsado del grupo '%s'", parts[1])
					}
					return
				}

				if strings.HasPrefix(command, "GROUP_LEAVE:") {
					parts := strings.SplitN(command, ":", 3)
					if len(parts) == 3 {
						crypto.RemoveMember(parts[1], parts[2])
						msgChan <- fmt.Sprintf("👋 [SISTEMA] %s salió del grupo '%s'", displayName, parts[1])
					}
					return
				}

				if strings.HasPrefix(command, "GROUP:") {
					parts := strings.SplitN(command, ":", 3)
					if len(parts) == 3 {
						msgChan <- fmt.Sprintf("💬 [GRUPO:%s] [%s]: %s", parts[1], displayName, parts[2])
						return
					}
				}

				msgChan <- fmt.Sprintf("💬 [%s]: %s", displayName, command)
			},
			OnDirectSessionActive: func(peerDID string) {
				msgChan <- fmt.Sprintf("🔐 [XTP] Conexión directa con %s (Noise IK, forward secrecy)", peerDID[:20]+"...")
			},
			OnDirectSessionLost: func(peerDID string) {
				msgChan <- fmt.Sprintf("💀 [XTP] Sesión directa perdida con %s", peerDID[:20]+"...")
			},
			OnFallbackToRelay: func(peerDID string) {
				msgChan <- fmt.Sprintf("🔄 [XTP] Usando relay con %s (hole punching falló)", peerDID[:20]+"...")
			},
			OnError: func(context string, err error) {
				fmt.Printf("[XTP] ⚠️ Error en %s: %v\n", context, err)
			},
		},
		xtp.DefaultManagerConfig(),
	)

	if globalTM != nil {
		globalTM.FSM().Send(xtp.EvConnectFaro, nil)
		if err == nil {
			globalTM.FSM().Send(xtp.EvFaroConnected, nil)
			globalTM.FSM().Send(xtp.EvAnnounceSent, nil)
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(" XION KERNEL v1.0.0 | Modo Seguro: ON")
	fmt.Println(" 🆔 TU IDENTIDAD:")
	fmt.Println(" DID: " + globalID.DID)
	pubEdHex := hex.EncodeToString(globalID.PubKeyEd)
	pubXHex := hex.EncodeToString(globalID.PubKeyX[:])
	fmt.Println(" PubKey Ed: " + pubEdHex)
	fmt.Println(" PubKey X: " + pubXHex)
	fmt.Println()
	fmt.Println(" 📋 Para que otro nodo te agregue, decile que ejecute:")
	fmt.Printf(" acl import %s %s %s\n", globalID.DID, pubEdHex, pubXHex)
	fmt.Printf(" alias add %s\n", globalID.DID)
	fmt.Println()
	if globalFaroAddr != "" {
		if globalUseWS {
			fmt.Printf(" 📡 Faro activo: %s (WSS último recurso)\n", globalFaroAddr)
		} else {
			fmt.Printf(" 📡 Faro activo: %s (UDP)\n", globalFaroAddr)
		}
	} else {
		fmt.Println(" 📡 Faro: NO CONFIGURADO")
	}

	cmdHistory = []string{}
	msgHistory = make(map[string][]string)

	go startNetworkListener()
	go startAnnounceLoop()
	go startWatchdog()

	go func() {
		select {
		case <-globalQuit:
			return
		case <-time.After(3 * time.Second):
			ts := fmt.Sprintf("%d", time.Now().Unix())
			sig := base64.StdEncoding.EncodeToString(globalID.SignMessage([]byte(ts)))
			msg := fmt.Sprintf("ANNOUNCE %s %s %s", globalID.DID, ts, sig)
			if err := sendToFaroShell(addPadding(msg)); err == nil {
				touchActivity()
			}
		}
	}()

	go func() {
		for msg := range msgChan {
			fmt.Print("\r\x1b[K")
			fmt.Println(msg)
			fmt.Print("xion@nodo:~$ ")
		}
	}()

	for {
		input := prompt.Input("xion@nodo:~$ ", completer,
			prompt.OptionPrefixTextColor(prompt.Yellow),
			prompt.OptionInputTextColor(prompt.White),
			prompt.OptionSuggestionBGColor(prompt.DarkGray),
			prompt.OptionSuggestionTextColor(prompt.White),
			prompt.OptionSelectedSuggestionBGColor(prompt.Yellow),
			prompt.OptionSelectedSuggestionTextColor(prompt.Black),
			prompt.OptionHistory(cmdHistory),
		)

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if len(cmdHistory) == 0 || cmdHistory[len(cmdHistory)-1] != input {
			cmdHistory = append(cmdHistory, input)
		}

		if input == "debug on" {
			xtp.DebugMode = true
			fmt.Println("✅ Debug ON")
			continue
		}
		if input == "debug off" {
			xtp.DebugMode = false
			fmt.Println("✅ Debug OFF")
			continue
		}
		if input == "xtp" || input == "xtp status" {
			if globalTM == nil {
				fmt.Println("❌ TransportManager no inicializado")
				continue
			}
			stats := globalTM.Stats()
			fmt.Println(" 📊 XTP Transport Manager")
			fmt.Printf(" Estado FSM: %s\n", stats.FSMState)
			fmt.Printf(" Sesiones directas: %d\n", stats.DirectSessions)
			fmt.Printf(" Relay activo: %v\n", !stats.RelayClosed)
			continue
		}

		if strings.HasPrefix(input, "xtp ") {
			parts := strings.SplitN(input, " ", 3)
			if len(parts) == 3 {
				target := parts[1]
				msg := parts[2]
				targetDID, _ := crypto.ResolveNode(target)
				if targetDID == "" {
					targetDID = target
				}
				if globalTM == nil {
					fmt.Println("❌ TransportManager no inicializado")
					continue
				}
				transport, err := globalTM.Send(targetDID, "CHAT:"+msg)
				if err != nil {
					fmt.Printf("❌ Error enviando por XTP: %v\n", err)
				} else {
					fmt.Printf("📤 Enviado por %s → %s\n", transport, targetDID[:20]+"...")
				}
			} else {
				fmt.Println("Uso: xtp <did|alias> <mensaje>")
				fmt.Println("Ejemplo: xtp juan hola mundo")
			}
			continue
		}

		if strings.HasPrefix(input, "/to ") {
			parts := strings.SplitN(input, " ", 3)
			if len(parts) == 2 {
				target := strings.TrimSpace(parts[1])
				if target == "off" {
					activeRecipient = ""
					fmt.Println("✅ Modo normal. Usá 'chat <alias>' o 'group send <alias>'.")
				} else if strings.HasPrefix(target, "group:") {
					alias := strings.TrimPrefix(target, "group:")
					activeRecipient = "group:" + alias
					if history, ok := msgHistory[alias]; ok && len(history) > 0 {
						fmt.Printf("📜 Historial de mensajes con grupo '%s':\n", alias)
						for i, msg := range history {
							fmt.Printf(" %d. %s\n", i+1, msg)
						}
					}
					fmt.Printf("📡 Modo grupo '%s'. Escribí y dale Enter.\n", alias)
				} else {
					activeRecipient = "chat:" + target
					if history, ok := msgHistory[target]; ok && len(history) > 0 {
						fmt.Printf("📜 Historial de mensajes con '%s':\n", target)
						for i, msg := range history {
							fmt.Printf(" %d. %s\n", i+1, msg)
						}
					}
					fmt.Printf("📡 Modo chat con '%s'. Escribí y dale Enter.\n", target)
				}
			} else if len(parts) == 3 {
				target := strings.TrimSpace(parts[1])
				flag := strings.TrimSpace(parts[2])
				if flag == "on" {
					activeRecipient = "chat:" + target
					if _, ok := msgHistory[target]; !ok {
						msgHistory[target] = []string{}
					}
					fmt.Printf("📡 Modo chat con '%s' (historial activado). Escribí y dale Enter.\n", target)
				} else if flag == "off" {
					activeRecipient = "chat:" + target
					delete(msgHistory, target)
					fmt.Printf("📡 Modo chat con '%s' (historial desactivado).\n", target)
				} else {
					fmt.Printf("⚠️ Uso: /to <alias> [on|off] o /to off\n")
				}
			}
			continue
		}

		if activeRecipient != "" && !isCommand(input) {
			if strings.HasPrefix(activeRecipient, "chat:") {
				alias := strings.TrimPrefix(activeRecipient, "chat:")
				if history, ok := msgHistory[alias]; ok {
					msgHistory[alias] = append(history, input)
				}
				fullCmd := fmt.Sprintf("chat %s %s", alias, input)
				output := commands.Execute(fullCmd, globalID)
				if output != "" {
					fmt.Println(output)
				}
			} else if strings.HasPrefix(activeRecipient, "group:") {
				alias := strings.TrimPrefix(activeRecipient, "group:")
				if history, ok := msgHistory[alias]; ok {
					msgHistory[alias] = append(history, input)
				}
				fullCmd := fmt.Sprintf("group send %s %s", alias, input)
				output := commands.Execute(fullCmd, globalID)
				if output != "" {
					fmt.Println(output)
				}
			}
			continue
		}

		if input == "exit" || input == "/exit" {
			fmt.Println("\n👋 Saliendo de la consola asegurada...")
			if globalTM != nil {
				globalTM.Close()
			}
			close(globalQuit)
			// msgChan no se cierra: el proceso termina con return
			if globalUseWS && globalConnWS != nil {
				globalConnWS.Close()
			}
			if !globalUseWS && globalConn != nil {
				globalConn.Close()
			}
			fmt.Println("🧹 Historial de comandos y mensajes eliminado (privacidad total).")
			return
		}

		output := commands.Execute(input, globalID)
		if output != "" {
			fmt.Println(output)
		}
	}
}
