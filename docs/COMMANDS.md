# XionIA - Referencia Completa de Comandos

## Identidad y Confianza

### `whoami`

Muestra tu identidad actual, DID y claves públicas.

```
xion@nodo:~$ whoami
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  🆔 TU IDENTIDAD
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  DID: did:maia:5XVNWhUtMNHLBVti6e93nSJZT25RtXUpeYYnywmHoC1i
  PubKey Ed25519: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
  PubKey X25519:  xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### `acl list`

Lista todos los nodos en tu lista de confianza (ACL).

```
xion@nodo:~$ acl list
📋 Nodos en tu lista de confianza:
  1. did:maia:ABC123... (alias: nodo-amigo)
  2. did:maia:DEF456... (alias: servidor-casa)
```

### `acl import <did> <ed_pubkey> <x_pubkey>`

Agrega un nodo a tu lista de confianza.

```
xion@nodo:~$ acl import did:maia:ABC123... ed_pubkey_hex x_pubkey_hex
✅ Nodo agregado a tu ACL
```

### `status`

Muestra el estado de la red y los pares configurados.

```
xion@nodo:~$ status
🛡️ [NODO] ACL indexada con 3 pares. Escuchando y listo.
```

### `status clear`

Limpia la lista de pares de confianza (no borra tu identidad).

```
xion@nodo:~$ status clear
✅ ACL limpiada. Reiniciá la shell para aplicar.
```

### `ping <did>`

Prueba la conexión con otro nodo y deriva la clave E2E.

```
xion@nodo:~$ ping did:maia:ABC123...
📡 Ping a did:maia:ABC123...
   ├── RTT: 45ms
   ├── Clave E2E derivada: ✅
   └── Estado: Conectado
```

---

## 🏷️ Alias Locales

### `alias add <nombre> <did>`

Crea un alias para un DID, para no tener que escribirlo completo.

```
xion@nodo:~$ alias add amigo did:maia:ABC123...
✅ Alias 'amigo' creado para did:maia:ABC123...
```

### `alias list`

Lista todos los alias guardados.

```
xion@nodo:~$ alias list
📋 Alias guardados:
  amigo → did:maia:ABC123...
  casa → did:maia:DEF456...
```

### `alias remove <nombre>`

Elimina un alias.

```
xion@nodo:~$ alias remove amigo
✅ Alias 'amigo' eliminado
```

---

## 👥 Grupos

### `group create <alias> <nombre>`

Crea un nuevo grupo. El alias es el identificador interno.

```
xion@nodo:~$ group create devs "Equipo de Desarrollo"
✅ Grupo 'devs' creado: Equipo de Desarrollo
```

### `group list`

Lista todos los grupos.

```
xion@nodo:~$ group list
📋 Grupos:
  [devs] Equipo de Desarrollo (3 miembros) - admin: did:maia:ABC...
  [amigos] Grupo de Amigos (5 miembros) - admin: did:maia:DEF...
```

### `group add <alias> <did|alias>`

Agrega un miembro al grupo (solo el admin puede hacerlo).

```
xion@nodo:~$ group add devs did:maia:XYZ789...
✅ did:maia:XYZ789... agregado al grupo 'devs'
📩 Sincronizado con todos los miembros
```

### `group remove <alias> [did|alias]`

Sin segundo argumento: salís vos del grupo.
Con segundo argumento: el admin remueve a otro miembro.

```
xion@nodo:~$ group remove devs
✅ Saliste del grupo 'devs'

xion@nodo:~$ group remove devs did:maia:XYZ789...
✅ did:maia:XYZ789... removido del grupo 'devs'
```

### `group send <alias> <mensaje>`

Envía un mensaje cifrado a todos los miembros del grupo.

```
xion@nodo:~$ group send devs "Reunión mañana a las 10"
✅ Mensaje enviado a 3 miembros del grupo 'devs'
```

### `group delete <alias>`

Elimina el grupo (solo el admin puede hacerlo).

```
xion@nodo:~$ group delete devs
✅ Grupo 'devs' eliminado
📩 Notificados todos los miembros
```

### `group info <alias>`

Muestra información detallada del grupo.

```
xion@nodo:~$ group info devs
📋 GRUPO: devs
   Nombre: Equipo de Desarrollo
   Admin: did:maia:ABC123...
   Creado: 2026-07-01
   Miembros (3):
     - did:maia:ABC123...
     - did:maia:DEF456...
     - did:maia:XYZ789...
