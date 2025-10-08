package Estructuras

import (
	"fmt"
	"os"
	"time"

	Utils "backend/Utils"
)

type INodo struct {

	/* 88 bytes */

	I_uid   int32     /* UID del usuario propietario del archivo */
	I_gid   int32     /* GID del grupo propietario del archivo */
	I_size  int32     /* Tamaño del archivo en bytes */
	I_atime float32   /* Último acceso al archivo */
	I_ctime float32   /* Último cambio de permisos */
	I_mtime float32   /* Última modificación del archivo */
	I_type  [1]byte   /* Indica si es archivo o carpeta 1=archivo, 0=carpeta */
	I_perm  [3]byte   /* Guarda los permisos del archivo */
	I_block [15]int32 /* 12 bloques directos, 1 indirecto simple, 1 indirecto doble, 1 indirecto triple */
}

func (inodo *INodo) Codificar(archivo *os.File, desplazamiento int64) error {
	err := Utils.EscribirAArchivo(archivo, desplazamiento, inodo)
	if err != nil {
		return fmt.Errorf("error escribiendo INodo al archivo: %w", err)
	}
	return nil
}

func (inodo *INodo) Decodificar(archivo *os.File, desplazamiento int64) error {
	err := Utils.LeerDeArchivo(archivo, desplazamiento, inodo)
	if err != nil {
		return fmt.Errorf("error leyendo INodo desde archivo: %w", err)
	}
	return nil
}

// CrearInodo - Crear y serializar un inodo, actualizando el bitmap de inodos
func (inodo *INodo) CrearInodo(
    archivo *os.File,
    sb *SuperBlock,
    tipoInodo byte, // Tipo de inodo (0 para directorio, 1 para archivo)
    tamaño int32,
    bloques [15]int32,
    permisos [3]byte,
) error {
    indiceInodo, err := sb.AsignarNuevoInodo(archivo)
    if err != nil {
        return fmt.Errorf("error asignando nuevo inodo: %w", err)
    }

    inodo.I_uid = 1
    inodo.I_gid = 1
    inodo.I_size = tamaño
    inodo.I_atime = float32(time.Now().Unix())
    inodo.I_ctime = float32(time.Now().Unix())
    inodo.I_mtime = float32(time.Now().Unix())
    inodo.I_block = bloques
    inodo.I_type = [1]byte{tipoInodo}
    inodo.I_perm = permisos

    err = sb.ActualizarBitmapInodo(archivo, indiceInodo, true)
    if err != nil {
        return fmt.Errorf("error actualizando el bitmap de inodos: %w", err)
    }

    desplazamientoInodo := int64(sb.S_inode_start + (indiceInodo * sb.S_inode_size))
    err = inodo.Codificar(archivo, desplazamientoInodo)
    if err != nil {
        return fmt.Errorf("error serializando el inodo en la posición %d: %w", desplazamientoInodo, err)
    }

    return nil
}

func (inodo *INodo) ActualizarTiempoAcceso() {
	inodo.I_atime = float32(time.Now().Unix())
}

func (inodo *INodo) ActualizarTiempoModificacion() {
	inodo.I_mtime = float32(time.Now().Unix())
}

func (inodo *INodo) ActualizarTiempoPermisos() {
	inodo.I_ctime = float32(time.Now().Unix())
}

// Imprimir atributos del inodo
func (inodo *INodo) Imprimir() {
	tiempoAcceso := time.Unix(int64(inodo.I_atime), 0)
	tiempoPermisos := time.Unix(int64(inodo.I_ctime), 0)
	tiempoModificacion := time.Unix(int64(inodo.I_mtime), 0)

	fmt.Printf("UID propietario: %d\n", inodo.I_uid)
	fmt.Printf("GID grupo: %d\n", inodo.I_gid)
	fmt.Printf("Dimension archivo: %d bytes\n", inodo.I_size)
	fmt.Printf("Ultimo acceso: %s\n", tiempoAcceso.Format(time.RFC3339))
	fmt.Printf("Ultimo cambio de permisos: %s\n", tiempoPermisos.Format(time.RFC3339))
	fmt.Printf("Ultima modificacion: %s\n", tiempoModificacion.Format(time.RFC3339))
	fmt.Printf("Bloques asignados: %v\n", inodo.I_block)
	fmt.Printf("Tipo de elemento: %s\n", string(inodo.I_type[:]))
	fmt.Printf("Permisos: %s\n", string(inodo.I_perm[:]))
}

