# 🛠️ Technical Reference

**XionIA Kernel** — Kernel de red soberano en user-space: túneles directos
ICE (pion/ice v4), sesiones Noise IK (flynn/noise), identidad DID Ed25519,
STUN soberano y faro amnésico. Corre en Android (APK y Termux), cualquier
Linux (Debian, Arch) y TV boxes ARM64.

---

## 1. Arquitectura

```
┌─────────────────────────────────────────────────────┐
│                 APLICACIONES                         │
│   SanJorge (APK Flutter) · Derinkuyu (consola)      │
├─────────────────────────────────────────────────────┤
│             pkg/sdk — LA FRONTERA                   │
│   exports FFI/cgo · las claves JAMÁS cruzan ·       │
│   cola display-ready · contrato de strings          │
├─────────────────────────────────────────────────────┤
│             pkg/kernel — FSM orquestadora           │
│   Send (direct→relay) · SendDirect (solo direct)    │
│   dispatch 0x01/0x02 · auto-heal · Wake · Stats     │
├──────────────┬──────────────┬───────────────────────┤
│  pkg/ice     │ pkg/session  │  pkg/identity         │
│  pion/ice v4 │ Noise IK     │  DID · Ed25519/X25519 │
│  probeNet    │ anti-replay  │  ACL · alias · grupos │
│  liveness    │ sin rekey    │                       │
│  Redial      │ (siesta)     │  pkg/xfer (archivos)  │
├──────────────┴──────────────┴───────────────────────┤
│           pkg/transport · pkg/signaling             │
│   blind relay (fallback) · cliente del faro         │
└──────────────────────────┬──────────────────────────┘
                           │ UDP / WSS
              ┌────────────┴────────────┐
              │    FARO AMNÉSICO       │
              │  UDP 3478 + STUN       │
              │  UDP 3479 · WSS 8443   │
              │  Gate DID · RAM only   │
              │  zero logs             │
              └─────────────────────────┘
```

### La Frontera (pkg/sdk)

Toda app consume el kernel exclusivamente vía `pkg/sdk`. Las claves privadas
viven del lado del kernel y jamás cruzan la frontera; la UI jamás importa
kernel/ice/signaling directo. En Android, `cmd/mobile` traduce cgo (ABI de 23
exports `Xionia*`) y `flutter_bridge.dart` los mapea a Dart.

### El Kernel (pkg/kernel)

Orquesta todo. Su API semántica:

| Función | Comportamiento |
|---|---|
| `Send(ctx, did, payload)` | Fast path: sesión Noise activa → **direct**. Sin sesión → relay del faro para ESTE mensaje + renegociación ICE en background. JAMÁS espera al túnel. |
| `SendDirect(did, payload)` | **Solo túnel directo.** Sin sesión Noise viva → error inmediato. La primitiva de XFER y del VoIP: jamás toca el faro, jamás degrada. |
| `Wake()` | Socket fresco del faro SIEMPRE + RedialAll secuenciado después (~3-5s). El despertar de background. |
| `TunnelStatus(did)` | `"direct"` / `"relay"` / `"offline"` — el testigo de la piel. |
| `OnMessage(handler)` | Entrega `(peerDID, payload)` descifrado a la aplicación. |

Regla inviolable: **el discriminador de paquetes no crece.** `0x01` = handshake
Noise, `0x02` = datos de aplicación. Grupos, XFER y voz viajan como payload
encima de esa sesión.

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

### Sesiones (pkg/session)

Noise IK con CipherSuite DH25519 + ChaChaPoly + SHA256. Initiator = DID
lexicográficamente menor (misma comparación que pkg/ice usa para
Dial/Accept — coherencia garantizada en ambas capas). Prologue:
`xionia-kernel:v1:<initiatorDID>:<responderDID>` — un handshake entre A y B
no sirve para B y A reordenados.

**Sin rekey intra-sesión, a propósito.** El rekey unilateral mataba una
dirección del túnel en silencio ("la siesta", v8.9). La clave vive lo que
vive la sesión; si hay que rotar, será protocolo negociado contra un
síntoma real. Cada sesión nueva nace con claves frescas X25519 (Forward
Secrecy por handshake).

### El Faro (amnésico)

El faro **solo presenta pares y reenvía señalización**. Jamás descifra
(ve base64 opaco), jamás persiste (RAM only), jamás loguea. Secuestrás la
máquina: te llevás RAM vacía.

```
ANNOUNCE <DID> <ts> <sigEd25519>|<padding>   → registro DID→addr (TTL)
ACK_IP                                        → confirmación de registro
ICE_INFO <targetDID> <senderDID>|<b64>|<pad>  → forward de señalización
RELAY <targetDID> <senderDID> <payload>|<pad> → blind relay (fallback)
```

- **Gate DID:** el ANNOUNCE se verifica con la firma Ed25519 contra el DID
  reclamado. Sin firma válida, descarte en silencio.
- **STUN RFC5389 en el mismo puerto 3478:** un solo puerto para todo el
  casamiento de pares.
- **Blind relay:** los mensajes de texto pueden viajar cifrados por el faro
  cuando no hay túnel — el faro ve ciphertext opaco, nada más. Con el
  túnel establecido, el faro desaparece del camino (verificado: 20 min de
  chat directo con el faro caído).
- Capacidad medida: **10.000 nodos ≈ 1.3 Mbps y 10 MB de RAM.**

---

## 2. Stack Criptográfico

