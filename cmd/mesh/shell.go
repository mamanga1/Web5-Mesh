package main

import (
	crypto_rand "crypto/rand" // ← PARCHE: para nonce del handshake
	"crypto/tls"              // ← PARCHE: para WSS fallback
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http" // ← PARCHE: para headers del handshake WS
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/c-bata/go-prompt"
	"github.com/gorilla/websocket"
	"github.com/mr-tron/base58"
	"web5-mesh/cmd/mesh/commands"
	"web5-mesh/src/crypto"
)

// ============================================================================
// RESTAURAR TERMINAL
// ============================================================================

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

// ============================================================================
// COMPLETER
// ============================================================================

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
	}

	s := []prompt.Suggest{}
	allCommands := []string{
		"whoami", "acl", "alias", "group", "faro", "help", "exit",
		"acl add", "acl import", "acl remove", "acl list", "acl clear",
		"alias add", "alias remove", "alias list",
		"group create", "group list", "group send", "group add", "group remove",
		"group invite", "group kick", "group leave", "group delete", "group info",
		"faro set", "faro reset",
	}

	for _, cmd := range allCommands {
		if strings.HasPrefix(cmd, text) {
			s = append(s, prompt.Suggest{Text: cmd})
		}
	}

	return s
}

// ============================================================================
// VARIABLES GLOBALES
// ============================================================================

var (
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
	msgHistory     map[string][]string
	lastPublicIP   string
)

// ============================================================================
// isCommand
// ============================================================================

func isCommand(input string) bool {
	known := []string{
		"whoami", "acl", "alias", "group", "faro", "help", "exit",
		"clear", "import", "export", "sign", "verify", "ls", "cat",
		"mkdir", "rm", "pwd", "touch", "edit", "mv", "cp", "rmdir",
		"chat", "ia", "browse", "host", "sync", "proxy",
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

// ============================================================================
// CONEXIÓN AL FARO — UDP default, TCP fallback, Gate DID
// ← PARCHE: función reemplazada + 2 funciones nuevas
// ============================================================================

func connectToFaroShell() error {
	if globalFaroAddr == "" {
		return fmt.Errorf("faro no configurado")
	}

	// 1. UDP primero (default)
	if err := connectUDPShell(); err == nil {
		return nil
	}

	// 2. WS fallback
	if err := connectWSShell(); err == nil {
		return nil
	}

	return fmt.Errorf("sin ruta al faro %s", globalFaroAddr)
}

// ← PARCHE: función nueva
func connectUDPShell() error {
	addr, err := net.ResolveUDPAddr("udp", globalFaroAddr)
	if err != nil {
		return fmt.Errorf("resolviendo UDP: %v", err)
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("conectando UDP: %v", err)
	}

	// Handshake DID
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

// ← PARCHE: función nueva
func connectWSShell() error {
	wsHost := globalFaroAddr
	if !strings.Contains(wsHost, ":") {
		wsHost += ":443"
	}
	if strings.HasSuffix(wsHost, ":54321") {
		wsHost = strings.TrimSuffix(wsHost, ":54321") + ":443"
	}

	// Handshake DID en headers
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
	dialer := websocket.Dialer{
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
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
		_, message, err := globalConnWS.ReadMessage()
		if err != nil {
			return "", err
		}
		return string(message), nil
	}

	buf := make([]byte, 65536)
	globalConn.SetReadDeadline(time.Now().Add(15 * time.Second))
	n, _, err := globalConn.ReadFromUDP(buf)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

// ============================================================================
// LISTENER + ROAMING
// ============================================================================

func startNetworkListener() {
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

			raw = stripPadding(raw)
			raw = extractPayload(raw)

			// === ACK_IP: Roaming ===
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
						crypto.SaveGroupDirect(alias, &group)
						msgChan <- fmt.Sprintf("🔄 [SISTEMA] Grupo '%s' sincronizado (miembros: %d)", alias, len(group.Members))
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

// ============================================================================
// ANNOUNCE LOOP
// ============================================================================

func startAnnounceLoop() {
	for globalConn == nil && globalConnWS == nil {
		select {
		case <-globalQuit:
			return
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
	ticker := time.NewTicker(15 * time.Second)
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
			_ = sendToFaroShell(addPadding(msg))
		}
	}
}

// ============================================================================
// SHELL PRINCIPAL
// ============================================================================

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

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println(" XION KERNEL v1.0.0 | Modo Seguro: ON")
	fmt.Println(" 🆔 TU IDENTIDAD:")
	fmt.Println(" DID: " + globalID.DID)

	pubEdHex := hex.EncodeToString(globalID.PubKeyEd)
	pubXHex := hex.EncodeToString(globalID.PubKeyX[:])
	fmt.Println(" PubKey Ed: " + pubEdHex)
	fmt.Println(" PubKey X:  " + pubXHex)
	fmt.Println()
	fmt.Println(" 📋 Para que otro nodo te agregue, decile que ejecute:")
	fmt.Printf("   acl import %s %s %s\n", globalID.DID, pubEdHex, pubXHex)
	fmt.Printf("   alias add  %s\n", globalID.DID)
	fmt.Println()
	if globalFaroAddr != "" {
		if globalUseWS {
			fmt.Printf(" 📡 Faro activo: %s (WSS fallback)\n", globalFaroAddr)
		} else {
			fmt.Printf(" 📡 Faro activo: %s (UDP default)\n", globalFaroAddr)
		}
	} else {
		fmt.Println(" 📡 Faro: NO CONFIGURADO")
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("🛡️ [NODO] ACL indexada con %d pares. Escuchando y listo.\n\n", len(globalACLIndex))

	cmdHistory = []string{}
	msgHistory = make(map[string][]string)

	go startNetworkListener()
	go startAnnounceLoop()

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

		// ============================================================
		// COMANDO /to
		// ============================================================
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
							fmt.Printf("  %d. %s\n", i+1, msg)
						}
					}
					fmt.Printf("📡 Modo grupo '%s'. Escribí y dale Enter.\n", alias)
				} else {
					activeRecipient = "chat:" + target
					if history, ok := msgHistory[target]; ok && len(history) > 0 {
						fmt.Printf("📜 Historial de mensajes con '%s':\n", target)
						for i, msg := range history {
							fmt.Printf("  %d. %s\n", i+1, msg)
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

		// ============================================================
		// MODO CONTEXTO ACTIVO
		// ============================================================
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
			close(globalQuit)
			close(msgChan)
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
