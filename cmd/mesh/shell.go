package main

import (
        "crypto/tls"
        "encoding/base64"
        "encoding/hex"
        "encoding/json"
        "fmt"
        "net"
        "os"
        "os/exec"
        "strings"
        "time"

        "github.com/c-bata/go-prompt"
        "github.com/gorilla/websocket"
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
// COMPLETER - Autocompletado
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
)

// ============================================================================
// CONEXIÓN AL FARO (WebSocket + fallback UDP)
// ============================================================================

func connectToFaroShell() error {
        if globalFaroAddr == "" {
                return fmt.Errorf("faro no configurado")
        }

        // Intentar WebSocket primero
        wsURL := fmt.Sprintf("wss://%s/ws", globalFaroAddr)
        
        dialer := websocket.Dialer{
                TLSClientConfig: &tls.Config{
                        InsecureSkipVerify: true,
                },
                HandshakeTimeout: 5 * time.Second,
        }
        
        wsConn, _, err := dialer.Dial(wsURL, nil)
        if err == nil {
                fmt.Printf("✅ Conectado al faro por WebSocket: %s\n", globalFaroAddr)
                globalConnWS = wsConn
                globalUseWS = true
                return nil
        }
        
        // Fallback a UDP
        fmt.Printf("⚠️ WebSocket falló (%v), usando UDP\n", err)
        
        addr, err := net.ResolveUDPAddr("udp", globalFaroAddr)
        if err != nil {
                return fmt.Errorf("error resolviendo faro: %v", err)
        }
        
        conn, err := net.DialUDP("udp", nil, addr)
        if err != nil {
                return fmt.Errorf("error conectando al faro: %v", err)
        }
        
        globalConn = conn
        globalUseWS = false
        fmt.Printf("✅ Conectado al faro por UDP: %s\n", globalFaroAddr)
        return nil
}

func sendToFaroShell(msg string) error {
        if globalUseWS {
                return globalConnWS.WriteMessage(websocket.TextMessage, []byte(msg))
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
// LISTENER UDP
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
                                select {
                                case <-globalQuit:
                                        return
                                default:
                                        continue
                                }
                        }

                        raw = stripPadding(raw)
                        raw = extractPayload(raw)

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

                        // ====================================================================
                        // SINCRONIZACIÓN DE GRUPOS
                        // ====================================================================

                        if strings.HasPrefix(inner.Cmd, "GROUP_SYNC:") {
                                parts := strings.SplitN(inner.Cmd, ":", 3)
                                if len(parts) == 3 {
                                        alias := parts[1]
                                        var group crypto.Group
                                        if json.Unmarshal([]byte(parts[2]), &group) == nil {
                                                crypto.SaveGroupDirect(alias, &group)
                                                fmt.Printf("\n🔄 [SISTEMA] Grupo '%s' sincronizado (miembros: %d)\n", alias, len(group.Members))
                                        }
                                }
                                continue
                        }

                        if strings.HasPrefix(inner.Cmd, "GROUP_DELETE:") {
                                parts := strings.SplitN(inner.Cmd, ":", 2)
                                if len(parts) == 2 {
                                        alias := parts[1]
                                        crypto.DeleteGroup(alias)
                                        fmt.Printf("\n🗑️ [SISTEMA] Grupo '%s' eliminado por el admin\n", alias)
                                }
                                continue
                        }

                        if strings.HasPrefix(inner.Cmd, "GROUP_KICKED:") {
                                parts := strings.SplitN(inner.Cmd, ":", 2)
                                if len(parts) == 2 {
                                        alias := parts[1]
                                        crypto.RemoveMember(alias, globalID.DID)
                                        fmt.Printf("\n👢 [SISTEMA] Fuiste expulsado del grupo '%s'\n", alias)
                                }
                                continue
                        }

                        if strings.HasPrefix(inner.Cmd, "GROUP_LEAVE:") {
                                parts := strings.SplitN(inner.Cmd, ":", 3)
                                if len(parts) == 3 {
                                        alias := parts[1]
                                        did := parts[2]
                                        crypto.RemoveMember(alias, did)
                                        fmt.Printf("\n👋 [SISTEMA] %s salió del grupo '%s'\n", displayName, alias)
                                }
                                continue
                        }

                        // ====================================================================
                        // MENSAJES DE CHAT DE GRUPO
                        // ====================================================================

                        if strings.HasPrefix(inner.Cmd, "GROUP:") {
                                parts := strings.SplitN(inner.Cmd, ":", 3)
                                if len(parts) == 3 {
                                        fmt.Printf("\n💬 [GRUPO:%s] [%s]: %s\n", parts[1], displayName, parts[2])
                                        continue
                                }
                        }

                        fmt.Printf("\n💬 [%s]: %s\n", displayName, inner.Cmd)
                }
        }
}

