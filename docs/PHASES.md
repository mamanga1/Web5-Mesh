# 🗺️ Roadmap & Development Phases

**XionIA Faraday** — Red overlay soberana con relay ciego, cifrado E2E
(Ed25519/X25519/ChaCha20-Poly1305), Jaula de Faraday, y Gate DID.
Corre en servidores Xeon, Raspberry Pi, TV boxes y celulares Android.

---

## ✅ FASE 1 — Fundación Criptográfica (v1.0) CONGELADA

**Fecha de cierre:** 29 de Julio de 2026
**Estado:** Funcional en producción. Tag v1.0.0.
**Repos congelados:**
- Web5-Mesh (faro + shell + crypto): https://github.com/mamanga1/Web5-Mesh
- xionia-xtp (XionChat, cliente Android de referencia): https://github.com/mamanga1/xionia-xtp

### Features implementadas y verificadas en el código:

| Feature | Estado | Archivo(s) |
|---------|--------|------------|
| Identidad descentralizada (DID `did:maia:*`) | ✅ | `src/crypto/identity.go` |
| Criptografía híbrida Ed25519 + X25519 | ✅ | `src/crypto/identity.go`, `cipher.go` |
| ACL con gestión de confianza | ✅ | `src/crypto/acl.go` |
| Cifrado E2E ChaCha20-Poly1305 | ✅ | `src/crypto/cipher.go` |
| Gate DID (handshake + autorización por DID) | ✅ | `src/crypto/gate.go`, `cmd/faro/main.go` |
| Faro dual UDP 54321 + WebSocket 443 | ✅ | `cmd/faro/main.go` |
| Cross-relay UDP↔WSS (un nodo UDP habla con uno WSS) | ✅ | `cmd/faro/main.go` (fix #1) |
| TTL en registry UDP (expira nodos zombies a 90s) | ✅ | `cmd/faro/main.go` (fix #2) |
| Verificación de firma del ANNOUNCE (anti-spoofing) | ✅ | `cmd/faro/main.go` (fix A) |
| Shell interactiva (go-prompt) | ✅ | `cmd/mesh/shell.go` |
| Comandos: whoami, acl, alias, chat, group, faro | ✅ | `cmd/mesh/commands/` |
| Alias locales | ✅ | `src/crypto/aliases.go` |
| Grupos cifrados | ✅ | `src/crypto/groups.go` |
| Jaula de Faraday (`.xion/`) | ✅ | `cmd/mesh/commands/` |
| Comandos Unix en jaula (ls, cat, rm, mv, cp, mkdir, touch, edit) | ✅ | `cmd/mesh/commands/` |
| import/export/sign/verify | ✅ | `cmd/mesh/commands/` |
| `clear --force` (wipe total) | ✅ | `cmd/mesh/commands/`, `mobile.go` |
| Compilación cross-arch (amd64, arm64) | ✅ | `build.sh` |
| Padding anti-DPI con `crypto/rand` | ✅ | `shell.go`, `mobile.go` |
| Permisos `0600` en archivos sensibles | ✅ | `src/crypto/` |
| `faro set <addr>` / `faro reset` | ✅ | `src/config/`, `cmd/mesh/commands/` |
| Roaming automático (cambio de IP vía ACK_IP) | ✅ | `shell.go`, `mobile.go` |
| Verificación de faros remotos (VERIFY_HASH) | ✅ | `cmd/faro/main.go` |
| Watchdog de reconexión (20s sin actividad → reconecta) | ✅ | `shell.go`, `mobile.go` |
| ANNOUNCE con retry a 3s (fix "hay que saludarse") | ✅ | `shell.go`, `mobile.go` |
| XionChat: cliente Android con notificaciones en background | ✅ | `xionia-xtp/` (repo separado) |

### Fixes aplicados sobre v1.0 :

| Fix | Descripción | Archivo |
|-----|-------------|---------|
| `math/rand` → `crypto/rand` | Padding criptográficamente seguro | `shell.go`, `mobile.go` |
| Permisos `0644` → `0600` | Archivos sensibles protegidos | `src/crypto/` |
| Roaming (ACK_IP) | Detección de cambio de IP + re-ANNOUNCE | `shell.go`, `mobile.go` |
| Cross-relay UDP↔WSS | RELAY busca en ambos registries (antes separados) | `cmd/faro/main.go` |
| TTL registry (90s) | Expira nodos zombies (antes eternos) | `cmd/faro/main.go` |
| verifyAnnounceSig | Verifica firma del ANNOUNCE (anti-spoofing de DID) | `cmd/faro/main.go` |
| Crash por cert faltante | WSS no mata el proceso UDP si faltan certs | `cmd/faro/main.go` |
| ACK falso en WSS | No manda ACK si la entrega WSS falla + limpia conn muerta | `cmd/faro/main.go` |
| stripPadding en ANNOUNCE | Saca padding antes de verificar firma (base64 fallaba) | `cmd/faro/main.go` |
| Gate logging | Loguea cuando el Gate rechaza (antes en silencio) | `cmd/faro/main.go` |
| Watchdog 20s + ANNOUNCE 10s + retry 3s | Reconexión más rápida, registro más frecuente | `shell.go`, `mobile.go` |
| connMu / quitMu / recover() | Anti-race-condition y anti-panic en mobile.go | `mobile.go` |

### Limitaciones conocidas de la Fase 1 (documentadas, no son bugs):

| Limitación | Causa | Solución |
|------------|-------|----------|
| El faro relayea TODOS los datos (cuello de botella) | Arquitectura de Fase 1 (relay ciego) | Fase 2: faro → signaling puro |
| Sin conexión directa P2P (hole punching) | No implementado | Fase 2: DirectTransport |
| Sin forward secrecy (ECDH estático + ChaCha20) | Noise IK existe en `src/crypto/noise.go` pero NO está conectado | Fase 2: SessionManager + Noise IK |
| Gate DID usa IP:puerto como clave (se rompe con NAT/roaming) | Diseño del Gate | Fase 2: Gate por DID |
| Sin cola de mensajes (store-and-forward) | El faro no almacena mensajes | Fase 2+: Event Ledger |
| XionChat: wakelock permanente consume batería | Contrapartida de no tener FCM/push | Pendiente: wakelock adaptativo |
| XionChat: TCL requiere exclusión manual del limpiador | OneKeyCleanService de TCL hace force stop | Documentar en README |

### Entregables técnicos de la Fase 1:

**Web5-Mesh:**
- `src/crypto/` — Identidad, cifrado, ACL, alias, grupos, Gate DID, Noise IK (no conectado)
- `src/config/` — Configuración de faro
- `cmd/faro/main.go` — Faro dual UDP/WebSocket con Gate DID, cross-relay, TTL
- `cmd/mesh/main.go` — Nodo cliente (entry point)
- `cmd/mesh/shell.go` — Shell interactiva con watchdog y ANNOUNCE
- `cmd/mesh/commands/` — Todos los comandos (whoami, acl, alias, chat, group, faro, etc.)
- `build.sh` — Compilación cross-arch

**xionia-xtp (XionChat):**
- `mobile.go` — Binding Go→Flutter (FFI/cgo) con watchdog, connMu, recover()
- `flutter_bridge.dart` — Clase Xionia (Dart FFI)
- `lib/main.dart` — UI Flutter con background service
- `XionApplication.kt` — Wakelock (⚠️ permanente, fix pendiente)
- `AndroidManifest.xml` — Foreground Service, permisos

---

## 🔒 FASE 2 — Kernel de Red Soberano (v2.0) PENDIENTE DE DISEÑO

**Fecha objetivo:** A definir después de congelar Fase 1.
**Estado:** NO iniciada. Requiere RFCs cerrados antes de escribir código.
**Repo:** Se abrirá un nuevo repositorio (`xionia-kernel`) o una rama `v2`
en Web5-Mesh. La Fase 1 queda congelada como referencia histórica.

### Objetivo:
Eliminar al faro del plano de datos. El faro pasa de relay a signaling puro
(registry + punch coordinator). Los nodos se hablan directo (Noise IK + UDP).

### Lo que NO está implementado (todo lo de esta fase es diseño pendiente):

| Feature | Estado real |
|---------|-------------|
| TransportManager (FSM del transporte) | ❌ No existe código |
| DirectTransport (Hole Punching + Noise IK) | ❌ No existe código (`src/crypto/noise.go` existe pero NO está conectado) |
| RelayTransport (fallback) | ❌ No existe como módulo separado |
| SessionManager (envuelve Noise) | ❌ No existe código |
| Faro como signaling puro (OPEN_SESSION, Punch Coordinator) | ❌ No existe código |
| Service Discovery | ❌ No existe código |
| Federaciones | ❌ No existe código |
| Event Ledger | ❌ No existe código |
| Navegador Xion (`browse`) | ❌ No existe código |
| Hosting soberano (`host`) | ❌ No existe código |
| Proxy SOCKS5 sobre U2P | ❌ No existe código |
| IA Colaborativa Distribuida | ❌ No existe código |
| Multi-faro failover | ❌ No existe código |

> **Nota:** El PHASES.md anterior listaba U2P/XTP y IA como "🚧 PoC".
> Eso era incorrecto: los archivos `src/xtp/` y `src/ia/` NO existen
> en el repositorio. Se corrige a ❌ No existe código.

### RFCs requeridos antes de escribir código:

| RFC | Título | Estado |
|-----|--------|--------|
| RFC-0001 | Arquitectura del Kernel | ❌ Pendiente |
| RFC-0002 | XTP Engine | ❌ Pendiente |
| RFC-0003 | FSM del Transporte | ❌ Pendiente |
| RFC-0004 | SessionManager (Noise IK) | ❌ Pendiente |
| RFC-0005 | Protocolo del Faro (Signaling) | ❌ Pendiente |
| RFC-0006 | DirectTransport (Hole Punching) | ❌ Pendiente |
| RFC-0007 | RelayTransport (fallback) | ❌ Pendiente |
| RFC-0008 | Event Ledger | ❌ Pendiente |
| RFC-0009 | Federation Manager | ❌ Pendiente |

---

## 🔒 FASE 3+ — Visiones futuras PENDIENTES DE DISEÑO

**Estado:** NO iniciadas. Son visión arquitectónica, no código.
No se detallan features porque requieren que la Fase 2 esté cerrada primero.

| Visión | Estado |
|--------|--------|
| XPT (Xion Privacy Transport) — Jami+++ sobre XionIA | ❌ Visión, sin diseño |
| Audio cifrado E2E (Opus + ChaCha20) | ❌ Visión, sin diseño |
| Xion Console (no la shell, la Console) | ❌ Visión, sin diseño |
| Swarm | ❌ Visión, sin diseño |
| IA distribuida (llama.cpp local + mercado de inferencia) | ❌ Visión, sin diseño |
| LoRa Gateway | ❌ Visión, sin diseño |
| MAIA | ❌ Visión, sin diseño |

---

## 📦 Estrategia de repositorios

| Repo | Contenido | Estado |
|------|-----------|--------|
| **Web5-Mesh** | Faro + shell + crypto (Fase 1) | 🔒 Congelado en v1.0. Solo bugs críticos. |
| **xionia-xtp** | XionChat, cliente Android de referencia (Fase 1) | 🔒 Congelado en v1.0. Solo bugs críticos. |
| **xionia-kernel** (nuevo) | Kernel XionIA: TransportManager, FSM, DirectTransport, SessionManager, Faro signaling (Fase 2) | ❌ No creado. Se abre cuando la Fase 1 esté taggeada y los RFCs cerrados. |

La Fase 1 queda como **referencia histórica**. No se desarrolla sobre ella.
La Fase 2 nace en un repo nuevo (o rama `v2`) con arquitectura limpia,
guiada por RFCs. Web5-Mesh y xionia-xtp se mantienen solo para bugs
críticos de la 1.0.

---

## 📊 Métricas de Progreso

| Fase | Fecha | Features reales | Estado |
|------|-------|-----------------|--------|
| v1.0 Fase 1 | Jul 2026 | 25 features verificadas en código | 🔒 Congelada |
| v2.0 Fase 2 | A definir | 0 features (requiere RFCs) | ❌ No iniciada |
| v3.0+ Fase 3 | A definir | 0 features (visión) | ❌ No iniciada |

---

## 💰 Solicitud de Financiamiento – Fase 2

Buscamos **$62,000 USD** para financiar los próximos 12 meses de
desarrollo full-time de la Fase 2.

| Categoría | Monto (USD) | % | Detalle |
|:----------|------------:|--:|:--------|
| **Desarrollo Full-Time (2 devs)** | $36,000 | 58% | Salarios + equipo core |
| **Hardware & Testing en Campo** | $9,000 | 15% | TV Boxes, Raspberry, movilidad NEA |
| **Auditorías de Seguridad** | $8,000 | 13% | Revisión externa cripto y protocolo |
| **Hosting / Servidores semilla** | $4,000 | 6% | Faros públicos y monitoreo |
| **Legal y misceláneos** | $5,000 | 8% | Estructura legal + gastos operativos |

**Total:** $62,000 USD

---

<div align="center">

*Hecho con orgullo, código y aguante desde Corrientes, Argentina.* 

</div>
```
