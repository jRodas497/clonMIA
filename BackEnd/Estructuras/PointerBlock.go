package Estructuras

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Estructura para almacenar bloques de punteros
type PointerBlock struct {
	B_apuntadores [16]int64
	// Punteros a bloques de carpetas o datos
}

// Localiza el primer puntero disponible y devuelve su indice
func (ba *PointerBlock) BuscarApuntadorLibre() (int, error) {
	// -1 o 0 indican punteros no asignados
	for i, puntero := range ba.B_apuntadores {
		if puntero == -1 || puntero == 0 { 
			return i, nil
		}
	}
	return -1, fmt.Errorf("no hay punteros disponibles en el bloque de apuntadores")
}

// Codificar serializa el PointerBlock en el archivo en la posicion especificada
func (ba *PointerBlock) Codificar(archivo *os.File, desplazamiento int64) error {
	_, err := archivo.Seek(desplazamiento, 0)
	if err != nil {
		return fmt.Errorf("error posicionando en el archivo: %w", err)
	}

	// Escribir estructura PointerBlock en el archivo
	err = binary.Write(archivo, binary.BigEndian, ba)
	if err != nil {
		return fmt.Errorf("error escribiendo el PointerBlock: %w", err)
	}
	return nil
}

// Decodificar deserializa el PointerBlock desde el archivo en la posicion especificada
func (ba *PointerBlock) Decodificar(archivo *os.File, desplazamiento int64) error {
	_, err := archivo.Seek(desplazamiento, 0)
	if err != nil {
		return fmt.Errorf("error posicionando en el archivo: %w", err)
	}

	// Leer estructura PointerBlock desde el archivo
	err = binary.Read(archivo, binary.BigEndian, ba)
	if err != nil {
		return fmt.Errorf("error leyendo el PointerBlock: %w", err)
	}
	return nil
}

