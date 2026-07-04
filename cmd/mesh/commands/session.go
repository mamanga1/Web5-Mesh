package commands

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"web5-mesh/src/crypto"
)

type Session struct {
	ID          int
	Type        string // "chat", "editor", "payment"
	Target      string // DID o nombre de archivo
	UnreadCount int
	IsActive    bool
}

var (
	sessions   = make(map[int]*Session)
	sessionMux sync.Mutex
	nextID     = 1
)

func init() {
	// Solo registramos /session aquí. /notif se registra en notif.go
	Register("/session", cmdSession)
}

func cmdSession(args []string, id *crypto.Identity) string {
	if len(args) == 0 {
		return "Uso: /session [list|new <tipo> <target>|attach <id>|detach]"
	}

	switch args[0] {
	case "list":
		sessionMux.Lock()
		defer sessionMux.Unlock()
		if len(sessions) == 0 {
			return "📋 No hay sesiones activas."
		}
		var result strings.Builder
		result.WriteString("📋 SESIONES ACTIVAS:\n")
		for _, s := range sessions {
			status := "🟢 Activa"
			if !s.IsActive {
				status = "⚪ Background"
			}
			unread := ""
			if s.UnreadCount > 0 {
				unread = fmt.Sprintf(" (🔴 %d sin leer)", s.UnreadCount)
			}
			result.WriteString(fmt.Sprintf("  [%d] %s: %s %s %s\n", s.ID, s.Type, s.Target, status, unread))
		}
		return strings.TrimSpace(result.String())

	case "new":
		if len(args) < 3 {
			return "Uso: /session new <tipo> <target> (ej: /session new chat did:maia:xxx)"
		}
		sessionMux.Lock()
		sid := nextID
		nextID++
		sessions[sid] = &Session{
			ID:       sid,
			Type:     args[1],
			Target:   args[2],
			IsActive: true,
		}
		sessionMux.Unlock()
		return fmt.Sprintf("✅ Sesión [%d] creada: %s con %s", sid, args[1], args[2])

	case "attach":
		if len(args) < 2 {
			return "Uso: /session attach <id>"
		}
		sid, err := strconv.Atoi(args[1])
		if err != nil {
			return "❌ ID de sesión no válido."
		}
		sessionMux.Lock()
		s, exists := sessions[sid]
		if exists {
			s.IsActive = true
			s.UnreadCount = 0 // Marcar como leído al adjuntar
		}
		sessionMux.Unlock()
		if !exists {
			return "❌ Sesión no encontrada."
		}
		return fmt.Sprintf("🔗 Conectado a la Sesión [%d] (%s: %s).", sid, s.Type, s.Target)

	case "detach":
		sessionMux.Lock()
		var activeSession *Session
		for _, s := range sessions {
			if s.IsActive {
				activeSession = s
				break
			}
		}
		if activeSession != nil {
			activeSession.IsActive = false
			sessionMux.Unlock()
			return fmt.Sprintf("⏸️ Sesión [%d] puesta en background.", activeSession.ID)
		}
		sessionMux.Unlock()
		return "⚠️ No hay ninguna sesión activa para desconectar."

	default:
		return "Uso: /session [list|new <tipo> <target>|attach <id>|detach]"
	}
}

// NotifySession permite que otros módulos (como el futuro chat) envíen alertas a una sesión específica
func NotifySession(sessionID int, msgType, message string) {
	sessionMux.Lock()
	s, exists := sessions[sessionID]
	if exists && !s.IsActive {
		s.UnreadCount++
	}
	sessionMux.Unlock()

	notifText := message
	if exists {
		notifText = fmt.Sprintf("[Sesión %d] %s", sessionID, message)
	}
	SendNotification(Notification{Type: msgType, Message: notifText})
}

// CreateSession crea una nueva sesión y retorna el ID (exportada para uso desde shell.go)
func CreateSession(sessionType, target string) int {
	sessionMux.Lock()
	defer sessionMux.Unlock()
	
	sid := nextID
	nextID++
	sessions[sid] = &Session{
		ID:       sid,
		Type:     sessionType,
		Target:   target,
		IsActive: true,
	}
	return sid
}
