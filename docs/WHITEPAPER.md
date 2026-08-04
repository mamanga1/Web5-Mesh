# 📜 XionIA Whitepaper: Sovereign Overlay Network

## Abstract

XionIA Faraday es una red overlay peer-to-peer soberana que opera sobre
la infraestructura de internet existente sin depender de servidores
centrales, DNS, certificados CA, ni servicios de terceros. Basada en
**relays ciegos zero-knowledge** y **criptografía E2E**
(Ed25519/X25519/ChaCha20-Poly1305), con **túneles U2P (User-to-Peer)**
como objetivo de Fase 2.

XionIA Faraday está diseñada para funcionar en hardware heterogéneo:
desde servidores Xeon hasta Raspberry Pi, TV boxes Android, y celulares.
La soberanía no es un lujo de datacenter; es un derecho de cada nodo.

---

## 1. Motivación

### El problema de la centralización

Las redes P2P existentes (BitTorrent, IPFS, Matrix, Nostr) dependen de
infraestructura centralizada en mayor o menor grado:

- **BitTorrent**: depende de trackers y DHT bootstrap nodes
- **IPFS**: depende de gateways y bootstrap nodes
- **Matrix**: depende de homeservers federados
- **Nostr**: depende de relays centralizados

Todas requieren DNS, certificados CA, y en la práctica, servidores
operados por organizaciones con jurisdicción legal.

### La soberanía como requisito

XionIA Faraday parte de una premisa diferente: **la red debe funcionar
aunque internet "oficial" se caiga, se censure, o se vigile**. Cada nodo
es soberano: genera su propia identidad, gestiona su propia confianza,
y opera sin permisos de terceros.

---

## 2. Arquitectura

### A. Identidad Descentralizada (DID)

Cada nodo genera un par de claves Ed25519 (firma) y X25519 (intercambio)
en el primer arranque. La identidad es un DID:

```
did:maia:<base58(PubKeyEd25519)>
```

No hay registro central. No hay CA. No hay DNS. El DID **es** la
identidad. Quien controla la clave privada, controla el DID.

### B. Relay Ciego (Faro)

El Faro es un relay zero-knowledge:

- **No descifra**: solo ve paquetes cifrados
- **No almacena**: registry en RAM, se pierde al reiniciar
- **No inspecciona**: no hay logging de contenido
- **No autentica contenido**: solo verifica que el nodo pasó el Gate DID
- **Gate DID**: autenticación Ed25519 obligatoria (handshake antes de operar)
- **Cross-relay UDP↔WSS**: un nodo en UDP puede comunicarse con uno en WSS
- **TTL registry**: expira nodos zombies a 90s sin ANNOUNCE
- **Verificación de firma del ANNOUNCE**: anti-spoofing de identidad

El faro es reemplazable: cualquier nodo puede correr un faro. La red
no depende de ningún faro específico.

### C. Soberanía en el Edge

XionIA Faraday está diseñada para hardware de bajo costo:

| Hardware | CPU | RAM | Uso |
|----------|-----|-----|-----|
| Raspberry Pi 4 | ARM Cortex-A72 | 1-8 GB | Nodo completo + faro |
| TV Box Android | ARM Cortex-A53 | 2-4 GB | Nodo móvil |
| Celular Android | ARM Cortex-A76 | 4-8 GB | Nodo móvil (XionChat) |
| Servidor Xeon | x86-64 | 16+ GB | Faro público + IA |

La compilación cross-arch (`build.sh`) genera binarios para amd64 y
arm64. El mismo código corre en todos.

### D. Túneles U2P (User-to-Peer)

> ⚠️ **Estado: Fase 2 — diseño pendiente.** U2P no está implementado.
> `src/crypto/noise.go` contiene el protocolo Noise IK completo y
> verificado, pero NO está conectado al transporte. Se conectará en
> Fase 2 con el TransportManager y el DirectTransport.

U2P es el protocolo de túneles propio de XionIA para perforar CGNAT
(Carrier-Grade NAT), el obstáculo principal para la conectividad P2P
en redes móviles y residenciales.

```
Nodo A (detrás de CGNAT) ←→ Nodo B (detrás de CGNAT)
         │                         │
         └─────── U2P Tunnel ──────┘
              (Noise IK + UDP)
```

U2P usa hole punching UDP asistido por el faro (que actúa como
rendezvous), con handshake Noise IK para autenticación y cifrado
del túnel.

*(Fase 2 — no implementado)*

### E. IA Colaborativa Distribuida

> ⚠️ **Estado: Fase 3+ — visión, sin diseño cerrado.** No existe código
> de IA en el repositorio. Se incluirá cuando la Fase 2 esté cerrada.

XionIA Faraday integra inferencia de IA local (llama.cpp) con un mercado
P2P de capacidad de cómputo:

- **Local**: cada nodo puede correr modelos GGUF (1-3B parámetros)
- **Distribuido**: nodos con GPU ofrecen inferencia a la red
- **Soberano**: sin APIs externas, sin dependencia de OpenAI/Anthropic/Google

*(Fase 3+ — visión, sin código)*

### F. XionChat: Cliente de Referencia

XionChat (repo: `xionia-xtp`) es el cliente Android de referencia para la
Fase 1. Binding Go→Flutter con notificaciones en background, watchdog de
reconexión, y Gate DID. Funciona en Android 12+ (probado en TCL, Samsung,
Poco/LineageOS).

