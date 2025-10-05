package Disk

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type RmDisk struct {
	ruta string // Ubicacion del disco a eliminar
}

func ParserRmdisk(tokens []string) (string, error) {
	var bufferSalida bytes.Buffer // Contenedor para capturar la salida

	cmd := &RmDisk{} // Crear instancia RmDisk

	// Unificar tokens en una cadena y procesar respetando comillas
	argumentos := strings.Join(tokens, " ")

	// Patron para localizar el parametro del comando rmdisk
	patron := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+`)

	// Buscar todas las coincidencias del patron en los argumentos
	coincidencias := patron.FindAllString(argumentos, -1)

	for _, coincidencia := range coincidencias {
		// Separar cada elemento en clave y valor usando "=" como separador
		claveValor := strings.SplitN(coincidencia, "=", 2)
		if len(claveValor) != 2 {
			return "", fmt.Errorf("formato de parametro invalido: %s", coincidencia)
		}
		clave, valor := strings.ToLower(claveValor[0]), claveValor[1]

		// Eliminar comillas del valor si estan presentes
		if strings.HasPrefix(valor, "\"") && strings.HasSuffix(valor, "\"") {
			valor = strings.Trim(valor, "\"")
		}

		// Evaluar el parametro -path
		switch clave {
		case "-path":
			// Validar que la ruta no este vacia
			if valor == "" {
				return "", errors.New("la ruta no puede estar vacia")
			}
			cmd.ruta = valor
		default:
			// Parametro no reconocido
			return "", fmt.Errorf("parametro desconocido: %s", clave)
		}
	}

	// Validar que el parametro -path haya sido especificado
	if cmd.ruta == "" {
		return "", errors.New("faltan parametros obligatorios: -path")
	}

	// Ejecutar la eliminacion del disco y capturar salida en el buffer
	err := ejecutarEliminacionDisco(cmd, &bufferSalida)
	if err != nil {
		return "", fmt.Errorf("fallo al eliminar el disco: %v", err)
	}

	// Devolver el contenido del buffer como texto
	return bufferSalida.String(), nil
}

func ejecutarEliminacionDisco(rmdisk *RmDisk, bufferSalida *bytes.Buffer) error {
	fmt.Fprintln(bufferSalida, "---------------------------- RMDISK ----------------------------")
	fmt.Fprintf(bufferSalida, "Procesando eliminacion del disco en: %s\n", rmdisk.ruta)

	// Comprobar archivo exista en el sistema
	if _, err := os.Stat(rmdisk.ruta); os.IsNotExist(err) {
		return fmt.Errorf("el archivo %s no se encuentra en el sistema", rmdisk.ruta)
	}

	// Eliminacion directa del archivo
	err := os.Remove(rmdisk.ruta)
	if err != nil {
		return fmt.Errorf("fallo durante la eliminacion del archivo: %v", err)
	}

	fmt.Fprintf(bufferSalida, "Disco ubicado en %s eliminado correctamente.\n", rmdisk.ruta)
	fmt.Fprintln(bufferSalida, "--------------------------------------------")
	return nil
}
