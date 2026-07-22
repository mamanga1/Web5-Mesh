#!/bin/bash
# build.sh - Compilación de XionIA Kernel (funciona en Xeon, TVBox, Celular)
# Uso: ./build.sh [arm64]

set -e

echo "🔨 Compilando XionIA Kernel..."

# Detectar arquitectura
ARCH=$(uname -m)
echo "📐 Arquitectura detectada: $ARCH"

# 1. Limpiar binarios anteriores
echo "🧹 Limpiando binarios anteriores..."
rm -f mesh faro mesh-* faro-* 2>/dev/null || true

# 2. Asegurar go.mod
if [ ! -f go.mod ]; then
    echo "📝 Creando go.mod..."
    cat << 'EOF' > go.mod
module web5-mesh

go 1.21
EOF
fi

# 3. Descargar dependencias
echo "📦 Descargando dependencias..."
go mod tidy

# 4. Compilar Faro (nativo)
echo "⚙️  Compilando Faro (nativo)..."
go build -o faro ./cmd/faro

# 5. Compilar Mesh (nativo)
echo "⚙️  Compilando Mesh (nativo)..."
go build -o mesh ./cmd/mesh

# 6. Verificar
if [ -f mesh ] && [ -f faro ]; then
    echo ""
    echo "✅ Compilación exitosa!"
    echo "📦 Binarios generados:"
    ls -lh mesh faro
else
    echo "❌ Error: No se generaron los binarios"
    exit 1
fi

# 7. Generar hashes
echo ""
echo "🔐 Generando hashes SHA256..."

mkdir -p dist

# Determinar nombre del archivo de hashes
HASH_FILE="dist/hashes-$(date +%Y%m%d).txt"
if [ -f dist/hashes-v1.0.1.txt ]; then
    HASH_FILE="dist/hashes-v1.0.1.txt"
fi

echo "# XionIA Kernel" > "$HASH_FILE"
echo "# Generado: $(date -u +'%Y-%m-%dT%H:%M:%SZ')" >> "$HASH_FILE"
echo "# Host: $(hostname)" >> "$HASH_FILE"
echo "# Arquitectura: $ARCH" >> "$HASH_FILE"
echo "" >> "$HASH_FILE"

# Hash del mesh
sha256sum mesh >> "$HASH_FILE"
# Hash del faro
sha256sum faro >> "$HASH_FILE"

echo ""
echo "✅ Hashes generados en $HASH_FILE"
echo ""
cat "$HASH_FILE"

# 8. Si se pide ARM64 y estamos en Xeon (x86_64), compilar cross
if [[ "$1" == "arm64" ]] && [[ "$ARCH" == "x86_64" ]]; then
    echo ""
    echo "🔧 Compilando para ARM64 (cross-compile)..."
    GOOS=linux GOARCH=arm64 go build -o faro-linux-arm64 ./cmd/faro
    GOOS=linux GOARCH=arm64 go build -o mesh-linux-arm64 ./cmd/mesh
    if [ -f mesh-linux-arm64 ] && [ -f faro-linux-arm64 ]; then
        echo "✅ ARM64 generados"
        ls -lh mesh-linux-arm64 faro-linux-arm64
        sha256sum mesh-linux-arm64 >> "$HASH_FILE"
        sha256sum faro-linux-arm64 >> "$HASH_FILE"
    fi
fi

echo ""
echo "🚀 Para ejecutar:"
echo "   Terminal 1: ./faro"
echo "   Terminal 2: ./mesh shell"
