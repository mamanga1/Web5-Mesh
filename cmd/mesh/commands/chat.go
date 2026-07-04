package commands

import (
	"fmt"
	"strings"
	"web5-mesh/src/crypto"
)

func init() {
	Register("chat", cmdChat)
}

func cmdChat(args []string, id *crypto.Identity) string {
	if len(args) < 2 {
		return "Uso: chat <did|alias> <mensaje>\nEj: chat amigo 'Hola, ¿cómo estás?'"
	}

	target := args[0]
	message := strings.Join(args[1:], " ")

	// Resolver alias → DID
	targetDID, isAlias := crypto.ResolveNode(target)
	if isAlias {
		fmt.Printf("🔗 Alias resuelto: %s → %s\n", target, targetDID)
	}

	if NetworkExecutor == nil {
		return "❌ NetworkExecutor no inicializado"
	}

	// Delegar al NetworkExecutor
	result := NetworkExecutor(id, targetDID, "CHAT:"+message)

	if strings.HasPrefix(result, "📩 ") {
		respuesta := strings.TrimPrefix(result, "📩 ")
		return fmt.Sprintf("✅ Mensaje enviado a %s\n💬 Respuesta: %s", target, respuesta)
	}

	return result
}
