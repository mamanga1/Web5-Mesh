package commands

import (
	"fmt"
	"strings"
	"web5-mesh/src/crypto"
)

// Estado global de las notificaciones (por defecto ON)
var NotificationsEnabled = true

func init() {
	Register("/notif", cmdNotif)
}

func cmdNotif(args []string, id *crypto.Identity) string {
	if len(args) == 0 {
		if NotificationsEnabled {
			return "🔔 Notificaciones: ON (Modo estándar)"
		}
		return "🔕 Notificaciones: OFF (Modo Zen / Sin interrupciones)"
	}

	switch strings.ToLower(args[0]) {
	case "on":
		NotificationsEnabled = true
		return "🔔 Notificaciones activadas. Verás alertas de la red."
	case "off":
		NotificationsEnabled = false
		return "🔕 Notificaciones desactivadas. Modo Zen activado."
	default:
		return "Uso: /notif [on|off]"
	}
}

// Función auxiliar para que otros módulos envíen notificaciones
func NotifyIfEnabled(level, message string) {
	if NotificationsEnabled {
		icon := "🔔"
		if level == "alert" {
			icon = "⚠️"
		} else if level == "msg" {
			icon = "💬"
		}
		fmt.Printf("\n%s [SISTEMA] %s\n", icon, message)
		fmt.Print("u2p@nodo:~$ ") // Redibuja el prompt para no perder el hilo
	}
}
