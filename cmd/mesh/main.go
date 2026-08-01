package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"web5-mesh/cmd/mesh/commands"
	"web5-mesh/src/config"
	"web5-mesh/src/crypto"
	"web5-mesh/src/xtp"
)

// Variables inyectadas por build.sh (ldflags)
var (
	buildCommit  string
	buildTime    string
	buildVersion string
)

const ListenPort = "54321"

func getFaroAddr() string {
	return config.GetFaroAddr()
}

func addPadding(payload string) string {
	size := 50 + int(time.Now().UnixNano()%150)
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	padding := make([]byte, size)
	for i := range padding {
		randBuf := make([]byte, 1)
		if _, err := rand.Read(randBuf); err != nil {
			padding[i] = charset[0]
			continue
		}
		padding[i] = charset[int(randBuf[0])%len(charset)]
	}
	return fmt.Sprintf("%s|%s", payload, string(padding))
}

func stripPadding(data string) string {
	if idx := strings.LastIndex(data, "|"); idx != -1 {
		return data[:idx]
	}
	return data
}

func extractPayload(raw string) string {
	if strings.HasPrefix(raw, "RELAY ") {
		fields := strings.Fields(raw)
		if len(fields) >= 4 {
			return fields[3]
		}
	} else if strings.HasPrefix(raw, "RESPONSE ") {
		fields := strings.Fields(raw)
		if len(fields) >= 3 {
			return fields[2]
		}
	} else if strings.HasPrefix(raw, "ACK ") {
		fields := strings.Fields(raw)
		if len(fields) >= 3 {
			return fields[2]
		}
	}
	return raw
}

type InnerPayload struct {
	FromDID string `json:"from"`
	TS      int64  `json:"ts"`
	Cmd     string `json:"cmd"`
	Sig     string `json:"sig"`
}

type peerKeys struct {
	DID       string
	PubKeyEd  []byte
	PubKeyX   []byte
	SharedKey []byte
}

func buildACLIndex(myID *crypto.Identity) (map[[4]byte]peerKeys, error) {
	acl, err := crypto.LoadACL()
	if err != nil {
		return nil, err
	}
	index := make(map[[4]byte]peerKeys)
	for did, peer := range acl.Peers {
		pubEd, err := hex.DecodeString(peer.PubKeyEd)
		if err != nil {
			continue
		}
		pubX, err := hex.DecodeString(peer.PubKeyX)
		if err != nil {
			continue
		}
		sharedKey, err := crypto.DeriveSharedKey(myID.PrivKeyX, pubX)
		if err != nil {
			continue
		}
		kid := crypto.DeriveKeyID(pubX)
		index[kid] = peerKeys{DID: did, PubKeyEd: pubEd, PubKeyX: pubX, SharedKey: sharedKey}
	}
	return index, nil
}

func buildEncryptedPayload(myID *crypto.Identity, sharedKey []byte, inner InnerPayload) (string, error) {
	innerJSON, _ := json.Marshal(inner)
	inner.Sig = base64.StdEncoding.EncodeToString(myID.SignMessage(innerJSON))
	innerJSON, _ = json.Marshal(inner)

	encrypted, err := crypto.EncryptPayload(sharedKey, innerJSON)
	if err != nil {
		return "", err
	}
	kid := crypto.DeriveKeyID(myID.PubKeyX[:])
	return fmt.Sprintf("%s|%s", hex.EncodeToString(kid[:]), base64.StdEncoding.EncodeToString(encrypted)), nil
}

func handleCommand(cmd string) string {
	if strings.HasPrefix(cmd, "PING") || strings.HasPrefix(cmd, "/ping") {
		return fmt.Sprintf("✅ PONG | %s", time.Now().Format("15:04:05"))
	}
	if strings.HasPrefix(cmd, "CHAT:") {
		return fmt.Sprintf("✅ Recibido: %s", strings.TrimPrefix(cmd, "CHAT:"))
	}
	return "✅ ACK"
}

