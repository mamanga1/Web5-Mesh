# XionIA: Un Kernel en User-Space

XionIA no es un messenger. No es una VPN. No es una aplicación de red tradicional.
Es un User-Space Kernel que gestiona hardware virtual distribuido: red overlay,
identidad criptográfica, túneles cifrados.

---

## La Definición Correcta

XionIA es un Protocolo de Estado Finito (FSM) Distribuido que corre en user-space.
No reemplaza al kernel del host (Linux, Windows, Android), sino que corre encima
de él, abstraendo la complejidad de la red y proporcionando un entorno soberano.

> "XionIA no es un entorno de ejecución Turing-completo. Es una máquina de estados
> finita distribuida. Esa diferencia no es semántica: es la razón por la cual las
> clases de vulnerabilidades de ejecución arbitraria no tienen dónde ocurrir en
> el protocolo."

---

## Teoría de la Computabilidad Aplicada a la Seguridad

### Entornos Turing-Completos (Propósito General)

Sistemas como Linux, gVisor, seL4, Erlang/OTP son entornos de ejecución
Turing-completos. Por definición, al ser capaces de ejecutar cualquier lógica
computable, están expuestos a:

- Buffer overflows en handlers de syscalls
- Escalada de privilegios en el scheduler
- Inyección de código vía intérpretes
- Race conditions en gestión de memoria
- Exploits de ejecución arbitraria

### XionIA: FSM Distribuido (Propósito Específico)

El protocolo XionIA NO es un entorno de ejecución. Es una FSM donde:

| Capacidad | Protocolo XionIA | Implementación en Go |
|---|---|---|
| Ejecución de código arbitrario | ❌ No existe | ✅ Heredado del runtime Go |
| Interpretación de bytecode | ❌ No existe | ❌ No se usa |
| Evaluación de expresiones dinámicas | ❌ No existe | ⚠️ json.Unmarshal a structs fijos |
| Gestión de memoria dinámica | ❌ No existe | ✅ Heredado del GC de Go |
| System calls genéricas | ❌ No existe | ✅ Heredado del runtime Go |
| Transiciones entre estados finitos | ✅ Única operación | ✅ Implementado |

> **Distinción crítica:** El protocolo es FSM no-Turing-completo. La implementación
> en Go corre sobre un runtime Turing-completo. La seguridad proviene de la
> simplicidad del protocolo, no de la perfección del código.

---

## Comparativa: Kernels Tradicionales vs. XionIA

| Concepto | Kernel Tradicional (Linux) | Protocolo XionIA | Implementación Go |
|---|---|---|---|
| Tipo | Turing-completo | FSM distribuido | Turing-completo (heredado) |
| Hardware que gestiona | CPU, RAM, disco, red física | Red overlay, identidad, túneles | Mismo que protocolo |
| Filesystem | ext4, NTFS, APFS | Jaula de Faraday (.xion/) | os package de Go |
| Procesos | PIDs, scheduling | Sesiones (chat, shell) | Goroutines |
| Red | TCP/IP stack | XTP sobre UDP/WebSocket | net package de Go |
| Identidad | UID/GID | DID (did:maia:xxxxx) | Estructuras en memoria |
| Seguridad | Permisos, SELinux | ACL, firmas Ed25519, cifrado E2E | Código Go + syscalls |
| Drivers | Drivers de hardware | Faros (controladores de red virtual) | Conexiones UDP/WS |
| IPC | Pipes, sockets | Túneles cifrados entre nodos | Channels + goroutines |
| Ejecución de código | ✅ Sí (arbitraria) | ❌ No (solo primitivas) | ✅ Heredado de Go runtime |

---

## Estados Posibles en XionIA (FSM)

### Estados Válidos:
```
├── Recibir paquete UDP/WebSocket
├── Verificar firma Ed25519
├── Derivar clave X25519
├── Cifrar/descifrar ChaCha20-Poly1305
├── Guardar en Jaula de Faraday
├── Reenviar vía XTP (Noise IK)
└── Responder a comando de shell
```

### Operaciones Imposibles en la FSM:
```
├── Ejecutar código arbitrario
├── Crear procesos (fuera de goroutines Go)
├── Asignar memoria dinámica (fuera del GC de Go)
├── Cargar módulos dinámicos
├── System calls genéricas (fuera de las que usa Go runtime)
├── Interpretar bytecode
└── Evaluar expresiones dinámicas
```

---

## La Analogía del Hardware Virtual

En los sistemas distribuidos modernos, la red es el hardware virtual.
XionIA gestiona ese hardware virtual exactamente como un kernel tradicional
gestiona el hardware físico.

```
┌─────────────────────────────────────────────┐
│  APLICACIONES XIONIA                        │
│  (Chat, Hosting, IA, Firma, Editor)         │
├─────────────────────────────────────────────┤
│  XIONIA FSM (User-Space Kernel)             │
│  ├── TransportManager (XTP/Noise IK)        │
│  ├── Gestor de Identidad (DID)              │
│  ├── Gestor de Sesiones                     │
│  ├── Jaula de Faraday (Filesystem Virtual)  │
│  ├── ACL (Control de Acceso)                │
│  └── Cifrado E2E (ChaCha20-Poly1305)        │
├─────────────────────────────────────────────┤
│  HARDWARE VIRTUAL (La Red)                  │
│  ├── Faros (Signaling + Coordinación)       │
│  ├── Túneles XTP sobre UDP (Buses)          │
│  └── DIDs (IDs de Hardware Virtual)         │
├─────────────────────────────────────────────┤
│  HOST OS (Linux/Windows/Android)            │
│  └── Provee: sockets UDP, TCP, filesystem   │
│  └── Superficie de ataque heredada          │
└─────────────────────────────────────────────┘
```

