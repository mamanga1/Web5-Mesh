# XionIA: Un Kernel en User-Space

XionIA no es un messenger. No es una VPN. No es una aplicación de red tradicional.
Es un User-Space Kernel que gestiona hardware virtual distribuido: red overlay,
identidad criptográfica, túneles cifrados. Desde la v9.2, además, curte cada
sesión con dos alfabetos (X25519 + ML-KEM-768) y abre el host remoto por DID
— sin exponer un solo puerto.

**Versión de este documento:** kernel v9.2.4 (parches 0016/0017 en producción) · release v1.5 · noviembre de 2026
**Fuente:** github.com/mamanga1/xionia-kernel · Releases: github.com/mamanga1/xionia-xtp

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
| Red | TCP/IP stack | ICE sobre UDP + Noise IKpsk2 (híbrido PQ) | pion/ice v4 + crypto/mlkem |
| Identidad | UID/GID | DID (did:maia:xxxxx) | Ed25519 + X25519 + ML-KEM-768 (node.key v2) |
| Seguridad | Permisos, SELinux | ACL, firmas, cifrado E2E | Código Go + syscalls |
| Drivers | Drivers de hardware | Faros (controladores de red virtual) | UDP/WSS |
| IPC | Pipes, sockets | Túneles cifrados entre nodos | Channels + goroutines |
| Administración remota | sshd en el puerto 22 | `remote`: SSH por DID (opt-in, 0 puertos) | sshd -i por stdin/stdout |
| Ejecución de código | ✅ Sí (arbitraria) | ❌ No (solo primitivas) | ✅ Heredado de Go runtime |

---

## Estados Posibles en XionIA (FSM)

### Operaciones válidas del kernel (v9.2.4, producción):

```
├── Recibir datagrama UDP (faro o túnel ICE)
├── Verificar firma Ed25519 del ANNOUNCE (gate del faro)
├── Deserializar ICE_INFO (base64 opaco al faro)
├── Handshake Noise IK / IKpsk2 (initiator = DID lexicográfico
│   menor; la rama clásica/PQ la decide el LARGO del msg1)
├── Decapsular ML-KEM-768 y curtir el PSK con dos alfabetos
├── Verificar PeerStatic contra la ACL (impostor = descarte)
├── Cifrar/descifrar ChaCha20-Poly1305 (anti-replay, ventana 64)
├── Dispatch por discriminador (0x01 handshake · 0x02 datos — congelado)
├── Verificar catálogo ROOT_ADMIN firmado y puentear SSH por
│   DID (opt-in del dueño — la capa host nace apagada)
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
├── Evaluar expresiones dinámicas
├── Negociar versiones o downgrades en el wire (no hay
│   diálogo: la rama la decide el largo del datagrama,
│   que es geometría, no conversación)
└── Exponer un puerto de administración (remote sirve el
    sshd por stdin/stdout del túnel — no hay socket que
    escanear)
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
│  ├── Session Mgr (Noise IK / IKpsk2)        │
│  ├── Capa PQ (pkg/pq — X25519 + ML-KEM-768) │
│  ├── Identidad (DID · Ed25519 · X25519 ·    │
│  │   ML-KEM — node.key v2, DID inmutable)   │
│  ├── ACL (control de acceso · 4ª clave KEM) │
│  ├── Auto-heal (ruta · liveness · decrypt)  │
│  ├── Remote (SSH por DID — opt-in sellado)  │
│  └── SendDirect (VoIP/XFER — sin relay)     │
├─────────────────────────────────────────────┤
│  HARDWARE VIRTUAL (La Red)                  │
│  ├── Faro (casamentero: STUN · blind relay  │
│  │   · amnésico — jamás decide)             │
│  ├── Túneles ICE sobre UDP (buses)          │
│  └── DIDs (IDs de hardware virtual)         │
├─────────────────────────────────────────────┤
│  HOST OS (Linux · Android · Termux)         │
│  └── Provee: sockets UDP, filesystem        │
│  └── Superficie de ataque heredada          │
└─────────────────────────────────────────────┘
```

