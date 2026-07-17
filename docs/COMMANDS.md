# XionIA - Referencia Completa de Comandos

## 📌 LEYENDA DE ESTADO

| Icono | Significado |
|:-----:|:------------|
| ✅ | Comando 100% funcional |
| 🚧 | En desarrollo |

---

## 🆔 Identidad y Confianza

### `whoami` ✅

Muestra tu identidad actual, DID y claves públicas. Incluye el comando `acl import` listo para copy-paste.

```
xion@nodo:~$ whoami
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  🆔 TU IDENTIDAD
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  DID: did:maia:5XVNWhUtMNHLBVti6e93nSJZT25RtXUpeYYnywmHoC1i
  PubKey Ed25519: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
  PubKey X25519:  xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

  📋 Para que otro nodo te agregue, decile que ejecute:
  acl import did:maia:5XVNWhUtMNHLBVti6e93nSJZT25RtXUpeYYnywmHoC1i ed_pubkey_hex x_pubkey_hex
  alias add <nick> did:maia:5XVNWhUtMNHLBVti6e93nSJZT25RtXUpeYYnywmHoC1i
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### `acl list` ✅

Lista todos los nodos en tu lista de confianza (ACL).

```
xion@nodo:~$ acl list
📋 NODOS DE CONFIANZA (ACL):
  ✅ did:maia:ABC123...
  ✅ did:maia:DEF456...
```

### `acl import <did> <ed_pubkey> <x_pubkey>` ✅

Agrega un nodo a tu lista de confianza importando sus claves.

```
xion@nodo:~$ acl import did:maia:ABC123... ed_pubkey_hex x_pubkey_hex
✅ Nodo agregado a tu ACL
```

### `acl add <did>` ✅

Agrega un nodo a tu ACL (si ya lo conoces).

```
xion@nodo:~$ acl add did:maia:ABC123...
✅ Nodo agregado a tu ACL
```

### `acl remove <did>` ✅

Elimina un nodo de tu ACL.

```
xion@nodo:~$ acl remove did:maia:ABC123...
✅ Nodo eliminado de tu ACL
```

### `acl clear` ✅

Limpia toda tu ACL (no borra tu identidad).

```
xion@nodo:~$ acl clear
✅ ACL limpiada
```

---

## 🏷️ Alias Locales

### `alias add <nombre> <did>` ✅

Crea un alias para un DID, para no tener que escribirlo completo.

```
xion@nodo:~$ alias add amigo did:maia:ABC123...
✅ Alias 'amigo' creado para did:maia:ABC123...
```

### `alias list` ✅

Lista todos los alias guardados.

```
xion@nodo:~$ alias list
📋 ALIAS GUARDADOS:
  amigo → did:maia:ABC123...
  casa → did:maia:DEF456...
```

### `alias remove <nombre>` ✅

Elimina un alias.

```
xion@nodo:~$ alias remove amigo
✅ Alias 'amigo' eliminado
```

---

## 👥 Grupos

### `group create <alias> <nombre>` ✅

Crea un nuevo grupo. El alias es el identificador interno.

```
xion@nodo:~$ group create devs "Equipo de Desarrollo"
✅ Grupo creado: devs ("Equipo de Desarrollo")
```

### `group list` ✅

Lista todos los grupos.

```
xion@nodo:~$ group list
📋 GRUPOS:
  [devs] "Equipo de Desarrollo" (3 miembros) - admin: did:maia:ABC...
  [pruebas] "Grupo de Prueba" (2 miembros) - admin: yo
```

### `group add <alias> <did|alias>` ✅

Agrega un miembro al grupo (solo el admin puede hacerlo).

```
xion@nodo:~$ group add devs did:maia:XYZ789...
✅ did:maia:XYZ789... agregado al grupo devs
📩 Grupo sincronizado con el nuevo miembro
```

### `group remove <alias> [did|alias]` ✅

**Sin argumentos:** sales del grupo.  
**Con argumento:** el admin remueve a otro miembro.

```
xion@nodo:~$ group remove devs
✅ Saliste del grupo 'devs'

