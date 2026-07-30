# 🗼 Faros — Infraestructura de Relay Ciego

**XionIA Faraday** — Los faros son la infraestructura de enrutamiento de
la red. Son relays ciegos: enrutan paquetes cifrados sin descifrar,
almacenar, ni inspeccionar contenido.

---

## ¿Qué es un Faro?

Un faro es un servidor que:

- **Recibe** paquetes UDP/WSS de nodos XionIA
- **Registra** la asociación DID → dirección IP:puerto (en RAM)
- **Reenvía** paquetes al nodo destino según el DID
- **No descifra** nada (zero-knowledge)
- **No almacena** nada persistente (registry en RAM)
- **No inspecciona** contenido
- **Gate DID**: autenticación Ed25519 obligatoria (handshake antes de operar)
- **Cross-relay UDP↔WSS**: un nodo en UDP puede comunicarse con uno en WSS
- **TTL registry**: expira nodos zombies a 90s sin ANNOUNCE
- **Verificación de firma del ANNOUNCE**: anti-spoofing de identidad
- **Gate logging**: loguea rechazos del Gate (diagnóstico de NAT/roaming)

### Handshake (Gate DID)

Antes de cualquier comando, el nodo debe autenticarse:

```
Nodo → Faro: {"did":"did:maia:...","pub":"...","nonce":"...","ts":...,"sig":"..."}
Faro → Nodo: {"ack":"ok","did":"did:maia:...","ts":...,"nodes":N}
```

Sin handshake válido, el faro descarta todo mensaje en silencio.

---

## Cualquiera puede levantar un faro

**No requiere permiso, registro, ni aprobación.** El código del faro es
público y está en el repositorio. Si tenés una máquina con un puerto UDP
abierto, tenés un faro. Así de simple.

```bash
# 3 comandos y tenés un faro funcionando
git clone https://github.com/mamanga1/Web5-Mesh
cd Web5-Mesh && go build -o faro ./cmd/faro/
./faro
```

Output:

```
🚀 Iniciando Faro Dual (UDP + WebSocket)
   Gate DID: solo nodos con did:maia válido
   Cross-relay: UDP↔WSS activo
   Registry TTL: 90s sin ANNOUNCE → expira
🛡️ [FARO-UDP] Relay Ciego en 0.0.0.0:54321 (Gate DID activo)
🛡️ [FARO-WS] WebSocket TLS en 0.0.0.0:443 (Gate DID activo)
```

Los nodos se conectan a tu faro con:

```bash
xion@nodo:~$ faro set <TU_IP>:54321
✅ Faro configurado: <TU_IP>:54321
```

No hay un "registro central de faros". No hay autoridad que apruebe.
La soberanía es eso: cualquiera puede ser infraestructura.

---

## Faros Públicos (lista comunitaria)

Esta tabla es solo una **referencia de faros conocidos**, no un gate de
entrada. Si levantás un faro y querés que aparezca acá, mandá un PR con
los datos. Si no, no importa — tu faro funciona igual.

| Nombre | Dirección | UDP | WSS | Operador | Región |
|--------|-----------|-----|-----|----------|--------|
| **Faro Principal** | `190.220.45.26:54321` | ✅ | ✅ (443) | mamanga1 | NEA, Argentina |
| **Faro Oracle** | `150.136.55.87:54321` | ✅ | ✅ (443) | mamanga1 | Oracle Cloud |

---

## Protocolo del Faro

| Comando | Descripción |
|---------|-------------|
| `ANNOUNCE <DID> <ts> <sig>` | Registra DID → dirección en RAM *(con verificación de firma Ed25519)* |
| `RELAY <target> <sender> <payload>` | Reenvía payload al target *(busca en registry UDP y WSS — cross-relay)* |
| `RESPONSE <target> <payload>` | Respuesta directa al último cliente |
| `WHERE_IS <DID>` | Consulta presencia *(busca en registry UDP y WSS)* |
| `VERIFY_HASH` | Devuelve hash SHA-256 del binario del faro |

### ACK_IP (Roaming)

Cuando un nodo envía un mensaje, el faro responde con `ACK_IP <ip_pública>`.
Si la IP pública cambió (roaming, cambio de red), el nodo detecta el cambio
y re-ANNOUNCE automáticamente.

---

## Correr tu propio Faro (detalles)

### Requisitos

- Go 1.21+
- Puerto 54321/udp abierto (UDP)
- Puerto 443/tcp abierto (WSS, opcional)
- Certificados TLS para WSS (opcional)

### Build con metadata

```bash
go build -ldflags "-X main.buildCommit=$(git rev-parse --short HEAD) \
                    -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
                    -X main.buildVersion=v1.0.0" \
         -o faro ./cmd/faro/
```

### Certificados WSS (opcional)

```bash
openssl req -x509 -newkey rsa:2048 \
  -keyout key.pem -out cert.pem \
  -days 365 -nodes \
  -subj "/CN=tu-faro.dominio.com"
```

> Si no hay certificados, el faro corre solo UDP. WSS se desactiva
> sin matar el proceso (fix de auditoría v1.0).

### Firewall