// ObtenerTodosLosIndicesDeBloques devuelve todos los índices de bloques utilizados por este inodo
func (inodo *INodo) ObtenerTodosLosIndicesDeBloques(archivo *os.File, sb *SuperBlock) ([]int32, error) {
    var indicesBloques []int32

    // 1. Obtener los bloques directos (0-11)
    for i := 0; i < 12; i++ {
        if inodo.I_block[i] != -1 {
            indicesBloques = append(indicesBloques, inodo.I_block[i])
        }
    }

    // 2. Procesar bloque indirecto simple (posición 12)
    if inodo.I_block[12] != -1 {
        // Agregar el índice del bloque de apuntadores simple
        indicesBloques = append(indicesBloques, inodo.I_block[12])

        // Cargar el bloque de apuntadores
        ba := &PointerBlock{}
        offsetBA := int64(sb.S_block_start + inodo.I_block[12]*sb.S_block_size)
        err := ba.Decodificar(archivo, offsetBA)
        if err != nil {
            return nil, fmt.Errorf("error leyendo bloque indirecto simple: %w", err)
        }

        // Obtener los bloques apuntados por este bloque
        for _, apuntador := range ba.B_apuntadores {
            if apuntador != -1 {
                indicesBloques = append(indicesBloques, int32(apuntador))
            }
        }
    }

    // 3. Procesar bloque indirecto doble (posición 13)
    if inodo.I_block[13] != -1 {
        indicesBloques = append(indicesBloques, inodo.I_block[13])

        // Cargar el bloque de apuntadores primario
        baPrimario := &PointerBlock{}
        offsetPrimario := int64(sb.S_block_start + inodo.I_block[13]*sb.S_block_size)
        err := baPrimario.Decodificar(archivo, offsetPrimario)
        if err != nil {
            return nil, fmt.Errorf("error leyendo bloque indirecto doble: %w", err)
        }

        // Para cada apuntador válido en el bloque primario
        for _, apuntadorPrimario := range baPrimario.B_apuntadores {
            if apuntadorPrimario != -1 {
                // Agregamos el índice del bloque secundario
                indicesBloques = append(indicesBloques, int32(apuntadorPrimario))

                // Cargamos el bloque secundario
                baSecundario := &PointerBlock{}
                offsetSecundario := int64(sb.S_block_start + int32(apuntadorPrimario)*sb.S_block_size)
                err := baSecundario.Decodificar(archivo, offsetSecundario)
                if err != nil {
                    return nil, fmt.Errorf("error leyendo bloque secundario: %w", err)
                }

                // Agregamos todos los bloques apuntados por el secundario
                for _, apuntadorSecundario := range baSecundario.B_apuntadores {
                    if apuntadorSecundario != -1 {
                        indicesBloques = append(indicesBloques, int32(apuntadorSecundario))
                    }
                }
            }
        }
    }

    // 4. Procesar bloque indirecto triple (posición 14)
    if inodo.I_block[14] != -1 {
        indicesBloques = append(indicesBloques, inodo.I_block[14])
        baPrimario := &PointerBlock{}
        offsetPrimario := int64(sb.S_block_start + inodo.I_block[14]*sb.S_block_size)
        err := baPrimario.Decodificar(archivo, offsetPrimario)
        if err != nil {
            return nil, fmt.Errorf("error leyendo bloque indirecto triple: %w", err)
        }

        // Para cada apuntador válido en el bloque primario
        for _, apuntadorPrimario := range baPrimario.B_apuntadores {
            if apuntadorPrimario != -1 {
                // Agregamos el índice del bloque secundario
                indicesBloques = append(indicesBloques, int32(apuntadorPrimario))

                // Cargamos el bloque secundario (nivel 2)
                baSecundario := &PointerBlock{}
                offsetSecundario := int64(sb.S_block_start + int32(apuntadorPrimario)*sb.S_block_size)
                err := baSecundario.Decodificar(archivo, offsetSecundario)
                if err != nil {
                    return nil, fmt.Errorf("error leyendo bloque secundario en triple: %w", err)
                }

                // Para cada apuntador válido en el bloque secundario
                for _, apuntadorSecundario := range baSecundario.B_apuntadores {
                    if apuntadorSecundario != -1 {
                        indicesBloques = append(indicesBloques, int32(apuntadorSecundario))
                        baTerciario := &PointerBlock{}
                        offsetTerciario := int64(sb.S_block_start + int32(apuntadorSecundario)*sb.S_block_size)
                        err := baTerciario.Decodificar(archivo, offsetTerciario)
                        if err != nil {
                            return nil, fmt.Errorf("error leyendo bloque terciario: %w", err)
                        }

                        // Agregamos todos los bloques apuntados por el terciario
                        for _, apuntadorTerciario := range baTerciario.B_apuntadores {
                            if apuntadorTerciario != -1 {
                                indicesBloques = append(indicesBloques, int32(apuntadorTerciario))
                            }
                        }
                    }
                }
            }
        }
    }

    return indicesBloques, nil
}

// AgregarBloque asigna un nuevo bloque al inodo y devuelve su índice
func (inodo *INodo) AgregarBloque(archivo *os.File, sb *SuperBlock) (int32, error) {
    // 1. Intentar asignar en bloques directos (0-11)
    for i := 0; i < 12; i++ {
        if inodo.I_block[i] == -1 {
            nuevoBloque, err := sb.AsignarNuevoBloque(archivo, inodo, i)
            if err != nil {
                return -1, fmt.Errorf("error al asignar bloque directo: %w", err)
            }

            inodo.ActualizarTiempoModificacion()
            return nuevoBloque, nil
        }
    }

    return inodo.AgregarBloqueConIndirección(archivo, sb)
}

// AgregarBloqueConIndirección maneja la asignación de bloques usando los niveles de indirección
func (inodo *INodo) AgregarBloqueConIndirección(archivo *os.File, sb *SuperBlock) (int32, error) {
    // Gestión de indirección simple (posición 12)
    if inodo.I_block[12] == -1 {
        indiceBloqueApuntadores, err := sb.AsignarNuevoBloque(archivo, inodo, 12)
        if err != nil {
            return -1, fmt.Errorf("error al crear bloque de apuntadores simple: %w", err)
        }

        ba := &PointerBlock{}
        for i := range ba.B_apuntadores {
            ba.B_apuntadores[i] = -1
        }

        offsetBA := int64(sb.S_block_start + indiceBloqueApuntadores*sb.S_block_size)
        if err := ba.Codificar(archivo, offsetBA); err != nil {
            return -1, fmt.Errorf("error al escribir bloque de apuntadores simple: %w", err)
        }
    }

    // Intentar agregar un bloque usando indirección simple
    indiceBloquesDatos, err := inodo.AgregarBloqueAIndireccionSimple(archivo, sb)
    if err == nil {
        return indiceBloquesDatos, nil
    }

    // Gestión de indirección doble (posición 13)
    if inodo.I_block[13] == -1 {
        indiceApuntadorDoble, err := sb.AsignarNuevoBloque(archivo, inodo, 13)
        if err != nil {
            return -1, fmt.Errorf("error al crear bloque de apuntadores doble: %w", err)
        }

        // Inicializar el bloque de apuntadores doble
        baDoble := &PointerBlock{}
        for i := range baDoble.B_apuntadores {
            baDoble.B_apuntadores[i] = -1
        }

        offsetBADoble := int64(sb.S_block_start + indiceApuntadorDoble*sb.S_block_size)
        if err := baDoble.Codificar(archivo, offsetBADoble); err != nil {
            return -1, fmt.Errorf("error al escribir bloque de apuntadores doble: %w", err)
        }
    }

    // Intentar agregar un bloque usando indirección doble
    indiceBloquesDatos, err = inodo.AgregarBloqueAIndireccionDoble(archivo, sb)
    if err == nil {
        return indiceBloquesDatos, nil
    }

    // Gestión de indirección triple (posición 14)
    if inodo.I_block[14] == -1 {
        indiceApuntadorTriple, err := sb.AsignarNuevoBloque(archivo, inodo, 14)
        if err != nil {
            return -1, fmt.Errorf("error al crear bloque de apuntadores triple: %w", err)
        }

        // Inicializar el bloque de apuntadores triple
        baTriple := &PointerBlock{}
        for i := range baTriple.B_apuntadores {
            baTriple.B_apuntadores[i] = -1
        }

        offsetBATriple := int64(sb.S_block_start + indiceApuntadorTriple*sb.S_block_size)
        if err := baTriple.Codificar(archivo, offsetBATriple); err != nil {
            return -1, fmt.Errorf("error al escribir bloque de apuntadores triple: %w", err)
        }
    }

    // Intentar agregar un bloque usando indirección triple
    indiceBloquesDatos, err = inodo.AgregarBloqueAIndireccionTriple(archivo, sb)
    if err == nil {
        return indiceBloquesDatos, nil
    }

    // Si llegamos aquí, todos los niveles de indirección están saturados
    return -1, fmt.Errorf("no hay espacio disponible para agregar más bloques al inodo")
}

