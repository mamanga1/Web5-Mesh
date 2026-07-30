package xtp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"web5-mesh/src/crypto"
)

// ============================================================================
// CONSTANTES
// ============================================================================

const (
	// PunchInterval: cada cuánto se envía un paquete de punch.
	PunchInterval = 200 * time.Millisecond

	// PunchTimeout: cuánto tiempo intentar el hole punching antes de
	// rendirse y caer a relay.
	PunchTimeout = 10 * time.Second

	// KeepaliveInterval: cada cuánto se envía un keepalive.
	KeepaliveInterval = 15 * time.Second

	// KeepaliveTimeout: si el peer no responde en este tiempo,
	// la sesión se considera muerta (3 keepalives perdidos).
	KeepaliveTimeout = 45 * time.Second

	// ReadBufferSize: tamaño del buffer de lectura del socket de punch.
	ReadBufferSize = 65536
)

// ============================================================================
// TIPOS DE PAQUETE (header de 1 byte)
// ============================================================================

type PacketType byte

const (
	PktPunch        PacketType = 0x01 // Hole punching (token + DID)
	PktNoise        PacketType = 0x02 // Handshake Noise IK
	PktData         PacketType = 0x03 // Datos cifrados (post-handshake)
	PktKeepalive    PacketType = 0x04 // Keepalive
	PktKeepaliveAck PacketType = 0x05 // Respuesta a keepalive
	PktClose        PacketType = 0x06 // Cierre de sesión
)

func (t PacketType) String() string {
	switch t {
	case PktPunch:
		return "PUNCH"
	case PktNoise:
		return "NOISE"
	case PktData:
		return "DATA"
	case PktKeepalive:
		return "KEEPALIVE"
	case PktKeepaliveAck:
		return "KEEPALIVE_ACK"
	case PktClose:
		return "CLOSE"
	default:
		return fmt.Sprintf("UNKNOWN(0x%02x)", byte(t))
	}
}

// ============================================================================
// INTERFAZ PARA ENVIAR MENSAJES AL FARO
// ============================================================================
// shell.go y mobile.go implementan esta interfaz. El DirectTransport
// la usa para enviar signaling al faro (OPEN_SESSION, PUNCH, SESSION_ACK).

type FaroSender interface {
	SendToFaro(msg string) error
}

// ============================================================================
// MENSAJES DEL FARO (ruteados por el listener principal)
// ============================================================================
// El listener principal (startNetworkListener en shell.go, runNode en
// mobile.go) deposita acá los mensajes de signaling del faro.

type FaroSignal struct {
	Type string // "SESSION_INFO", "SESSION_INCOMING", "PUNCH_NOW", "SESSION_REDIRECT"
	Raw  string // El mensaje completo del faro
}

// ============================================================================
// CALLBACKS
// ============================================================================

type DirectCallbacks struct {
	OnPunchComplete   func(peerDID string, peerAddr *net.UDPAddr)
	OnSessionActive   func(peerDID string)
	OnMessage         func(peerDID string, plaintext []byte)
	OnSessionLost     func(peerDID string)
	OnFallbackToRelay func(peerDID string)
	OnClose           func(peerDID string)
}

// ============================================================================
// DIRECT TRANSPORT
// ============================================================================

type DirectTransport struct {
	mu sync.Mutex

	// Identidad del nodo (para Noise IK). Se setea con SetIdentity().
	identity *crypto.Identity

	// DID propio
	myDID string

	// Peer
	peerDID      string
	peerEndpoint string // "ip:port" público del peer (del faro)
	peerAddr     *net.UDPAddr

	// Socket UDP dedicado para hole punching y comunicación directa.
	// Es ListenUDP (no DialUDP) para recibir de múltiples fuentes.
	conn *net.UDPConn

	// Sesión Noise IK
	session *Session

	// FSM del transporte
	fsm *FSM

	// Faro
	faro         FaroSender
	FaroMessages chan FaroSignal

	// Callbacks
	cb DirectCallbacks

	// Token de punch (aleatorio, identifica esta sesión de punch)
	punchToken string

	// Estado interno
	punching bool
	active   bool
	closed   bool
	lastRecv time.Time
	quit     chan struct{}
	quitOnce sync.Once

	// Canal de mensajes descifrados (para polling)
	dataChan chan []byte
}

