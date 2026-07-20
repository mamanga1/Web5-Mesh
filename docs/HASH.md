# 🔐 Verificación de Integridad de Binarios XionIA

XionIA permite verificar que los binarios (`mesh` y `faro`) son **auténticos** y **no han sido modificados**.

Esto es clave para:
- **Auditoría:** Saber que el Faro que estás usando es el oficial.
- **Seguridad:** Detectar si un Faro fue comprometido (reemplazado por una versión maliciosa).
- **Transparencia:** Cualquier usuario puede verificar sin confiar en nadie.

---

## 📋 Archivos de Hashes

Los hashes oficiales se publican en el repositorio:

```
https://github.com/mamanga1/Web5-Mesh/blob/main/dist/hashes.txt
```

**Formato:**
```
# XionIA Kernel
# Generado: 2026-07-19T22:07:04Z
# Host: debian

mesh-linux-amd64  1f6c834e26d172e58c7dfdbe6baa58188fd0577641e97223549a09b55f9a6f21  9393212
faro-linux-amd64  60703683cf9e3beaafbf4821168384cf5f61260d803cb9ce46497c880b5bc86a  9112747
```

---

## 🧪 Verificación Local (con el repo clonado)

### 1. Compilar y generar hashes

```bash
cd ~/Web5-Mesh
./build.sh
```

### 2. Verificar el binario `mesh`

```bash
./mesh verify
```

**Salida esperada:**
```
🔍 Verificación de binario
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ mesh-linux-amd64 verificado
  hash: 1f6c834e...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### 3. Verificar el binario `faro` localmente

```bash
./verify_all.sh
```

**Salida esperada:**
```
🔍 Verificando mesh...
✅ mesh-linux-amd64 verificado
  hash: 1f6c834e...

🔍 Verificando faro...
✅ faro-linux-amd64 verificado
  hash: 60703683...
```

---

## 📡 Verificación Remota (sin clonar el repo)

### Requisitos
- **Cualquier consola** (Linux, macOS, Windows con WSL, o con Python instalado)
- **Cliente UDP** (`nc` en Linux/macOS, o un script simple en Python)
- **El Faro debe estar corriendo** y accesible en la red

---

### Opción 1: Puerto UDP (54321) — con `nc`

```bash
echo '{"cmd":"VERIFY_HASH"}' | nc -u -w2 190.220.45.26 54321
```

**Respuesta esperada:**
```json
{"hash":"60703683cf9e3beaafbf4821168384cf5f61260d803cb9ce46497c880b5bc86a","size":9112747,"commit":"","built":"","version":""}
```

---

### Opción 2: Puerto WebSocket (443) — con `wscat` o Python

#### Con `wscat` (instalar con npm)

```bash
wscat -c wss://190.220.45.26:443/ws
> {"cmd":"VERIFY_HASH"}
```

#### Con Python (sin dependencias)

```bash
python3 -c "
import socket, json, ssl, base64
# WebSocket handshake manual
import websocket  # si está instalado
# O usar una herramienta como wscat
"
```

**Nota:** El puerto 443 requiere una conexión WebSocket. Para verificaciones simples desde consola, recomendamos usar el puerto UDP 54321.

---

### Opción 3: Script `verify_faro.py` (universal)

```bash
curl -s https://raw.githubusercontent.com/mamanga1/Web5-Mesh/main/verify_faro.py -o verify_faro.py
python3 verify_faro.py 190.220.45.26:54321
```

**Salida esperada:**
```
🔍 Verificando faro 190.220.45.26:54321
   Release: v1.0.1

📡 Faro responde:
   Hash:   60703683cf9e3bea...
   Size:   9,112,747 bytes
   Commit:
   Versión:

📋 GitHub oficial (v1.0.1):
   Hash:   60703683cf9e3bea...
   Size:   9,112,747 bytes

✅ FARO VERIFICADO
   El binario corresponde exactamente al release oficial.
```

---

## 🔍 Comparación Manual

Si querés verificar manualmente:

```bash
# 1. Preguntar al Faro (UDP 54321)
FARO_HASH=$(echo '{"cmd":"VERIFY_HASH"}' | nc -u -w2 190.220.45.26 54321 | grep -o '"hash":"[^"]*"' | cut -d'"' -f4)

# 2. Descargar hash oficial
OFFICIAL_HASH=$(curl -s https://raw.githubusercontent.com/mamanga1/Web5-Mesh/main/dist/hashes.txt | grep faro-linux-amd64 | awk '{print $2}')

# 3. Comparar
if [ "$FARO_HASH" == "$OFFICIAL_HASH" ]; then
    echo "✅ FARO VERIFICADO"
else
    echo "🚨 FARO NO VERIFICADO"
fi
```

---

### Para WebSocket (443) con `wscat`

```bash
# Instalar wscat si no lo tenés
npm install -g wscat

# Conectar al Faro
wscat -c wss://190.220.45.26:443/ws

# Enviar el comando
{"cmd":"VERIFY_HASH"}

# Respuesta:
{"hash":"60703683cf9e3bea...","size":9112747,"commit":"","built":"","version":""}
```

---

## 🛡️ Seguridad

| Qué protege | Cómo |
|-------------|------|
| **Faro modificado** | El hash no coincide con el oficial |
| **Ataque MITM** | El hash se compara con GitHub (HTTPS) |
| **Binario viejo** | El tamaño y hash no coinciden |
| **Faro clonado** | La firma (cuando se agregue) lo detectará |

---

## 📦 Publicación de Hashes

Los hashes se actualizan automáticamente con cada release:

1. Se compila con `./build.sh`
2. Se genera `dist/hashes.txt`
3. Se sube al repo con `git add dist/hashes.txt`
4. Se publica en GitHub Releases

---

## 🔄 Cómo actualizar los hashes

```bash
cd ~/Web5-Mesh
./build.sh
git add dist/hashes.txt
git commit -m "release: actualizar hashes SHA256"
git push origin main
```

---

*XionIA - La Xión Digital 🦾*
