package crypto

import (
"encoding/hex"
"testing"
)

func TestACLValidateBinding(t *testing.T) {
acl := &ACL{Peers: make(map[string]PeerInfo)}

pubX := []byte("12345678901234567890123456789012") // 32 bytes
pubXHex := hex.EncodeToString(pubX)

acl.Add(PeerInfo{
DID:      "did:maia:node1",
PubKeyEd: "ed25519pubkeyhex",
PubKeyX:  pubXHex,
})

// 1. Debe validar OK con el DID y PubKeyX correctos
if !acl.ValidateBinding("did:maia:node1", pubX) {
t.Errorf("ValidateBinding debio dar true para el par correcto")
}

// 2. Debe rebotar si la clave X25519 es distinta
wrongX := []byte("99999999999999999999999999999999")
if acl.ValidateBinding("did:maia:node1", wrongX) {
t.Errorf("ValidateBinding debio dar false para clave X25519 incorrecta")
}

// 3. Debe rebotar si el DID no existe
if acl.ValidateBinding("did:maia:inexistente", pubX) {
t.Errorf("ValidateBinding debio dar false para DID inexistente")
}
}
