# 🔍 Auditoría y Transparencia de Binarios

## Principio Fundamental

**Confianza cero, verificabilidad total.**

XionIA no pide que confíes ciegamente en el código. Te da las herramientas para verificar que lo que ejecutás es exactamente lo que está en el repositorio.

## ¿Qué es la Auditoría por Hash?

Cada binario oficial (faro y cliente) tiene un hash SHA-256 único que lo identifica. Si alguien modifica una sola línea de código, el hash cambia completamente.

**Esto garantiza:**
- ✅ El binario no fue modificado después de compilar
- ✅ El código fuente coincide con el binario ejecutado
- ✅ No hay backdoors ocultos
- ✅ Transparencia total

## Proceso de Build Determinista

Para que cualquier persona pueda compilar el código y obtener el mismo hash, usamos builds deterministas:

```bash
# Compilar el faro
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o faro ./cmd/faro

# Compilar el cliente
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o mesh ./cmd/mesh

# Generar hash
sha256sum faro > FARO_HASH.txt
sha256sum mesh > MESH_HASH.txt
```

**Parámetros críticos:**
- `CGO_ENABLED=0` → Build estático, sin dependencias del sistema
- `-trimpath` → Elimina rutas absolutas del binario (reproducible)
- `-ldflags="-s -w"` → Elimina símbolos y debug info (binario idéntico)

## Verificación Manual

Cualquier persona puede verificar que el binario coincide con el código:

### 1. Descargar el código
```bash
git clone https://github.com/mamanga1/Web5-Mesh
cd Web5-Mesh
```

### 2. Compilar el faro
```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o faro ./cmd/faro
```

### 3. Verificar el hash
```bash
sha256sum faro
```

### 4. Comparar con el hash publicado
```bash
cat FARO_HASH.txt
```

Si los hashes coinciden → **el binario es legítimo**.

## Auto-Verificación del Faro

El faro se auto-verifica al arrancar:

1. Calcula el hash de su propio binario
2. Lee el hash esperado desde `FARO_HASH.txt`
3. Si no coinciden → **el faro se niega a arrancar**

```bash
$ ./faro
🔍 Verificando integridad del binario...
✅ Hash verificado: abc123...
🛡️ [FARO] Relay Ciego en 0.0.0.0:54321
```

Si alguien compromete el servidor y reemplaza el binario:

```bash
$ ./faro
🔍 Verificando integridad del binario...
❌ CRITICAL: Hash mismatch!
   Esperado: abc123...
   Obtenido: xyz789...
   El binario fue modificado. Abortando.
```

## Comando /verify en el Cliente

Desde la shell del cliente, podés verificar en cualquier momento:

```bash
xion@nodo:~$ /verify
```

**Output:**
```
🔍 Verificando integridad...

📦 Cliente (mesh):
   Hash local:   abc123...
   Hash GitHub:  abc123...
   Estado:       ✅ MATCH

📡 Faro (190.220.45.26:54321):
   Hash remoto:  def456...
   Hash GitHub:  def456...
   Estado:       ✅ MATCH

✅ Todos los componentes son legítimos.
```

Si hay un mismatch:
```
❌ CRITICAL_MISMATCH en el faro!
   El binario del faro no coincide con el publicado en GitHub.
   Posible compromiso del servidor.
```

## ¿Qué Protege Esto?

### Escenario 1: Compromiso del Servidor
Si alguien entra al servidor del faro y reemplaza el binario con uno con backdoor:
- ❌ El faro no arranca (auto-verificación falla)
- ❌ Los clientes detectan el mismatch con `/verify`
- ✅ La red se alerta inmediatamente

### Escenario 2: Duda del Usuario
Si un tercero quiere usar la red pero no confía:
- ✅ Puede auditar el código (todo es open source)
- ✅ Puede compilar su propio binario
- ✅ Puede verificar que el hash coincide
- ✅ Puede levantar su propio faro

### Escenario 3: Supply Chain Attack
Si alguien intenta inyectar código malicioso en el proceso de build:
- ✅ El hash cambia
- ✅ La auto-verificación lo detecta
- ✅ Los usuarios pueden comparar hashes

##_hashes Publicados

Los hashes de cada release oficial se publican en:
- `FARO_HASH.txt` → Hash del faro
- `MESH_HASH.txt` → Hash del cliente

Estos archivos están en el repositorio y son inmutables (firmados con GPG en releases tagged).

## Levantar Tu Propio Faro

Si querés máxima soberanía, podés levantar tu propio faro:

```bash
# 1. Clonar el repositorio
git clone https://github.com/mamanga1/Web5-Mesh
cd Web5-Mesh

# 2. Compilar el faro
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o faro ./cmd/faro

# 3. Verificar el hash
sha256sum faro
cat FARO_HASH.txt
# Deben coincidir

# 4. Ejecutar el faro
./faro
```

El faro es:
- ✅ **Ciego**: No puede descifrar tráfico (E2E)
- ✅ **Sin logs de IPs completas**: Solo guarda los primeros octetos
- ✅ **Auditable**: Código 100% abierto
- ✅ **Verificable**: Hash público y auto-verificación

## Flujo de Auditoría Completo

```
1. Usuario descarga código de GitHub
   ↓
2. Usuario compila con build determinista
   ↓
3. Usuario verifica hash contra FARO_HASH.txt
   ↓
4. Si coincide → binario legítimo
   ↓
5. Usuario levanta su propio faro (opcional)
   ↓
6. Usuario configura su cliente para usar su faro
   ↓
7. Usuario ejecuta /verify periódicamente
   ↓
8. Si hay mismatch → alerta de compromiso
```

## Conclusión

XionIA no pide confianza. Ofrece verificabilidad.

Cada línea de código es auditable. Cada binario es verificable. Cada faro es transparente.

**Porque la privacidad no se basa en promesas, se basa en matemáticas.**
