package Utils

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Convierte a bytes
func ConvertirABytes(size int, unidad string) (int, error) {
	switch unidad {
	case "B":
		return size, nil // Devuelve en bytes
	case "K":
		return size * 1024, nil // Kilobytes a bytes
	case "M":
		return size * 1024 * 1024, nil // Megabytes a bytes
	default:
		return 0, errors.New("unidad inválida") // Devuelve un error si es inválida
	}
}

// Lista con el abecedario
var abecedario = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M",
	"N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z",
}

// Mapa para almacenar la asignación de letras a los diferentes paths
var rutaALetra = make(map[string]string)

// Índice para la siguiente letra en el abecedario
var siguienteIndiceLEtra = 0

// Obtiene la letra asignada a un path
func ObtenerLetra(ruta string) (string, error) {
	if _, existe := rutaALetra[ruta]; !existe {
		if siguienteIndiceLEtra < len(abecedario) {
			rutaALetra[ruta] = abecedario[siguienteIndiceLEtra]
			siguienteIndiceLEtra++
		} else {
			return "", errors.New("no más letras disponibles para asignación")
		}
	}

	return rutaALetra[ruta], nil
}

// Elimina la letra asignada a un path
func EliminarLetra(ruta string) {
	delete(rutaALetra, ruta)
}

// Lee datos desde un archivo binario en la posición especificada
func LeerDeArchivo(archivo *os.File, desplazamiento int64, datos interface{}) error {
	_, err := archivo.Seek(desplazamiento, 0)
	if err != nil {
		return fmt.Errorf("falló al buscar el desplazamiento %d: %w", desplazamiento, err)
	}

	err = binary.Read(archivo, binary.LittleEndian, datos)
	if err != nil {
		return fmt.Errorf("falló al leer datos del archivo: %w", err)
	}

	return nil
}

// Escribe datos a un archivo binario en la posición especificada
func EscribirAArchivo(archivo *os.File, desplazamiento int64, datos interface{}) error {
	_, err := archivo.Seek(desplazamiento, 0)
	if err != nil {
		return fmt.Errorf("falló al buscar el desplazamiento %d: %w", desplazamiento, err)
	}

	err = binary.Write(archivo, binary.LittleEndian, datos)
	if err != nil {
		return fmt.Errorf("falló al escribir datos en el archivo: %w", err)
	}

	return nil
}

// Crea las carpetas padre si no existen
func CrearDirectoriosPadre(ruta string) error {
	directorio := filepath.Dir(ruta)
	// os.MkdirAll solo crea las carpetas que no existen
	err := os.MkdirAll(directorio, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error al crear las carpetas padre: %v", err)
	}
	return nil
}

// Obtiene el nombre del archivo .dot y el nombre de la imagen de salida
func ObtenerNombresArchivos(ruta string) (string, string) {
	directorio := filepath.Dir(ruta)
	nombreBase := strings.TrimSuffix(filepath.Base(ruta), filepath.Ext(ruta))
	nombreArchivoDot := filepath.Join(directorio, nombreBase+".dot")
	// Devolver la ruta de salida como .png junto al .dot
	imagenSalida := filepath.Join(directorio, nombreBase+".png")
	return nombreArchivoDot, imagenSalida
}

// Primero devuelve el primer elemento de un slice
func Primero[T any](slice []T) (T, error) {
	if len(slice) == 0 {
		var cero T
		return cero, errors.New("el slice está vacío")
	}
	return slice[0], nil
}

// Elimina un elemento de un slice en el índice dado
func EliminarElemento[T any](slice []T, indice int) []T {
	if indice < 0 || indice >= len(slice) {
		return slice // Índice fuera de rango, devolver el slice original
	}
	return append(slice[:indice], slice[indice+1:]...)
}

// Divide una cadena en pedazos y las almacena en una lista
func DividirCadenaEnChunks(s string) []string {
	var chunks []string
	for i := 0; i < len(s); i += 64 {
		fin := i + 64
		if fin > len(s) {
			fin = len(s)
		}
		chunks = append(chunks, s[i:fin])
	}
	return chunks
}

// Obtiene las carpetas padres y el directorio de destino
func ObtenerDirectoriosPadre(ruta string) ([]string, string) {
	/*	Normalizar el path */
	ruta = filepath.Clean(ruta)

	/*  Dividir el path en sus componentes */
	componentes := strings.Split(ruta, string(filepath.Separator))

	/*  Lista para almacenar las rutas de las carpetas padres */
	var directoriosPadre []string

	/*  Construir las rutas de las carpetas padres, excluyendo la última carpeta */
	for i := 1; i < len(componentes)-1; i++ {
		directoriosPadre = append(directoriosPadre, componentes[i])
	}

	/*  La última carpeta es la carpeta de destino */
	directorioDestino := componentes[len(componentes)-1]

	return directoriosPadre, directorioDestino
}

// Valida que el path tenga la extensión correcta
func ValidarExtensionDisco(ruta string) bool {
	return strings.HasSuffix(strings.ToLower(ruta), ".mia")
}

// Convierte bytes de unidad
func FormatearSize(bytes int) string {
	if bytes >= 1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	} else if bytes >= 1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%d bytes", bytes)
}

//	--------------------------------------------
//  ---------------- UTILIDADES ----------------
//	- ConvertirABytes: Convierte tamaños a bytes
//	- ObtenerLetra: Asigna letras a rutas de discos
//	- LeerDeArchivo/EscribirAArchivo: I/O binario
//	- CrearDirectoriosPadre: Crea directorios
//	- ValidarExtensionDisco: Valida extensiones .mia
//	- FormatearSize: Formatea tamaños legibles
//	--------------------------------------------
