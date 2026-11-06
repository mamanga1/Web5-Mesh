# XionIA: Un Kernel en User-Space

XionIA no es un messenger. No es una VPN. No es una aplicación de red tradicional.
Es un User-Space Kernel que gestiona hardware virtual distribuido: red overlay,
identidad criptográfica, túneles cifrados.

---

## La Definición Correcta

XionIA es un Protocolo de Estado Finito (FSM) Distribuido que corre en user-space.
No reemplaza al kernel del host (Linux, Android), sino que corre encima de él,
abstrayendo la complejidad de la red y proporcionando un entorno soberano.

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
| Filesystem | ext4, NTFS, APFS | dataDir privado (0600) | os package de Go |
| Procesos | PIDs, scheduling | Sesiones Noise por peer | Goroutines |
| Red | TCP/IP stack | ICE sobre UDP + Noise IK | pion/ice v4 |
| Identidad | UID/GID | DID (did:maia:xxxxx) | Ed25519 + X25519 |
| Seguridad | Permisos, SELinux | ACL, firmas, cifrado E2E | Código Go + syscalls |
| Drivers | Drivers de hardware | Faros (controladores de red virtual) | UDP/WSS |
| IPC | Pipes, sockets | Túneles cifrados entre nodos | Channels + goroutines |
| Ejecución de código | ✅ Sí (arbitraria) | ❌ No (solo primitivas) | ✅ Heredado de Go runtime |

---

## Estados Posibles en XionIA (FSM)

### Operaciones válidas del kernel (v9.1, producción):

```
├── Recibir datagrama UDP (faro o túnel ICE)
├── Verificar firma Ed25519 del ANNOUNCE (gate del faro)
├── Deserializar ICE_INFO (base64 opaco al faro)
├── Handshake Noise IK (initiator = DID lexicográfico menor)
├── Verificar PeerStatic contra la ACL (impostor = descarte)
├── Cifrar/descifrar ChaCha20-Poly1305 (anti-replay, ventana 64)
├── Dispatch por discriminador (0x01 handshake · 0x02 datos — congelado)
├── Detectar muerte de sesión y renegociar (auto-heal)
└── Entregar payload a la aplicación (pkg/sdk)
```

