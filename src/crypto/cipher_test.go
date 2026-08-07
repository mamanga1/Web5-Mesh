package crypto

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	// Clave simétrica de 32 bytes (256 bits) para ChaCha20-Poly1305
	sharedKey := make([]byte, 32)
	if _, err := rand.Read(sharedKey); err != nil {
		t.Fatalf("Error generando clave de prueba: %v", err)
	}

	plaintext := []byte("Mensaje de prueba desde el Búnker - Web5 Mesh AON-CHP")

	// 1. Cifrar
	ciphertext, err := EncryptPayload(sharedKey, plaintext)
	if err != nil {
		t.Fatalf("EncryptPayload falló: %v", err)
	}

	// 2. Descifrar
	decrypted, err := DecryptPayload(sharedKey, ciphertext)
	if err != nil {
		t.Fatalf("DecryptPayload falló: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("El texto descifrado no coincide. Esperado: %s, obtenido: %s",
			plaintext, decrypted)
	}
}

func TestUniqueNonces(t *testing.T) {
	sharedKey := make([]byte, 32)
	if _, err := rand.Read(sharedKey); err != nil {
		t.Fatalf("Error generando clave: %v", err)
	}

	plaintext := []byte("Prueba de nonces aleatorios")

	// Ciframos el MISMO texto con la MISMA clave dos veces seguidas
	c1, err1 := EncryptPayload(sharedKey, plaintext)
	c2, err2 := EncryptPayload(sharedKey, plaintext)

	if err1 != nil || err2 != nil {
		t.Fatalf("Error en el cifrado: %v / %v", err1, err2)
	}

	// Los ciphertexts (y sus nonces) DEBEN ser completamente distintos
	if bytes.Equal(c1, c2) {
		t.Errorf("CRÍTICO: Dos cifrados consecutivos produjeron el mismo ciphertext/nonce.")
	}
}
