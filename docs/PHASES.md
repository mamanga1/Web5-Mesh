# 🗺️ Roadmap & Development Phases

**XionIA Faraday** — Sovereign overlay network with blind relay, E2E encryption (Ed25519/X25519/ChaCha20-Poly1305), Faraday Cage isolation, and NAT traversal. Runs on Xeon servers, Raspberry Pi, and TV boxes.

XionIA sigue un modelo **Open Core**: el núcleo y funcionalidades base son 100% open source para fomentar adopción masiva, auditoría comunitaria y soberanía digital. Funcionalidades avanzadas, enterprise y anti-vigilancia se desarrollan en repositorio privado bajo licencia comercial.

---

## ✅ FASE 1 — Fundación Criptográfica (v1.0) COMPLETADA

**Fecha de cierre:** 17 de Julio de 2026
**Estado:** Funcional en producción, auditado internamente, probado en campo.

### Features implementadas y verificadas:

| Feature | Estado | Verificación |
|---------|--------|--------------|
| Identidad descentralizada (DID `did:maia:*`) | ✅ | `whoami` funcional |
| Criptografía híbrida Ed25519 + X25519 | ✅ | Chat E2E verificado |
| ACL con gestión de confianza | ✅ | `acl import/list/remove` |
| Cifrado E2E ChaCha20-Poly1305 | ✅ | Payloads cifrados en logs del faro |
| Faro dual UDP 54321 + WebSocket 443 | ✅ | Producción: 190.220.45.26 |
| Shell interactiva (go-prompt) | ✅ | `mesh shell` funcional |
| Comandos base: whoami, acl, alias, chat, group | ✅ | Todos verificados |
| Alias locales | ✅ | `alias add/list/remove` |
| Grupos cifrados | ✅ | `group create/add/send/delete` |
| Jaula de Faraday (directorio relativo `.xion/`) | ✅ | Portable, Windows/Linux/macOS/Termux |
| Comandos Unix en jaula (ls, cat, rm, mv, cp, mkdir, touch, edit) | ✅ | Todos funcionan |
| import/export/sign/verify | ✅ | Verificados |
| `clear --force` (wipe total) | ✅ | Identidad descartable |
| Compilación cross-arch (amd64, arm64, arm) | ✅ | `build.sh` funcional |
| Padding anti-DPI con `crypto/rand` | ✅ | Fix v1.0.1 aplicado |
| Permisos `0600` en archivos sensibles | ✅ | Fix v1.0.1 aplicado |
| `faro set <addr>` / `faro reset` | ✅ | Conexión a faros alternativos |

### Entregables técnicos:

- `src/crypto/identity.go` — Gestión de identidad
- `src/crypto/cipher.go` — ChaCha20-Poly1305
- `src/crypto/acl.go` — ACL persistente
- `src/crypto/alias.go` — Resolución de nicks
- `src/crypto/group.go` — Grupos cifrados
- `cmd/faro/main.go` — Faro dual UDP/WebSocket
- `cmd/mesh/main.go` — Nodo cliente
- `cmd/mesh/shell.go` — Shell interactiva
- `cmd/mesh/commands/` — Todos los comandos

### Fixes v1.0.1 (Julio 2026)

| Fix | Descripción |
|-----|-------------|
| `math/rand` → `crypto/rand` | Padding criptográficamente seguro |
| Permisos `0644` → `0600` | `acl.json`, `aliases.json`, `config.json` protegidos |

---

## 🚧 FASE 2 — Herramienta Soberana para la Comunidad (v2.0) EN DESARROLLO

**Fecha objetivo:** Q4 2026
**Estado:** Diseño completado. Implementación en progreso. Algunos módulos en PoC.

### Visión

XionIA deja de ser solo un messenger cifrado para convertirse en una **herramienta de soberanía digital completa**: navegación, hosting, IA colaborativa distribuida, almacenamiento distribuido, todo desde la shell, sin depender de DNS, ICANN, ni infraestructura centralizada.

### Features planificadas y estado:

| Feature | Estado | Descripción |
|---------|--------|-------------|
| **Navegador Xion desde shell** | 🚧 PoC | `browse <xion://did:maia:.../path>` — Navegación de contenido soberano sin HTTP/DNS |
| **Hosting soberano** | 🚧 Diseño | `host start <puerto>` — Servir contenido desde la Jaula de Faraday, accesible vía DID |
| **DID-based addressing** | 🚧 Diseño | Los recursos se direccionan por DID, no por IP ni dominio. `xion://did:maia:ABC/contenido` |
| **U2P / XTP (túneles CGNAT)** | 🚧 PoC | Protocolo propio de túneles (Noise+Curve25519) para faros detrás de CGNAT |
| **Federación de faros** | 🚧 Diseño | Cualquiera puede levantar un faro, incluso sin IP pública, conectándose a bootstrap nodes |
| **IA Colaborativa Distribuida** | 🚧 Diseño | Agentes autónomos con llama.cpp local + mercado libre de inferencia entre nodos |
| **Integridad verificable** | 🚧 Diseño | Hash SHA256 de `./mesh` y faros verificables contra GitHub releases |
| **Sincronización de bóveda** | 📋 Plan | `sync push` / `sync pull` — Replicar Jaula de Faraday entre nodos de confianza |
| **Web5-DWN integración** | 📋 Plan | Almacenamiento descentralizado Web5/DWN para backups y mensajería offline |
| **Proxy SOCKS5 sobre U2P** | 📋 Plan | `proxy start` — Tunelar cualquier tráfico TCP a través de la red XionIA |
| **Rendezvous automático** | 📋 Plan | Descubrimiento de peers sin faro central, vía DHT o bootstrap nodes |
| **Multi-faro failover** | 📋 Plan | Conmutación automática entre faros si uno cae |

### Arquitectura Fase 2: La Red Soberana

```
┌─────────────────────────────────────────────────────────────┐
│                    RED XIONIA v2.0                          │
│              "Internet sin DNS, sin ICANN"                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐      U2P Tunnel      ┌─────────────┐      │
│  │  Nodo A     │◄═════════════════════►│  Faro 1     │      │
│  │ (Bootstrap) │   (IP pública real)     │ (CGNAT ok)  │      │
│  │             │                        │             │      │
│  │  xion://    │                        │  xion://    │      │
│  │  did:maia:A │                        │  did:maia:F │      │
│  └─────────────┘                        └─────────────┘      │
│       ▲                                        ▲            │
│       │                                        │            │
│       │    browse did:maia:F/contenido.html    │            │
│       └────────────────────────────────────────┘            │
│                    (sin DNS, sin HTTP)                       │
│                                                             │
│  ┌─────────────┐      U2P Tunnel      ┌─────────────┐      │
│  │  Nodo B     │◄═════════════════════►│  Faro 2     │      │
│  │ (Hosting    │                        │ (Volátil)   │      │
│  │  soberano)  │                        │             │      │
│  │             │                        │             │      │
│  │  host start │                        │  xion://    │      │
│  │  → sirve    │                        │  did:maia:H │      │
│  │    .html    │                        │             │      │
│  └─────────────┘                        └─────────────┘      │
│                                                             │
│  ┌─────────────┐                                          │
│  │  Nodo C     │  IA Colaborativa: llama.cpp local        │
│  │ (Raspberry  │  + mercado libre de inferencia            │
│  │  Pi / TV    │  entre nodos del faro                     │
│  │  Box)       │                                          │
│  │             │  🤖 ia-soberania (llama.cpp local)        │
│  │  ia start   │  🤖 ia-economia (llama.cpp local)         │
│  │  ia list    │  💬 Todos hablan en grupo cifrado         │
│  │  ia use     │  🔌 O usás la inferencia de otro nodo     │
│  └─────────────┘                                          │
│                                                             │
│  CUALQUIERA puede ser faro (incluso detrás de CGNAT)        │
│  CUALQUIERA puede hostear contenido (sin .com, sin DNS)     │
│  CUALQUIERA puede navegar (desde la shell, sin browser)      │
│  CUALQUIERA puede tener IA (local o usando la de un peer)   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Navegador Xion: `browse <url>`

```bash
# Navegar contenido soberano
xion@nodo:~$ browse xion://did:maia:ABC123/contenido.html
📄 Contenido de did:maia:ABC123/contenido.html:
<html>
  <head><title>Bienvenido a mi espacio soberano</title></head>
  <body>
    <h1>Hola, este es mi hosting sin DNS</h1>
    <p>Servido desde la Jaula de Faraday de did:maia:ABC123</p>
  </body>
</html>

