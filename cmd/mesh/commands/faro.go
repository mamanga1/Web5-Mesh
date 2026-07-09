package commands

import (
	"fmt"
	"web5-mesh/src/config"
	"web5-mesh/src/crypto"
)

func init() {
	Register("faro", cmdFaro)
}

func cmdFaro(args []string, id *crypto.Identity) string {
	if len(args) == 0 {
		currentFaro := config.GetFaroAddr()
		return fmt.Sprintf("📡 Faro activo: %s\n\nUsá 'faro set <ip:puerto>' para cambiar\nUsá 'faro reset' para volver al faro público", currentFaro)
	}

	switch args[0] {
	case "set":
		if len(args) < 2 {
			return "❌ Uso: faro set <ip:puerto>"
		}
		addr := args[1]
		if err := config.SetFaroAddr(addr); err != nil {
			return fmt.Sprintf("❌ Error guardando configuración: %v", err)
		}
		return fmt.Sprintf("✅ Faro configurado: %s\nReiniciá la shell para aplicar los cambios.", addr)

	case "reset":
		if err := config.SetFaroAddr(config.DefaultFaro); err != nil {
			return fmt.Sprintf("❌ Error reseteando configuración: %v", err)
		}
		return fmt.Sprintf("✅ Faro reseteado a: %s\nReiniciá la shell para aplicar los cambios.", config.DefaultFaro)

	default:
		return "❌ Uso: faro [set <ip:puerto>|reset]"
	}
}
