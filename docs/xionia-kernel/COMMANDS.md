# Derinkuyu — Referencia Completa de Comandos

**La consola soberana de XionIA.** El Shell es el Rey: toda función del
kernel vive acá primero. Y desde v9.2, la Jaula es el taller completo —
escribir, firmar, exportar, enviar, todo dentro del workspace soberano.

Y desde `release-v15`, tres verdades que esta referencia ya no calla: el
kernel viaja con **criptografía poscuántica a bordo — y en la consola
ya está viva** (§ 🔐) —, el faro de producción, kati, se opera **en
remoto por SSH** (§ 📡), y la consola abre **la capa host del
ecosistema: SSH root por DID**, con clave efímera por sesión y registro
firmado (§ 🕳️). Probado en campo: tvbox ↔ xeon, ambos detrás de WARP.

---

## 📌 LEYENDA DE ESTADO

| Icono | Significado |
|:-----:|:------------|
| ✅ | Comando funcional (v9.2 + 0016/0017, probado) |
| 🆕 | Agregado en esta revisión (remote nivel 1.2 · PQ viva) |

```
Arranque:   ./derinkuyu -faro 190.220.45.26:443
Prompt:     xion@DerinKuyu:~$          (modo normal)
            chat[tvbox]:~$             (chat continuo activo)

Puertos:    190.220.45.26:443 — la fachada (UDP: señalización + STUN ·
            TCP: WSS): la dirección a usar, viva incluso con saldo cero.
            Nativos detrás: 55123/UDP · 8444/TCP.
            (zappa hará del 443 el default de fábrica.)

Tab: autocompletar · ↑↓: historial (persiste en ~/.xion/history.txt)
Ctrl+T: alterna ssh ↔ chat con un túnel remoto vivo (multi.go, § ⌨️)
En chat activo, los comandos de la Jaula van con / (/ls, /edit…)
```

El banner de arranque ya cuenta la película completa — identidad
poscuántica incluida (datos reales del tvbox, sesión de campo
2026-11-22):

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  DERINKUYU dev  |  Kernel XionIA  |  Jaula: ✓ ACTIVA
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  🆔 IDENTIDAD
  DID:    did:maia:GR7PciB87Y7eJnTjyxf5sJXTURuh8C96aUwbEdWnsnP5
  PubEd:  e50c77eab467405c…
  PubX:   86ad8fd0ae70de49…
  Huella PQ (X25519+ML-KEM-768): f84b-a277-c946-b632

  📡 RED
  Faro:   190.220.45.26:443  ● CONECTADO
  Peers:  2 en ACL  |  Aliases: 2

  📋 PARA AGREGAR ESTE NODO:

  acl import did:maia:GR7P… e50c77ea… 86ad8fd0… cd924a940a86c503…(ML-KEM-768)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

## 🆙 Novedades de esta revisión

| Qué | Detalle |
|---|---|
| 🕳️ `remote` — SSH por DID | Nivel 1.2: las tres puertas, catálogo sellado, clave efímera por sesión, registro firmado, multitarea Ctrl+T. Probado en campo: túnel de 52 s tvbox↔xeon detrás de WARP. |
| 🔐 PQ a la vista | `share` y `whoami` muestran PubKem + Huella PQ; `acl import` acepta la 4ª clave (opcional). |
| ⌨️ Multitarea | Con el ssh del túnel vivo, Ctrl+T alterna ssh ↔ chat sin cortar nada (multi.go). |
| 📡 Fachada 443 | El arranque clásico era `…:3478`; hoy el semilla es `:443` (anti-DPI — ver `FAROS.md`). |

---

## 🆔 Identidad y Confianza

### `share` ✅

Tu paquete de identidad al clipboard (OSC52) + backup en `~/.xion/share.txt`.
Es lo que le pasás a otro nodo para que te agregue. Con node.key v2 el
paquete trae las **cuatro claves** — la 4ª es la pública ML-KEM-768
(1184 B en hex: 2368 caracteres, por eso el paste es largo):

