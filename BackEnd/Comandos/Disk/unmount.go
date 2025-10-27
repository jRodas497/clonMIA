package Disk

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	Estructuras "backend/Estructuras"
	Global "backend/Global"
)

// Unmount estructura para representar el comando unmount
type Unmount struct {
	id string // ID de la partición a desmontar
}

// ParserUnmount parsea el comando unmount
func ParserUnmount(tokens []string) (string, error) {
	var outputBuffer bytes.Buffer
	cmd := &Unmount{}

	// Parsear argumento -id
	for _, token := range tokens {
		if strings.HasPrefix(token, "-id=") {
			cmd.id = strings.TrimPrefix(token, "-id=")
		}
	}

	// ID no esté vacío
	if cmd.id == "" {
		return "", errors.New("parámetros requeridos: -id")
	}

	// Ejecutar el comando unmount y capturar los mensajes importantes en el buffer
	err := comandoUnmount(cmd, &outputBuffer)
	if err != nil {
		return "", err
	}

	return outputBuffer.String(), nil
}

// comandoUnmount ejecuta el comando unmount para desmontar la partición con el ID especificado
func comandoUnmount(unmount *Unmount, outputBuffer *bytes.Buffer) error {
	fmt.Fprintln(outputBuffer, "========================== UNMOUNT ==========================")

	// Verificar si el ID de la partición existe en las particiones montadas globales
	mountedPath, exists := Global.ParticionesMontadas[unmount.id]
	if !exists {
		return fmt.Errorf("error: la partición con ID '%s' no está montada", unmount.id)
	}

	// Abrir el archivo del disco
	file, err := os.OpenFile(mountedPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo el archivo del disco: %v", err)
	}
	defer file.Close()

	// Leer el MBR del disco
	var mbr Estructuras.MBR
	err = mbr.Decodificar(file)
	if err != nil {
		return fmt.Errorf("error deserializando el MBR: %v", err)
	}

	// Buscar la partición en el MBR que tiene el ID especificado
	found := false
	for i := range mbr.MbrPartitions {
		partition := &mbr.MbrPartitions[i] // Obtener referencia a la partición
		partitionID := strings.TrimSpace(string(partition.Part_id[:]))
		if partitionID == unmount.id {
			// Desmontar la partición: Cambiar el valor del correlativo a 0
			err = partition.MontarParticion(0, "")
			if err != nil {
				return fmt.Errorf("error desmontando la partición: %v", err)
			}

			// Actualizar el MBR en el archivo después del desmontaje
			// _, err = file.Seek(0, 0)
   			// if err != nil {
       		// 	return fmt.Errorf("error al posicionarse en el archivo: %v", err)
   			// }

			// Acá
			err = mbr.Codificar(file)
			//

			if err != nil {
				return fmt.Errorf("error al actualizar el MBR en el disco: %v", err)
			}

			found = true
			break
		}
	}

	// Si no se encontró la partición con el ID, devolver error
	if !found {
		return fmt.Errorf("error: no se encontró la partición con ID '%s' en el disco", unmount.id)
	}

	// Remover el ID de la partición de la lista de particiones montadas
	delete(Global.ParticionesMontadas, unmount.id)

	// Imprimir el estado después del desmontaje
	fmt.Fprintf(outputBuffer, "Partición con ID '%s' desmontada exitosamente.\n", unmount.id)
	fmt.Fprintln(outputBuffer, "\n=== Particiones Montadas ===")
	for id, path := range Global.ParticionesMontadas {
		fmt.Fprintf(outputBuffer, "ID: %s | Path: %s\n", id, path)
	}
	fmt.Fprintln(outputBuffer, "===========================================================")

	return nil
}
