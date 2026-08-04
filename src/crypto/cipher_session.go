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
	"sync"
	"time"
)

type CipherSession struct {
	Handshake  *HandshakeState
	LastRekey  time.Time
	RekeyEvery time.Duration
	mu         sync.Mutex
}

func NewCipherSession(hs *HandshakeState) *CipherSession {
	return &CipherSession{
		Handshake:  hs,
		LastRekey:  time.Now(),
		RekeyEvery: 5 * time.Minute,
	}
}

func (s *CipherSession) Encrypt(plaintext []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Since(s.LastRekey) > s.RekeyEvery {
		s.Handshake.Rekey()
		s.LastRekey = time.Now()
	}

	return s.Handshake.Encrypt(plaintext)
}

func (s *CipherSession) Decrypt(ciphertext []byte) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.Handshake.Decrypt(ciphertext)
}

func (s *CipherSession) ForceRekey() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Handshake.Rekey()
	s.LastRekey = time.Now()
}
