package xtp

import (
	"fmt"
	"net"
        "sync"
	
        "web5-mesh/src/crypto"
)

// ============================================================================
// CALLBACKS DEL TRANSPORT MANAGER
// ============================================================================
// Shell.go y mobile.go registran estos callbacks para recibir
// notificaciones del transporte (mensajes, cambios de estado, etc.).

type ManagerCallbacks struct {
	// OnMessage: se llama cuando se recibe un mensaje de un peer
	// (ya sea por conexión directa o por relay).
	OnMessage func(peerDID string, displayName string, command string)

	// OnDirectSessionActive: se llama cuando se establece una sesión
	// directa con un peer (Noise IK completo, forward secrecy activo).
	OnDirectSessionActive func(peerDID string)

	// OnDirectSessionLost: se llama cuando se pierde una sesión directa
	// (keepalive timeout, peer se desconectó).
	OnDirectSessionLost func(peerDID string)

	// OnFallbackToRelay: se llama cuando el hole punching falla y
	// se cae a relay (sin forward secrecy).
	OnFallbackToRelay func(peerDID string)

	// OnStateChange: se llama cuando el FSM cambia de estado.
	OnStateChange func(from, to State, event Event)

	// OnError: se llama cuando hay un error.
	OnError func(context string, err error)
}

// ============================================================================
// TRANSPORT MANAGER
// ============================================================================

// TransportManager orquesta todos los componentes del transporte XTP:
//
//   - FSM (state.go): máquina de estados del transporte.
//   - DirectTransport (direct.go): hole punching + Noise IK (forward secrecy).
//   - RelayTransport (relay.go): fallback a través del faro (Fase 1).
//   - SessionManager (session.go): sesiones Noise IK.
//
// Shell.go y mobile.go interactúan SOLO con el TransportManager.
// No necesitan saber si un mensaje va por conexión directa o por relay.
//
// Flujo de envío:
//
//	Send(peerDID, "CHAT:hola")
//	  → ¿Hay sesión directa activa con peerDID?
//	    → SÍ: DirectTransport.Send() (Noise IK, forward secrecy)
//	    → NO: ¿Podemos establecer sesión directa?
//	      → SÍ: OpenSession() → hole punching → Noise IK → Send()
//	      → NO: RelayTransport.Send() (fallback, sin forward secrecy)
//
// Flujo de recepción:
//
//	HandleIncoming(raw)  [mensaje del faro]
//	  → ¿Es signaling? (SESSION_INFO, SESSION_INCOMING, PUNCH_NOW)
//	    → SÍ: rutear al DirectTransport
//	    → NO: RelayTransport.HandleIncoming() (descifrar Fase 1)
type TransportManager struct {
	mu sync.RWMutex

	// Identidad del nodo
	identity *crypto.Identity

	// Faro (para enviar mensajes)
	faro FaroSender

	// FSM del transporte
	fsm *FSM

	// Transportes
	relay  *RelayTransport
	direct map[string]*DirectTransport // peerDID → DirectTransport

	// ACL index (para relay y para buscar claves X25519)
	aclIndex map[[4]byte]PeerKeys
	aclByDID map[string]PeerKeys

	// Callbacks
	cb ManagerCallbacks

	// Estado
	closed bool

	// Configuración
	autoDirect bool // Intentar conexión directa automáticamente (default: true)
}

// ManagerConfig configura el TransportManager.
type ManagerConfig struct {
	// AutoDirect: si es true, el manager intenta establecer una conexión
	// directa (hole punching + Noise IK) antes de usar relay.
	// Si es false, siempre usa relay (útil para testing o redes donde
	// el hole punching no funciona).
	AutoDirect bool
}

// DefaultManagerConfig devuelve la configuración por defecto.
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		AutoDirect: true,
	}
}

