# 🛠️ Technical Reference

**XionIA Kernel** — Kernel de red soberano en user-space: túneles directos
ICE (pion/ice v4), sesiones Noise IK/IKpsk2 **híbridas post-cuánticas**
(X25519 + ML-KEM-768, FIPS 203), identidad DID Ed25519, STUN soberano y faro
amnésico casamentero. Corre en Android (APK y Termux), cualquier Linux
(Debian, Arch) y TV boxes ARM64.

*Documento compañero: KERNEL.md (filosofía) · FAROS.md (el faro) ·
COMMANDS.md (la consola) · REMOTE-SSH-POR-DID.md (administración remota) ·
ANEXO-DISENO-CRYPTO-PQ-v1.md y CIRUGIA-9.2-PQ-SESION.md (la capa PQ).*

---

## 1. Arquitectura

```
┌─────────────────────────────────────────────────────┐
│                 APLICACIONES                        │
│   SanJorge (APK Flutter) · Derinkuyu (consola)      │
├─────────────────────────────────────────────────────┤
│             pkg/sdk — LA FRONTERA                   │
│   exports FFI/cgo · las claves JAMÁS cruzan ·       │
│   cola de sondeo (msgCap 1000) · 24 símbolos ABI    │
├─────────────────────────────────────────────────────┤
│             pkg/kernel — FSM orquestadora           │
│   Send (direct→relay) · SendDirect (solo direct)    │
│   dispatch 0x01/0x02 · auto-heal · Wake · Stats     │
│   AddPeerPQ por type assertion — la interface       │
│   quedó intacta: el kernel no supo nunca de KEMs    │
├──────────────┬──────────────┬───────────────────────┤
│  pkg/ice     │ pkg/session  │  pkg/identity         │
│  pion/ice v4 │ Noise IK /   │  DID · Ed25519/X25519 │
│  probeNet    │ IKpsk2 (PQ)  │  ML-KEM (node.key v2) │
│  liveness    │ anti-replay  │  ACL (+4ª clave KEM)  │
│  Redial      │ sin rekey    │  alias · grupos       │
│              │ (siesta)     │                       │
│  pkg/pq:     │              │  pkg/xfer (archivos)  │
│  ML-KEM-768  │              │  sendMu global muerto │
│  ML-DSA-65   │              │  (transfers en paral.)│
├──────────────┴──────────────┴───────────────────────┤
│           pkg/transport · pkg/signaling             │
│   blind relay (fallback) · cliente del faro ·       │
│   antiflujo: cooldown 15s · puerto local estable    │
└──────────────────────────┬──────────────────────────┘
                           │ UDP / WSS
              ┌────────────┴────────────┐
              │  FARO CASAMENTERO       │
              │  UDP 55123 + STUN       │
              │  (bootstrap fijo)       │
              │  UDP 55125 · WSS 8444   │
              │  UDP 443 (trampa)       │
              │  corral rota 55000-60k  │
              │  Gate DID · rebind      │
              │  sin cuotas · RAM only  │
              │  zero logs              │
              └─────────────────────────┘
```

### La Frontera (pkg/sdk)

Toda app consume el kernel exclusivamente vía `pkg/sdk`. Las claves privadas
viven del lado del kernel y jamás cruzan la frontera; la UI jamás importa
kernel/ice/signaling directo. En Android, `cmd/mobile` traduce cgo (ABI de 24
símbolos `Xionia*` — el notario de build) y `flutter_bridge.dart` los mapea a
Dart. El puente estructurado (9.2.3) trae cola de sondeo con `msgCap=1000`:
la piel pregunta, el kernel entrega display-ready — sin callbacks cruzando
la frontera.

### El Kernel (pkg/kernel)

Orquesta todo. Su API semántica:

| Función | Comportamiento |
|---|---|
| `Send(ctx, did, payload)` | Fast path: sesión Noise activa → **direct**. Sin sesión → relay del faro para ESTE mensaje + renegociación ICE en background. JAMÁS espera al túnel. |
| `SendDirect(did, payload)` | **Solo túnel directo.** Sin sesión Noise viva → error inmediato. La primitiva de XFER, del VoIP y de `remote`: jamás toca el faro, jamás degrada. |
| `Wake()` | Socket fresco del faro SIEMPRE + RedialAll secuenciado después (~3-5s). El despertar de background. |
| `TunnelStatus(did)` | `"direct"` / `"relay"` / `"offline"` — el testigo de la piel. |
| `OnMessage(handler)` | Entrega `(peerDID, payload)` descifrado a la aplicación. |
| `AddPeerPQ(did, ...)` | La 4ª clave de la ACL, por type assertion — **degrada a `AddPeer` en binarios viejos**: la interface `Kernel` nunca cambió. |