// AgregarBloqueAIndireccionSimple asigna un bloque usando indirección simple
func (inodo *INodo) AgregarBloqueAIndireccionSimple(archivo *os.File, sb *SuperBlock) (int32, error) {
    if inodo.I_block[12] == -1 {
        return -1, fmt.Errorf("no existe bloque indirecto simple")
    }

    // Cargar el bloque de apuntadores
    ba := &PointerBlock{}
    offsetBA := int64(sb.S_block_start + inodo.I_block[12]*sb.S_block_size)
    if err := ba.Decodificar(archivo, offsetBA); err != nil {
        return -1, fmt.Errorf("error al leer bloque de apuntadores: %w", err)
    }

    // Buscar un espacio libre en el bloque de apuntadores
    indiceLibre, err := ba.BuscarApuntadorLibre()
    if err != nil {
        // No hay espacio libre en este bloque de apuntadores
        return -1, fmt.Errorf("bloque indirecto simple lleno: %w", err)
    }

    // Encontrar un bloque libre en el bitmap
    nuevoIndiceBloques, err := sb.BuscarSiguienteBloqueLibre(archivo)
    if err != nil {
        return -1, fmt.Errorf("error buscando bloque libre: %w", err)
    }

    // Actualizar el bitmap para marcar este bloque como usado
    if err := sb.ActualizarBitmapBloque(archivo, nuevoIndiceBloques, true); err != nil {
        return -1, fmt.Errorf("error actualizando bitmap: %w", err)
    }

    // Inicializar el nuevo bloque con ceros
    bufferCeros := make([]byte, sb.S_block_size)
    offsetBloques := int64(sb.S_block_start + nuevoIndiceBloques*sb.S_block_size)
    if _, err := archivo.WriteAt(bufferCeros, offsetBloques); err != nil {
        return -1, fmt.Errorf("error inicializando bloque nuevo: %w", err)
    }

    // Actualizar el bloque de apuntadores
    ba.B_apuntadores[indiceLibre] = int64(nuevoIndiceBloques)

    // Escribir el bloque de apuntadores actualizado al disco
    if err := ba.Codificar(archivo, offsetBA); err != nil {
        return -1, fmt.Errorf("error escribiendo bloque de apuntadores: %w", err)
    }

    // Actualizar contadores en el superbloque
    sb.ActualizarSuperblockDespuesAsignacionBloque()

    // Información de depuración sobre el bloque asignado
    fmt.Printf("Bloque asignado mediante indirección simple: %d (posición %d del bloque de apuntadores)\n",
        nuevoIndiceBloques, indiceLibre)

    return nuevoIndiceBloques, nil
}

// AgregarBloqueAIndireccionDoble asigna un bloque usando indirección doble
func (inodo *INodo) AgregarBloqueAIndireccionDoble(archivo *os.File, sb *SuperBlock) (int32, error) {
    if inodo.I_block[13] == -1 {
        return -1, fmt.Errorf("no existe bloque indirecto doble")
    }

    // Cargar el bloque de apuntadores primario
    baPrimario := &PointerBlock{}
    offsetPrimario := int64(sb.S_block_start + inodo.I_block[13]*sb.S_block_size)
    if err := baPrimario.Decodificar(archivo, offsetPrimario); err != nil {
        return -1, fmt.Errorf("error al leer bloque de apuntadores primario: %w", err)
    }

    // Buscar un espacio libre en el bloque de apuntadores primario o usar uno existente
    for i, apuntadorPrimario := range baPrimario.B_apuntadores {
        if apuntadorPrimario == -1 {
            // Crear un nuevo bloque de apuntadores secundario
            nuevoIndiceSecundario, err := sb.BuscarSiguienteBloqueLibre(archivo)
            if err != nil {
                return -1, fmt.Errorf("error buscando bloque libre para apuntadores secundario: %w", err)
            }

            // Marcar como usado en el bitmap
            if err := sb.ActualizarBitmapBloque(archivo, nuevoIndiceSecundario, true); err != nil {
                return -1, fmt.Errorf("error actualizando bitmap: %w", err)
            }

            // Inicializar el bloque de apuntadores secundario
            baSecundario := &PointerBlock{}
            for j := range baSecundario.B_apuntadores {
                baSecundario.B_apuntadores[j] = -1
            }

            // Escribir el bloque de apuntadores secundario al disco
            offsetSecundario := int64(sb.S_block_start + nuevoIndiceSecundario*sb.S_block_size)
            if err := baSecundario.Codificar(archivo, offsetSecundario); err != nil {
                return -1, fmt.Errorf("error escribiendo bloque de apuntadores secundario: %w", err)
            }

            // Actualizar el bloque de apuntadores primario
            baPrimario.B_apuntadores[i] = int64(nuevoIndiceSecundario)
            if err := baPrimario.Codificar(archivo, offsetPrimario); err != nil {
                return -1, fmt.Errorf("error actualizando bloque de apuntadores primario: %w", err)
            }

            // Asignar un bloque de datos dentro del bloque secundario
            // Encontrar un bloque libre para datos
            nuevoIndiceDatos, err := sb.BuscarSiguienteBloqueLibre(archivo)
            if err != nil {
                return -1, fmt.Errorf("error buscando bloque libre para datos: %w", err)
            }

            // Marcar como usado en el bitmap
            if err := sb.ActualizarBitmapBloque(archivo, nuevoIndiceDatos, true); err != nil {
                return -1, fmt.Errorf("error actualizando bitmap: %w", err)
            }

            // Asignar el bloque de datos en el bloque secundario
            baSecundario.B_apuntadores[0] = int64(nuevoIndiceDatos)
            if err := baSecundario.Codificar(archivo, offsetSecundario); err != nil {
                return -1, fmt.Errorf("error actualizando bloque de apuntadores secundario: %w", err)
            }

            // Actualizar contadores en el superbloque
            sb.ActualizarSuperblockDespuesAsignacionBloque()
            sb.ActualizarSuperblockDespuesAsignacionBloque() // Una vez para cada bloque asignado

            return nuevoIndiceDatos, nil
        } else {
            // Usar un bloque secundario existente
            baSecundario := &PointerBlock{}
            offsetSecundario := int64(sb.S_block_start + int32(apuntadorPrimario)*sb.S_block_size)
            if err := baSecundario.Decodificar(archivo, offsetSecundario); err != nil {
                return -1, fmt.Errorf("error leyendo bloque de apuntadores secundario: %w", err)
            }

            // Buscar espacio en el bloque secundario
            indiceLibreSecundario, err := baSecundario.BuscarApuntadorLibre()
            if err != nil {
                // Este bloque secundario está lleno, intentar con el siguiente
                continue
            }

            // Encontrar un bloque libre para datos
            nuevoIndiceDatos, err := sb.BuscarSiguienteBloqueLibre(archivo)
            if err != nil {
                return -1, fmt.Errorf("error buscando bloque libre para datos: %w", err)
            }

            // Marcar como usado en el bitmap
            if err := sb.ActualizarBitmapBloque(archivo, nuevoIndiceDatos, true); err != nil {
                return -1, fmt.Errorf("error actualizando bitmap: %w", err)
            }

            // Asignar el bloque de datos en el bloque secundario
            baSecundario.B_apuntadores[indiceLibreSecundario] = int64(nuevoIndiceDatos)
            if err := baSecundario.Codificar(archivo, offsetSecundario); err != nil {
                return -1, fmt.Errorf("error actualizando bloque de apuntadores secundario: %w", err)
            }

            // Actualizar contadores en el superbloque
            sb.ActualizarSuperblockDespuesAsignacionBloque()

            return nuevoIndiceDatos, nil
        }
    }

    // Si llegamos aquí, el bloque de apuntadores primario está lleno
    return -1, fmt.Errorf("bloque de apuntadores primario lleno")
}

