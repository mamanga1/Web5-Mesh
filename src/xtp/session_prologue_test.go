package xtp

import (
	"testing"

	"web5-mesh/src/crypto"
)

func TestHandshake_PrologueBinding(t *testing.T) {
	// Crear dos identities de prueba
	idA, err := crypto.LoadOrCreateIdentity()
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity A: %v", err)
	}
	idB, err := crypto.LoadOrCreateIdentity()
	if err != nil {
		t.Fatalf("LoadOrCreateIdentity B: %v", err)
	}

	var peerPubX [32]byte
	copy(peerPubX[:], idB.PubKeyX[:32])

	prologue := crypto.BuildNoisePrologue(idA.DID, idB.DID)

	hsA, err := crypto.NewHandshakeIK(true, idA.PrivKeyX, idA.PubKeyX, &peerPubX, prologue)
	if err != nil {
		t.Fatalf("NewHandshakeIK initiator: %v", err)
	}
	hsB, err := crypto.NewHandshakeIK(false, idB.PrivKeyX, idB.PubKeyX, nil, prologue)
	if err != nil {
		t.Fatalf("NewHandshakeIK responder: %v", err)
	}

	m1, _, err := hsA.WriteHandshake([]byte("meta"))
	if err != nil {
		t.Fatalf("hsA.WriteHandshake: %v", err)
	}
	_, _, err = hsB.ReadHandshake(m1)
	if err != nil {
		t.Fatalf("hsB.ReadHandshake: %v", err)
	}
	m2, _, err := hsB.WriteHandshake(nil)
	if err != nil {
		t.Fatalf("hsB.WriteHandshake 2: %v", err)
	}
	_, _, err = hsA.ReadHandshake(m2)
	if err != nil {
		t.Fatalf("hsA.ReadHandshake 2: %v", err)
	}

	if !hsA.IsCompleted() || !hsB.IsCompleted() {
		t.Fatalf("handshake no completado")
	}
}
