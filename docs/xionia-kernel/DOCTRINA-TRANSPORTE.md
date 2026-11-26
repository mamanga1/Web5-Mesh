# Doctrina de Soberanía de Transporte — XionIA & Faro

> Ley de arquitectura del ecosistema **XionIA / SanJorge / Faro**.
> Rige a partir del parche **0032** en adelante. Se asienta junto al credo
> `docs/VOIP-Z0.md` como documento rector del repo. Quien toque
> `pkg/kernel`, `pkg/ice`, `pkg/session`, `pkg/voip` o `faro-server`
> jura primero sobre esta página.

---

## 1. Principio Rector

El **Faro** (`faro-server`) es un **punto de encuentro efímero y buzón mínimo**,
nunca una muleta de red ni un servidor de transporte masivo.

La soberanía de conectividad pertenece a los nodos: quien tiene IP pública,
ruta alcanzable, VPN o apertura NAT válida negocia y sostiene su túnel
**`DIRECT`** (Pion ICE + Noise IK/IKpsk2); quien opera detrás de un CGNAT
cerrado o con el equipo suspendido conserva el respaldo mínimo de texto por
**`relay`** hasta que su red permita enlace directo.

**Corolario del saludo:** una vez producido el saludo criptográfico, la
conexión debe ser directa si ambos nodos pueden alcanzarse. El que no pueda,
solo letritas.

---

## 2. Las Cinco Reglas de Hierro

### Regla 1 — DIRECT es para quienes tienen ruta; `relay` es solo buzón de letritas y encuentro

* El transporte de alta carga (VoIP, archivos/XFER y flujo continuo) **solo
  cursa por `DIRECT`** (`SendDirect` estricto, sin caída silenciosa a `relay`).
* Si un par no logra perforar NAT tras su ventana de negociación (`Dial`), no
  se insiste en bucle contra el Faro: se asume su condición de red y queda en
  `relay` únicamente para mensajería liviana y señalización puntual.

### Regla 2 — Cero tormentas y cero ping-pong en el Faro

* Un peer en estado `failed` o `signal-error` **no reintenta solo en segundo
  plano** ni dispara `ICE_INFO` periódicos.
* La señalización (`ICE_INFO`) solo se dispara por **evidencia real**:
  * arranque de nodo,
  * cambio efectivo de interfaz/IP (`watchRoute`),
  * despertar de ciclo de vida (`Wake` al volver a foreground),
  * envío explícito que encuentra el túnel caído (respetando siempre el
    enfriamiento de `15 s` por par).
* En régimen conectado (`DIRECT`), el tráfico de señalización hacia el Faro
  debe ser **cero** (solo `ANNOUNCE` de presencia dentro del TTL).

### Regla 3 — El directo se defiende solo sin molestar al Faro

* Una vez abierto el túnel UDP directo, su sostén y detección de muerte
  dependen exclusivamente del enlace extremo a extremo
  (`__PING__` / `__PONG__` cada `8 s` en `livenessLoop`), alimentando el
  mapping NAT sin intervención del servidor.
* El Faro **no es niñera ni intermediario de rutina**: un par en `DIRECT`
  consolidado no vuelve a consultarle ni le envía señal de sostén; la
  verificación de vida es entre los propios extremos.
* Mientras haya tráfico entrante o latido sano en foreground, **prohibido
  resetear o renegociar** por timers arbitrarios.

### Regla 4 — Respeto a la física del sistema operativo (sin andamios)

* En móviles (Android), cuando la app pasa a background o se suspende, el
  proceso se congela por diseño para no quemar batería ni imponer servicios
  invasivos. Se acepta la caída natural del mapping NAT durante el sueño.
* Al volver a primer plano (`resumed`), el reenganche golpea **primero
  directo**: se reusan los candidatos / IP:puerto ya conocidos del par y, si
  el socket o la ruta siguen vivos o alcanzables, se retoma `DIRECT` **sin
  tocar al Faro** (cero `ICE_INFO`, cero reintentos repetidos contra el
  servidor).
