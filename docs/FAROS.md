# 📡 FAROS.md - Guía de Faros XionIA

## ¿Qué es un Faro?

Un Faro es un relay ciego que permite la comunicación entre nodos cuando no pueden conectarse directamente (NAT, CGNAT, firewalls).

- **No lee contenido**: Solo retransmite blobs cifrados
- **No guarda logs persistentes**: Opera en RAM volátil
- **No sabe quién sos**: Solo ve DIDs y blobs cifrados
- **Trunca IPs en logs**: Privacidad por diseño
- **Soporte dual**: UDP (54321) + WebSocket (443)
- **ACK**: Confirma entrega de mensajes
- **Hash verificable**: El nodo verifica el binario del faro contra GitHub releases

---

## 🚀 Ejecución del Faro

### Compilar

```bash
go build -o faro ./cmd/faro
```

### Ejecutar (UDP + WebSocket simultáneo)

```bash
./faro
sudo -b ./faro > faro.log 2>&1 (modo root para puerto 443
```

**Output:**
```
🚀 Iniciando Faro Dual (UDP + WebSocket)
🛡️ [FARO-UDP] Relay Ciego en 0.0.0.0:54321
🛡️ [FARO-WS] WebSocket TLS en 0.0.0.0:443
```

### Certificados TLS (WebSocket)

```bash
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=faro.local"
```

### Background

```bash
nohup ./faro > faro.log 2>&1 &
tail -f ~/Web5-Mesh/faro.log
(sudo)pkill -f faro
```

---

## 🎮 Comandos de Faro desde Mesh Shell

Desde `./mesh shell`, el nodo gestiona su conexión a faros:

### `faro set <addr>`

Configura un faro diferente al default.

```bash
xion@nodo:~$ faro set 192.168.1.100:54321
✅ Faro configurado: 192.168.1.100:54321

xion@nodo:~$ faro set 150.136.55.87:443
✅ Faro configurado: 150.136.55.87:443 (WebSocket)
```

**El faro default es** `150.136.55.87:443` (WebSocket) o el definido en `config.json`.

### `faro reset`

Restaura al faro default.

```bash
xion@nodo:~$ faro reset
✅ Faro reseteado a default: 150.136.55.87:443
```

## 🔧 Configuración de Red

### Port Forwarding (UDP 54321)

| Campo | Valor |
|-------|-------|
| Nombre | XionIA-Faro-UDP |
| Protocolo | UDP |
| Puerto Externo | 54321 |
| IP Interna | 192.168.1.x |
| Puerto Interno | 54321 |

### Firewall

```bash
# Linux (ufw)
sudo ufw allow 54321/udp
sudo ufw allow 443/tcp
sudo ufw allow 51820/udp  # U2P

# Oracle Cloud
# Security List: UDP 54321, TCP 443, UDP 51820 desde 0.0.0.0/0
```

---

## 🧠 CGNAT y U2P

### El problema

CGNAT bloquea conexiones entrantes. Tu IP "pública" es compartida con miles de usuarios.

### La solución: U2P/XTP

Protocolo propio (Noise IK + Curve25519 + UDP hole punching) que perfora CGNAT:

```
Nodo A (CGNAT) ──(saliente U2P)──> Bootstrap (IP pública) <──(saliente U2P)── Nodo B (CGNAT)
```

El faro volátil **inicia** conexión saliente. El CGNAT abre un agujero. Persistent keepalive (25s) lo mantiene abierto.

### Modos de Faro

| Modo | IP Pública | Uso |
|------|------------|-----|
| **Server** | ✅ Sí | Escucha U2P entrante en 51820 |
| **Volátil** | ❌ No | Se conecta a bootstrap node |

---

## 📋 Protocolo del Faro

| Comando | Función |
|---------|---------|
| `ANNOUNCE` | Registra DID → dirección en RAM |
| `RELAY` | Reenvía payload cifrado, guarda lastClient |
| `RESPONSE` | Reenvía a lastClient |
| `ACK` | Confirma entrega al emisor |
| `WHERE_IS` | Responde READY / NOT_FOUND |
| `VERIFY_HASH` | Responde hash SHA256 del binario |

---

## 🔍 Verificación

```bash
# Ver que escucha
sudo ss -ulnp | grep 54321
sudo ss -tlnp | grep :443

# Ver logs
tail -f faro.log

# Ver tráfico
sudo tcpdump -i any udp port 54321
```

---

## ✅ Checklist para Levantar un Faro

- [ ] Compilar: `go build -trimpath -ldflags="-s -w" -o faro ./cmd/faro`
- [ ] Generar certificados TLS
- [ ] Configurar port forwarding UDP 54321
- [ ] Configurar firewall: 54321/udp, 443/tcp, 51820/udp
- [ ] Ejecutar: `nohup ./faro > faro.log 2>&1 &`
- [ ] Verificar logs
- [ ] Probar conexión desde otro nodo

---

*XionIA Faraday — Sovereign overlay network with blind relay, E2E encryption, Faraday Cage isolation, and NAT traversal.*
*Última actualización: 17 de Julio de 2026*
