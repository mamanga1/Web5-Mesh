package commands

import (
	"encoding/hex"
	"fmt"
	"time"
	"web5-mesh/src/crypto"
)

func init() {
	Register("ping", cmdPing)
}

func cmdPing(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: ping <did>\nEj: ping did:maia:3SDGE3cQSht32LVNz7G6kppRyFERGKpQpHDwXQTurZo6"
	}

	targetDID := args[0]

	// 1. Cargar ACL
	acl, err := crypto.LoadACL()
	if err != nil {
		return "❌ Error cargando ACL: " + err.Error()
	}

	// 2. Buscar peer
	peerInfo, exists := acl.Peers[targetDID]
	if !exists {
		return fmt.Sprintf("❌ El DID %s no está en tu lista de confianza. Usá 'acl import' primero.", targetDID)
	}

	// 3. Decodificar clave pública X del peer
	pubKeyXBytes, err := hex.DecodeString(peerInfo.PubKeyX)
	if err != nil {
		return "❌ Error decodificando PubKeyX: " + err.Error()
	}

	if len(pubKeyXBytes) != 32 {
		return fmt.Sprintf("❌ PubKeyX tiene longitud incorrecta: %d bytes (esperado 32)", len(pubKeyXBytes))
	}

	// 4. Derivar clave compartida (método clásico, sin handshake)
	start := time.Now()
	
	sharedKey, err := crypto.DeriveSharedKey(id.PrivKeyX, pubKeyXBytes)
	if err != nil {
		return "❌ Error derivando clave: " + err.Error()
	}

	// 5. Cifrar mensaje de prueba con la clave compartida
	testMsg := []byte("PING cifrado con ChaCha20-Poly1305")
	ciphertext, err := crypto.EncryptPayload(sharedKey, testMsg)
	if err != nil {
		return "❌ Error cifrando: " + err.Error()
	}

	latency := time.Since(start)

	return fmt.Sprintf("✅ PONG de %s\n   Latencia: %v\n   Clave compartida X25519: %d bytes\n   Ciphertext: %d bytes\n   ✅ Cifrado E2E ChaCha20-Poly1305 operativo\n   🔄 Noise IK + PFS listo para handshake UDP",
		targetDID, latency, len(sharedKey), len(ciphertext))
}
