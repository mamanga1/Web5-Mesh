package crypto

import (
	"errors"
	"sync"

	"github.com/flynn/noise"
)

type HandshakeState struct {
	cs          noise.CipherSuite
	hs          *noise.HandshakeState
	sendCipher  *noise.CipherState
	recvCipher  *noise.CipherState
	isInitiator bool
	completed   bool
	mu          sync.Mutex
}

func NewHandshakeIK(isInitiator bool, myPrivX *[32]byte, myPubX *[32]byte, theirPubX *[32]byte) (*HandshakeState, error) {
	cs := noise.NewCipherSuite(noise.DH25519, noise.CipherChaChaPoly, noise.HashSHA256)

	var hs *noise.HandshakeState
	var err error

	if isInitiator {
		if theirPubX == nil {
			return nil, errors.New("iniciador IK necesita la clave pública del peer")
		}
		hs, err = noise.NewHandshakeState(noise.Config{
			CipherSuite:   cs,
			Pattern:       noise.HandshakeIK,
			Initiator:     true,
			StaticKeypair: noise.DHKey{Private: myPrivX[:], Public: myPubX[:]},
			PeerStatic:    theirPubX[:],
		})
	} else {
		hs, err = noise.NewHandshakeState(noise.Config{
			CipherSuite:   cs,
			Pattern:       noise.HandshakeIK,
			Initiator:     false,
			StaticKeypair: noise.DHKey{Private: myPrivX[:], Public: myPubX[:]},
		})
	}

	if err != nil {
		return nil, err
	}

	return &HandshakeState{
		cs:          cs,
		hs:          hs,
		isInitiator: isInitiator,
	}, nil
}

func (h *HandshakeState) WriteHandshake(payload []byte) (msg []byte, completed bool, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	out, c1, c2, err := h.hs.WriteMessage(nil, payload)
	if err != nil {
		return nil, false, err
	}

	if c1 != nil && c2 != nil {
		if h.isInitiator {
			h.sendCipher = c1
			h.recvCipher = c2
		} else {
			h.sendCipher = c2
			h.recvCipher = c1
		}
		h.completed = true
	}

	return out, h.completed, nil
}

func (h *HandshakeState) ReadHandshake(msg []byte) (payload []byte, completed bool, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	p, c1, c2, err := h.hs.ReadMessage(nil, msg)
	if err != nil {
		return nil, false, err
	}

	if c1 != nil && c2 != nil {
		if h.isInitiator {
			h.sendCipher = c1
			h.recvCipher = c2
		} else {
			h.sendCipher = c2
			h.recvCipher = c1
		}
		h.completed = true
	}

	return p, h.completed, nil
}

func (h *HandshakeState) IsCompleted() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.completed
}

func (h *HandshakeState) Encrypt(plaintext []byte) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sendCipher == nil {
		return nil, errors.New("handshake no completado")
	}
	return h.sendCipher.Encrypt(nil, nil, plaintext)
}

func (h *HandshakeState) Decrypt(ciphertext []byte) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.recvCipher == nil {
		return nil, errors.New("handshake no completado")
	}
	return h.recvCipher.Decrypt(nil, nil, ciphertext)
}

func (h *HandshakeState) Rekey() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.sendCipher != nil {
		h.sendCipher.Rekey()
	}
	if h.recvCipher != nil {
		h.recvCipher.Rekey()
	}
}

// ============================================================================
// FIX #1: PeerStatic — necesario para verificar la identidad del iniciador
// ============================================================================

// PeerStatic retorna la clave pública estática del peer, verificada
// criptográficamente durante el handshake Noise IK.
// Solo válido después de que el handshake se completó.
// El respondedor DEBE comparar esto contra la PubKeyX del ACL para el
// DID reclamado. Si no coincide, cerrar la sesión (suplantación).
func (h *HandshakeState) PeerStatic() []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.completed {
		return nil
	}
	return h.hs.PeerStatic()
}
