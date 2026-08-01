package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"web5-mesh/src/crypto"
)

func init() {
	Register("acl", cmdACL)
}

type PeerInfo struct {
	PubKeyEd string `json:"pubkey_ed"`
	PubKeyX  string `json:"pubkey_x"`
}

type ACL struct {
	Peers map[string]PeerInfo `json:"peers"`
}

// getACLPath SIEMPRE usa el directorio actual
func getACLPath() string {
	return "acl.json"
}

func loadLocalACL() ACL {
	acl := ACL{Peers: make(map[string]PeerInfo)}
	data, err := os.ReadFile(getACLPath())
	if err == nil {
		json.Unmarshal(data, &acl)
	}
	return acl
}

func saveLocalACL(acl ACL) error {
	newData, err := json.MarshalIndent(acl, "", "  ")
	if err != nil {
		return err
	}
	// FIX 18: permisos 0600 (solo owner lee/escribe).
	// Antes usaba 0644 que permitía lectura a cualquier usuario del sistema.
	return os.WriteFile(getACLPath(), newData, 0600)
}

func cmdACL(args []string, id *crypto.Identity) string {
	if len(args) < 1 {
		return "Uso: acl [add <did> | import <did> <pubkey_ed> <pubkey_x> | remove <did> | clear | list]"
	}

	switch strings.ToLower(args[0]) {
	case "list":
		acl := loadLocalACL()
		if len(acl.Peers) == 0 {
			return "📋 Tu lista de confianza está vacía."
		}
		var result strings.Builder
		result.WriteString("📋 NODOS DE CONFIANZA (ACL):\n")
		for did, info := range acl.Peers {
			hasKeys := "✅"
			if info.PubKeyEd == "" || info.PubKeyX == "" {
				hasKeys = "⚠️ (faltan claves)"
			}
			result.WriteString(fmt.Sprintf("  %s %s\n", hasKeys, did))
		}
		return strings.TrimSpace(result.String())

	case "add":
		if len(args) < 2 {
			return "Uso: acl add <did>"
		}
		targetDID := args[1]
		acl := loadLocalACL()
		if _, exists := acl.Peers[targetDID]; exists {
			return fmt.Sprintf("✅ El DID %s ya está en tu lista de confianza.", targetDID)
		}
		acl.Peers[targetDID] = PeerInfo{PubKeyEd: "", PubKeyX: ""}
		if err := saveLocalACL(acl); err != nil {
			return fmt.Sprintf("❌ Error guardando ACL: %v", err)
		}
		return fmt.Sprintf("✅ DID '%s' agregado. Usá 'acl import' para agregar las claves públicas.", targetDID)

	case "import":
		if len(args) < 4 {
			return "Uso: acl import <did> <pubkey_ed> <pubkey_x>"
		}
		targetDID, pubKeyEd, pubKeyX := args[1], args[2], args[3]
		acl := loadLocalACL()
		acl.Peers[targetDID] = PeerInfo{PubKeyEd: pubKeyEd, PubKeyX: pubKeyX}
		if err := saveLocalACL(acl); err != nil {
			return fmt.Sprintf("❌ Error guardando ACL: %v", err)
		}
		return fmt.Sprintf("✅ Claves públicas importadas correctamente para %s", targetDID)

	case "remove":
		if len(args) < 2 {
			return "Uso: acl remove <did>"
		}
		targetDID := args[1]
		acl := loadLocalACL()
		if _, exists := acl.Peers[targetDID]; !exists {
			return fmt.Sprintf("❌ El DID %s no está en tu lista de confianza.", targetDID)
		}
		delete(acl.Peers, targetDID)
		if err := saveLocalACL(acl); err != nil {
			return fmt.Sprintf("❌ Error guardando ACL: %v", err)
		}
		return fmt.Sprintf("✅ DID '%s' eliminado de la lista de confianza.", targetDID)

	case "clear":
		acl := ACL{Peers: make(map[string]PeerInfo)}
		if err := saveLocalACL(acl); err != nil {
			return fmt.Sprintf("❌ Error limpiando ACL: %v", err)
		}
		return "✅ Lista de confianza limpiada completamente."

	default:
		return "Uso: acl [add <did> | import <did> <pubkey_ed> <pubkey_x> | remove <did> | clear | list]"
	}
}