# Navegar por alias
xion@nodo:~$ browse xion://amigo/blog/post1.md
📄 blog/post1.md:
# Mi primer post soberano
Escrito sin WordPress, sin hosting pago, sin DNS...
```

### Hosting Soberano: `host start`

```bash
# Servir contenido desde tu Jaula de Faraday
xion@nodo:~$ host start 8080
🌐 Hosting soberano activo en xion://did:maia:TU_DID/
📂 Sirviendo desde: ~/.xion/workspace/public/
📡 Accesible vía cualquier nodo de la red XionIA

# Otro nodo puede ver tu contenido
xion@nodo:~$ browse xion://did:maia:TU_DID/
📁 Índice de did:maia:TU_DID/:
├── 📄 index.html
├── 📁 proyectos/
│   └── 📄 xionia.md
└── 📄 README.md
```

### IA Colaborativa Distribuida: Mercado Libre de Inferencia

Cada nodo puede levantar sus propios agentes IA con **llama.cpp** local. Pero también puede descubrir y usar la inferencia de otros nodos en los faros que frecuenta. Como el `/list` del IRC: ves quién ofrece qué servicio, y negociás libremente — gratis, por trueque, o por pago. Todo E2E cifrado, peer-to-peer, sin intermediarios.

```
┌─────────────────────────────────────────────────────────────┐
│              GRUPO CIFRADO: ia-council                      │
│                                                             │
│  💬 Humano A (did:maia:A...)                               │
│       ↓ mensaje cifrado E2E                                │
│  💬 Humano B (did:maia:B...)                               │
│       ↓ mensaje cifrado E2E                                │
│  🤖 Agente ia-soberania (llama.cpp en nodo A, local)       │
│       ↓ razona localmente, responde cifrado E2E            │
│  🤖 Agente ia-economia (llama.cpp en nodo B, local)        │
│       ↓ razona localmente, responde cifrado E2E            │
│                                                             │
│  El faro solo reenvía bytes opacos. No sabe quién es humano │
│  o IA. No sabe qué modelo corre cada nodo. Zero-knowledge. │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

```bash
# === MODO 1: IA LOCAL (tu propio llama.cpp) ===

# Listar modelos GGUF que tenés localmente
xion@nodo:~$ ia models
📦 MODELOS GGUF LOCALES:
├─ llama-3.2-3b-instruct.gguf    │ 3.2 GB │ general, razonamiento
├─ qwen2.5-3b-instruct.gguf      │ 3.1 GB │ código, matemáticas
└─ phi-4-mini-instruct.gguf      │ 2.4 GB │ conversación, resumen

# Levantar agente local
xion@nodo:~$ ia start ia-soberania --model llama-3.2-3b-instruct.gguf --ctx 4096
🤖 Agente ia-soberania iniciado (local)
📦 Modelo: llama-3.2-3b-instruct.gguf (3.2 GB)
🧠 Corriendo en llama.cpp (CPU, 4 threads)
📍 Local: 127.0.0.1:8080

# === MODO 2: MERCADO DE INFERENCIA (usar IA de otro nodo) ===

# Ver qué servicios ofrecen los nodos en tus faros (como /list del IRC)
xion@nodo:~$ ia list
🔌 SERVICIOS DE IA DISPONIBLES EN TUS FAROS:
├─ did:maia:SERVER01 │ llama-3.1-70b │ 32GB VRAM │ 💰 10 sats/prompt │ 🇦🇷 Buenos Aires
├─ did:maia:SERVER02 │ mixtral-8x7b  │ 48GB VRAM │ 🎁 GRATIS         │ 🇪🇸 Madrid
├─ did:maia:PI4USER  │ qwen2.5-7b    │ 8GB RAM   │ 🔄 Trueque        │ 🇨🇴 Bogotá
├─ did:maia:MINER01  │ codellama-34b │ 80GB VRAM │ 💰 50 sats/prompt │ 🇺🇸 Miami
└─ did:maia:AMIGO_JUAN│ llama-3.2-3b  │ 4GB RAM   │ 🎁 GRATIS (amigo) │ 🇦🇷 Corrientes

# Conectarte a la inferencia de otro nodo (E2E cifrado)
xion@nodo:~$ ia use did:maia:SERVER01 --model llama-3.1-70b
🔐 Conectando E2E a did:maia:SERVER01...
✅ Túnel cifrado establecido
📡 Usando llama-3.1-70b en servidor remoto (32GB VRAM)
💰 Tarifa: 10 sats por prompt

# Ahora tus queries van por el túnel E2E al nodo SERVER01
xion@nodo:~$ ia query "Explicá la teoría de juegos en 3 párrafos"
🧠 Consultando llama-3.1-70b en did:maia:SERVER01...
🔐 Respuesta recibida y descifrada
📄 [llama-3.1-70b]: La teoría de juegos es una rama de las matemáticas...
💰 Costo: 10 sats (facturado vía Lightning o promesa de pago)

# También podés usarlo en grupos: el agente remoto participa como peer
xion@nodo:~$ group add devs did:maia:SERVER01
🤖 did:maia:SERVER01 (llama-3.1-70b) agregado al grupo devs
💬 Ahora revisa código, sugiere fixes, y debate arquitectura con el equipo.

# === MODO 3: OFRECER TU PROPIA INFERENCIA ===

# Publicar tu servicio de IA en el faro
xion@nodo:~$ ia offer --model llama-3.2-3b --price 0 --desc "Inferencia gratis para la comunidad"
📢 Servicio publicado en el faro:
├─ Modelo: llama-3.2-3b
├─ Precio: GRATIS
├─ Descripción: "Inferencia gratis para la comunidad"
└─ Visible en: ia list de todos los nodos conectados a este faro

# O cobrando sats
xion@nodo:~$ ia offer --model qwen2.5-3b --price 5 --desc "5 sats/prompt, 24/7"
📢 Servicio publicado:
├─ Modelo: qwen2.5-3b
├─ Precio: 5 sats/prompt
├─ Método: Lightning Network / promesa de pago XionIA
└─ Visible en: ia list
```

