# XionIA - Documentación Técnica (v1.0)

## 1. Arquitectura

XionIA es una red overlay soberana: nodos se conectan a un **Faro Ciego** (relay zero-knowledge) para superar NATs y firewalls.

| Componente | Función | Persistencia |
|:---|:---|:---|
| **Nodo (`./mesh`)** | Cliente. Shell interactiva, cifrado E2E, IA local | `~/.xion/` |
| **Faro (`./faro`)** | Relay ciego. No descifra. No loguea IPs | RAM solo |
| **Jaula de Faraday** | Workspace aislado del host | `~/.xion/workspace/` |

### Transporte
- **UDP 54321:** Nativo, preferido
- **WebSocket 443:** Para firewalls corporativos (TLS)
- **U2P/XTP:** Túneles propios (Noise+Curve25519) para perforar CGNAT

### Faro Ciego
```
ANNOUNCE → DID → IP:Puerto (registry en RAM)
RELAY    → Reenvía payload cifrado, ACK al emisor
RESPONSE → Reenvía a lastClient
WHERE_IS → READY / NOT_FOUND
```
El faro **nunca** ve contenido, metadata de quién habla con quién, ni IPs completas en logs.

---

## 2. Stack Criptográfico

| Propósito | Algoritmo |
|:---|:---|
| Identidad + Firmas | Ed25519 |
| Intercambio de claves | X25519 (ECDH) |
| Cifrado simétrico | ChaCha20-Poly1305 (AEAD) |
| Handshake | Noise Protocol IK |
| Anti-DPI | Padding con `crypto/rand` (50-200 bytes) |

---

## 3. Integridad Verificable

Todos los binarios se verifican contra GitHub releases:

```bash
$ mesh verify
🔐 Hash: sha256:a3f7c2... (12.4 MB)
✅ Coincide con github.com/xionia/web5-mesh/releases/v1.0.1

$ faro verify 190.220.45.26:54321
🔐 Hash: sha256:e5d9a1... (4.2 MB)
✅ Faro auditado. Conexión permitida.
```

El faro expone endpoint `VERIFY_HASH`: responde con hash SHA256, tamaño, commit y firma Ed25519 del release.

---

## 4. Directorios del Nodo

```
~/.xion/
├── node.key          # Clave privada (0600)
├── acl.json          # Nodos de confianza (0600)
├── aliases.json      # Alias locales (0600)
├── config.json       # Config del faro (0600)
├── workspace/        # Archivos de usuario
│   └── archivo.pdf
└── ia/               # Modelos GGUF + embeddings
    └── llama-3.2-3b-instruct.gguf
```

---

## 5. IA Colaborativa

Cada nodo corre **llama.cpp** local. Los agentes participan en grupos cifrados E2E como peers autónomos.

```bash
$ ia start ia-soberania --model llama-3.2-3b.gguf
$ ia list          # Ver servicios en el faro (como /list IRC)
$ ia use <did>     # Usar inferencia de otro nodo (E2E)
$ ia offer         # Publicar tu inferencia
```

Mercado libre: gratis, trueque, o sats. Sin intermediarios.

---

## 6. Comandos Core

| Comando | Función |
|:---|:---|
| `whoami` | DID, claves públicas |
| `acl import/add/remove/list` | Gestión de confianza |
| `alias add/remove/list` | Alias locales |
| `chat <did/alias> <msg>` | Mensaje E2E |
| `group create/add/send/delete` | Grupos cifrados |
| `faro set/reset/status` | Configuración del faro |
| `import/export` | Mover archivos host ↔ jaula |
| `sign/verify` | Firma Ed25519 |
| `clear --force` | Wipe total de identidad |
| `mesh verify` | Verificar hash del binario |

---

## 7. Compilación

```bash
# Faro
go build -trimpath -ldflags="-s -w" -o faro ./cmd/faro

# Nodo
go build -trimpath -ldflags="-s -w" -o mesh ./cmd/mesh

# Cross-compile
GOOS=linux GOARCH=arm64 go build -o mesh-arm64 ./cmd/mesh
```

---

## 8. Roadmap

| Fase | Estado | Hitos |
|:---|:---|:---|
| **Fase 1** | ✅ Cerrada | Faro dual, E2E, shell, ACL, Jaula |
| **Fase 2** | 🚧 En curso | U2P, hosting, navegador, IA colaborativa |
| **Fase 3** | 🔵 Plan | DHT, sharding, IA federada |

Ver [PHASES.md](PHASES.md) para detalle completo.

---

*XionIA Faraday — Sovereign overlay network. E2E encryption. Faraday Cage isolation. NAT traversal.*
*Última actualización: 17 de Julio de 2026*
