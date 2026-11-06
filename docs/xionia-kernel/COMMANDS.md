# Derinkuyu — Referencia Completa de Comandos

**La consola soberana de XionIA.** El Shell es el Rey: toda función del
kernel vive acá primero. Y desde v9.2, la Jaula es el taller completo —
escribir, firmar, exportar, enviar, todo dentro del workspace soberano.

---

## 📌 LEYENDA DE ESTADO

| Icono | Significado |
|:-----:|:------------|
| ✅ | Comando funcional (v9.2, probado) |

```
Arranque:   ./derinkuyu -faro 190.220.45.26:3478
Prompt:     xion@DerinKuyu:~$          (modo normal)
            chat[tvbox]:~$             (chat continuo activo)

Tab: autocompletar · ↑↓: historial (persiste en ~/.xion/history.txt)
En chat activo, los comandos de la Jaula van con / (/ls, /edit…)
```

---

## 🆔 Identidad y Confianza

### `share` ✅

Tu paquete de identidad al clipboard (OSC52) + backup en `~/.xion/share.txt`.
Es lo que le pasás a otro nodo para que te agregue.

```
xion@DerinKuyu:~$ share
── SHARE ──
⚡ Copiado al clipboard (OSC52).
  acl import did:maia:7LWN... ed_pubkey_hex x_pubkey_hex
  Backup: /home/mamanga/.xion/share.txt
```

### `whoami` ✅

Tu identidad completa y el estado del faro.

```
xion@DerinKuyu:~$ whoami
MI IDENTIDAD
  DID:    did:maia:7LWNmJESzEXuwYXKxqUi62mwtt5gw18GM9sjerLoAxn6
  PubEd:  a3b2c1...
  PubX:   9f8e7d...
  Faro:   190.220.45.26:3478  conectado
  Peers:  4 en ACL
```

### `acl import <did> <ed_hex> <x_hex>` ✅

Agrega un nodo a tu lista de confianza (el paquete que genera `share` del
otro nodo). La ACL vive en vivo: persiste al instante.

```
xion@DerinKuyu:~$ acl import did:maia:FKKAG... ed_hex x_hex
✅ Peer importado: did:maia:FKKAG…
   Tip: alias add <nombre> did:maia:FKKAG…
```

### `acl list` / `peers` ✅

```
xion@DerinKuyu:~$ peers
PEERS (2)
  1. [tvbox]
     did:maia:FKKAGLuHFi3QkZacfZQonEWHiiQVgnehfUdUyHdsTTY5
  2. [poco]
     did:maia:5XVNWhUtMNHLBVti6e93nSJZT25RtXUpeYYnywmHoC1i
```

### `acl remove <alias|DID>` ✅

Elimina un nodo (y limpia el chat activo si era con él).

---

## 🏷️ Alias

Alias → DID, persistido en `~/.xion/aliases.json` — mismo formato que la
APK: los dos mundos comparten el grafo social.

### `alias add <nombre> <DID>` ✅

```
xion@DerinKuyu:~$ alias add tvbox did:maia:FKKAG…
✅ Alias 'tvbox' → did:maia:FKKAG…
```

### `alias list` ✅ · `alias remove <nombre>` ✅

```
xion@DerinKuyu:~$ alias list
ALIASES (2)
  poco              did:maia:5XVN…
  tvbox             did:maia:FKKA…
```

---

## 💬 Comunicación E2E

### `to <alias|DID>` ✅ — el modo chat

Activa el chat continuo: todo lo que escribis va al peer, sin comandos.
`to off` sale. Los comandos siguen disponibles adentro (`xfer`, `ping`,
`status`… y la Jaula con `/`).

```
xion@DerinKuyu:~$ to tvbox
💬 Chat con 'tvbox'. Escribí y Enter. ('to off' para salir)
chat[tvbox]:~$ hola
[00:55:23] 📤 Vos → tvbox via direct: hola
[00:55:29] 💬 tvbox: como va
```

El camino de cada mensaje: **direct si hay sesión Noise viva, relay del
faro si no** — con renegociación ICE en background. El mensaje jamás
espera al túnel.

### `send <alias|DID> <mensaje>` ✅

Mensaje único sin entrar al modo chat.

```
xion@DerinKuyu:~$ send tvbox prueba del túnel
[01:12:03] 📤 → tvbox via direct: prueba del túnel
```

