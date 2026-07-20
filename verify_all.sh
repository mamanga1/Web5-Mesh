#!/bin/bash

ARCH=$(uname -m)
if [ "$ARCH" == "aarch64" ] || [ "$ARCH" == "arm64" ]; then
    MESH_BIN="mesh-linux-arm64"
    FARO_BIN="faro-linux-arm64"
    FARO_HASH_TAG="faro-linux-arm64"
else
    MESH_BIN="mesh"
    FARO_BIN="faro"
    FARO_HASH_TAG="faro-linux-amd64"
fi

echo "🔍 Verificando mesh ($ARCH)..."
./$MESH_BIN verify

echo ""
echo "🔍 Verificando faro ($ARCH)..."
FARO_HASH=$(sha256sum $FARO_BIN | awk '{print $1}')
OFFICIAL=$(grep $FARO_HASH_TAG dist/hashes.txt | awk '{print $2}')
if [ "$FARO_HASH" == "$OFFICIAL" ]; then
    echo "✅ $FARO_HASH_TAG verificado"
    echo "  hash: ${FARO_HASH:0:16}..."
else
    echo "❌ hash no coincide"
    echo "  local: $FARO_HASH"
    echo "  oficial: $OFFICIAL"
fi
