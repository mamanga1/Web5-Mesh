// Copyright (C) 2026 Fernando Martin Lopez. All Rights Reserved.
// SPDX-License-Identifier: AGPL-3.0-only WITH Commons-Clause-1.0
//
// This file is part of Web5-Mesh — sovereign network kernel prototype (Fase 1).
// Use of this source code is governed by the AGPLv3 + Commons Clause
// license that can be found in the LICENSE file at the root of this repo.
//
// Commercial use, SaaS deployment, or resale without a commercial license
// agreement is strictly prohibited. Contact the author for licensing.

package crypto

import (
	"encoding/binary"
	"errors"
	"sync/atomic"

	"golang.org/x/crypto/chacha20poly1305"
)

// DeriveKeyID toma los primeros 4 bytes de PubKeyX para indexación O(1)
func DeriveKeyID(pubKeyX []byte) [4]byte {
	var id [4]byte
	if len(pubKeyX) >= 4 {
		copy(id[:], pubKeyX[:4])
	}
	return id
}

// FIX 13: contador atómico para nonces determinísticos.
// El nonce aleatorio de 12 bytes tiene riesgo de colisión a ~2^48 mensajes.
// Con contador, la colisión es imposible (se necesitan 2^64 mensajes).
var nonceCounter uint64

func EncryptPayload(sharedKey []byte, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(sharedKey)
	if err != nil {
		return nil, err
	}

	// Nonce de 12 bytes: 4 bytes en cero + 8 bytes de contador little-endian.
	// Cada llamada a EncryptPayload incrementa el contador atómicamente.
	nonce := make([]byte, aead.NonceSize())
	counter := atomic.AddUint64(&nonceCounter, 1)
	binary.LittleEndian.PutUint64(nonce[4:], counter)

	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

func DecryptPayload(sharedKey []byte, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(sharedKey)
	if err != nil {
		return nil, err
	}
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize+aead.Overhead() {
		return nil, errors.New("payload corrupto o demasiado corto")
	}
	nonce := ciphertext[:nonceSize]
	actualCipher := ciphertext[nonceSize:]
	return aead.Open(nil, nonce, actualCipher, nil)
}