| Capa | Algoritmo | Biblioteca | Uso |
|------|-----------|------------|-----|
| Firma | Ed25519 | crypto/ed25519 | Identidad, Gate del faro, firma de documentos (sign) |
| Intercambio | X25519 (Noise IK) | flynn/noise | Handshake del túnel, FS por sesión |
| Cifrado | ChaCha20-Poly1305 | flynn/noise | Sesión E2E |
| KDF | HKDF-SHA256 | flynn/noise | Derivación de claves de sesión |
| Anti-replay | Ventana deslizante 64 | propio | Secuencia monotónica por sesión |
| Padding | `crypto/rand` 50–200 B | propio | Anti-DPI en mensajes al faro |

### Flujo E2E (el camino completo de un mensaje)

```
1. ANNOUNCE — cada nodo se registra ante el faro con firma Ed25519 (Gate)
2. ICE_INFO — ufrag + pwd + candidatos viajan por el faro (JSON en base64,
   opacos para él)
3. Perforación — pion/ice establece ice.Conn directo UDP (NAT, CGNAT, VPN)
4. Handshake Noise IK (0x01) — initiator = DID lexicográfico menor
   · PeerStatic queda enclavada por IK y se verifica contra la ACL:
     un impostor de DID resulta en sesión descartada
5. Datos (0x02) — [seq 8B BE][ChaCha20-Poly1305], anti-replay ventana 64
6. El faro queda fuera del camino — el túnel es P2P puro
```

### Formato en el wire (sesión Noise)

```
[discriminador 1B][seq 8B big-endian][AEAD(payload) + tag 16B]
        0x01 handshake  └── 0x02 datos ──┘
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
| `__PING__` | RTT E2E (comando `ping` de la consola) | — |

---

## 4. Directorios del Nodo

```
$XION_HOME (consola: ~/.xion · Android: storage privado de la app)
├── node.key            # Identidad Ed25519 + X25519 (0600)
├── acl.json            # Peers autorizados (0600)
├── aliases.json        # Alias → DID (0600)
├── groups.json         # Grupos (0600)
├── faros.json          # Faro persistente — auto-conecta al abrir (0600)
├── history.txt         # Historial de la shell
├── xionia.log          # Log del SDK (Android)
├── incoming/           # Archivos XFER recibidos
└── workspace/          # La Jaula soberana de la consola
```

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

- Go 1.22+
- NDK r27 (para el `.so` de Android)
- Flutter (para la piel SanJorge)

### Build

```bash
# Consola (Linux nativo)
go build -o derinkuyu ./cmd/derinkuyu/

# Faro (Linux — la única plataforma del faro, a propósito)
go build -o faro-server ./cmd/faro-server/

# Tests
go test ./...

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

### Despliegue de consolas (probado)

```bash
# En el nodo dev:
python3 -m http.server 8080
# En el nodo remoto:
wget http://<dev>:8080/derinkuyu && md5sum derinkuyu   # md5-notario
```

---

## 7. Seguridad

### Permisos de archivos

| Archivo | Permisos | Razón |
|---------|----------|-------|
| `node.key` | `0600` | Claves privadas Ed25519 + X25519 |
| `acl.json` | `0600` | Peers autorizados |
| `aliases.json` | `0600` | Grafo social local |
| `groups.json` | `0600` | Grupos |
| `faros.json` | `0600` | Infraestructura del usuario |
| `incoming/`, `workspace/` | `0700` | Contenido del usuario |

### Anti-DPI

Todo mensaje al faro lleva padding aleatorio (50–200 bytes) — el tamaño
del paquete no revela contenido. Un ANNOUNCE y otro tienen el mismo aspecto.

### Anti-replay

Secuencia monotónica de 64 bits por sesión + ventana deslizante de 64.
Reproduce un paquete capturado: lo rechaza la ventana.

### Anti-suplantación (tres capas)

1. **Gate del faro:** ANNOUNCE firmado Ed25519, verificado contra el DID.
2. **Noise IK:** el iniciador enclava la estática del responder — un
   responder falso no puede completar el handshake.
3. **Verificación contra ACL:** al completar, `PeerStatic` se compara con
   la PubKeyX de la ACL para ese DID. impostor = sesión descartada.

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

---

## 8. Roadmap

| Obra | Estado | Contenido |
|------|--------|-----------|
| **v9.1** | ✅ Producción | XFER consola · Piel Soberana · consola enriquecida (workspace + sign/verify) |
| **VoIP solo-direct** | 🔄 Diseño sellado | Frames Opus (60–120 B) por SendDirect — jamás relay. Señalización CALL/ACCEPT/BYE como payload de sesión. "El chat perdona 2s, una llamada no perdona 200ms." |
| **Firma de binarios** | 🔄 Planificado | Releases con SHA256 + firma Ed25519 del proyecto |
| **Builds reproducibles** | 🔄 Planificado | Compilar la fuente = mismo hash que el binario oficial |
| **LATIDO** | 💤 Diseño | Idle 60s + push soberano por el faro + buzón (versión paga) |
| **Consola web** | 📐 Spec listo | derinkuyu_v5.html — "sin XionIA podés leer, con XionIA podés publicar" |
| **Federación de faros** | 🔭 Horizonte | La evolución — jamás DHT global |

---

Última actualización: Octubre de 2026

*Documento técnico de XionIA — Fernando Martin Lopez, Corrientes, Argentina* 🧉
