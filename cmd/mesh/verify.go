package main

import (
"crypto/sha256"
"encoding/hex"
"fmt"
"io"
"os"
"runtime"
"strings"
)

// Variables inyectadas en build (se llenan con -ldflags)
var (
buildCommit  string
buildTime    string
buildVersion string
)

// selfVerify verifica el binario actual contra hashes.txt local
func selfVerify() (bool, string) {
// 1. Obtener ruta del binario actual
exe, err := os.Executable()
if err != nil {
return false, fmt.Sprintf("no puedo leerme: %v", err)
}

// 2. Calcular hash
f, err := os.Open(exe)
if err != nil {
return false, fmt.Sprintf("error abriendo: %v", err)
}
defer f.Close()

h := sha256.New()
if _, err := io.Copy(h, f); err != nil {
return false, fmt.Sprintf("error calculando hash: %v", err)
}
localHash := hex.EncodeToString(h.Sum(nil))

// 3. Leer hashes.txt local
hashFile := "dist/hashes.txt"
data, err := os.ReadFile(hashFile)
if err != nil {
return false, fmt.Sprintf("no se encontró %s (ejecutá build.sh primero)", hashFile)
}

// 4. Buscar este binario
binaryName := fmt.Sprintf("mesh-%s-%s", runtime.GOOS, runtime.GOARCH)
lines := strings.Split(string(data), "\n")

var officialHash string
for _, line := range lines {
if strings.HasPrefix(line, binaryName) {
parts := strings.Fields(line)
if len(parts) >= 2 {
officialHash = parts[1]
break
}
}
}

if officialHash == "" {
return false, fmt.Sprintf("binario %s no encontrado en hashes.txt", binaryName)
}

if localHash != officialHash {
return false, fmt.Sprintf("hash no coincide\n  local: %s\n  oficial: %s", localHash, officialHash)
}

return true, fmt.Sprintf("✅ %s verificado\n  hash: %s", binaryName, localHash[:16]+"...")
}

// printVerify muestra el resultado de la verificación
func printVerify(ok bool, msg string) {
fmt.Println("🔍 Verificación de binario")
fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
if ok {
fmt.Printf("✅ %s\n", msg)
} else {
fmt.Printf("❌ %s\n", msg)
}
fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