// NewDirectTransport crea un nuevo transporte directo.
func NewDirectTransport(myDID string, fsm *FSM, faro FaroSender, cb DirectCallbacks) *DirectTransport {
	return &DirectTransport{
		myDID:        myDID,
		fsm:          fsm,
		faro:         faro,
		cb:           cb,
		FaroMessages: make(chan FaroSignal, 32),
		dataChan:     make(chan []byte, 100),
		quit:         make(chan struct{}),
	}
}

// SetIdentity configura la identidad del nodo (necesaria para Noise IK).
// DEBE llamarse ANTES de OpenSession o HandleIncomingSession.
func (dt *DirectTransport) SetIdentity(identity *crypto.Identity) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.identity = identity
}

// ============================================================================
// ABRIR SESIÓN (signaling con el faro)
// ============================================================================

// OpenSession inicia una sesión directa con un peer.
//
// Flujo:
//  1. Abrir socket de punch.
//  2. Enviar OPEN_SESSION al faro.
//  3. Esperar SESSION_INFO (endpoints del peer).
//  4. Hole punching UDP.
//  5. Si funciona: handshake Noise IK → sesión directa.
//  6. Si falla: fallback a relay.
//
// peerDID: DID del peer.
// peerPubX: clave pública X25519 del peer (del ACL).
func (dt *DirectTransport) OpenSession(peerDID string, peerPubX *[32]byte) error {
	dt.mu.Lock()
	if dt.closed {
		dt.mu.Unlock()
		return fmt.Errorf("transporte cerrado")
	}
	if dt.identity == nil {
		dt.mu.Unlock()
		return fmt.Errorf("identidad no configurada (llamar SetIdentity antes)")
	}
	dt.peerDID = peerDID
	dt.mu.Unlock()

	dt.fsm.SetPeerDID(peerDID)

	// 1. Abrir socket UDP dedicado para punch
	if err := dt.openPunchSocket(); err != nil {
		return fmt.Errorf("abriendo socket de punch: %w", err)
	}

	// 2. Generar token de punch
	tokenBytes := make([]byte, 16)
	rand.Read(tokenBytes)
	dt.mu.Lock()
	dt.punchToken = hex.EncodeToString(tokenBytes)
	dt.mu.Unlock()

	// 3. Enviar OPEN_SESSION al faro
	dt.fsm.Send(EvDiscoverPeer, map[string]interface{}{"peer": peerDID})

	openMsg := fmt.Sprintf("OPEN_SESSION %s %s", peerDID, dt.myDID)
	if err := dt.faro.SendToFaro(openMsg); err != nil {
		return fmt.Errorf("enviando OPEN_SESSION: %w", err)
	}

	fmt.Printf("[XTP] 📤 OPEN_SESSION → %s\n", peerDID[:20]+"...")

	// 4. Esperar SESSION_INFO del faro (timeout 10s)
	sessionInfo, err := dt.waitForFaroSignal("SESSION_INFO", 10*time.Second)
	if err != nil {
		dt.fsm.Send(EvPeerNotFound, map[string]interface{}{"peer": peerDID})
		if dt.cb.OnFallbackToRelay != nil {
			dt.cb.OnFallbackToRelay(peerDID)
		}
		return fmt.Errorf("esperando SESSION_INFO: %w", err)
	}

	// Parsear SESSION_INFO: "SESSION_INFO <targetDID> <targetEndpoint> <myEndpoint>"
	parts := strings.Fields(sessionInfo)
	if len(parts) < 3 {
		return fmt.Errorf("SESSION_INFO inválido: %s", sessionInfo)
	}

	dt.mu.Lock()
	dt.peerEndpoint = parts[2]
	dt.mu.Unlock()

	peerAddr, err := net.ResolveUDPAddr("udp", parts[2])
	if err != nil {
		return fmt.Errorf("resolviendo endpoint del peer %s: %w", parts[2], err)
	}

	dt.mu.Lock()
	dt.peerAddr = peerAddr
	dt.mu.Unlock()

	dt.fsm.Send(EvPeerFound, map[string]interface{}{
		"peer":     peerDID,
		"endpoint": parts[2],
	})

	fmt.Printf("[XTP] 📥 SESSION_INFO: peer en %s\n", parts[2])

	// 5. Iniciar hole punching
	return dt.punch(peerPubX)
}

