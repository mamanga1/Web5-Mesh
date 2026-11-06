# 🗼 Faros — El Casamentero Soberano

**XionIA Kernel** — Los faros presentan pares, prestan STUN y sostienen el blind relay de respaldo. Jamás ven contenido, jamás guardan nada, y cuando el túnel se establece, desaparecen del camino.

---

## ¿Qué es un Faro?

Un faro es un servidor que:

- **Recibe** ANNOUNCEs de nodos XionIA (con firma Ed25519 — Gate DID)
- **Registra** la asociación DID → dirección IP:puerto **en RAM** (amnésico)
- **Presenta** los pares: intercambia señalización ICE (base64 opaco para él)
- **Presta STUN** — RFC5389, en el mismo puerto UDP que la señalización
- **Re-transmite** texto cifrado (blind relay) cuando no hay túnel todavía
- **No descifra** nada (zero-knowledge: sin claves, sin contenido)
- **No almacena** nada persistente (registry en RAM, cero logs)
- **No inspecciona** contenido
- **No le interesa** si lo que presenta es un chat, un archivo o una llamada

### La doctrina del faro

> **El faro es escalera, no andamio.** Presenta a los pares, los ayuda a encontrarse — y cuando el túnel ICE se establece, sale del camino. Verificado en producción: 20 minutos de conversación directa entre dos nodos con el faro caído. El círculo ya constituido no lo necesita.

### Protocolo del faro

| Comando | Descripción |
|---------|-------------|
| `ANNOUNCE <DID> <ts> <sig>` | Registro DID → dirección (verificación de firma Ed25519, anti-spoofing) |
| `ACK_IP` | Confirmación de registro — y detección de roaming: si tu IP pública cambió, el kernel re-ANNOUNCE |
| `ICE_INFO <target> <sender> <b64>` | Forward de señalización ICE entre pares (opaco para el faro) |
| `RELAY <target> <sender> <payload>` | Blind relay: transporta ciphertext, nada más |
| STUN (RFC5389) | Binding requests/responses en el mismo puerto — STUN soberano |

Todo mensaje al faro lleva **padding aleatorio de 50–200 bytes** (anti-DPI): un ANNOUNCE y cualquier otro paquete tienen el mismo aspecto en el wire.

---

## Tu propio faro: la guía completa

**No requiere permiso, registro, ni aprobación.** El binario es tuyo, el puerto es tuyo, la máquina es tuya. Si tenés un Linux con un puerto UDP abierto, tenés un faro.

### Requisitos

- Cualquier Linux (x86 o ARM64: una Xeon, un Raspberry Pi, una TV Box)
- Go 1.22+ (o el binario precompilado)
- **Puerto 443/UDP / 55123/UDP** (señalización + STUN — el corazón)
- **Puerto 443/TCP / 8444/TCP** (WSS — fallback para redes restrictivas)
- ~10 MB de RAM. Consume lo que un enchufe de 10W

---

## Estrategia de Fachada: El Puerto 443 (Evasión DPI / CGNAT)

Para saltar las restricciones de operadoras móviles (Claro, Personal) y permitir conexión **incluso con saldo cero**, la fachada externa recomendada para el Faro Semilla es enmascarar la entrada en el puerto **443** (utilizado mundialmente por HTTPS/QUIC).

El servicio interno del faro sigue escuchando en sus puertos nativos (`55123` UDP y `8444` WSS), mientras que el sistema operativo o el router hacen la traducción transparente.

### Opción A: En tu casa con el router (ej. Cudy)

1. **Reserva de IP fija en LAN:** Asignar IP fija al faro (ej. `192.168.1.195`).
2. **Reenvío de puertos (Virtual Server / Port Forwarding):**

| Nombre | Protocolo | Puerto Externo | IP Interna | Puerto Interno |
|---|---|---|---|---|
| `Xionia-443-UDP` | UDP | **443** | 192.168.1.195 | **55123** |
| `Xionia-Kernel-WSS` | TCP | **443** | 192.168.1.195 | **8444** |

### Opción B: En una VPS en la nube (ej. Oracle Cloud con Linux)

Si tenés tu faro en un VPS, aplicás la regla de redirección directamente con `iptables`:

