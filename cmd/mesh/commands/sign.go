// Copyright (C) 2026 Fernando Martin Lopez. All Rights Reserved.
// SPDX-License-Identifier: AGPL-3.0-only WITH Commons-Clause-1.0
//
// This file is part of Web5-Mesh — sovereign network kernel prototype (Fase 1).
// Use of this source code is governed by the AGPLv3 + Commons Clause
// license that can be found in the LICENSE file at the root of this repo.
//
// Commercial use, SaaS deployment, or resale without a commercial license
// agreement is strictly prohibited. Contact the author for licensing.

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
	Register("sign", cmdSign)
}

func cmdSign(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: sign <nombre_archivo_en_bóveda>"
	}

	filename := args[0]
	
	// Buscar el archivo en la bóveda soberana
	filePath := filepath.Join(getSecureDir(), filename)
	
	// Leer archivo
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Sprintf("❌ Error leyendo archivo: %v\n💡 Asegurate de que el archivo exista en la bóveda. Usá /ls para ver el contenido.", err)
	}

	// Calcular hash SHA256
	hash := sha256.Sum256(data)
	hashHex := hex.EncodeToString(hash[:])

	// Firmar el hash con clave privada Ed25519
	sig := id.SignMessage(hash[:])
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	// Obtener información del archivo
	fileInfo, _ := os.Stat(filePath)
	fileSize := fileInfo.Size()

	// Crear metadata de la firma
	sigFilename := filename + ".sig"
	sigData := map[string]interface{}{
		"hash":      hashHex,
		"signature": sigB64,
		"timestamp": time.Now().Unix(),
		"signer":    id.DID,
		"filename":  filename,
		"size":      fileSize,
		"algorithm": "SHA256+Ed25519",
	}
	
	sigJSON, _ := json.MarshalIndent(sigData, "", "  ")
	
	// Guardar la firma en la bóveda
	sigPath := filepath.Join(getSecureDir(), sigFilename)
	err = os.WriteFile(sigPath, sigJSON, 0600)
	if err != nil {
		return fmt.Sprintf("❌ Error guardando firma: %v", err)
	}

	return fmt.Sprintf("✅ Archivo firmado criptográficamente:\n   ├── Archivo: %s (%d bytes)\n   ├── Hash SHA256: %s...\n   ├── Firma: %s\n   ├── Firmante: %s\n   └── Timestamp: %s", 
		filename, fileSize, hashHex[:16], sigPath, id.DID[:20]+"...", time.Now().Format("2006-01-02 15:04:05"))
}