> **Nota de producción (v9.2.4):** el Faro es signaling puro — presenta pares, reenvía
> ICE_INFO mientras nadie tiene túnel, re-transmite texto cifrado (blind relay)
> como red de seguridad, y atiende además la trampa UDP 443 (el puerto del HTTPS,
> para las murallas de los carriers). Jamás toca contenido legible: lo que
> transporta es ciphertext. Y es **casamentero, no árbitro**: verifica la firma
> Ed25519 del DID y presenta; jamás decide quién habla ni cuántas veces — la cuota
> `MaxAddrsPerDID` que expulsaba teléfonos en roaming WiFi↔LTE murió con el parche
> 0016, y desde entonces la firma es el único pase. Con el túnel establecido, el
> faro desaparece del camino — verificado en producción: 20 minutos de
> conversación directa con el faro caído.

---

## La Capa Post-Cuántica (v9.2): Dos Alfabetos por Sesión

> "Dos alfabetos por sesión: basta que uno siga vivo para que la conversación
> siga cerrada."

### La amenaza: harvest-now, decrypt-later (HNDL)

La auditoría HNDL de la pila dio un veredicto limpio: el único intercambio de
claves clásico que alimenta claves de sesión es el Noise IK de `pkg/session` —
todo lo demás ya es simétrico (PSK de señalización del enjambre, AEAD de datos,
túnel lab) y por lo tanto PQ-neutro (Grover deja ChaCha20 en ~128-bit de
trabajo). La deuda existía y tenía una sola dirección: **cada handshake IK
grabado hoy se abría el día que existiera un adversario cuántico con la
grabación.** Los archivos de hoy son los de dentro de diez años.

### El injerto: IKpsk2 híbrido con la stdlib

| Pieza | Qué es |
|---|---|
| ML-KEM-768 | KEM categoría 3, **FIPS 203 final**, stdlib `crypto/mlkem` — pure Go, CGO=0 |
| Patrón IKpsk2 | flynn/noise v1.1.0 — **la librería que ya estaba en go.mod**; cero dependencias nuevas |
| PSK híbrido | SHA-256 del secreto X25519 ‖ secreto ML-KEM ‖ transcripción completa (efímera, ct, pública destino) — 32 B exactos |
| ML-DSA-65 | co-firma de identidad (FIPS 204), stdlib `crypto/mldsa` — go.mod en 1.27 |

El cable casi no cambia:

| Mensaje | IK clásico | IKpsk2 híbrido |
|---|---|---|
| msg1 | 96 B, payload vacío | **1218 B** — payload = blob KEM de 1122 B (versión 1 B · epoch 1 B · efímera X25519 32 B · ct ML-KEM 1088 B) |
| msg2 | 48 B | 48 B (el PSK se mezcla al final — mismo 2-RTT de siempre) |
| Claves de sesión | curtidas con X25519 | curtidas con **X25519 + ML-KEM a la vez** |

1218 + 8 (UDP) + 20 (IP) = **1246 B < 1500**: un datagrama, sin fragmentación.

La FSM ni se enteró: la interface `Kernel` quedó intacta — el Manager arma el
híbrido desde la ACL y `AddPeerPQ` se consume por type assertion, degradando a
`AddPeer` en binarios viejos. El kernel no supo nunca de KEMs.

### Defensas por capas (verificadas con tests de vandalismo)

- **El blob KEM viaja DENTRO del AEAD del msg1**: el pasivo que graba ni
  siquiera ve el ciphertext KEM; mover un byte revienta el tag antes de decapsular.
- **Downgrade ruidoso por construcción**: no hay negociación en el wire — el
  respondedor decide la rama por el LARGO del msg1. Recortar el blob para
  "volver a legado" exige re-cifrar la estática bajo una efímera que el atacante
  no tiene: muerte ruidosa en el tag (`TestDowngradeImposible`).
