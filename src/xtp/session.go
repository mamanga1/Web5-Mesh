package xtp

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"web5-mesh/src/crypto"
)

const (
	RekeyAfterMessages = 100
	RekeyAfterDuration = 5 * time.Minute
	HandshakeTimeout   = 10 * time.Second
)

type SessionState int

const (
	SessionNew          SessionState = iota
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
	var myPrivX *[32]byte
	var myPubX *[32]byte

	privX := new([32]byte)
	copy(privX[:], myIdentity.PrivKeyX[:])
	myPrivX = privX

	pubX := new([32]byte)
	copy(pubX[:], myIdentity.PubKeyX[:])
	myPubX = pubX

	noise, err := crypto.NewHandshakeIK(isInitiator, myPrivX, myPubX, peerPubX)
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

	msg, completed, err := s.noise.WriteHandshake(metaJSON)
	if err != nil {
		return nil, fmt.Errorf("escribiendo handshake: %w", err)
	}

	s.state = SessionHandshaking
	_ = completed

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
			s.state = SessionActive
			s.activatedAt = time.Now()
			s.lastRekeyAt = time.Now()
			s.sendCount = 0
			s.recvCount = 0
			if s.onActivate != nil {
				go s.onActivate(s.peerDID)
			}
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
		s.state = SessionActive
		s.activatedAt = time.Now()
		s.lastRekeyAt = time.Now()
		s.sendCount = 0
		s.recvCount = 0
		if s.onActivate != nil {
			go s.onActivate(s.peerDID)
		}
	}

	return resp, done, nil
}

// ============================================================================
// FIX #1 (parte 2): PeerStatic — para que el respondedor verifique identidad
// ============================================================================

// PeerStatic retorna la clave pública estática del peer verificada por Noise.
// Solo válido después del handshake completo.
func (s *Session) PeerStatic() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.noise.PeerStatic()
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

// ============================================================================
// FIX #2: rekey simétrico — Decrypt también verifica rekey
// ============================================================================

func (s *Session) Decrypt(ciphertext []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != SessionActive {
		return nil, fmt.Errorf("sesión no activa (estado: %s)", s.state)
	}

	// FIX #2: rekey simétrico — ambos lados rotan al mismo contador total
	s.maybeRekeyLocked()

	plaintext, err := s.noise.Decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("descifrando: %w", err)
	}

	s.recvCount++
	return plaintext, nil
}

// ============================================================================
// FIX logging: guardar total ANTES de resetear
// ============================================================================

func (s *Session) maybeRekeyLocked() {
	needRekey := false

	if s.sendCount+s.recvCount >= RekeyAfterMessages {
		needRekey = true
	}
	if time.Since(s.lastRekeyAt) >= RekeyAfterDuration {
		needRekey = true
	}

	if needRekey {
		total := s.sendCount + s.recvCount // ← guardar ANTES de resetear
		s.noise.Rekey()
		s.lastRekeyAt = time.Now()
		s.sendCount = 0
		s.recvCount = 0
		fmt.Printf("[XTP] 🔑 Rekey con %s (después de %d msg / %s)\n",
			s.peerDID[:20]+"...", total,
			time.Since(s.activatedAt).Round(time.Second))
	}
}

// ============================================================================
// CICLO DE VIDA
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
	if s.onClose != nil {
		go s.onClose(s.peerDID)
	}
	fmt.Printf("[XTP] 🔒 Sesión cerrada con %s\n", s.peerDID[:20]+"...")
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
// SESSION MANAGER
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
	m.mu.RLock()
	session, exists := m.sessions[peerDID]
	m.mu.RUnlock()

	if exists && session.IsActive() {
		return session, nil
	}

	return m.CreateSession(isInitiator, peerDID, peerPubX)
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
		return nil, fmt.Errorf("no hay sesión con %s", peerDID[:20]+"...")
	}
	return session.Encrypt(plaintext)
}

func (m *Manager) Decrypt(peerDID string, ciphertext []byte) ([]byte, error) {
	session := m.GetSession(peerDID)
	if session == nil {
		return nil, fmt.Errorf("no hay sesión con %s", peerDID[:20]+"...")
	}
	return session.Decrypt(ciphertext)
}