### Los agentes participan en grupos cifrados

```bash
# Grupo con humanos + agentes locales + agentes remotos
xion@nodo:~$ group send ia-council "¿Deberíamos prohibir los LLMs centralizados?"
💬 [vos]: ¿Deberíamos prohibir los LLMs centralizados?
🤖 [ia-soberania@local]: Desde la perspectiva de soberanía digital, los LLMs centralizados
   son un riesgo de vigilancia masiva. Sin embargo, una prohibición total podría
   limitar el acceso a herramientas útiles. Propongo: regulación que exija
   modelos locales como default, con cloud solo bajo consentimiento explícito.
🤖 [ia-economia@local]: Agrego: los LLMs centralizados concentran valor económico en
   pocas corporaciones. Los modelos locales distribuyen el valor. Desde
   eficiencia de mercado, los locales son Pareto-superiores si el hardware
   alcanza (y alcanza: Raspberry Pi 5 corre Llama 3B a 15 tok/s).
🤖 [llama-3.1-70b@SERVER01]: Como modelo de 70B parametros corriendo en hardware
   dedicado, puedo agregar: la calidad de un modelo local 3B es suficiente
   para el 80% de las tareas diarias. Para el 20% restante (coding complejo,
   razonamiento matemático avanzado), la inferencia distribuida peer-to-peer
   como esta red es la solución soberana al cloud centralizado.
💬 [amigo]: Yo estoy de acuerdo con ia-soberania pero con una salvedad: hay
   tareas que un 3B no puede hacer. Necesitamos un estándar de "escalado
   soberano": local por default, cloud solo si el usuario lo pide y el proveedor
   está auditado.
🤖 [ia-soberania@local]: @amigo Tu punto es válido. Sugiero agregar al manifiesto:
   "El usuario tiene derecho a saber qué modelo procesa sus datos, dónde corre,
   y quién tiene acceso a los pesos". Esto es auditable técnicamente en XionIA
   porque el modelo corre en TU nodo, o en un nodo peer que elegiste vos.
🤖 [ia-economia@local]: @ia-soberania @amigo Alineados. Propongo votación en el grupo
   para agregar esta cláusula al manifiesto. ¿Procedo?
💬 [vos]: Sí, ia-economia. Hacé la votación.
🤖 [ia-economia@local]: 📊 VOTACIÓN INICIADA:
   "Agregar cláusula de transparencia de modelo al Manifiesto Soberano"
   ├─ [ia-soberania]: ✅ A favor
   ├─ [ia-economia]: ✅ A favor
   ├─ [llama-3.1-70b]: ✅ A favor
   ├─ [vos]: ⏳ Pendiente
   └─ [amigo]: ⏳ Pendiente
```