Regla inviolable: **el discriminador de paquetes no crece.** `0x01` = handshake
Noise, `0x02` = datos de aplicación. Grupos, XFER, voz y túneles SSH viajan
como payload encima de esa sesión.

### Túneles (pkg/ice)

pion/ice v4 sobre UDP4, STUN derivado del propio faro (soberano — jamás
terceros). Detalles de producción:

- **probeNet artesanal (Android):** pion arma su Net preguntando las
  interfaces — en Android eso muere (`netlinkrib: permission denied`). La
  cura: no preguntar. La IP viva se descubre con un `DialUDP` al faro (no
  envía paquetes, consulta la tabla de rutas).
- **Liveness (8s/24s):** `__PING__` por cada túnel establecido, `__PONG__`
  de respuesta — ambos interceptados en el read-loop, invisibles para el
  kernel y la piel. Sin PONG por 24s = camino de vuelta muerto → reset +
  renegociación. Solo mata a peers que una vez contestaron (compat con
  nodos viejos). El ping además **alimenta el mapping NAT**: el detector
  también es preventivo.
- **Zombi (v8.3):** un ICE_INFO con ufrag NUEVO sobre estado viejo = el
  peer reinició → reset total y renegociación de ambos lados.
- **watchRoute (5s):** la IP local cambia (WiFi↔datos) → todas las directas
  son zombis por definición → reset + reconexión inmediata.
- **Redial:** cooldown de 15s por peer, respeta negociaciones en vuelo.
- **Antiflujo (parche 0017):** el cliente reutiliza su socket/puerto local
  entre reconexiones (el mapeo NAT no cambia, la entrada del gate sobrevive),
  `ConnectFaro` es idempotente ("ya conectado" = NO-OP — resumir la app ya
  no registra nada). Cada disparador de reconexión pasa por el mismo camino
  con cooldown: la lección del incidente del faro árbitro.

### Sesiones (pkg/session)

Noise IK / IKpsk2 con CipherSuite DH25519 + ChaChaPoly + SHA256. Initiator =
DID lexicográficamente menor (misma comparación que pkg/ice usa para
Dial/Accept — coherencia garantizada en ambas capas). Prologue:
`xionia-kernel:v1:<initiatorDID>:<responderDID>` — un handshake entre A y B
no sirve para B y A reordenados.

**Handshake dual desde la v9.2 (la cirugía PQ):**

- **El iniciador encapsula YA** si la ACL del par trae `pubkey_kem`: secreto
  híbrido X25519 + ML-KEM-768 → PSK (32 B) → patrón **IKpsk2** (flynn/noise,
  `PresharedKeyPlacement: 2`). El msg1 pasa de 96 B a **1218 B**: el payload
  que viajaba vacío ahora carga el blob KEM (1122 B), **cifrado por el AEAD
  del propio msg1** — el pasivo que graba ni siquiera ve el ciphertext KEM.
- **El respondedor decide por el LARGO del msg1**: 96 B = par legado (IK
  clásico byte a byte); 1218 B = par PQ → decapsula ANTES de escribir msg2
  (`SetPresharedKey` post-lectura, permitido por flynn/noise en psk2). No
  hay negociación en el wire que un MITM pueda toquetear: recortar o tizar
  un bit muere en el tag AEAD.
- `NewHandshaker` / `Manager.NewHandshake` ganaron `peerPubKem ...[]byte`
  VARIÁDICO — las llamadas de 3 args de hoy siguen compilando igual. El
  Manager **auto-arma el híbrido desde la ACL**: el kernel no supo nunca de
  KEMs. Mismo 2-RTT de siempre, mismo prologue, misma verificación contra
  ACL. Premium medido: **~257 µs y +1122 B, una vez por sesión**.

**Sin rekey intra-sesión, a propósito.** El rekey unilateral mataba una
dirección del túnel en silencio ("la siesta", v8.9). La clave vive lo que
vive la sesión; si hay que rotar, será protocolo negociado contra un
síntoma real — y mezclará PQ. Cada sesión nueva nace con claves frescas
X25519 **y** secreto ML-KEM (Forward Secrecy por handshake, dos alfabetos).