// HandleIncomingSession procesa un SESSION_INCOMING del faro.
// El faro avisa que otro nodo quiere hablar con nosotros (somos el respondedor).
//
// raw: el mensaje completo del faro ("SESSION_INCOMING <senderDID> <senderEndpoint>").
func (dt *DirectTransport) HandleIncomingSession(raw string) error {
	dt.mu.Lock()
	if dt.identity == nil {
		dt.mu.Unlock()
		return fmt.Errorf("identidad no configurada (llamar SetIdentity antes)")
	}
	dt.mu.Unlock()

	// Parsear: "SESSION_INCOMING <senderDID> <senderEndpoint>"
	parts := strings.Fields(raw)
	if len(parts) < 3 {
		return fmt.Errorf("SESSION_INCOMING inválido: %s", raw)
	}
	senderDID := parts[1]
	senderEndpoint := parts[2]

	if senderEndpoint == "ws" {
		// El peer está en WSS, no podemos hacer hole punching UDP.
		fmt.Printf("[XTP] ⚠️ Peer %s está en WSS, usando relay\n", senderDID[:20]+"...")
		if dt.cb.OnFallbackToRelay != nil {
			dt.cb.OnFallbackToRelay(senderDID)
		}
		return nil
	}

	senderAddr, err := net.ResolveUDPAddr("udp", senderEndpoint)
	if err != nil {
		return fmt.Errorf("resolviendo endpoint del sender %s: %w", senderEndpoint, err)
	}

	dt.mu.Lock()
	dt.peerDID = senderDID
	dt.peerEndpoint = senderEndpoint
	dt.peerAddr = senderAddr
	dt.mu.Unlock()

	dt.fsm.SetPeerDID(senderDID)
	dt.fsm.Send(EvPeerFound, map[string]interface{}{
		"peer":     senderDID,
		"endpoint": senderEndpoint,
	})

	fmt.Printf("[XTP] 📥 SESSION_INCOMING: %s desde %s\n", senderDID[:20]+"...", senderEndpoint)

	// Abrir socket de punch
	if err := dt.openPunchSocket(); err != nil {
		return fmt.Errorf("abriendo socket de punch: %w", err)
	}

	// Generar token de punch
	tokenBytes := make([]byte, 16)
	rand.Read(tokenBytes)
	dt.mu.Lock()
	dt.punchToken = hex.EncodeToString(tokenBytes)
	dt.punching = true
	dt.mu.Unlock()

	// Como respondedor: escuchar paquetes del iniciador y responder con punch.
	go dt.readLoop()
	go dt.sendPunchPackets()

	// Esperar a que el hole punching funcione y el Noise handshake se complete.
	// El readLoop procesa los paquetes entrantes y dispara las transiciones.
	go func() {
		deadline := time.After(PunchTimeout + HandshakeTimeout)
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-dt.quit:
				return
			case <-deadline:
				dt.mu.Lock()
				active := dt.active
				dt.punching = false
				dt.mu.Unlock()
				if !active {
					fmt.Printf("[XTP] ❌ Timeout esperando sesión directa con %s\n", senderDID[:20]+"...")
					dt.fsm.Send(EvPunchFailed, map[string]interface{}{"peer": senderDID})
					if dt.cb.OnFallbackToRelay != nil {
						dt.cb.OnFallbackToRelay(senderDID)
					}
				}
				return
			case <-ticker.C:
				dt.mu.Lock()
				active := dt.active
				dt.mu.Unlock()
				if active {
					return // Sesión activa, listo
				}
			}
		}
	}()

	return nil
}

// ============================================================================
// SOCKET DE PUNCH
// ============================================================================

func (dt *DirectTransport) openPunchSocket() error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if dt.conn != nil {
		return nil // Ya abierto
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return err
	}

	conn.SetReadBuffer(ReadBufferSize)
	conn.SetWriteBuffer(ReadBufferSize)

	dt.conn = conn
	fmt.Printf("[XTP] 🔌 Socket de punch abierto en %s\n", conn.LocalAddr().String())
	return nil
}

