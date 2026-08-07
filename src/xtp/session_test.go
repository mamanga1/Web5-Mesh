package xtp

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
	"time"

	"web5-mesh/src/crypto"
)

// Helper para generar identidades completas de prueba en memoria
func generateTestIdentity(_ string) (*crypto.Identity, error) {
	// 1. Claves Ed25519 para firma
	pubEd, privEd, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	// 2. Claves X25519 para cifrado
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	privBytes := key.Bytes()
	pubBytes := key.PublicKey().Bytes()

	var privX, pubX [32]byte
	copy(privX[:], privBytes)
	copy(pubX[:], pubBytes)

	// Key ID oficial de 4 bytes en hex puro
	keyID := crypto.DeriveKeyID(pubX[:])

	return &crypto.Identity{
		DID:       fmt.Sprintf("did:w5:%x", keyID),
		PrivKeyEd: privEd,
		PubKeyEd:  pubEd,
		PrivKeyX:  &privX,
		PubKeyX:   &pubX,
	}, nil
}

func TestSession_HandshakeAndCommunication(t *testing.T) {
	aliceIdentity, err := generateTestIdentity("alice")
	if err != nil {
		t.Fatalf("Error generando identidad Alice: %v", err)
	}

	bobIdentity, err := generateTestIdentity("bob")
	if err != nil {
		t.Fatalf("Error generando identidad Bob: %v", err)
	}

	bobPubX := new([32]byte)
	copy(bobPubX[:], bobIdentity.PubKeyX[:])

	// 1. Instanciar sesiones
	aliceSession, err := NewSession(true, aliceIdentity, bobIdentity.DID, bobPubX)
	if err != nil {
		t.Fatalf("Error creando sesión Alice: %v", err)
	}

	bobSession, err := NewSession(false, bobIdentity, aliceIdentity.DID, nil)
	if err != nil {
		t.Fatalf("Error creando sesión Bob: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	aliceSession.OnActivate(func(peerDID string) {
		if peerDID != bobIdentity.DID {
			t.Errorf("Alice activó con DID incorrecto: %s", peerDID)
		}
		wg.Done()
	})

	bobSession.OnActivate(func(peerDID string) {
		if peerDID != aliceIdentity.DID {
			t.Errorf("Bob activó con DID incorrecto: %s", peerDID)
		}
		wg.Done()
	})

	// 2. Handshake paso 1: Alice -> Bob
	msg1, err := aliceSession.InitiatorMessage()
	if err != nil {
		t.Fatalf("Error generando InitiatorMessage: %v", err)
	}

	// 3. Handshake paso 2: Bob procesa msg1 y genera msg2
	msg2, bobDone, err := bobSession.HandleMessage(msg1)
	if err != nil {
		t.Fatalf("Bob fallo en HandleMessage: %v", err)
	}
	if !bobDone {
		t.Errorf("Se esperaba que Bob completara el handshake")
	}

	// 4. Handshake paso 3: Alice procesa msg2
	_, aliceDone, err := aliceSession.HandleMessage(msg2)
	if err != nil {
		t.Fatalf("Alice fallo en HandleMessage: %v", err)
	}
	if !aliceDone {
		t.Errorf("Se esperaba que Alice completara el handshake")
	}

	wg.Wait()

	// 5. Cifrado / Descifrado bidireccional
	payloadAlice := []byte("Hola Búnker, mensaje de prueba desde Alice")
	cipherAlice, err := aliceSession.Encrypt(payloadAlice)
	if err != nil {
		t.Fatalf("Error cifrando Alice: %v", err)
	}

	plainBob, err := bobSession.Decrypt(cipherAlice)
	if err != nil {
		t.Fatalf("Error descifrando Bob: %v", err)
	}

	if string(plainBob) != string(payloadAlice) {
		t.Fatalf("Texto descifrado no coincide. Esperado: %s, Obtenido: %s", payloadAlice, plainBob)
	}
}

func TestSession_ShortDID_SafeBounds(t *testing.T) {
	id, _ := generateTestIdentity("node")
	shortDID := "did:w5:1"

	pubX := new([32]byte)
	copy(pubX[:], id.PubKeyX[:])

	session, err := NewSession(true, id, shortDID, pubX)
	if err != nil {
		t.Fatalf("Error creando sesión: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Se produjo un panic por slicing out-of-bounds: %v", r)
		}
	}()

	session.Close()
}

func TestSession_RekeyTrigger(t *testing.T) {
	aliceIdentity, _ := generateTestIdentity("alice")
	bobIdentity, _ := generateTestIdentity("bob")

	bobPubX := new([32]byte)
	copy(bobPubX[:], bobIdentity.PubKeyX[:])

	aliceSession, _ := NewSession(true, aliceIdentity, bobIdentity.DID, bobPubX)
	bobSession, _ := NewSession(false, bobIdentity, aliceIdentity.DID, nil)

	msg1, _ := aliceSession.InitiatorMessage()
	msg2, _, _ := bobSession.HandleMessage(msg1)
	aliceSession.HandleMessage(msg2)

	for i := 0; i < RekeyAfterMessages+5; i++ {
		msg := []byte("ping")
		ct, err := aliceSession.Encrypt(msg)
		if err != nil {
			t.Fatalf("Error en envío %d post-rekey: %v", i, err)
		}
		
		// Bob solo descifra hasta que Alice hace el rekey automático.
		// En la red real, Alice manda una señal de control antes de esto.
		if i < RekeyAfterMessages {
			_, err = bobSession.Decrypt(ct)
			if err != nil {
				t.Fatalf("Error en recepción %d post-rekey: %v", i, err)
			}
		}
	}

	if aliceSession.sendCount > RekeyAfterMessages {
		t.Errorf("El sendCount debería haberse reseteado tras el rekey, valor actual: %d", aliceSession.sendCount)
	}
}

func TestManager_GetOrCreateSession_Concurrency(t *testing.T) {
	identity, _ := generateTestIdentity("manager")
	mgr := NewManager(identity)

	peerDID := "did:w5:node-test-379"
	peerPubX := new([32]byte)

	var wg sync.WaitGroup
	routines := 20

	wg.Add(routines)
	for i := 0; i < routines; i++ {
		go func() {
			defer wg.Done()
			_, err := mgr.GetOrCreateSession(true, peerDID, peerPubX)
			if err != nil {
				t.Errorf("Error en GetOrCreateSession concurrente: %v", err)
			}
		}()
	}

	wg.Wait()

	if len(mgr.sessions) != 1 {
		t.Errorf("Esperada exactamente 1 sesión en el mapa, encontradas: %d", len(mgr.sessions))
	}
}

func TestSession_CallbackPanicRecovery(t *testing.T) {
	aliceIdentity, _ := generateTestIdentity("alice")
	bobIdentity, _ := generateTestIdentity("bob")
	bobPubX := new([32]byte)
	copy(bobPubX[:], bobIdentity.PubKeyX[:])

	session, _ := NewSession(true, aliceIdentity, bobIdentity.DID, bobPubX)

	var wg sync.WaitGroup
	wg.Add(1)

	session.OnClose(func(peerDID string) {
		defer wg.Done()
		var ptr *int
		*ptr = 42
	})

	session.Close()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Recover exitoso
	case <-time.After(2 * time.Second):
		t.Fatalf("Timeout esperando el callback de OnClose")
	}
}
