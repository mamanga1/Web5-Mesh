package main

import (
        "crypto/tls"
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

        "github.com/gorilla/websocket"
        "web5-mesh/cmd/mesh/commands"
        "web5-mesh/src/config"
        "web5-mesh/src/crypto"
)

const ListenPort = "54321"

var (
        connWS       *websocket.Conn
        connUDP      *net.UDPConn
        useWebSocket bool
)

func getFaroAddr() string {
        return config.GetFaroAddr()
}

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

func connectToFaro() error {
        faroAddr := getFaroAddr()
        
        // Intentar WebSocket primero
        wsURL := fmt.Sprintf("wss://%s/ws", faroAddr)
        
        dialer := websocket.Dialer{
                TLSClientConfig: &tls.Config{
                        InsecureSkipVerify: true,
                },
                HandshakeTimeout: 5 * time.Second,
        }
        
        wsConn, _, err := dialer.Dial(wsURL, nil)
        if err == nil {
                fmt.Printf("✅ Conectado al faro por WebSocket: %s\n", faroAddr)
                connWS = wsConn
                useWebSocket = true
                return nil
        }
        
        // Fallback a UDP
        fmt.Printf("⚠️ WebSocket falló (%v), usando UDP\n", err)
        
        addr, err := net.ResolveUDPAddr("udp", faroAddr)
        if err != nil {
                return fmt.Errorf("error resolviendo faro: %v", err)
        }
        
        conn, err := net.DialUDP("udp", nil, addr)
        if err != nil {
                return fmt.Errorf("error conectando al faro: %v", err)
        }
        
        connUDP = conn
        useWebSocket = false
        fmt.Printf("✅ Conectado al faro por UDP: %s\n", faroAddr)
        return nil
}

func sendToFaro(msg string) error {
        if useWebSocket {
                return connWS.WriteMessage(websocket.TextMessage, []byte(msg))
        }
        _, err := connUDP.Write([]byte(msg))
        return err
}

func readFromFaro() (string, error) {
        if useWebSocket {
                _, message, err := connWS.ReadMessage()
                if err != nil {
                        return "", err
                }
                return string(message), nil
        }
        
        buf := make([]byte, 4096)
        connUDP.SetReadDeadline(time.Now().Add(15 * time.Second))
        n, _, err := connUDP.ReadFromUDP(buf)
        if err != nil {
                return "", err
        }
        return string(buf[:n]), nil
}

// ============================================================
// ExecuteRealCommand - VERSIÓN CON WEBSOCKET + FALLBACK UDP
// ============================================================
func ExecuteRealCommand(myID *crypto.Identity, targetDID, command string) string {
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

        if err := connectToFaro(); err != nil {
                return fmt.Sprintf("❌ Error conectando al Faro: %v", err)
        }
        
        if useWebSocket {
                defer connWS.Close()
        } else {
                defer connUDP.Close()
        }

        relayCmd := fmt.Sprintf("RELAY %s %s %s", targetDID, myID.DID, addPadding(payload))
        if err := sendToFaro(relayCmd); err != nil {
                return fmt.Sprintf("❌ Error enviando: %v", err)
        }

        respRaw, err := readFromFaro()
        if err != nil {
                return "⏳ Timeout: El nodo destino no respondió a tiempo"
        }

        respRaw = stripPadding(respRaw)
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
        
        if err := connectToFaro(); err != nil {
                log.Fatalf("❌ Error conectando al Faro: %v", err)
        }
        
        if useWebSocket {
                defer connWS.Close()
        } else {
                defer connUDP.Close()
        }

        inner := InnerPayload{FromDID: myID.DID, TS: time.Now().Unix(), Cmd: command}
        payload, err := buildEncryptedPayload(myID, sharedKey, inner)
        if err != nil {
                log.Fatalf("❌ Error cifrando: %v", err)
        }

        if err := sendToFaro(fmt.Sprintf("RELAY %s %s %s", targetDID, myID.DID, addPadding(payload))); err != nil {
                log.Fatalf("❌ Error enviando: %v", err)
        }

        respRaw, err := readFromFaro()
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

func runNode(myID *crypto.Identity) {
        aclIndex, _ := buildACLIndex(myID)
        fmt.Printf("🛡️ [NODO] ACL indexada con %d pares. Escuchando...\n", len(aclIndex))
        fmt.Println("📡 Keep-alive activo cada 15s.")

        if err := connectToFaro(); err != nil {
                log.Fatalf("❌ Error conectando al Faro: %v", err)
        }
        
        if useWebSocket {
                defer connWS.Close()
        } else {
                defer connUDP.Close()
        }

        go func() {
                for {
                        ts := fmt.Sprintf("%d", time.Now().Unix())
                        msg := fmt.Sprintf("ANNOUNCE %s %s %s", myID.DID, ts, base64.StdEncoding.EncodeToString(myID.SignMessage([]byte(ts))))
                        sendToFaro(addPadding(msg))
                        time.Sleep(15 * time.Second)
                }
        }()

        for {
                raw, err := readFromFaro()
                if err != nil {
                        continue
                }

                raw = stripPadding(raw)
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

                sendToFaro(fmt.Sprintf("RESPONSE %s %s", peer.DID, addPadding(respPayload)))
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