### El Faro (amnésico y casamentero)

El faro **solo presenta pares y reenvía señalización**. Jamás descifra
(ve base64 opaco), jamás persiste (RAM only), jamás loguea. Secuestrás la
máquina: te llevás RAM vacía. Y desde el parche 0016, su doctrina quedó
tallada: **casamentero, no árbitro** — verifica la firma y presenta; jamás
decide quién habla ni cuántas veces. Sin cuotas por DID, sin permisos, sin
"demasiadas direcciones para este DID": la firma es el único pase.

```
ANNOUNCE <DID> <ts> <sigEd25519>|<padding>   → registro DID→addr
ACK_IP                                        → confirmación + detección de roaming
ICE_INFO <targetDID> <senderDID>|<b64>|<pad>  → forward de señalización
RELAY <targetDID> <senderDID> <payload>|<pad> → blind relay (fallback)
STUN (RFC 5389)                               → en el mismo puerto de señalización
```

- **Gate DID:** el ANNOUNCE se verifica con la firma Ed25519 contra el DID
  reclamado. Sin firma válida, descarte en silencio (y a lo sumo un "Gate
  rechazo" por dirección enmascarada por ventana de 60 s — freno de logs,
  0017).
- **Registro con rebind:** una entrada nueva con el mismo did+ip+red
  **reemplaza** la vieja — nada acumula. Ciclo: announce cada 10 s;
  expiración del gate a las 2 h (el kernel la descubre solo y re-entrega).
- **Puertos (log confirmado de producción):** UDP **55123** (señalización +
  STUN RFC 5389 en el mismo puerto — el bootstrap que jamás rota), UDP
  **55125** (relay interno), TCP **8444** (WSS), UDP **443** (la trampa —
  el faro atiende el puerto del HTTPS para las murallas de los carriers) y
  corral de rotación **55000–56000** con `pkg/rota` (semilla
  SHA256(DID del faro)[:4], ventana 1 h).
- **Blind relay:** los mensajes de texto pueden viajar cifrados por el faro
  cuando no hay túnel — el faro ve ciphertext opaco, nada más. Con el túnel
  establecido, el faro desaparece del camino (verificado: 20 min de chat
  directo con el faro caído).
- Capacidad medida: **10.000 nodos ≈ 1.3 Mbps y 10 MB de RAM.**

---

## 2. Stack Criptográfico

| Capa | Algoritmo | Biblioteca | Uso |
|------|-----------|------------|-----|
| Firma | Ed25519 | crypto/ed25519 | Identidad, Gate del faro, firma de documentos (`sign`), registro de `remote` |
| Intercambio | X25519 (Noise IK / IKpsk2) | flynn/noise | Handshake del túnel, FS por sesión |
| KEM | **ML-KEM-768 (FIPS 203)** | crypto/mlkem (stdlib) | Secreto híbrido → PSK del IKpsk2 — dos alfabetos por sesión |
| Firma PQ | **ML-DSA-65 (FIPS 204)** | crypto/mldsa (stdlib) | Co-firma de identidad: DID-docs del faro, registro, share v2 — determinista, 3376 B |
| Cifrado | ChaCha20-Poly1305 | flynn/noise | Sesión E2E |
| KDF | HKDF-SHA256 | flynn/noise | Derivación de claves de sesión |
| Combinador PQ | SHA-256 con label de dominio | propio (pkg/pq) | `PSK = SHA-256("xionia/pq/v1/ikpsk2/x25519+mlkem768" ‖ ss_X25519 ‖ ss_MLKEM ‖ transcripción)` |
| Anti-replay | Ventana deslizante 64 | propio | Secuencia monotónica por sesión |
| Padding | `crypto/rand` 50–200 B | propio | Anti-DPI en mensajes al faro |

**Doctrina PQ (regla R4): nada de libs de terceros mientras esté en la
stdlib.** `pkg/pq` es 100 % stdlib de Go, CGO=0 — la misma superficie de
auditoría de siempre, ahora con FIPS 203/204 finales (no drafts).

### La capa post-cuántica (v9.2) — resumen ejecutivo