// ============================================================================
// HOLE PUNCHING
// ============================================================================

func (dt *DirectTransport) punch(peerPubX *[32]byte) error {
	dt.mu.Lock()
	dt.punching = true
	dt.mu.Unlock()

	dt.fsm.Send(EvStartPunch, map[string]interface{}{"peer": dt.peerDID})

	// Iniciar readLoop y sendPunchPackets
	go dt.readLoop()
	go dt.sendPunchPackets()

	// Avisar al faro que estamos haciendo punch
	dt.mu.Lock()
	localAddr := ""
	if dt.conn != nil {
		localAddr = dt.conn.LocalAddr().String()
	}
	dt.mu.Unlock()

	punchMsg := fmt.Sprintf("PUNCH %s %s %s", dt.peerDID, dt.myDID, localAddr)
	dt.faro.SendToFaro(punchMsg)

	// Esperar a que el hole punching funcione o timeout
	deadline := time.After(PunchTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-dt.quit:
			return fmt.Errorf("transporte cerrado durante punch")

		case <-deadline:
			dt.mu.Lock()
			dt.punching = false
			dt.mu.Unlock()

			dt.fsm.Send(EvPunchFailed, map[string]interface{}{"peer": dt.peerDID})
			fmt.Printf("[XTP] ❌ Hole punching falló con %s (timeout %s)\n",
				dt.peerDID[:20]+"...", PunchTimeout)

			if dt.cb.OnFallbackToRelay != nil {
				dt.cb.OnFallbackToRelay(dt.peerDID)
			}
			return fmt.Errorf("hole punching timeout")

		case <-ticker.C:
			dt.mu.Lock()
			punching := dt.punching
			dt.mu.Unlock()

			if !punching {
				// El readLoop detectó un paquete del peer
				fmt.Printf("[XTP] ✅ Hole punching exitoso con %s\n", dt.peerDID[:20]+"...")

				dt.fsm.Send(EvPunchComplete, map[string]interface{}{
					"peer": dt.peerDID,
					"addr": dt.peerAddr.String(),
				})

				if dt.cb.OnPunchComplete != nil {
					dt.cb.OnPunchComplete(dt.peerDID, dt.peerAddr)
				}

				// Iniciar handshake Noise IK
				return dt.noiseHandshake(peerPubX)
			}
		}
	}
}

func (dt *DirectTransport) sendPunchPackets() {
	ticker := time.NewTicker(PunchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dt.quit:
			return
		case <-ticker.C:
			dt.mu.Lock()
			punching := dt.punching
			peerAddr := dt.peerAddr
			conn := dt.conn
			token := dt.punchToken
			myDID := dt.myDID
			dt.mu.Unlock()

			if !punching || peerAddr == nil || conn == nil {
				return
			}

			pkt := buildPunchPacket(token, myDID)
			conn.WriteToUDP(pkt, peerAddr)
		}
	}
}

func buildPunchPacket(token, did string) []byte {
	data := []byte{byte(PktPunch)}
	data = append(data, []byte(token)...)
	data = append(data, '|')
	data = append(data, []byte(did)...)
	return data
}

// ============================================================================
// READ LOOP
// ============================================================================

func (dt *DirectTransport) readLoop() {
	buf := make([]byte, ReadBufferSize)

	for {
		select {
		case <-dt.quit:
			return
		default:
		}

		dt.mu.Lock()
		conn := dt.conn
		dt.mu.Unlock()

		if conn == nil {
			return
		}

		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return // Socket cerrado
		}

		if n < 1 {
			continue
		}

		pktType := PacketType(buf[0])
		payload := make([]byte, n-1)
		copy(payload, buf[1:n])

		dt.mu.Lock()
		dt.lastRecv = time.Now()
		dt.mu.Unlock()

		switch pktType {
		case PktPunch:
			dt.handlePunchPacket(payload, remoteAddr)
		case PktNoise:
			dt.handleNoisePacket(payload)
		case PktData:
			dt.handleDataPacket(payload)
		case PktKeepalive:
			dt.handleKeepalive(remoteAddr)
		case PktKeepaliveAck:
			// lastRecv ya se actualizó arriba
		case PktClose:
			fmt.Printf("[XTP] 🔒 Peer %s cerró la sesión\n", dt.peerDID[:20]+"...")
			dt.Close()
			return
		}
	}
}

