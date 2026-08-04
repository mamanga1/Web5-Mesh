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
	"fmt"
	"os"
	"web5-mesh/src/crypto"
)

func init() {
	Register("/status", cmdStatus)
}

func cmdStatus(args []string, id *crypto.Identity) string {
	// Si el argumento es "clear", borrar solo la ACL
	if len(args) > 0 && args[0] == "clear" {
		_, err := os.Stat("acl.json")
		if err != nil {
			return "ℹ️ No hay ACL para borrar."
		}

		err = os.Remove("acl.json")
		if err != nil {
			return fmt.Sprintf("❌ Error borrando acl.json: %v", err)
		}

		return "✅ Lista de pares de confianza limpiada:\n   └── acl.json eliminado\n   💡 Usá 'acl import' para agregar nuevos pares."
	}

	// Mostrar estado normal
	acl, err := crypto.LoadACL()
	if err != nil {
		return fmt.Sprintf("❌ Error cargando ACL: %v", err)
	}

	result := fmt.Sprintf("🟢 ESTADO DE LA RED XION\n")
	result += fmt.Sprintf("├── Mi DID: %s\n", id.DID)
	result += fmt.Sprintf("├── Pares de confianza: %d\n", len(acl.Peers))
	result += "└── Detalle de pares:\n"

	if len(acl.Peers) == 0 {
		result += "    (Ningún par configurado. Usá 'acl import' para agregar nodos)\n"
	} else {
		for did, peer := range acl.Peers {
			// Acortar el DID para que se vea prolijo en pantalla
			shortDID := did
			if len(did) > 24 {
				shortDID = did[:10] + "..." + did[len(did)-10:]
			}

			hasEd := peer.PubKeyEd != ""
			hasX := peer.PubKeyX != ""
			estado := "⚪ Pendiente"
			if hasEd && hasX {
				estado = "🟢 Configurado"
			} else if hasEd || hasX {
				estado = "🟡 Parcial"
			}

			result += fmt.Sprintf("    ├── %s (%s)\n", shortDID, estado)
		}
	}
	return result
}
