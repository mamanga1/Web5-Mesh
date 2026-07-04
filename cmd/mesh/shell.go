package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
	"web5-mesh/cmd/mesh/commands"
	"web5-mesh/src/crypto"
)

const colorPrompt = "\x1b[38;5;208m"
const colorReset = "\x1b[0m"

func getPrompt() string {
	return fmt.Sprintf("%sxion@nodo:~$%s ", colorPrompt, colorReset)
}

var commandHistory []string

var (
	currentLine      []rune
	currentCursorPos int
	lineMu           sync.Mutex
)

func redrawLine(line []rune, cursorPos int) {
	fmt.Print("\r\033[K" + getPrompt() + string(line))
	if cursorPos < len(line) {
		fmt.Printf("\033[%dD", len(line)-cursorPos)
	}
}

func readLine() (string, error) {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	lineMu.Lock()
	currentLine = []rune{}
	currentCursorPos = 0
	lineMu.Unlock()

	var historyIndex int = -1

	for {
		b := make([]byte, 256)
		n, err := os.Stdin.Read(b)
		if err != nil {
			return "", err
		}

		needsRedraw := false
		for i := 0; i < n; i++ {
			char := b[i]
			if char == '\r' || char == '\n' {
				fmt.Println()

				lineMu.Lock()
				result := string(currentLine)
				currentLine = []rune{}
				currentCursorPos = 0
				lineMu.Unlock()

				if len(result) > 0 {
					commandHistory = append(commandHistory, result)
					if len(commandHistory) > 100 {
						commandHistory = commandHistory[1:]
					}
				}

				return result, nil
			}
			if char == '\b' || char == '\x7f' {
				lineMu.Lock()
				if currentCursorPos > 0 {
					currentLine = append(currentLine[:currentCursorPos-1], currentLine[currentCursorPos:]...)
					currentCursorPos--
					needsRedraw = true
				}
				lineMu.Unlock()
				continue
			}
			if char == '\x1b' && i+2 < n && b[i+1] == '[' {
				switch b[i+2] {
				case 'A':
					lineMu.Lock()
					if len(commandHistory) > 0 {
						if historyIndex == -1 {
							historyIndex = len(commandHistory) - 1
						} else if historyIndex > 0 {
							historyIndex--
						}
						currentLine = []rune(commandHistory[historyIndex])
						currentCursorPos = len(currentLine)
						needsRedraw = true
					}
					lineMu.Unlock()
					i += 2
				case 'B':
					lineMu.Lock()
					if historyIndex != -1 {
						if historyIndex < len(commandHistory)-1 {
							historyIndex++
							currentLine = []rune(commandHistory[historyIndex])
						} else {
							historyIndex = -1
							currentLine = []rune{}
						}
						currentCursorPos = len(currentLine)
						needsRedraw = true
					}
					lineMu.Unlock()
					i += 2
				case 'C':
					lineMu.Lock()
					if currentCursorPos < len(currentLine) {
						currentCursorPos++
						needsRedraw = true
					}
					lineMu.Unlock()
					i += 2
				case 'D':
					lineMu.Lock()
					if currentCursorPos > 0 {
						currentCursorPos--
						needsRedraw = true
					}
					lineMu.Unlock()
					i += 2
				}
				continue
			}
			if char >= 32 && char <= 126 {
				lineMu.Lock()
				currentLine = append(currentLine[:currentCursorPos], append([]rune{rune(char)}, currentLine[currentCursorPos:]...)...)
				currentCursorPos++
				needsRedraw = true
				lineMu.Unlock()
			}
		}
		if needsRedraw {
			lineMu.Lock()
			redrawLine(currentLine, currentCursorPos)
			lineMu.Unlock()
		}
	}
}

