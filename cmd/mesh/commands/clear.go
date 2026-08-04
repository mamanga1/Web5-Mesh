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
"path/filepath"

"web5-mesh/src/crypto"
)

func init() {
Register("clear", cmdClear)
}

func cmdClear(args []string, id *crypto.Identity) string {
// Requerir flag --force
if len(args) == 0 || args[0] != "--force" {
return "⚠️  Este comando borra tu identidad, ACL y todos los datos locales.\n   Usá 'clear --force' para confirmar."
}

// Archivos a eliminar (en directorio actual y en ~/.xion/)
files := []string{
"node.key",
"acl.json",
"aliases.json",
"alias.json",
".shell_history",
"config.json",
}

var deleted []string

// 1. Eliminar archivos en el directorio actual
for _, f := range files {
if _, err := os.Stat(f); err == nil {
if err := os.Remove(f); err == nil {
deleted = append(deleted, f)
}
}
}

// 2. Eliminar archivos en ~/.xion/
home, err := os.UserHomeDir()
if err == nil {
xionDir := filepath.Join(home, ".xion")
for _, f := range files {
path := filepath.Join(xionDir, f)
if _, err := os.Stat(path); err == nil {
if err := os.Remove(path); err == nil {
deleted = append(deleted, "~/.xion/"+f)
}
}
}
// Eliminar el directorio .xion si está vacío
os.RemoveAll(xionDir)
deleted = append(deleted, "~/.xion/ (directorio)")
}

if len(deleted) == 0 {
return "ℹ️ No hay archivos de identidad, ACL, alias o configuración para borrar."
}

result := "✅ Jaula de Faraday reiniciada:\n"
for _, f := range deleted {
result += fmt.Sprintf("   ├── %s eliminado\n", f)
}
result += "   └── Reiniciá la shell para generar nueva identidad."

return result
}
