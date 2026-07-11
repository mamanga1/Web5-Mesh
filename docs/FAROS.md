# 📡 FAROS.md - Guía de Faros XionIA

## ¿Qué es un Faro?

Un Faro es un nodo relay ciego que permite la comunicación entre nodos XionIA cuando no pueden conectarse directamente (NAT, CGNAT, firewalls). El Faro:

- **No lee contenido**: Solo retransmite blobs cifrados
- **No guarda logs persistentes**: Opera en RAM volátil
- **No sabe quién sos**: Solo ve DIDs y blobs cifrados
- **Trunca IPs en logs**: Por privacidad, no muestra IPs completas
- **Soporte dual**: UDP (54321) + WebSocket (443)
- **ACK**: Confirma entrega de mensajes

---

## 🚀 Ejecución del Faro

### Compilar

```bash
go build -trimpath -ldflags="-s -w" -o faro ./cmd/faro
```

### Ejecutar (UDP + WebSocket simultáneo)

```bash
./faro
```

**Output:**
```
🚀 Iniciando Faro Dual (UDP + WebSocket)
🛡️ [FARO-UDP] Relay Ciego en 0.0.0.0:54321
🛡️ [FARO-WS] WebSocket TLS en 0.0.0.0:443
```

El Faro arranca en **dos puertos simultáneamente:**
- **54321 UDP** (rápido, para redes locales)
- **443 TCP** (WebSocket con TLS, para firewalls)

### Variables de entorno (para el cliente)

```bash
# El faro escucha en 0.0.0.0:54321 y 0.0.0.0:443
# Los clientes usan FARO_ADDR para conectarse
export FARO_ADDR="190.220.45.26:54321"  # UDP
# o
export FARO_ADDR="190.220.45.26:443"    # WebSocket
```

### Certificados TLS (para WebSocket)

El Faro espera `cert.pem` y `key.pem` en el directorio de ejecución.

**Generar certificados autofirmados:**

```bash
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=faro.local"
```

**Output esperado:**
```
🛡️ [FARO-WS] WebSocket TLS en 0.0.0.0:443
```

**Nota:** Si no hay certificados, el WebSocket no arrancará.

### Ejecutar en background

```bash
nohup ./faro > faro.log 2>&1 &
tail -f faro.log
```

---

## 🔧 Configuración del Router

### UDP (54321)

**Por qué necesitas abrir el puerto:**  
El Faro escucha en UDP 54321. Para que nodos externos puedan conectarse, tu router debe redirigir el tráfico entrante.

**Configuración de Port Forwarding:**

| Campo | Valor |
|-------|-------|
| **Nombre** | XionIA-Faro-UDP |
| **Protocolo** | UDP |
| **Puerto Externo** | 54321 |
| **IP Interna** | 192.168.1.x (tu IP) |
| **Puerto Interno** | 54321 |

### WebSocket (443)

**El puerto 443 usa TCP**, el mismo que HTTPS. Si ya tenés un servidor web, puede haber conflicto.

**Verificar disponibilidad de 443:**
```bash
sudo netstat -tulpn | grep :443
```

---

## 🛡️ Firewall Local

### Linux (ufw)

```bash
sudo ufw allow 54321/udp
sudo ufw allow 443/tcp   # para WebSocket
sudo ufw enable
```

### Oracle Cloud (Security List)

| Source Type | Source CIDR | Protocol | Port | Description |
|-------------|-------------|----------|------|-------------|
| CIDR | 0.0.0.0/0 | UDP | 54321 | XionIA UDP |
| CIDR | 0.0.0.0/0 | TCP | 443 | XionIA WebSocket |

---

## 🧠 CGNAT vs IP Pública

### El mito del puerto 443

**El puerto 443 NO salta CGNAT.** Si tu ISP te asigna una IP privada (CGNAT), ningún puerto va a recibir conexiones entrantes, ni 443 ni ningún otro.

**¿Para qué sirve el 443 entonces?**

- Los ISP **no bloquean** el tráfico HTTPS (puerto 443).
- En redes corporativas o públicas, el puerto 54321 (UDP) suele estar bloqueado.
- El 443 está abierto en casi todos los firewalls.
- Si tenés **IP pública**, podés usar 443 para que nodos detrás de firewalls estrictos puedan conectarse a tu Faro.

### ¿Cómo se salta CGNAT?

**La única forma real:**

1. **Levantar un Faro con IP pública** (ej: Oracle Free Tier, VPS).
2. **Todos los nodos (incluso detrás de CGNAT) se conectan SALIENTE a ese Faro**.
3. **El Faro reenvía los mensajes** entre los nodos.

