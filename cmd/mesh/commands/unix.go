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
	"io"
	"os"
	"path/filepath"
	"strings"

	"web5-mesh/src/crypto"
)

// getSecureDir devuelve la ruta a la bóveda soberana (relativa al directorio de ejecución)
func getSecureDir() string {
	return filepath.Join(".xion", "workspace")
}

func init() {
	// Registrar comandos con y sin /
	Register("/pwd", cmdPwd)
	Register("pwd", cmdPwd)
	Register("/ls", cmdLs)
	Register("ls", cmdLs)
	Register("/mkdir", cmdMkdir)
	Register("mkdir", cmdMkdir)
	Register("/cat", cmdCat)
	Register("cat", cmdCat)
	Register("/rm", cmdRm)
	Register("rm", cmdRm)
	Register("/rmdir", cmdRmdir)
	Register("rmdir", cmdRmdir)
	Register("/mv", cmdMv)
	Register("mv", cmdMv)
	Register("/cp", cmdCp)
	Register("cp", cmdCp)
	Register("/touch", cmdTouch)
	Register("touch", cmdTouch)

	// Crear el directorio si no existe al iniciar
	workspaceDir := getSecureDir()
	os.MkdirAll(workspaceDir, 0700)
}

func cmdPwd(args []string, id *crypto.Identity) string {
	absPath, _ := filepath.Abs(getSecureDir())
	return fmt.Sprintf("📂 %s", absPath)
}

func cmdLs(args []string, id *crypto.Identity) string {
	workspaceDir := getSecureDir()
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return fmt.Sprintf("❌ Error leyendo directorio: %v", err)
	}

	if len(entries) == 0 {
		return "📂 El directorio está vacío. Usa mkdir <nombre> para crear algo."
	}

	result := "📂 Contenido de tu espacio de trabajo:\n"
	for _, entry := range entries {
		if entry.IsDir() {
			result += fmt.Sprintf("  📁 %s/\n", entry.Name())
		} else {
			info, _ := entry.Info()
			size := info.Size()
			result += fmt.Sprintf("  📄 %s (%d bytes)\n", entry.Name(), size)
		}
	}
	return result
}

func cmdMkdir(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: mkdir <nombre_de_carpeta>"
	}

	dirName := args[0]
	if dirName == ".." || dirName == "/" || dirName == "." || strings.Contains(dirName, "/") || strings.Contains(dirName, "\\") {
		return "⚠️ Nombre de directorio no válido."
	}

	targetPath := filepath.Join(getSecureDir(), dirName)
	err := os.MkdirAll(targetPath, 0700)
	if err != nil {
		return fmt.Sprintf("❌ Error creando directorio: %v", err)
	}

	return fmt.Sprintf("✅ Directorio '%s' creado en tu espacio soberano.", dirName)
}

func cmdCat(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: cat <nombre_archivo>"
	}

	fileName := args[0]
	if strings.Contains(fileName, "/") || strings.Contains(fileName, "..") || strings.Contains(fileName, "\\") {
		return "⚠️ Ruta no válida. Solo nombres de archivo en el directorio actual."
	}

	targetPath := filepath.Join(getSecureDir(), fileName)
	content, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Sprintf("❌ No se pudo leer el archivo: %v", err)
	}

	return fmt.Sprintf("📄 --- %s ---\n%s\n--- FIN ---", fileName, string(content))
}

func cmdRm(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: rm <nombre_archivo>"
	}

	fileName := args[0]
	if strings.Contains(fileName, "/") || strings.Contains(fileName, "..") || strings.Contains(fileName, "\\") {
		return "⚠️ Ruta no válida. Solo nombres de archivo en el directorio actual."
	}

	targetPath := filepath.Join(getSecureDir(), fileName)
	err := os.Remove(targetPath)
	if err != nil {
		return fmt.Sprintf("❌ Error borrando archivo: %v", err)
	}

	return fmt.Sprintf("✅ Archivo '%s' borrado.", fileName)
}

func cmdRmdir(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: rmdir <nombre_carpeta>"
	}

	dirName := args[0]
	if dirName == ".." || dirName == "/" || dirName == "." || strings.Contains(dirName, "/") || strings.Contains(dirName, "\\") {
		return "⚠️ Nombre de directorio no válido."
	}

	targetPath := filepath.Join(getSecureDir(), dirName)

	// Verificar que esté vacía
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return fmt.Sprintf("❌ Error leyendo directorio: %v", err)
	}

	if len(entries) > 0 {
		return fmt.Sprintf("❌ El directorio no está vacío (tiene %d elementos). Borra el contenido primero.", len(entries))
	}

	err = os.Remove(targetPath)
	if err != nil {
		return fmt.Sprintf("❌ Error borrando directorio: %v", err)
	}

	return fmt.Sprintf("✅ Directorio '%s' borrado.", dirName)
}

func cmdMv(args []string, id *crypto.Identity) string {
	if len(args) < 2 {
		return "Uso: mv <origen> <destino>"
	}

	srcName := args[0]
	dstName := args[1]

	if strings.Contains(srcName, "/") || strings.Contains(srcName, "..") || strings.Contains(srcName, "\\") ||
		strings.Contains(dstName, "/") || strings.Contains(dstName, "..") || strings.Contains(dstName, "\\") {
		return "⚠️ Rutas no válidas. Solo nombres en el directorio actual."
	}

	srcPath := filepath.Join(getSecureDir(), srcName)
	dstPath := filepath.Join(getSecureDir(), dstName)

	err := os.Rename(srcPath, dstPath)
	if err != nil {
		return fmt.Sprintf("❌ Error moviendo archivo: %v", err)
	}

	return fmt.Sprintf("✅ '%s' movido a '%s'.", srcName, dstName)
}

func cmdCp(args []string, id *crypto.Identity) string {
	if len(args) < 2 {
		return "Uso: cp <origen> <destino>"
	}

	srcName := args[0]
	dstName := args[1]

	if strings.Contains(srcName, "/") || strings.Contains(srcName, "..") || strings.Contains(srcName, "\\") ||
		strings.Contains(dstName, "/") || strings.Contains(dstName, "..") || strings.Contains(dstName, "\\") {
		return "⚠️ Rutas no válidas. Solo nombres en el directorio actual."
	}

	srcPath := filepath.Join(getSecureDir(), srcName)
	dstPath := filepath.Join(getSecureDir(), dstName)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Sprintf("❌ No se pudo abrir origen: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Sprintf("❌ No se pudo crear destino: %v", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Sprintf("❌ Error copiando: %v", err)
	}

	return fmt.Sprintf("✅ '%s' copiado a '%s'.", srcName, dstName)
}

func cmdTouch(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: touch <nombre_archivo>"
	}

	fileName := args[0]
	if strings.Contains(fileName, "/") || strings.Contains(fileName, "..") || strings.Contains(fileName, "\\") {
		return "⚠️ Nombre no válido."
	}

	targetPath := filepath.Join(getSecureDir(), fileName)

	// Crear archivo vacío o actualizar timestamp con permisos 0600
	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Sprintf("❌ Error creando archivo: %v", err)
	}
	file.Close()

	return fmt.Sprintf("✅ Archivo '%s' creado/actualizado.", fileName)
}
