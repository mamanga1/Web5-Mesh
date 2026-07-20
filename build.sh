#!/bin/bash
# build.sh - Compilación de XionIA Kernel
# Funciona en Linux, macOS, Cloud Shell
# Uso: ./build.sh [arm64]

set -e  # Salir si hay error

echo "🔨 Compilando XionIA Kernel..."

# 1. Limpiar binarios anteriores
echo "🧹 Limpiando binarios anteriores..."
rm -f mesh faro mesh-linux-arm64 faro-linux-arm64

# 2. Asegurar que go.mod existe
if [ ! -f go.mod ]; then
    echo "📝 Creando go.mod..."
    cat << 'EOF' > go.mod
module web5-mesh

go 1.21
EOF
fi

# 3. Verificar que el module name sea correcto
FIRST_LINE=$(head -n 1 go.mod)
if [[ "$FIRST_LINE" != "module web5-mesh" ]]; then
    echo "⚠️  Corrigiendo go.mod..."
    # Cross-platform sed
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' '1s/.*/module web5-mesh/' go.mod
    else
        sed -i '1s/.*/module web5-mesh/' go.mod
    fi
fi

# 4. Descargar dependencias
echo "📦 Descargando dependencias..."
go mod tidy

# 5. Compilar el Faro (nativo)
echo "⚙️  Compilando Faro (nativo)..."
go build -o faro ./cmd/faro

# 6. Compilar el Mesh (nativo)
echo "⚙️  Compilando Mesh (nativo)..."
go build -o mesh ./cmd/mesh

# 7. Verificar que se crearon ambos
if [ -f mesh ] && [ -f faro ]; then
    echo ""
    echo "✅ Compilación nativa exitosa!"
    echo ""
    echo "📦 Binarios generados:"
    ls -lh mesh faro
    echo ""
    echo "🚀 Para ejecutar:"
    echo "   Terminal 1: ./faro"
    echo "   Terminal 2: ./mesh shell"
else
    echo "❌ Error: No se generaron los binarios"
    exit 1
fi

# 8. Compilación ARM64 (opcional)
if [[ "$1" == "arm64" ]]; then
    echo ""
    echo "🔧 Compilando para ARM64 (TV Boxes, Raspberry Pi)..."

    # Compilar Faro para ARM64
    echo "⚙️  Compilando Faro (ARM64)..."
    GOOS=linux GOARCH=arm64 go build -o faro-linux-arm64 ./cmd/faro

    # Compilar Mesh para ARM64
    echo "⚙️  Compilando Mesh (ARM64)..."
    GOOS=linux GOARCH=arm64 go build -o mesh-linux-arm64 ./cmd/mesh

    if [ -f mesh-linux-arm64 ] && [ -f faro-linux-arm64 ]; then
        echo ""
        echo "✅ Compilación ARM64 exitosa!"
        echo ""
        echo "📦 Binarios ARM64 generados:"
        ls -lh mesh-linux-arm64 faro-linux-arm64
        echo ""
        echo "📲 Para usar en TV Box / Raspberry Pi:"
        echo "   scp mesh-linux-arm64 faro-linux-arm64 user@tvbox:~/"
        echo "   ssh user@tvbox"
        echo "   chmod +x mesh-linux-arm64 faro-linux-arm64"
        echo "   ./faro-linux-arm64"
    else
        echo "❌ Error: No se generaron los binarios ARM64"
        exit 1
    fi
fi

# ============================================================
# 9. Generar hashes SHA256 de todos los binarios
# ============================================================
echo ""
echo "🔐 Generando hashes SHA256..."

# Crear directorio dist si no existe
mkdir -p dist

# Copiar binarios a dist (si existen)
[ -f mesh ] && cp mesh dist/mesh-linux-amd64
[ -f faro ] && cp faro dist/faro-linux-amd64
[ -f mesh-linux-arm64 ] && cp mesh-linux-arm64 dist/mesh-linux-arm64
[ -f faro-linux-arm64 ] && cp faro-linux-arm64 dist/faro-linux-arm64

# Generar hashes.txt
echo "# XionIA Kernel" > dist/hashes.txt
echo "# Generado: $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> dist/hashes.txt
echo "# Host: $(hostname)" >> dist/hashes.txt
echo "" >> dist/hashes.txt

for f in dist/mesh-* dist/faro-*; do
    if [ -f "$f" ]; then
        HASH=$(sha256sum "$f" | awk '{print $1}')
        SIZE=$(stat -c%s "$f" 2>/dev/null || stat -f%z "$f" 2>/dev/null)
        NAME=$(basename "$f")
        echo "$NAME  $HASH  $SIZE" >> dist/hashes.txt
    fi
done

echo ""
echo "✅ Hashes generados en dist/hashes.txt"
echo ""
cat dist/hashes.txt
echo ""
echo "📋 Para verificar un binario:"
echo "   ./mesh verify"
