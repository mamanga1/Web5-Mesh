package crypto

import (
"crypto/ed25519"
"crypto/rand"
"encoding/base64"
"encoding/json"
"fmt"
"testing"
"time"

"github.com/mr-tron/base58"
)

func helperBuildHandshake(t *testing.T, pub ed25519.PublicKey, priv ed25519.PrivateKey) ([]byte, Handshake) {
t.Helper()
nonce := make([]byte, 32)
if _, err := rand.Read(nonce); err != nil {
t.Fatalf("error generando nonce: %v", err)
}

ts := time.Now().Unix()
nonceB64 := base64.StdEncoding.EncodeToString(nonce)
did := fmt.Sprintf("did:maia:%s", base58.Encode(pub))
msg := fmt.Sprintf("%s|%d|%s", did, ts, nonceB64)

sig := ed25519.Sign(priv, []byte(msg))

hs := Handshake{
DID:   did,
Pub:   base58.Encode(pub),
TS:    ts,
Nonce: nonceB64,
Sig:   base64.StdEncoding.EncodeToString(sig),
}

data, err := json.Marshal(hs)
if err != nil {
t.Fatalf("error marchaleando handshake: %v", err)
}
return data, hs
}

func TestVerifyHandshakeSuccess(t *testing.T) {
pub, priv, err := ed25519.GenerateKey(rand.Reader)
if err != nil {
t.Fatalf("error generando clave ed25519: %v", err)
}

data, hs := helperBuildHandshake(t, pub, priv)
gate := NewGate(100, time.Hour)
defer gate.Close()

addr := "192.168.1.50:4000"
gotDID, err := gate.VerifyHandshake(data, addr)
if err != nil {
t.Fatalf("VerifyHandshake fallo inesperadamente: %v", err)
}

if gotDID != hs.DID {
t.Errorf("DID no coincide. Esperado %s, obtenido %s", hs.DID, gotDID)
}

if !gate.IsAllowed(addr) {
t.Errorf("La IP %s deberia estar autorizada", addr)
}
}

func TestHandshakeSizeLimit(t *testing.T) {
gate := NewGate(100, time.Hour)
defer gate.Close()

// Payload masivo > 1024 bytes para intentar DoS de deserialización
hugeData := make([]byte, MaxHandshakeSizeBytes+100)

_, err := gate.VerifyHandshake(hugeData, "10.0.0.1:1234")
if err == nil {
t.Errorf("Esperaba fallo por limite de tamano, pero no hubo error")
}
}

func TestHandshakeRateLimit(t *testing.T) {
gate := NewGate(100, time.Hour)
defer gate.Close()

pub, priv, _ := ed25519.GenerateKey(rand.Reader)
addr := "192.168.1.100:5000"

// Agotamos las 10 peticiones permitidas por minuto
for i := 0; i < MaxHandshakesPerMinute; i++ {
data, _ := helperBuildHandshake(t, pub, priv)
_, err := gate.VerifyHandshake(data, addr)
if err != nil {
t.Fatalf("Peticion %d dentro del limite fallo: %v", i+1, err)
}
}

// La peticion 11 debe rebotar por Rate Limit ANTES de cualquier analisis crypto
data, _ := helperBuildHandshake(t, pub, priv)
_, err := gate.VerifyHandshake(data, addr)
if err == nil {
t.Errorf("Esperaba rebote por Rate Limit en la peticion 11, pero se permitio")
}
}

func TestHandshakeAntiReplay(t *testing.T) {
pub, priv, _ := ed25519.GenerateKey(rand.Reader)
data, _ := helperBuildHandshake(t, pub, priv)

gate := NewGate(100, time.Hour)
defer gate.Close()

// Primera vez pasa
_, err1 := gate.VerifyHandshake(data, "192.168.1.10:1111")
if err1 != nil {
t.Fatalf("Primer intento fallo: %v", err1)
}

// Segundo intento con la misma data (mismo nonce) debe fallar
_, err2 := gate.VerifyHandshake(data, "192.168.1.11:2222")
if err2 == nil {
t.Errorf("Replay attack permitido: el mismo nonce se proceso dos veces")
}
}
