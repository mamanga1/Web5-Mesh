package commands

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"web5-mesh/src/crypto"
)

func init() {
	Register("verify", cmdVerify)
}

func cmdVerify(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: verify <nombre_archivo_en_bóveda>"
	}

	filename := args[0]
	filePath := filepath.Join(getSecureDir(), filename)
	sigPath := filePath + ".sig"

	// Leer archivo original
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Sprintf("❌ Error leyendo archivo: %v", err)
	}

	// Leer archivo de firma
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		return fmt.Sprintf("❌ No se encontró archivo de firma: %v\n💡 Primero firmá el archivo con: sign %s", err, filename)
	}

	// Parsear metadata de la firma
	var sigMeta map[string]interface{}
	err = json.Unmarshal(sigData, &sigMeta)
	if err != nil {
		return fmt.Sprintf("❌ Error leyendo metadata de firma: %v", err)
	}

	// Extraer datos de la firma
	storedHash, ok := sigMeta["hash"].(string)
	if !ok {
		return "❌ Metadata de firma inválida: falta hash"
	}
	sigB64, ok := sigMeta["signature"].(string)
	if !ok {
		return "❌ Metadata de firma inválida: falta signature"
	}
	signerDID, ok := sigMeta["signer"].(string)
	if !ok {
		return "❌ Metadata de firma inválida: falta signer"
	}
	timestamp, ok := sigMeta["timestamp"].(float64)
	if !ok {
		return "❌ Metadata de firma inválida: falta timestamp"
	}

	// 1. VERIFICAR INTEGRIDAD (Hash)
	calculatedHash := sha256.Sum256(data)
	calculatedHashHex := hex.EncodeToString(calculatedHash[:])

	if calculatedHashHex != storedHash {
		return fmt.Sprintf("❌ INTEGRIDAD COMPROMETIDA:\n   ├── Hash esperado: %s...\n   ├── Hash actual: %s...\n   └── El archivo fue modificado después de la firma.", storedHash[:16], calculatedHashHex[:16])
	}

	// 2. VERIFICAR AUTENTICIDAD (Firma Ed25519)
	sigBytes, err := base64.StdEncoding.DecodeString(sigB64)
	if err != nil {
		return fmt.Sprintf("❌ Error decodificando firma: %v", err)
	}

	// Determinar qué clave pública usar para verificar
	var pubKeyBytes []byte
	var isSelfSigned bool

	if signerDID == id.DID {
		// Firma propia: usar nuestra propia clave pública (ya es []byte)
		pubKeyBytes = id.PubKeyEd
		isSelfSigned = true
	} else {
		// Firma de otro: buscar en la ACL
		acl, err := crypto.LoadACL()
		if err != nil {
			return fmt.Sprintf("⚠️ Hash válido pero no se pudo cargar ACL: %v", err)
		}

		peer, exists := acl.Peers[signerDID]
		if !exists || peer.PubKeyEd == "" {
			return fmt.Sprintf("✅ INTEGRIDAD: ✅ VÁLIDA\n   ├── Hash: %s...\n   ├── Archivo no modificado\n   ├── Firmante: %s\n   └── ⚠️ AUTENTICIDAD: No verificable (firmante no está en tu ACL)", 
				calculatedHashHex[:16], signerDID[:20]+"...")
		}

		pubKeyBytes, err = hex.DecodeString(peer.PubKeyEd)
		if err != nil {
			return fmt.Sprintf("❌ Error decodificando clave pública: %v", err)
		}
	}

	// Verificar firma
	isValid := crypto.VerifyMessage(pubKeyBytes, calculatedHash[:], sigBytes)

	if !isValid {
		return fmt.Sprintf("❌ FIRMA INVÁLIDA:\n   ├── Hash: ✅ Válido\n   ├── Firma: ❌ Inválida\n   └── El archivo pudo ser firmado por alguien diferente o la firma fue alterada.")
	}

	// Todo válido
	sigTime := time.Unix(int64(timestamp), 0)
	signerLabel := "Tú mismo"
	if !isSelfSigned {
		signerLabel = signerDID[:20] + "..."
	}

	return fmt.Sprintf("✅ VERIFICACIÓN EXITOSA:\n   ├── Integridad: ✅ Hash válido (%s...)\n   ├── Autenticidad: ✅ Firma válida\n   ├── Firmante: %s\n   ├── Fecha de firma: %s\n   └── El archivo es auténtico y no fue modificado.", 
		calculatedHashHex[:16], signerLabel, sigTime.Format("2006-01-02 15:04:05"))
}