func (dt *DirectTransport) handlePunchPacket(payload []byte, remoteAddr *net.UDPAddr) {
	parts := strings.SplitN(string(payload), "|", 2)
	if len(parts) != 2 {
		return
	}
	peerDID := parts[1]

	dt.mu.Lock()
	expectedPeer := dt.peerDID
	wasPunching := dt.punching
	conn := dt.conn
	token := dt.punchToken
	myDID := dt.myDID
	dt.mu.Unlock()

	if expectedPeer != "" && peerDID != expectedPeer {
		return
	}

	if wasPunching {
		dt.mu.Lock()
		dt.punching = false
		dt.peerAddr = remoteAddr
		dt.mu.Unlock()

		fmt.Printf("[XTP] 👊 Punch recibido de %s (%s)\n",
			peerDID[:20]+"...", remoteAddr.String())

		// Responder con un punch (abrir el NAT del peer)
		if conn != nil {
			pkt := buildPunchPacket(token, myDID)
			conn.WriteToUDP(pkt, remoteAddr)
		}
	}
}

// ============================================================================
// NOISE IK HANDSHAKE (directo, sin faro)
// ============================================================================

func (dt *DirectTransport) noiseHandshake(peerPubX *[32]byte) error {
	dt.fsm.Send(EvStartNoise, map[string]interface{}{"peer": dt.peerDID})

	dt.mu.Lock()
	identity := dt.identity
	dt.mu.Unlock()

	if identity == nil {
		dt.fsm.Send(EvNoiseFailed, map[string]interface{}{"peer": dt.peerDID})
		return fmt.Errorf("identidad no configurada")
	}

	// El que llamó a OpenSession es el iniciador.
	// El que recibió HandleIncomingSession es el respondedor.
	isInitiator := true
	dt.mu.Lock()
	if dt.peerEndpoint == "" {
		isInitiator = false
	}
	dt.mu.Unlock()

	session, err := NewSession(isInitiator, identity, dt.peerDID, peerPubX)
	if err != nil {
		dt.fsm.Send(EvNoiseFailed, map[string]interface{}{"peer": dt.peerDID, "err": err})
		return fmt.Errorf("creando sesión Noise IK: %w", err)
	}

	dt.mu.Lock()
	dt.session = session
	dt.mu.Unlock()

	if isInitiator {
		// Iniciador: enviar mensaje 1
		msg, err := session.InitiatorMessage()
		if err != nil {
			dt.fsm.Send(EvNoiseFailed, map[string]interface{}{"peer": dt.peerDID, "err": err})
			return fmt.Errorf("generando mensaje de handshake: %w", err)
		}

		pkt := append([]byte{byte(PktNoise)}, msg...)

		dt.mu.Lock()
		conn := dt.conn
		peerAddr := dt.peerAddr
		dt.mu.Unlock()

		if conn == nil || peerAddr == nil {
			return fmt.Errorf("socket o dirección del peer no disponible")
		}

		if _, err := conn.WriteToUDP(pkt, peerAddr); err != nil {
			dt.fsm.Send(EvNoiseFailed, map[string]interface{}{"peer": dt.peerDID, "err": err})
			return fmt.Errorf("enviando handshake Noise: %w", err)
		}

		fmt.Printf("[XTP] 📤 Noise IK msg 1 → %s\n", dt.peerDID[:20]+"...")
	}

	// Esperar a que la sesión se active (el readLoop procesa los mensajes)
	deadline := time.After(HandshakeTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-dt.quit:
			return fmt.Errorf("transporte cerrado durante Noise handshake")
		case <-deadline:
			dt.fsm.Send(EvNoiseFailed, map[string]interface{}{"peer": dt.peerDID})
			if dt.cb.OnFallbackToRelay != nil {
				dt.cb.OnFallbackToRelay(dt.peerDID)
			}
			return fmt.Errorf("Noise handshake timeout")
		case <-ticker.C:
			if session.IsActive() {
				dt.onNoiseComplete()
				return nil
			}
		}
	}
}