// NewTransportManager crea un nuevo TransportManager.
//
// identity: identidad del nodo (para Noise IK y firmas).
// faro: interfaz para enviar mensajes al faro.
// aclIndex: index del ACL (KeyID → PeerKeys).
// cb: callbacks para notificar a la UI.
// config: configuración (AutoDirect, etc.).
func NewTransportManager(
	identity *crypto.Identity,
	faro FaroSender,
	aclIndex map[[4]byte]PeerKeys,
	cb ManagerCallbacks,
	config ManagerConfig,
) *TransportManager {
	// Construir index por DID
	aclByDID := make(map[string]PeerKeys, len(aclIndex))
	for _, pk := range aclIndex {
		aclByDID[pk.DID] = pk
	}

	// Crear FSM
	fsm := NewFSM()

	// Crear RelayTransport
	relayCb := RelayCallbacks{
		OnMessage: func(peerDID, displayName, command string) {
			if cb.OnMessage != nil {
				cb.OnMessage(peerDID, displayName, command)
			}
		},
		OnError: func(peerDID string, err error) {
			if cb.OnError != nil {
				cb.OnError("relay:"+peerDID[:15], err)
			}
		},
	}
	relay := NewRelayTransport(identity, faro, aclIndex, relayCb)

	tm := &TransportManager{
		identity:   identity,
		faro:       faro,
		fsm:        fsm,
		relay:      relay,
		direct:     make(map[string]*DirectTransport),
		aclIndex:   aclIndex,
		aclByDID:   aclByDID,
		cb:         cb,
		autoDirect: config.AutoDirect,
	}

	// Registrar callback de cambio de estado del FSM
	fsm.OnEnter(Direct, func(from, to State, event Event, meta map[string]interface{}) {
		if cb.OnStateChange != nil {
			cb.OnStateChange(from, to, event)
		}
	})
	fsm.OnEnter(RelayFallback, func(from, to State, event Event, meta map[string]interface{}) {
		if cb.OnStateChange != nil {
			cb.OnStateChange(from, to, event)
		}
	})

	return tm
}

// ============================================================================
// ENVIAR MENSAJES
// ============================================================================

// Send envía un mensaje a un peer. Decide automáticamente si usar
// conexión directa (Noise IK) o relay (fallback Fase 1).
//
// peerDID: DID del peer destino.
// command: el comando/mensaje (ej: "CHAT:hola mundo").
//
// Retorna el tipo de transporte usado ("direct" o "relay") y error.
func (tm *TransportManager) Send(peerDID string, command string) (transport string, err error) {
	tm.mu.RLock()
	if tm.closed {
		tm.mu.RUnlock()
		return "", fmt.Errorf("transport manager cerrado")
	}
	direct, hasDirect := tm.direct[peerDID]
	autoDirect := tm.autoDirect
	tm.mu.RUnlock()

	// 1. Si hay sesión directa activa, usarla
	if hasDirect && direct.IsActive() {
		if err := direct.Send([]byte(command)); err != nil {
			// La sesión directa falló, caer a relay
			if tm.cb.OnError != nil {
				tm.cb.OnError("direct:"+peerDID[:15], fmt.Errorf("envío directo falló, cayendo a relay: %w", err))
			}
			// No retornar error, intentar relay abajo
		} else {
			return "direct", nil
		}
	}

	// 2. Si AutoDirect está activado y no hay sesión directa, intentar una
	if autoDirect && !hasDirect {
		peer, hasPeer := tm.getPeerKeys(peerDID)
		if hasPeer {
			peerPubX := new([32]byte)
			copy(peerPubX[:], peer.PubKeyEd[:32]) // X25519 pub key

			// Intentar establecer sesión directa en background
			go tm.tryDirectSession(peerDID, peerPubX)

			// Mientras tanto, enviar por relay (no bloquear al usuario)
			if err := tm.relay.Send(peerDID, command); err != nil {
				return "", fmt.Errorf("enviando por relay: %w", err)
			}
			return "relay", nil
		}
	}

	// 3. Fallback: relay
	if err := tm.relay.Send(peerDID, command); err != nil {
		return "", fmt.Errorf("enviando por relay: %w", err)
	}
	return "relay", nil
}

