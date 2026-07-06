package commands

import (
	"fmt"
	"os"

	"web5-mesh/src/crypto"
)

func init() {
	Register("clear", cmdClear)
}

func cmdClear(args []string, id *crypto.Identity) string {
	// Rechazar argumentos
	if len(args) > 0 {
		return "❌ Uso: clear (sin argumentos)\n   Este comando borra tu identidad y ACL."
	}

	// Verificar si los archivos existen
	_, errKey := os.Stat("node.key")
	_, errACL := os.Stat("acl.json")

	if errKey != nil && errACL != nil {
		return "ℹ️ No hay archivos de identidad ni ACL para borrar."
	}

	// Pedir confirmación
	fmt.Print("⚠️  ¿Estás seguro? Esto borrará tu identidad y ACL. (s/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	
	if confirm != "s" && confirm != "S" && confirm != "si" && confirm != "SI" {
		return "❌ Operación cancelada."
	}

	// Borrar node.key (identidad)
	if errKey == nil {
		err := os.Remove("node.key")
		if err != nil {
			return fmt.Sprintf("❌ Error borrando node.key: %v", err)
		}
	}

	// Borrar acl.json (lista de confianza)
	if errACL == nil {
		err := os.Remove("acl.json")
		if err != nil {
			return fmt.Sprintf("❌ Error borrando acl.json: %v", err)
		}
	}

	return "✅ Jaula de Faraday reiniciada:\n   ├── node.key eliminado (identidad borrada)\n   ├── acl.json eliminado (pares de confianza borrados)\n   └── Reiniciá la shell para generar nueva identidad."
}