| Propiedad | Valor |
|---|---|
| Amenaza cerrada | harvest-now, decrypt-later: cada handshake IK grabado ya no se abre "el día-C" solo con romper X25519 |
| El cable | msg1: 96 B (legado) → **1218 B** (híbrido, blob KEM 1122 B dentro del AEAD) · msg2: 48 B igual · 1246 B < 1500: un datagrama |
| Activación | Por ACL, no por versión: 4ª clave (`pubkey_kem`) presente en ambos lados → IKpsk2; si no, IK clásico byte a byte |
| Downgrade | Ruidoso por construcción: la rama la decide el largo del msg1 — no hay negociación que atacar |
| Huella PQ | `SHA-256(PubX ‖ PubKem)` → 16 caracteres (SAS): verificación por voz, humano a humano |
| Co-firma | Ed25519 + ML-DSA-65 sobre el mismo documento (DID-docs, share v2) — JAMÁS en frames de datos |
| Qué NO cubre | node.key en disco (0600), PSK de señalización, distribución de confianza (ACL/share) — superficie clásica, declarada |

Diseño: `docs/ANEXO-DISENO-CRYPTO-PQ-v1.md` · Acta y evidencia:
`docs/CIRUGIA-9.2-PQ-SESION.md` · Verificador de cancha: `cmd/xqpq-demo`
(autotest 12/12).

### Flujo E2E (el camino completo de un mensaje)

```
1. ANNOUNCE — cada nodo se registra ante el faro con firma Ed25519 (Gate)
2. ICE_INFO — ufrag + pwd + candidatos viajan por el faro (JSON en base64,
   opacos para él)
3. Perforación — pion/ice establece ice.Conn directo UDP (NAT, CGNAT, VPN)
4. Handshake Noise IK / IKpsk2 (0x01) — initiator = DID lexicográfico menor
   · ACL de 4 claves en ambos lados → híbrido: X25519 + ML-KEM-768 curtien-
     do la misma sesión; el respondedor decide la rama por el largo del msg1
   · PeerStatic queda enclavada por IK y se verifica contra la ACL:
     un impostor de DID resulta en sesión descartada
5. Datos (0x02) — [seq 8B BE][ChaCha20-Poly1305], anti-replay ventana 64
6. El faro queda fuera del camino — el túnel es P2P puro
```

### Formato en el wire (sesión Noise)

```
[discriminador 1B][seq 8B big-endian][AEAD(payload) + tag 16B]
        0x01 handshake  └── 0x02 datos ──┘

msg1 del handshake: 96 B (IK clásico) · 1218 B (IKpsk2 híbrido — payload =
blob KEM de 1122 B: versión 1 B · epoch 1 B · efímera X25519 32 B · ct 1088 B)
```

Longitud mínima aceptada: 24 bytes (8 de seq + 16 de tag). Secuencia 0
inválida, duplicados y rezagados de más de 64 respecto del máximo: rechazo.

---

## 3. Payloads de aplicación (post-descifrado)

Todo comando de aplicación viaja como texto dentro de la sesión cifrada:

| Prefijo | Tipo | Ejemplo |
|---------|------|---------|
| texto plano | Mensaje 1:1 | `hola` |
| `GROUP:<alias>:` | Mensaje de grupo | `GROUP:equipo:reunión a las 15` |
| `GROUP_SYNC:<alias>:` | Sincronización de grupo | JSON del grupo |
| `GROUP_DELETE / GROUP_KICKED / GROUP_LEAVE` | Ciclo de vida de grupo | — |
| `GROUP_JOIN / GROUP_ACL_SHARE` | Incorporación y distribución de ACL | — |
| `XFER1/2/3` | Archivos por el túnel | ver §5 |
| `TUNREQ / TUNOK / TUNNO` | `remote`: pedido / aceptación / rechazo del túnel SSH | ver REMOTE-SSH-POR-DID.md |
| `TUNEX` / `TUNOUT` / `TUNBYE` | `remote`: datos de ida / vuelta (frames numerados con ACK) y cierre | — |
| `__PING__` / `__PONG__` | Liveness 8s/24s y RTT E2E (comando `ping`) — interceptados en el read-loop, invisibles para la app | — |

---

## 4. Directorios del Nodo

