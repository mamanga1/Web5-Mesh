# 🗺️ Roadmap & Development Phases

**XionIA Faraday** — Sovereign overlay network with blind relay, E2E encryption (Ed25519/X25519/ChaCha20-Poly1305), Faraday Cage isolation, and NAT traversal. Runs on Xeon servers, Raspberry Pi, and TV boxes.

---

## ✅ FASE 1 — Fundación Criptográfica (v1.0) COMPLETADA

**Fecha de cierre:** 22 de Julio de 2026  
**Estado:** Funcional en producción

### Features implementadas y verificadas:

| Feature | Estado |
|---------|--------|
| Identidad descentralizada (DID `did:maia:*`) | ✅ |
| Criptografía híbrida Ed25519 + X25519 | ✅ |
| ACL con gestión de confianza | ✅ |
| Cifrado E2E ChaCha20-Poly1305 | ✅ |
| Faro dual UDP 54321 + WebSocket 443 | ✅ |
| Shell interactiva (go-prompt) | ✅ |
| Comandos: whoami, acl, alias, chat, group | ✅ |
| Alias locales | ✅ |
| Grupos cifrados | ✅ |
| Jaula de Faraday (`.xion/`) | ✅ |
| Comandos Unix en jaula (ls, cat, rm, mv, cp, mkdir, touch, edit) | ✅ |
| import/export/sign/verify | ✅ |
| `clear --force` (wipe total) | ✅ |
| Compilación cross-arch (amd64, arm64) | ✅ |
| Padding anti-DPI con `crypto/rand` | ✅ |
| Permisos `0600` en archivos sensibles | ✅ |
| `faro set <addr>` / `faro reset` | ✅ |
| **Roaming automático (cambio de IP)** | ✅ |
| **Verificación de faros remotos (VERIFY_HASH)** | ✅ |

### Entregables técnicos:

- `src/crypto/` — Identidad, cifrado, ACL, alias, grupos
- `cmd/faro/main.go` — Faro dual UDP/WebSocket
- `cmd/mesh/main.go` — Nodo cliente
- `cmd/mesh/shell.go` — Shell interactiva
- `cmd/mesh/commands/` — Todos los comandos

### Fixes aplicados (v1.0.1):

| Fix | Descripción |
|-----|-------------|
| `math/rand` → `crypto/rand` | Padding criptográficamente seguro |
| Permisos `0644` → `0600` | Archivos sensibles protegidos |
| Roaming | Detección de cambio de IP + reconexión |
| ACK_IP | Faro responde con IP pública |

---

## 🚧 FASE 2 — Herramienta Soberana (v2.0) EN DESARROLLO

**Fecha objetivo:** Q4 2026  
**Estado:** Diseño y planificación

### Features planificadas:

| Feature | Estado | Descripción |
|---------|--------|-------------|
| **Navegador Xion desde shell** | 📋 Plan | `browse <xion://did:maia:.../path>` |
| **Hosting soberano** | 📋 Plan | `host start <puerto>` — Servir contenido desde la Jaula de Faraday |
| **U2P / XTP (túneles CGNAT)** | 🚧 PoC | Protocolo propio de túneles (Noise+Curve25519) |
| **Federación de faros** | 📋 Plan | Cualquiera puede levantar un faro sin IP pública |
| **IA Colaborativa Distribuida** | 🚧 PoC | Agentes autónomos con llama.cpp local + mercado libre de inferencia entre nodos |
| **Proxy SOCKS5 sobre U2P** | 📋 Plan | `proxy start` — Tunelar tráfico TCP a través de la red XionIA |
| **Rendezvous automático** | 📋 Plan | Descubrimiento de peers sin faro central |
| **Multi-faro failover** | 📋 Plan | Conmutación automática entre faros si uno cae |

### Entregables técnicos planificados:

| Módulo | Archivo | Estado |
|--------|---------|--------|
| U2P Tunnel | `src/xtp/u2p_tunnel.go` | 🚧 PoC |
| U2P Handshake Noise | `src/xtp/noise_handshake.go` | 🚧 PoC |
| Navegador Xion | `cmd/mesh/commands/browse.go` | 📋 Plan |
| Hosting soberano | `cmd/mesh/commands/host.go` | 📋 Plan |
| IA: Wrapper llama.cpp | `src/ia/llama_cpp.go` | 🚧 PoC |
| IA: Mercado de inferencia | `src/ia/inference_market.go` | 🚧 PoC |
| Proxy SOCKS5 | `cmd/mesh/commands/proxy.go` | 📋 Plan |
| Verificación de faros | `cmd/mesh/commands/faro.go` | 📋 Plan |

