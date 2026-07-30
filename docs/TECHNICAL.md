# 🛠️ Technical Reference

**XionIA Faraday** — Red overlay soberana con relay ciego, cifrado E2E
(Ed25519/X25519/ChaCha20-Poly1305), Jaula de Faraday, y Gate DID.
Corre en servidores Xeon, Raspberry Pi, TV boxes y celulares Android.

---

## 1. Arquitectura

```
┌─────────────────────────────────────────────────────┐
│                    NODO XIONIA                       │
│                                                     │
│  ┌───────────┐  ┌───────────┐  ┌───────────────┐   │
│  │  Shell    │  │  Jaula    │  │  IA Local     │   │
│  │  (prompt) │  │  Faraday  │  │  (llama.cpp)  │   │
│  └─────┬─────┘  └─────┬─────┘  └───────┬───────┘   │
│        │              │                │            │
│  ┌─────┴──────────────┴────────────────┴───────┐   │
│  │           src/crypto/ (core)                 │   │
│  │  identity · cipher · acl · alias · groups    │   │
│  │  gate · noise (no conectado — Fase 2)        │   │
│  └─────────────────────┬───────────────────────┘   │
│                        │                            │
│  ┌─────────────────────┴───────────────────────┐   │
│  │           Transporte                         │   │
│  │  UDP 54321 (default) + WSS 443 (fallback)    │   │
│  │  U2P/XTP: Túneles propios (Noise+Curve25519) │   │
│  │           para perforar CGNAT                │   │
│  │           (Fase 2 — no implementado)         │   │
│  └─────────────────────┬───────────────────────┘   │
└────────────────────────┼────────────────────────────┘
                         │
              ┌──────────┴──────────┐
              │    FARO CIEGO       │
              │  (zero-knowledge)   │
              │  UDP 54321 + WSS 443│
              │  Gate DID activo    │
              │  Cross-relay UDP↔WSS│
              │  TTL registry 90s   │
              └─────────────────────┘
```

### Faro Ciego

El faro **solo enruta**. Nunca descifra, nunca almacena, nunca inspecciona.
Es un relay zero-knowledge: ve paquetes cifrados y los reenvía.

Protocolo del faro:

```
ANNOUNCE <DID> <timestamp> <firma>     → registra DID → dirección en RAM
RELAY <targetDID> <senderDID> <payload> → reenvía payload al target
RESPONSE <targetDID> <payload>          → respuesta directa
WHERE_IS <DID>                          → consulta presencia
VERIFY_HASH                             → hash SHA-256 del binario del faro
```

### Gate DID (v1.0)

Todo nodo debe autenticarse con un handshake Ed25519 antes de operar:

```
Nodo → Faro: {"did":"did:maia:...","pub":"...","nonce":"...","ts":...,"sig":"..."}
Faro → Nodo: {"ack":"ok","did":"did:maia:...","ts":...,"nodes":N}
```

Sin handshake válido, el faro descarta **todo** mensaje del nodo en silencio.

### Fixes de auditoría (v1.0, Jul 2026)

| Fix | Descripción |
|-----|-------------|
| Cross-relay UDP↔WSS | El RELAY busca el destino en ambos registries (UDP y WSS) |
| TTL registry (90s) | Expira nodos zombies (antes eternos) |
| verifyAnnounceSig | Verifica firma del ANNOUNCE (anti-spoofing de DID) |
| stripPadding en ANNOUNCE | Saca padding antes de verificar firma |
| Crash por cert faltante | WSS no mata el proceso UDP si faltan certs |
| ACK falso en WSS | No manda ACK si la entrega WSS falla + limpia conn muerta |
| Gate logging | Loguea cuando el Gate rechaza (antes en silencio) |
| Watchdog 20s + ANNOUNCE 10s + retry 3s | Reconexión más rápida, registro más frecuente |
| connMu / quitMu / recover() | Anti-race-condition y anti-panic en mobile.go |

### Cliente de referencia: XionChat (xionia-xtp)

XionChat es el cliente Android de referencia para la Fase 1. Repo separado:
`github.com/mamanga1/xionia-xtp`. Binding Go→Flutter (FFI/cgo) con
notificaciones en background vía `flutter_background_service`, watchdog
de reconexión (20s), y ANNOUNCE con retry (3s).

---

## 2. Stack Criptográfico

