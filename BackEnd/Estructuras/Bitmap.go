package Estructuras

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	BitBloqueLibre   = 0
	BitBloqueOcupado = 1
)

// Genera los Bitmaps de inodos y bloques en el archivo especificado
func (sb *SuperBlock) CrearBitMaps(archivo *os.File) error {
	err := sb.crearBitmap(archivo, sb.S_bm_inode_start, sb.S_inodes_count+sb.S_free_inodes_count, false)
	if err != nil {
		return fmt.Errorf("error generando bitmap de inodos: %w", err)
	}

	err = sb.crearBitmap(archivo, sb.S_bm_block_start, sb.S_blocks_count+sb.S_free_blocks_count, false)
	if err != nil {
		return fmt.Errorf("error generando bitmap de bloques: %w", err)
	}

	return nil
}

func (sb *SuperBlock) crearBitmap(archivo *os.File, inicio int32, cantidad int32, ocupado bool) error {
	_, err := archivo.Seek(int64(inicio), 0)
	if err != nil {
		return fmt.Errorf("error buscando el inicio del bitmap: %w", err)
	}

	cantidadBytes := (cantidad + 7) / 8

	// Crear el buffer de bytes con todos los bits en 0 (libres) o 1 (ocupados)
	byteRelleno := byte(0x00) // 00000000 (todos los bloques libres)
	if ocupado {
		byteRelleno = 0xFF // 11111111 (todos los bloques ocupados)
	}

	buffer := make([]byte, cantidadBytes)
	for i := range buffer {
		buffer[i] = byteRelleno
	}

	// Escribir el buffer en el archivo
	err = binary.Write(archivo, binary.LittleEndian, buffer)
	if err != nil {
		return fmt.Errorf("error escribiendo el bitmap: %w", err)
	}

	return nil
}

// Actualiza el bitmap de inodos
func (sb *SuperBlock) ActualizarBitmapInodo(archivo *os.File, posicion int32, ocupado bool) error {
	return sb.actualizarBitmap(archivo, sb.S_bm_inode_start, posicion, ocupado)
}

// Actualiza el bitmap de bloques
func (sb *SuperBlock) ActualizarBitmapBloque(archivo *os.File, posicion int32, ocupado bool) error {
	return sb.actualizarBitmap(archivo, sb.S_bm_block_start, posicion, ocupado)
}

// Funcion auxiliar que actualiza un bit en un bitmap
func (sb *SuperBlock) actualizarBitmap(archivo *os.File, inicio int32, posicion int32, ocupado bool) error {
	// Calcular el byte y el bit dentro de ese byte
	indiceByte := posicion / 8
	desplazamientoBit := posicion % 8

	// Mover el puntero al byte correspondiente
	_, err := archivo.Seek(int64(inicio)+int64(indiceByte), 0)
	if err != nil {
		return fmt.Errorf("error buscando la posicion en el bitmap: %w", err)
	}

	// Leer byte actual
	var valorByte byte
	err = binary.Read(archivo, binary.LittleEndian, &valorByte)
	if err != nil {
		return fmt.Errorf("error leyendo el byte del bitmap: %w", err)
	}

	// Actualizar el bit correspondiente dentro del byte
	if ocupado {
		valorByte |= (1 << desplazamientoBit) // Poner el bit a 1 (ocupado)
	} else {
		valorByte &= ^(1 << desplazamientoBit) // Poner el bit a 0 (libre)
	}

	_, err = archivo.Seek(int64(inicio)+int64(indiceByte), 0)
	if err != nil {
		return fmt.Errorf("error buscando la posicion en el bitmap para escribir: %w", err)
	}

	// Escribir el byte actualizado de vuelta en el archivo
	err = binary.Write(archivo, binary.LittleEndian, &valorByte)
	if err != nil {
		return fmt.Errorf("error escribiendo el byte actualizado del bitmap: %w", err)
	}

	return nil
}