func (dt *DirectTransport) handleNoisePacket(payload []byte) {
	dt.mu.Lock()
	session := dt.session
	conn := dt.conn
	peerAddr := dt.peerAddr
	identity := dt.identity
	peerDID := dt.peerDID
	dt.mu.Unlock()

	// Si no hay sesión creada, crearla como respondedor
	if session == nil {
		if identity == nil {
			fmt.Printf("[XTP] ⚠️ Noise recibido sin identidad configurada\n")
			return
		}

		var err error
		session, err = NewSession(false, identity, peerDID, nil)
		if err != nil {
			fmt.Printf("[XTP] ❌ Error creando sesión Noise (respondedor): %v\n", err)
			return
		}

		dt.mu.Lock()
		dt.session = session
		dt.mu.Unlock()
	}

	response, completed, err := session.HandleMessage(payload)
	if err != nil {
		fmt.Printf("[XTP] ❌ Error en Noise handshake: %v\n", err)
		return
	}

	if response != nil && conn != nil && peerAddr != nil {
		pkt := append([]byte{byte(PktNoise)}, response...)
		conn.WriteToUDP(pkt, peerAddr)
		fmt.Printf("[XTP] 📤 Noise IK msg 2 → %s\n", peerDID[:20]+"...")
	}

	if completed {
		fmt.Printf("[XTP] ✅ Noise IK completo con %s\n", peerDID[:20]+"...")
	}
}

func (dt *DirectTransport) onNoiseComplete() {
	dt.mu.Lock()
	dt.active = true
	peerDID := dt.peerDID
	myDID := dt.myDID
	dt.mu.Unlock()

	dt.fsm.Send(EvNoiseComplete, map[string]interface{}{"peer": peerDID})

	// Avisar al faro que la sesión directa está activa
	ackMsg := fmt.Sprintf("SESSION_ACK %s %s", peerDID, myDID)
	dt.faro.SendToFaro(ackMsg)

	fmt.Printf("[XTP] 🔐 Sesión directa activa con %s (Noise IK)\n", peerDID[:20]+"...")

	if dt.cb.OnSessionActive != nil {
		dt.cb.OnSessionActive(peerDID)
	}

	go dt.keepaliveLoop()
}

// ============================================================================
// COMUNICACIÓN DIRECTA (post-handshake)
// ============================================================================

// Send envía un mensaje cifrado al peer por la conexión directa.
func (dt *DirectTransport) Send(plaintext []byte) error {
	dt.mu.Lock()
	session := dt.session
	conn := dt.conn
	peerAddr := dt.peerAddr
	active := dt.active
	dt.mu.Unlock()

	if !active || session == nil || conn == nil || peerAddr == nil {
		return fmt.Errorf("sesión directa no activa")
	}

	ciphertext, err := session.Encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("cifrando: %w", err)
	}

	pkt := make([]byte, 1+len(ciphertext))
	pkt[0] = byte(PktData)
	copy(pkt[1:], ciphertext)

	if _, err := conn.WriteToUDP(pkt, peerAddr); err != nil {
		return fmt.Errorf("enviando datos: %w", err)
	}

	return nil
}

func (dt *DirectTransport) handleDataPacket(payload []byte) {
	dt.mu.Lock()
	session := dt.session
	peerDID := dt.peerDID
	dt.mu.Unlock()

	if session == nil || !session.IsActive() {
		fmt.Printf("[XTP] ⚠️ Datos recibidos sin sesión activa\n")
		return
	}

	plaintext, err := session.Decrypt(payload)
	if err != nil {
		fmt.Printf("[XTP] ❌ Error descifrando datos: %v\n", err)
		return
	}

	if dt.cb.OnMessage != nil {
		dt.cb.OnMessage(peerDID, plaintext)
	}

	select {
	case dt.dataChan <- plaintext:
	default:
		fmt.Printf("[XTP] ⚠️ Canal de datos lleno, mensaje descartado\n")
	}
}

