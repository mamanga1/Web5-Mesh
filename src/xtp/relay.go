package xtp

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"web5-mesh/src/crypto"
)

// ============================================================================
// TIPOS DEL ACL INDEX (mismos que usan mobile.go y shell.go)
// ============================================================================

// PeerKeys contiene las claves de un peer autorizado (del ACL).
type PeerKeys struct {
	DID       string
	PubKeyEd  []byte
	SharedKey []byte
}

// ============================================================================
// INNER PAYLOAD (formato de Fase 1)
// ============================================================================

// InnerPayload es el mensaje antes de cifrar (formato de Fase 1).
type InnerPayload struct {
	FromDID string `json:"from"`
	TS      int64  `json:"ts"`
	Cmd     string `json:"cmd"`
	Sig     string `json:"sig"`
}

// ============================================================================
// CALLBACKS
// ============================================================================

type RelayCallbacks struct {
	// OnMessage: se llama cuando se recibe un mensaje descifrado del peer.
	OnMessage func(peerDID string, displayName string, command string)

	// OnError: se llama cuando hay un error (descifrado, firma, etc.).
	OnError func(peerDID string, err error)
}

// ============================================================================
// RELAY TRANSPORT
// ============================================================================

// RelayTransport envía y recibe mensajes a través del faro (relay ciego).
// Usa el cifrado de Fase 1: ECDH estático + ChaCha20-Poly1305.
//
// No tiene forward secrecy (la clave compartida es estática).
// Para forward secrecy, usar DirectTransport (Noise IK).
type RelayTransport struct {
	mu sync.RWMutex

	// Identidad del nodo
	identity *crypto.Identity

	// Faro (para enviar mensajes)
	faro FaroSender

	// ACL index: KeyID (4 bytes) → PeerKeys
	// Se usa para buscar la clave compartida al recibir un mensaje.
	aclIndex map[[4]byte]PeerKeys

	// DID → PeerKeys (para buscar por DID al enviar)
	aclByDID map[string]PeerKeys

	// Callbacks
	cb RelayCallbacks

	// Estado
	closed bool

	// Canal de mensajes entrantes (para polling)
	incomingChan chan IncomingMessage
}

// IncomingMessage es un mensaje descifrado recibido del faro.
type IncomingMessage struct {
	PeerDID     string
	DisplayName string
	Command     string
	Timestamp   int64
}

// NewRelayTransport crea un nuevo transporte relay.
//
// identity: identidad del nodo (para firmar y derivar claves).
// faro: interfaz para enviar mensajes al faro.
// aclIndex: index del ACL (KeyID → PeerKeys). Se obtiene con buildACLIndex().
// cb: callbacks (opcionales).
func NewRelayTransport(identity *crypto.Identity, faro FaroSender, aclIndex map[[4]byte]PeerKeys, cb RelayCallbacks) *RelayTransport {
	// Construir index por DID (para buscar al enviar)
	aclByDID := make(map[string]PeerKeys, len(aclIndex))
	for _, pk := range aclIndex {
		aclByDID[pk.DID] = pk
	}

	return &RelayTransport{
		identity:     identity,
		faro:         faro,
		aclIndex:     aclIndex,
		aclByDID:     aclByDID,
		cb:           cb,
		incomingChan: make(chan IncomingMessage, 100),
	}
}

// ============================================================================
// ENVIAR
// ============================================================================

// Send envía un mensaje a un peer a través del faro (relay).
//
// peerDID: DID del peer destino.
// command: el comando/mensaje a enviar (ej: "CHAT:hola mundo").
//
// Flujo:
//  1. Buscar la clave compartida del peer en el ACL.
//  2. Construir InnerPayload {from, ts, cmd, sig}.
//  3. Firmar con Ed25519.
//  4. Cifrar con ChaCha20-Poly1305.
//  5. Prepend KeyID + agregar padding.
//  6. Enviar como RELAY al faro.
func (rt *RelayTransport) Send(peerDID string, command string) error {
	rt.mu.RLock()
	if rt.closed {
		rt.mu.RUnlock()
		return fmt.Errorf("relay transport cerrado")
	}
	peer, exists := rt.aclByDID[peerDID]
	rt.mu.RUnlock()

	if !exists {
		return fmt.Errorf("peer %s no está en el ACL", peerDID[:20]+"...")
	}

	// 1. Construir InnerPayload
	inner := InnerPayload{
		FromDID: rt.identity.DID,
		TS:      time.Now().Unix(),
		Cmd:     command,
	}

	// 2. Firmar (sin el campo Sig)
	innerJSON, _ := json.Marshal(inner)
	inner.Sig = base64.StdEncoding.EncodeToString(
		rt.identity.SignMessage(innerJSON),
	)
	innerJSON, _ = json.Marshal(inner)

	// 3. Cifrar con ChaCha20-Poly1305
	ciphertext, err := crypto.EncryptPayload(peer.SharedKey, innerJSON)
	if err != nil {
		return fmt.Errorf("cifrando payload: %w", err)
	}

	// 4. Prepend KeyID
	kid := crypto.DeriveKeyID(rt.identity.PubKeyX[:])
	payload := fmt.Sprintf("%s|%s",
		hex.EncodeToString(kid[:]),
		base64.StdEncoding.EncodeToString(ciphertext),
	)

	// 5. Agregar padding (anti-DPI)
	payload = addRelayPadding(payload)

	// 6. Enviar como RELAY al faro
	relayMsg := fmt.Sprintf("RELAY %s %s %s", peerDID, rt.identity.DID, payload)
	if err := rt.faro.SendToFaro(relayMsg); err != nil {
		return fmt.Errorf("enviando RELAY al faro: %w", err)
	}

	return nil
}

