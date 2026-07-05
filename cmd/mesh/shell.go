package main

import (
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
			{Text: "faro", Description: "Gestión de Faros"},
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
	case "faro":
		if len(words) == 1 {
			return []prompt.Suggest{
				{Text: "add", Description: "Agregar Faro"},
				{Text: "list", Description: "Listar Faros"},
				{Text: "remove", Description: "Eliminar Faro"},
				{Text: "test", Description: "Probar Faros"},
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
		"faro add", "faro list", "faro remove", "faro test",
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
	globalACLIndex map[[4]byte]peerKeys
	globalID       *crypto.Identity
	globalFaroAddr string
)

// ============================================================================
// LISTENER UDP (CON RESPUESTA AL FARO - ESTE ES EL ÚNICO CAMBIO)
// ============================================================================

func startNetworkListener() {
	if globalConn == nil {
		return
	}

	buf := make([]byte, 65536)
	for {
		n, _, err := globalConn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		raw := stripPadding(string(buf[:n]))
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

		// === MANEJO DE MENSAJES DE GRUPO ===
		if strings.HasPrefix(inner.Cmd, "GROUP:") {
			parts := strings.SplitN(inner.Cmd, ":", 3)
			if len(parts) == 3 {
				fmt.Printf("\n💬 [GRUPO:%s] [%s]: %s\n", parts[1], displayName, parts[2])
				
				// >>> RESPUESTA AL FARO PARA MENSAJES DE GRUPO <<<
				respText := "✅ Recibido en grupo"
				respInner := InnerPayload{FromDID: globalID.DID, TS: time.Now().Unix(), Cmd: respText}
				respPayload, _ := buildEncryptedPayload(globalID, peer.SharedKey, respInner)
				globalConn.Write([]byte(fmt.Sprintf("RESPONSE %s %s", peer.DID, addPadding(respPayload))))
				continue
			}
		}

		// === MANEJO DE MENSAJES NORMALES ===
		fmt.Printf("\n💬 [%s]: %s\n", displayName, inner.Cmd)
		
		// >>> RESPUESTA AL FARO PARA MENSAJES NORMALES (LO QUE FALTABA) <<<
		respText := handleCommand(inner.Cmd)
		respInner := InnerPayload{FromDID: globalID.DID, TS: time.Now().Unix(), Cmd: respText}
		respPayload, _ := buildEncryptedPayload(globalID, peer.SharedKey, respInner)
		globalConn.Write([]byte(fmt.Sprintf("RESPONSE %s %s", peer.DID, addPadding(respPayload))))
	}
}

// ============================================================================
// ANNOUNCE LOOP
// ============================================================================

func startAnnounceLoop() {
	if globalConn == nil {
		return
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ts := fmt.Sprintf("%d", time.Now().Unix())
		sig := base64.StdEncoding.EncodeToString(globalID.SignMessage([]byte(ts)))
		msg := fmt.Sprintf("ANNOUNCE %s %s %s", globalID.DID, ts, sig)
		globalConn.Write([]byte(addPadding(msg)))
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

	globalFaroAddr = FaroAddr

	if globalFaroAddr != "" {
		faroAddr, _ := net.ResolveUDPAddr("udp", globalFaroAddr)
		globalConn, _ = net.DialUDP("udp", nil, faroAddr)
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
			if globalConn != nil {
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