---

## 💰 Solicitud de Financiamiento – Fase 2

Buscamos **$62,000 USD** para financiar los próximos 12 meses de desarrollo full-time de la Fase 2.

### Desglose detallado del presupuesto:

| Categoría | Monto (USD) | % | Detalle |
|:----------|------------:|--:|:--------|
| **Desarrollo Full-Time (2 devs)** | $36,000 | 58% | Salarios + equipo core |
| **Hardware & Testing en Campo** | $9,000 | 15% | TV Boxes, Raspberry, movilidad NEA |
| **Auditorías de Seguridad** | $8,000 | 13% | Revisión externa cripto y protocolo |
| **Hosting / Servidores semilla** | $4,000 | 6% | Faros públicos y monitoreo |
| **Legal y misceláneos** | $5,000 | 8% | Estructura legal + gastos operativos |

**Total:** $62,000 USD

### Uso de los fondos:
Este financiamiento permitirá pasar de un PoC funcional a una plataforma madura y estable para adopción masiva en entornos reales de baja conectividad.

---


## 🏢 FASE 3 — XionIA Faraday Corporativo (v3.0)

**Fecha objetivo:** 2027  
**Estado:** Diseño arquitectónico

### Visión

**XionIA XPT** (Xion Privacy Transport) — Un Jami+++ sobre la red XionIA.  
Audio cifrado E2E, sin servidores de señalización, sin STUN/TURN, sin exposición de IP.  
Capacidad de evasión de triangulación de torres celulares.  
Totalmente privado e indetectable.

### Features planificadas:

| Feature | Descripción |
|---------|-------------|
| **Jami+++ sobre XionIA** | Llamadas de audio cifradas E2E sobre el protocolo XionIA |
| **XPT (Xion Privacy Transport)** | Protocolo de transporte diseñado para evadir triangulación de torres |
| **Signalización sin servidor** | Descubrimiento de pares vía DHT, sin servidores centrales |
| **Audio cifrado E2E** | Opus + ChaCha20-Poly1305 sobre el túnel XionIA |
| **Evasión de triangulación** | Paquetes con timings variables, enmascaramiento de tráfico, rutas aleatorias |
| **Indetectable** | Tráfico camuflado como tráfico web normal (HTTPS/WebSocket) |
| **Sin exposición de IP** | Todo el tráfico pasa por la red XionIA, nunca se expone la IP real |

### Arquitectura XPT:

```
┌─────────────────────────────────────────────────────────────┐
│                    XIONIA XPT (v3.0)                        │
│              "Jami+++ sobre la red soberana"                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐         ┌─────────────┐                   │
│  │  Nodo A     │────────►│  Faro 1     │                   │
│  │  (Llamada   │  XPT    │  (Relay)    │                   │
│  │   entrante) │◄────────│             │                   │
│  └─────────────┘         └─────────────┘                   │
│       │                          │                          │
│       │                          │                          │
│       ▼                          ▼                          │
│  ┌─────────────┐         ┌─────────────┐                   │
│  │  Nodo B     │────────►│  Faro 2     │                   │
│  │  (Llamada   │  XPT    │  (Relay)    │                   │
│  │   saliente) │◄────────│             │                   │
│  └─────────────┘         └─────────────┘                   │
│                                                             │
│  ● Audio: Opus (códec) + ChaCha20-Poly1305                 │
│  ● Transporte: XPT sobre UDP/WebSocket                     │
│  ● Señalización: DHT + DID (sin servidores)                │
│  ● Anti-triangulación: Timings variables + routing aleatorio│
│  ● Indetectable: Tráfico camuflado como HTTPS              │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```
---

## 📊 Métricas de Progreso

| Fase | Fecha | Features |
|------|-------|----------|
| v1.0 Fase 1 | Jul 2026 | ✅ 20 |
| v2.0 Fase 2 | Q4 2026 | 🚧 10+ |
| v3.0 Fase 3 | 2027 | 📋 5+ |

