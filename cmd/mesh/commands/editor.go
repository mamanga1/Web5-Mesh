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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"web5-mesh/src/crypto"
)

func init() {
	Register("/edit", cmdEdit)
}

func cmdEdit(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: /edit <nombre_archivo>"
	}

	fileName := args[0]
	if strings.Contains(fileName, "/") || strings.Contains(fileName, "..") {
		return "⚠️ Nombre de archivo no válido."
	}

	targetPath := filepath.Join(getSecureDir(), fileName)

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  EDITOR SOBERANO: %s\n", fileName)
	fmt.Println("  Escribe tu texto. Usa ':wq' para guardar y salir, ':q' para salir sin guardar.")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	var lines []string
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()

		if line == ":q" {
			return "⚠️ Salida sin guardar cambios."
		}
		if line == ":wq" {
			break
		}

		lines = append(lines, line)
	}

	// Guardar en el archivo
	content := strings.Join(lines, "\n")
	err := os.WriteFile(targetPath, []byte(content), 0600)
	if err != nil {
		return fmt.Sprintf("❌ Error al guardar el archivo: %v", err)
	}

	return fmt.Sprintf("✅ Archivo '%s' guardado exitosamente en tu espacio soberano.", fileName)
}