// ============================================================================
// ANNOUNCE LOOP
// ============================================================================

func startAnnounceLoop() {
        if !globalUseWS && globalConn == nil {
                return
        }

        ticker := time.NewTicker(15 * time.Second)
        defer ticker.Stop()

        for {
                select {
                case <-globalQuit:
                        return
                case <-ticker.C:
                        ts := fmt.Sprintf("%d", time.Now().Unix())
                        sig := base64.StdEncoding.EncodeToString(globalID.SignMessage([]byte(ts)))
                        msg := fmt.Sprintf("ANNOUNCE %s %s %s", globalID.DID, ts, sig)
                        sendToFaroShell(addPadding(msg))
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
        fmt.Println("  XION KERNEL v1.0.0 | Modo Seguro: ON")
        fmt.Println("  🆔 TU IDENTIDAD:")
        fmt.Println("  DID:        " + globalID.DID)

        pubEdHex := hex.EncodeToString(globalID.PubKeyEd)
        pubXHex := hex.EncodeToString(globalID.PubKeyX[:])
        fmt.Println("  PubKey Ed:  " + pubEdHex)
        fmt.Println("  PubKey X:   " + pubXHex)
        fmt.Println()
        fmt.Println("  📋 Para que otro nodo te agregue, decile que ejecute:")
        fmt.Printf("  acl import %s %s %s\n", globalID.DID, pubEdHex, pubXHex)
        fmt.Printf("  alias add <nick> %s\n", globalID.DID)
        fmt.Println()
        if globalFaroAddr != "" {
                fmt.Printf("  📡 Faro activo: %s\n", globalFaroAddr)
        } else {
                fmt.Println("  📡 Faro: NO CONFIGURADO")
        }
        fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
        fmt.Printf("🛡️ [NODO] ACL indexada con %d pares. Escuchando y listo.\n\n", len(globalACLIndex))

        go startNetworkListener()
        go startAnnounceLoop()

        for {
                input := prompt.Input("xion@nodo:~$ ", completer,
                        prompt.OptionPrefixTextColor(prompt.Yellow),
                        prompt.OptionInputTextColor(prompt.White),
                        prompt.OptionSuggestionBGColor(prompt.DarkGray),
                        prompt.OptionSuggestionTextColor(prompt.White),
                        prompt.OptionSelectedSuggestionBGColor(prompt.Yellow),
                        prompt.OptionSelectedSuggestionTextColor(prompt.Black),
                )

                input = strings.TrimSpace(input)
                if input == "" {
                        continue
                }

                if input == "exit" || input == "/exit" {
                        fmt.Println("\n👋 Saliendo de la consola asegurada...")
                        close(globalQuit)
                        if globalUseWS && globalConnWS != nil {
                                globalConnWS.Close()
                        }
                        if !globalUseWS && globalConn != nil {
                                globalConn.Close()
                        }
                        return
                }

                output := commands.Execute(input, globalID)
                if output != "" {
                        fmt.Println(output)
                }
        }
}
