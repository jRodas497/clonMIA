package User

import (
	"bytes"
	"fmt"

	Global "backend/Global"
	Estructuras "backend/Estructuras"
)

type LOGOUT struct{}

// Comando LOGOUT y captura los mensajes importantes
func ParserLogout(tokens []string) (string, error) {
	var bufferSalida bytes.Buffer

	// El comando Logout sin parametros
	if len(tokens) > 1 {
		return "", fmt.Errorf("el comando Logout no acepta parametros")
	}

	err := comandoLogout(&bufferSalida)
	if err != nil {
		return "", err
	}

	return bufferSalida.String(), nil
}

// Comando LOGOUT, captura los mensajes importantes en buffer
func comandoLogout(bufferSalida *bytes.Buffer) error {
	// Verifica si hay una sesion activa
	if Global.UsuarioActual == nil || !Global.UsuarioActual.Estado {
		return fmt.Errorf("no hay ninguna sesion activa")
	}

	fmt.Fprintf(bufferSalida, "Cerrando sesion de usuario: %s\n", Global.UsuarioActual.Nombre)

	// Reiniciar la estructura del usuario actual
	Global.UsuarioActual = &Estructuras.Usuario{}

	fmt.Fprintln(bufferSalida, "Sesion cerrada correctamente.")

	return nil
}
