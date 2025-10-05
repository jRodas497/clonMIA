package Estructuras

import (
	"fmt"
	"os"
	"strings"
	"time"

	Utils "backend/Utils"
)

// Genera una carpeta dentro de un inodo especifico
func (sb *SuperBlock) crearCarpetaEnInodo(archivo *os.File, indiceInodo int32, directoriosPadre []string, directorioDestino string) error {
	// Instanciar nuevo inodo
	inodo := &INodo{}

	// Deserializar el inodo
	err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
	if err != nil {
		return fmt.Errorf("error al deserializar inodo %d: %v", indiceInodo, err)
	}

	// Verificar si el inodo corresponde a una carpeta
	if inodo.I_type[0] != '0' {
		return nil
	}

	// Recorrer cada bloque del inodo (punteros)
	for _, indiceBloques := range inodo.I_block {
		// Si el bloque no existe, terminar
		if indiceBloques == -1 {
			break
		}

		bloque := &FolderBlock{}

		err := bloque.Decodificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
		if err != nil {
			return fmt.Errorf("error al deserializar bloque %d: %v", indiceBloques, err)
		}

		for indiceContenido := 2; indiceContenido < len(bloque.B_cont); indiceContenido++ {
			contenido := bloque.B_cont[indiceContenido]

			// Si existen directorios padre en la ruta
			if len(directoriosPadre) != 0 {
				// Si el contenido esta vacio, salir
				if contenido.B_inodo == -1 {
					fmt.Printf("No se encontró espacio para el directorio padre en inodo %d en la posición %d, terminando.\n", indiceInodo, indiceContenido)
					break
				}

				// Obtener el directorio padre mas proximo
				directorioPadre, err := Utils.Primero(directoriosPadre)
				if err != nil {
					return err
				}

				nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")
				nombreDirectorioPadre := strings.Trim(directorioPadre, "\x00 ")

				// Nombre coincide con directorio padre
				if strings.EqualFold(nombreContenido, nombreDirectorioPadre) {
					// Llamada recursiva para continuar creando carpetas
					err := sb.crearCarpetaEnInodo(archivo, contenido.B_inodo, Utils.EliminarElemento(directoriosPadre, 0), directorioDestino)
					if err != nil {
						return err
					}
					return nil
				}
			} else {
				if contenido.B_inodo != -1 {
					fmt.Printf("Inodo %d está ocupado, yendo al siguiente.\n", contenido.B_inodo)
					continue
				}

				// Asignar nombre del directorio al bloque
				copy(contenido.B_name[:], directorioDestino)
				contenido.B_inodo = sb.S_inodes_count

				// Actualizar bloque con nuevo contenido
				bloque.B_cont[indiceContenido] = contenido

				// Serializar el bloque
				err = bloque.Codificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
				if err != nil {
					return fmt.Errorf("error al serializar el bloque %d: %v", indiceBloques, err)
				}

				// Crear inodo de la nueva carpeta
				inodoCarpeta := &INodo{
					I_uid:   1,
					I_gid:   1,
					I_size:  0,
					I_atime: float32(time.Now().Unix()),
					I_ctime: float32(time.Now().Unix()),
					I_mtime: float32(time.Now().Unix()),
					I_block: [15]int32{sb.S_blocks_count, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
					I_type:  [1]byte{'0'}, // Tipo carpeta
					I_perm:  [3]byte{'6', '6', '4'},
				}

				// Serializar inodo de la nueva carpeta
				err = inodoCarpeta.Codificar(archivo, int64(sb.S_first_ino))
				if err != nil {
					return fmt.Errorf("error al serializar el inodo del directorio '%s': %v", directorioDestino, err)
				}

				// Actualizar bitmap de inodos
				err = sb.ActualizarBitmapInodo(archivo, sb.S_inodes_count, true)
				if err != nil {
					return fmt.Errorf("error al actualizar el bitmap de inodos para el directorio '%s': %v", directorioDestino, err)
				}

				// Actualizar superbloque tras asignacion de inodo
				sb.ActualizarSuperblockDespuesAsignacionInodo()

				// Generar bloque para la nueva carpeta
				bloqueCarpeta := &FolderBlock{
					B_cont: [4]FolderContent{
						{B_name: [12]byte{'.'}, B_inodo: contenido.B_inodo},
						{B_name: [12]byte{'.', '.'}, B_inodo: indiceInodo},
						{B_name: [12]byte{'-'}, B_inodo: -1},
						{B_name: [12]byte{'-'}, B_inodo: -1},
					},
				}

				// Serializar bloque de la carpeta
				err = bloqueCarpeta.Codificar(archivo, int64(sb.S_first_blo))
				if err != nil {
					return fmt.Errorf("error al serializar el bloque del directorio '%s': %v", directorioDestino, err)
				}

				// Actualizar bitmap de bloques
				err = sb.ActualizarBitmapBloque(archivo, sb.S_blocks_count, true)
				if err != nil {
					return fmt.Errorf("error al actualizar el bitmap de bloques para el directorio '%s': %v", directorioDestino, err)
				}

				// Actualizar superbloque tras asignacion de bloque
				sb.ActualizarSuperblockDespuesAsignacionBloque()

				fmt.Printf("Directorio '%s' creado correctamente en inodo %d.\n", directorioDestino, sb.S_inodes_count) // Depuración
				return nil
			}
		}
	}

	fmt.Printf("No se encontraron bloques disponibles para crear la carpeta '%s' en inodo %d\n", directorioDestino, indiceInodo)
	return nil
}

// CrearCarpeta genera una carpeta en el sistema de archivos
func (sb *SuperBlock) CrearCarpeta(archivo *os.File, directoriosPadre []string, directorioDestino string) error {
	// Si directoriosPadre esta vacio trabajar solo con inodo raiz
	if len(directoriosPadre) == 0 {
		return sb.crearCarpetaEnInodo(archivo, 0, directoriosPadre, directorioDestino)
	}

	// Recorrer inodo para el inodo padre
	for i := int32(0); i < sb.S_inodes_count; i++ {
		// Desde inodo 0
		err := sb.crearCarpetaEnInodo(archivo, i, directoriosPadre, directorioDestino)
		if err != nil {
			return err
		}
	}

	return nil
}
