package commands

import (
	"fmt"
	"os"
	"web5-mesh/src/crypto"
)

func init() {
	Register("/exit", cmdExit)
}

func cmdExit(args []string, id *crypto.Identity) string {
	fmt.Println("👋 Saliendo...")
	os.Exit(0)
	return ""
}