> **Nota:** En Fase 1, el Faro actúa como relay. En Fase 2 (xionia-kernel),
> el Faro pasa a ser signaling puro — nunca toca datos de usuario si existe
> ruta directa entre nodos.

---

## Comparación con Sistemas Similares

| Sistema | Tipo | Turing-Completo | Superficie de Ataque |
|---|---|---|---|
| Linux | Kernel de propósito general | ✅ Sí | Enorme (300+ syscalls) |
| gVisor | Kernel en user-space (Google) | ✅ Sí | Grande (implementa syscalls) |
| seL4 | Microkernel verificado | ✅ Sí | Mediana (propósito general) |
| Erlang/OTP | Runtime de propósito general | ✅ Sí | Grande (VM completa) |
| **Protocolo XionIA** | FSM distribuido | ❌ **No** | **Mínima** |
| **Implementación Go** | Aplicación compilada | ✅ Sí (heredado) | Media (Go + host) |

---

## El Principio de Seguridad

> "No es invulnerabilidad por perfección. Es reducción de superficie de ataque
> por diseño arquitectónico basado en teoría de la computabilidad."

XionIA no es invulnerable porque sea perfecto. Es **diferentemente seguro**
porque el protocolo es una FSM simple, y las clases de vulnerabilidades que
afectan a sistemas Turing-completos no tienen "dónde" ocurrir en esa FSM.

La implementación en Go debe auditarse por separado: dependencias, concurrencia,
manejo de memoria, y superficie de ataque del runtime.

---

## La Prueba de Fuego

| Escenario | WhatsApp/Signal | XionIA |
|---|---|---|
| Hackeás el servidor central | ✅ Control total: mensajes, metadatos, contactos | ❌ No hay servidor central |
| Hackeás un nodo | N/A | Solo tráfico cifrado sin la clave privada del usuario |
| Comprometés el faro | N/A | Solo bytes opacos. Sin contenido, sin metadata |

> "Si hackeás un nodo de XionIA, solo has conseguido un nodo que ha dejado de
> obedecer el protocolo; no has comprometido el sistema, simplemente has causado
> una denegación de servicio local."

Eso es zero-knowledge por diseño arquitectónico. **Eso es XionIA.**

---

## Implicancias para la Auditoría de Seguridad

Cuando XionIA sea auditado (planificado en Fase 2 con financiamiento NLnet),
el enfoque será dual:

### Auditoría del Protocolo (FSM):
1. **Verificación formal de la FSM** — demostrar matemáticamente que todas las
   transiciones de estado son seguras.
2. **Pruebas de resistencia** — fuzzing de protocolo para forzar estados inválidos.
3. **Análisis de side-channels** — timing attacks, traffic analysis.

### Auditoría de la Implementación (Go):
1. **Stack criptográfico** — Ed25519, X25519, ChaCha20-Poly1305, Noise IK.
2. **Dependencias** — go.mod, vulnerabilidades conocidas en librerías.
3. **Concurrencia** — race conditions en variables compartidas.
   En v1.0.1 se resolvieron los principales con `connMu`, `quitMu`, `nodeMu`.
4. **Superficie de ataque del runtime** — Go + host OS.

> **El objetivo no es encontrar "vulnerabilidades tipo CVE" en el protocolo
> (que no existen por diseño), sino verificar que la implementación es fiel
> a la especificación formal de la FSM, y que la superficie de ataque del
> runtime está minimizada.**

---

## Mitigaciones de Implementación

### Implementadas en v1.0.1:
| Mitigación | Descripción | Estado |
|---|---|---|
| Mutex en conexiones | `connMu`, `quitMu`, `nodeMu` — sin race conditions | ✅ Implementado |
| Permisos 0600 | Archivos sensibles protegidos | ✅ Implementado |
| Padding anti-DPI | `crypto/rand`, tamaño aleatorio 50-200 bytes | ✅ Implementado |
| Zero logs en Faro | El Faro no loguea contenido ni metadata | ✅ Implementado |
| Gate DID | Handshake Ed25519 obligatorio antes de procesar | ✅ Implementado |

### Planificadas para Fase 2 (xionia-kernel):
| Mitigación | Descripción | Estado |
|---|---|---|
| Seccomp estricto | Filtro de syscalls: solo las necesarias | 🔄 Planificado |
| Memoria pinned | Claves fuera del GC (Mmap + Mlock) | 🔄 Planificado |
| Build estático hardeneado | Sin reflexión, tipos estrictos | 🔄 Planificado |
| Forward Secrecy completo | Noise IK conectado, rekey automático | 🔄 En desarrollo |

---

> **"XionIA: Un User-Space Kernel que gestiona hardware virtual distribuido
> mediante un Protocolo de Estado Finito. Diferentemente seguro, porque las
> clases clásicas de vulnerabilidades no aplican a su arquitectura de protocolo."**

*Documento técnico de XionIA — Fernando Martin Lopez, Corrientes, Argentina *