- **Rechazo implícito de FIPS 203**: un ct bien formado pero adulterado no da
  error — da un secreto seudorrandom; el grito llega en el AEAD del msg2. El
  atacante no aprende ni por error cuál byte le pifió.
- **Híbrido siempre, jamás KEM solo (R1)**: si los retículos caen, queda X25519;
  si la curva cae, queda ML-KEM. Ningún fallo solo abre la sesión.

### Compatibilidad por adición (la red migra sin darse cuenta)

| Escenario | Resultado |
|---|---|
| ACL de 3 claves (la de toda la vida) | IK clásico de 96 B, byte a byte como hoy |
| ACL de 4 claves (share v2) | IKpsk2, msg1 de 1218 B — el híbrido se activa por ACL, no por versión |
| Nodo parcheado ↔ nodo viejo | Hablan IK clásico: cero rotura durante la migración |
| node.key v1 → v2 | Migración automática al cargar: la semilla ML-KEM se agrega, **el DID no cambia** |
| Re-import de 3 claves sobre un peer PQ | Degrada a legado a propósito — la importación es la fuente de verdad |

### La Huella PQ (SAS)

`SHA-256(PubX ‖ PubKem)` truncada a 8 B en 4 grupos — 16 caracteres legibles
(p. ej. `6e55-6e43-a7f6-ba2c`). Es un SAS: por diseño puede viajar por cualquier
canal y se verifica dictando por voz, humano a humano. El cierre anti-MITM del
primer contacto, sin infraestructura.

### El presupuesto (medido, no estimado)

| Operación | Tiempo |
|---|---|
| Handshake IK clásico | 561 µs |
| Handshake híbrido completo | 819 µs |
| **Premium PQ por sesión** | **~257 µs y +1122 B — UNA vez por sesión** |

### Qué la capa PQ NO protege (honestidad de charter)

El node.key en disco (0600), el PSK de señalización y la distribución de
confianza (ACL/share) siguen siendo la superficie clásica de siempre — y así se
publica. PQ donde decimos que es PQ: el KEX de sesión. La co-firma ML-DSA-65
(3376 B junto a la Ed25519, ambas deterministas) queda para documentos de
identidad — DID-docs del faro, entradas de registro, anuncios de share v2 —
JAMÁS para frames de datos: 3,3 KB por paquete es un impuesto estúpido.

---

## Remote: El Brazo Exacto (SSH por DID, v9.2)

> "El VPN les da la red entera; esto les da el brazo exacto, con recibo firmado.
> La llave vive exactamente lo que vive la sesión."

`remote` abre una sesión SSH real sobre el host de otro nodo — el Linux donde
corre esa consola, no su jaula — viajando por la sesión Noise que ya existe
entre ambos kernels. Cero infraestructura nueva: la misma consola, el mismo
kernel, la misma identidad, el mismo transporte. Todo lo nuevo vive en
`cmd/derinkuyu/remote.go`; `pkg/` quedó intacto.

| Propiedad | Cómo se cumple |
|---|---|
| **0 puertos nuevos** | El sshd de sesión no escucha en ninguna interfaz: UNA conexión por stdin/stdout (`sshd -i`) enganchada al túnel. No hay puerto que escanear, y el `sshd_config` del host no se toca. |
| **0 credenciales persistentes** | Par Ed25519 generado en memoria por sesión; muere con ella. No hay `.pem` que robar ni password que rotar. |
| **Solo directo, jamás relay** | Si no hay camino ICE directo, no hay túnel: la sesión administrativa jamás pasa por un tercero. |
| **Todo con recibo** | Cada sesión — y cada intento, incluidos los rechazados — queda firmada Ed25519 por el nodo: qué DID entró, con qué clave, cuándo abrió, cuándo cerró, cuántos bytes cruzaron. |
| **Opt-in del dueño** | La capa host nace apagada. La habilita el dueño, DID por DID, en un catálogo firmado (`remote allow` / `remote revoke`). |