// tryDirectSession intenta establecer una sesión directa con un peer.
// Se ejecuta en background (goroutine). Si funciona, los mensajes
// futuros usan la conexión directa. Si falla, se queda en relay.
func (tm *TransportManager) tryDirectSession(peerDID string, peerPubX *[32]byte) {
	tm.mu.Lock()
	if tm.closed {
		tm.mu.Unlock()
		return
	}
	// Verificar si ya hay una sesión directa (puede haberse creado
	// mientras esperábamos el lock)
	if _, exists := tm.direct[peerDID]; exists {
		tm.mu.Unlock()
		return
	}

	// Crear DirectTransport
	dtCb := DirectCallbacks{
		OnPunchComplete: func(peerDID string, peerAddr *net.UDPAddr) {
			fmt.Printf("[XTP-MGR] 👊 Hole punching exitoso con %s\n", peerDID[:20]+"...")
		},
		OnSessionActive: func(peerDID string) {
			fmt.Printf("[XTP-MGR] 🔐 Sesión directa activa con %s\n", peerDID[:20]+"...")
			if tm.cb.OnDirectSessionActive != nil {
				tm.cb.OnDirectSessionActive(peerDID)
			}
		},
		OnMessage: func(peerDID string, plaintext []byte) {
			// Mensaje recibido por conexión directa
			displayName := crypto.ResolveDID(peerDID)
			if tm.cb.OnMessage != nil {
				tm.cb.OnMessage(peerDID, displayName, string(plaintext))
			}
		},
		OnSessionLost: func(peerDID string) {
			fmt.Printf("[XTP-MGR] 💀 Sesión directa perdida con %s\n", peerDID[:20]+"...")
			if tm.cb.OnDirectSessionLost != nil {
				tm.cb.OnDirectSessionLost(peerDID)
			}
			// Limpiar la sesión directa
			tm.mu.Lock()
			delete(tm.direct, peerDID)
			tm.mu.Unlock()
		},
		OnFallbackToRelay: func(peerDID string) {
			fmt.Printf("[XTP-MGR] 🔄 Fallback a relay con %s\n", peerDID[:20]+"...")
			if tm.cb.OnFallbackToRelay != nil {
				tm.cb.OnFallbackToRelay(peerDID)
			}
			// Limpiar la sesión directa fallida
			tm.mu.Lock()
			delete(tm.direct, peerDID)
			tm.mu.Unlock()
		},
		OnClose: func(peerDID string) {
			tm.mu.Lock()
			delete(tm.direct, peerDID)
			tm.mu.Unlock()
		},
	}

	dt := NewDirectTransport(tm.identity.DID, tm.fsm, tm.faro, dtCb)
	dt.SetIdentity(tm.identity)
	tm.direct[peerDID] = dt
	tm.mu.Unlock()

	// Intentar abrir sesión directa
	if err := dt.OpenSession(peerDID, peerPubX); err != nil {
		fmt.Printf("[XTP-MGR] ❌ Sesión directa falló con %s: %v\n", peerDID[:20]+"...", err)
		tm.mu.Lock()
		delete(tm.direct, peerDID)
		tm.mu.Unlock()
	}
}

// ============================================================================
// RECIBIR MENSAJES
// ============================================================================

