# XionIA - Documentación Técnica Completa

## Arquitectura

XionIA es una **red de túneles cifrados soberanos** entre pares de confianza. No es P2P, no es Kademlia, no es DHT. Es un modelo de intermediario agnóstico con **Jaula de Faraday** como entorno de trabajo aislado.

### Comparación con otros modelos

| Característica | Kademlia / DHT | XION Faraday |
|:---------------|:---------------|:-------------|
| **Modelo** | Malla descentralizada | Red de túneles punto a punto |
| **Descubrimiento** | Tablas de routing distribuidas | Faro Ciego central (volátil) |
| **Enrutamiento** | Saltos entre nodos (O(log N)) | Túneles directos (O(1)) |
| **Confianza** | Cualquiera participa | ACL explícita (solo confiables) |
| **Anonimato** | IPs visibles | IPs ocultas detrás del Faro |
| **Productividad** | Solo mensajería | Sistema Operativo sobre udp |

### Principio fundamental: El host es un medio hostil

Todo lo que toca el sistema operativo puede estar comprometido. Por eso XionIA crea su propio espacio aislado: la **Jaula de Faraday** (`~/.xion/`).

```
┌─────────────────────────────────────────────┐
│  HOST (Medio Hostil)                        │
│  ┌───────────────────────────────────────┐  │
│  │  Web5-Mesh/ (Código fuente)           │  │
│  └───────────────────────────────────────┘  │
│                                             │
│  ┌───────────────────────────────────────┐  │
│  │  ~/.xion/ (Jaula de Faraday)          │  │
│  │  ├── config.json (config nodo)        │  │
│  │  ├── aliases.json (alias locales)     │  │
│  │  ├── identity.key (claves privadas)   │  │
│  │  ├── acl.json (pares de confianza)    │  │
│  │  ├── workspace/ (docs del usuario)    │  │
│  │  │   ├── informe.md                   │  │
│  │  │   ├── informe.pdf (0600)           │  │
│  │  │   └── informe.pdf.sig (firma)      │  │
│  │  ├── inbox/ (archivos recibidos)      │  │
│  │  ├── archive/ (histórico cifrado)     │  │
│  │  └── sessions/ (chats persistentes)   │  │
│  └───────────────────────────────────────┘  │
│                                             │
│  ═══════════════════════════════════════    │
│  Túnel Cifrado (UDP + Padding)              │
│  ═══════════════════════════════════════    │
│         ↓                                   │
│  ┌───────────────────────────────────────┐  │
│  │  FARO (Relay ciego en RAM)            │  │
│  │  - No lee contenido                   │  │
│  │  - No guarda logs                     │  │
│  │  - Solo retransmite blobs cifrados    │  │
│  │  - Puerto dinámico (anti-drop ISP)    │  │
│  └───────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

## Arquitectura de Escalado

### Capas de la Red

| Capa | Función | Escalabilidad |
|:-----|:--------|:--------------|
| **Faro Local** | Registra nodos de su región (ACL local) | Hasta 10,000 nodos |
| **Faro Federado** | Se comunica con otros Faros | Hasta 1,000 Faros |
| **Red Global** | Intercambio de rutas entre Faros | Millones de nodos |

### Diagrama de Faros Federados

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           FAROS FEDERATIVOS                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌──────────────┐         ┌──────────────┐         ┌──────────────┐      │
│   │   FARO A     │────────▶│   FARO B     │────────▶│   FARO C     │      │
│   │  (Región 1)  │◀────────│  (Región 2)  │◀────────│  (Región 3)  │      │
│   └──────┬───────┘         └──────┬───────┘         └──────┬───────┘      │
│          │                        │                        │               │
│          ▼                        ▼                        ▼               │
│   ┌──────────────┐         ┌──────────────┐         ┌──────────────┐      │
│   │   Nodos A    │         │   Nodos B    │         │   Nodos C    │      │
│   │   (ACL A)    │         │   (ACL B)    │         │   (ACL C)    │      │
│   └──────────────┘         └──────────────┘         └──────────────┘      │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│ Los Faros se federan entre sí. Cada Faro conoce a otros Faros.             │
│ Un nodo en Faro A puede encontrar a un nodo en Faro C.                     │
│ Directorio público en iap2p.uk (sin servidor central de discovery).        │
└─────────────────────────────────────────────────────────────────────────────┘
```

## 🔐 Criptografía

### Stack Criptográfico