```bash
sudo ufw allow 54321/udp  # Faro UDP
sudo ufw allow 443/tcp    # Faro WSS
```

### Despliegue con systemd

```ini
# /etc/systemd/system/xionia-faro.service
[Unit]
Description=XionIA Faro (relay ciego)
After=network.target

[Service]
Type=simple
User=xionia
WorkingDirectory=/opt/xionia/faro
ExecStart=/opt/xionia/faro/faro
Restart=always
RestartSec=5

# Hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/xionia/faro

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable xionia-faro
sudo systemctl start xionia-faro
```

---

## CGNAT y U2P

> ⚠️ **Estado: Fase 2 — no implementado.** U2P/XTP no existe en el
> código actual. El faro de Fase 1 es un relay ciego (UDP + WSS).
> U2P se implementará en Fase 2 con el TransportManager y el
> DirectTransport (Noise IK + hole punching).

### El problema del CGNAT

La mayoría de conexiones residenciales y móviles están detrás de CGNAT
(Carrier-Grade NAT):

```
Celular → CGNAT (comparte IP con miles) → Internet → Faro
```

El CGNAT asigna puertos dinámicos que cambian, y no permite conexiones
entrantes no solicitadas. Esto hace imposible el P2P directo.

### Solución: U2P (Fase 2)

U2P (User-to-Peer) es el protocolo de túneles de XionIA:

```
Nodo A (CGNAT) ←── U2P Tunnel ──→ Nodo B (CGNAT)
                    │
              Hole punching UDP
              asistido por Faro
              + Noise IK handshake
```

Protocolo propio (Noise IK + Curve25519 + UDP hole punching) que perfora
CGNAT. El faro actúa como rendezvous: intercambia endpoints entre los
dos nodos, y ellos establecen el túnel directo.

*(Fase 2 — no implementado)*

---

## Modos de Faro

| Modo | Descripción | Uso |
|------|-------------|-----|
| **Público** | Abierto a cualquier nodo con DID válido | Infraestructura comunitaria |
| **Privado** | Solo nodos autorizados (ACL del faro) | Redes corporativas |
| **Efímero** | Se apaga después de N horas | Redes temporales, eventos |
| **Server** | Corre como servicio (systemd) | Infraestructura permanente |
| **Volátil** | Corre en RAM (tmpfs), sin disco | Máxima privacidad |

> Los modos Server/Volátil con U2P en puerto 51820 son **Fase 2
> (no implementado)**. El faro de Fase 1 opera en modo relay ciego
> (UDP 54321 + WSS 443).

---

## Limitaciones actuales (Fase 1)

| Limitación | Descripción | Solución |
|------------|-------------|----------|
| **Un solo faro por nodo** | No hay multi-faro failover. Si tu faro se cae, te quedás sin conectividad hasta cambiar manualmente con `faro set`. | Fase 2: multi-faro failover |
| **Sin descubrimiento automático** | El nodo no busca faros disponibles; se conecta al que configuraste. | Fase 2: descubrimiento |
| **Sin federación** | Los faros no se hablan entre sí. Cada faro es independiente. | Fase 2: federación |
| **El faro relayea TODOS los datos** | Cuello de botella: todo el tráfico pasa por el faro. | Fase 2: faro → signaling puro + conexiones directas P2P |

---

## Métricas del Faro

El faro expone métricas básicas por UDP:

```bash
echo "STATS" | nc -u -w1 190.220.45.26 54321
```

```json
{
  "nodes": 42,
  "ws_nodes": 7,
  "gate_entries": 49,
  "uptime_seconds": 86400,
  "relay_count": 15234,
  "version": "v1.0.0",
  "commit": "7a2f3b1"
}
```

---

## Preguntas Frecuentes

**¿El faro puede ver mis mensajes?**
No. El faro solo ve paquetes cifrados (ChaCha20-Poly1305). No tiene las
claves para descifrar. Es un relay zero-knowledge.

**¿Qué pasa si el faro se cae?**
Los nodos pierden conectividad temporalmente. Al volver el faro (o al
conectarse a otro con `faro set`), los nodos re-ANNOUNCE y la red se
recupera. No hay pérdida de datos (los mensajes no se almacenan en el faro).

**¿Puedo correr un faro en mi casa?**
Sí. Solo necesitás un puerto UDP abierto (54321) y una IP pública (o
DDNS). Un Raspberry Pi 4 es suficiente. 3 comandos y está andando.

**¿Necesito permiso para levantar un faro?**
No. Cualquiera puede levantar un faro. El código es público, no hay
registro central, no hay autoridad que apruebe. La soberanía es eso.

**¿El faro escala?**
El faro de Fase 1 es un relay ciego con registry en RAM. Escala a miles
de nodos sin problema (cada entrada son ~100 bytes). Para decenas de
miles, se puede sharding por DID. La escalabilidad real viene en Fase 2
con la eliminación del relay (conexiones directas P2P).

**¿Por qué UDP y no TCP?**
UDP es más eficiente para relay (sin handshake, sin control de flujo).
WSS (TCP) es el fallback para redes que bloquean UDP (firewalls
corporativos, VPNs como Cloudflare WARP).

---

_Última actualización: 30 de Julio de 2026_
