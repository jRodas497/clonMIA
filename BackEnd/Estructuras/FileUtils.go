package Estructuras

import (
	Utils "backend/Utils"
	"fmt"
	"os"
	"strings"
	"time"
)

// Se genera un archivo dentro de un inodo especifico
func (sb *SuperBlock) crearArchivoEnInodo(archivo *os.File, indiceInodo int32, directoriosPadre []string, archivoDestino string, dimensionArchivo int, contenidoArchivo []string) error {
	// Instanciar nuevo inodo
	inodo := &INodo{}

	// Deserializar el inodo
	err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
	if err != nil {
		return fmt.Errorf("error al deserializar inodo %d: %v", indiceInodo, err)
	}

	// Verificar si el inodo es de tipo archivo
	if inodo.I_type[0] == '1' {
		return nil
	}

	// Recorrer cada bloque del inodo (punteros)
	for _, indiceBloques := range inodo.I_block {
		// Si el bloque no existe, terminar
		if indiceBloques == -1 {
			break
		}

		// Instanciar nuevo bloque de carpeta
		bloque := &FolderBlock{}

		// Deserializar el bloque
		err := bloque.Decodificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
		if err != nil {
			return fmt.Errorf("error al deserializar bloque %d: %v", indiceBloques, err)
		}

		// Procesar cada elemento del bloque, omitiendo . y ..
		for indiceContenido := 2; indiceContenido < len(bloque.B_cont); indiceContenido++ {
			// Obtener contenido del bloque
			contenido := bloque.B_cont[indiceContenido]

			if len(directoriosPadre) != 0 {
				if contenido.B_inodo == -1 {
					break
				}

				directorioPadre, err := Utils.Primero(directoriosPadre)
				if err != nil {
					return err
				}

				nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")
				nombreDirectorioPadre := strings.Trim(directorioPadre, "\x00 ")

				// Si el nombre coincide con el directorio padre
				if strings.EqualFold(nombreContenido, nombreDirectorioPadre) {
					err := sb.crearArchivoEnInodo(archivo, contenido.B_inodo, Utils.EliminarElemento(directoriosPadre, 0), archivoDestino, dimensionArchivo, contenidoArchivo)
					if err != nil {
						return err
					}
					return nil
				}
			} else {
				// Continuar con el siguiente si esta ocupado
				if contenido.B_inodo != -1 {
					continue
				}

				// Actualizar contenido del bloque
				copy(contenido.B_name[:], []byte(archivoDestino))
				contenido.B_inodo = sb.S_inodes_count

				// Actualizar el bloque
				bloque.B_cont[indiceContenido] = contenido

				// Serializar el bloque
				err = bloque.Codificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
				if err != nil {
					return fmt.Errorf("error al serializar bloque %d: %v", indiceBloques, err)
				}

				inodoArchivo := &INodo{
					I_uid:   1,
					I_gid:   1,
					I_size:  int32(dimensionArchivo),
					I_atime: float32(time.Now().Unix()),
					I_ctime: float32(time.Now().Unix()),
					I_mtime: float32(time.Now().Unix()),
					I_block: [15]int32{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
					I_type:  [1]byte{'1'},
					I_perm:  [3]byte{'6', '6', '4'},
				}

				// Crear bloques del archivo
				for i := 0; i < len(contenidoArchivo); i++ {
					inodoArchivo.I_block[i] = sb.S_blocks_count

					// Crear bloque del archivo
					bloqueArchivo := &FileBlock{
						B_cont: [64]byte{},
					}
					copy(bloqueArchivo.B_cont[:], contenidoArchivo[i])

					// Serializar el bloque
					err = bloqueArchivo.Codificar(archivo, int64(sb.S_first_blo))
					if err != nil {
						return fmt.Errorf("error al serializar bloque de archivo: %v", err)
					}

					// Actualizar bitmap de bloques
					err = sb.ActualizarBitmapBloque(archivo, sb.S_blocks_count, true)
					if err != nil {
						return fmt.Errorf("error al actualizar bitmap de bloque: %v", err)
					}

					// Actualizar superbloque tras asignacion de bloque
					sb.ActualizarSuperblockDespuesAsignacionBloque()
				}

				err = inodoArchivo.Codificar(archivo, int64(sb.S_first_ino))
				if err != nil {
					return fmt.Errorf("error al serializar inodo del archivo: %v", err)
				}

				// Actualizar bitmap de inodos
				err = sb.ActualizarBitmapInodo(archivo, sb.S_inodes_count, true)
				if err != nil {
					return fmt.Errorf("error al actualizar bitmap de inodo: %v", err)
				}

				// Actualizar superbloque tras asignacion de inodo
				sb.ActualizarSuperblockDespuesAsignacionInodo()

				return nil
			}
		}
	}
	return nil
}

// Se genera un archivo en el sistema de archivos
func (sb *SuperBlock) CrearArchivo(archivo *os.File, directoriosPadre []string, archivoDestino string, dimension int, contenido []string) error {
	fmt.Printf("Creando archivo '%s' con tamaño %d\n", archivoDestino, dimension)
	if len(directoriosPadre) == 0 {
		return sb.crearArchivoEnInodo(archivo, 0, directoriosPadre, archivoDestino, dimension, contenido)
	}

	// Recorrer cada inodo para buscar el inodo padre
	for i := int32(0); i < sb.S_inodes_count; i++ {
		err := sb.crearArchivoEnInodo(archivo, i, directoriosPadre, archivoDestino, dimension, contenido)
		if err != nil {
			return err
		}
	}

	fmt.Printf("Archivo '%s' creado exitosamente.\n", archivoDestino)
	return nil
}
