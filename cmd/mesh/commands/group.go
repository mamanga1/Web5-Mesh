package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"web5-mesh/src/crypto"
)

func init() {
	Register("/group", cmdGroup)
}

func cmdGroup(args []string, id *crypto.Identity) string {
	if len(args) == 0 {
		return "Uso: /group [create|list|add|remove|delete|leave|send|info] [alias] [args...]"
	}

	switch strings.ToLower(args[0]) {
	case "create":
		if len(args) < 3 {
			return "Uso: /group create <alias> <nombre>\nEj: /group create devs 'Desarrolladores XionIA'"
		}
		alias := strings.ToLower(args[1])
		name := strings.Join(args[2:], " ")

		if err := crypto.CreateGroup(alias, name, id.DID); err != nil {
			return fmt.Sprintf("❌ Error creando grupo: %v", err)
		}

		return fmt.Sprintf("✅ Grupo creado: %s (%s)", alias, name)

	case "list":
		store, err := crypto.LoadGroups()
		if err != nil {
			return fmt.Sprintf("❌ Error cargando grupos: %v", err)
		}
		if len(store.Groups) == 0 {
			return "📋 No hay grupos creados."
		}

		var sb strings.Builder
		sb.WriteString("📋 GRUPOS:\n")
		for alias, group := range store.Groups {
			sb.WriteString(fmt.Sprintf("  [%s] %s (%d miembros) - admin: %s\n",
				alias, group.Name, len(group.Members), crypto.ResolveDID(group.Admin)))
		}
		return strings.TrimSpace(sb.String())

	case "add":
		if len(args) < 3 {
			return "Uso: /group add <alias_grupo> <did|alias>\nEj: /group add devs amigo"
		}
		groupAlias := strings.ToLower(args[1])
		target := args[2]

		group, exists := crypto.GetGroup(groupAlias)
		if !exists {
			return fmt.Sprintf("❌ Grupo %s no encontrado", groupAlias)
		}
		if group.Admin != id.DID {
			return "❌ Solo el admin puede agregar miembros"
		}

		targetDID, _ := crypto.ResolveNode(target)

		if err := crypto.AddMember(groupAlias, targetDID); err != nil {
			return fmt.Sprintf("❌ Error agregando miembro: %v", err)
		}

		if NetworkExecutor != nil && targetDID != id.DID {
			updatedGroup, _ := crypto.GetGroup(groupAlias)
			groupJSON, _ := json.Marshal(updatedGroup)
			syncMsg := fmt.Sprintf("GROUP_SYNC:%s:%s", groupAlias, string(groupJSON))
			NetworkExecutor(id, targetDID, syncMsg)
		}

		return fmt.Sprintf("✅ %s agregado al grupo %s\n📩 Grupo sincronizado con el nuevo miembro", target, groupAlias)

	case "remove":
		if len(args) < 2 {
			return "Uso: /group remove <alias_grupo> [did|alias]\n  Sin miembro: salís vos del grupo\n  Con miembro: el admin remueve a otro"
		}
		groupAlias := strings.ToLower(args[1])

		group, exists := crypto.GetGroup(groupAlias)
		if !exists {
			return fmt.Sprintf("❌ Grupo %s no encontrado", groupAlias)
		}

		// Si NO hay segundo argumento: el propio usuario sale del grupo
		if len(args) == 2 {
			if err := crypto.RemoveMember(groupAlias, id.DID); err != nil {
				return fmt.Sprintf("❌ Error saliendo del grupo: %v", err)
			}

			// Si era el admin y quedan miembros, el grupo se elimina
			if group.Admin == id.DID {
				crypto.DeleteGroup(groupAlias)

				if NetworkExecutor != nil {
					for _, memberDID := range group.Members {
						if memberDID == id.DID {
							continue
						}
						NetworkExecutor(id, memberDID, "GROUP_DELETE:"+groupAlias)
					}
				}

				return fmt.Sprintf("👋 Saliste del grupo %s\n🗑️ Grupo eliminado (eras el admin)\n📩 Miembros notificados", groupAlias)
			}

			// Si no era el admin, notificar al admin que salí
			if NetworkExecutor != nil {
				NetworkExecutor(id, group.Admin, fmt.Sprintf("GROUP_LEAVE:%s:%s", groupAlias, id.DID))
			}

			return fmt.Sprintf("👋 Saliste del grupo %s\n📩 Admin notificado", groupAlias)
		}

		// Si HAY segundo argumento: solo el admin puede remover a otros
		if group.Admin != id.DID {
			return "❌ Solo el admin puede remover miembros"
		}

		target := args[2]
		targetDID, _ := crypto.ResolveNode(target)

		if err := crypto.RemoveMember(groupAlias, targetDID); err != nil {
			return fmt.Sprintf("❌ Error removiendo miembro: %v", err)
		}

		if NetworkExecutor != nil {
			NetworkExecutor(id, targetDID, "GROUP_KICKED:"+groupAlias)
		}

		return fmt.Sprintf("✅ %s removido del grupo %s\n📩 Miembro notificado", target, groupAlias)

	case "leave":
		if len(args) < 2 {
			return "Uso: /group leave <alias_grupo>"
		}
		return cmdGroup([]string{"remove", args[1]}, id)

	case "delete":
		if len(args) < 2 {
			return "Uso: /group delete <alias_grupo>"
		}
		groupAlias := strings.ToLower(args[1])

		group, exists := crypto.GetGroup(groupAlias)
		if !exists {
			return fmt.Sprintf("❌ Grupo %s no encontrado", groupAlias)
		}
		if group.Admin != id.DID {
			return "❌ Solo el admin puede eliminar el grupo"
		}

		if err := crypto.DeleteGroup(groupAlias); err != nil {
			return fmt.Sprintf("❌ Error eliminando grupo: %v", err)
		}

		if NetworkExecutor != nil {
			for _, memberDID := range group.Members {
				if memberDID == id.DID {
					continue
				}
				NetworkExecutor(id, memberDID, "GROUP_DELETE:"+groupAlias)
			}
		}

		return fmt.Sprintf("✅ Grupo %s eliminado\n📩 Miembros notificados", groupAlias)

	case "send":
		if len(args) < 3 {
			return "Uso: /group send <alias_grupo> <mensaje>"
		}
		groupAlias := strings.ToLower(args[1])
		message := strings.Join(args[2:], " ")

		group, exists := crypto.GetGroup(groupAlias)
		if !exists {
			return fmt.Sprintf("❌ Grupo %s no encontrado", groupAlias)
		}

		if NetworkExecutor == nil {
			return "❌ NetworkExecutor no inicializado"
		}

		sentCount := 0
		for _, memberDID := range group.Members {
			if memberDID == id.DID {
				continue
			}
			NetworkExecutor(id, memberDID, "GROUP:"+groupAlias+":"+message)
			sentCount++
		}

		return fmt.Sprintf("✅ Mensaje enviado al grupo %s (%d miembros)", groupAlias, sentCount)

	case "info":
		if len(args) < 2 {
			return "Uso: /group info <alias_grupo>"
		}
		groupAlias := strings.ToLower(args[1])

		group, exists := crypto.GetGroup(groupAlias)
		if !exists {
			return fmt.Sprintf("❌ Grupo %s no encontrado", groupAlias)
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📋 GRUPO: %s\n", groupAlias))
		sb.WriteString(fmt.Sprintf("   Nombre: %s\n", group.Name))
		sb.WriteString(fmt.Sprintf("   Admin: %s\n", crypto.ResolveDID(group.Admin)))
		sb.WriteString(fmt.Sprintf("   Creado: %s\n", group.Created))
		sb.WriteString(fmt.Sprintf("   Miembros (%d):\n", len(group.Members)))
		for _, m := range group.Members {
			sb.WriteString(fmt.Sprintf("     - %s\n", crypto.ResolveDID(m)))
		}
		return strings.TrimSpace(sb.String())

	default:
		return "Uso: /group [create|list|add|remove|delete|leave|send|info]"
	}
}
