#!/bin/bash
# build.sh - Compilación multiplataforma de XIONIA
# Uso: ./build.sh [--all|--clean]
#   --all   : Compilar para todas las plataformas
#   --clean : Limpiar binarios y directorio dist

set -e

VERSION="v1.0.0"
OUTPUT_DIR="./dist"
mkdir -p "$OUTPUT_DIR"

echo "🔨 XIONIA Builder - $VERSION"
echo "================================"

# ============================================
# FUNCIONES
# ============================================

clean() {
    echo "🧹 Limpiando binarios..."
    rm -f mesh faro mesh-* faro-* 2>/dev/null || true
    rm -rf "$OUTPUT_DIR" 2>/dev/null || true
    mkdir -p "$OUTPUT_DIR"
    echo "✅ Limpieza completada"
}

compile() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT=$3
    local EXT=$4

    echo "📦 Compilando $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" -o "$OUTPUT_DIR/$OUTPUT$EXT" ./cmd/mesh
    if [ $? -eq 0 ]; then
        echo "✅ $OUTPUT$EXT listo"
    else
        echo "❌ Error compilando $OUTPUT$EXT"
    fi
}

compile_faro() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT=$3
    local EXT=$4

    echo "📦 Compilando Faro $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" -o "$OUTPUT_DIR/$OUTPUT$EXT" ./cmd/faro
    if [ $? -eq 0 ]; then
        echo "✅ $OUTPUT$EXT listo"
    else
        echo "❌ Error compilando $OUTPUT$EXT"
    fi
}

generate_hashes() {
    echo ""
    echo "🔐 Generando hashes SHA256..."
    HASH_FILE="$OUTPUT_DIR/hashes-$(date +%Y%m%d).txt"
    echo "# XIONIA Hashes - $VERSION" > "$HASH_FILE"
    echo "# Generado: $(date -u +'%Y-%m-%dT%H:%M:%SZ')" >> "$HASH_FILE"
    echo "" >> "$HASH_FILE"
    
    for file in "$OUTPUT_DIR"/*; do
        if [ -f "$file" ] && [[ ! "$file" == *.txt ]]; then
            sha256sum "$file" >> "$HASH_FILE"
        fi
    done
    
    echo "✅ Hashes guardados en $HASH_FILE"
    cat "$HASH_FILE"
}

# ============================================
# COMPILACIÓN NATIVA
# ============================================

build_native() {
    echo ""
    echo "📀 Compilación nativa..."
    go build -ldflags="-s -w" -o mesh ./cmd/mesh
    go build -ldflags="-s -w" -o faro ./cmd/faro
    
    # Copiar a dist
    cp mesh "$OUTPUT_DIR/mesh-linux-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" 2>/dev/null || true
    cp faro "$OUTPUT_DIR/faro-linux-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" 2>/dev/null || true
    
    echo "✅ Nativo compilado"
}

# ============================================
# COMPILACIÓN MULTIPLATAFORMA
# ============================================

build_all() {
    echo ""
    echo "🌍 Compilación multiplataforma..."

    # Mesh
    compile "linux" "amd64" "mesh-linux-amd64" ""
    compile "linux" "arm64" "mesh-linux-arm64" ""
    compile "windows" "amd64" "mesh-windows-amd64" ".exe"
    compile "darwin" "amd64" "mesh-darwin-amd64" ""
    compile "darwin" "arm64" "mesh-darwin-arm64" ""
    compile "android" "arm64" "mesh-android-arm64" ""

    # Faro
    compile_faro "linux" "amd64" "faro-linux-amd64" ""
    compile_faro "linux" "arm64" "faro-linux-arm64" ""
    compile_faro "windows" "amd64" "faro-windows-amd64" ".exe"
    compile_faro "darwin" "amd64" "faro-darwin-amd64" ""
    compile_faro "darwin" "arm64" "faro-darwin-arm64" ""
    compile_faro "android" "arm64" "faro-android-arm64" ""

    generate_hashes
}

# ============================================
# MENSAJE FINAL
# ============================================

show_help() {
    echo ""
    echo "Uso: ./build.sh [--all|--clean|--help]"
    echo ""
    echo "  --all    : Compilar para todas las plataformas"
    echo "  --clean  : Limpiar binarios"
    echo "  --help   : Mostrar esta ayuda"
    echo ""
    echo "Plataformas soportadas:"
    echo "  - Linux (amd64, arm64)"
    echo "  - Windows (amd64)"
    echo "  - macOS (amd64, arm64)"
    echo "  - Android/Termux (arm64)"
    echo ""
    echo "Binarios generados en: $OUTPUT_DIR/"
}

# ============================================
# MAIN
# ============================================

case "$1" in
    --clean)
        clean
        ;;
    --all)
        clean
        build_all
        ;;
    --help|-h)
        show_help
        ;;
    *)
        build_native
        generate_hashes
        ;;
esac

echo ""
echo "🚀 Para ejecutar:"
echo "   ./mesh shell"
echo "   ./faro"