// AgregarBloqueAIndireccionTriple asigna un bloque usando indirección triple
func (inodo *INodo) AgregarBloqueAIndireccionTriple(archivo *os.File, sb *SuperBlock) (int32, error) {
    if inodo.I_block[14] == -1 {
        return -1, fmt.Errorf("no existe bloque indirecto triple")
    }

    // Cargar el bloque de apuntadores primario (nivel 1)
    baPrimario := &PointerBlock{}
    offsetPrimario := int64(sb.S_block_start + inodo.I_block[14]*sb.S_block_size)
    if err := baPrimario.Decodificar(archivo, offsetPrimario); err != nil {
        return -1, fmt.Errorf("error al leer bloque de apuntadores primario: %w", err)
    }

    // Buscar un espacio libre en el bloque de apuntadores primario o usar uno existente
    for indicePrimario, apuntadorPrimario := range baPrimario.B_apuntadores {
        if apuntadorPrimario == -1 {
            // Crear un nuevo bloque de apuntadores secundario (nivel 2)
            nuevoIndiceSecundario, err := sb.BuscarSiguienteBloqueLibre(archivo)
            if err != nil {
                return -1, fmt.Errorf("error buscando bloque libre para apuntadores secundario: %w", err)
            }

            // Marcar como usado en el bitmap
            if err := sb.ActualizarBitmapBloque(archivo, nuevoIndiceSecundario, true); err != nil {
                return -1, fmt.Errorf("error actualizando bitmap: %w", err)
            }

            // Inicializar el bloque de apuntadores secundario
            baSecundario := &PointerBlock{}
            for j := range baSecundario.B_apuntadores {
                baSecundario.B_apuntadores[j] = -1
            }

            // Crear un nuevo bloque de apuntadores terciario (nivel 3)
            nuevoIndiceTerciario, err := sb.BuscarSiguienteBloqueLibre(archivo)
            if err != nil {
                return -1, fmt.Errorf("error buscando bloque libre para apuntadores terciario: %w", err)
            }

            // Marcar como usado en el bitmap
            if err := sb.ActualizarBitmapBloque(archivo, nuevoIndiceTerciario, true); err != nil {
                return -1, fmt.Errorf("error actualizando bitmap: %w", err)
            }

            // Inicializar el bloque de apuntadores terciario
            baTerciario := &PointerBlock{}
            for j := range baTerciario.B_apuntadores {
                baTerciario.B_apuntadores[j] = -1
            }

            // Encontrar un bloque libre para datos
            nuevoIndiceDatos, err := sb.BuscarSiguienteBloqueLibre(archivo)
            if err != nil {
                return -1, fmt.Errorf("error buscando bloque libre para datos: %w", err)
            }

            // Marcar como usado en el bitmap
            if err := sb.ActualizarBitmapBloque(archivo, nuevoIndiceDatos, true); err != nil {
                return -1, fmt.Errorf("error actualizando bitmap: %w", err)
            }

            // Asignar el bloque de datos en el bloque terciario
            baTerciario.B_apuntadores[0] = int64(nuevoIndiceDatos)

            // Escribir el bloque de apuntadores terciario
            offsetTerciario := int64(sb.S_block_start + nuevoIndiceTerciario*sb.S_block_size)
            if err := baTerciario.Codificar(archivo, offsetTerciario); err != nil {
                return -1, fmt.Errorf("error escribiendo bloque de apuntadores terciario: %w", err)
            }

            // Asignar el bloque terciario en el bloque secundario
            baSecundario.B_apuntadores[0] = int64(nuevoIndiceTerciario)

            // Escribir el bloque de apuntadores secundario
            offsetSecundario := int64(sb.S_block_start + nuevoIndiceSecundario*sb.S_block_size)
            if err := baSecundario.Codificar(archivo, offsetSecundario); err != nil {
                return -1, fmt.Errorf("error escribiendo bloque de apuntadores secundario: %w", err)
            }

            // Asignar el bloque secundario en el bloque primario
            baPrimario.B_apuntadores[indicePrimario] = int64(nuevoIndiceSecundario)

            // Escribir el bloque de apuntadores primario actualizado
            if err := baPrimario.Codificar(archivo, offsetPrimario); err != nil {
                return -1, fmt.Errorf("error actualizando bloque de apuntadores primario: %w", err)
            }

            // Actualizar contadores en el superbloque
            sb.ActualizarSuperblockDespuesAsignacionBloque() // Para el bloque secundario
            sb.ActualizarSuperblockDespuesAsignacionBloque() // Para el bloque terciario
            sb.ActualizarSuperblockDespuesAsignacionBloque() // Para el bloque de datos

            return nuevoIndiceDatos, nil
        } else {
            // Bloque secundario ya existe, cargarlo
            baSecundario := &PointerBlock{}
            offsetSecundario := int64(sb.S_block_start + int32(apuntadorPrimario)*sb.S_block_size)
            if err := baSecundario.Decodificar(archivo, offsetSecundario); err != nil {
                return -1, fmt.Errorf("error leyendo bloque de apuntadores secundario: %w", err)
            }

            // Buscar espacio en el bloque secundario
            for indiceSecundario, apuntadorSecundario := range baSecundario.B_apuntadores {
                if apuntadorSecundario == -1 {
                    // Crear un nuevo bloque de apuntadores terciario
                    nuevoIndiceTerciario, err := sb.BuscarSiguienteBloqueLibre(archivo)
                    if err != nil {
                        return -1, fmt.Errorf("error buscando bloque libre para apuntadores terciario: %w", err)
                    }

                    // Marcar como usado en el bitmap
                    if err := sb.ActualizarBitmapBloque(archivo, nuevoIndiceTerciario, true); err != nil {
                        return -1, fmt.Errorf("error actualizando bitmap: %w", err)
                    }

                    // Inicializar el bloque de apuntadores terciario
                    baTerciario := &PointerBlock{}
                    for j := range baTerciario.B_apuntadores {
                        baTerciario.B_apuntadores[j] = -1
                    }

                    // Encontrar un bloque libre para datos
                    nuevoIndiceDatos, err := sb.BuscarSiguienteBloqueLibre(archivo)
                    if err != nil {
                        return -1, fmt.Errorf("error buscando bloque libre para datos: %w", err)
                    }

                    // Marcar como usado en el bitmap
                    if err := sb.ActualizarBitmapBloque(archivo, nuevoIndiceDatos, true); err != nil {
                        return -1, fmt.Errorf("error actualizando bitmap: %w", err)
                    }

                    // Asignar el bloque de datos en el bloque terciario
                    baTerciario.B_apuntadores[0] = int64(nuevoIndiceDatos)

                    // Escribir el bloque de apuntadores terciario
                    offsetTerciario := int64(sb.S_block_start + nuevoIndiceTerciario*sb.S_block_size)
                    if err := baTerciario.Codificar(archivo, offsetTerciario); err != nil {
                        return -1, fmt.Errorf("error escribiendo bloque de apuntadores terciario: %w", err)
                    }

                    // Asignar el bloque terciario en el bloque secundario
                    baSecundario.B_apuntadores[indiceSecundario] = int64(nuevoIndiceTerciario)

                    // Escribir el bloque de apuntadores secundario actualizado
                    if err := baSecundario.Codificar(archivo, offsetSecundario); err != nil {
                        return -1, fmt.Errorf("error actualizando bloque de apuntadores secundario: %w", err)
                    }

                    // Actualizar contadores en el superbloque
                    sb.ActualizarSuperblockDespuesAsignacionBloque() // Para el bloque terciario
                    sb.ActualizarSuperblockDespuesAsignacionBloque() // Para el bloque de datos

                    return nuevoIndiceDatos, nil
                } else {
                    // Bloque terciario ya existe, cargarlo
                    baTerciario := &PointerBlock{}
                    offsetTerciario := int64(sb.S_block_start + int32(apuntadorSecundario)*sb.S_block_size)
                    if err := baTerciario.Decodificar(archivo, offsetTerciario); err != nil {
                        return -1, fmt.Errorf("error leyendo bloque de apuntadores terciario: %w", err)
                    }

                    // Buscar espacio en el bloque terciario
                    indiceLibreTerciario, err := baTerciario.BuscarApuntadorLibre()
                    if err != nil {
                        // Este bloque terciario está lleno, intentar con el siguiente
                        continue
                    }

                    // Encontrar un bloque libre para datos
                    nuevoIndiceDatos, err := sb.BuscarSiguienteBloqueLibre(archivo)
                    if err != nil {
                        return -1, fmt.Errorf("error buscando bloque libre para datos: %w", err)
                    }

                    // Marcar como usado en el bitmap
                    if err := sb.ActualizarBitmapBloque(archivo, nuevoIndiceDatos, true); err != nil {
                        return -1, fmt.Errorf("error actualizando bitmap: %w", err)
                    }

                    // Asignar el bloque de datos en el bloque terciario
                    baTerciario.B_apuntadores[indiceLibreTerciario] = int64(nuevoIndiceDatos)

                    // Escribir el bloque de apuntadores terciario actualizado
                    if err := baTerciario.Codificar(archivo, offsetTerciario); err != nil {
                        return -1, fmt.Errorf("error actualizando bloque de apuntadores terciario: %w", err)
                    }

                    // Actualizar contadores en el superbloque
                    sb.ActualizarSuperblockDespuesAsignacionBloque()

                    return nuevoIndiceDatos, nil
                }
            }
            // Si todos los apuntadores secundarios están llenos, continuar con el siguiente primario
        }
    }

    // Si llegamos aquí, todos los niveles de indirección están llenos
    return -1, fmt.Errorf("todos los bloques de apuntadores de indirección triple están llenos")
}