xion@nodo:~$ group remove devs did:maia:XYZ789...
✅ Miembro removido del grupo 'devs'
```

### `group send <alias> <mensaje>` ✅

Envía un mensaje cifrado a todos los miembros del grupo.

```
xion@nodo:~$ group send devs "Reunión mañana a las 10"
✅ Mensaje enviado al grupo devs (3 miembros)
```

### `group delete <alias>` ✅

Elimina el grupo (solo admin).

```
xion@nodo:~$ group delete devs
✅ Grupo 'devs' eliminado
```

### `group info <alias>` ✅

Muestra información detallada del grupo.

```
xion@nodo:~$ group info devs
📋 GRUPO: devs
   Nombre: "Equipo de Desarrollo"
   Admin: yo
   Creado: 2026-07-10
   Miembros (3):
     - yo
     - amigo
     - otro
```

### `group leave <alias>` ✅

Sales del grupo (cualquier miembro puede hacerlo).

```
xion@nodo:~$ group leave devs
✅ Saliste del grupo 'devs'
```

---

## 💬 Comunicación E2E

### `chat <did|alias> <mensaje>` ✅

Envía un mensaje cifrado punto a punto a otro nodo.

```
xion@nodo:~$ chat amigo "Hola, ¿cómo estás?"
🔗 Alias resuelto: amigo → did:maia:ABC123...
✅ Conectado al faro por UDP: 190.220.45.26:54321
✅ Mensaje entregado
💬 [amigo]: CHAT:Hola, todo bien
```

---

## 🛡️ Jaula de Faraday (Bóveda Soberana)

### `import <ruta_host>` ✅

Mete un archivo del host a la bóveda soberana.

```
xion@nodo:~$ import ~/documento.pdf
✅ Archivo ingresado a la bóveda:
   ├── Origen: /home/usuario/documento.pdf
   ├── Destino: /home/usuario/.xion/workspace/documento.pdf
   └── Permisos: 0600 (solo tú)
```

### `export <archivo> [ruta_destino]` ✅

Saca un archivo de la bóveda al host.

```
xion@nodo:~$ export documento.pdf ~/Desktop/
✅ Archivo exportado de la bóveda:
   ├── Origen (Bóveda): ~/.xion/workspace/documento.pdf
   ├── Destino (Host): ~/Desktop/documento.pdf
   └── Permisos: 0644
```

### `sign <archivo>` ✅

Firma criptográficamente un archivo (SHA256 + Ed25519).

```
xion@nodo:~$ sign documento.pdf
✅ Archivo firmado criptográficamente:
   ├── Archivo: documento.pdf
   ├── Hash SHA256: 2ee498c8fa0f778e...
   ├── Firma: documento.pdf.sig
   ├── Firmante: did:maia:5XVNWhUtMNH...
   └── Timestamp: 2026-07-10 18:30:00
```

### `verify <archivo>` ✅

Verifica integridad y autenticidad de un archivo firmado.

```
xion@nodo:~$ verify documento.pdf
✅ VERIFICACIÓN EXITOSA:
   ├── Integridad: ✅ Hash válido
   ├── Autenticidad: ✅ Firma válida
   ├── Firmante: Tú mismo
   └── El archivo es auténtico y no fue modificado.
```

---

## 📂 Comandos Unix (en la bóveda)

Todos estos comandos operan **dentro de la Jaula de Faraday** (`~/.xion/workspace/`).

### `pwd` ✅

Muestra el directorio actual de la bóveda.

```
xion@nodo:~$ pwd
📂 /home/usuario/.xion/workspace
```

### `ls` ✅

Lista el contenido de la bóveda.

```
xion@nodo:~$ ls
📂 Contenido de tu espacio de trabajo:
  📁 proyectos/
  📄 documento.pdf (2345 bytes)
