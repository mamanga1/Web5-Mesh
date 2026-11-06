# DerinKuyu-SSH — Administración remota por DID (comando `remote`)

**Referencia técnica completa** · `xionia-kernel` v9.2 · noviembre de 2026
**Línea de producto:** DerinKuyu-SSH (gratuita) · **Diseño de referencia:** MEMO-INTERNO-TUNEL-SSH-POR-DID, directivas D1–D10 · **Evidencia:** `captura-test.txt` (test real entre dos nodos, 5–6/11/2026)

---

## 1. Qué es

`remote` es el comando de la consola DerinKuyu que abre una **sesión SSH real sobre el host de otro nodo de la red** — el Linux donde corre esa consola, no su jaula — viajando por la sesión Noise directa que ya existe entre ambos kernels.

Lo que lo distingue de cualquier acceso SSH/VPN tradicional:

| Propiedad | Cómo se cumple |
|---|---|
| **0 puertos nuevos** | El `sshd` de sesión no escucha en ningún puerto de ninguna interfaz: sirve UNA única conexión por stdin/stdout (`sshd -i`), enganchado al túnel. No hay daemon expuesto ni puerto que escanear. |
| **0 credenciales persistentes** | La consola origen genera un par Ed25519 **en memoria** por sesión; la clave pública viaja cifrada dentro del pedido. Cuando la sesión cae, la llave deja de existir. No hay `.pem` que robar, no hay password que rotar. |
| **Solo directo, jamás relay** | El túnel viaja por la sesión Noise entre los dos kernels. Si no hay camino directo, no hay túnel: la sesión administrativa no pasa nunca por un tercero. |
| **Todo con recibo** | Cada sesión — y cada intento, incluidos los rechazados — queda registrado y firmado Ed25519 por el nodo: qué DID entró, qué clave usó, cuándo abrió, cuándo cerró, cuántos bytes cruzaron. |
| **Opt-in del dueño** | La capa host está **apagada por defecto**. La habilita el dueño del nodo objetivo, DID por DID, en un catálogo firmado. |

**La doctrina en una frase:** la llave vive exactamente lo que vive la sesión. El VPN les da la red entera; esto les da el brazo exacto, con recibo firmado.

---

## 2. El principio: una consola, un kernel, una identidad, un transporte

El comando no agrega infraestructura: reutiliza la que ya funciona. La misma consola (DerinKuyu), el mismo kernel (XionIA), la misma identidad (el DID — la única credencial que viaja) y el mismo transporte (la sesión Noise existente, el único canal). Todo lo nuevo vive en `cmd/derinkuyu/remote.go` (+ tres anclas en `main.go`); **`pkg/` quedó intacto**. Cualquier diseño que cree un segundo ejemplar de cualquiera de esas cuatro unidades está mal por definición.

---

## 3. El modelo de capas

| Capa | Qué es | Estado |
|---|---|---|
| **1 · Jaula del nodo** | El sistema de archivos sellado de la consola remota | Ya existía (comandos de jaula + XFER) |
| **2 · Host del nodo** | El Linux real donde corre esa consola | **v9.2 — este documento** |
| **3 · LAN del nodo** | Otros hosts de su red local, por alias | Fase 3 (bastión por catálogo) |

**Regla de oro:** cada capa se abre únicamente con autorización explícita del dueño de esa capa, y toda sesión abierta queda registrada y firmada.

---

## 4. Las tres puertas (se atraviesan en orden, ninguna sustituye a la otra)

| Puerta | Quién decide | Qué controla |
|---|---|---|
| **1 · ACL del nodo** | El dueño, con `acl import` | QUIÉN puede hablar con el nodo — la misma lista del chat y de XFER |
| **2 · Catálogo ROOT_ADMIN** | El dueño, con `remote allow` / `remote revoke` | QUÉ puede hacer: la capa host, apagada por defecto, habilitada DID por DID en `remote-admin.json` (firmado Ed25519) |
| **3 · Clave efímera** | El protocolo, solo | DÓNDE y POR CUÁNTO: una llave nueva por sesión que muere con ella |

Un DID puede estar en la ACL (chatea, transfiere archivos) y aun así no tener capa host. **Estar bienvenido no es tener las llaves de la casa.**

---

## 5. La doble muralla

Noise por fuera (transporte entre kernels, autenticado por DID), SSH por dentro (sesión de trabajo). El kernel del nodo destino puentea bytes de SSH cifrados: **puede firmar el recibo — quién entró, cuándo, cuánto tráfico — sin poder leer el contenido de la sesión.** Auditoría sin curiosidad: así duerme el dueño del nodo.