// LiberarBloque libera un bloque específico y actualiza el bitmap
func (inodo *INodo) LiberarBloque(archivo *os.File, sb *SuperBlock, indiceBloque int32) error {
    // Marcar el bloque como libre en el bitmap de bloques
    if err := sb.ActualizarBitmapBloque(archivo, indiceBloque, false); err != nil {
        return fmt.Errorf("error liberando bloque %d: %w", indiceBloque, err)
    }
    
    // Actualizar los contadores del superbloque después de la desasignación
    sb.ActualizarSuperblockDespuesDesasignacionBloque()
    
    return nil
}

// LiberarTodosLosBloques libera todos los bloques asociados a un inodo
func (inodo *INodo) LiberarTodosLosBloques(archivo *os.File, sb *SuperBlock) error {
    // Obtener todos los bloques utilizados por este inodo
    bloques, err := inodo.ObtenerTodosLosIndicesDeBloques(archivo, sb)
    if err != nil {
        return fmt.Errorf("error obteniendo índices de bloques: %w", err)
    }

    // Liberar cada bloque individualmente
    for _, indiceBloque := range bloques {
        if err := inodo.LiberarBloque(archivo, sb, indiceBloque); err != nil {
            return fmt.Errorf("error liberando bloque %d: %w", indiceBloque, err)
        }
    }

    // Reiniciar todos los apuntadores del inodo a valores por defecto
    for i := range inodo.I_block {
        inodo.I_block[i] = -1
    }

    // Resetear el tamaño del archivo y actualizar tiempo de modificación
    inodo.I_size = 0
    inodo.ActualizarTiempoModificacion()

    fmt.Printf("Liberados %d bloques del inodo exitosamente\n", len(bloques))
    return nil
}