| Componente | Algoritmo | Uso |
|:-----------|:----------|:----|
| **Identidad** | Ed25519 | Firma de mensajes, DIDs y archivos |
| **Intercambio de claves** | X25519 | Derivación de clave compartida E2E |
| **Cifrado simétrico** | ChaCha20-Poly1305 | Cifrado de mensajes y archivos |
| **Hash** | SHA-256 | Integridad de archivos y sellado |
| **Identificador** | Base58 | DID legible (`did:maia:xxxxx`) |

### Flujo de Cifrado E2E

```
1. Nodo A genera par de claves Ed25519 (firma) + X25519 (cifrado)
2. Nodo A anuncia su DID + PubKey al Faro
3. Nodo B quiere hablar con A:
   a. Consulta WHERE_IS did:maia:A al Faro
   b. Faro devuelve PubKey de A
   c. B deriva clave compartida con X25519
   d. B cifra mensaje con ChaCha20
   e. Faro retransmite blob cifrado a A
   f. A descifra con su clave privada
```

**Anti-DPI:** Cada paquete UDP lleva **50-200 bytes de padding aleatorio**. El tráfico parece ruido blanco, no una VPN.

### Sellado de Documentos (Fase 2)

Dos modos de exportación, ambos dentro de la Jaula:

**Modo Fantasma (`/export archivo.pdf --clean`):**
```json
{
  "author": null,
  "creator": null,
  "producer": null,
  "creation_date": "0000:00:00 00:00:00",
  "mod_date": "0000:00:00 00:00:00"
}
```
Cero metadatos. El archivo es anónimo e irrastreable.

**Modo Sellado (`/export archivo.pdf --seal --author="Fernando"`):**
```json
{
  "author": "Fernando Martin Lopez",
  "did": "did:maia:GVdM6WixFeVdswv89uwxNVuWNtT8c2Jx5pgNgfkF6Tw6",
  "creation_date": "2026-06-29T19:45:00Z",
  "signature": "ed25519:abc123...",
  "hash": "sha256:def456..."
}
```
Autoría criptográficamente verificable, timestamp inmutable, integridad garantizada.

## 🛡️ Seguridad

| Capa | Implementación | Estado |
|:-----|:---------------|:------:|
| Transporte | UDP Nativo + Padding Aleatorio (anti-DPI) | ✅ |
| Relay | Faro Ciego (RAM volátil, cero logs) | ✅ |
| NAT Traversal | Keep-alive cada 15s (CGNAT friendly) | ✅ |
| Identidad | Ed25519 DIDs (`did:maia:Base58`) | ✅ |
| Control de Acceso | ACL (whitelist) + Pares de confianza | ✅ |
| Firma de Archivos | SHA256 + Ed25519 | ✅ |
| Bóveda Segura | Jaula de Faraday (`~/.xion/`, 0600) | ✅ |
| Handshake | Noise Protocol IK (Perfect Forward Secrecy) | ✅ |
| Puerto Dinámico | Rotación 42069-42169 (anti-drop ISP) | 🔲 |
| Cero exec.Command | Todo en Go nativo, sin shell del host | ✅ |


## 📦 Documentos Soberanos (COMPLETADO — Fase 1)

Ciclo cerrado dentro de la Jaula de Faraday para importar, firmar, verificar y exportar archivos con integridad criptográfica garantizada.

**Documentación completa:** Ver [FARADAY.md](FARADAY.md)

### Comandos

| Comando | Acción | Estado |
|:--------|:-------|:------:|
| `/import <archivo>` | Meter archivo del host a la bóveda (0600) | ✅ |
| `/sign <archivo>` | Firmar con Ed25519 + SHA256 | ✅ |
| `/verify <archivo>` | Verificar integridad y autenticidad | ✅ |
| `/export <archivo> <destino>` | Sacar archivo de la bóveda al host | ✅ |

### Flujo Completo

**Paso 1: IMPORT** — Meter archivo del host a la bóveda
```
xion@nodo:~$ import ~/documento.pdf
✅ Archivo ingresado a la bóveda:
   ├── Origen: /home/usuario/documento.pdf
   ├── Destino: ~/.xion/workspace/documento.pdf
   └── Permisos: 0600 (solo tú)
```