Las tres puertas se atraviesan en orden, ninguna sustituye a la otra: **ACL**
(quién puede hablar — la misma del chat y de XFER), **catálogo ROOT_ADMIN
firmado** (qué puede hacer — la capa host, `remote-admin.json` sellado por el
dueño) y **clave efímera** (dónde y por cuánto — la decide el protocolo, no hay
quien la copie).

La doble muralla: Noise por fuera (transporte entre kernels, autenticado por
DID), SSH por dentro (sesión de trabajo). El kernel del nodo destino puentea
bytes de SSH cifrados: **firma el recibo sin poder leer la sesión — auditoría
sin curiosidad.** Así duerme el dueño del nodo.

> "Estar bienvenido no es tener las llaves de la casa."

Probado en campo: túneles tvbox↔xeon detrás de WARP; contabilidad de bytes
espejada en ambos extremos (36,0 KB: 16,2 ↑ / 19,8 ↓) con cierre limpio —
viajando por canal híbrido PQ desde la v9.2.

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

> Nota v9.2: la industria converge en el mismo híbrido — TLS 1.3 está adoptando
> X25519MLKEM768 como grupo por defecto. XionIA no inventó la receta: la copió
> con los ingredientes de la stdlib de Go y la injertó en el Noise IK que ya
> tenía — sin protocolos nuevos, sin dependencias nuevas, sin puertos nuevos.

---

## El Principio de Seguridad

> "No es invulnerabilidad por perfección. Es reducción de superficie de ataque
> por diseño arquitectónico basado en teoría de la computabilidad."

XionIA no es invulnerable porque sea perfecto. Es **diferentemente seguro**
porque el protocolo es una FSM simple, y las clases de vulnerabilidades que
afectan a sistemas Turing-completos no tienen "dónde" ocurrir en esa FSM.

La implementación en Go debe auditarse por separado: dependencias, concurrencia,
manejo de memoria, superficie de ataque del runtime — y desde la v9.2, la
fidelidad de `pkg/pq` a FIPS 203/204 (vectores oficiales incluidos).

---

## La Prueba de Fuego

| Escenario | WhatsApp/Signal | XionIA |
|---|---|---|
| Hackeás el servidor central | ✅ Control total: mensajes, metadatos, contactos | ❌ No hay servidor central |
| Hackeás un nodo | N/A | Solo tráfico cifrado sin la clave privada del usuario |
| Secuestrás el faro | N/A | RAM vacía: amnésico, zero logs. El círculo ya establecido sigue hablando directo sin él |
| Le cambiás las reglas al faro (cuotas, permisos) | N/A | No hay reglas que cambiar: el faro casamentero no decide — la firma es el único pase |
| Grabás todo el tráfico de hoy y esperás tu computadora cuántica (HNDL) | Los ECDH clásicos grabados se abren el día-C | ❌ Hay que romper X25519 **Y** ML-KEM-768 del mismo saludo — y el blob KEM ni siquiera viaja en claro |
| Queres el SSH del nodo ajeno | Puerto que escanear, credenciales que robar | ❌ No hay puerto ni `.pem`: sshd -i por el túnel, llave efímera que muere con la sesión, opt-in del dueño |

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
   transiciones de estado son seguras. La v9.2 agregó una transición nueva (la
   rama por largo del msg1: 96 B legado / 1218 B híbrido): probar que es una
   partición sin ambigüedad del espacio de mensajes es parte del trabajo.
2. **Pruebas de resistencia** — fuzzing de protocolo para forzar estados
   inválidos. Los tests de vandalismo ya verdes marcan el molde:
   `TestDowngradeImposible`, `TestPrologueAmarraDIDs`,
   `TestImpersonacionCaeEnACL`, `TestRechazoImplicitoTamperCT`.