---

## 6. Requisitos

**En ambos nodos:**

- Consola `derinkuyu` v9.2 o superior, conectada al mismo faro.
- ACL cruzada (`acl import` del DID del par, con sus claves públicas) y alias opcional (`alias add`).
- Camino directo ICE entre los pares (el túnel es solo directo).

**En el nodo OBJETIVO (el que recibe el SSH), además:**

- OpenSSH instalado — `sshd` de stock: el de sesión lo lanza el kernel bajo demanda y **el `sshd_config` del host no se toca**.
- Si se va a operar como root, la consola corre **con privilegios** (ver §13: es la encarnación del opt-in del dueño):

```bash
sudo env HOME=/home/tuusuario ./derinkuyu -faro <tu-faro>:443
```

**En el nodo OPERADOR:** nada especial — corre sin privilegios, como un usuario más.

---

## 7. Referencia de comandos

| Comando | Qué hace |
|---|---|
| `remote <alias\|DID> [usuario]` | Abrir túnel SSH hacia ese nodo (usuario por defecto: `root`) |
| `remote allow <alias\|DID>` | **Solo el dueño**: habilitar ROOT_ADMIN para ese DID — el catálogo se re-firma |
| `remote revoke <alias\|DID>` | **Solo el dueño**: revocar ROOT_ADMIN — los túneles ya abiertos siguen hasta cerrarse |
| `remote admins` | Ver el catálogo sellado (firma verificada) |
| `remote list` | Túneles vivos |
| `remote close <id>` | Cerrar un túnel |
| `remote log [n]` | Registro firmado de sesiones e intentos (últimos n) |
| `remote log verify` | Re-verificar TODAS las firmas del registro |

---

## 8. Protocolo del túnel

Los frames viajan por el mismo gancho de mensajería del kernel (demux por prefijo, igual que XFER y ping):

| Frame | Significado |
|---|---|
| `TUNREQ` | Pedido de túnel con la clave pública efímera (cifrada en el canal) |
| `TUNOK` | Aceptado: el nodo destino ya ejecutó su `sshd` de sesión |
| `TUNNO` | Rechazado (ej. `NOT_ADMIN`: el dueño no habilitó ROOT_ADMIN) |
| `TUNEX` / `TUNOUT` | Datos del túnel de ida / vuelta (frames numerados con ACK) |
| `TUNBYE` | Cierre limpio del túnel |

**Del lado operador:** la consola puentea el túnel a `127.0.0.1` en un **puerto local efímero** y el cliente `ssh` local se conecta ahí — el operador tiene la experiencia OpenSSH completa (shell, pty, sftp) dentro de su consola.

**Del lado destino:** si ACL y catálogo autorizan, el nodo crea un **directorio de sesión efímero** bajo `/run/xionia` (un `authorized_keys` que nace y muere con la sesión), lanza `sshd -i` (una conexión por stdin/stdout, sin escuchar puerto alguno) y puentea el túnel con ese proceso.

---

## 9. Cómo se construyó y probó (el registro del desarrollo)

1. **Diseño:** MEMO-INTERNO-TUNEL-SSH-POR-DID fijó las directivas D1–D10 (kernel intacto, adaptador de flujo sobre frames numerados con ACK, OpenSSH de stock en ambos extremos, clave efímera por sesión, cero puertos nuevos, registro firmado, prohibido el túnel `ip:puerto` arbitrario).
2. **Implementación:** `cmd/derinkuyu/remote.go` + parche de tres anclas sobre `main.go` (`parche-main.go.diff`), aplicado limpio con `git apply`.
3. **Calidad:** `go vet` limpio · `go test ./cmd/derinkuyu/ -race -count=1` OK (detector de carreras limpio).
4. **Revisión externa:** tres condiciones exigidas por la revisión — ROOT_ADMIN con configuración única del dueño, clave efímera por sesión, autorización explícita + auditoría firmada — verificadas las tres.
5. **Test en vivo (5–6/11/2026):** dos nodos reales — xeon (`mamanga@node-379-core`, linux/amd64, operador) ↔ jubilarte (`alarm@alarm`, Arch ARM, objetivo) — al faro `190.220.45.26:443`, túnel directo. El test incluyó dos intentos fallidos deliberadamente diagnosticados (ver §9.3), la corrección de una línea, la sesión root exitosa, el cierre, la higiene, la auditoría y la revocación. Transcripción textual: `captura-test.txt`.

