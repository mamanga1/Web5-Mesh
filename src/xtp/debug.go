package xtp

import "fmt"

// XTPDebug controla si se imprimen los logs internos de XTP/FSM.
// Por defecto false (silencio). Se activa con "debug on" en la shell.
var XTPDebug bool

func xtpDebugf(format string, args ...interface{}) {
	if XTPDebug {
		fmt.Printf("\x1b[2m"+format+"\x1b[0m\n", args...)
	}
}