3. **Análisis de side-channels** — timing attacks, traffic analysis.

### Auditoría de la Implementación (Go):

1. **Stack criptográfico** — Ed25519, X25519, ChaCha20-Poly1305, Noise IK/IKpsk2
   (flynn/noise y pion/ice, maduras y auditadas por la comunidad) + desde la v9.2
   `crypto/mlkem` y `crypto/mldsa` de la stdlib (FIPS 203/204). Regla R4: nada
   de libs PQ de terceros mientras esté en la stdlib.
2. **Dependencias** — go.mod, vulnerabilidades conocidas en librerías.
3. **Concurrencia** — race conditions. Regla interna del proyecto: cada cleanup
   solo limpia su propio estado, locks de lectura cortos, gatillos de a dos.
   GATE de la casa: `go build ./... && go vet ./... && go test -race -count=1 ./...`
   — 16/16 paquetes en verde, obligatorio antes de cualquier release. Parches
   numerados con sha256 verificado en pre-vuelo.
4. **Superficie de ataque del runtime** — Go + host OS. Regla operativa: el faro
   de producción compila siempre su propio binario (pull → build → restart); los
   binarios de release pública jamás se deployan en producción.

> **El objetivo no es encontrar "vulnerabilidades tipo CVE" en el protocolo
> (que no existen por diseño), sino verificar que la implementación es fiel
> a la especificación formal de la FSM, y que la superficie de ataque del
> runtime está minimizada.**

---

## Mitigaciones de Implementación

### Implementadas y validadas en producción (v9.2.4):

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
| **Handshake híbrido PQ** | IKpsk2: X25519 + ML-KEM-768 (FIPS 203) alimentando el PSK — dos alfabetos por sesión, premium ~257 µs | ✅ Producción (v9.2) |
| **Blob KEM cifrado en el wire** | Viaja dentro del AEAD del msg1 — el pasivo ni ve el ciphertext KEM | ✅ Producción |
| **Downgrade ruidoso** | La rama legado/PQ la decide el largo del msg1; recortarla muere en el tag AEAD | ✅ Producción |
| **Huella PQ (SAS)** | 16 caracteres verificables por voz — cierre anti-MITM del primer contacto | ✅ Producción |
| **Faro casamentero (0016)** | Sin cuotas por DID: rebind que reemplaza — la firma es el único pase | ✅ Producción |
| **Antiflujo (0017)** | Cooldown de 15 s + puerto local estable + freno de logs + ConnectFaro idempotente | ✅ Producción |
| **Trampa UDP 443** | El faro atiende 443/udp (ritual setcap): evasión DPI — verificado sin saldo en Claro AR (puerta lateral frágil: no se promete como feature) | ✅ Producción |
| **Remote sellado** | Capa host apagada por defecto; catálogo ROOT_ADMIN firmado; clave efímera por sesión; registro de sesiones firmado | ✅ Producción |

### Horizonte (planificadas):

| Mitigación | Descripción | Estado |
|---|---|---|
| Seccomp estricto | Filtro de syscalls: solo las necesarias | 🔄 Planificado |
| Memoria pinned | Claves fuera del GC (Mmap + Mlock) | 🔄 Planificado |
| Build reproducible | Cualquiera compila y obtiene el mismo hash — puerta trasera imposible | 🔄 Planificado |
| Firma de binarios | Releases firmados con Ed25519 del proyecto (hoy: SHA256SUMS verificado) | 🔄 Planificado |
| CI (GitHub Actions) | El GATE en cada push/PR — media hora, evita main roto para siempre | 🔄 Planificado |
| Auditoría formal | Verificación matemática de la FSM | 🔄 Horizonte |
| Enjambre CF (buzón v1) | Bootstrap por HTTPS/443 a dominios inocuos: anillo híbrido de dos tiers (Cloudflare + faros soberanos), POST ≤4 KB, lectura única, TTL 120 s — **CF nunca solo** | 🔄 Diseño aprobado |
| XIP-1 «tarjeta sellada» | Primer contacto sin canal privado: código corto humano + PAKE + blob efímero de un solo uso (TTL 15 min) en el buzón — el faro como caja de canje ciega | 🔄 Diseño |
| Voz Zappa (SanJorge 2.0) | Opus como un canal más de la sesión híbrida — cero servidores de voz; la media jamás pasa por el edge | 🔄 Planificado |
| SLH-DSA (ancla génesis) | Solo si algún día hace falta; exigiría `x/crypto/slhdsa` (terceros) — con nod explícito (R4) | 🔄 Reservado |