### `xfer <alias|DID> <archivo>` ✅ — archivos por la Jaula

Envía un archivo **solo por el túnel directo**. Jamás relay, jamás el
faro. Chunks con ACK por chunk: llega completo o no llega.

```
chat[tvbox]:~$ xfer tvbox /tmp/prueba.txt
📤 XFER /tmp/prueba.txt → tvbox ...
📤 XFER prueba.txt → did:maia:FKKAG…: 1 chunks (26 B)
📤 XFER prueba.txt: 1/1
✅ XFER prueba.txt completo → did:maia:FKKAG… (1ms)
```

En el receptor (llega solo, con el anuncio y la confirmación):

```
📥 XFER entrante de did:maia:7LWN…: prueba.txt (1 chunks)
📎 xeon te envió prueba.txt (26 B) → /home/alarm/.xion/incoming/prueba.txt
```

Los archivos recibidos caen en `~/.xion/incoming/` — pasalos a tu Jaula
con `import ~/.xion/incoming/<archivo>`.

### `ping <alias|DID>` ✅

Latencia E2E del camino actual.

```
xion@DerinKuyu:~$ ping tvbox
🏓 pong de tvbox  rtt=14ms
```

---

## 🛡️ La Jaula — el taller soberano

El workspace del nodo: `~/.xion/workspace/` (0700). Todo lo que sigue
opera **exclusivamente adentro** — el escape es imposible por diseño:

```
xion@DerinKuyu:~$ rm ../notas.txt
❌ fuera de la Jaula
```

Rutas relativas solamente, todo `..` que salga se rechaza. El paso
host↔Jaula es SIEMPRE explícito: `import` / `export`. Nada entra ni
sale sin comando.

**El flujo del trabajo soberano:**

```
import → edit → pdf → sign → xfer → verify
 host    Jaula   PDF   firma  túnel   receptor
```

### Comandos de archivo ✅

| Comando | Qué hace |
|---|---|
| `pwd` | Dónde estás en la Jaula |
| `ls [dir]` | Contenido con tamaños |
| `cd <dir>` · `cd /` | Navegar (vuelve a raíz con `/`) |
| `mkdir <carpeta>` | Crear (0700) |
| `rm <archivo>` · `rmdir <carpeta vacía>` | Eliminar (cada uno lo suyo) |
| `mv` · `cp <origen> <destino>` | Mover / copiar |
| `touch <archivo>` | Crear vacío |
| `cat <archivo>` | Ver contenido |
| `head` · `tail [-n N] <archivo>` | Primeras / últimas N líneas |
| `grep "patrón" <archivo>` | Buscar (con número de línea) |

```
xion@DerinKuyu:~$ ls
📂 /home/mamanga/.xion/workspace
  📄 hostname (14 B)
  📄 notas.pdf (1.8 KB)
  📄 notas.txt (37 B)
  📄 notas.txt.sig (401 B)
```

```
xion@DerinKuyu:~$ cat notas.txt
📄 --- notas.txt ---
La soberanía no se pide.
Se compila.
--- FIN ---
```

### `edit <archivo>` ✅ — el editor

Multilinea, sin dependencias. Si el archivo existe, muestra el contenido
actual (la edición lo REEMPLAZA). Una línea con solo `.` termina y guarda.
Ctrl+C o Ctrl+D descartan.

```
xion@DerinKuyu:~$ edit notas.txt
✍️  Escribí. Una línea con solo . termina y guarda · Ctrl+C descarta
│ La soberanía no se pide.
│ Se compila.
│ .
✅ Guardado: notas.txt (2 líneas)
```

### `import <ruta_host>` ✅ — host → Jaula

El único camino de entrada. Acepta `~`. Permisos 0600.

```
xion@DerinKuyu:~$ import /etc/hostname
✅ Archivo ingresado a la Jaula:
   ├── Origen: /etc/hostname
   ├── Destino: /home/mamanga/.xion/workspace/hostname
   └── 14 B · permisos 0600
```

### `export <archivo> [ruta_destino] ✅ — Jaula → host

El único camino de salida. Sin destino, cae al directorio actual del
host. Si el destino es una carpeta (`/tmp/`), el archivo va adentro.

```
xion@DerinKuyu:~$ export carta.txt /tmp/
✅ Archivo exportado de la Jaula:
   ├── Origen (Jaula): /home/mamanga/.xion/workspace/docs/carta.txt
   └── Destino (host): /tmp/carta.txt · 37 B
```

