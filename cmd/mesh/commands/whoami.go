package commands

import (
	"encoding/hex"
	"fmt"
	"web5-mesh/src/crypto"
)

func init() {
	Register("whoami", cmdWhoami)
}

func cmdWhoami(args []string, id *crypto.Identity) string {
	if id == nil {
		return "❌ No se pudo cargar la identidad"
	}

	pubEdHex := hex.EncodeToString(id.PubKeyEd)
	
	var pubXHex string
	if id.PubKeyX != nil {
		pubXHex = hex.EncodeToString(id.PubKeyX[:])
	} else {
		pubXHex = "(no disponible)"
	}

	return fmt.Sprintf(`🆔 TU IDENTIDAD XIONIA:

  DID:        %s
  PubKey Ed:  %s
  PubKey X:   %s

📋 Para agregar este nodo a otro peer, copiá esta línea:
  acl import %s %s %s
`, id.DID, pubEdHex, pubXHex, id.DID, pubEdHex, pubXHex)
}