// Verifica si un bloque en el bitmap esta libre
func (sb *SuperBlock) verificarBloqueLibre(archivo *os.File, inicio int32, posicion int32) (bool, error) {
	// Calcular el byte y el bit dentro del byte
	indiceByte := posicion / 8
	desplazamientoBit := posicion % 8

	_, err := archivo.Seek(int64(inicio)+int64(indiceByte), 0)
	if err != nil {
		return false, fmt.Errorf("error buscando la posicion en el bitmap: %w", err)
	}

	var valorByte byte
	err = binary.Read(archivo, binary.LittleEndian, &valorByte)
	if err != nil {
		return false, fmt.Errorf("error leyendo el byte del bitmap: %w", err)
	}

	return (valorByte & (1 << desplazamientoBit)) == 0, nil
}

// Verifica si un inodo en el bitmap esta libre
func (sb *SuperBlock) verificarInodoLibre(archivo *os.File, inicio int32, posicion int32) (bool, error) {
	indiceByte := posicion / 8        // Calcular el byte dentro del bitmap
	desplazamientoBit := posicion % 8 // Calcular el bit dentro del byte

	_, err := archivo.Seek(int64(inicio)+int64(indiceByte), 0)
	if err != nil {
		return false, fmt.Errorf("error buscando el byte en el bitmap de inodos: %w", err)
	}

	var valorByte byte
	err = binary.Read(archivo, binary.LittleEndian, &valorByte)
	if err != nil {
		return false, fmt.Errorf("error leyendo el byte del bitmap de inodos: %w", err)
	}

	return (valorByte & (1 << desplazamientoBit)) == 0, nil
}

// LiberarBloque libera un bloque específico, lo marca como disponible en el bitmap y borra su contenido
func (sb *SuperBlock) LiberarBloque(archivo *os.File, indiceBloque int32) error {
    // Calcular la posición del bloque en el archivo
    offsetBloque := int64(sb.S_block_start + indiceBloque*sb.S_block_size)
    
    // Crear buffer de ceros para limpiar completamente el bloque
    bufferCeros := make([]byte, sb.S_block_size)
    
    // Sobrescribir el contenido del bloque con ceros
    _, err := archivo.WriteAt(bufferCeros, offsetBloque)
    if err != nil {
        return fmt.Errorf("error al borrar el contenido del bloque %d: %w", indiceBloque, err)
    }

    // Marcar el bloque como disponible en el bitmap correspondiente
    err = sb.ActualizarBitmapBloque(archivo, indiceBloque, false)
    if err != nil {
        return fmt.Errorf("error al marcar como libre el bloque %d: %w", indiceBloque, err)
    }
    
    // Incrementar contador de bloques libres en el superbloque
    sb.ActualizarSuperblockDespuesDesasignacionBloque()
    
    return nil
}

// LiberarInodo libera un inodo específico, lo marca como disponible en el bitmap y reinicia sus metadatos
func (sb *SuperBlock) LiberarInodo(archivo *os.File, indiceInodo int32) error {
    // Instanciar estructura para manipular el inodo
    inodo := &INodo{}
    
    // Calcular ubicación del inodo en el archivo
    offsetInodo := int64(sb.S_inode_start + indiceInodo*sb.S_inode_size)
    
    // Cargar el inodo desde el disco para su limpieza
    err := inodo.Decodificar(archivo, offsetInodo)
    if err != nil {
        return fmt.Errorf("error al cargar inodo %d para su liberación: %w", indiceInodo, err)
    }
    
    // Reiniciar todos los metadatos del inodo a valores por defecto
    inodo.I_size = 0                    // Tamaño del archivo a cero
    inodo.I_type[0] = '0'              // Restablecer tipo de inodo
    // Inicializar todos los bloques como no asignados
    inodo.I_block = [15]int32{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}

    // Escribir el inodo limpio de vuelta al disco
    err = inodo.Codificar(archivo, offsetInodo)
    if err != nil {
        return fmt.Errorf("error al escribir el inodo limpio %d: %w", indiceInodo, err)
    }
    
    // Marcar el inodo como disponible en el bitmap de inodos
    err = sb.ActualizarBitmapInodo(archivo, indiceInodo, false)
    if err != nil {
        return fmt.Errorf("error al marcar como libre el inodo %d: %w", indiceInodo, err)
    }

    // Incrementar contador de inodos libres en el superbloque
    sb.ActualizarSuperblockDespuesDesasignacionInodo()

    return nil
}