// ============================================================================
// RECIBIR
// ============================================================================

// HandleIncoming procesa un mensaje entrante del faro.
//
// raw: el payload crudo recibido del faro (después de stripPadding y
// extractPayload en el listener principal).
//
// Flujo:
//  1. Separar KeyID | ciphertext.
//  2. Buscar la clave compartida en el ACL index (por KeyID).
//  3. Descifrar con ChaCha20-Poly1305.
//  4. Verificar firma Ed25519.
//  5. Verificar timestamp (anti-replay, ±60s).
//  6. Entregar al consumidor (callback + canal).
//
// Retorna true si el mensaje se procesó correctamente, false si no
// (no es un mensaje válido, no está en el ACL, firma inválida, etc.).
func (rt *RelayTransport) HandleIncoming(raw string) bool {
	rt.mu.RLock()
	if rt.closed {
		rt.mu.RUnlock()
		return false
	}
	aclIndex := rt.aclIndex
	rt.mu.RUnlock()

	// 1. Separar KeyID | ciphertext
	parts := strings.SplitN(raw, "|", 2)
	if len(parts) != 2 {
		return false
	}

	kidBytes, err := hex.DecodeString(parts[0])
	if err != nil || len(kidBytes) != 4 {
		return false
	}

	var kid [4]byte
	copy(kid[:], kidBytes)

	// 2. Buscar en el ACL index
	peer, exists := aclIndex[kid]
	if !exists {
		return false // Peer no autorizado
	}

	// 3. Descifrar
	ciphertext, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	plaintext, err := crypto.DecryptPayload(peer.SharedKey, ciphertext)
	if err != nil {
		if rt.cb.OnError != nil {
			rt.cb.OnError(peer.DID, fmt.Errorf("descifrando: %w", err))
		}
		return false
	}

	// 4. Parsear InnerPayload
	var inner InnerPayload
	if json.Unmarshal(plaintext, &inner) != nil {
		return false
	}

	// 5. Verificar firma Ed25519
	innerForVerify := inner
	innerForVerify.Sig = ""
	verifyJSON, _ := json.Marshal(innerForVerify)
	sigBytes, err := base64.StdEncoding.DecodeString(inner.Sig)
	if err != nil {
		return false
	}

	if !crypto.VerifyMessage(peer.PubKeyEd, verifyJSON, sigBytes) {
		if rt.cb.OnError != nil {
			rt.cb.OnError(peer.DID, fmt.Errorf("firma inválida"))
		}
		return false
	}

	// 6. Verificar timestamp (anti-replay, ±60s)
	if time.Now().Unix()-inner.TS > 60 {
		return false // Mensaje viejo, descartar
	}

	// 7. Resolver nombre para mostrar
	displayName := crypto.ResolveDID(peer.DID)

	// 8. Entregar al consumidor
	if rt.cb.OnMessage != nil {
		rt.cb.OnMessage(peer.DID, displayName, inner.Cmd)
	}

	// También depositar en el canal (para polling)
	msg := IncomingMessage{
		PeerDID:     peer.DID,
		DisplayName: displayName,
		Command:     inner.Cmd,
		Timestamp:   inner.TS,
	}
	select {
	case rt.incomingChan <- msg:
	default:
		// Canal lleno, descartar
	}

	return true
}

// Receive devuelve el próximo mensaje entrante (bloqueante con timeout).
func (rt *RelayTransport) Receive(timeout time.Duration) (*IncomingMessage, error) {
	select {
	case msg := <-rt.incomingChan:
		return &msg, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout esperando mensaje relay")
	}
}

// ============================================================================
// ACL (actualización en caliente)
// ============================================================================

// UpdateACL reemplaza el ACL index (cuando el usuario agrega/elimina peers).
func (rt *RelayTransport) UpdateACL(aclIndex map[[4]byte]PeerKeys) {
	aclByDID := make(map[string]PeerKeys, len(aclIndex))
	for _, pk := range aclIndex {
		aclByDID[pk.DID] = pk
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.aclIndex = aclIndex
	rt.aclByDID = aclByDID
}

// ============================================================================
// CICLO DE VIDA
// ============================================================================

// IsClosed devuelve true si el transporte está cerrado.
func (rt *RelayTransport) IsClosed() bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.closed
}

// Close cierra el transporte relay.
func (rt *RelayTransport) Close() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.closed = true
}

// ============================================================================
// PADDING (anti-DPI, mismo que Fase 1)
// ============================================================================

// addRelayPadding agrega padding aleatorio al payload (anti-DPI).
// Mismo formato que addPadding() en mobile.go y shell.go.
func addRelayPadding(payload string) string {
	randSize := make([]byte, 1)
	rand.Read(randSize)
	size := 50 + int(randSize[0])%151 // 50-200 bytes, crypto/rand

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	padding := make([]byte, size)
	for i := range padding {
		randBuf := make([]byte, 1)
		rand.Read(randBuf)
		padding[i] = charset[int(randBuf[0])%len(charset)]
	}

	return fmt.Sprintf("%s|%s", payload, string(padding))
}