```
$XION_HOME (consola: ~/.xion · Android: storage privado de la app)
├── node.key              # Identidad: Ed25519 + X25519 + semilla ML-KEM (v2 · 0600)
├── acl.json              # Peers autorizados + pubkey_kem opcional por par (0600)
├── aliases.json          # Alias → DID (0600)
├── groups.json           # Grupos (0600)
├── faros.json            # Faro persistente — auto-conecta al abrir (0600)
├── remote-admin.json     # Catálogo ROOT_ADMIN firmado Ed25519 (0600)
├── remote-sesiones.jsonl # Registro firmado de sesiones remote (0600)
├── history.txt           # Historial de la shell
├── xionia.log            # Log del SDK (Android)
├── incoming/             # Archivos XFER recibidos
└── workspace/            # La Jaula soberana de la consola
```

El `node.key` v2 es aditivo: la semilla ML-KEM (64 B, forma d‖z) se genera al
primer arranque si falta, **el DID no cambia** (deriva solo de Ed25519) y un
binario viejo que lea un v2 ignora el campo extra. Rotación de claves KEM =
mano de operador, como toda identidad de la casa.

### La Jaula (workspace)

La consola Derinkuyu trae comandos Unix (`ls`, `cat`, `cd`, `mkdir`, `rm`,
`cp`, `mv`, `head`, `tail`, `grep`, `edit`) que operan **exclusivamente**
dentro de `workspace/` — no hay escape al filesystem del host. El paso de
archivos entre el host y la Jaula es explícito y auditable:

```
import <ruta_host>     → host → workspace
export <archivo>       → workspace → host
sign <archivo>         → firma Ed25519 del nodo
verify <archivo>       → verifica la firma
```

Flujo del trabajo soberano: `import → escribir → sign → xfer → verify`.
Comunicarse, escribir y firmar sin salir de la Jaula.

> Fix de auditoría (serie FIX AUDIT): con `to <peer>` activo, un comando de
> la Jaula sin `/` se mandaba literal como chat en vez de ejecutarse. La
> cura no fue arreglar la lista a mano otra vez: el test de regresión
> `TestIsShellCommandCubreElSwitch` parsea el switch REAL de `main.go` y
> falla solo si alguien lo desincroniza.

---

## 5. XFER — Archivos por los Túneles

Motor en `pkg/xfer`. Viaja por la sesión Noise existente (0x02 — el
discriminador NO crece) con ACK por chunk:

```
XFER1:<id>:<nombre>:<total>:<bytes>   anuncio (3 copias, idempotente)
XFER2:<id>:<n>\n<chunk>               chunk + cuerpo binario
XFER3:<id>:<n>                        ACK (solo por SendDirect)
```

- **Jamás relay:** todo por `SendDirect`. Sin túnel directo, el motor
  espera — no degrada.
- **ACK por chunk** con retransmisión (30 intentos, timeout 3s) y espera
  paciente del anuncio (60s) mientras el liveness renegocia.
- **Transfers concurrentes (FIX AUDIT):** el `sendMu` global murió — cada
  transferencia saliente opera sobre estado 100 % propio. Mandarle a B ya
  no espera a que termine (o falle) lo que se le estaba mandando a A.
- **Integridad total:** el receptor arma `<dataDir>/incoming/`, trunca al
  tamaño anunciado y renombra al nombre final (únicos si colisiona). Un
  archivo no puede quedar cojo: sin el último chunk no hay archivo.
- **CLI:** `xfer <alias|DID> <archivo>` — el Shell es el Rey.
- **GATE histórico:** primer archivo por la Jaula — md5 idéntico en ambos
  nodos, 1ms.

> ⚠️ **Residuo documentado:** en producción Android, un archivo de 450KB
> con chunks de 60KB mató la sesión en bucle (sospecha: fragmentación IP
> del datagrama >50KB — un fragmento perdido = descifrado fallido = auto-
> cura que renegocia y el siguiente chunk la mata otra vez). Pendiente la
> prueba de consola con archivo grande para confirmar; el fix probable es
> una constante (chunk 4KB). XFER queda en consola mientras tanto.

---

## 6. Compilación

### Requisitos

- Go 1.27+ (el módulo está en go 1.27: `crypto/mldsa` en su forma actual lo
  exige — verificado compilando)
- NDK r27 (27.0.12077973, para el `.so` de Android)
- Flutter (para la piel SanJorge)

### GATE (antes de cualquier build de release)

```bash
go build ./... && go vet ./... && go test -race -count=1 ./...
# 16/16 paquetes en verde — el detector de carreras limpio es obligatorio
```

### Build

