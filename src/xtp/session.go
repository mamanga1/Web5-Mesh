package xtp

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"web5-mesh/src/crypto"
)

// ============================================================================
// SESSION MANAGER & CONSTANTS
// ============================================================================

const (
	RekeyAfterMessages = 100
	RekeyAfterDuration = 5 * time.Minute
	HandshakeTimeout   = 10 * time.Second
)

type SessionState int

const (
	SessionNew SessionState = iota
	SessionHandshaking
	SessionActive
	SessionClosed
)

func (s SessionState) String() string {
	switch s {
	case SessionNew:
		return "NEW"
	case SessionHandshaking:
		return "HANDSHAKING"
	case SessionActive:
		return "ACTIVE"
	case SessionClosed:
		return "CLOSED"
	default:
		return "UNKNOWN"
	}
}

type Session struct {
	mu sync.Mutex

	myDID    string
	peerDID  string
	peerPubX *[32]byte

	noise       *crypto.HandshakeState
	isInitiator bool

	state       SessionState
	createdAt   time.Time
	activatedAt time.Time
	lastRekeyAt time.Time

	sendCount int
	recvCount int

	onActivate func(peerDID string)
	onClose    func(peerDID string)
}

func NewSession(isInitiator bool, myIdentity *crypto.Identity, peerDID string, peerPubX *[32]byte) (*Session, error) {
	if myIdentity == nil {
		return nil, fmt.Errorf("myIdentity no puede ser nil")
	}

	privX := new([32]byte)
	copy(privX[:], myIdentity.PrivKeyX[:])

	pubX := new([32]byte)
	copy(pubX[:], myIdentity.PubKeyX[:])

	var prologue []byte
	if peerDID != "" {
		if isInitiator {
			prologue = crypto.BuildNoisePrologue(myIdentity.DID, peerDID)
		} else {
			prologue = crypto.BuildNoisePrologue(peerDID, myIdentity.DID)
		}
	}
	noise, err := crypto.NewHandshakeIK(isInitiator, privX, pubX, peerPubX, prologue)
	if err != nil {
		return nil, fmt.Errorf("creando Noise IK: %w", err)
	}

	return &Session{
		myDID:       myIdentity.DID,
		peerDID:     peerDID,
		peerPubX:    peerPubX,
		noise:       noise,
		isInitiator: isInitiator,
		state:       SessionNew,
		createdAt:   time.Now(),
	}, nil
}

// ============================================================================
// HANDSHAKE
// ============================================================================

func (s *Session) InitiatorMessage() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isInitiator {
		return nil, fmt.Errorf("solo el iniciador puede generar el primer mensaje")
	}
	if s.state != SessionNew {
		return nil, fmt.Errorf("sesión en estado %s, esperado NEW", s.state)
	}

	meta := sessionMeta{
		FromDID:   s.myDID,
		Timestamp: time.Now().Unix(),
		Version:   "xtp/1.0",
	}
	metaJSON, _ := json.Marshal(meta)

	msg, _, err := s.noise.WriteHandshake(metaJSON)
	if err != nil {
		return nil, fmt.Errorf("escribiendo handshake: %w", err)
	}

	s.state = SessionHandshaking
	return msg, nil
}

func (s *Session) HandleMessage(msg []byte) (response []byte, completed bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == SessionClosed {
		return nil, false, fmt.Errorf("sesión cerrada")
	}

	if s.isInitiator {
		_, done, err := s.noise.ReadHandshake(msg)
		if err != nil {
			return nil, false, fmt.Errorf("leyendo handshake (iniciador): %w", err)
		}

		if done {
			s.markActiveLocked()
		}
		return nil, done, nil
	}

	_, _, err = s.noise.ReadHandshake(msg)
	if err != nil {
		return nil, false, fmt.Errorf("leyendo handshake (respondedor): %w", err)
	}

	resp, done, err := s.noise.WriteHandshake(nil)
	if err != nil {
		return nil, false, fmt.Errorf("escribiendo handshake (respondedor): %w", err)
	}

	s.state = SessionHandshaking

	if done {
		s.markActiveLocked()
	}

	return resp, done, nil
}

func (s *Session) markActiveLocked() {
	s.state = SessionActive
	s.activatedAt = time.Now()
	s.lastRekeyAt = time.Now()
	s.sendCount = 0
	s.recvCount = 0

	dispatchCallback("onActivate", s.onActivate, s.peerDID)
}

// ============================================================================
// COMUNICACIÓN (post-handshake)
// ============================================================================

func (s *Session) Encrypt(plaintext []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionActive {
		return nil, fmt.Errorf("sesión no activa (estado: %s)", s.state)
	}

	s.maybeRekeyLocked()

	ciphertext, err := s.noise.Encrypt(plaintext)
	if err != nil {
		return nil, fmt.Errorf("cifrando: %w", err)
	}

	s.sendCount++
	return ciphertext, nil
}

func (s *Session) Decrypt(ciphertext []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionActive {
		return nil, fmt.Errorf("sesión no activa (estado: %s)", s.state)
	}

	plaintext, err := s.noise.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("descifrando: %w", err)
	}

	s.recvCount++
	return plaintext, nil
}