```
xion@DerinKuyu:~$ share
── SHARE ──
⚡ Copiado al clipboard (OSC52).
  acl import did:maia:GR7P… e50c77ea… 86ad8fd0… cd924a940a86c503…9b88e4f73…
  Backup: /home/alarm/.xion/share.txt
```

Si la terminal no confirma OSC52, la consola te lo dice en la cara
(parche de auditoría: antes decía "copiado" aunque no hubiera copiado
nada) — copiás a mano y santas pascuas.

### `whoami` ✅

Tu identidad completa y el estado del faro — ahora con la capa PQ a la
vista:

```
xion@DerinKuyu:~$ whoami
MI IDENTIDAD
  DID:    did:maia:GR7PciB87Y7eJnTjyxf5sJXTURuh8C96aUwbEdWnsnP5
  PubEd:  e50c77ea…
  PubX:   86ad8fd0…
  PubKem: cd924a940a86c503…25373ab9909e
  Huella PQ: f84b-a277-c946-b632
  Faro:   190.220.45.26:443  conectado
  Peers:  2 en ACL
```

Si en lugar de `PubKem` ves "(sin semilla ML-KEM — reiniciar el nodo
para migrar node.key a v2)": reiniciá y listo — la migración es atómica
y el DID no cambia.

### `acl import <did> <ed_hex> <x_hex> [kem_hex]` ✅

Agrega un nodo a tu lista de confianza (el paquete que genera `share`
del otro nodo). La ACL vive en vivo: persiste al instante. La 4ª clave
es opcional: con ella el peer habla **híbrido PQ**; sin ella queda
legado (IK clásico) y todo sigue andando igual.

```
xion@DerinKuyu:~$ acl import did:maia:GR7P… e50c77ea… 86ad8fd0… cd924a94…
✅ Peer importado: did:maia:GR7P…
   Tip: alias add <nombre> did:maia:GR7P…
```

Para subir a híbrido un peer que ya estaba en 3 claves: re-importalo
con las 4 en ambos nodos y el túnel pasa a híbrido solo.

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

## 🕳️ `remote` — SSH por DID (nivel 1.2) 🆕

**La capa host del ecosistema: consola remota — root o el usuario que
elijas — por el mismo túnel Noise, con una clave SSH que nace y muere
con cada sesión.** Hermano de sangre del XFER: mismos frames de texto
por la sesión que ya existe, mismo `SendDirect` — **jamás relay**: el
faro no ve NI UN BYTE del ssh.

### Las tres puertas

| Puerta | Qué controla | Dónde vive |
|---|---|---|
| 1 · ACL del kernel | quién puede hablarme | `~/.xion/acl.json` (la de siempre) |
| 2 · catálogo sellado | quién puede administrar el host | `~/.xion/remote-admin.json` — firmado Ed25519 por el dueño. Nivel 1: un solo perfil, ROOT_ADMIN |
| 3 · clave efímera | con qué clave y por cuánto tiempo | ed25519 que nace y muere con la sesión; el sshd de sesión NO acepta passwords ni claves permanentes |

### `remote <alias|DID> [usuario]` ✅ — el túnel

El cliente genera un par ed25519 efímero y lo manda DENTRO del pedido
(cifrado por Noise). Si sos ROOT_ADMIN en ese nodo, el bastión levanta
`sshd -i -f <cfg>` con config aislado, host key propio y un
authorized_keys de UNA clave: la tuya de esta sesión. Sesión real de
campo (xeon → tvbox, ambos detrás de WARP):

