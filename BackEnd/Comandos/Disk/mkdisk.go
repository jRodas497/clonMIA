package Disk

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	Estructuras "backend/Estructuras"
	Utils "backend/Utils"
)

const (
	UnitK = "K"
	UnitM = "M"
	FitBF = "BF"
	FitFF = "FF"
	FitWF = "WF"
)

type MkDisk struct {
	/* Tamaño */
	size int
	/* Dimensional (K o M) */
	unit string
	/* Ajuste (BF, FF, WF) */
	fit string
	/* Ruta del disco */
	path string
}

func ParserMkdisk(tokens []string) (string, error) {
	cmd := &MkDisk{}
	var outputBuffer bytes.Buffer // Capturar los prints

	args := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-size=\d+|-unit=[kKmM]|-fit=[bBfFwW]{2}|-path="[^"]+"|-path=[^\s]+`)
	matches := re.FindAllString(args, -1)

	for _, match := range matches {
		kv := strings.SplitN(match, "=", 2)
		if len(kv) != 2 {
			return "", fmt.Errorf("formato de parámetro inválido: %s", match)
		}
		key, value := strings.ToLower(kv[0]), kv[1]

		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		switch key {
		case "-size":
			size, err := strconv.Atoi(value)
			if err != nil || size <= 0 {
				return "", errors.New("el tamaño debe ser un número entero positivo")
			}
			cmd.size = size
		case "-unit":
			value = strings.ToUpper(value)
			if value != UnitK && value != UnitM {
				return "", errors.New("la unidad debe ser K o M")
			}
			cmd.unit = value
		case "-fit":
			value = strings.ToUpper(value)
			if value != FitBF && value != FitFF && value != FitWF {
				return "", errors.New("el ajuste debe ser BF, FF o WF")
			}
			cmd.fit = value
		case "-path":
			if value == "" {
				return "", errors.New("el path no puede estar vacío")
			}
			if !strings.HasSuffix(value, ".mia") {
				return "", errors.New("el archivo debe tener la extensión .mia")
			}
			cmd.path = value
		default:
			return "", fmt.Errorf("parámetro desconocido: %s", key)
		}
	}

	if cmd.size == 0 {
		return "", errors.New("faltan parámetros requeridos: -size")
	}
	if cmd.path == "" {
		return "", errors.New("faltan parámetros requeridos: -path")
	}
	if cmd.unit == "" {
		cmd.unit = UnitM
	}
	if cmd.fit == "" {
		cmd.fit = FitFF
	}

	// Crear el disco con los parámetros proporcionados y capturar la salida en el buffer
	err := commandMkdisk(cmd, &outputBuffer)
	if err != nil {
		return "", fmt.Errorf("error al crear el disco: %v", err)
	}

	// Retorna el contenido del buffer, no el objeto MkDisk
	return outputBuffer.String(), nil
}

func commandMkdisk(mkdisk *MkDisk, outputBuffer *bytes.Buffer) error {
	// Redirigir la salida a outputBuffer
	fmt.Fprintln(outputBuffer, "---------------------------- MKDISK ----------------------------")
	fmt.Fprintf(outputBuffer, "Creando disco con tamaño: %d %s\n", mkdisk.size, mkdisk.unit)

	// Convertir el tamaño a bytes
	sizeBytes, err := Utils.ConvertirABytes(mkdisk.size, mkdisk.unit)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error convirtiendo el tamaño:", err)
		return err
	}

	// Crear el disco con el tamaño proporcionado
	err = createDisk(mkdisk, sizeBytes, outputBuffer)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error creando el disco:", err)
		return err
	}

	// Crear el MBR con el tamaño proporcionado
	err = createMBR(mkdisk, sizeBytes, outputBuffer)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error creando el MBR:", err)
		return err
	}

	fmt.Fprintln(outputBuffer, "--------------------------------------------")
	return nil
}

func createDisk(mkdisk *MkDisk, sizeBytes int, outputBuffer *bytes.Buffer) error {
	// Crear las carpetas necesarias
	err := os.MkdirAll(filepath.Dir(mkdisk.path), os.ModePerm)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error creando directorios:", err)
		return err
	}

	// Crear el archivo binario
	file, err := os.Create(mkdisk.path)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error creando archivo:", err)
		return err
	}
	defer file.Close()

	// Escribir en el archivo usando un buffer de 1 MB
	buffer := make([]byte, 1024*1024) // Crea un buffer de 1 MB
	for sizeBytes > 0 {
		writeSize := len(buffer)
		if sizeBytes < writeSize {
			writeSize = sizeBytes // Ajusta el tamaño de escritura si es menor que el buffer
		}
		if _, err := file.Write(buffer[:writeSize]); err != nil {
			return err // Devuelve un error si la escritura falla
		}
		sizeBytes -= writeSize // Resta el tamaño escrito del tamaño total
	}
	fmt.Fprintln(outputBuffer, "Disco creado exitosamente:", mkdisk.path)
	return nil
}

func createMBR(mkdisk *MkDisk, sizeBytes int, outputBuffer *bytes.Buffer) error {
	// Abrir el archivo del disco para escritura
	file, err := os.OpenFile(mkdisk.path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error abriendo el archivo:", err)
		return err
	}
	defer file.Close()

	// Crear el MBR con los valores proporcionados
	mbr := &Estructuras.MBR{
		MbrSize:          int32(sizeBytes),
		MbrCreacionDate:  float32(time.Now().Unix()),
		MbrDiskSignature: rand.Int31(),
		MbrDiskFit:       [1]byte{mkdisk.fit[0]},
		MbrPartitions: [4]Estructuras.Particion{
			{Part_status: [1]byte{'0'}, Part_type: [1]byte{'0'}, Part_fit: [1]byte{'0'}, Part_start: -1, Part_size: -1, Part_name: [16]byte{}, Part_correlative: -1, Part_id: [4]byte{}},
			{Part_status: [1]byte{'0'}, Part_type: [1]byte{'0'}, Part_fit: [1]byte{'0'}, Part_start: -1, Part_size: -1, Part_name: [16]byte{}, Part_correlative: -1, Part_id: [4]byte{}},
			{Part_status: [1]byte{'0'}, Part_type: [1]byte{'0'}, Part_fit: [1]byte{'0'}, Part_start: -1, Part_size: -1, Part_name: [16]byte{}, Part_correlative: -1, Part_id: [4]byte{}},
			{Part_status: [1]byte{'0'}, Part_type: [1]byte{'0'}, Part_fit: [1]byte{'0'}, Part_start: -1, Part_size: -1, Part_name: [16]byte{}, Part_correlative: -1, Part_id: [4]byte{}},
		},
	}

	// Serializar el MBR en el archivo usando el puntero de archivo `file`
	err = mbr.Codificar(file)
	if err != nil {
		fmt.Fprintln(outputBuffer, "Error serializando el MBR en el archivo:", err)
		return err
	}

	// Agregar mensajes al buffer
	fmt.Fprintln(outputBuffer, "MBR creado exitosamente en el disco.")
	// Imprimir el MBR creado
	mbr.Imprimir()
	fmt.Println("--------------------------------------------")

	return nil
}