func runInteractiveShell() {
	id, err := crypto.LoadOrCreateIdentity()
	if err != nil {
		fmt.Printf("❌ Error de identidad: %v\n", err)
		return
	}

	// Registrar el DID propio para mostrarlo como "yo"
	crypto.SetSelfDID(id.DID)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  XION KERNEL v1.0.0 | Modo Seguro: ON")
	fmt.Println("  DID: " + id.DID)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	aclIndex, _ := buildACLIndex(id)
	fmt.Printf("🛡️ [NODO] ACL indexada con %d pares. Escuchando y listo.\n", len(aclIndex))

	faroAddr, _ := net.ResolveUDPAddr("udp", FaroAddr)
	conn, _ := net.DialUDP("udp", nil, faroAddr)

	go func() {
		for {
			ts := fmt.Sprintf("%d", time.Now().Unix())
			sig := base64.StdEncoding.EncodeToString(id.SignMessage([]byte(ts)))
			msg := fmt.Sprintf("ANNOUNCE %s %s %s", id.DID, ts, sig)
			conn.Write([]byte(addPadding(msg)))
			time.Sleep(15 * time.Second)
		}
	}()

	go func() {
		buf := make([]byte, 65536)
		for {
			n, _, err := conn.ReadFromUDP(buf)
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

			peer, exists := aclIndex[kid]
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

			// === MANEJO DE SINCRONIZACIÓN DE GRUPO ===
			if strings.HasPrefix(inner.Cmd, "GROUP_SYNC:") {
				parts := strings.SplitN(inner.Cmd, ":", 3)
				if len(parts) == 3 {
					groupAlias := parts[1]
					groupJSON := parts[2]

					var group crypto.Group
					if err := json.Unmarshal([]byte(groupJSON), &group); err == nil {
						crypto.SaveGroupDirect(groupAlias, &group)
						sid := commands.CreateSession("group", groupAlias)

						lineMu.Lock()
						savedLine := make([]rune, len(currentLine))
						copy(savedLine, currentLine)
						savedPos := currentCursorPos
						lineMu.Unlock()

						fmt.Print("\r\033[K")
						displayName := crypto.ResolveDID(peer.DID)
						fmt.Printf("📩 [%s] te agregó al grupo: %s (%s)\n", displayName, groupAlias, group.Name)
						fmt.Printf("✅ Sesión [%d] creada automáticamente\n", sid)
						fmt.Print(getPrompt() + string(savedLine))
						if savedPos < len(savedLine) {
							fmt.Printf("\033[%dD", len(savedLine)-savedPos)
						}
						os.Stdout.Sync()
					}
					continue
				}
			}

			// === MANEJO DE ELIMINACIÓN DE GRUPO ===
			if strings.HasPrefix(inner.Cmd, "GROUP_DELETE:") {
				parts := strings.SplitN(inner.Cmd, ":", 2)
				if len(parts) == 2 {
					groupAlias := parts[1]
					crypto.DeleteGroup(groupAlias)

					lineMu.Lock()
					savedLine := make([]rune, len(currentLine))
					copy(savedLine, currentLine)
					savedPos := currentCursorPos
					lineMu.Unlock()

					fmt.Print("\r\033[K")
					displayName := crypto.ResolveDID(peer.DID)
					fmt.Printf("🗑️ [%s] eliminó el grupo: %s\n", displayName, groupAlias)
					fmt.Print(getPrompt() + string(savedLine))
					if savedPos < len(savedLine) {
						fmt.Printf("\033[%dD", len(savedLine)-savedPos)
					}
					os.Stdout.Sync()
					continue
				}
			}

			// === MANEJO DE SALIDA VOLUNTARIA DE MIEMBRO ===
			if strings.HasPrefix(inner.Cmd, "GROUP_LEAVE:") {
				parts := strings.SplitN(inner.Cmd, ":", 3)
				if len(parts) == 3 {
					groupAlias := parts[1]
					leaverDID := parts[2]

					crypto.RemoveMember(groupAlias, leaverDID)

					lineMu.Lock()
					savedLine := make([]rune, len(currentLine))
					copy(savedLine, currentLine)
					savedPos := currentCursorPos
					lineMu.Unlock()

					fmt.Print("\r\033[K")
					displayName := crypto.ResolveDID(leaverDID)
					fmt.Printf("👋 [%s] salió del grupo: %s\n", displayName, groupAlias)
					fmt.Print(getPrompt() + string(savedLine))
					if savedPos < len(savedLine) {
						fmt.Printf("\033[%dD", len(savedLine)-savedPos)
					}
					os.Stdout.Sync()
					continue
				}
			}

			// === MANEJO DE KICK (expulsado por admin) ===
			if strings.HasPrefix(inner.Cmd, "GROUP_KICKED:") {
				parts := strings.SplitN(inner.Cmd, ":", 2)
				if len(parts) == 2 {
					groupAlias := parts[1]
					crypto.DeleteGroup(groupAlias)

					lineMu.Lock()
					savedLine := make([]rune, len(currentLine))
					copy(savedLine, currentLine)
					savedPos := currentCursorPos
					lineMu.Unlock()

					fmt.Print("\r\033[K")
					displayName := crypto.ResolveDID(peer.DID)
					fmt.Printf("🚪 [%s] te expulsó del grupo: %s\n", displayName, groupAlias)
					fmt.Print(getPrompt() + string(savedLine))
					if savedPos < len(savedLine) {
						fmt.Printf("\033[%dD", len(savedLine)-savedPos)
					}
					os.Stdout.Sync()
					continue
				}
			}

			// === MANEJO DE INVITACIONES A GRUPO ===
			if strings.HasPrefix(inner.Cmd, "GROUP_INVITE:") {
				parts := strings.SplitN(inner.Cmd, ":", 3)
				if len(parts) == 3 {
					groupAlias := parts[1]
					inviterDID := parts[2]

					lineMu.Lock()
					savedLine := make([]rune, len(currentLine))
					copy(savedLine, currentLine)
					savedPos := currentCursorPos
					lineMu.Unlock()

					fmt.Print("\r\033[K")
					displayName := crypto.ResolveDID(inviterDID)
					fmt.Printf("📩 [%s] te invitó al grupo: %s\n", displayName, groupAlias)
					fmt.Print(getPrompt() + string(savedLine))
					if savedPos < len(savedLine) {
						fmt.Printf("\033[%dD", len(savedLine)-savedPos)
					}
					os.Stdout.Sync()
					continue
				}
			}

			// === MANEJO DE MENSAJES DE GRUPO ===
			if strings.HasPrefix(inner.Cmd, "GROUP:") {
				parts := strings.SplitN(inner.Cmd, ":", 3)
				if len(parts) == 3 {
					groupAlias := parts[1]
					message := parts[2]

					lineMu.Lock()
					savedLine := make([]rune, len(currentLine))
					copy(savedLine, currentLine)
					savedPos := currentCursorPos
					lineMu.Unlock()

					fmt.Print("\r\033[K")
					displayName := crypto.ResolveDID(peer.DID)
					fmt.Printf("💬 [GRUPO:%s] [%s]: %s\n", groupAlias, displayName, message)
					fmt.Print(getPrompt() + string(savedLine))
					if savedPos < len(savedLine) {
						fmt.Printf("\033[%dD", len(savedLine)-savedPos)
					}
					os.Stdout.Sync()

					respText := "✅ Recibido en grupo"
					respInner := InnerPayload{FromDID: id.DID, TS: time.Now().Unix(), Cmd: respText}
					respPayload, _ := buildEncryptedPayload(id, peer.SharedKey, respInner)
					conn.Write([]byte(fmt.Sprintf("RESPONSE %s %s", peer.DID, addPadding(respPayload))))
					continue
				}
			}

			// === MANEJO DE MENSAJES NORMALES (chat directo) ===
			lineMu.Lock()
			savedLine := make([]rune, len(currentLine))
			copy(savedLine, currentLine)
			savedPos := currentCursorPos
			lineMu.Unlock()

			fmt.Print("\r\033[K")
			displayName := crypto.ResolveDID(peer.DID)
			fmt.Printf("💬 [%s]: %s\n", displayName, inner.Cmd)
			fmt.Print(getPrompt() + string(savedLine))
			if savedPos < len(savedLine) {
				fmt.Printf("\033[%dD", len(savedLine)-savedPos)
			}
			os.Stdout.Sync()

			respText := handleCommand(inner.Cmd)
			respInner := InnerPayload{FromDID: id.DID, TS: time.Now().Unix(), Cmd: respText}
			respPayload, _ := buildEncryptedPayload(id, peer.SharedKey, respInner)
			conn.Write([]byte(fmt.Sprintf("RESPONSE %s %s", peer.DID, addPadding(respPayload))))
		}
	}()

	for {
		fmt.Print(getPrompt())
		input, err := readLine()
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		output := commands.Execute(input, id)
		if output != "" {
			fmt.Println(output)
		}

		if input == "exit" || input == "/exit" {
			fmt.Println("👋 Saliendo de la consola asegurada...")
			break
		}
	}
}
