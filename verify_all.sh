#!/bin/bash
echo "🔍 Verificando mesh..."
./mesh verify

echo ""
echo "🔍 Verificando faro..."
FARO_HASH=$(sha256sum faro | awk '{print $1}')
OFFICIAL=$(grep faro-linux-amd64 dist/hashes.txt | awk '{print $2}')
if [ "$FARO_HASH" == "$OFFICIAL" ]; then
    echo "✅ faro-linux-amd64 verificado"
    echo "  hash: ${FARO_HASH:0:16}..."
else
    echo "❌ hash no coincide"
    echo "  local: $FARO_HASH"
    echo "  oficial: $OFFICIAL"
fi
