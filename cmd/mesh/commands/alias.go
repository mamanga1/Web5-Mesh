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
	"strings"
	"web5-mesh/src/crypto"
)

func init() {
	Register("/alias", cmdAlias)
}

func cmdAlias(args []string, id *crypto.Identity) string {
	if len(args) == 0 {
		return "Uso: /alias [add|remove|list] [nombre] [did]\nEj: /alias add amigo did:maia:xxx"
	}

	switch strings.ToLower(args[0]) {
	case "add":
		if len(args) < 3 {
			return "Uso: /alias add <nombre> <did>"
		}
		alias := strings.ToLower(args[1])
		did := args[2]
		if !strings.HasPrefix(did, "did:maia:") {
			return "❌ DID inválido. Debe empezar con did:maia:"
		}
		if err := crypto.AddAlias(alias, did); err != nil {
			return fmt.Sprintf("❌ Error guardando alias: %v", err)
		}
		return fmt.Sprintf("✅ Alias guardado: %s → %s", alias, did)

	case "remove":
		if len(args) < 2 {
			return "Uso: /alias remove <nombre>"
		}
		alias := strings.ToLower(args[1])
		if err := crypto.RemoveAlias(alias); err != nil {
			return fmt.Sprintf("❌ Error eliminando alias: %v", err)
		}
		return fmt.Sprintf("✅ Alias eliminado: %s", alias)

	case "list":
		store, err := crypto.LoadAliases()
		if err != nil {
			return fmt.Sprintf("❌ Error cargando alias: %v", err)
		}
		if len(store.Aliases) == 0 {
			return "📋 No hay alias guardados."
		}
		var sb strings.Builder
		sb.WriteString("📋 ALIAS GUARDADOS:\n")
		for alias, did := range store.Aliases {
			// Truncar DID para que quepa
			short := did
			if len(did) > 30 {
				short = did[:15] + "..." + did[len(did)-10:]
			}
			sb.WriteString(fmt.Sprintf("  %s → %s\n", alias, short))
		}
		return strings.TrimSpace(sb.String())

	default:
		return "Uso: /alias [add|remove|list]"
	}
}
