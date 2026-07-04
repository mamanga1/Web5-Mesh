package commands

import (
	"fmt"
	"strings"
	"time"
	"web5-mesh/src/crypto"
)

var (
	faroRunning = false
	faroLog     = []string{}
)

func init() {
	Register("/faro", cmdFaro)
}

func cmdFaro(args []string, id *crypto.Identity) string {
	if len(args) == 0 {
		return "Uso: /faro [on|off|log]"
	}

	switch strings.ToLower(args[0]) {
	case "on":
		if faroRunning {
			return "⚠️ El FARO ya está activo en este shell."
		}
		faroRunning = true
		faroLog = append(faroLog, fmt.Sprintf("[%s] FARO iniciado en 0.0.0.0:54321", time.Now().Format("15:04:05")))
		go startFaroEngine() // Aquí llamamos a la lógica real del faro en background
		return "✅ FARO iniciado en segundo plano (Puerto 54321)"

	case "off":
		if !faroRunning {
			return "⚠️ El FARO no está activo."
		}
		faroRunning = false
		faroLog = append(faroLog, fmt.Sprintf("[%s] FARO detenido", time.Now().Format("15:04:05")))
		stopFaroEngine() // Señal para detener el engine
		return "✅ FARO detenido"

	case "log":
		if len(faroLog) == 0 {
			return "📋 No hay actividad registrada en el FARO."
		}
		return "📋 LOG DEL FARO:\n" + strings.Join(faroLog, "\n")

	default:
		return fmt.Sprintf("Argumento no reconocido: %s. Use: on, off, log", args[0])
	}
}

// Placeholder para la integración real. 
// Mañana conectamos esto con la lógica real de cmd/faro/main.go
func startFaroEngine() {
	// TODO: Iniciar listener UDP del faro aquí
	faroLog = append(faroLog, fmt.Sprintf("[%s] [DEBUG] Relay escuchando...", time.Now().Format("15:04:05")))
}

func stopFaroEngine() {
	// TODO: Cerrar listener UDP del faro
	faroLog = append(faroLog, fmt.Sprintf("[%s] [DEBUG] Relay cerrado", time.Now().Format("15:04:05")))
}
