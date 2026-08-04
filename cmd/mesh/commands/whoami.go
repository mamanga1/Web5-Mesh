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
	"encoding/hex"
	"fmt"
	"web5-mesh/src/crypto"
)

func init() {
	Register("whoami", cmdWhoami)
}

func cmdWhoami(args []string, id *crypto.Identity) string {
	if id == nil {
		return "❌ No se pudo cargar la identidad"
	}

	pubEdHex := hex.EncodeToString(id.PubKeyEd)
	
	var pubXHex string
	if id.PubKeyX != nil {
		pubXHex = hex.EncodeToString(id.PubKeyX[:])
	} else {
		pubXHex = "(no disponible)"
	}

	return fmt.Sprintf(`🆔 TU IDENTIDAD XIONIA:

  DID:        %s
  PubKey Ed:  %s
  PubKey X:   %s

📋 Para agregar este nodo a otro peer, copiá esta línea:
  acl import %s %s %s
`, id.DID, pubEdHex, pubXHex, id.DID, pubEdHex, pubXHex)
}