func ExecuteRealCommand(myID *crypto.Identity, targetDID, command string) string {
	if targetDID == "" {
		return "❌ DID destino vacío"
	}

	if strings.HasPrefix(command, "GROUP_") || strings.HasPrefix(command, "GROUP:") {
		acl, err := crypto.LoadACL()
		if err != nil {
			return fmt.Sprintf("❌ Error cargando ACL: %v", err)
		}
		_, pubX, err := acl.GetPeerKeys(targetDID)
		if err != nil {
			return "❌ DID no encontrado en tu acl.json."
		}
		sharedKey, err := crypto.DeriveSharedKey(myID.PrivKeyX, pubX)
		if err != nil {
			return fmt.Sprintf("❌ Error derivando clave: %v", err)
		}
		inner := InnerPayload{
			FromDID: myID.DID,
			TS:      time.Now().Unix(),
			Cmd:     command,
		}
		payload, err := buildEncryptedPayload(myID, sharedKey, inner)
		if err != nil {
			return fmt.Sprintf("❌ Error cifrando: %v", err)
		}
		relayCmd := fmt.Sprintf("RELAY %s %s %s", targetDID, myID.DID, addPadding(payload))
		if err := sendToFaroShell(relayCmd); err != nil {
			return fmt.Sprintf("❌ Error enviando: %v", err)
		}
		return "📤 Mensaje de grupo enviado (relay)"
	}

	if globalTM != nil {
		transport, err := globalTM.Send(targetDID, command)
		if err != nil {
			return fmt.Sprintf("❌ XTP error: %v", err)
		}
		return fmt.Sprintf("📤 Mensaje enviado (%s)", transport)
	}

	return "❌ Transporte XTP no inicializado"
}

func init() {
	commands.SetNetworkExecutor(ExecuteRealCommand)
}

func runShell(myID *crypto.Identity, targetDID, command string) {
	acl, _ := crypto.LoadACL()
	_, pubX, err := acl.GetPeerKeys(targetDID)
	if err != nil {
		log.Fatalf("❌ DID no encontrado en ACL")
	}
	sharedKey, _ := crypto.DeriveSharedKey(myID.PrivKeyX, pubX)

	inner := InnerPayload{FromDID: myID.DID, TS: time.Now().Unix(), Cmd: command}
	payload, err := buildEncryptedPayload(myID, sharedKey, inner)
	if err != nil {
		log.Fatalf("❌ Error cifrando: %v", err)
	}

	if err := sendToFaroShell(fmt.Sprintf("RELAY %s %s %s", targetDID, myID.DID, addPadding(payload))); err != nil {
		log.Fatalf("❌ Error enviando: %v", err)
	}

	respRaw, err := readFromFaroShell()
	if err != nil {
		log.Fatalf("⏳ Timeout esperando respuesta")
	}
	respRaw = stripPadding(respRaw)
	respRaw = extractPayload(respRaw)

	parts := strings.SplitN(respRaw, "|", 2)
	if len(parts) == 2 {
		ciphertext, _ := base64.StdEncoding.DecodeString(parts[1])
		plaintext, _ := crypto.DecryptPayload(sharedKey, ciphertext)
		var innerResp InnerPayload
		if json.Unmarshal(plaintext, &innerResp) == nil {
			fmt.Printf("📩 Respuesta descifrada: %s\n", innerResp.Cmd)
			return
		}
	}
	fmt.Printf("📩 Respuesta (cruda): %s\n", respRaw)
}

