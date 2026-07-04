package commands

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
	"web5-mesh/src/crypto"
)

func init() {
	Register("ssh", cmdSSH)
}

func cmdSSH(args []string, id *crypto.Identity) string {
	if len(args) < 2 {
		return "Uso: ssh <usuario@host> \"comando\"\nEj: ssh mamanga@192.168.1.238 \"uptime\""
	}

	target := args[0]
	command := strings.Join(args[1:], " ") // Permite comandos con espacios

	parts := strings.Split(target, "@")
	if len(parts) != 2 {
		return "⚠️ Formato inválido. Usá: usuario@host"
	}

	user := parts[0]
	host := parts[1]

	// NOTA DE SOBERANÍA: En una V2, podemos configurar esto para usar 
	// tu node.key como clave de autenticación SSH sin contraseña.
	// Por ahora, usamos autenticación por contraseña interactiva.
	fmt.Print("🔑 Contraseña para " + user + "@" + host + ": ")
	var password string
	fmt.Scanln(&password)

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		// En una red soberana de confianza, ignoramos la verificación estricta de host key 
		// para evitar bloqueos por cambios de IP, pero en producción se puede endurecer.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return fmt.Sprintf("❌ Fallo de conexión a %s: %v", host, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Sprintf("❌ Fallo al crear sesión en %s: %v", host, err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return fmt.Sprintf("⚠️ Comando ejecutado con errores en %s:\n%s", host, string(output))
	}

	return fmt.Sprintf("✅ Respuesta de %s:\n%s", host, strings.TrimSpace(string(output)))
}