* Solo si la ruta está efectivamente muerta o la red cambió de verdad,
  `Wake` verifica que el Faro responda (`reconnectFaro` honesto) y dispara
  un **único** reenganche (`RedialAll` con `everConnected`) para recuperar
  `DIRECT` en pocos segundos sin requerir mensajes manuales.

### Regla 5 — Faro liviano, sin estado pesado y blindado en el borde

* `faro-server` opera en RAM pura con TTL corto (`90 s`), `Gate DID Ed25519`
  obligatorio y descarte silencioso (o muestreado) de paquetes no registrados.
* Si algún día se requiere mayor tasa de éxito tras NAT simétrico cerrado, la
  respuesta no es engordar el Faro ni meter cachés complejas: es desplegar
  nodos **TURN** voluntarios o que el usuario configure su propia ruta/VPN.

---

## 3. Directiva de Consolidación DIRECT (adenda GLM, mandato 0032)

### 3.1 — Dejen en paz al Faro una vez consolidado el DIRECT

* Cuando dos nodos ya se reconocieron y establecieron conexión `DIRECT`
  (Pion ICE + Noise/PQ), **no deben seguir yendo y viniendo al Faro ni
  consultarle más**.
* El túnel `DIRECT` se sostiene y se verifica **entre los propios extremos**
  (`__PING__` / `__PONG__` y tráfico directo); el Faro **no es niñera ni
  intermediario de rutina** una vez abierto el camino directo.

### 3.2 — Candidatos / IP:puerto entre pares y reenganche directo al salir del sueño

* Los nodos ya cruzaron identidad y saludo criptográfico y **conocen sus
  pares**: al negociar o reenganchar tras una pausa / minimizado /
  suspensión, priorizan el **intercambio directo de candidatos / IP:puerto**
  y el golpe directo entre pares, en vez de andar preguntándole de nuevo al
  Faro a cada rato si está o no está.
* Si el socket local o la ruta **aún sigue viva o alcanzable**, se reengancha
  directo **sin rebotes de `ICE_INFO` en el Faro** ni reintentos repetitivos
  contra el servidor.
* El Faro queda **estrictamente como punto de encuentro inicial / buzón
  mínimo** cuando no hay ruta directa abierta o cambia la red de forma
  efectiva: cero ping-pong, cero consultas repetidas al Faro en sesión ni al
  despertar si ya se tiene el par y el camino directo.

### 3.3 — Aplicación concreta sobre el código

* Sobre `pkg/ice/ice.go`, `pkg/kernel/kernel.go` y `faro-server`: cortar
  toda ida y vuelta innecesaria al Faro una vez en `DIRECT` o al volver de
  minimizado / suspensión.
* Código limpio, **sin timers periódicos de reintento ICE**, medible con
  **cero `ICE_INFO forward`** en régimen activo.

---

## 4. Mandato Operativo para Próximos Parches (GLM)

* **Prohibido** agregar timers periódicos de reintento ICE (`ticker` de
  auto-dial) en `pkg/ice` o `pkg/kernel`.
* **Prohibido** enrutar audio (VoIP) o transferencia de archivos por `relay`.
* **Prohibido** el ir y venir de rutina con el Faro de pares en `DIRECT`
  consolidado, en sesión o al volver de minimizado / suspensión, mientras el
  socket o la ruta sigan vivos: el reenganche golpea primero directo entre
  pares con los candidatos / IP:puerto ya conocidos, y solo toca al Faro si
  la ruta murió de verdad o la red cambió en forma efectiva.
* **Obligatorio** medir todo cambio contra `~/faro.log`: con pares en
  conversación activa en foreground, el contador de `ICE_INFO forward` debe
  mantenerse en **0** y el tráfico útil en el Faro en **0 bytes**.

---

*Asentada como ley de arquitectura. Toda divergencia debe declararse en el
parche y justificarse contra este documento.*