---

## 3. Comparativa con redes existentes

| Feature | BitTorrent | IPFS | Matrix | Nostr | **XionIA** |
|---------|-----------|------|--------|-------|------------|
| Identidad | ❌ Hash | ❌ Hash | ✅ Usuario | ✅ Clave | **✅ DID soberano** |
| Cifrado E2E | ❌ No | ❌ No | ⚠️ Opcional | ❌ No | **✅ Siempre** |
| Relay ciego | ⚠️ Tracker | ⚠️ Gateway | ❌ Server | ⚠️ Relay | **✅ Faro** |
| CGNAT | ❌ No | ❌ No | ❌ No | ❌ No | **📋 Fase 2 (U2P, diseño pendiente)** |
| IA distribuida | ❌ No | ❌ No | ❌ No | ❌ No | **📋 Fase 3+ (visión)** |
| Funciona sin DNS | ❌ No | ❌ No | ❌ No | ⚠️ Parcial | **✅ Sí** |
| Hardware mínimo | ⚠️ PC | ⚠️ PC | ⚠️ Server | ⚠️ PC | **✅ Raspberry Pi / celular** |

---

## 4. Modelo de Confianza: ACL

XionIA Faraday usa un modelo de confianza explícito: **solo te comunicás
con quien vos autorizás**.

```
acl import <DID> <PubKeyEd> <PubKeyX>   # Autorizar un peer
acl remove <DID>                         # Revocar autorización
acl list                                 # Ver peers autorizados
```

No hay "amigos de amigos". No hay descubrimiento automático. No hay
spam. Cada relación de confianza es explícita y revocable.

### Key ID (KID)

Para evitar enviar la clave pública completa en cada mensaje, se usa un
Key ID de 4 bytes:

```
KID = SHA-256(PubKeyX)[:4]
```

El receptor busca el KID en su ACL para encontrar la clave compartida.
Colisión de KID: ~1 en 4 mil millones por par de claves.

---

## 5. Stack Técnico

| Componente | Tecnología |
|------------|-----------|
| Lenguaje | Go 1.21+ |
| Firma | Ed25519 (`crypto/ed25519`) |
| Intercambio | X25519 (`golang.org/x/crypto/curve25519`) |
| Cifrado | ChaCha20-Poly1305 (`golang.org/x/crypto/chacha20poly1305`) |
| Handshake | Noise Protocol IK *(implementado, NO conectado — Fase 2)* |
| Shell | `go-prompt` (autocompletado, historial) |
| WebSocket | `gorilla/websocket` |
| Transporte | UDP 54321 (default) + WSS 443 (fallback) |
| IA | llama.cpp (GGUF) *(Fase 3+ — visión, sin código)* |
| Cliente Android | Flutter + FFI/cgo (XionChat, repo xionia-xtp) |

---

## 6. Despliegue

### Nodo mínimo (Raspberry Pi)

```bash
# Compilar
GOOS=linux GOARCH=arm64 go build -o mesh ./cmd/mesh/

# Configurar faro
./mesh
xion@nodo:~$ faro set 190.220.45.26:54321

# Listo
xion@nodo:~$ whoami
DID: did:maia:BXCUEnU6Dbh...
```

### Faro público

```bash
# Compilar con metadata
go build -ldflags "-X main.buildVersion=v1.0.0" -o faro ./cmd/faro/

# Generar certs para WSS
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem \
  -days 365 -nodes -subj "/CN=faro.xionia.org"

# Correr
./faro
# 🛡️ [FARO-UDP] Relay Ciego en 0.0.0.0:54321 (Gate DID activo)
# 🛡️ [FARO-WS] WebSocket TLS en 0.0.0.0:443 (Gate DID activo)
```

### XionChat (Android)

```bash
cd xionia-xtp
./build.sh android
cp android/jniLibs/arm64-v8a/libxionia.so \
   xionchat_flutter/android/app/src/main/jniLibs/arm64-v8a/
cd xionchat_flutter
flutter build apk --release
```

---

## 7. Limitaciones conocidas de la Fase 1

| Limitación | Causa | Solución |
|------------|-------|----------|
| El faro relayea TODOS los datos | Arquitectura de Fase 1 (relay ciego) | Fase 2: faro → signaling puro |
| Sin conexión directa P2P | No implementado | Fase 2: DirectTransport (U2P) |
| Sin forward secrecy | Noise IK no conectado | Fase 2: SessionManager + Noise IK |
| Gate DID usa IP:puerto como clave | Diseño del Gate | Fase 2: Gate por DID |
| Sin cola de mensajes | El faro no almacena | Fase 2+: Event Ledger |

---

## 8. Roadmap

| Fase | Estado | Contenido |
|------|--------|-----------|
| **Fase 1** | 🔒 Congelada (v1.0.1) | Crypto, ACL, faro dual, shell, jaula, grupos, XionChat |
| **Fase 2** | ❌ Pendiente de diseño | 🔄 En desarrollo en xionia-kernel (próximamente público). Sprint 1+2 completados: TransportManager, Noise IK, DirectTransport, Faro signaling puro — 95 tests. |
| **Fase 3** | ❌ Visión | audio E2E, Xion Console, Swarm, etc |

---

_XionIA Faraday — La red que no le pide permiso a nadie._

_Última actualización: 4 de Agosto de 2026
