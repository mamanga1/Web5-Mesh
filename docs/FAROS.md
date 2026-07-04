# 📡 FAROS.md - Guía de Faros XionIA

## ¿Qué es un Faro?

Un Faro es un nodo relay ciego que permite la comunicación entre nodos XionIA cuando no pueden conectarse directamente (NAT, CGNAT, firewalls). El Faro:

- **No lee contenido**: Solo retransmite blobs cifrados
- **No guarda logs persistentes**: Opera en RAM volátil
- **No sabe quién sos**: Solo ve DIDs y blobs cifrados
- **Trunca IPs en logs**: Por privacidad, no muestra IPs completas
- **Es esencial**: Sin Faro, los nodos detrás de NAT no pueden comunicarse

---

## 🚀 Ejecución del Faro

### Compilar

```bash
go build -o faro cmd/faro/main.go
```

### Ejecutar

```bash
./faro
```

**Output:**
```
🛡️ [FARO] Relay Ciego en 0.0.0.0:54321 (Logs con IPs truncadas)
```

El Faro arranca en el puerto **54321 UDP** (hardcodeado). No hay flags de configuración.

### Ejecutar en background

```bash
./faro > faro.log 2>&1 &
tail -f faro.log
```

---

## 🔧 Configuración del Router (OBLIGATORIO)

### ¿Por qué necesito abrir el puerto?

El Faro escucha en el puerto UDP 54321. Para que nodos externos (desde internet) puedan conectarse, tu router debe redirigir el tráfico entrante en ese puerto hacia tu máquina.

**Sin port forwarding:**
- ❌ Nodos en tu LAN pueden conectarse (IPs privadas)
- ❌ Nodos externos NO pueden conectarse (router bloquea)

**Con port forwarding:**
- ✅ Nodos en tu LAN pueden conectarse
- ✅ Nodos externos pueden conectarse

### Paso 1: Identificar tu IP Interna

```bash
# Linux/macOS
ip addr show | grep "inet " | grep -v 127.0.0.1

# Windows
ipconfig | findstr /i "IPv4"
```

**Ejemplo:** `192.168.1.238`

### Paso 2: Acceder al Router

1. Abrí un navegador
2. Entrá a la IP del router (comúnmente `192.168.1.1` o `192.168.0.1`)
3. Logueate con usuario/admin (viene en la etiqueta del router)

### Paso 3: Configurar Port Forwarding

Buscá la sección **"Port Forwarding"**, **"Virtual Server"**, o **"NAT"** en tu router.

**Configuración:**

| Campo | Valor |
|-------|-------|
| **Nombre** | XionIA-Faro |
| **Protocolo** | UDP |
| **Puerto Externo** | 54321 |
| **IP Interna** | 192.168.1.238 (tu IP) |
| **Puerto Interno** | 54321 |

### Paso 4: Verificar que Funciona

**Desde otra red (usá tu celular con datos móviles):**

```bash
# Compilar un nodo
go build -o mesh ./cmd/mesh

# Conectar al Faro (etapa 1 faro de mamanga podes conectarte con vpn previo)
./mesh shell
```

**En el Faro, deberías ver:**
```
[FARO] 📥 ANNOUNCE: did:maia:xxxxx desde 190.220.*.*
```

Si no ves el ANNOUNCE, el puerto no está abierto correctamente.

---

## 🌐 Obtener tu IP Pública

```bash
curl ifconfig.me
```

**Ejemplo:** `190.220.45.26`

**Nota:** Si tu ISP te da IP dinámica (cambia cada tanto), vas a tener que actualizar el port forwarding si cambia. Solución: usar un servicio de DNS dinámico (DuckDNS, No-IP).

---

## 🛡️ Firewall Local

### Linux (ufw)

```bash
sudo ufw allow 54321/udp
```

### Linux (iptables)

```bash
sudo iptables -A INPUT -p udp --dport 54321 -j ACCEPT
```

### Windows (PowerShell como administrador)

```powershell
New-NetFirewallRule -DisplayName "XionIA Faro" -Direction Inbound -Protocol UDP -LocalPort 54321 -Action Allow
```

---

## 📋 Comandos del Protocolo

El Faro entiende 4 comandos:

### ANNOUNCE
Un nodo se registra en el Faro.

```
ANNOUNCE did:maia:xxxxx
```

El Faro guarda `DID → dirección UDP` en su registro en RAM.

### RELAY
Un nodo pide al Faro que retransmita un mensaje a otro nodo.

```
RELAY did:destino did:emisor payload_cifrado
```

El Faro busca el destino en su registro y reenvía. Si no lo encuentra, falla.

### RESPONSE
Un nodo responde a un mensaje recibido vía RELAY.

```
RESPONSE did:destino payload_cifrado
```

El Faro usa `lastClient` (registro temporal del último emisor) para reenviar la respuesta.

### WHERE_IS
Un nodo pregunta si otro DID está registrado en el Faro.

```
WHERE_IS did:xxxxx
```

El Faro responde `READY` si existe, `NOT_FOUND` si no.

---

## 🔍 Verificación

### Ver que el Faro está escuchando

```bash
sudo ss -ulnp | grep 54321
```

**Output esperado:**
```
UNCONN 0 0 *:54321 *:* users:(("faro",pid=650,fd=3))
```

### Ver tráfico en tiempo real

```bash
sudo tcpdump -i any udp port 54321
```

### Ver logs

```bash
# Si ejecutaste con background
tail -f faro.log
```

---

## 🔄 Problemas Comunes

### "No puedo conectarme desde afuera"

**Causas posibles:**
1. ❌ Port forwarding no configurado en router
2. ❌ Firewall local bloqueando
3. ❌ IP pública cambió (IP dinámica)
4. ❌ ISP usa CGNAT (no tenés IP pública real)

**Solución:**
```bash
# Verificar que el Faro está escuchando
sudo ss -ulnp | grep 54321
```

### "Mi ISP no me da IP pública (CGNAT)"

**Problema:** Algunos ISPs usan CGNAT, donde múltiples clientes comparten la misma IP pública. En este caso, el port forwarding no funciona.

**Solución:**
1. Llamar al ISP y pedir IP pública
2. Usar un VPS como Faro
3. Conectarse a otro Faro que sí tenga IP pública

---

## 🧭 Checklist para Levantar un Faro

- [ ] Compilar el Faro: `go build -o faro cmd/faro/main.go`
- [ ] Identificar IP interna: `ip addr show`
- [ ] Acceder al router: `192.168.1.1`
- [ ] Configurar port forwarding: UDP 54321 → IP interna
- [ ] Configurar firewall local: permitir UDP 54321
- [ ] Ejecutar el Faro: `./faro`
- [ ] Verificar IP pública: `curl ifconfig.me`
- [ ] Probar desde otra red: conectar nodo desde celular
- [ ] Confirmar ANNOUNCE en logs del Faro

---

## 🔮 Limitaciones Actuales (Fase 1)

- ❌ Puerto hardcodeado (54321), no se puede cambiar
- ❌ Sin título ni banner descriptivo
- ❌ Sin detección automática de IP pública
- ❌ Sin búsqueda de puerto libre
- ❌ Sin flags de configuración

**Estas limitaciones se resuelven en Fase 2** con Faros Inteligentes.

---

*Última actualización: Junio 2026*
```

