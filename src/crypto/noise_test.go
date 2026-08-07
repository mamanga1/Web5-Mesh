package crypto

import (
"crypto/rand"
"testing"

"golang.org/x/crypto/curve25519"
)

func TestNoiseHandshakeIKWithPrologueSuccess(t *testing.T) {
var initPriv, initPub, respPriv, respPub [32]byte
rand.Read(initPriv[:])
curve25519.ScalarBaseMult(&initPub, &initPriv)
rand.Read(respPriv[:])
curve25519.ScalarBaseMult(&respPub, &respPriv)

prologue := BuildNoisePrologue("did:maia:initiator", "did:maia:responder")

initHS, err := NewHandshakeIK(true, &initPriv, &initPub, &respPub, prologue)
if err != nil {
t.Fatalf("error creando hs iniciador: %v", err)
}

respHS, err := NewHandshakeIK(false, &respPriv, &respPub, nil, prologue)
if err != nil {
t.Fatalf("error creando hs respondedor: %v", err)
}

// 1. Mensaje 1: Iniciador -> Respondedor (Aún no completa IK, requiere mensaje 2)
msg1, comp1, err := initHS.WriteHandshake([]byte("hello responder"))
if err != nil {
t.Fatalf("write msg1: %v", err)
}
if comp1 {
t.Errorf("comp1 deberia ser false en el mensaje 1 de Noise IK")
}

payload1, comp2, err := respHS.ReadHandshake(msg1)
if err != nil {
t.Fatalf("read msg1: %v", err)
}
if string(payload1) != "hello responder" {
t.Errorf("payload1 no coincide")
}
if comp2 {
t.Errorf("comp2 deberia ser false en el mensaje 1 de Noise IK")
}

// 2. Mensaje 2: Respondedor -> Iniciador (Acá se completa la sesión en ambos extremos)
msg2, comp3, err := respHS.WriteHandshake([]byte("hello initiator"))
if err != nil {
t.Fatalf("write msg2: %v", err)
}
if !comp3 {
t.Errorf("comp3 (respondedor) debio marcar completed=true tras enviar mensaje 2")
}

payload2, comp4, err := initHS.ReadHandshake(msg2)
if err != nil {
t.Fatalf("read msg2: %v", err)
}
if string(payload2) != "hello initiator" {
t.Errorf("payload2 no coincide")
}
if !comp4 {
t.Errorf("comp4 (iniciador) debio marcar completed=true tras procesar mensaje 2")
}

// Probar que el canal cifrado post-handshake funciona correctamente
cipherText, err := initHS.Encrypt([]byte("mensaje secreto mesh"))
if err != nil {
t.Fatalf("error cifrando post-handshake: %v", err)
}

plainText, err := respHS.Decrypt(cipherText)
if err != nil {
t.Fatalf("error descifrando post-handshake: %v", err)
}

if string(plainText) != "mensaje secreto mesh" {
t.Errorf("el mensaje cifrado post-handshake no coincide")
}
}

func TestNoiseHandshakeIKPrologueMismatch(t *testing.T) {
var initPriv, initPub, respPriv, respPub [32]byte
rand.Read(initPriv[:])
curve25519.ScalarBaseMult(&initPub, &initPriv)
rand.Read(respPriv[:])
curve25519.ScalarBaseMult(&respPub, &respPriv)

prologue1 := BuildNoisePrologue("did:maia:alice", "did:maia:bob")
prologue2 := BuildNoisePrologue("did:maia:suplantador", "did:maia:bob")

initHS, _ := NewHandshakeIK(true, &initPriv, &initPub, &respPub, prologue1)
respHS, _ := NewHandshakeIK(false, &respPriv, &respPub, nil, prologue2)

msg1, _, err := initHS.WriteHandshake([]byte("hello"))
if err != nil {
t.Fatalf("write msg1: %v", err)
}

// Debe reventar por fallo de autenticación al diferir los prologues
_, _, err = respHS.ReadHandshake(msg1)
if err == nil {
t.Errorf("Esperaba fallo de autenticacion por mismatch de prologue, pero paso")
}
}