```
xion@DerinKuyu:~$ remote tvbox root
🕳 abriendo túnel SSH hacia tvbox (solo directo, jamás relay)...
✅ Túnel listo — el nodo remoto ejecutó su openssh de sesión (puerto local 42461).
   Clave efímera SHA256:gE2ZeRTKW5JnMUFyshvk2E2deprpiL9awYwneA36l3M · host de sesión SHA256:ln9MngvLKiffiSB6+KIjCWLEWoFybglMS7V6+2ugeOA. Exit para volver.
   · MULTITAREA: Ctrl+T alterna ssh ↔ chat en vivo (multi.go)
Warning: Permanently added '[127.0.0.1]:42461' (ED25519) to the list of known hosts.
[root@alarm ~]# ip a
2: end0: <BROADCAST,MULTICAST,UP,LOWER_UP> ... inet 192.168.1.110/24 ...
5: wg0: <POINTOPOINT,NOARP,UP,LOWER_UP> ... inet 172.16.0.2/32 ...
[root@alarm ~]# exit
logout
Connection to 127.0.0.1 closed.

🔒 sesión ssh cerrada limpia
🔒 túnel b17cc8d13e4eabded2705c92af02a0d5 con did:maia:G…EdWnsnP5 cerrado (↑29.5KB ↓33.2KB, 52s)
```

Del lado del bastión (tvbox), el mismo túnel:

```
🕳 openssh de sesión listo ante el pedido de did:maia:8…BfnXNzom — clave SHA256:gE2ZeRTKW5JnMUFyshvk2E2deprpiL9awYwneA36l3M · host SHA256:ln9MngvLKiffiSB6+KIjCWLEWoFybglMS7V6+2ugeOA (ROOT_ADMIN)
🔒 túnel b17cc8d13e4eabded2705c92af02a0d5 con did:maia:8…BfnXNzom cerrado (↑33.2KB ↓29.5KB, 52s)
```

Verificación cruzada: mismo ID de túnel, mismas huellas de clave y
host, contadores espejados (lo que uno sube, el otro baja) — dos
extremos independientes contando la misma historia.

Sin `[usuario]` el destino es `root@`; con `remote tvbox alarm`
entrás como alarm. Sin TTY (o sin `ssh` local), la consola devuelve el
comando armado y listo: `ssh -i ~/.xion/tun-key-<id> -p <puerto>
root@127.0.0.1` — válido unos segundos, vence solo.

### `remote allow <alias|DID>` · `remote revoke <alias|DID>` ✅ — la puerta 2 en mano del dueño

```
xion@DerinKuyu:~$ remote allow xeon
✅ ROOT_ADMIN habilitado para xeon (did:maia:8…) — catálogo re-firmado
   (los túneles ya abiertos siguen hasta cerrarse: remote list para verlos)
```

`revoke` saca el DID y re-firma el catálogo. Los túneles vivos no se
cortan en seco: siguen hasta cerrarse. Nadie se habilita solo — el
catálogo lo firma la clave del dueño, y si la firma no cierra, es
**fail closed**.

### `remote admins` ✅ — el catálogo sellado

```
xion@DerinKuyu:~$ remote admins
Catálogo ROOT_ADMIN (remote-admin.json) — firma verificada ✔
  xeon  did:maia:8…BfnXNzom
```