```bash
# Redireccionar el tráfico entrante de 443/UDP a 55123/UDP en el kernel
sudo iptables -t nat -A PREROUTING -p udp --dport 443 -j REDIRECT --to-ports 55123

# Permitir la entrada de los paquetes
sudo iptables -I INPUT -p udp --dport 443 -j ACCEPT
sudo iptables -I INPUT -p udp --dport 55123 -j ACCEPT
sudo iptables -I INPUT -p tcp --dport 8444 -j ACCEPT

# Guardar persistentemente las reglas
sudo apt install -y iptables-persistent
sudo netfilter-persistent save

```

---

### Levantando el servicio (systemd)

1. **Descargar / Compilar:**

```bash
git clone [https://github.com/mamanga1/xionia-kernel](https://github.com/mamanga1/xionia-kernel)
cd xionia-kernel && go build -o faro-server ./cmd/faro-server/

```

2. **Crear la unidad de servicio (`/etc/systemd/system/faro.service`):**

```ini
[Unit]
Description=XionIA Faro (Signaling + STUN + WSS, Amnésico)
After=network.target

[Service]
Type=simple
User=xionia
WorkingDirectory=/opt/xionia/faro
ExecStart=/opt/xionia/faro/faro-server -udp 55123 -wss 8444
Restart=always
RestartSec=5

# Hardening: aislamiento estricto del sistema
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target

```

3. **Activar y arrancar:**

```bash
sudo systemctl enable faro
sudo systemctl start faro

```

4. **Conexión desde los nodos:**

* **Terminal:** `xion@nodo:~$ faro <TU_IP_PUBLICA>:443`
* **App SanJorge:** Tocar el ícono de la nube → Cargar `<TU_IP_PUBLICA>:443`. Queda guardado en `faros.json` o se escanea vía código QR.

---

## ¿Cuánto banco?

**10.000 nodos ≈ 1.3 Mbps y 10 MB de RAM.** Un Raspberry Pi alimenta un pueblo entero. La entrada del registry son ~100 bytes en RAM: el faro es amnésico por diseño, no por escasez. Con una Netbook añeja de bajo consumo 24/7 (4 GB RAM), podés sostener sin despeinarte más de 100.000 nodos activos.

---

## Faros Semilla Oficiales

Los binarios de SanJorge y Derinkuyu traen configurados por defecto los siguientes faros semilla para garantizar arranque *plug-and-play*:

| Nombre | Dirección Pública | STUN | WSS (TCP) | Operador / Ubicación |
| --- | --- | --- | --- | --- |
| **Faro Semilla (Kati)** | `190.220.45.26:443` | ✅ (RFC5389 en 443/UDP) | `443` | mamanga1 · Corrientes, Argentina |
| **Faro Respaldo (VPS)** | `141.148.42.190:443` | ✅ (RFC5389 en 443/UDP) | `8444` | Oracle Cloud · San Pablo / USA |

> El faro semilla es **escalera de arranque, no dependencia**: cambiás al tuyo con un comando o desde la app y quedás completamente fuera de la infraestructura ajena. Ese es el diseño soberano.

---

## Preguntas Frecuentes

**¿El faro puede ver mis mensajes?**

No. Tres razones apiladas: lo que transporta es ciphertext (ChaCha20-Poly1305), no tiene las claves (nunca salen del dispositivo), y cuando hay túnel directo el tráfico no pasa por él. Verificable con `tcpdump`: el faro solo ve QUÉ IP:puerto habla, jamás QUÉ dicen.

**¿Por qué usamos el puerto 443 para el faro?**

El puerto `443` es la autopista principal de HTTPS/QUIC. Las operadoras no pueden bloquearlo sin romper la conectividad web global. Esto permite que la red funcione sin crédito o bajo cortafuegos restrictivos.

**¿Qué pasa si el faro se cae?**

El círculo ya constituido sigue hablando directo (verificado en pruebas de campo: +20 min de streaming/chat directo con faro apagado). Los mensajes a nodos detrás de relay esperarán la reconexión. No se pierden datos en el faro porque no existen datos en él: es 100% amnésico.

---

Última actualización: Octubre de 2026 — `xionia-kernel v9.1` en producción.
