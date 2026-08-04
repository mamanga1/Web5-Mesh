// Copyright (C) 2026 Fernando Martin Lopez. All Rights Reserved.
// SPDX-License-Identifier: AGPL-3.0-only WITH Commons-Clause-1.0
//
// This file is part of Web5-Mesh — sovereign network kernel prototype (Fase 1).
// Use of this source code is governed by the AGPLv3 + Commons Clause
// license that can be found in the LICENSE file at the root of this repo.
//
// Commercial use, SaaS deployment, or resale without a commercial license
// agreement is strictly prohibited. Contact the author for licensing.

package commands

import (
	"fmt"
	"strings"
	"web5-mesh/src/crypto"
)

type CommandFunc func(args []string, id *crypto.Identity) string

var registry = make(map[string]CommandFunc)
var NetworkExecutor func(myID *crypto.Identity, targetDID, cmd string) string

func SetNetworkExecutor(fn func(myID *crypto.Identity, targetDID, cmd string) string) {
	NetworkExecutor = fn
}

func Register(name string, fn CommandFunc) {
	// Limpiamos la barra al registrar para que todo sea uniforme internamente
	cleanName := strings.TrimPrefix(name, "/")
	registry[cleanName] = fn
}

func Execute(input string, id *crypto.Identity) string {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return ""
	}

	cmdName := parts[0]
	// Limpiamos la barra si el usuario la escribió, para que 'ping' y '/ping' funcionen igual
	cmdName = strings.TrimPrefix(cmdName, "/")
	
	args := parts[1:]

	fn, ok := registry[cmdName]
	if !ok {
		return fmt.Sprintf("Comando no reconocido: '%s'. Usá 'help' para ver la lista.", cmdName)
	}

	return fn(args, id)
}