**Flujo:**

```
Nodo A (CGNAT) ──(saliente)──> Faro (IP pública) ──(saliente)──> Nodo B (CGNAT)
```

**El Faro actúa como relay**, no como túnel CGNAT. Los nodos siempre inician la conexión hacia afuera.

### Puertos para conexiones salientes

- **UDP (54321)**: Más rápido, recomendado.
- **WebSocket (443)**: Más lento (TLS overhead), pero pasa firewalls más restrictivos.

**Ambos requieren IP pública en el Faro.** La diferencia es qué puerto puede atravesar el firewall del nodo cliente.

---

## 📋 Protocolo del Faro

El Faro entiende 4 comandos:

### ANNOUNCE
Un nodo se registra en el Faro.

```
ANNOUNCE did:maia:xxxxx
```

El Faro guarda `DID → dirección` en su registro en RAM.

### RELAY
Un nodo pide al Faro que retransmita un mensaje a otro nodo.

```
RELAY did:destino did:emisor payload_cifrado
```

El Faro busca el destino en su registro y reenvía. Si existe, envía ACK al emisor.

### RESPONSE
Un nodo responde a un mensaje recibido vía RELAY.

```
RESPONSE did:destino payload_cifrado
```

El Faro usa `lastClient` para reenviar la respuesta al nodo original.

### ACK
Confirmación de entrega (enviada automáticamente por el Faro).

```
ACK did:emisor did:destino
```

---

## 🔍 Verificación

### Ver que el Faro está escuchando

```bash
# UDP
sudo ss -ulnp | grep 54321

# WebSocket
sudo ss -tlnp | grep :443
```

**Output esperado:**
```
UNCONN 0 0 *:54321 *:* users:(("faro",pid=650,fd=3))
LISTEN 0 0 *:443 *:* users:(("faro",pid=650,fd=4))
```

### Ver logs en tiempo real

```bash
tail -f faro.log
```

**Logs esperados:**
```
🚀 Iniciando Faro Dual (UDP + WebSocket)
🛡️ [FARO-UDP] Relay Ciego en 0.0.0.0:54321
🛡️ [FARO-WS] WebSocket TLS en 0.0.0.0:443
[FARO-UDP] 📥 ANNOUNCE: did:maia:xxxxx desde 190.220.*.*
[FARO-UDP] 📤 RELAY: reenviando a did:maia:yyyyy (104.28.*.*)
[FARO-UDP] ✅ ACK enviado a did:maia:xxxxx
```

### Ver tráfico en tiempo real

```bash
sudo tcpdump -i any udp port 54321
sudo tcpdump -i any tcp port 443
```

---

## 🔄 Problemas Comunes

### "No puedo conectarme desde afuera"

| Problema | Solución |
|----------|----------|
| Port forwarding no configurado | Configurar en el router |
| Firewall local bloqueando | `sudo ufw allow 54321/udp` |
| IP pública cambió | Usar DNS dinámico |
| ISP usa CGNAT | Usar WebSocket (443) o VPS |

### "WebSocket no arranca"

**Causa:** No hay certificados TLS.

**Solución:**
```bash
openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=faro.local"
```

### "El puerto 443 está ocupado"

**Causa:** Otro servicio (nginx, apache) usa el puerto 443.

**Soluciones:**
1. Detener el otro servicio
2. Usar solo UDP (54321)
3. Modificar el puerto en el código (futuro)

---

## 🧭 Checklist para Levantar un Faro

- [ ] Compilar el Faro: `go build -trimpath -ldflags="-s -w" -o faro ./cmd/faro`
- [ ] Generar certificados TLS: `openssl req ...`
- [ ] Identificar IP interna: `ip addr show`
- [ ] Configurar port forwarding: UDP 54321 → IP interna
- [ ] Configurar firewall local: `sudo ufw allow 54321/udp`
- [ ] Ejecutar el Faro: `nohup ./faro > faro.log 2>&1 &`
- [ ] Verificar logs: `tail -f faro.log`
- [ ] Probar desde otra red: conectar nodo desde celular
- [ ] Confirmar ANNOUNCE y ACK en logs

---

## 🔮 Limitaciones Actuales

- ❌ Puerto UDP hardcodeado (54321)
- ❌ No se puede deshabilitar WebSocket (443)
- ❌ Certificados TLS fijos (`cert.pem`, `key.pem`)
- ❌ Sin detección automática de IP pública

**Estas limitaciones se resolverán en futuras versiones.**

---

*Última actualización: Julio 2026*
