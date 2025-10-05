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