| Capa | Algoritmo | Uso |
|------|-----------|-----|
| Firma | Ed25519 | Identidad, autenticación de mensajes |
| Intercambio | X25519 (ECDH) | Derivación de clave compartida |
| Cifrado | ChaCha20-Poly1305 | Cifrado E2E de payloads |
| KDF | SHA-256 | Derivación de Key IDs (4 bytes) |
| Padding | `crypto/rand` | Anti-DPI (tamaño aleatorio 50-200 bytes) |
| Handshake | Noise Protocol IK *(implementado en `src/crypto/noise.go`, NO conectado — Fase 2)* |
| Gate | Ed25519 challenge-response | Autenticación ante el faro |

### Flujo de cifrado E2E

```
1. Alice tiene la clave pública X25519 de Bob (vía ACL)
2. Alice deriva SharedKey = ECDH(Alice.PrivX, Bob.PubX)
3. Alice construye InnerPayload {from, ts, cmd, sig}
4. Alice firma el payload con Ed25519
5. Alice cifra con ChaCha20-Poly1305 usando SharedKey
6. Alice prepende KeyID (4 bytes, SHA-256 de PubX)
7. Alice agrega padding aleatorio (anti-DPI)
8. Alice envía: RELAY <bobDID> <aliceDID> <kid|ciphertext|padding>

Bob recibe:
1. Extrae KeyID → busca en su ACL → encuentra SharedKey
2. Descifra con ChaCha20-Poly1305
3. Verifica firma Ed25519 de Alice
4. Verifica timestamp (±60s, anti-replay)
5. Procesa el comando
```

---

## 3. Formato de Mensajes

### Payload cifrado (sobre el wire)

```
<kid_hex>|<ciphertext_base64>|<padding_random>
```

- `kid_hex`: 4 bytes en hex (Key ID = SHA-256(PubKeyX)[:4])
- `ciphertext_base64`: InnerPayload cifrado con ChaCha20-Poly1305
- `padding_random`: 50-200 bytes aleatorios (`crypto/rand`), anti-DPI

### InnerPayload (antes de cifrar)

```json
{
  "from": "did:maia:...",
  "ts": 1753912345,
  "cmd": "CHAT:hola mundo",
  "sig": "base64(firma Ed25519 del JSON sin sig)"
}
```

### Comandos soportados en `cmd`

| Prefijo | Tipo | Ejemplo |
|---------|------|---------|
| `CHAT:` | Mensaje directo | `CHAT:hola` |
| `GROUP:<alias>:` | Mensaje de grupo | `GROUP:equipo:reunión a las 15` |
| `GROUP_SYNC:<alias>:` | Sincronización de grupo | JSON del grupo |
| `GROUP_DELETE:<alias>` | Eliminación de grupo | — |
| `GROUP_KICKED:<alias>` | Expulsión de grupo | — |
| `GROUP_LEAVE:<alias>:` | Salida de grupo | `GROUP_LEAVE:equipo:did:maia:...` |

---

## 4. Directorios del Nodo

```
$XION_HOME/ (default: ~/.xion)
├── node.key            # Identidad Ed25519 + X25519 (permisos 0600)
├── acl.json            # Lista de peers autorizados (permisos 0600)
├── aliases.json        # Alias locales (permisos 0600)
├── groups/             # Grupos cifrados
│   └── <alias>.json
├── .xion/              # Jaula de Faraday (sandbox)
│   ├── documentos/
│   ├── descargas/
│   └── proyectos/
├── config.json         # Configuración de faro
├── xionia.log          # Log del nodo
└── ia/                 # Modelos GGUF + embeddings (Fase 2 — no implementado)
```

### Jaula de Faraday

Todos los comandos de archivo (`ls`, `cat`, `rm`, `mv`, `cp`, `mkdir`,
`touch`, `edit`) operan **exclusivamente** dentro de `.xion/`. No hay
forma de escapar al sistema de archivos del host desde la shell.

```
xion@nodo:~$ ls /etc/passwd
❌ Acceso denegado: fuera de la Jaula de Faraday

xion@nodo:~$ ls
documentos/  descargas/  proyectos/
```

---

## 5. IA Colaborativa

> ⚠️ **Estado: Fase 2 — no existe código.** Los comandos `ia` descritos
> a continuación son diseño pendiente. No hay implementación en el
> repositorio. Se incluirán cuando la Fase 2 esté cerrada.