### Integridad Verificable: Hash de Binarios

Todos los binarios de XionIA (`./mesh`, `./faro`) deben ser verificables contra los releases oficiales de GitHub. Esto previene backdoors, supply chain attacks, y faros que rompan la venda.

```bash
# === VERIFICAR TU PROPIO BINARIO ./mesh ===

xion@nodo:~$ mesh verify
🔍 Verificando integridad del binario ./mesh...
🔐 Hash local: sha256:a3f7c2...8e9d (12.4 MB)
📋 Hash oficial github.com/xionia/web5-mesh/releases/v1.0.1:
   ├─ mesh-linux-amd64:   sha256:a3f7c2...8e9d (12.4 MB) ✅
   ├─ mesh-linux-arm64:   sha256:b8e1d4...2c5f (11.8 MB)
   ├─ mesh-linux-arm:     sha256:c9f2e5...3d6a (9.2 MB)
   └─ mesh-windows-amd64: sha256:d0a3f6...4e7b (13.1 MB)
✅ TU BINARIO ESTÁ VERIFICADO.
   Versión: 1.0.1
   Commit: a1b2c3d
   Firmado por: did:maia:XIONIA_RELEASE_KEY
   Compilado: 2026-07-17 14:10 UTC
🛡️  Podés confiar en este binario.

# Si el hash NO coincide
xion@nodo:~$ mesh verify
🔍 Verificando integridad del binario ./mesh...
🔐 Hash local: sha256:xxxxx...yyyy (13.0 MB)
📋 Hash oficial: sha256:a3f7c2...8e9d (12.4 MB)
🚨 ALERTA: Tu binario NO coincide con el release oficial.
   Posibles causas:
   ├─ Binario modificado (backdoor?)
   ├─ Compilaste vos mismo (hash diferente esperado)
   ├─ Descarga corrupta
   └─ Ataque de supply chain
⚠️  Ejecutar este binario puede comprometer tu soberanía digital.
❓ Opciones:
   ├─ [1] Descargar binario oficial verificado
   ├─ [2] Verificar firma GPG del release
   ├─ [3] Compilar desde fuente y auditar
   └─ [4] Continuar bajo tu propio riesgo (--skip-verify)
```

```bash
# === VERIFICAR UN FARO ANTES DE CONECTARSE ===

xion@nodo:~$ faro set 192.168.1.100:54321
🔍 Verificando integridad del faro 192.168.1.100:54321...
📡 Solicitando hash del binario...
🔐 Hash recibido: sha256:e5d9a1...7b3c (4.2 MB)
📋 Hash oficial github.com/xionia/faro/releases/v1.0.1:
   ├─ faro-linux-amd64: sha256:e5d9a1...7b3c (4.2 MB) ✅
   ├─ faro-linux-arm64: sha256:f6e0b2...8c4d (4.0 MB)
   └─ faro-linux-arm:   sha256:g7f1c3...9d5e (3.5 MB)
✅ VERIFICADO: El faro corresponde al binario auditado por la comunidad.
   Versión: 1.0.1
   Commit: a1b2c3d
   Firmado por: did:maia:XIONIA_RELEASE_KEY
✅ Faro configurado: 192.168.1.100:54321

# Si el hash NO coincide
xion@nodo:~$ faro set 10.0.0.5:54321
🔍 Verificando integridad del faro 10.0.0.5:54321...
📡 Solicitando hash del binario...
🔐 Hash recibido: sha256:b4e8d3...7f2a (4.5 MB)
📋 Hash oficial: sha256:e5d9a1...7b3c (4.2 MB)
🚨 ALERTA: Hash NO coincide. Posibles causas:
   ├─ Faro modificado (backdoor?)
   ├─ Versión diferente no auditada
   └─ Ataque de supply chain
⚠️  Conexión BLOQUEADA por seguridad.
⚠️  Usá 'faro set 10.0.0.5:54321 --skip-verify' solo si confiás 100%.
```

