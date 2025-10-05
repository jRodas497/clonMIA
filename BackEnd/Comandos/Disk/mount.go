package Disk

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	Estructuras "backend/Estructuras"
	Global "backend/Global"
	Utils "backend/Utils"
)

type Mount struct {
	ruta   string
	nombre string
}

func ParserMount(tokens []string) (string, error) {
	var bufferSalida bytes.Buffer
	cmd := &Mount{}

	argumentos := strings.Join(tokens, " ")
	patron := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-name="[^"]+"|-name=[^\s]+`)
	coincidencias := patron.FindAllString(argumentos, -1)

	for _, coincidencia := range coincidencias {
		claveValor := strings.SplitN(coincidencia, "=", 2)
		if len(claveValor) != 2 {
			return "", fmt.Errorf("formato de parametro invalido: %s", coincidencia)
		}
		clave, valor := strings.ToLower(claveValor[0]), claveValor[1]
		if strings.HasPrefix(valor, "\"") && strings.HasSuffix(valor, "\"") {
			valor = strings.Trim(valor, "\"")
		}

		switch clave {
		case "-path":
			if valor == "" {
				return "", errors.New("la ruta no puede estar vacia")
			}
			cmd.ruta = valor
		case "-name":
			if valor == "" {
				return "", errors.New("el nombre no puede estar vacio")
			}
			cmd.nombre = valor
		default:
			return "", fmt.Errorf("parametro desconocido: %s", clave)
		}
	}

	if cmd.ruta == "" {
		return "", errors.New("faltan parametros requeridos: -path")
	}
	if cmd.nombre == "" {
		return "", errors.New("faltan parametros requeridos: -name")
	}

	err := ejecutarComandoMount(cmd, &bufferSalida)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	return bufferSalida.String(), nil
}

func ejecutarComandoMount(mount *Mount, bufferSalida *bytes.Buffer) error {
	fmt.Fprintln(bufferSalida, "========================== MOUNT ==========================")

	archivo, err := os.OpenFile(mount.ruta, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo el archivo del disco en la ruta: %s: %v", mount.ruta, err)
	}
	defer archivo.Close()

	var mbr Estructuras.MBR
	if err := mbr.Decodificar(archivo); err != nil {
		return fmt.Errorf("error deserializando el MBR: %v", err)
	}

	particion, indiceParticion := mbr.ObtenerParticionPorNombre(mount.nombre)
	if particion == nil {
		return fmt.Errorf("error: la partición '%s' no existe en el disco", mount.nombre)
	}

	if err := verificarParticionYaMontada(mount); err != nil {
		return err
	}

	idParticion, err := GenerarIdParticion(mount, indiceParticion)
	if err != nil {
		return fmt.Errorf("error generando el ID de la partición: %v", err)
	}

	Global.ParticionesMontadas[idParticion] = mount.ruta
	particion.MontarParticion(indiceParticion, idParticion)
	mbr.MbrPartitions[indiceParticion] = *particion

	if err := mbr.Codificar(archivo); err != nil {
		return fmt.Errorf("error serializando el MBR de vuelta al disco: %v", err)
	}

	imprimirParticionesMontadas(bufferSalida, mount.nombre, idParticion)
	return nil
}

func imprimirParticionesMontadas(bufferSalida *bytes.Buffer, nombreParticion string, idParticion string) {
	fmt.Fprintf(bufferSalida, "Partición '%s' montada correctamente con ID: %s\n", nombreParticion, idParticion)
	fmt.Fprintln(bufferSalida, "\n=== Particiones Montadas ===")
	for id, ruta := range Global.ParticionesMontadas {
		fmt.Fprintf(bufferSalida, "ID: %s | Ruta: %s\n", id, ruta)
	}
	fmt.Fprintln(bufferSalida, "===========================================================")
}

func verificarParticionYaMontada(mount *Mount) error {
	for id, rutaMontada := range Global.ParticionesMontadas {
		if rutaMontada == mount.ruta && strings.Contains(id, mount.nombre) {
			return fmt.Errorf("error: la partición '%s' ya está montada con ID: %s", mount.nombre, id)
		}
	}
	return nil
}

// Genera un ID unico para la particion montada
func GenerarIdParticion(mount *Mount, indiceParticion int) (string, error) {
	ultimosDosDiigtos := Global.Carnet[len(Global.Carnet)-2:]
	letra, err := Utils.ObtenerLetra(mount.ruta)
	if err != nil {
		return "", err
	}

	idParticion := fmt.Sprintf("%s%d%s", ultimosDosDiigtos, indiceParticion+1, letra)
	return idParticion, nil
}
