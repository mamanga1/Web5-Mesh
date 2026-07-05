package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"web5-mesh/cmd/mesh/commands"
	"web5-mesh/src/crypto"
)

const FaroAddr = "190.220.45.26:54321"
const ListenPort = "54321"

func addPadding(payload string) string {
	size := 50 + int(time.Now().UnixNano()%150)
	padding := make([]byte, size)
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := range padding {
		padding[i] = charset[rand.Intn(len(charset))]
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
			fmt.Printf("❌ Error decodificando PubKeyEd para %s: %v\n", did, err)
			continue
		}
		pubX, err := hex.DecodeString(peer.PubKeyX)
		if err != nil {
			fmt.Printf("❌ Error decodificando PubKeyX para %s: %v\n", did, err)
			continue
		}
		sharedKey, err := crypto.DeriveSharedKey(myID.PrivKeyX, pubX)
		if err != nil {
			fmt.Printf("❌ Error derivando shared key para %s: %v\n", did, err)
			continue
		}
		kid := crypto.DeriveKeyID(pubX)
		index[kid] = peerKeys{DID: did, PubKeyEd: pubEd, SharedKey: sharedKey}
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

// ============================================================
// ExecuteRealCommand - VERSIÓN CON RELAY (LA QUE FUNCIONABA)
// ============================================================
func ExecuteRealCommand(myID *crypto.Identity, targetDID, command string) string {
	// 1. Cargar ACL y obtener claves del peer
	acl, err := crypto.LoadACL()
	if err != nil {
		return fmt.Sprintf("❌ Error cargando ACL: %v", err)
	}

	_, pubX, err := acl.GetPeerKeys(targetDID)
	if err != nil {
		return "❌ DID no encontrado en tu acl.json."
	}

	// 2. Derivar clave compartida
	sharedKey, err := crypto.DeriveSharedKey(myID.PrivKeyX, pubX)
	if err != nil {
		return fmt.Sprintf("❌ Error derivando clave: %v", err)
	}

	// 3. Construir payload cifrado
	inner := InnerPayload{
		FromDID: myID.DID,
		TS:      time.Now().Unix(),
		Cmd:     command,
	}

	payload, err := buildEncryptedPayload(myID, sharedKey, inner)
	if err != nil {
		return fmt.Sprintf("❌ Error cifrando: %v", err)
	}

	// 4. Conectar al Faro
	faroAddr, err := net.ResolveUDPAddr("udp", FaroAddr)
	if err != nil {
		return fmt.Sprintf("❌ Error resolviendo Faro: %v", err)
	}

	conn, err := net.DialUDP("udp", nil, faroAddr)
	if err != nil {
		return fmt.Sprintf("❌ Error conectando al Faro: %v", err)
	}
	defer conn.Close()

	// 5. Enviar RELAY al Faro (la versión que funcionaba)
	relayCmd := fmt.Sprintf("RELAY %s %s %s", targetDID, myID.DID, addPadding(payload))
	conn.Write([]byte(relayCmd))

	// 6. Esperar respuesta del Faro (que reenvía la respuesta del destino)
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	buf := make([]byte, 4096)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return "⏳ Timeout: El nodo destino no respondió a tiempo"
	}

	// 7. Procesar respuesta
	respRaw := stripPadding(string(buf[:n]))
	respRaw = extractPayload(respRaw)
	parts := strings.SplitN(respRaw, "|", 2)
	if len(parts) == 2 {
		ciphertext, _ := base64.StdEncoding.DecodeString(parts[1])
		plaintext, _ := crypto.DecryptPayload(sharedKey, ciphertext)
		var innerResp InnerPayload
		if json.Unmarshal(plaintext, &innerResp) == nil {
			return fmt.Sprintf("📩 %s", innerResp.Cmd)
		}
	}
	return fmt.Sprintf("📩 Respuesta cruda: %s", respRaw)
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
	faroAddr, _ := net.ResolveUDPAddr("udp", FaroAddr)
	conn, _ := net.DialUDP("udp", nil, faroAddr)
	defer conn.Close()

	inner := InnerPayload{FromDID: myID.DID, TS: time.Now().Unix(), Cmd: command}
	payload, err := buildEncryptedPayload(myID, sharedKey, inner)
	if err != nil {
		log.Fatalf("❌ Error cifrando: %v", err)
	}

	conn.Write([]byte(fmt.Sprintf("RELAY %s %s %s", targetDID, myID.DID, addPadding(payload))))

	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	buf := make([]byte, 4096)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		log.Fatalf("⏳ Timeout esperando respuesta")
	}

	respRaw := stripPadding(string(buf[:n]))
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

func runNode(myID *crypto.Identity) {
	aclIndex, _ := buildACLIndex(myID)
	fmt.Printf("🛡️ [NODO] ACL indexada con %d pares. Escuchando...\n", len(aclIndex))
	fmt.Println("📡 Keep-alive activo cada 15s.")

	faroAddr, _ := net.ResolveUDPAddr("udp", FaroAddr)
	conn, _ := net.DialUDP("udp", nil, faroAddr)
	defer conn.Close()

	go func() {
		for {
			ts := fmt.Sprintf("%d", time.Now().Unix())
			msg := fmt.Sprintf("ANNOUNCE %s %s %s", myID.DID, ts, base64.StdEncoding.EncodeToString(myID.SignMessage([]byte(ts))))
			conn.Write([]byte(addPadding(msg)))
			time.Sleep(15 * time.Second)
		}
	}()

	buf := make([]byte, 4096)
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

		conn.Write([]byte(fmt.Sprintf("RESPONSE %s %s", peer.DID, addPadding(respPayload))))
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