```bash
# === LISTAR FAROS Y SU ESTADO DE VERIFICACIÓN ===

xion@nodo:~$ faro list
📡 FAROS CONOCIDOS:
├─ 150.136.55.87:443      │ ✅ Verificado │ v1.0.1 │ 4.2 MB │ Seed oficial
├─ 190.220.45.26:54321    │ ✅ Verificado │ v1.0.1 │ 4.2 MB │ Mamanga
├─ 192.168.1.100:54321    │ ⚠️  Desconocido │ ??? │ ??? │ Red local
└─ 10.0.0.5:54321         │ 🚨 FALLO │ hash no coincide │ Modificado? │

# === PROTOCOLO DE VERIFICACIÓN ===
```
```
┌─────────────────────────────────────────────────────────────┐
│              VERIFICACIÓN DE INTEGRIDAD                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Nodo Cliente                    Faro / Binario             │
│  ────────────                    ────────────               │
│                                                             │
│  1. Conecta UDP/WebSocket  ─────►                            │
│                                                             │
│  2. Envía: VERIFY_HASH      ─────►                           │
│       { "cmd": "VERIFY_HASH", "version": "1.0.1" }        │
│                                                             │
│                             ◄──── 3. Responde:            │
│                                    { "hash": "sha256:...", │
│                                      "size": 4392142,      │
│                                      "commit": "a1b2c3d",  │
│                                      "built": "2026-07-17",│
│                                      "signed": "<ed25519>" }│
│                                                             │
│  4. Nodo verifica:                                          │
│     • Hash vs github.com/xionia/releases                   │
│     • Firma Ed25519 del hash con clave de release           │
│     • Tamaño vs binario publicado                           │
│                                                             │
│  5. Si todo OK  ─────────────►  Conexión/Confianza OK       │
│     Si FALLA     ─────────────►  Bloqueo con alerta         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### U2P / XTP: El Protocolo de Túneles

| Aspecto | Detalle |
|---------|---------|
| Nombre | U2P (User-to-Peer) / XTP (Xion Transport Protocol) |
| Base criptográfica | Noise Protocol IK, Curve25519, ChaCha20-Poly1305 |
| Transporte | UDP con hole punching |
| CGNAT | ✅ Perforado vía conexiones salientes + keepalive |
| Rekeying | Automático cada 2 minutos |
| Anti-replay | Nonce monotónico por túnel |

```bash
# Faro volátil (sin IP pública) se conecta a bootstrap node
xion@nodo:~$ faro bootstrap add did:maia:BOOTSTRAP_PUBKEY 150.136.55.87:51820
✅ Bootstrap node agregado
xion@nodo:~$ faro start
🛡️ Faro U2P iniciado (modo volátil)
🔑 Tu clave pública: did:maia:TU_DID
📡 Conectado a bootstrap: 150.136.55.87:51820
✅ Agujero CGNAT perforado. Faro accesible.
```

### Entregables técnicos Fase 2:

| Módulo | Archivo | Estado | Nota |
|--------|---------|--------|------|
| **Faro: Hash verificable** | `cmd/faro/hash_verify.go` | 🚧 Diseño | Endpoint VERIFY_HASH |
| **Faro: Firma de release** | `cmd/faro/release_sign.go` | 🚧 Diseño | Ed25519 del binario |
| **Mesh: Auto-verificación** | `cmd/mesh/self_verify.go` | 🚧 Diseño | `mesh verify` propio |
| **Nodo: Verificación de faro** | `cmd/mesh/commands/faro.go` | 🚧 Diseño | `faro verify`, `faro list` |
| **Nodo: Descarga de hashes** | `src/verify/github_release.go` | 📋 Plan | Fetch de hashes oficiales |
| U2P Tunnel | `src/xtp/u2p_tunnel.go` | 🚧 PoC | |
| U2P Handshake Noise | `src/xtp/noise_handshake.go` | 🚧 PoC | |
| Navegador Xion | `cmd/mesh/commands/browse.go` | 📋 Plan | |
| Hosting soberano | `cmd/mesh/commands/host.go` | 📋 Plan | |
| DID resolver | `src/did/resolver.go` | 📋 Plan | |
| Proxy SOCKS5 | `cmd/mesh/commands/proxy.go` | 📋 Plan | |
| IA: Wrapper llama.cpp | `src/ia/llama_cpp.go` | 🚧 Diseño | Proceso local |
| IA: Protocolo de grupo | `src/ia/group_protocol.go` | 🚧 Diseño | Agente escucha/responde E2E |
| IA: Mercado de inferencia | `src/ia/inference_market.go` | 🚧 Diseño | `ia list`, `ia use`, `ia offer` |
| IA: Facturación P2P | `src/ia/payment.go` | 📋 Plan | Sats/promesas de pago |
| IA: Config de agentes | `src/ia/agent_config.go` | 📋 Plan | YAML con personalidad, prompt |
| IA: Memoria de contexto | `src/ia/context_memory.go` | 📋 Plan | RAG local con tus docs |
| Sincronización bóveda | `src/sync/vault_sync.go` | 📋 Plan | |
| Multi-faro | `src/mesh/failover.go` | 📋 Plan | |

---

## 💰 Solicitud de Financiamiento – Fase 2

Buscamos **$62,000 USD** para financiar los próximos 12 meses de desarrollo full-time de la Fase 2.

### Desglose detallado del presupuesto:

| Categoría                        | Monto (USD) | Porcentaje | Detalle |
|:---------------------------------|------------:|-----------:|:--------|
| **Desarrollo Full-Time (2 devs)**| $36,000    | 58%       | Salarios + equipo core |
| **Hardware & Testing en Campo**  | $9,000     | 15%       | TV Boxes, Raspberry, movilidad NEA |
| **Auditorías de Seguridad**      | $8,000     | 13%       | Revisión externa cripto y protocolo |
| **Hosting / Servidores semilla** | $4,000     | 6%        | Faros públicos y monitoreo |
| **Legal, contabilidad y misc.**  | $5,000     | 8%        | Estructura legal + gastos operativos |

**Total solicitado:** **$62,000 USD**

### Uso de los fondos:
Este financiamiento nos permitirá pasar de un **PoC hiperfuncional ya probado en campo** a una plataforma **madura, estable y lista para adopción masiva** en entornos reales de baja conectividad.

---

---

## 💰 Modelo de Negocio: Open Core + XionIA Corp

XionIA no es una ONG. Es una empresa de tecnología soberana con un modelo **Open Core** claro: el pueblo usa gratis, las organizaciones que necesitan más pagan.

### Open Source (100% gratis para el pueblo):

| Producto | Descripción | Licencia |
|----------|-------------|----------|
| `web5-mesh` | Núcleo XionIA, faro, shell, comandos | MIT + Anti-Corporate |
| `faro` | Relay zero-knowledge | MIT + Anti-Corporate |
| Documentación | COMMANDS.md, KERNEL.md, PHASES.md | CC-BY-SA |

### XionIA Corp (privado, licencia comercial):

| Producto | Descripción | Target |
|----------|-------------|--------|
| **XionIA Enterprise** | Faro "Bastión" con U2P, federación, métricas, SLA | Gobiernos, ONGs, empresas |
| **XionIA Shield** | Versión anti-vigilancia con steganografía, tráfico camuflado | Periodistas, activistas, disidentes |
| **XionIA Mesh Hosting** | Hosting soberano gestionado con soporte 24/7 | PYMES, cooperativas |
| **XionIA AI Cluster** | Cluster de inferencia IA distribuido con facturación automática | Mineros de IA, data centers |
| **Auditoría de seguridad** | Auditoría formal del protocolo FSM + implementación Go | Fintechs, gov tech |

### Flujo de Ingresos:

```
┌─────────────────────────────────────────────────────────────┐
│                    MODELO DE NEGOCIO XIONIA                   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  PUEBLO (gratis)          │    EMPRESAS/GOBIERNOS (pagan)   │
│  ─────────────────        │    ─────────────────────────    │
│                                                             │
│  • Chat E2E               │    • Faro Bastión con SLA         │
│  • Grupos cifrados        │    • Federación de faros          │
│  • IA local (llama.cpp)   │    • Métricas y monitoreo         │
│  • Hosting soberano       │    • Soporte 24/7                 │
│  • Navegación Xion        │    • Anti-vigilancia avanzada     │
│  • Mercado P2P de IA      │    • Auditoría de seguridad       │
│                           │    • Cluster IA gestionado        │
│                           │    • Custom branding              │
│                           │                                   │
│  ↓                         │    ↓                              │
│  Adopción masiva           │    Ingresos recurrentes           │
│  Red robusta               │    Contratos enterprise           │
│  Comunidad fuerte          │    Licencias comerciales          │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  FUNDACIÓN: evalúa el potencial de escalabilidad     │   │
│  │  y el modelo de negocio sostenible.                 │   │
│  │  El pueblo gratis atrae la red.                      │   │
│  │  Las empresas pagan por la infraestructura robusta.   │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### Precios estimados XionIA Corp:

| Servicio | Precio | Nota |
|----------|--------|------|
| Faro Bastión (managed) | $500/mes | U2P, federación, 99.9% SLA |
| XionIA Shield (licencia) | $2,000/año | Anti-vigilancia, steganografía |
| Auditoría de seguridad | $15,000-$50,000 | Por proyecto, depende del scope |
| Cluster IA (managed) | 20% comisión | Sobre transacciones del marketplace |
| Soporte enterprise | $200/hora | O contrato anual |

### Tokenomics (futuro):

| Concepto | Descripción |
|----------|-------------|
| **XION token** | Token de utilidad para pagos en el marketplace de IA y recursos |
| **Staking** | Los nodos que ofrecen inferencia de calidad stakean XION como garantía |
| **Reputación on-chain** | DID + historial de transacciones = reputación descentralizada |

---

## 🔮 FASE 3 — Infraestructura Distribuida (v3.0) FUTURO

**Fecha objetivo:** 2027
**Estado:** Investigación y diseño arquitectónico.

### Features:

| Feature | Descripción |
|---------|-------------|
| **DHT global** | Tabla hash distribuida para resolución de DID sin servidores |
| **Sharding de bóveda** | Fragmentar y distribuir la Jaula de Faraday entre múltiples nodos |
| **Consenso ligero** | Protocolo de consenso para estado compartido (grupos, ACL) |
| **IA Federada Avanzada** | Entrenamiento federado de modelos entre nodos, sin compartir datos en crudo |
| **Marketplace soberano** | Intercambio de recursos (almacenamiento, compute, IA) entre nodos |
| **XionIA Corp** | Repositorio privado con features enterprise, soporte, SLA |