### 9.1 Identidades y ACL cruzada (puerta 1)

```text
# en cada nodo, importar el DID del otro con sus claves públicas:
xion@DerinKuyu:~$ acl import did:maia:<DID-del-par> <PubEd> <PubX>
✅ Peer importado: did:maia:…

# alias opcional para operar por nombre:
xion@DerinKuyu:~$ alias add jubilarte did:maia:<DID-del-par>
✅ Alias 'jubilarte' → did:maia:…
```

### 9.2 El dueño habilita (puerta 2)

```text
# en el nodo OBJETIVO:
xion@DerinKuyu:~$ remote allow xeon
✅ ROOT_ADMIN habilitado para xeon (did:maia:4…FYH1hCrx) — catálogo re-firmado

xion@DerinKuyu:~$ remote admins
Catálogo ROOT_ADMIN (remote-admin.json) — firma verificada ✔
  xeon  did:maia:4…FYH1hCrx
```

### 9.3 El operador conecta (puerta 3)

```text
# en el nodo OPERADOR:
xion@DerinKuyu:~$ remote jubilarte root
🕳 abriendo túnel SSH hacia jubilarte (solo directo, jamás relay)...
✅ Túnel listo — el nodo remoto ejecutó su openssh de sesión (puerto local 35563).
   Clave efímera SHA256:… · host de sesión SHA256:…. Exit para volver.

[root@alarm ~]# whoami
root
```

**La lección del test (dos intentos fallidos documentados):** con la consola del objetivo corriendo **sin privilegios**, el handshake se completa (~3 KB por lado) y la autenticación cae en `Permission denied (publickey)`: un `sshd -i` nacido de un proceso sin privilegios no puede autenticar a root. Determinístico. La corrección es una línea — relanzar en el objetivo conservando `HOME` (misma identidad, misma ACL, mismo catálogo):

```bash
sudo env HOME=/home/alarm ./derinkuyu -faro 190.220.45.26:443
```

Esa línea **es** el opt-in del dueño encarnado: el nodo que ofrece capa host corre con los privilegios que su dueño decide darle. Y cada intento fallido quedó igualmente registrado y firmado.

### 9.4 Cierre limpio

```text
[root@alarm ~]# exit
logout
Connection to 127.0.0.1 closed.

sesión ssh cerrada limpia
🔒 túnel ffa638dd… cerrado (↑31.8KB ↓29.8KB, 4m36s)
```

La contabilidad es espejada: ambos nodos reportan los mismos bytes en ambas direcciones.

### 9.5 Higiene post-sesión

```bash
pgrep -af sshd       # → solo el listener del sistema y las sesiones propias del operador.
                     #    CERO 'sshd -i' huérfanos: el de sesión murió con el túnel.
ls -la /run/xionia   # → no existe: el directorio efímero desapareció al cerrar
```

```text
xion@DerinKuyu:~$ remote list
sin túneles vivos
```

### 9.6 Auditoría firmada

```text
xion@DerinKuyu:~$ remote log 10
  2026-11-06 02:20:04  ABIERTO  xeon  clave SHA256:Gkb1gm… · host SHA256:fcAVmc…
  2026-11-06 02:24:39  CERRADO  xeon  ↑31.8KB ↓29.8KB  276s (EOF)

xion@DerinKuyu:~$ remote log verify
✔ 6 registros, TODAS las firmas verificadas (Ed25519 del nodo)
```

### 9.7 Revocación (el portón del dueño, en ambas direcciones)

```text
# en el objetivo:
xion@DerinKuyu:~$ remote revoke xeon
✅ ROOT_ADMIN revocado a xeon — catálogo re-firmado

# reintento desde el operador:
❌ remote: el nodo remoto rechazó: NOT_ADMIN
   El dueño de ese nodo tiene que habilitarte: remote allow <tu-DID>
```

Túnel rechazado de 0 B / 0 s. La revocación vive en el nodo, no arriba: no hay registro central que purgar ni permiso que pedir.

---

## 10. Qué muere y qué queda con cada sesión

| Muere con la sesión | Queda (persistente) |
|---|---|
| Clave efímera (pública y privada) | El registro firmado de la sesión (recibo Ed25519) |
| Directorio `/run/xionia/sess-*` | El catálogo sellado `remote-admin.json` |
| Proceso `sshd -i` | La ACL del nodo |
| Puerto local del operador | La identidad DID de cada nodo |

No hay credencial que expirar porque no hay credencial persistente. Lo que se administra es la **autorización** (`allow`/`revoke`), no las llaves.