### `sign <archivo>` ✅ — firma criptográfica

Firma con la clave Ed25519 de TU nodo (la de tu DID) sobre el SHA256 del
archivo. Genera `<archivo>.sig` al lado: JSON con DID, clave pública,
hash, firma y timestamp.

```
xion@DerinKuyu:~$ sign notas.txt
✅ Archivo firmado criptográficamente:
   ├── Archivo: notas.txt
   ├── Hash SHA256: cbd2f679077be702…
   ├── Firma: notas.txt.sig
   └── Firmante: did:maia:7LWNmJESzEXuwYXKxqUi62mwtt5gw18GM9sjerLoAxn6
```

### `verify <archivo>` ✅ — verificación

Dos controles: **integridad** (el hash coincide — el archivo no cambió
desde la firma) y **autenticidad** (la firma Ed25519 es válida). La clave
pública del firmante: si sos vos, tu propia clave; si es otro, **la que
ya está en tu ACL** — la misma que usó el túnel. La verificación no
necesita infraestructura: la identidad ya está distribuida.

```
xion@DerinKuyu:~$ verify notas.txt
✅ VERIFICACIÓN EXITOSA:
   ├── Integridad: hash válido
   ├── Autenticidad: firma Ed25519 válida
   └── Firmante: Tú mismo
```

Firmado por OTRO nodo (recibido por xfer + su .sig):

```
xion@DerinKuyu:~$ verify contrato.pdf
✅ VERIFICACIÓN EXITOSA:
   ├── Integridad: hash válido
   ├── Autenticidad: firma Ed25519 válida
   └── Firmante: xeon
```

### `pdf <archivo> [título]` ✅ — exportar a PDF

Arma el PDF del texto, con **pie de procedencia en cada página**: tu DID
truncado + fecha + numeración. Procedencia visible a ojo + `sign` para
la procedencia criptográfica: el documento que dice quién lo escribió
Y lo demuestra.

```
xion@DerinKuyu:~$ pdf notas.txt "Manifiesto de pruebas"
✅ PDF generado: notas.pdf
   Pie de página con tu DID. Ahora: sign notas.txt → prueba criptográfica.
```

> Fuentes core (Helvetica): acentos á é ñ ¿ funcionan; comillas curvas y
> rayas se convierten a ASCII. (DejaVu embebida con UTF-8 completo:
> pendiente de decisión.)

---

## 🔧 Sistema

### `status` ✅

```
xion@DerinKuyu:~$ status
ESTADO
  Faro:      190.220.45.26:3478  [SÍ]
  Peers ACL: 4
  Aliases:   4
  Sesiones:  2
  TX: 1523  RX: 1498
  Uptime:    8439s
```

### `faro [addr]` ✅

Sin argumentos: el faro activo y su estado. Con dirección: configura
(aplica al reiniciar con `-faro`).

```
xion@DerinKuyu:~$ faro
  Faro activo: 190.220.45.26:3478  [conectado]

xion@DerinKuyu:~$ faro 192.168.1.100:3478
✅ Faro configurado a 192.168.1.100:3478
   Reiniciá con: ./derinkuyu -faro 192.168.1.100:3478
```

### `help` ✅ · `version` ✅ · `clear` / `cls` ✅

`help` lista todo (incluida la Jaula). `clear` LIMPIA LA PANTALLA —
no borra nada más (los destructivos son los `reset`).

### `exit` / `quit` / `q` ✅

Cierra limpio: kernel, túneles, faro.

---

## ⚠️ Comandos Destructivos

### `reset` ✅ — con escalera de advertencia

```
xion@DerinKuyu:~$ reset
RESET:
  reset acl      Borrar todos los peers
  reset aliases  Borrar todos los alias
  reset did      Borrar identidad ⚠️ IRREVERSIBLE
  reset total    Borrar TODO ⚠️ IRREVERSIBLE
```

`reset did` te deja sin identidad: al reiniciar, DID nuevo — todo tu
grado social queda huérfano. `reset total` además borra aliases.
⚠️ **Hoy ni `reset total` toca la Jaula** — workspace e incoming
sobreviven a los resets (residuo registrado; decisión del dueño si
`reset total` debe arrasarla también).

---

*Referencia completa de comandos de Derinkuyu — XionIA Kernel v9.2 ·
Corrientes, Argentina* 🧉