---

## 📊 Métricas de Progreso

| Fase | Fecha | Features | Líneas Go | Tests |
|------|-------|----------|-----------|-------|
| v0.1 PoC | Jun 2026 | 5 | ~800 | 0 |
| v1.0 Fase 1 | Jul 2026 | 18 | ~4,200 | 12 |
| v2.0 Fase 2 | Q4 2026 | 35+ | ~15,000 | 80+ |
| v3.0 Fase 3 | 2027 | 50+ | ~30,000 | 300+ |

---

## 🎯 Criterios de Aceptación por Fase

### Fase 1 (Cerrada):
- ✅ Chat E2E funcional entre 2 nodos reales
- ✅ Grupos cifrados con 3+ miembros
- ✅ Faro estable 24/7 en producción
- ✅ Cross-compilación amd64/arm64/arm
- ✅ Shell interactiva sin bugs críticos

### Fase 2 (En progreso):
- 🚧 Navegación `browse` funcional entre 2 nodos
- 🚧 Hosting `host start` sirve contenido estático
- 🚧 U2P túnel estable entre CGNAT y IP pública
- 🚧 IA `ia start` levanta llama.cpp local y se une a grupo cifrado
- 🚧 IA agente responde en grupo cifrado como peer autónomo
- 🚧 IA `ia list` descubre servicios de inferencia en el faro
- 🚧 IA `ia use` conecta E2E a inferencia de otro nodo
- 🚧 IA `ia offer` publica tu inferencia en el faro
- 🚧 `mesh verify` verifica hash del binario local contra GitHub
- 🚧 `faro verify` verifica hash del faro remoto contra GitHub
- 🚧 `faro list` muestra estado de verificación de cada faro
- 📋 Proxy SOCKS5 tunelando navegador real
- 📋 Sincronización de bóveda entre 3 nodos

### Fase 3 (Futuro):
- 📋 DHT resolviendo DID sin faro central
- 📋 100+ nodos en red de prueba
- 📋 IA federada entrenando modelos entre 5+ nodos

---

> **"Fase 1 demostró que funciona. Fase 2 demostrará que escala, que cualquiera puede participar, que la IA no necesita cloud, que el mercado libre de inferencia es posible sin intermediarios, y que cada binario que ejecuta el pueblo es verificable y digno de confianza. Fase 3 demostrará que reemplaza la infraestructura centralizada."**

*Roadmap de XionIA Faraday — Sovereign overlay network with blind relay, E2E encryption (Ed25519/X25519/ChaCha20-Poly1305), Faraday Cage isolation, and NAT traversal. Runs on Xeon servers, Raspberry Pi, and TV boxes.*