**Paso 2: SIGN** — Firmar criptográficamente
```
xion@nodo:~$ sign documento.pdf
✅ Archivo firmado criptográficamente:
   ├── Archivo: documento.pdf (23 bytes)
   ├── Hash SHA256: 2ee498c8fa0f778e...
   ├── Firma: documento.pdf.sig
   ├── Firmante: did:maia:5XVNWhUtMNH...
   └── Timestamp: 2026-06-17 11:21:15
```

**Paso 3: VERIFY** — Verificar integridad y autenticidad
```
xion@nodo:~$ verify documento.pdf
✅ VERIFICACIÓN EXITOSA:
   ├── Integridad: ✅ Hash válido
   ├── Autenticidad: ✅ Firma válida
   ├── Firmante: Tú mismo
   └── El archivo es auténtico y no fue modificado.
```

**Paso 4: EXPORT** — Sacar archivo de la bóveda
```
xion@nodo:~$ export documento.pdf ~/Desktop/
✅ Archivo exportado de la bóveda:
   ├── Origen (Bóveda): ~/.xion/workspace/documento.pdf
   ├── Destino (Host): ~/Desktop/documento.pdf
   └── Permisos: 0644 (listo para compartir)
```

### Características de seguridad

- ✅ **Permisos 0600** en la bóveda (solo el dueño puede leer)
- ✅ **Hash SHA256** para integridad
- ✅ **Firma Ed25519** para autenticidad del firmante
- ✅ **Archivo .sig separado** para verificación sin modificar el original
- ✅ **Timestamp** de firma para trazabilidad
- ✅ **Cero exec.Command** — todo en Go nativo
- ✅ **Transporte agnóstico** — el archivo + firma pueden enviarse por cualquier medio (email, USB, Web)

### Archivos involucrados

- `cmd/mesh/commands/import.go` — Importar a bóveda
- `cmd/mesh/commands/sign.go` — Firmar con Ed25519
- `cmd/mesh/commands/verify.go` — Verificar firma + hash
- `cmd/mesh/commands/export.go` — Exportar desde bóveda

### Casos de uso

**Firmar un contrato:**
```
import ~/contrato.pdf
sign contrato.pdf
verify contrato.pdf
export contrato.pdf ~/Desktop/
export contrato.pdf.sig ~/Desktop/
# Enviá ambos archivos por email al receptor
```

**Detectar manipulación:**
```
# Alguien modificó el archivo en el camino
verify contrato.pdf
❌ INTEGRIDAD COMPROMETIDA:
   ├── Hash esperado: 2ee498c8fa0f778e...
   ├── Hash actual: 9f8e7d6c5b4a3210...
   └── El archivo fue modificado después de la firma.
```

---

## 📝 Flujo de Trabajo Soberano (Fase 2)

Ciclo cerrado dentro de la Jaula. La información nunca toca el mundo exterior hasta que el usuario da la orden final.

### Comandos del pipeline

| Comando | Acción | Destino |
|:--------|:-------|:--------|
| `/edit archivo.md` | Crear/editar en el shell | `~/.xion/workspace/` |
| `/export archivo.pdf --seal` | Convertir + sellar metadatos | PDF/DOC en workspace |
| `/export archivo.pdf --clean` | Convertir + borrar metadatos | Archivo fantasma |
| `/send archivo.pdf @colega` | Enviar a otro nodo | Red iAP2P (cifrado E2E) |
| `/print archivo.pdf` | Imprimir en red | Impresora vía IPP (puerto 631) |
| `/save-local archivo.pdf` | Guardar offline | Disco local/USB cifrado |
| `/inbox` | Ver archivos recibidos | Archivos de otros nodos |

### Cliente IPP Nativo (sin CUPS)

El binario actúa como cliente IPP ligero. Sin `lpr`, sin `lp`, sin spooler del SO.

```
Shell (/print) → Binario Xion → Socket TCP directo → Impresora (puerto 631)
                    ↓
              Sin CUPS, sin lpr, sin logs del SO
```

**Ventajas:**
- Cero logs en `/var/log/cups/`
- Sin cola de impresión del SO
- Si falla la impresión, el archivo se borra de memoria
- Cross-platform (Linux/Windows/macOS sin drivers)
- Auditoría cero: el SO no registra la impresión

**Comandos:**
```bash
/printer list          # Descubrimiento mDNS (_ipp._tcp)
/printer set 192.168.1.50  # Configurar impresora por defecto
/print informe.pdf     # Envío directo al puerto 631
/print informe.pdf --copies=2
```

## 💬 Chat Asíncrono con Sesiones (Fase 2)

