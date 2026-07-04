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

## 💬 Comunicación E2E

### `chat <did> <mensaje>`

Envía un mensaje cifrado punto a punto a otro nodo.

```
xion@nodo:~$ chat did:maia:ABC123... "Hola, ¿cómo estás?"
✅ Mensaje enviado y cifrado E2E
```

---

## 📻 Multitarea y Sesiones

### `session list`

Lista todas las sesiones activas.

```
xion@nodo:~$ session list
📻 Sesiones activas:
  [1] chat con did:maia:ABC123...
  [2] ssh a did:maia:DEF456...
```

### `session new <tipo> <target>`

Crea una nueva sesión (chat, ssh, etc.).

```
xion@nodo:~$ session new chat did:maia:ABC123...
[CHAT] Modo conversación activado.
Escribí tu mensaje o 'session detach' para salir.
```

### `session attach <id>`

Se conecta a una sesión existente.

```
xion@nodo:~$ session attach 1
[CHAT] Conectado a sesión con did:maia:ABC123...
```

### `session detach`

Pone la sesión actual en background y vuelve al inicio.

```
[CHAT] > session detach
📻 Sesión puesta en background.
xion@nodo:~$
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

```
xion@nodo:~$ help session
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  📻 AYUDA: SESIONES (Multitarea)
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


---

*Referencia completa de comandos de XionIA - La Xión Digital 🦾*