> **Nota sobre Forward Secrecy:** Noise IKpsk2 proporciona FS por handshake — cada
> sesión nueva nace con claves frescas de X25519 **y** un secreto ML-KEM que jamás
> viajó en claro. El rekeying *durante* la sesión fue evaluado y **eliminado a
> propósito**: la implementación unilateral mataba una dirección del túnel en
> silencio (la "siesta", curada en v8.9). Si algún día hace falta rekey
> intra-sesión, será negociado por protocolo — y mezclará PQ — contra un síntoma
> real, no por paranoia teórica.

---

## El Ecosistema que Nace del Kernel (diseño con nod)

Todo lo anterior es el kernel. Lo que sigue son los frentes que crecen de él,
cada uno con su documento de diseño y sin una línea de código antes del nod:

- **El enjambre de faros serverless** ataca la muralla con la infraestructura ya
  pagada: señalización de bootstrap que viaja como web normal (HTTPS/443 a
  dominios inocuos), invisible al DPI que caza UDP no-web. Anillo híbrido de dos
  tiers — 9 dominios Cloudflare (un Worker + Durable Objects, un buzón global
  por topic) y los faros físicos soberanos (KATI, las VPS Oracle). El contrato
  v1 es idéntico en ambos tiers: POST ≤4 KB, GET lee-una-vez-y-borra, TTL 120 s.
  Regla tallada: **CF nunca solo** — toda lista de puertas incluye al menos un
  faro soberano. Aditivo, jamás sustitutivo: el faro UDP queda intacto.
- **El superworker de Cloudflare** toma la cara pública (web, dossier, canjes,
  beacon) mientras KATI sigue siendo el faro de datos. Regla de oro: zero-knowledge
  — ninguna clave, semilla o material de sesión toca el edge. Si Cloudflare
  desaparece mañana, se reconstruye en cualquier VPS en una tarde; eso solo es
  verdad si nada sensible vive ahí.
- **XIP-1, la tarjeta sellada**, cierra el eslabón que faltaba del primer
  contacto: el que invita deja en el buzón un sobre cifrado que el faro no puede
  abrir, canjeable una única vez con un código corto dictado por cualquier canal
  (WhatsApp incluido: canal de códigos desechables, no de secretos). Cierre
  anti-MITM: Huella PQ por voz.
- **Zappa (SanJorge 2.0)** es la voz sobre la sesión híbrida: Opus como un canal
  más de la sesión que ya multiplexa chat, túneles y xfer. Cero servidores de
  voz; la media jamás pasa por el edge. Es también donde arranca la caja: el
  core queda gratis para siempre — el cobro empieza con lo nuevo.

---

> **"XionIA: Un User-Space Kernel que gestiona hardware virtual distribuido
> mediante un Protocolo de Estado Finito. Diferentemente seguro, porque las
> clases clásicas de vulnerabilidades no aplican a su arquitectura de protocolo.
> Y desde la v9.2, diferentemente difícil de abrir: aunque la física del futuro
> rompa la curva, la sesión sigue atada a los retículos."**

*Documento técnico de XionIA — Fernando Martin Lopez, Corrientes, Argentina* 🧉

*Actualizado a v9.2.4 (parches 0016/0017 en producción) · release v1.5 · noviembre de 2026*
