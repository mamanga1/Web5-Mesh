package commands

import (
	"fmt"
	"sync"
)

type Notification struct {
	Type    string // "info", "alert", "msg"
	Message string
}

var (
	notifQueue   = make(chan Notification, 100)
	notifEnabled = true
	notifMutex   sync.Mutex
)

func SetNotifications(enabled bool) {
	notifMutex.Lock()
	defer notifMutex.Unlock()
	notifEnabled = enabled
}

func AreNotificationsEnabled() bool {
	notifMutex.Lock()
	defer notifMutex.Unlock()
	return notifEnabled
}

func SendNotification(notif Notification) {
	select {
	case notifQueue <- notif:
	default:
		// Cola llena, descartamos la más antigua para no bloquear (seguridad ante todo)
		<-notifQueue
		notifQueue <- notif
	}
}

// FlushNotifications imprime las alertas pendientes justo antes del prompt.
// Es 100% seguro porque no interrumpe la lectura de teclado en curso.
func FlushNotifications() string {
	var output string
	
	notifMutex.Lock()
	enabled := notifEnabled
	notifMutex.Unlock()

	if !enabled {
		// Si está en modo zen, vaciamos la cola silenciosamente
		for len(notifQueue) > 0 {
			<-notifQueue
		}
		return ""
	}

	// Imprimir todas las pendientes
	for len(notifQueue) > 0 {
		notif := <-notifQueue
		icon := "🔔"
		if notif.Type == "alert" {
			icon = "⚠️"
		} else if notif.Type == "msg" {
			icon = "💬"
		}
		output += fmt.Sprintf("\n%s [SISTEMA] %s", icon, notif.Message)
	}
	
	return output
}
