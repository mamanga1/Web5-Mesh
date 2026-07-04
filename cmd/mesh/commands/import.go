package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"web5-mesh/src/crypto"
)

func init() {
	Register("import", cmdImport)
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func cmdImport(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: import <ruta_archivo_host> [nombre_en_bóveda]"
	}

	srcPath := expandHome(args[0])
	destName := filepath.Base(srcPath)
	if len(args) > 1 {
		destName = args[1]
	}

	secureDir := getSecureDir()

	// Crear bóveda si no existe
	err := os.MkdirAll(secureDir, 0700)
	if err != nil {
		return fmt.Sprintf("❌ Error creando bóveda: %v", err)
	}

	// Abrir archivo origen (puede estar en cualquier parte del sistema)
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Sprintf("❌ No se pudo leer: %v", err)
	}
	defer srcFile.Close()

	// Crear archivo destino en la bóveda
	destPath := filepath.Join(secureDir, destName)
	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Sprintf("❌ No se pudo crear en bóveda: %v", err)
	}
	defer destFile.Close()

	// Copiar bytes
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Sprintf("❌ Error copiando: %v", err)
	}

	return fmt.Sprintf("✅ Archivo ingresado a la bóveda:\n   ├── Origen: %s\n   ├── Destino: %s\n   └── Permisos: 0600 (solo tú)", srcPath, destPath)
}