### Arquitectura

```
┌─────────────────────────────────────────┐
│           NODO CON IA LOCAL             │
│                                         │
│  ┌─────────────┐  ┌─────────────────┐  │
│  │  Shell Xion │  │  llama.cpp      │  │
│  │  (comando   │──│  (GGUF local)   │  │
│  │   "ia")     │  │  Puerto 8080    │  │
│  └─────────────┘  └─────────────────┘  │
│                                         │
│  Modelos:                               │
│  - llama-3.2-3b-instruct.gguf (2GB)    │
│  - phi-3-mini-4k-instruct.gguf (2.3GB) │
│  - qwen2.5-1.5b-instruct.gguf (1GB)   │
└─────────────────────────────────────────┘
```

### Comandos (diseño pendiente)

```bash
ia start <modelo>          # Iniciar servidor llama.cpp local
ia list                    # Listar modelos disponibles
ia use <modelo>            # Seleccionar modelo activo
ia offer <modelo> <precio> # Ofrecer inferencia a la red
```

### Mercado de Inferencia P2P

Los nodos pueden ofrecer capacidad de inferencia a otros nodos:

```
Nodo A (sin GPU) → "necesito inferencia" → Faro → broadcast
Nodo B (con GPU) → "ofrezco inferencia a 0.001 XION/token" → Faro → relay
Nodo A ←→ Nodo B: sesión directa (U2P) para inferencia
```

*(Fase 2 — no implementado)*

---

## 6. Compilación

### Requisitos

- Go 1.21+
- OpenSSL (para certs del faro WSS)
- `go-prompt` (shell interactiva)
- `gorilla/websocket` (WSS)

### Build

```bash
# Nodo completo (shell + crypto + transporte)
go build -o mesh ./cmd/mesh/

# Faro
go build -o faro ./cmd/faro/

# Cross-compile para Android (arm64)
./build.sh android
# → android/jniLibs/arm64-v8a/libxionia.so

# Cross-compile para Raspberry Pi (arm64)
GOOS=linux GOARCH=arm64 go build -o mesh-arm64 ./cmd/mesh/
```

### Flags de build (faro)

```bash
go build -ldflags "-X main.buildCommit=$(git rev-parse --short HEAD) \
                    -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
                    -X main.buildVersion=v1.0.0" \
         -o faro ./cmd/faro/
```

---

## 7. Seguridad

### Permisos de archivos

| Archivo | Permisos | Razón |
|---------|----------|-------|
| `node.key` | `0600` | Clave privada Ed25519 + X25519 |
| `acl.json` | `0600` | Lista de peers autorizados |
| `aliases.json` | `0600` | Alias locales |
| `groups/*.json` | `0600` | Grupos cifrados |
| `config.json` | `0644` | Configuración de faro (no sensible) |
| `xionia.log` | `0644` | Log del nodo |

### Anti-DPI

Todo mensaje incluye padding aleatorio (50-200 bytes, `crypto/rand`)
para que el tamaño del paquete no revele el contenido. Un "hola" y un
mensaje de 500 caracteres tienen tamaños similares en el wire.

### Anti-replay

Todo InnerPayload incluye un timestamp (`ts`). El receptor rechaza
mensajes con `|now - ts| > 60s`.

### Anti-spoofing

- **Gate DID**: handshake Ed25519 obligatorio antes de operar.
- **verifyAnnounceSig**: el ANNOUNCE verifica que la firma corresponda
  al DID reclamado (reconstruye la clave pública desde el DID).
- **Firma de mensajes**: todo InnerPayload está firmado con Ed25519.

### Jaula de Faraday

Los comandos de archivo operan exclusivamente dentro de `.xion/`.
No hay `../`, no hay symlinks, no hay escape.

---

## 8. Roadmap

| Fase | Estado | Contenido |
|------|--------|-----------|
| **Fase 1** | 🔒 Congelada (v1.0) | Crypto, ACL, faro dual, shell, jaula, grupos, XionChat |
| **Fase 2** | ❌ Pendiente de diseño | U2P, hosting, navegador, IA colaborativa *(requiere RFCs cerrados)* |
| **Fase 3** | ❌ Visión | XPT (Jami++), audio E2E, Xion Console, Swarm |

---

Última actualización: 30 de Julio de 2026_