---

## 11. Auditoría firmada — alcance y límite honesto

Cada sesión de capa host registra: DID entrante, huella de la clave efímera, huella del host de sesión, hora de inicio y fin, bytes intercambiados, duración y causa de cierre. El registro se firma con la clave Ed25519 del nodo y `remote log verify` re-verifica todas las firmas. Los intentos fallidos y los rechazos también dejan recibo: **no existe sesión ni intento sin registro.**

Límite declarado: esto acredita la integridad criptográfica del registro. No constituye aún una auditoría independiente de la resistencia del almacenamiento frente a un administrador hostil del propio nodo — esa revisión está en la hoja de ruta.

---

## 12. Límites conocidos en v9.2

| Límite | Detalle | Camino |
|---|---|---|
| Sesión única por consola | Mientras operás el SSH, el REPL queda en la sesión | UX-1 (multitarea) en desarrollo |
| ROOT_ADMIN todo-o-nada | Habilita la capa host completa, DID por DID | Granularidad jaula/host/bastión — siguiente escalón del catálogo |
| Revocación no corta túneles abiertos | Los túneles ya abiertos siguen hasta cerrarse (propiedad documentada) | Corte inmediato, en evaluación |
| Doble NAT restrictivo | El test cubrió camino directo; doble NAT de ambos lados pendiente | Gate de campo en cola |
| Frontera SDK | `remote.go` consume `pkg/kernel` directo | Frontera SDK en v9.3 |
| Aceptación manual | Las puertas viven en la consola, con humano presente | Daemon desatendido (`xionia-node`) = Fase B |

---

## 13. Diagnóstico rápido

| Síntoma | Causa | Solución |
|---|---|---|
| `Permission denied (publickey)` tras handshake completo (~3 KB) | La consola del objetivo corre sin privilegios: un `sshd -i` sin setuid no autentica a root | Relanzar en el objetivo: `sudo env HOME=/home/<usuario> ./derinkuyu -faro <faro>:443` |
| `NOT_ADMIN` | El dueño no habilitó tu DID — o lo revocó | Que el dueño ejecute `remote allow <tu-DID>` en su consola |
| El pedido no avanza / no llega `TUNOK` | No hay camino directo entre los pares | Verificar faro y conectividad ICE: el túnel es solo directo, jamás relay |
| `sshd` no encontrado en el objetivo | OpenSSH no instalado | Instalar `openssh` en el nodo objetivo (no se toca su configuración) |

---

## 14. La evidencia (números del test real, 5–6/11/2026)

| Métrica | Valor |
|---|---|
| Sesión root | 276 s (4m36s) · shell interactiva · egress verificado (`curl ifconfig.me`) |
| Tráfico del túnel | 61.6 KB (↑31.8 ↓29.8) · contabilidad espejada en ambos nodos |
| Claves efímeras | Una nueva por intento (cbAPQV…, sWswmK…, Gkb1gm…) — ninguna sobrevivió |
| Registro firmado | 6 registros Ed25519, TODAS las firmas verificadas — incluidos 2 intentos fallidos |
| Revocación | Rechazo `NOT_ADMIN` inmediato · túnel de 0 B / 0 s |
| Residuos | 0 puertos abiertos · 0 procesos huérfanos · 0 credenciales persistentes |
| Higiene | `/run/xionia` no existe tras el cierre · `remote list` → sin túneles vivos |

Transcripción completa de ambas consolas: `captura-test.txt`.

---

## 15. Qué NO es — y qué viene

**No es una VPN.** El túnel `ip:puerto` arbitrario está prohibido por diseño (directiva D7): la capa 3 (otros hosts de la LAN del nodo) se alcanzará únicamente por alias del catálogo sellado — la IP de destino nunca sale del nodo. No hay cliente SSH propio ni fork de `sshd`: OpenSSH de stock en ambos extremos.

**Hoja de ruta (en orden):** granularidad del catálogo → UX-1 multitarea de la consola → UX-2 multi-consola simultánea (N canales sobre la misma sesión Noise) → Fase B daemon desatendido → capa post-cuántica (X25519 + ML-KEM-768). La evolución no agrega privilegios ni toca el transporte: mejora la experiencia de operación sin sacrificar aislamiento, identidad, autorización ni trazabilidad.

---

*DerinKuyu-SSH v9.2 — línea gratuita del proyecto XionIA. XionIA aporta la infraestructura de confianza y comunicación; DerinKuyu aporta el control operativo.*
