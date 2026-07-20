# 🔐 Verificación de Integridad de Binarios XionIA

XionIA permite verificar que los binarios (`mesh` y `faro`) son **auténticos** y **no han sido modificados**.

Esto es clave para:
- **Auditoría:** Saber que el Faro que estás usando es el oficial.
- **Seguridad:** Detectar si un Faro fue comprometido (reemplazado por una versión maliciosa).
- **Transparencia:** Cualquier usuario puede verificar sin confiar en nadie.

---

## 📋 Sistema de Verificación

XionIA usa **minisign** para firmar los hashes de los binarios.

- **Clave pública:** `release.pub` (disponible en el repo)
- **Clave privada:** Solo en posesión del equipo de desarrollo (NUNCA subida)
- **Archivos de hashes:** Publicados en GitHub Releases con su firma `.minisig`

---

## 🧪 Verificación Local (con el repo clonado)

### 1. Compilar

```bash
cd ~/Web5-Mesh
./build.sh
```

### 2. Verificar con `verify.sh`

```bash
./verify.sh
```

**Salida esperada:**
```
🔍 Verificando XionIA v1.0.1
Signature and comment signature verified
Trusted comment: XionIA v1.0.1

✅ Si ves 'Signature verified', los hashes son auténticos.
📋 Podés comparar el hash de tu binario con el de hashes-v1.0.1.txt
```

---

## 📡 Verificación Remota (sin clonar el repo)

### Requisitos
- **minisign instalado** ([descargar](https://github.com/jedisct1/minisign/releases))
- **curl** o **wget**

### 1. Descargar los archivos necesarios

```bash
# Clave pública
curl -LO https://raw.githubusercontent.com/mamanga1/Web5-Mesh/main/release.pub

# Hashes y firma
curl -LO https://github.com/mamanga1/Web5-Mesh/releases/download/v1.0.1/hashes-v1.0.1.txt
curl -LO https://github.com/mamanga1/Web5-Mesh/releases/download/v1.0.1/hashes-v1.0.1.txt.minisig
```

### 2. Verificar la firma

```bash
PUBKEY=$(grep -v '^untrusted comment:' release.pub | tr -d ' ')
minisign -Vm hashes-v1.0.1.txt -P "$PUBKEY"
```

**Salida esperada:**
```
Signature and comment signature verified
Trusted comment: XionIA v1.0.1
```

---

## 🔍 Verificar un Binario Descargado

```bash
# 1. Descargar el binario (ej: faro)
curl -LO https://github.com/mamanga1/Web5-Mesh/releases/download/v1.0.1/faro

# 2. Calcular su hash
sha256sum faro

# 3. Buscar el hash en hashes-v1.0.1.txt
cat hashes-v1.0.1.txt | grep faro
```

**Si los hashes coinciden, el binario es el oficial.**

---

## 📡 Verificación del Faro Remoto (UDP)

### Con `nc` (Linux/macOS)

```bash
echo '{"cmd":"VERIFY_HASH"}' | nc -u -w2 190.220.45.26 54321
```

**Respuesta esperada:**
```json
{"hash":"60703683cf9e3bea...","size":9112747,"commit":"","built":"","version":""}
```

### Comparar con el hash oficial

```bash
FARO_HASH=$(echo '{"cmd":"VERIFY_HASH"}' | nc -u -w2 190.220.45.26 54321 | grep -o '"hash":"[^"]*"' | cut -d'"' -f4)
OFFICIAL_HASH=$(grep faro-linux-amd64 hashes-v1.0.1.txt | awk '{print $2}')

if [ "$FARO_HASH" == "$OFFICIAL_HASH" ]; then
    echo "✅ FARO VERIFICADO"
else
    echo "🚨 FARO NO VERIFICADO"
fi
```

---

## 🛡️ Seguridad

| Qué protege | Cómo |
|-------------|------|
| **Faro modificado** | El hash no coincide con el oficial |
| **Ataque MITM** | Los hashes están firmados con minisign |
| **Binario no oficial** | La firma no se verifica con `release.pub` |
| **Release falso** | Solo los releases oficiales tienen firma válida |

---

## 📦 Publicación de Hashes

1. Se compila con `./build.sh`
2. Se genera `hashes-vX.X.X.txt`
3. Se firma con minisign: `minisign -Sm hashes-vX.X.X.txt -s release.key`
4. Se suben ambos archivos a GitHub Releases

---

*XionIA - La Xión Digital 🦾*

---

**Comandante, ¿subimos?** 🧉🦾