### Operaciones imposibles en la FSM:

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
│  APLICACIONES                               │
│  (SanJorge APK · Derinkuyu · VoIP · XFER)   │
├─────────────────────────────────────────────┤
│  pkg/sdk — LA FRONTERA                      │
│  (las claves jamás cruzan, la UI jamás      │
│   importa el kernel directo)                │
├─────────────────────────────────────────────┤
│  XIONIA FSM (User-Space Kernel)             │
│  ├── ICE Agent (pion/ice v4 — túneles)      │
│  ├── Session Manager (Noise IK)             │
│  ├── Gestor de Identidad (DID · Ed25519)    │
│  ├── ACL (control de acceso)                │
│  ├── Auto-heal (ruta · liveness · decrypt)  │
│  └── SendDirect (VoIP/XFER — sin relay)     │
├─────────────────────────────────────────────┤
│  HARDWARE VIRTUAL (La Red)                  │
│  ├── Faro (signaling + STUN · amnésico)     │
│  ├── Túneles ICE sobre UDP (buses)          │
│  └── DIDs (IDs de hardware virtual)         │
├─────────────────────────────────────────────┤
│  HOST OS (Linux · Android · Termux)         │
│  └── Provee: sockets UDP, filesystem        │
│  └── Superficie de ataque heredada          │
└─────────────────────────────────────────────┘
```

> **Nota de producción:** el Faro es signaling puro — presenta pares, reenvía
> ICE_INFO mientras nadie tiene túnel, y re-transmite texto cifrado (blind relay)
> como red de seguridad. Jamás toca contenido legible: lo que transporta es
> ciphertext. Con el túnel establecido, el faro desaparece del camino — verificado
> en producción: 20 minutos de conversación directa con el faro caído.

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
| Secuestrás el faro | N/A | RAM vacía: amnésico, zero logs. El círculo ya establecido sigue hablando directo sin él |

> "Si hackeás un nodo de XionIA, solo has conseguido un nodo que ha dejado de
> obedecer el protocolo; no has comprometido el sistema, simplemente has causado
> una denegación de servicio local."

Eso es zero-knowledge por diseño arquitectónico. **Eso es XionIA.**

Y a diferencia de los sistemas que piden fe, esto se puede **verificar con
tcpdump**: el faro jamás ve QUÉ viaja, en directo el tráfico ni pasa por él,
el cable es ilegible, y las claves nunca salieron del dispositivo.

---

## Implicancias para la Auditoría de Seguridad

Cuando XionIA sea auditado, el enfoque será dual:

### Auditoría del Protocolo (FSM):

1. **Verificación formal de la FSM** — demostrar matemáticamente que todas las
   transiciones de estado son seguras.
2. **Pruebas de resistencia** — fuzzing de protocolo para forzar estados inválidos.
3. **Análisis de side-channels** — timing attacks, traffic analysis.

### Auditoría de la Implementación (Go):

1. **Stack criptográfico** — Ed25519, X25519, ChaCha20-Poly1305, Noise IK
   (librerías flynn/noise y pion/ice — maduras y auditadas por la comunidad).
2. **Dependencias** — go.mod, vulnerabilidades conocidas en librerías.
3. **Concurrencia** — race conditions. Regla interna del proyecto: cada cleanup
   solo limpia su propio estado, locks de lectura cortos, gatillos de a dos.
4. **Superficie de ataque del runtime** — Go + host OS.

> **El objetivo no es encontrar "vulnerabilidades tipo CVE" en el protocolo
> (que no existen por diseño), sino verificar que la implementación es fiel
> a la especificación formal de la FSM, y que la superficie de ataque del
> runtime está minimizada.**

---

## Mitigaciones de Implementación

### Implementadas y validadas en producción (v9.1):

| Mitigación | Descripción | Estado |
|---|---|---|
| Gate DID en el faro | ANNOUNCE con firma Ed25519 obligatoria antes de procesar | ✅ Producción |
| Zero logs en el faro | Amnésico: RAM only, secuestrarlo = llevarte RAM vacía | ✅ Producción |
| Padding anti-DPI | Tamaño aleatorio 50–200 bytes en mensajes al faro | ✅ Producción |
| Permisos 0600 | Claves y datos privados del nodo | ✅ Producción |
| Anti-suplantación | PeerStatic del handshake vs. ACL — impostor = sesión descartada | ✅ Producción |
| Anti-replay | Ventana deslizante de 64 secuencias por sesión | ✅ Producción |
| Auto-cura de sesión | 3 descifrados fallados → renegociación (~3s) — la cura de la siesta | ✅ Producción |
| Liveness invisible | `__PING__`/`__PONG__` cada 8s bajo el kernel — alimenta el NAT | ✅ Producción |
| Watchdog de ruta | Cambio de red (WiFi↔datos) detectado en 3s | ✅ Producción |
| Net artesanal Android | probeNet sin netlink: la IP se descubre con DialUDP al faro | ✅ Producción |
| STUN soberano | Derivado del propio faro — jamás terceros hardcodeados | ✅ Producción |
| Discriminador congelado | 0x01 handshake · 0x02 datos — la regla que no crece | ✅ Producción |
| Sin rekey unilateral | La clave vive lo que vive la sesión (lección de la siesta) | ✅ Producción |

### Horizonte (planificadas):

| Mitigación | Descripción | Estado |
|---|---|---|
| Seccomp estricto | Filtro de syscalls: solo las necesarias | 🔄 Planificado |
| Memoria pinned | Claves fuera del GC (Mmap + Mlock) | 🔄 Planificado |
| Build reproducible | Cualquiera compila y obtiene el mismo hash — puerta trasera imposible | 🔄 Planificado |
| Firma de binarios | Releases firmados con Ed25519 del proyecto | 🔄 Planificado |
| Auditoría formal | Verificación matemática de la FSM | 🔄 Horizonte |

> **Nota sobre Forward Secrecy:** Noise IK proporciona FS por handshake — cada
> sesión nueva nace con claves frescas de X25519. El rekeying *durante* la sesión
> fue evaluado y **eliminado a propósito**: la implementación unilateral mataba
> una dirección del túnel en silencio (la "siesta", curada en v8.9). Si algún día
> hace falta rekey intra-sesión, será negociado por protocolo — contra un
> síntoma real, no por paranoia teórica.

---

> **"XionIA: Un User-Space Kernel que gestiona hardware virtual distribuido
> mediante un Protocolo de Estado Finito. Diferentemente seguro, porque las
> clases clásicas de vulnerabilidades no aplican a su arquitectura de protocolo."**

*Documento técnico de XionIA — Fernando Martin Lopez, Corrientes, Argentina* 🧉