Sistema de sesiones tipo "pestañas de navegador" que libera la shell para multitarea.

### Alias Locales

Archivo `~/.xion/aliases.json` mapea nombres humanos a DIDs:

```bash
/alias add amigo did:maia:GVdM6WixFeVdswv89uwxNVuWNtT8c2Jx5pgNgfkF6Tw6
/alias list
/alias remove amigo
/chat amigo "hola"    # Resuelve alias → DID internamente
```

### Sesiones Persistentes

```bash
/chat amigo "hola"
# → Crea sesión [1] automáticamente
# → Envía mensaje en goroutine (no bloquea shell)
# → Notificación: 🔔 [1] amigo respondió

/session list
# 📋 SESIONES ACTIVAS:
#   [1] chat: amigo 🟢 Activa (🔴 2 sin leer)
#   [2] editor: nota.txt ⚪ Background

/session attach 1
# 🔗 Conectado a Sesión [1]
# → Prompt cambia, escribís directo sin /chat
# → Todo va al chat con amigo

/session detach
# ⏸️ Sesión [1] en background
# → Vuelve al shell principal
# → Notificaciones siguen llegando
```

### Historial Persistente

Cada sesión guarda su historial en `~/.xion/sessions/<id>.json`:
- Mensajes enviados/recibidos
- Timestamps
- Firma criptográfica de cada mensaje
- Consultable offline sin conexión al Faro

## 🤖 MaIA Local-First (Fase 2)

Inferencia de IA **dentro de la red del Faro**, sin APIs externas.

### Arquitectura

```
┌─────────────────────────────────────────┐
│  RED DEL FARO                           │
│                                         │
│  ┌──────────────┐                       │
│  │   FARO       │                       │
│  │  + MaIA      │  ← Modelo GGUF local  │
│  │  (endpoint)  │    cuantizado         │
│  └──────┬───────┘                       │
│         │                               │
│         ▼                               │
│  ┌──────────────────────────────────┐   │
│  │  Nodos conectados al Faro        │   │
│  │  - Nodo A → /ia "consulta"       │   │
│  │  - Nodo B → /ia "consulta"       │   │
│  │  - Nodo C → /ia "consulta"       │   │
│  └──────────────────────────────────┘   │
│                                         │
│  Datos NUNCA salen de la red del Faro   │
└─────────────────────────────────────────┘
```

### Características

- Cada Faro puede levantar su propio modelo (GGUF cuantizado)
- Nodos hacen consultas vía `/ia "prompt"`
- Inferencia compartida en la red local/regional del Faro
- Sin dependencia de OpenAI, Anthropic, Google
- Fallback a inferencia mínima en hardware limitado

## 📁 Estructura del Proyecto

```
Web5-Mesh/
├── cmd/                          # Aplicaciones principales
│   ├── faro/
│   │   └── main.go               # Relay ciego (RAM-only, zero logs)
│   └── mesh/
│       ├── main.go               # Entry point del nodo XionIA
│       ├── shell.go              # Shell interactiva (REPL)
│       └── commands/             # Comandos del shell
│           ├── acl.go            # Gestión de Access Control List
│           ├── alias.go          # Alias locales (nodos y Faros)
│           ├── chat.go           # Chat E2E asíncrono
│           ├── clear.go          # Limpiar pantalla
│           ├── editor.go         # Editor soberano integrado
│           ├── exit.go           # Salir del shell
│           ├── export.go         # Exportar PDF/DOC con sellado
│           ├── faro.go           # Comandos del Faro
│           ├── help.go           # Ayuda del shell
│           ├── import.go         # Importar archivos a la Jaula
│           ├── inbox.go          # Archivos recibidos
│           ├── notif.go          # Gestión de notificaciones
│           ├── notifier.go       # Sistema de notificaciones
│           ├── ping.go           # Ping a otros nodos
│           ├── print.go          # Impresión directa (IPP nativo)
│           ├── printer.go        # Gestión de impresoras
│           ├── registry.go       # Registro de nodos conocidos
│           ├── save.go           # Guardado local cifrado
│           ├── send.go           # Envío de archivos E2E
│           ├── session.go        # Gestión de sesiones
│           ├── sign.go           # Firmar archivos (Ed25519)
│           ├── ssh.go            # Túnel SSH sobre XionIA
│           ├── status.go         # Estado de la red y pares
│           ├── unix.go           # Comandos Unix-like
│           ├── verify.go         # Verificar firmas
│           └── whoami.go         # Mostrar identidad (DID)
│
├── src/                          # Código fuente compartido
│   ├── crypto/                   # Criptografía
│   │   ├── acl.go                # Access Control List
│   │   ├── cipher.go             # ChaCha20-Poly1305
│   │   ├── cipher_session.go     # Sesiones PFS
│   │   ├── identity.go           # Ed25519/X25519
│   │   └── noise.go              # Handshake Noise IK
│   ├── printer/                  # Cliente IPP nativo
│   │   ├── ipp.go                # Protocolo IPP sobre TCP
│   │   └── mdns.go               # Descubrimiento mDNS
│   └── maia/                     # Integración IA local
│       └── client.go             # Cliente para MaIA en Faro
│
├── docs/                         # Documentación
│   ├── COMMANDS.md               # Referencia de comandos
│   ├── FARADAY.md                # Jaula de Faraday
│   ├── FAROS.md                  # Guía de Faros
│   ├── KERNEL.md                 # Kernel XionIA
│   ├── MANIFIESTO.md             # Manifiesto del proyecto
│   ├── PHASES.md                 # Roadmap de fases
│   ├── SESSIONS.md               # Gestión de sesiones
│   ├── TECHNICAL.md              # Este archivo
│   └── WHITEPAPER.md             # Whitepaper completo
│
├── build.sh                      # Script de compilación
├── go.mod                        # Dependencias Go
├── go.sum                        # Checksums
├── LICENSE-TRINCHERA.md          # Licencia MIT anti-corporativa
└── README.md                     # Introducción
```