// VerificarYLiberarBloquesIndirectosVacios verifica y libera bloques de apuntadores que quedaron vacíos
func (inodo *INodo) VerificarYLiberarBloquesIndirectosVacios(archivo *os.File, sb *SuperBlock) error {
    // 1. Verificar bloque indirecto simple (posición 12)
    if inodo.I_block[12] != -1 {
        ba := &PointerBlock{}
        offsetBA := int64(sb.S_block_start + inodo.I_block[12]*sb.S_block_size)
        if err := ba.Decodificar(archivo, offsetBA); err != nil {
            return fmt.Errorf("error leyendo bloque indirecto simple: %w", err)
        }

        // Verificar si está completamente vacío (todos los apuntadores son -1)
        estaVacio := true
        for _, apuntador := range ba.B_apuntadores {
            if apuntador != -1 {
                estaVacio = false
                break
            }
        }

        if estaVacio {
            // Liberar el bloque de apuntadores simple
            fmt.Printf("Liberando bloque de apuntadores simple %d (sin referencias)\n", inodo.I_block[12])
            if err := inodo.LiberarBloque(archivo, sb, inodo.I_block[12]); err != nil {
                return fmt.Errorf("error liberando bloque indirecto simple: %w", err)
            }
            // Actualizar la referencia en el inodo
            inodo.I_block[12] = -1
        }
    }

    // 2. Verificar bloque indirecto doble (posición 13)
    if inodo.I_block[13] != -1 {
        baPrimario := &PointerBlock{}
        offsetPrimario := int64(sb.S_block_start + inodo.I_block[13]*sb.S_block_size)
        if err := baPrimario.Decodificar(archivo, offsetPrimario); err != nil {
            return fmt.Errorf("error leyendo bloque indirecto doble: %w", err)
        }

        // Verificar bloques secundarios y marcar los que están vacíos
        bloquesSecundariosVacios := make([]int, 0)
        todosPrimariosVacios := true

        for i, apuntadorPrimario := range baPrimario.B_apuntadores {
            if apuntadorPrimario != -1 {
                // Cargar el bloque secundario correspondiente
                baSecundario := &PointerBlock{}
                offsetSecundario := int64(sb.S_block_start + int32(apuntadorPrimario)*sb.S_block_size)
                if err := baSecundario.Decodificar(archivo, offsetSecundario); err != nil {
                    return fmt.Errorf("error leyendo bloque secundario: %w", err)
                }

                // Verificar si el bloque secundario está vacío
                estaVacio := true
                for _, apuntadorSecundario := range baSecundario.B_apuntadores {
                    if apuntadorSecundario != -1 {
                        estaVacio = false
                        break
                    }
                }

                if estaVacio {
                    // Marcar para liberación posterior
                    bloquesSecundariosVacios = append(bloquesSecundariosVacios, i)
                    fmt.Printf("Marcando bloque secundario %d para liberación (sin referencias)\n", apuntadorPrimario)
                } else {
                    todosPrimariosVacios = false
                }
            }
        }

        // Liberar los bloques secundarios vacíos y actualizar referencias
        for _, indice := range bloquesSecundariosVacios {
            indiceBloqueSecundario := baPrimario.B_apuntadores[indice]
            // Liberar el bloque secundario
            if err := inodo.LiberarBloque(archivo, sb, int32(indiceBloqueSecundario)); err != nil {
                return fmt.Errorf("error liberando bloque secundario: %w", err)
            }
            // Actualizar referencia en el bloque primario
            baPrimario.B_apuntadores[indice] = -1
        }

        // Si hay bloques secundarios liberados, guardar el bloque primario actualizado
        if len(bloquesSecundariosVacios) > 0 {
            if err := baPrimario.Codificar(archivo, offsetPrimario); err != nil {
                return fmt.Errorf("error actualizando bloque primario: %w", err)
            }
        }

        // Si todos los bloques secundarios están vacíos/liberados, liberar el bloque primario
        if todosPrimariosVacios {
            fmt.Printf("Liberando bloque de apuntadores doble %d (sin referencias)\n", inodo.I_block[13])
            if err := inodo.LiberarBloque(archivo, sb, inodo.I_block[13]); err != nil {
                return fmt.Errorf("error liberando bloque indirecto doble: %w", err)
            }
            inodo.I_block[13] = -1
        }
    }

    // 3. Verificar bloque indirecto triple (posición 14)
    if inodo.I_block[14] != -1 {
        baPrimario := &PointerBlock{}
        offsetPrimario := int64(sb.S_block_start + inodo.I_block[14]*sb.S_block_size)
        if err := baPrimario.Decodificar(archivo, offsetPrimario); err != nil {
            return fmt.Errorf("error leyendo bloque indirecto triple: %w", err)
        }

        contadorPrimariosVacios := 0
        todosPrimariosVacios := true

        // Procesar cada bloque secundario en el nivel primario
        for indicePrimario, apuntadorPrimario := range baPrimario.B_apuntadores {
            if apuntadorPrimario == -1 {
                contadorPrimariosVacios++
                continue
            }

            baSecundario := &PointerBlock{}
            offsetSecundario := int64(sb.S_block_start + int32(apuntadorPrimario)*sb.S_block_size)
            if err := baSecundario.Decodificar(archivo, offsetSecundario); err != nil {
                return fmt.Errorf("error leyendo bloque secundario en triple: %w", err)
            }

            contadorSecundariosVacios := 0
            todosSecundariosVacios := true

            // Procesar cada bloque terciario en el nivel secundario
            for indiceSecundario, apuntadorSecundario := range baSecundario.B_apuntadores {
                if apuntadorSecundario == -1 {
                    contadorSecundariosVacios++
                    continue
                }

                baTerciario := &PointerBlock{}
                offsetTerciario := int64(sb.S_block_start + int32(apuntadorSecundario)*sb.S_block_size)
                if err := baTerciario.Decodificar(archivo, offsetTerciario); err != nil {
                    return fmt.Errorf("error leyendo bloque terciario: %w", err)
                }

                // Verificar si el bloque terciario está completamente vacío
                estaVacio := true
                for _, apuntadorTerciario := range baTerciario.B_apuntadores {
                    if apuntadorTerciario != -1 {
                        estaVacio = false
                        break
                    }
                }

                if estaVacio {
                    // Liberar bloque terciario vacío
                    fmt.Printf("Liberando bloque terciario %d (sin referencias)\n", apuntadorSecundario)
                    if err := inodo.LiberarBloque(archivo, sb, int32(apuntadorSecundario)); err != nil {
                        return fmt.Errorf("error liberando bloque terciario: %w", err)
                    }
                    baSecundario.B_apuntadores[indiceSecundario] = -1
                } else {
                    todosSecundariosVacios = false
                }
            }

            // Si el bloque secundario quedó completamente vacío, liberarlo
            if todosSecundariosVacios {
                fmt.Printf("Liberando bloque secundario %d en triple (sin referencias)\n", apuntadorPrimario)
                if err := inodo.LiberarBloque(archivo, sb, int32(apuntadorPrimario)); err != nil {
                    return fmt.Errorf("error liberando bloque secundario en triple: %w", err)
                }
                baPrimario.B_apuntadores[indicePrimario] = -1
            } else {
                // Si hubo cambios en el secundario, guardarlo
                if contadorSecundariosVacios > 0 && contadorSecundariosVacios < len(baSecundario.B_apuntadores) {
                    if err := baSecundario.Codificar(archivo, offsetSecundario); err != nil {
                        return fmt.Errorf("error actualizando bloque secundario: %w", err)
                    }
                }
                todosPrimariosVacios = false
            }
        }

        // Si hubo cambios en el primario pero no está completamente vacío, guardarlo
        if !todosPrimariosVacios && contadorPrimariosVacios > 0 {
            if err := baPrimario.Codificar(archivo, offsetPrimario); err != nil {
                return fmt.Errorf("error actualizando bloque primario triple: %w", err)
            }
        }

        // Si el bloque primario quedó completamente vacío, liberarlo
        if todosPrimariosVacios {
            fmt.Printf("Liberando bloque de apuntadores triple %d (sin referencias)\n", inodo.I_block[14])
            if err := inodo.LiberarBloque(archivo, sb, inodo.I_block[14]); err != nil {
                return fmt.Errorf("error liberando bloque indirecto triple: %w", err)
            }
            inodo.I_block[14] = -1
        }
    }

    return nil
}