```

---

## 💬 Comunicación E2E

### `chat <did|alias> <mensaje>`

Envía un mensaje cifrado punto a punto a otro nodo.

```
xion@nodo:~$ chat did:maia:ABC123... "Hola, ¿cómo estás?"
✅ Mensaje enviado y cifrado E2E

xion@nodo:~$ chat amigo "Hola"
✅ Mensaje enviado y cifrado E2E
```

---

## 🛡️ Jaula de Faraday (Bóveda Soberana)

### `import <ruta_host>`

Mete un archivo del host a la bóveda soberana.

```
xion@nodo:~$ import ~/documento.pdf
✅ Archivo ingresado a la bóveda:
   ├── Origen: /home/usuario/documento.pdf
   ├── Destino: /home/usuario/.u2p/workspace/documento.pdf
   └── Permisos: 0600 (solo tú)
```

### `sign <archivo>`

Firma criptográficamente un archivo (SHA256 + Ed25519).

```
xion@nodo:~$ sign documento.pdf
✅ Archivo firmado criptográficamente:
   ├── Archivo: documento.pdf (23 bytes)
   ├── Hash SHA256: 2ee498c8fa0f778e...
   ├── Firma: documento.pdf.sig
   ├── Firmante: did:maia:5XVNWhUtMNH...
   └── Timestamp: 2026-06-17 11:21:15
```

### `verify <archivo>`

Verifica integridad y autenticidad de un archivo firmado.

```
xion@nodo:~$ verify documento.pdf
✅ VERIFICACIÓN EXITOSA:
   ├── Integridad: ✅ Hash válido
   ├── Autenticidad: ✅ Firma válida
   ├── Firmante: Tú mismo
   └── El archivo es auténtico y no fue modificado.
```

### `export <archivo> [ruta_destino]`

Saca un archivo de la bóveda al host.

```
xion@nodo:~$ export documento.pdf ~/Desktop/
✅ Archivo exportado de la bóveda:
   ├── Origen (Bóveda): ~/.u2p/workspace/documento.pdf
   ├── Destino (Host): ~/Desktop/documento.pdf
   └── Permisos: 0644
```

---

## 📂 Comandos Unix (en la bóveda)

Todos estos comandos operan **dentro de la Jaula de Faraday** (`~/.u2p/workspace/`).

| Comando | Descripción | Ejemplo |
|:--------|:------------|:--------|
| `/ls` | Lista el contenido de la bóveda | `/ls` |
| `/cat <archivo>` | Muestra el contenido de un archivo | `/cat notas.txt` |
| `/rm <archivo>` | Borra un archivo de la bóveda | `/rm viejo.pdf` |
| `/rmdir <carpeta>` | Borra una carpeta vacía | `/rmdir tmp/` |
| `/mv <origen> <destino>` | Mueve o renombra un archivo | `/mv doc.pdf backup.pdf` |
| `/cp <origen> <destino>` | Copia un archivo dentro de la bóveda | `/cp doc.pdf copia.pdf` |
| `/touch <archivo>` | Crea un archivo vacío | `/touch nuevo.txt` |
| `/mkdir <carpeta>` | Crea una carpeta en la bóveda | `/mkdir proyectos/` |
| `/edit <archivo>` | Editor de texto integrado. Usá `:wq` para guardar. | `/edit config.txt` |
| `/pwd` | Muestra el directorio actual de la bóveda | `/pwd` |

---

## 🔧 Sistema

### `help` / `help <tema>`

Muestra la ayuda general o detallada de un tema específico.

Temas disponibles: `acl`, `alias`, `group`, `chat`, `unix`, `docs`

```
xion@nodo:~$ help alias
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  🏷️ AYUDA: ALIAS LOCALES
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  [guía paso a paso]
```

### `clear`

**⚠️ PELIGRO:** Borra tu identidad y ACL completamente. Irreversible.

```
xion@nodo:~$ clear
⚠️ ¿Estás seguro? Esta acción borrará tu identidad y ACL. (s/n)
```

### `exit`

Sale de la shell de forma segura.

```
xion@nodo:~$ exit
👋 Sesión cerrada.
```

*Referencia completa de comandos de XionIA - La Xión Digital 🦾*