func (s *Session) maybeRekeyLocked() {
	needRekey := false

	if s.sendCount+s.recvCount >= RekeyAfterMessages {
		needRekey = true
	}
	if time.Since(s.lastRekeyAt) >= RekeyAfterDuration {
		needRekey = true
	}

	if needRekey {
		total := s.sendCount + s.recvCount
		uptime := time.Since(s.activatedAt).Round(time.Second)

		s.noise.Rekey()
		s.lastRekeyAt = time.Now()
		s.sendCount = 0
		s.recvCount = 0

		Debugf("[XTP] 🔑 Rekey con %s (después de %d msg / %s)\n",
			safeDID(s.peerDID, 20)+"...", total, uptime)
	}
}

// ============================================================================
// CICLO DE VIDA & STATS
// ============================================================================

func (s *Session) State() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Session) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == SessionActive
}

func (s *Session) PeerDID() string {
	return s.peerDID
}

func (s *Session) IsInitiator() bool {
	return s.isInitiator
}

func (s *Session) Stats() SessionStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return SessionStats{
		PeerDID:     s.peerDID,
		State:       s.state.String(),
		IsInitiator: s.isInitiator,
		SendCount:   s.sendCount,
		RecvCount:   s.recvCount,
		CreatedAt:   s.createdAt,
		ActivatedAt: s.activatedAt,
		LastRekeyAt: s.lastRekeyAt,
		Uptime:      time.Since(s.createdAt).Round(time.Second),
	}
}

type SessionStats struct {
	PeerDID     string        `json:"peer_did"`
	State       string        `json:"state"`
	IsInitiator bool          `json:"is_initiator"`
	SendCount   int           `json:"send_count"`
	RecvCount   int           `json:"recv_count"`
	CreatedAt   time.Time     `json:"created_at"`
	ActivatedAt time.Time     `json:"activated_at"`
	LastRekeyAt time.Time     `json:"last_rekey_at"`
	Uptime      time.Duration `json:"uptime"`
}

func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == SessionClosed {
		return
	}
	s.state = SessionClosed

	dispatchCallback("onClose", s.onClose, s.peerDID)

	Debugf("[XTP] 🔒 Sesión cerrada con %s\n", safeDID(s.peerDID, 20)+"...")
}

func (s *Session) OnActivate(cb func(peerDID string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onActivate = cb
}

func (s *Session) OnClose(cb func(peerDID string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onClose = cb
}

type sessionMeta struct {
	FromDID   string `json:"from_did"`
	Timestamp int64  `json:"ts"`
	Version   string `json:"version"`
}

// ============================================================================
// MANAGER DE MÚLTIPLES SESIONES
// ============================================================================

type Manager struct {
	mu       sync.RWMutex
	identity *crypto.Identity
	sessions map[string]*Session
}

func NewManager(identity *crypto.Identity) *Manager {
	return &Manager{
		identity: identity,
		sessions: make(map[string]*Session),
	}
}

func (m *Manager) CreateSession(isInitiator bool, peerDID string, peerPubX *[32]byte) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.sessions[peerDID]; ok {
		existing.Close()
	}

	session, err := NewSession(isInitiator, m.identity, peerDID, peerPubX)
	if err != nil {
		return nil, err
	}

	m.sessions[peerDID] = session
	return session, nil
}

func (m *Manager) GetSession(peerDID string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[peerDID]
}

func (m *Manager) GetOrCreateSession(isInitiator bool, peerDID string, peerPubX *[32]byte) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, exists := m.sessions[peerDID]; exists {
		if session.IsActive() {
			return session, nil
		}
		session.Close()
	}

	session, err := NewSession(isInitiator, m.identity, peerDID, peerPubX)
	if err != nil {
		return nil, err
	}

	m.sessions[peerDID] = session
	return session, nil
}

func (m *Manager) CloseSession(peerDID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[peerDID]; ok {
		session.Close()
		delete(m.sessions, peerDID)
	}
}

func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for did, session := range m.sessions {
		session.Close()
		delete(m.sessions, did)
	}
}

func (m *Manager) ActiveSessions() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, session := range m.sessions {
		if session.IsActive() {
			count++
		}
	}
	return count
}

func (m *Manager) ListSessions() []SessionStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make([]SessionStats, 0, len(m.sessions))
	for _, session := range m.sessions {
		stats = append(stats, session.Stats())
	}
	return stats
}

func (m *Manager) Encrypt(peerDID string, plaintext []byte) ([]byte, error) {
	session := m.GetSession(peerDID)
	if session == nil {
		return nil, fmt.Errorf("no hay sesión con %s", safeDID(peerDID, 20)+"...")
	}
	return session.Encrypt(plaintext)
}

func (m *Manager) Decrypt(peerDID string, ciphertext []byte) ([]byte, error) {
	session := m.GetSession(peerDID)
	if session == nil {
		return nil, fmt.Errorf("no hay sesión con %s", safeDID(peerDID, 20)+"...")
	}
	return session.Decrypt(ciphertext)
}

// ============================================================================
// HELPERS
// ============================================================================

func dispatchCallback(cbName string, cb func(string), peerDID string) {
	if cb == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				Debugf("[XTP] ⚠️ Panic en %s: %v\n", cbName, r)
			}
		}()
		cb(peerDID)
	}()
}
