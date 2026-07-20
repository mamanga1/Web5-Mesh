#!/bin/bash
VERSION="v1.0.1"

echo "🔍 Verificando XionIA $VERSION"

# Clave pública hardcodeada (la que generó minisign)
PUBKEY="RWSotIaW2DtepDGiGUUI69JskbPKtslnvqON9bKkkVwrJnG0+GbmpBAw"

# Descargar hashes y firma
curl -sLO https://github.com/mamanga1/Web5-Mesh/releases/download/$VERSION/hashes-${VERSION}.txt
curl -sLO https://github.com/mamanga1/Web5-Mesh/releases/download/$VERSION/hashes-${VERSION}.txt.minisig

# Verificar firma
minisign -Vm hashes-${VERSION}.txt -P "$PUBKEY"

echo ""
echo "✅ Si ves 'Signature verified', los hashes son auténticos."
echo "📋 Podés comparar el hash de tu binario con el de hashes-${VERSION}.txt"
