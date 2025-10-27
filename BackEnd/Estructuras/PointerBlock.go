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

// LeerIndireccioSimple lee los bloques a través de indirección simple
func (ba *PointerBlock) LeerIndireccioSimple(archivo *os.File, sb *SuperBlock) ([]int32, error) {
    var bloques []int32
    for _, apuntador := range ba.B_apuntadores {
        if apuntador != -1 {
            bloques = append(bloques, int32(apuntador))
        }
    }
    return bloques, nil
}

// LeerIndireccioDoble lee los bloques a través de indirección doble
func (ba *PointerBlock) LeerIndireccioDoble(archivo *os.File, sb *SuperBlock) ([]int32, error) {
    var bloques []int32
    for _, apuntador := range ba.B_apuntadores {
        if apuntador != -1 {
            // Leer el bloque de apuntadores secundario
            bloqueSecundario := &PointerBlock{}
            err := bloqueSecundario.Decodificar(archivo, int64(sb.S_block_start+int32(apuntador)*sb.S_block_size))
            if err != nil {
                return nil, err
            }
            // Obtener los bloques de este nivel
            bloquesSecundarios, err := bloqueSecundario.LeerIndireccioSimple(archivo, sb)
            if err != nil {
                return nil, err
            }
            bloques = append(bloques, bloquesSecundarios...)
        }
    }
    return bloques, nil
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

// LiberarSiVacio libera el bloque si está vacío y actualiza referencias
func (ba *PointerBlock) LiberarSiVacio(archivo *os.File, sb *SuperBlock, indiceBloques int32, inodoPadre *INodo, indiceApuntador int) error {
    if ba.EstaVacio() {
        // Marcar bloque como libre en bitmap
        if err := sb.ActualizarBitmapBloque(archivo, indiceBloques, false); err != nil {
            return err
        }

        // Actualizar referencia en inodo padre
        if inodoPadre != nil && indiceApuntador >= 0 {
            inodoPadre.I_block[indiceApuntador] = -1
            return inodoPadre.Codificar(archivo, sb.CalcularDesplazamientoInodo(inodoPadre.I_uid))
        }

        // Actualizar contador en superbloque
        sb.ActualizarSuperblockDespuesDesasignacionBloque()
    }
    return nil
}

// EstaVacio verifica si todos los apuntadores están libres (-1)
func (ba *PointerBlock) EstaVacio() bool {
    return ba.ContarApuntadoresLibres() == len(ba.B_apuntadores)
}

// EstablecerApuntador establece un valor específico en un índice dado
func (ba *PointerBlock) EstablecerApuntador(indice int, valor int64) error {
    if indice < 0 || indice >= len(ba.B_apuntadores) {
        return fmt.Errorf("índice %d fuera de rango [0-%d]", indice, len(ba.B_apuntadores)-1)
    }
    ba.B_apuntadores[indice] = valor
    return nil
}

// ObtenerApuntador obtiene el valor de un apuntador en un índice dado
func (ba *PointerBlock) ObtenerApuntador(indice int) (int64, error) {
    if indice < 0 || indice >= len(ba.B_apuntadores) {
        return -1, fmt.Errorf("índice %d fuera de rango [0-%d]", indice, len(ba.B_apuntadores)-1)
    }
    return ba.B_apuntadores[indice], nil
}

// EstaLleno verifica si todos los apuntadores están ocupados
func (ba *PointerBlock) EstaLleno() bool {
    for _, apuntador := range ba.B_apuntadores {
        if apuntador == -1 {
            return false
        }
    }
    return true
}

// ContarApuntadoresLibres cuenta cuántos apuntadores libres hay en el bloque
func (ba *PointerBlock) ContarApuntadoresLibres() int {
    contador := 0
    for _, apuntador := range ba.B_apuntadores {
        if apuntador == -1 {
            contador++
        }
    }
    return contador
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

// LeerIndireccioTriple lee los bloques a través de indirección triple
func (ba *PointerBlock) LeerIndireccioTriple(archivo *os.File, sb *SuperBlock) ([]int32, error) {
    var bloques []int32
    for _, apuntadorPrimario := range ba.B_apuntadores {
        if apuntadorPrimario != -1 {
            // Leer el bloque de apuntadores secundario
            bloqueSecundario := &PointerBlock{}
            offsetSecundario := int64(sb.S_block_start + int32(apuntadorPrimario)*sb.S_block_size)
            if err := bloqueSecundario.Decodificar(archivo, offsetSecundario); err != nil {
                return nil, fmt.Errorf("error leyendo bloque secundario: %w", err)
            }

            for _, apuntadorSecundario := range bloqueSecundario.B_apuntadores {
                if apuntadorSecundario != -1 {
                    // Leer el bloque de apuntadores terciario
                    bloqueTerciario := &PointerBlock{}
                    offsetTerciario := int64(sb.S_block_start + int32(apuntadorSecundario)*sb.S_block_size)
                    if err := bloqueTerciario.Decodificar(archivo, offsetTerciario); err != nil {
                        return nil, fmt.Errorf("error leyendo bloque terciario: %w", err)
                    }

                    // Obtener los bloques de este nivel
                    bloquesTerciarios, err := bloqueTerciario.LeerIndireccioSimple(archivo, sb)
                    if err != nil {
                        return nil, err
                    }
                    bloques = append(bloques, bloquesTerciarios...)
                }
            }
        }
    }
    return bloques, nil
}