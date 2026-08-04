// Copyright (C) 2026 Fernando Martin Lopez. All Rights Reserved.
// SPDX-License-Identifier: AGPL-3.0-only WITH Commons-Clause-1.0
//
// This file is part of Web5-Mesh — sovereign network kernel prototype (Fase 1).
// Use of this source code is governed by the AGPLv3 + Commons Clause
// license that can be found in the LICENSE file at the root of this repo.
//
// Commercial use, SaaS deployment, or resale without a commercial license
// agreement is strictly prohibited. Contact the author for licensing.

package commands

import (
	"fmt"
	"strings"
	"web5-mesh/src/crypto"
)

func init() {
	Register("help", cmdHelp)
}

func cmdHelp(args []string, id *crypto.Identity) string {
	// Si el usuario pide un tema específico (ej: "help session")
	if len(args) > 0 {
		topic := strings.ToLower(args[0])
		switch topic {
		case "acl", "confianza":
			return getHelpACL()
		case "chat":
			return getHelpChat()
		case "alias":
			return getHelpAlias()
		case "unix", "archivos":
			return getHelpUnix()
		case "docs", "documentos":
			return getHelpDocs()
		default:
			return fmt.Sprintf("❌ Tema '%s' no encontrado. Usa 'help' para ver la lista general.", topic)
		}
	}

	// Help general (COMPLETO)
	var help strings.Builder
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("  🛡️ XION KERNEL v1.0.0 - AYUDA GENERAL\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	help.WriteString("  IDENTIDAD Y CONFIANZA:\n")
	help.WriteString("    whoami                     - Ver tu identidad y claves públicas\n")
	help.WriteString("    acl list                   - Ver nodos en tu lista de confianza\n")
	help.WriteString("    acl import <did> <ed> <x>  - Agregar un nodo a tu confianza\n")
	help.WriteString("    status                     - Ver pares configurados en tu red de confianza\n")
	help.WriteString("    ping <did>                 - Probar conexión y derivación de clave E2E\n\n")

	help.WriteString("  ALIAS LOCALES:\n")
	help.WriteString("    /alias add <nombre> <did>  - Crear alias para un DID\n")
	help.WriteString("    /alias list                - Ver todos los alias guardados\n")
	help.WriteString("    /alias remove <nombre>     - Eliminar un alias\n")
	help.WriteString("    💡 Tip: chat <alias> \"mensaje\" en vez de pegar DID\n\n")

	help.WriteString("  GRUPOS:\n")
	help.WriteString("    /group create <alias> <nombre>  - Crear grupo\n")
	help.WriteString("    /group list                     - Listar grupos\n")
	help.WriteString("    /group add <alias> <did>        - Agregar miembro\n")
	help.WriteString("    /group remove <alias> [did]     - Remover miembro o salir\n")
	help.WriteString("    /group send <alias> <mensaje>   - Enviar mensaje al grupo\n")
	help.WriteString("    /group delete <alias>           - Eliminar grupo\n")
	help.WriteString("    /group info <alias>             - Info del grupo\n\n")

	help.WriteString("  COMUNICACIÓN E2E:\n")
	help.WriteString("    chat <did|alias> <mensaje> - Enviar mensaje cifrado punto a punto\n\n")

	// ============================================================
	// CORREGIDO: /to con comillas cerradas correctamente
	// ============================================================
	help.WriteString("    /to <did|alias>       - Fijar conversación con un alias/DID\n")
	help.WriteString("    /to on                - Grabar conversación\n")
	help.WriteString("    /to off               - Dejar de grabar conversación\n")
	help.WriteString("    /to off               - Salir del modo fijado\n\n")

	help.WriteString("  DOCUMENTOS SOBERANOS:\n")
	help.WriteString("    import <archivo>           - Meter archivo del host a la bóveda\n")
	help.WriteString("    sign <archivo>             - Firmar criptográficamente (Ed25519)\n")
	help.WriteString("    verify <archivo>           - Verificar integridad y autenticidad\n")
	help.WriteString("    export <archivo> <destino> - Sacar archivo de la bóveda al host\n")
	help.WriteString("    💡 Tip: Escribe 'help docs' para flujo completo\n\n")

	help.WriteString("  ESPACIO DE TRABAJO SOBERANO (Unix-like):\n")
	help.WriteString("    pwd                        - Mostrar directorio de trabajo\n")
	help.WriteString("    ls                         - Listar archivos en tu espacio u2p\n")
	help.WriteString("    mkdir <nombre>             - Crear carpeta para proyectos\n")
	help.WriteString("    cat <archivo>              - Leer contenido de un archivo\n")
	help.WriteString("    rm <archivo>               - Borrar archivo\n")
	help.WriteString("    rmdir <carpeta>            - Borrar carpeta vacía\n")
	help.WriteString("    mv <origen> <destino>      - Mover/renombrar archivo\n")
	help.WriteString("    cp <origen> <destino>      - Copiar archivo\n")
	help.WriteString("    touch <archivo>            - Crear archivo vacío\n")
	help.WriteString("    edit <archivo>             - Editor de texto blindado (usa :wq para guardar)\n")
	help.WriteString("    💡 Tip: Escribe 'help unix' para ver todos los comandos\n\n")

	help.WriteString("  SISTEMA:\n")
	help.WriteString("    help                       - Este mensaje\n")
	help.WriteString("    help <tema>                - Ayuda detallada (ej: help alias, unix, docs)\n")
	help.WriteString("    clear                      - Borrar identidad y ACL (⚠️ IRREVERSIBLE)\n")
	help.WriteString("    exit                       - Salir de la consola asegurada\n\n")

	help.WriteString("🔒 NOTA DE SEGURIDAD:\n")
	help.WriteString("Cero exec.Command. El host solo ve un proceso \"xion/mesh\" y tráfico UDP cifrado.\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return help.String()
}

func getHelpACL() string {
	var help strings.Builder
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("  🔒 AYUDA: ACL (Lista de Confianza)\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("Comandos:\n")
	help.WriteString("  acl list                   - Ver nodos en tu lista de confianza\n")
	help.WriteString("  acl import <did> <ed> <x>  - Agregar un nodo a tu confianza\n")
	help.WriteString("\n📋 Ejemplo:\n")
	help.WriteString("  acl import did:maia:xxx abc123... def456...\n")
	help.WriteString("\n💡 Las claves públicas las obtiene el otro nodo con 'whoami'\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return help.String()
}

func getHelpChat() string {
	var help strings.Builder
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("  💬 AYUDA: CHAT (Mensajería E2E)\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("Comandos:\n")
	help.WriteString("  chat <did|alias> <mensaje> - Enviar mensaje cifrado\n")
	help.WriteString("\n📋 Ejemplos:\n")
	help.WriteString("  chat did:maia:xxx \"Hola, ¿cómo estás?\"\n")
	help.WriteString("  chat amigo \"Hola\"  (si tenés alias 'amigo')\n")
	help.WriteString("\n💡 Usá 'help alias' para crear alias y no pegar DIDs largos\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return help.String()
}

func getHelpAlias() string {
	var help strings.Builder
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("  🏷️ AYUDA: ALIAS LOCALES\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("Comandos:\n")
	help.WriteString("  /alias add <nombre> <did>  - Crear alias para un DID\n")
	help.WriteString("  /alias list                - Ver todos los alias guardados\n")
	help.WriteString("  /alias remove <nombre>     - Eliminar un alias\n")
	help.WriteString("\n📋 Ejemplo:\n")
	help.WriteString("  /alias add amigo did:maia:GVdM6Wix...\n")
	help.WriteString("  /alias list\n")
	help.WriteString("  chat amigo \"Hola\"  (en vez de pegar el DID completo)\n")
	help.WriteString("\n💡 Los alias se guardan en alias.json\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return help.String()
}

func getHelpUnix() string {
	var help strings.Builder
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("  📁 AYUDA: COMANDOS UNIX-LIKE\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("Todos operan dentro de la Jaula de Faraday (~/.xion/workspace/)\n")
	help.WriteString("\nComandos:\n")
	help.WriteString("  pwd                        - Mostrar directorio actual\n")
	help.WriteString("  ls                         - Listar archivos\n")
	help.WriteString("  mkdir <nombre>             - Crear carpeta\n")
	help.WriteString("  cat <archivo>              - Ver contenido\n")
	help.WriteString("  rm <archivo>               - Borrar archivo\n")
	help.WriteString("  rmdir <carpeta>            - Borrar carpeta vacía\n")
	help.WriteString("  mv <origen> <destino>      - Mover/renombrar\n")
	help.WriteString("  cp <origen> <destino>      - Copiar archivo\n")
	help.WriteString("  touch <archivo>            - Crear archivo vacío\n")
	help.WriteString("  edit <archivo>             - Editor integrado (:wq para guardar)\n")
	help.WriteString("\n💡 Todos los archivos tienen permisos 0600 (solo tú)\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return help.String()
}

func getHelpDocs() string {
	var help strings.Builder
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("  📄 AYUDA: DOCUMENTOS SOBERANOS\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	help.WriteString("Flujo completo: Import → Sign → Verify → Export\n")
	help.WriteString("\nComandos:\n")
	help.WriteString("  import <archivo>           - Meter archivo del host a la bóveda\n")
	help.WriteString("  sign <archivo>             - Firmar con Ed25519 + SHA256\n")
	help.WriteString("  verify <archivo>           - Verificar integridad y autenticidad\n")
	help.WriteString("  export <archivo> <destino> - Sacar archivo de la bóveda al host\n")
	help.WriteString("\n📋 Ejemplo:\n")
	help.WriteString("  import ~/contrato.pdf\n")
	help.WriteString("  sign contrato.pdf\n")
	help.WriteString("  verify contrato.pdf\n")
	help.WriteString("  export contrato.pdf ~/Desktop/\n")
	help.WriteString("\n💡 El archivo + firma (.sig) pueden enviarse por cualquier medio\n")
	help.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return help.String()
}