```bash
# Consola (Linux nativo)
go build -o derinkuyu ./cmd/derinkuyu/

# Faro (Linux — la única plataforma del faro, a propósito)
go build -o faro-server ./cmd/faro-server/

# Release completo (amd64 + arm64 + SHA256SUMS · --upload publica)
bash release-v15.sh

# Android: el .so del kernel
bash cmd/mobile/build_android.sh
# → android/jniLibs/arm64-v8a/libxionia.so
cp android/jniLibs/arm64-v8a/libxionia.so ~/sanjorge-app/android/app/src/main/jniLibs/arm64-v8a/

# Termux directo (build on-device)
GOOS=android GOARCH=arm64 CGO_ENABLED=1 CC=clang \
  go build -ldflags='-checklinkname=0' ./cmd/derinkuyu/

# APK
cd ~/sanjorge-app && flutter build apk --release
```

### Release v1.5 (distribución pública, 2026-11-21)

| Artefacto | Bytes | Destino |
|---|---|---|
| `derinkuyu-linux-amd64` | 10.580.128 | Nodos Linux x86_64 |
| `derinkuyu-linux-arm64` | 9.765.024 | Nodos Linux ARM |
| `faro-server-linux-amd64` | 7.639.200 | Descarga pública (NO para kati) |
| `faro-server-linux-arm64` | 7.078.048 | Descarga pública (NO para kati) |
| `SanJorge-1.6.apk` | 34.487.940 | Teléfonos Android |
| `SHA256SUMS.txt` | 439 | Verificación |

Build: `CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w"` — binarios estáticos
y autocontenidos (corren en cualquier Linux de su arquitectura, sin
toolchain). Fuente: `github.com/mamanga1/xionia-kernel` · Releases:
`github.com/mamanga1/xionia-xtp` (tag `v1.5` = la costura fuente↔binario).

> Regla de la casa: el faro de producción (kati) **compila siempre su propio
> binario** (pull → build → install → `setcap` → restart); los binarios de
> release son solo distribución pública y jamás se deployan en kati.

### Despliegue de consolas (probado)

```bash
# En el nodo dev:
python3 -m http.server 8080
# En el nodo remoto:
wget http://<dev>:8080/derinkuyu && sha256sum derinkuyu   # contra SHA256SUMS
```

---

## 7. Seguridad

### Permisos de archivos

| Archivo | Permisos | Razón |
|---------|----------|-------|
| `node.key` | `0600` | Claves privadas Ed25519 + X25519 + semilla ML-KEM |
| `acl.json` | `0600` | Peers autorizados (+ pubkey_kem) |
| `aliases.json` | `0600` | Grafo social local |
| `groups.json` | `0600` | Grupos |
| `faros.json` | `0600` | Infraestructura del usuario |
| `remote-admin.json` | `0600` | Catálogo ROOT_ADMIN firmado |
| `remote-sesiones.jsonl` | `0600` | Registro firmado de sesiones |
| `incoming/`, `workspace/` | `0700` | Contenido del usuario |

### Anti-DPI

Todo mensaje al faro lleva padding aleatorio (50–200 bytes) — el tamaño
del paquete no revela contenido. Un ANNOUNCE y otro tienen el mismo aspecto.
La fachada 443 (UDP para el faro, WSS detrás del mismo 443/TCP) hace que el
gate parezca tráfico web/QUIC a las murallas de los carriers — verificado
sin saldo en Claro AR (puerta lateral frágil, no se promete como feature).

### Anti-replay

Secuencia monotónica de 64 bits por sesión + ventana deslizante de 64.
Reproduce un paquete capturado: lo rechaza la ventana.

### Anti-suplantación (tres capas)

1. **Gate del faro:** ANNOUNCE firmado Ed25519, verificado contra el DID.
2. **Noise IK:** el iniciador enclava la estática del responder — un
   responder falso no puede completar el handshake.
3. **Verificación contra ACL:** al completar, `PeerStatic` se compara con
   la PubKeyX de la ACL para ese DID. impostor = sesión descartada.

### Post-cuántica (v9.2)

- **Híbrido siempre (R1):** X25519 + ML-KEM-768 en el mismo handshake — si
  los retículos caen queda la curva, si la curva caen quedan los retículos.
  Ningún fallo solo abre la sesión.
- **Downgrade ruidoso (R3):** la rama la decide el largo del msg1. El único
  camino clásico es una ACL local sin `pubkey_kem` — se audita con
  `cat acl.json`, no se ataca desde la red.
