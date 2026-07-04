package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"web5-mesh/src/crypto"
)

func init() {
	Register("export", cmdExport)
}

func cmdExport(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: export <nombre_archivo_en_bóveda> [ruta_destino_host]"
	}

	filename := args[0]
	
	// Ruta origen en la bóveda
	srcPath := filepath.Join(getSecureDir(), filename)
	
	// Ruta destino en el host
	var destPath string
	if len(args) > 1 {
		destPath = expandHome(args[1])
	} else {
		// Por defecto, exportar al home del usuario
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Sprintf("❌ No se pudo determinar el directorio home: %v", err)
		}
		destPath = filepath.Join(home, filename)
	}

	// Abrir archivo de la bóveda
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Sprintf("❌ No se encontró el archivo en la bóveda: %v\n💡 Usá /ls para ver el contenido de tu bóveda.", err)
	}
	defer srcFile.Close()

	// Crear archivo destino en el host
	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Sprintf("❌ No se pudo crear archivo en el host: %v", err)
	}
	defer destFile.Close()

	// Copiar bytes
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Sprintf("❌ Error copiando archivo: %v", err)
	}

	return fmt.Sprintf("✅ Archivo exportado de la bóveda:\n   ├── Origen (Bóveda): %s\n   ├── Destino (Host): %s\n   └── Permisos: 0644 (listo para compartir)", srcPath, destPath)
}