// Receive devuelve el próximo mensaje descifrado del peer (bloqueante con timeout).
func (dt *DirectTransport) Receive(timeout time.Duration) ([]byte, error) {
	select {
	case msg := <-dt.dataChan:
		return msg, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout esperando mensaje")
	case <-dt.quit:
		return nil, fmt.Errorf("transporte cerrado")
	}
}

// ============================================================================
// KEEPALIVE
// ============================================================================

func (dt *DirectTransport) keepaliveLoop() {
	ticker := time.NewTicker(KeepaliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-dt.quit:
			return
		case <-ticker.C:
			dt.mu.Lock()
			active := dt.active
			conn := dt.conn
			peerAddr := dt.peerAddr
			lastRecv := dt.lastRecv
			peerDID := dt.peerDID
			dt.mu.Unlock()

			if !active || conn == nil || peerAddr == nil {
				return
			}

			if time.Since(lastRecv) > KeepaliveTimeout {
				fmt.Printf("[XTP] 💀 Peer %s no responde hace %s, sesión muerta\n",
					peerDID[:20]+"...", time.Since(lastRecv).Round(time.Second))

				dt.mu.Lock()
				dt.active = false
				dt.mu.Unlock()

				dt.fsm.Send(EvKeepaliveTimeout, map[string]interface{}{"peer": peerDID})

				if dt.cb.OnSessionLost != nil {
					dt.cb.OnSessionLost(peerDID)
				}
				return
			}

			pkt := []byte{byte(PktKeepalive)}
			conn.WriteToUDP(pkt, peerAddr)
			dt.fsm.Send(EvKeepaliveTick, map[string]interface{}{"peer": peerDID})
		}
	}
}

func (dt *DirectTransport) handleKeepalive(remoteAddr *net.UDPAddr) {
	dt.mu.Lock()
	conn := dt.conn
	dt.mu.Unlock()

	if conn == nil {
		return
	}

	pkt := []byte{byte(PktKeepaliveAck)}
	conn.WriteToUDP(pkt, remoteAddr)
}

// ============================================================================
// CICLO DE VIDA
// ============================================================================

func (dt *DirectTransport) IsActive() bool {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.active
}

func (dt *DirectTransport) PeerDID() string {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.peerDID
}

func (dt *DirectTransport) PeerAddr() *net.UDPAddr {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.peerAddr
}

func (dt *DirectTransport) Session() *Session {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	return dt.session
}

// Close cierra el transporte directo.
func (dt *DirectTransport) Close() {
	dt.quitOnce.Do(func() {
		close(dt.quit)
	})

	dt.mu.Lock()
	defer dt.mu.Unlock()

	if dt.closed {
		return
	}
	dt.closed = true
	dt.active = false
	dt.punching = false

	// Enviar paquete de cierre al peer
	if dt.conn != nil && dt.peerAddr != nil {
		pkt := []byte{byte(PktClose)}
		dt.conn.WriteToUDP(pkt, dt.peerAddr)
	}

	// Cerrar sesión Noise IK
	if dt.session != nil {
		dt.session.Close()
	}

	// Cerrar socket
	if dt.conn != nil {
		dt.conn.Close()
		dt.conn = nil
	}

	// Avisar al faro
	if dt.faro != nil && dt.peerDID != "" {
		closeMsg := fmt.Sprintf("CLOSE_SESSION %s %s", dt.peerDID, dt.myDID)
		dt.faro.SendToFaro(closeMsg)
	}

	dt.fsm.Send(EvCloseSession, map[string]interface{}{"peer": dt.peerDID})

	if dt.cb.OnClose != nil {
		dt.cb.OnClose(dt.peerDID)
	}

	fmt.Printf("[XTP] 🔒 Transporte directo cerrado con %s\n", dt.peerDID[:20]+"...")
}

// ============================================================================
// UTILIDADES INTERNAS
// ============================================================================

func (dt *DirectTransport) waitForFaroSignal(signalType string, timeout time.Duration) (string, error) {
	deadline := time.After(timeout)
	for {
		select {
		case sig := <-dt.FaroMessages:
			if sig.Type == signalType {
				return sig.Raw, nil
			}
		case <-deadline:
			return "", fmt.Errorf("timeout esperando %s del faro", signalType)
		case <-dt.quit:
			return "", fmt.Errorf("transporte cerrado")
		}
	}
}