// LeerDatos lee el contenido completo del archivo desde los bloques del inodo
func (inodo *INodo) LeerDatos(archivo *os.File, sb *SuperBlock) ([]byte, error) {
    // Si el archivo está vacío, retornar array vacío
    if inodo.I_size == 0 {
        return []byte{}, nil
    }

    // Obtener todos los bloques de datos (sin incluir bloques de apuntadores)
    indicesBloques, err := inodo.ObtenerIndicesBloquesDatos(archivo, sb)
    if err != nil {
        return nil, fmt.Errorf("error obteniendo bloques de datos: %w", err)
    }

    // Calcular cuántos bytes debemos leer en total (respetando I_size)
    bytesALeer := int(inodo.I_size)
    resultado := make([]byte, 0, bytesALeer)

    // Leer cada bloque secuencialmente
    for _, indiceBloque := range indicesBloques {
        if bytesALeer <= 0 {
            break // Ya leímos todo lo que necesitábamos según I_size
        }

        // Leer el bloque como FileBlock
        bloqueArchivo := &FileBlock{}
        offsetBloque := int64(sb.S_block_start + indiceBloque*sb.S_block_size)
        if err := bloqueArchivo.Decodificar(archivo, offsetBloque); err != nil {
            return nil, fmt.Errorf("error leyendo bloque %d: %w", indiceBloque, err)
        }

        // Determinar cuántos bytes leer de este bloque
        bytesDeEsteBloque := DimensionBloque
        if bytesDeEsteBloque > bytesALeer {
            bytesDeEsteBloque = bytesALeer
        }

        // Añadir los bytes al resultado
        resultado = append(resultado, bloqueArchivo.B_contenido[:bytesDeEsteBloque]...)
        bytesALeer -= bytesDeEsteBloque
    }

    fmt.Printf("Datos leídos exitosamente: %d bytes desde %d bloques\n", 
        len(resultado), len(indicesBloques))

    return resultado, nil
}

// EscribirDatos escribe datos en los bloques del inodo
func (inodo *INodo) EscribirDatos(archivo *os.File, sb *SuperBlock, datos []byte) error {
    // Obtener el tamaño actual y el nuevo tamaño
    tamañoAnterior := inodo.I_size
    nuevoTamaño := int32(len(datos))

    fmt.Printf("Escribiendo datos: tamaño anterior=%d, nuevo tamaño=%d\n", tamañoAnterior, nuevoTamaño)

    // Si el tamaño cambió, necesitamos redimensionar los bloques
    if tamañoAnterior != nuevoTamaño {
        // Si había bloques anteriores, liberarlos todos
        if tamañoAnterior > 0 {
            fmt.Printf("Liberando bloques existentes antes de redimensionar\n")
            if err := inodo.LiberarTodosLosBloques(archivo, sb); err != nil {
                return fmt.Errorf("error liberando bloques existentes: %w", err)
            }
        }

        // Si el nuevo tamaño es 0, solo limpiar y salir
        if nuevoTamaño == 0 {
            inodo.I_size = 0
            inodo.ActualizarTiempoModificacion()
            fmt.Printf("Archivo vaciado correctamente\n")
            return nil
        }

        // Calcular cuántos bloques necesitamos para el nuevo tamaño
        bloquesNecesarios := (nuevoTamaño + sb.S_block_size - 1) / sb.S_block_size
        fmt.Printf("Asignando %d bloques para almacenar %d bytes\n", bloquesNecesarios, nuevoTamaño)

        // Asignar los bloques necesarios uno por uno
        for i := int32(0); i < bloquesNecesarios; i++ {
            _, err := inodo.AgregarBloque(archivo, sb)
            if err != nil {
                return fmt.Errorf("error asignando bloque %d durante redimensionado: %w", i, err)
            }
        }
    }

    // Si no hay datos que escribir, salir
    if len(datos) == 0 {
        return nil
    }

    // Obtener todos los bloques de datos disponibles
    indicesBloques, err := inodo.ObtenerIndicesBloquesDatos(archivo, sb)
    if err != nil {
        return fmt.Errorf("error obteniendo bloques de datos para escritura: %w", err)
    }

    // Verificar que tengamos suficientes bloques para los datos
    bloquesEsperados := (nuevoTamaño + sb.S_block_size - 1) / sb.S_block_size
    if int32(len(indicesBloques)) < bloquesEsperados {
        return fmt.Errorf("insuficientes bloques asignados: disponibles=%d, necesarios=%d",
            len(indicesBloques), bloquesEsperados)
    }

    // Escribir los datos en los bloques secuencialmente
    offsetDatos := 0
    for i, indiceBloque := range indicesBloques {
        // Calcular la posición del bloque en el disco
        offsetBloque := int64(sb.S_block_start + indiceBloque*sb.S_block_size)

        // Determinar cuántos bytes escribir en este bloque
        bytesAEscribir := int(sb.S_block_size)
        if offsetDatos+bytesAEscribir > len(datos) {
            bytesAEscribir = len(datos) - offsetDatos
        }

        // Si no hay más datos para escribir, salir del bucle
        if bytesAEscribir <= 0 {
            break
        }

        // Crear un buffer del tamaño del bloque, inicializado con ceros
        bufferBloque := make([]byte, sb.S_block_size)
        
        // Copiar los datos del usuario al buffer
        copy(bufferBloque, datos[offsetDatos:offsetDatos+bytesAEscribir])

        // Escribir el bloque completo al disco
        if _, err := archivo.WriteAt(bufferBloque, offsetBloque); err != nil {
            return fmt.Errorf("error escribiendo datos al bloque %d (índice %d): %w", 
                i, indiceBloque, err)
        }

        offsetDatos += bytesAEscribir
        
        // Si ya escribimos todos los datos, salir
        if offsetDatos >= len(datos) {
            break
        }

        fmt.Printf("Bloque %d escrito: %d bytes (offset global: %d)\n", 
            indiceBloque, bytesAEscribir, offsetDatos)
    }

    // Actualizar metadatos del inodo
    inodo.I_size = nuevoTamaño
    inodo.ActualizarTiempoModificacion()

    // Informar sobre la operación completada
    fmt.Printf("Escritura completada exitosamente: %d bytes en %d bloques\n",
        nuevoTamaño, len(indicesBloques))

    return nil
}

