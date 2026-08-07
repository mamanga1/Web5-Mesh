package xtp

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"web5-mesh/src/crypto"
)

// ============================================================================
// MOCKS & HELPERS
// ============================================================================

type MockFaroSender struct {
	mu          sync.Mutex
	Messages    []string
	ErrToReturn error
}

func (m *MockFaroSender) SendToFaro(msg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ErrToReturn != nil {
		return m.ErrToReturn
	}
	m.Messages = append(m.Messages, msg)
	return nil
}

func (m *MockFaroSender) LastMessage() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Messages) == 0 {
		return ""
	}
	return m.Messages[len(m.Messages)-1]
}

func setupManagerTestEnv(t *testing.T) (*crypto.Identity, *crypto.Identity, map[[4]byte]PeerKeys, *MockFaroSender) {
	aliceId, err := generateTestIdentity("alice")
	if err != nil {
		t.Fatalf("Error generando identidad Alice: %v", err)
	}

	bobId, err := generateTestIdentity("bob")
	if err != nil {
		t.Fatalf("Error generando identidad Bob: %v", err)
	}

	faro := &MockFaroSender{}

	prefix := crypto.DeriveKeyID(bobId.PubKeyX[:])

	// Clave simétrica dummy de 32 bytes para que ChaCha20 no reviente
	mockSharedKey := make([]byte, 32)
	for i := range mockSharedKey {
		mockSharedKey[i] = 0x42
	}

	bobKeys := PeerKeys{
		DID:       bobId.DID,
		PubKeyX:   bobId.PubKeyX[:],
		SharedKey: mockSharedKey,
	}

	aclIndex := map[[4]byte]PeerKeys{
		prefix: bobKeys,
	}

	return aliceId, bobId, aclIndex, faro
}

// ============================================================================
// TESTS UNITARIOS
// ============================================================================

func TestTransportManager_SendFallbackRelay(t *testing.T) {
	aliceId, bobId, aclIndex, faro := setupManagerTestEnv(t)

	var relayMsgReceived bool
	var mu sync.Mutex

	cb := ManagerCallbacks{
		OnMessage: func(peerDID, displayName, command string) {
			mu.Lock()
			relayMsgReceived = true
			mu.Unlock()
		},
	}

	cfg := DefaultManagerConfig()
	cfg.AutoDirect = false // Forzar solo relay para test aislado

	tm := NewTransportManager(aliceId, faro, aclIndex, cb, cfg)
	defer tm.Close()

	transport, err := tm.Send(bobId.DID, "PING_RELAY")
	if err != nil {
		t.Fatalf("Error enviando mensaje: %v", err)
	}

	if transport != "relay" {
		t.Errorf("Se esperaba transporte 'relay', obtenido: '%s'", transport)
	}

	if faro.LastMessage() == "" {
		t.Errorf("Se esperaba que el faro registrara el envío del mensaje")
	}

	mu.Lock()
	_ = relayMsgReceived
	mu.Unlock()
}

func TestTransportManager_ACL_Filtering(t *testing.T) {
	aliceId, _, aclIndex, faro := setupManagerTestEnv(t)

	cfg := DefaultManagerConfig()
	cfg.AutoDirect = false

	tm := NewTransportManager(aliceId, faro, aclIndex, ManagerCallbacks{}, cfg)
	defer tm.Close()

	untrustedId, _ := generateTestIdentity("untrusted")

	fakeIncomingSignal := fmt.Sprintf("SESSION_INCOMING %s candidate_data_here", untrustedId.DID)
	handled := tm.HandleIncoming(fakeIncomingSignal)

	if !handled {
		t.Errorf("HandleIncoming debió reconocer la señal del faro aunque se rechace el DID")
	}

	if tm.IsDirectActive(untrustedId.DID) {
		t.Errorf("Se creó sesión directa para un peer fuera del ACL")
	}
}

func TestTransportManager_UpdateACL(t *testing.T) {
	aliceId, _, aclIndex, faro := setupManagerTestEnv(t)

	// ¡ACÁ ESTABA EL CUELGUE! Apagamos WebRTC directo para que no se quede esperando timeout.
	cfg := DefaultManagerConfig()
	cfg.AutoDirect = false

	tm := NewTransportManager(aliceId, faro, aclIndex, ManagerCallbacks{}, cfg)
	defer tm.Close()

	charlieId, _ := generateTestIdentity("charlie")
	charliePrefix := crypto.DeriveKeyID(charlieId.PubKeyX[:])

	mockSharedKey := make([]byte, 32)
	for i := range mockSharedKey {
		mockSharedKey[i] = 0x42
	}

	charlieKeys := PeerKeys{
		DID:       charlieId.DID,
		PubKeyX:   charlieId.PubKeyX[:],
		SharedKey: mockSharedKey,
	}

	newACL := map[[4]byte]PeerKeys{
		charliePrefix: charlieKeys,
	}

	tm.UpdateACL(newACL)

	transport, err := tm.Send(charlieId.DID, "HOLA_CHARLIE")
	if err != nil {
		t.Fatalf("Error enviando a Charlie tras actualizar ACL: %v", err)
	}

	if transport != "relay" {
		t.Errorf("Esperado transporte 'relay', obtenido: %s", transport)
	}
}

func TestTransportManager_IsFaroSignal(t *testing.T) {
	aliceId, _, aclIndex, faro := setupManagerTestEnv(t)

	cfg := DefaultManagerConfig()
	cfg.AutoDirect = false

	tm := NewTransportManager(aliceId, faro, aclIndex, ManagerCallbacks{}, cfg)
	defer tm.Close()

	tests := []struct {
		raw      string
		expected bool
	}{
		{"SESSION_INFO payload", true},
		{"SESSION_INCOMING did:w5:123", true},
		{"PUNCH_NOW did:w5:123", true},
		{"SESSION_REDIRECT target", true},
		{"SESSION_ERROR err_code", true},
		{"DATA_NORMAL payload", false},
		{"CHAT_MSG hola", false},
	}

	for _, tt := range tests {
		got := tm.isFaroSignal(tt.raw)
		if got != tt.expected {
			t.Errorf("isFaroSignal(%q) = %v; esperado %v", tt.raw, got, tt.expected)
		}
	}
}

func TestTransportManager_ConcurrentSendAndClose(t *testing.T) {
	aliceId, bobId, aclIndex, faro := setupManagerTestEnv(t)

	cfg := DefaultManagerConfig()
	cfg.AutoDirect = false // Prevenimos que los hilos concurrentes cuelguen la goroutine

	tm := NewTransportManager(aliceId, faro, aclIndex, ManagerCallbacks{}, cfg)

	var wg sync.WaitGroup
	workers := 10

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_, _ = tm.Send(bobId.DID, fmt.Sprintf("msg-%d-%d", id, j))
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	// Cerrar manager a mitad de la carga
	time.Sleep(10 * time.Millisecond)
	tm.Close()

	wg.Wait()

	stats := tm.Stats()
	if !stats.RelayClosed {
		t.Errorf("Se esperaba que Relay estuviera cerrado tras el Close()")
	}
}

func TestTransportManager_SafeDID_NoPanic(t *testing.T) {
	aliceId, _, aclIndex, faro := setupManagerTestEnv(t)

	cfg := DefaultManagerConfig()
	cfg.AutoDirect = false

	tm := NewTransportManager(aliceId, faro, aclIndex, ManagerCallbacks{}, cfg)
	defer tm.Close()

	shortDID := "d:1"

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Panic capturado por slicing de DID corto: %v", r)
		}
	}()

	_, _ = tm.Send(shortDID, "TEST")
}