Catálogo corrupto o vacío = fail closed ("nadie puede abrir la capa
host"). Se repara con `remote allow <DID>`, que re-firma el catálogo
entero.

### `remote list` · `remote close <prefijo>` ✅ — túneles vivos

```
xion@DerinKuyu:~$ remote list
  b17cc8d1…  cliente  tvbox  ↑1.5KB ↓2.1KB  listo
      edad 12s — remote close b17cc8d1
```

Cada nodo ve ambos roles: `bastión` = te abrieron, `cliente` = abriste.
`remote close b17cc8d1` corta por prefijo del ID.

### `remote log [n]` · `remote log verify` ✅ — el registro firmado

La evidencia, no la promesa: cada sesión queda en
`~/.xion/remote-sesiones.jsonl` — JSON por línea con DID, huellas de
clave y host, bytes, duración y motivo de cierre — firmado con la
Ed25519 del nodo. `remote log` muestra las últimas n (default 10):

```
xion@DerinKuyu:~$ remote log
  2026-11-22 00:13:55  ABIERTO  tvbox  clave SHA256:gE2ZeRTK… · host SHA256:ln9Mngv…
  2026-11-22 00:14:47  CERRADO  tvbox  ↑29.5KB ↓33.2KB  52s (EOF)
   (remote log verify re-verifica todas las firmas)

xion@DerinKuyu:~$ remote log verify
✔ 2 registros, TODAS las firmas verificadas (Ed25519 del nodo)
```

Si una firma no cierra: `✘ N de M registros con firma INVÁLIDA —
investigalo ya`. Ese registro es lo que un auditor pide: quién entró,
con qué clave, cuánto movió, cuánto duró — y firmado, no prometido.

### Rechazos (TUNNO) y qué te dice la consola

| Motivo | Qué significa |
|---|---|
| `NOT_ADMIN` | no estás en el catálogo → el dueño debe hacer `remote allow <tu-DID>` |
| `BAD_KEY` | tu clave efímera no llegó bien (¿versión vieja del kit en algún lado?) |
| `SSHD_DOWN` / `SSHD_NOT_READY` | el nodo no pudo levantar su sshd de sesión (el diagnóstico queda en SU consola) |
| `RATE_LIMIT` / `TOO_MANY` | demasiados túneles: esperá un minuto o cerrá alguno |
| `CLOSING` | el engine se está apagando |

### Bajo el capó: protocolo TUN v2

Frames de texto por la sesión Noise (herencia XFER), header hasta el
primer `\n`:

| Frame | Qué hace |
|---|---|
| `TUNREQ:<id>` + authorized_keys | el pedido; id de 128 bits de entropía |
| `TUNOK:<id>` + huella de host | sshd de sesión VIVO: la sonda leyó su banner SSH-2.0 real |
| `TUNNO:<id>:<motivo>` | rechazo idempotente |
| `TUNC:` / `TUNS:` `<id>:<n>` | datos — chunks de 1 KB numerados |
| `TUNACKC` / `TUNACKS` | ACK por chunk (stop-and-wait: interactivo, retransmite barato) |
| `TUNBYE:<id>:<motivo>` | cierre limpio de cualquier lado |

Garantías selladas por auditoría (niveles 0 → 1.2):

- **Jamás relay** — mismo `SendDirect` que el XFER. La "chispa" (fix K)
  despierta la sesión directa dormida con un ping por el camino del
  chat; el túnel en sí jamás viaja por el faro. Paciencia: 24 s (el
  directo en frío, medido en campo detrás de WARP: ~20 s).
- **Un sshd PROPIO por sesión** (`sshd -i -f <cfg>`), sin fallback a
  ningún servidor ajeno (H).
- **Sin passwords, sin agente, sin claves del operador**: el ssh
  cliente va con `IdentitiesOnly`, `BatchMode`,
  `PasswordAuthentication=no` y known_hosts propio que muere con el
  túnel.
- **Límites y vencimientos**: túneles totales y por DID, rate limit de
  ejecuciones de sshd, cierre IDLE de ambos lados (E, F).
- **Ninguna clave sobrevive a un reinicio**: `tun-sess/` se barre al
  arrancar — un SIGKILL deja residuo, el residuo se borra (J).
- **El `^C` durante el ssh es DEL ssh** — la consola no se muere; los
  eventos del remote se acumulan y se muestran al volver.

### Los archivos en disco (`~/.xion/`)

| Archivo | Qué es |
|---|---|
| `remote-admin.json` | catálogo ROOT_ADMIN sellado (firma Ed25519 del dueño) |
| `remote-sesiones.jsonl` | registro firmado de sesiones — lo que lee `remote log` |
| `tun-key-<id>` · `tun-kh-<id>` | clave y known_hosts efímeros del cliente (0600, borrados al salir; la privada vive en memoria) |
| `tun-sess/` | sesiones del bastión (host key + authorized_keys + sshd_config) — se barren al arrancar |

### ⌨️ Multitarea: Ctrl+T (multi.go) 🆕

Con el ssh del túnel en pantalla, **Ctrl+T alterna ssh ↔ chat sin
cortar nada**. El mismo teclado, la misma pantalla, dos corrientes:

```
· CHAT con tvbox — Enter manda · ^U limpia · ^C descarta · Ctrl+T vuelve al ssh
too bien amigo?
📤 Vos → tvbox: too bien amigo?
[00:14:17] 💬 tvbox: si perfecto che

· SSH — Ctrl+T vuelve al chat
curl ifconfig.me
104.28.214.35
```

- En modo ssh: cada byte viaja al proceso remoto (`-tt`: el `^C` es
  SIGINT remoto; el Ctrl+T local NUNCA llega al remoto).
- En modo chat: Enter manda, Backspace borra un rune UTF-8 completo,
  `^U` limpia la línea, `^C` la descarta.
- ¿A quién le habla el chat? Al chat activo (`to <alias>`) si lo hay;
  si no, al nodo del túnel. Podés abrir el ssh a tvbox y chatear con
  otro peer a la vez.
- Limitación asumida: una sola corriente en pantalla — el output del
  ssh puede pisar la línea de chat en curso (nada se pierde: el
  mensaje que llega se imprime entero).

---

## 🔐 Criptografía Poscuántica — ya la tenemos, y en la consola ya está viva

No es plan ni roadmap: **está en el kernel y funciona** (auditado,
HEAD `33b8d17`).

**Qué es.** Un KEM híbrido **X25519 + ML-KEM-768** (NIST FIPS 203,
`crypto/mlkem` de la stdlib — cero dependencias nuevas): el secreto
combinado de las dos mitades alimenta el Noise IK del kernel como
PresharedKey (patrón **IKpsk2** de flynn/noise v1.1.0). El blob viaja
como payload del mensaje 1 del handshake — un datagrama que ya
existía: el msg1 pasa de 96 B a 1218 B (blob de 1122 B: versión + epoch
+ X25519 efímera 32 B + ciphertext ML-KEM 1088 B). Sin puertos nuevos,
sin capas nuevas, sin dependencias nuevas. **Mata el
harvest-now-decrypt-later**: grabar el tráfico de hoy ya no sirve para
descifrarlo mañana.

**Qué no es.** No reemplaza a Noise (lo alimenta), no toca al
PSK_señal del enjambre de faros, no es una librería PQ de propósito
general.

**Compatibilidad por adición.** Un peer sin clave ML-KEM — tres claves
(legado) o la APK SanJorge, que se hizo **sin PQ a propósito** — habla
IK clásico con un nodo PQ. La red jamás se parte por esto. El
respondedor decide la rama por el LARGO del mensaje: **downgrade
imposible por diseño**, cualquier adulteración revienta ruidosa.

**La huella PQ.** `f84b-a277-c946-b632` — la identidad corta del
híbrido, visible en el banner y en `whoami` de cada nodo. Se compara
fuera de banda (en persona, por voz): es el control anti-MITM de la
capa PQ.

**Co-firma de DID-docs.** Ed25519 ‖ **ML-DSA-65** (NIST FIPS 204)
sobre el MISMO mensaje, ambas obligatorias (`pkg/pq/firma.go`,
`crypto/mldsa` desde Go 1.25).

| | Estado auditado (HEAD `33b8d17`) |
|---|---|
| Consola Derinkuyu | **Viva**: re-importá el peer con las 4 claves (`acl import <did> <ed> <x> <kem>`) en ambos nodos y el túnel pasa a híbrido solo. `share`/`whoami` ya muestran PubKem + Huella PQ |
| node.key | Migración v2 atómica: semilla ML-KEM con backup `.bak`; si algo falla, arranca en legado — jamás pierde el DID |
| APK SanJorge | **Sin PQ a propósito** (1.6): habla IK clásico con nodos PQ — la red no se parte. Cuando el SDK de zappa exporte/importe las 4 claves, sube a híbrido sin tocar nada más |
| Anti-impersonación | El chequeo PeerStatic-vs-ACL cubre ambas ramas (IK e IKpsk2): impostor = sesión descartada |

Detalle fino: `AUDITORIA_33b8d17.md` §3 · `docs/ANEXO-DISENO-CRYPTO-PQ-v1.md` ·
`docs/CIRUGIA-9.2-PQ-SESION.md`.

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

### `export <archivo> [ruta_destino]` ✅ — Jaula → host

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
  Faro:      190.220.45.26:443  [SÍ]
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
  Faro activo: 190.220.45.26:443  [conectado]

xion@DerinKuyu:~$ faro 192.168.1.100:55123
✅ Faro configurado a 192.168.1.100:55123
   Reiniciá con: ./derinkuyu -faro 192.168.1.100:55123
```

> **Puertos:** `:443` es la fachada pública del faro (antifiltro — viva
> incluso con saldo cero); `55123/UDP` es el puerto nativo detrás, para
> LAN o redes de confianza (`8444/TCP` WSS). Ver `FAROS.md`. En zappa el
> `443` pasa a ser default de fábrica.

### `help` ✅ · `version` ✅ · `clear` / `cls` ✅

`help` lista todo — Jaula incluida, y desde el nivel 1.2 también el
bloque `remote` completo (túnel, allow/revoke, admins, list, log).
`version` muestra `Derinkuyu <ver> commit:<hash> built:<fecha>` (viene
por ldflags; `dev` si compilás a mano). `clear` LIMPIA LA PANTALLA —
no borra nada más (los destructivos son los `reset`).

### `exit` / `quit` / `q` ✅

Cierra limpio: kernel, túneles, faro. Los túneles remote vivos se
cierran con TUNBYE y quedan asentados en el registro firmado.

---

## 📡 Kati por SSH — el faro en remoto

El faro de producción (kati, `190.220.45.26:443`) se administra desde
donde estés, por SSH:

```bash
ssh ubuntu@190.220.45.26
```

Adentro, el orden sagrado de cada despliegue. **Kati compila soberano**:
jamás se le instala un binario de release — los releases de github son
sólo distribución.

```bash
cd /home/ubuntu/xionia-kernel && git pull     # el fuente fresco de main
go build ./cmd/faro-server/                    # kati compila el suyo
sudo install faro-server /home/ubuntu/xionia-kernel/faro-server
sudo setcap cap_net_bind_service=+ep /home/ubuntu/xionia-kernel/faro-server
sudo systemctl restart faro
tail -f ~/faro.log | grep -v ANNOUNCE     # verlo vivir, sin el ruido del latido
```

⚠️ **setcap SIEMPRE después de install.** La capability vive en el
ARCHIVO, no en el comando: todo binario recién compilado nace pelado.
Sin setcap, el 443 da `permission denied` — los puertos menores a 1024
son territorio de root en Linux (lección de Gemini, asentada en
`FAROS.md`, Opción C).

Verificación del despliegue — los tres greps, desde la raíz del repo:

```bash
grep 'ANTIFLUJO 0017' pkg/signaling/signaling.go    # → presente
grep 'deberiaLogRechazo' cmd/faro-server/main.go    # → presente
grep -r MaxAddrsPerDID .                            # → NADA (cuota muerta)
```

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

## 📚 Documentación hermana

| Documento | Qué cubre |
|---|---|
| `FAROS.md` | El casamentero soberano: ANNOUNCE, STUN, blind relay, tu propio faro, fachada 443 |
| `REMOTE-SSH-POR-DID.md` | El diseño del túnel SSH por DID (el memo de las tres puertas) |
| `ANEXO-DISENO-CRYPTO-PQ-v1.md` | El diseño de la capa poscuántica (§3 y §4) |
| `CIRUGIA-9.2-PQ-SESION.md` | Cómo se injertó el PQ en la sesión Noise |
| `AUDITORIA_33b8d17.md` | La auditoría del HEAD actual del kernel |

---

*Referencia completa de comandos de Derinkuyu — XionIA Kernel v9.2 ·
parches 0016/0017 (`release-v15`) · remote nivel 1.2 · PQ viva ·
Corrientes, Argentina* 🧉

*Verificada contra `xionia-audit-33b8d17` (fuentes, 2026-11-21) y la
sesión de campo del 2026-11-22: tvbox ↔ xeon detrás de WARP (AS13335),
túnel SSH por DID de 52 s, cierre limpio con contadores espejados.*