## 🔜 Roadmap: Fase 2 — Herramienta Soberana (Código Libre)

### Milestones

| Milestone | Feature | Descripción |
|:----------|:--------|:------------|
| **M2.1** | Alias + Chat asíncrono | `~/.xion/aliases.json`, sesiones tipo pestaña, notificaciones background, historial persistente |
| **M2.2** | Faro dinámico | Puerto rotable 42069-42169, alias/topics, anti-drop ISP, logs truncados |
| **M2.3** | Procesador de texto + PDF | Editor mejorado, export PDF/DOC, sellado criptográfico, metadatos limpios |
| **M2.4** | Impresora de red | Cliente IPP nativo, mDNS discovery, cero CUPS/lpr, cero logs del SO |
| **M2.5** | MaIA Local-First | Modelo GGUF en Faro, consultas `/ia`, inferencia dentro de la red del Faro |
| **M2.6** | Shell mejorado | Modo sesión activa, help contextual, autocompletado |

### NO hacemos en Fase 2 (va a Fase 3 corporativa):

- ❌ Renombrar Faro → Bastión / Nación
- ❌ Directorio distribuido de Faros (`/faro list`)
- ❌ Gossip protocol entre Faros
- ❌ Modo dialer (conexión saliente pura)
- ❌ Sistema de alias distribuido global
- ❌ Hosting descentralizado
- ❌ Dead Man's Switch

### Alternativas soberanas para Fase 2:

- ✅ Sección en iap2p.uk donde la gente publica sus Faros manualmente
- ✅ Cada usuario monta su Faro (fijo o dinámico)
- ✅ Sin dependencia central, sin servidor dedicado

## 🏢 Fase 3 — XION Faraday Suite Enterprise (Privada)

**Inicio estimado:** Q3 2027  
**Modelo:** Licencias, soporte y servicios. Código fuente privado.

| Feature | Descripción |
|:--------|:------------|
| **Renombrar a Bastión/Nación** | Nomenclatura corporativa soberana |
| **XION Messenger Corporativo** | Cumplimiento normativo + auditoría |
| **Binarios personalizados** | Dead Man's Switch + branding |
| **Jaula de Faraday Enterprise** | Dashboards de administración |
| **Clustering + HA** | Alta disponibilidad con SLAs |
| **Soporte 24/7** | Consultoría especializada |
| **Anti-triangulación** | Protecciones avanzadas contra rastreo |
| **Gossip protocol** | Directorio distribuido real entre Bastiones |
| **Modo dialer puro** | Bastiones invisibles detrás de NAT/CGNAT |
| **Alias distribuido global** | Resolución de alias cross-Bastión |

---

*Documentación técnica de XionIA - La Xion Digital 🦾*  
*Última actualización: 29 de junio de 2026*