// ObtenerIndicesBloquesDatos devuelve solo los índices de bloques que contienen datos (no apuntadores)
func (inodo *INodo) ObtenerIndicesBloquesDatos(archivo *os.File, sb *SuperBlock) ([]int32, error) {
    bloquesDatos := []int32{}

    // 1. Bloques directos (0-11) - siempre son bloques de datos
    for i := 0; i < 12; i++ {
        if inodo.I_block[i] != -1 {
            bloquesDatos = append(bloquesDatos, inodo.I_block[i])
        }
    }

    // 2. Bloques en indirección simple (solo los apuntados, no el bloque 12)
    if inodo.I_block[12] != -1 {
        ba := &PointerBlock{}
        offsetBA := int64(sb.S_block_start + inodo.I_block[12]*sb.S_block_size)
        if err := ba.Decodificar(archivo, offsetBA); err != nil {
            return nil, fmt.Errorf("error leyendo bloque indirecto simple: %w", err)
        }

        for _, apuntador := range ba.B_apuntadores {
            if apuntador != -1 {
                bloquesDatos = append(bloquesDatos, int32(apuntador))
            }
        }
    }

    // 3. Bloques en indirección doble (solo los bloques finales, no los apuntadores)
    if inodo.I_block[13] != -1 {
        baPrimario := &PointerBlock{}
        offsetPrimario := int64(sb.S_block_start + inodo.I_block[13]*sb.S_block_size)
        if err := baPrimario.Decodificar(archivo, offsetPrimario); err != nil {
            return nil, fmt.Errorf("error leyendo bloque indirecto doble: %w", err)
        }

        for _, apuntadorPrimario := range baPrimario.B_apuntadores {
            if apuntadorPrimario != -1 {
                baSecundario := &PointerBlock{}
                offsetSecundario := int64(sb.S_block_start + int32(apuntadorPrimario)*sb.S_block_size)
                if err := baSecundario.Decodificar(archivo, offsetSecundario); err != nil {
                    return nil, fmt.Errorf("error leyendo bloque secundario: %w", err)
                }

                for _, apuntadorSecundario := range baSecundario.B_apuntadores {
                    if apuntadorSecundario != -1 {
                        bloquesDatos = append(bloquesDatos, int32(apuntadorSecundario))
                    }
                }
            }
        }
    }

    // 4. Bloques en indirección triple (solo los bloques finales, no los apuntadores)
    if inodo.I_block[14] != -1 {
        baPrimario := &PointerBlock{}
        offsetPrimario := int64(sb.S_block_start + inodo.I_block[14]*sb.S_block_size)
        if err := baPrimario.Decodificar(archivo, offsetPrimario); err != nil {
            return nil, fmt.Errorf("error leyendo bloque indirecto triple: %w", err)
        }

        for _, apuntadorPrimario := range baPrimario.B_apuntadores {
            if apuntadorPrimario != -1 {
                baSecundario := &PointerBlock{}
                offsetSecundario := int64(sb.S_block_start + int32(apuntadorPrimario)*sb.S_block_size)
                if err := baSecundario.Decodificar(archivo, offsetSecundario); err != nil {
                    return nil, fmt.Errorf("error leyendo bloque secundario en indirección triple: %w", err)
                }

                for _, apuntadorSecundario := range baSecundario.B_apuntadores {
                    if apuntadorSecundario != -1 {
                        baTerciario := &PointerBlock{}
                        offsetTerciario := int64(sb.S_block_start + int32(apuntadorSecundario)*sb.S_block_size)
                        if err := baTerciario.Decodificar(archivo, offsetTerciario); err != nil {
                            return nil, fmt.Errorf("error leyendo bloque terciario: %w", err)
                        }

                        for _, apuntadorTerciario := range baTerciario.B_apuntadores {
                            if apuntadorTerciario != -1 {
                                bloquesDatos = append(bloquesDatos, int32(apuntadorTerciario))
                            }
                        }
                    }
                }
            }
        }
    }

    return bloquesDatos, nil
}

// NuevoInodoVacio crea un inodo vacío con valores predeterminados
func NuevoInodoVacio() *INodo {
    inodo := &INodo{}

    // Establecer usuario y grupo root por defecto
    inodo.I_uid = 1
    inodo.I_gid = 1
    inodo.I_size = 0

    // Inicializar todos los tiempos al momento actual
    tiempoActual := float32(time.Now().Unix())
    inodo.I_atime = tiempoActual
    inodo.I_ctime = tiempoActual
    inodo.I_mtime = tiempoActual

    // Tipo por defecto (se asignará después según sea archivo o carpeta)
    inodo.I_type[0] = '0' // Por defecto carpeta, se puede cambiar a '1' para archivo

    // Permisos por defecto
    inodo.I_perm = [3]byte{'0', '0', '0'}

    // Inicializar todos los bloques como no asignados
    for i := range inodo.I_block {
        inodo.I_block[i] = -1
    }

    return inodo
}