// HandleIncoming procesa un mensaje entrante del faro.
// El listener principal (shell.go / mobile.go) llama a este método
// para CADA mensaje que recibe del faro.
//
// El manager decide si es signaling (para DirectTransport) o datos
// (para RelayTransport).
//
// raw: el mensaje crudo del faro (después de stripPadding y extractPayload).
//
// Retorna true si el mensaje se procesó, false si no se reconoció.
func (tm *TransportManager) HandleIncoming(raw string) bool {
	tm.mu.RLock()
	if tm.closed {
		tm.mu.RUnlock()
		return false
	}
	tm.mu.RUnlock()

	// 1. Verificar si es signaling del faro
	if tm.isFaroSignal(raw) {
		return tm.handleFaroSignal(raw)
	}

	// 2. Verificar si es un paquete de transporte directo
	// (los paquetes directos tienen un header de 1 byte: PktData, PktNoise, etc.)
	// Estos NO vienen del faro, vienen del socket de punch del DirectTransport.
	// El listener principal no los ve (los maneja el readLoop del DirectTransport).
	// Así que acá solo manejamos mensajes del faro.

	// 3. Es un mensaje de relay (datos cifrados de Fase 1)
	return tm.relay.HandleIncoming(raw)
}

// isFaroSignal verifica si un mensaje es signaling del faro.
func (tm *TransportManager) isFaroSignal(raw string) bool {
	signalPrefixes := []string{
		"SESSION_INFO ",
		"SESSION_INCOMING ",
		"PUNCH_NOW ",
		"SESSION_REDIRECT ",
		"SESSION_ERROR ",
	}
	for _, prefix := range signalPrefixes {
		if len(raw) >= len(prefix) && raw[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// handleFaroSignal procesa un mensaje de signaling del faro.
func (tm *TransportManager) handleFaroSignal(raw string) bool {
	// Determinar el tipo de señal
	var signalType string
	if len(raw) >= 13 && raw[:13] == "SESSION_INFO " {
		signalType = "SESSION_INFO"
	} else if len(raw) >= 17 && raw[:17] == "SESSION_INCOMING " {
		signalType = "SESSION_INCOMING"
	} else if len(raw) >= 10 && raw[:10] == "PUNCH_NOW " {
		signalType = "PUNCH_NOW"
	} else if len(raw) >= 17 && raw[:17] == "SESSION_REDIRECT " {
		signalType = "SESSION_REDIRECT"
	} else if len(raw) >= 14 && raw[:14] == "SESSION_ERROR " {
		signalType = "SESSION_ERROR"
	} else {
		return false
	}

	// Para SESSION_INCOMING: crear un DirectTransport como respondedor
	if signalType == "SESSION_INCOMING" {
		// Extraer senderDID del mensaje
		parts := splitFields(raw)
		if len(parts) < 2 {
			return false
		}
		senderDID := parts[1]

		tm.mu.Lock()
		dt, exists := tm.direct[senderDID]
		if !exists {
			// Crear DirectTransport para el peer entrante
			dtCb := DirectCallbacks{
				OnSessionActive: func(peerDID string) {
					if tm.cb.OnDirectSessionActive != nil {
						tm.cb.OnDirectSessionActive(peerDID)
					}
				},
				OnMessage: func(peerDID string, plaintext []byte) {
					displayName := crypto.ResolveDID(peerDID)
					if tm.cb.OnMessage != nil {
						tm.cb.OnMessage(peerDID, displayName, string(plaintext))
					}
				},
				OnSessionLost: func(peerDID string) {
					if tm.cb.OnDirectSessionLost != nil {
						tm.cb.OnDirectSessionLost(peerDID)
					}
					tm.mu.Lock()
					delete(tm.direct, peerDID)
					tm.mu.Unlock()
				},
				OnFallbackToRelay: func(peerDID string) {
					if tm.cb.OnFallbackToRelay != nil {
						tm.cb.OnFallbackToRelay(peerDID)
					}
					tm.mu.Lock()
					delete(tm.direct, peerDID)
					tm.mu.Unlock()
				},
				OnClose: func(peerDID string) {
					tm.mu.Lock()
					delete(tm.direct, peerDID)
					tm.mu.Unlock()
				},
			}
			dt = NewDirectTransport(tm.identity.DID, tm.fsm, tm.faro, dtCb)
			dt.SetIdentity(tm.identity)
			tm.direct[senderDID] = dt
		}
		tm.mu.Unlock()

		// Procesar el SESSION_INCOMING
		if err := dt.HandleIncomingSession(raw); err != nil {
			if tm.cb.OnError != nil {
				tm.cb.OnError("session_incoming:"+senderDID[:15], err)
			}
			tm.mu.Lock()
			delete(tm.direct, senderDID)
			tm.mu.Unlock()
		}
		return true
	}

	// Para SESSION_INFO, PUNCH_NOW, SESSION_REDIRECT: rutear al
	// DirectTransport del peer correspondiente.
	parts := splitFields(raw)
	if len(parts) < 2 {
		return false
	}
	peerDID := parts[1]

	tm.mu.RLock()
	dt, exists := tm.direct[peerDID]
	tm.mu.RUnlock()

	if exists {
		// Depositar la señal en el canal del DirectTransport
		select {
		case dt.FaroMessages <- FaroSignal{Type: signalType, Raw: raw}:
		default:
			// Canal lleno, descartar
		}
		return true
	}

	return false
}

// ============================================================================
// ACL (actualización en caliente)
// ============================================================================

// UpdateACL reemplaza el ACL index en todos los transportes.
func (tm *TransportManager) UpdateACL(aclIndex map[[4]byte]PeerKeys) {
	aclByDID := make(map[string]PeerKeys, len(aclIndex))
	for _, pk := range aclIndex {
		aclByDID[pk.DID] = pk
	}

	tm.mu.Lock()
	tm.aclIndex = aclIndex
	tm.aclByDID = aclByDID
	tm.mu.Unlock()

	// Actualizar el RelayTransport
	tm.relay.UpdateACL(aclIndex)
}

// ============================================================================
// CONSULTAS
// ============================================================================

// IsDirectActive devuelve true si hay una sesión directa activa con un peer.
func (tm *TransportManager) IsDirectActive(peerDID string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	dt, exists := tm.direct[peerDID]
	return exists && dt.IsActive()
}

// ActiveDirectSessions devuelve el número de sesiones directas activas.
func (tm *TransportManager) ActiveDirectSessions() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	count := 0
	for _, dt := range tm.direct {
		if dt.IsActive() {
			count++
		}
	}
	return count
}

// FSM devuelve el FSM del transporte (para diagnóstico).
func (tm *TransportManager) FSM() *FSM {
	return tm.fsm
}

// Stats devuelve estadísticas del transporte.
type TransportStats struct {
	DirectSessions int    `json:"direct_sessions"`
	FSMState       string `json:"fsm_state"`
	RelayClosed    bool   `json:"relay_closed"`
}

func (tm *TransportManager) Stats() TransportStats {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return TransportStats{
		DirectSessions: len(tm.direct),
		FSMState:       tm.fsm.Current().String(),
		RelayClosed:    tm.relay.IsClosed(),
	}
}

// ============================================================================
// CICLO DE VIDA
// ============================================================================

// Close cierra todos los transportes y el manager.
func (tm *TransportManager) Close() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.closed {
		return
	}
	tm.closed = true

	// Cerrar todas las sesiones directas
	for peerDID, dt := range tm.direct {
		dt.Close()
		delete(tm.direct, peerDID)
	}

	// Cerrar el relay
	tm.relay.Close()

	fmt.Printf("[XTP-MGR] 🔒 TransportManager cerrado\n")
}

// ============================================================================
// UTILIDADES INTERNAS
// ============================================================================

// getPeerKeys busca las claves de un peer en el ACL.
func (tm *TransportManager) getPeerKeys(peerDID string) (PeerKeys, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	pk, exists := tm.aclByDID[peerDID]
	return pk, exists
}

// splitFields es strings.Fields pero sin importar strings
// (para evitar importar strings solo para esto).
func splitFields(s string) []string {
	var fields []string
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' {
			if start >= 0 {
				fields = append(fields, s[start:i])
				start = -1
			}
		} else {
			if start < 0 {
				start = i
			}
		}
	}
	if start >= 0 {
		fields = append(fields, s[start:])
	}
	return fields
}
