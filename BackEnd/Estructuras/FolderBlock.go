package Estructuras

import (
	"fmt"
	"os"

	Utils "backend/Utils"
)

// Representa un bloque de carpeta de 4 contenidos
type FolderBlock struct {
	/*  Total: 64 bytes  */
	B_cont [4]FolderContent
}

// Representa el contenido dentro de un bloque de carpeta 
type FolderContent struct {
	/*  Total: 16 bytes  */
	B_name  [12]byte
	B_inodo int32
}

// Serializa la estructura FolderBlock en un archivo binario en la posicion especificada
func (bc *FolderBlock) Codificar(archivo *os.File, desplazamiento int64) error {
	err := Utils.EscribirAArchivo(archivo, desplazamiento, bc)
	if err != nil {
		return fmt.Errorf("error escribiendo FolderBlock al archivo: %w", err)
	}
	return nil
}

// Deserializa la estructura FolderBlock desde un archivo binario en la posicion especificada
func (bc *FolderBlock) Decodificar(archivo *os.File, desplazamiento int64) error {
	err := Utils.LeerDeArchivo(archivo, desplazamiento, bc)
	if err != nil {
		return fmt.Errorf("error leyendo FolderBlock desde archivo: %w", err)
	}
	return nil
}

func (bc *FolderBlock) Imprimir() {
	for i, contenido := range bc.B_cont {
		nombre := string(contenido.B_name[:])
		fmt.Printf("Contenido %d:\n", i+1)
		fmt.Printf("  Nombre: %s\n", nombre)
		fmt.Printf("  Inodo: %d\n", contenido.B_inodo)
	}
}

// NuevoBloqueDirectorio crea un bloque de carpeta inicial con entradas dadas
func NuevoBloqueDirectorio(selfInodo int32, parentInodo int32, entradas map[string]int32) *FolderBlock {
	fb := &FolderBlock{}
	// Inicializar con entradas vacías
	for i := range fb.B_cont {
		fb.B_cont[i].B_name = [12]byte{}
		fb.B_cont[i].B_inodo = -1
	}

	// '.'
	copy(fb.B_cont[0].B_name[:], ".")
	fb.B_cont[0].B_inodo = selfInodo

	// '..'
	copy(fb.B_cont[1].B_name[:], "..")
	fb.B_cont[1].B_inodo = parentInodo

	// Rellenar con entradas adicionales hasta 2 más
	idx := 2
	for nombre, inodo := range entradas {
		if idx >= len(fb.B_cont) {
			break
		}
		copy(fb.B_cont[idx].B_name[:], nombre)
		fb.B_cont[idx].B_inodo = inodo
		idx++
	}

	return fb
}