// FIX 17: runNode ahora usa XTP (TransportManager) en vez de
// protocolo legacy sin Noise. Inicializa globalTM y procesa mensajes
// a través del manager, que intenta directo primero y cae a relay.
func runNode(myID *crypto.Identity) {
	aclIndex, _ := buildACLIndex(myID)
	fmt.Printf("🛡️ [NODO] ACL indexada con %d pares. Escuchando...\n", len(aclIndex))
	fmt.Println("📡 Keep-alive activo cada 15s.")
	fmt.Println("🔐 XTP: transporte directo Noise IK activo")

	if globalConn == nil && globalConnWS == nil {
		if err := connectToFaroShell(); err != nil {
			log.Fatalf("❌ Error conectando al Faro: %v", err)
		}
	}

	// Inicializar TransportManager (igual que en runInteractiveShell)
	xtpACLIndex := make(map[[4]byte]xtp.PeerKeys, len(aclIndex))
	for kid, pk := range aclIndex {
		xtpACLIndex[kid] = xtp.PeerKeys{
			DID:       pk.DID,
			PubKeyEd:  pk.PubKeyEd,
			PubKeyX:   pk.PubKeyX,
			SharedKey: pk.SharedKey,
		}
	}

	globalTM = xtp.NewTransportManager(
		myID,
		faroSenderShell{},
		xtpACLIndex,
		xtp.ManagerCallbacks{
			OnMessage: func(peerDID, displayName, command string) {
				fmt.Printf("📩 [%s] %s\n", displayName, command)
				respText := handleCommand(command)
				if globalTM != nil {
					globalTM.Send(peerDID, respText)
				}
			},
			OnDirectSessionActive: func(peerDID string) {
				fmt.Printf("🔐 [XTP] Conexión directa con %s\n", peerDID[:20]+"...")
			},
			OnDirectSessionLost: func(peerDID string) {
				fmt.Printf("💀 [XTP] Sesión directa perdida con %s\n", peerDID[:20]+"...")
			},
			OnFallbackToRelay: func(peerDID string) {
				fmt.Printf("🔄 [XTP] Usando relay con %s\n", peerDID[:20]+"...")
			},
			OnError: func(context string, err error) {
				fmt.Printf("[XTP] ⚠️ Error en %s: %v\n", context, err)
			},
		},
		xtp.DefaultManagerConfig(),
	)

	globalTM.FSM().Send(xtp.EvConnectFaro, nil)
	globalTM.FSM().Send(xtp.EvFaroConnected, nil)
	globalTM.FSM().Send(xtp.EvAnnounceSent, nil)

	go func() {
		for {
			ts := fmt.Sprintf("%d", time.Now().Unix())
			msg := fmt.Sprintf("ANNOUNCE %s %s %s", myID.DID, ts, base64.StdEncoding.EncodeToString(myID.SignMessage([]byte(ts))))
			sendToFaroShell(addPadding(msg))
			time.Sleep(15 * time.Second)
		}
	}()

	for {
		raw, err := readFromFaroShell()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		raw = stripPadding(raw)
		raw = extractPayload(raw)

		// XTP maneja señales de faro y mensajes relay/directo
		if globalTM != nil && globalTM.HandleIncoming(raw) {
			continue
		}

		// Fallback: mensajes que XTP no reconoce (ACK_IP, etc.)
		if strings.HasPrefix(raw, "ACK_IP ") || strings.HasPrefix(raw, "ACK") {
			continue
		}

		// Legacy: mensajes relay que XTP no procesó
		parts := strings.SplitN(raw, "|", 2)
		if len(parts) != 2 {
			continue
		}
		kidBytes, _ := hex.DecodeString(parts[0])
		if len(kidBytes) != 4 {
			continue
		}
		var kid [4]byte
		copy(kid[:], kidBytes)

		peer, exists := aclIndex[kid]
		if !exists {
			continue
		}
		ciphertext, _ := base64.StdEncoding.DecodeString(parts[1])
		plaintext, _ := crypto.DecryptPayload(peer.SharedKey, ciphertext)
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

		fmt.Printf("📩 [%s] %s\n", peer.DID, inner.Cmd)
		respText := handleCommand(inner.Cmd)
		respInner := InnerPayload{FromDID: myID.DID, TS: time.Now().Unix(), Cmd: respText}
		respPayload, _ := buildEncryptedPayload(myID, peer.SharedKey, respInner)
		sendToFaroShell(fmt.Sprintf("RESPONSE %s %s", peer.DID, addPadding(respPayload)))
	}
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "shell" {
		runInteractiveShell()
		return
	}

	toFlag := flag.String("to", "", "DID destino")
	cmdFlag := flag.String("cmd", "", "Comando a ejecutar")
	flag.Parse()

	id, err := crypto.LoadOrCreateIdentity()
	if err != nil {
		log.Fatalf("❌ Error de identidad: %v", err)
	}

	if *toFlag != "" && *cmdFlag != "" {
		runShell(id, *toFlag, *cmdFlag)
	} else {
		runNode(id)
	}
}