```

### `mkdir <nombre>` ✅

Crea una carpeta en la bóveda.

```
xion@nodo:~$ mkdir proyectos
✅ Directorio 'proyectos' creado en tu espacio soberano.
```

### `cat <archivo>` ✅

Muestra el contenido de un archivo.

```
xion@nodo:~$ cat notas.txt
📄 --- notas.txt ---
Hoy es un buen día para la soberanía digital.
--- FIN ---
```

### `rm <archivo>` ✅

Borra un archivo de la bóveda.

```
xion@nodo:~$ rm viejo.txt
✅ Archivo 'viejo.txt' borrado.
```

### `rmdir <carpeta>` ✅

Borra una carpeta vacía.

```
xion@nodo:~$ rmdir tmp/
✅ Directorio 'tmp' borrado.
```

### `mv <origen> <destino>` ✅

Mueve o renombra un archivo dentro de la bóveda.

```
xion@nodo:~$ mv doc.pdf backup.pdf
✅ 'doc.pdf' movido a 'backup.pdf'.
```

### `cp <origen> <destino>` ✅

Copia un archivo dentro de la bóveda.

```
xion@nodo:~$ cp doc.pdf copia.pdf
✅ 'doc.pdf' copiado a 'copia.pdf'.
```

### `touch <archivo>` ✅

Crea un archivo vacío o actualiza su timestamp.

```
xion@nodo:~$ touch nuevo.txt
✅ Archivo 'nuevo.txt' creado/actualizado.
```

### `edit <archivo>` / `/edit <archivo>` ✅

Editor de texto integrado (usa `:wq` para guardar y salir).

```
xion@nodo:~$ edit config.txt
(abre el editor)
:wq  (para guardar y salir)
✅ Archivo guardado.
```

**Nota:** Todos los comandos Unix aceptan tanto la versión con `/` (ej: `/ls`) como sin él (ej: `ls`).

---

## 🔧 Sistema

### `help` / `help <tema>` ✅

Muestra la ayuda general o detallada de un tema específico.

```
xion@nodo:~$ help
```

### `exit` ✅

Sale de la shell de forma segura.

```
xion@nodo:~$ exit
👋 Sesión cerrada.
mamanga@node-379-core:~/Web5-Mesh$
```

**Comportamiento:**
- ✅ Cierra la conexión con el faro
- ✅ Cierra los canales de comunicación
- ✅ Restaura la terminal a su estado original
- ✅ Vuelve al shell del sistema

---

## 📡 Faro

### `faro set <addr>` ✅

Configura la dirección de un faro diferente al default. Útil para conectar a faros alternativos o privados.

```
xion@nodo:~$ faro set 192.168.1.100:54321
✅ Faro configurado: 192.168.1.100:54321

xion@nodo:~$ faro set 150.136.55.87:443
✅ Faro configurado: 150.136.55.87:443 (WebSocket)
```

**Nota:** El faro default es `150.136.55.87:443` (WebSocket) o el definido en `config.json`.

### `faro reset` ✅

Restaura la configuración del faro al default.

```
xion@nodo:~$ faro reset
✅ Faro reseteado a default: 150.136.55.87:443
```
---

## ⚠️ Comandos Destructivos

### `clear` ⚠️

**⚠️ PELIGRO:** Borra tu identidad y ACL completamente. **Irreversible.**

```
xion@nodo:~$ clear
⚠️  Este comando borra tu identidad y ACL.
   Usá 'clear --force' para confirmar.
```

**Para forzar el borrado sin confirmación:**

```
xion@nodo:~$ clear --force
✅ Jaula de Faraday reiniciada:
   ├── node.key eliminado (identidad borrada)
   ├── acl.json eliminado (pares de confianza borrados)
   └── Reiniciá la shell para generar nueva identidad.
```

**Uso recomendado:** Siempre usar `clear` sin `--force` para evitar borrados accidentales. El `--force` solo para scripts o cuando estés 100% seguro.

---

*Referencia completa de comandos de XionIA - La Xión Digital 🦾*