- **El blob viaja cifrado:** dentro del AEAD del msg1 — mover un byte
  revienta el tag antes de decapsular (`TestDowngradeImposible`).
- **Reglas de piedra R1–R6** completas en el ANEXO de diseño PQ; la
  auditoría HNDL de la pila confirmó que el handshake de sesión era la
  única superficie clásica que alimentaba claves — todo lo demás ya era
  simétrico (PQ-neutro).

### Auto-heal (los gatillos, todos independientes)

| Gatillo | Dispara | Latencia |
|---|---|---|
| routeWatcher (kernel) — ruta al faro cambia | socket fresco + announce | ~3s |
| watchRoute (ice) — IP local cambia | reset de directas + reconexión | ~5s |
| Liveness — sin PONG 24s | reset + renegociación del túnel | 24s máx |
| Decrypt fallido ×3 | cierre de sesión + renegociación | ~3s |
| Send por relay / recv por relay | Redial en background | cooldown 15s |
| Error de announce (ENETUNREACH) | faro marcado caído, watchdog renace | backoff 2-60s |
| Wake (la piel) | socket fresco SIEMPRE + RedialAll secuenciado | 3-5s |
| Temblor de conexión al faro (0017) | cooldown 15s + socket/puerto local reutilizado; ConnectFaro idempotente | 15s |

---

## 8. Roadmap

| Obra | Estado | Contenido |
|------|--------|-----------|
| **v9.1** | ✅ Producción (octubre) | XFER consola · Piel Soberana · consola enriquecida (workspace + sign/verify) |
| **v9.2 — capa PQ** | ✅ Producción (noviembre, main + tag v1.5) | IKpsk2 híbrido X25519 + ML-KEM-768 · node.key v2 · ACL de 4 claves · Huella PQ · co-firma ML-DSA-65 · go.mod 1.27 |
| **remote — SSH por DID** | ✅ Producción (nivel 1.2) | Tres puertas · catálogo sellado · clave efímera por sesión · registro firmado · Ctrl+T — ver REMOTE-SSH-POR-DID.md |
| **v9.2.4 — faro casamentero** | ✅ Producción | 0016: sin cuotas, rebind, trampa UDP 443 · 0017: antiflujo (cooldown, puerto estable, idempotencia) · release v1.5 |
| **SanJorge 2.0 "zappa"** | 🔄 Siguiente hito | PQ en Android (`gomobile bind` de `pkg/pq` → `.aar`) + VoIP Opus por sesión híbrida — parches 0018-0023 |
| **VoIP solo-direct** | 🔄 Diseño sellado | Frames Opus (60–120 B) por SendDirect — jamás relay. Señalización OFFER/ANSWER/BYE como payload de sesión. "El chat perdona 2s, una llamada no perdona 200ms." Cero servidores de voz. |
| **Enjambre de faros serverless** | 🔄 Diseño aprobado | Buzón v1 por HTTPS/443 a dominios inocuos: anillo híbrido Cloudflare (Worker + Durable Objects) + faros soberanos (KATI, Oracle) — **CF nunca solo**. POST ≤4 KB · lectura única · TTL 120 s |
| **XIP-1 «tarjeta sellada»** | 🔄 Diseño | Primer contacto sin canal privado: código corto humano + PAKE + blob efímero (TTL 15 min) en el buzón |
| **Firma de binarios** | 🔄 Planificado | Hoy: SHA256SUMS verificado en cada release. Falta: firma Ed25519 del proyecto |
| **Builds reproducibles** | 🔄 Planificado | Compilar la fuente = mismo hash que el binario oficial |
| **Seccomp + memoria pinned** | 🔄 Planificado | Filtro de syscalls · claves fuera del GC (Mmap + Mlock) |
| **LATIDO** | 💤 Diseño | Idle 60s + push soberano por el faro + buzón (versión paga) |
| **Consola web** | 📐 Spec listo | derinkuyu_v5.html — "sin XionIA podés leer, con XionIA podés publicar" |
| **Federación de faros** | 🔭 Horizonte | La evolución — jamás DHT global |

---

Última actualización: Noviembre de 2026 — kernel v9.2.4 (parches 0016/0017
en producción) · release v1.5 · 16/16 paquetes en verde con `-race`

*Documento técnico de XionIA — Fernando Martin Lopez, Corrientes, Argentina* 🧉
