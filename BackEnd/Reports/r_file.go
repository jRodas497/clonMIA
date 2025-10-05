package Reports

import (
	"fmt"
	"os"
	"path/filepath"

	Estructuras "backend/Estructuras"
	Utils "backend/Utils"
)

// Genera un reporte con el nombre y contenido de un archivo específico
func ReporteArchivo(sb *Estructuras.SuperBlock, rutaDisco string, ruta string, rutaArchivo string) error {
	err := Utils.CrearDirectoriosPadre(ruta)
	if err != nil {
		return fmt.Errorf("error al crear directorios: %v", err)
	}

	archivo, err := os.Open(rutaDisco)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}
	defer archivo.Close()

	indiceInodo, err := buscarInodoArchivo(sb, archivo, rutaArchivo)
	if err != nil {
		return fmt.Errorf("error al buscar el inodo del archivo: %v", err)
	}

	contenido, err := leerContenidoArchivo(sb, archivo, indiceInodo)
	if err != nil {
		return fmt.Errorf("error al leer el contenido del archivo: %v", err)
	}

	reporte, err := os.Create(ruta)
	if err != nil {
		return fmt.Errorf("error al crear el archivo de reporte: %v", err)
	}
	defer reporte.Close()

	_, nombreArchivo := filepath.Split(rutaArchivo)
	texto := fmt.Sprintf("Nombre del archivo: %s\n\nContenido del archivo:\n%s", nombreArchivo, contenido)

	_, err = reporte.WriteString(texto)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo de reporte: %v", err)
	}

	fmt.Println("Reporte del archivo generado:", ruta)
	return nil
}

// Busca el inodo del archivo especificado por su ruta
func buscarInodoArchivo(sb *Estructuras.SuperBlock, archivo *os.File, rutaArchivo string) (int32, error) {
	indiceActual := int32(0)
	directorios, nombreArchivo := Utils.ObtenerDirectoriosPadre(rutaArchivo)
	for _, dir := range directorios {
		inodo, err := leerInodo(sb, archivo, indiceActual)
		if err != nil {
			return -1, fmt.Errorf("error al leer inodo: %v", err)
		}
		encontrado, siguiente := buscarInodoEnDirectorio(inodo, archivo, dir, sb)
		if !encontrado {
			return -1, fmt.Errorf("directorio '%s' no encontrado", dir)
		}
		indiceActual = siguiente
	}
	inodo, err := leerInodo(sb, archivo, indiceActual)
	if err != nil {
		return -1, fmt.Errorf("error al leer inodo final: %v", err)
	}
	encontrado, inodoArchivo := buscarInodoEnDirectorio(inodo, archivo, nombreArchivo, sb)
	if !encontrado {
		return -1, fmt.Errorf("archivo '%s' no encontrado", nombreArchivo)
	}
	return inodoArchivo, nil
}

// Lee el contenido de un archivo dado su inodo
func leerContenidoArchivo(sb *Estructuras.SuperBlock, archivo *os.File, indiceInodo int32) (string, error) {
	inodo, err := leerInodo(sb, archivo, indiceInodo)
	if err != nil {
		return "", fmt.Errorf("error al leer inodo del archivo: %v", err)
	}
	var contenido string
	for _, idxBloque := range inodo.I_block {
		if idxBloque == -1 {
			continue
		}
		bloque, err := leerBloqueArchivo(sb, archivo, idxBloque)
		if err != nil {
			return "", fmt.Errorf("error al leer bloque de archivo: %v", err)
		}
		contenido += string(bloque.B_cont[:])
	}
	return contenido, nil
}

// Lee un inodo en la posición dada
func leerInodo(sb *Estructuras.SuperBlock, archivo *os.File, indiceInodo int32) (*Estructuras.INodo, error) {
	inodo := &Estructuras.INodo{}
	offset := int64(sb.S_inode_start + indiceInodo*sb.S_inode_size)
	err := inodo.Decodificar(archivo, offset)
	if err != nil {
		return nil, fmt.Errorf("error al decodificar inodo: %v", err)
	}
	return inodo, nil
}

// Lee un bloque de archivo en la posición dada
func leerBloqueArchivo(sb *Estructuras.SuperBlock, archivo *os.File, idxBloque int32) (*Estructuras.FileBlock, error) {
	bloque := &Estructuras.FileBlock{}
	offset := int64(sb.S_block_start + idxBloque*sb.S_block_size)
	err := bloque.Decodificar(archivo, offset)
	if err != nil {
		return nil, fmt.Errorf("error al decodificar bloque de archivo: %v", err)
	}
	return bloque, nil
}

// Busca un inodo dentro de un bloque de directorio
func buscarInodoEnDirectorio(inodo *Estructuras.INodo, archivo *os.File, nombre string, sb *Estructuras.SuperBlock) (bool, int32) {
	for _, idxBloque := range inodo.I_block {
		if idxBloque == -1 {
			continue
		}
		bloque := &Estructuras.FolderBlock{}
		offset := int64(sb.S_block_start + idxBloque*sb.S_block_size)
		err := bloque.Decodificar(archivo, offset)
		if err != nil {
			continue
		}
		for _, contenido := range bloque.B_cont {
			if string(contenido.B_name[:]) == nombre {
				return true, contenido.B_inodo
			}
		}
	}
	return false, -1
}
