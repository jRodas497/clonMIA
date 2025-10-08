package Estructuras

import (
	"fmt"
	"os"
	"strings"
	"time"

	Utils "backend/Utils"
)

// Acá | diferente a createFolderInInode pero se suponen hacen lo mismo
// crearCarpetaEnInodo crea una carpeta en un inodo específico utilizando las funciones avanzadas
func (sb *SuperBlock) crearCarpetaEnInodo(archivo *os.File, indiceInodo int32, directoriosPadre []string, directorioDestino string, registrarJournal bool) error {
    // Crear un nuevo inodo
    inodo := &INodo{}
    fmt.Printf("Deserializando inodo %d\n", indiceInodo) // Depuración

    // Deserializar el inodo
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al deserializar inodo %d: %v", indiceInodo, err)
    }
    fmt.Printf("Inodo %d deserializado. Tipo: %c\n", indiceInodo, inodo.I_type[0]) // Depuración

    // Verificar si el inodo es de tipo carpeta
    if inodo.I_type[0] != '0' {
        fmt.Printf("Inodo %d no es una carpeta, es de tipo: %c\n", indiceInodo, inodo.I_type[0]) // Depuración
        return nil
    }

    // Iterar sobre cada bloque del inodo (apuntadores)
    indicesBloques, err := inodo.ObtenerIndicesBloquesDatos(archivo, sb)
    if err != nil {
        return fmt.Errorf("error obteniendo bloques de datos: %v", err)
    }

    // Si no hay bloques, verificar si podemos agregar uno
    if len(indicesBloques) == 0 {
        fmt.Printf("El inodo %d no tiene bloques asignados, añadiendo uno nuevo\n", indiceInodo)
        nuevoIndiceBloque, err := inodo.AgregarBloque(archivo, sb)
        if err != nil {
            return fmt.Errorf("error añadiendo nuevo bloque al inodo %d: %v", indiceInodo, err)
        }
        indicesBloques = []int32{nuevoIndiceBloque}

        // Inicializar el nuevo bloque de carpeta con entradas . y ..
        nuevoBloque := &FolderBlock{
            B_cont: [4]FolderContent{
                {B_name: [12]byte{'.'}, B_inodo: indiceInodo},
                {B_name: [12]byte{'.', '.'}, B_inodo: indiceInodo}, // Por defecto apunta a sí mismo, se actualizará si es necesario
                {B_name: [12]byte{'-'}, B_inodo: -1},
                {B_name: [12]byte{'-'}, B_inodo: -1},
            },
        }
        offsetBloque := int64(sb.S_block_start + nuevoIndiceBloque*sb.S_block_size)
        if err := nuevoBloque.Codificar(archivo, offsetBloque); err != nil {
            return fmt.Errorf("error inicializando nuevo bloque de carpeta: %v", err)
        }
    }

    // Iterar sobre los bloques existentes
    for _, indiceBloque := range indicesBloques {
        fmt.Printf("Procesando bloque %d del inodo %d\n", indiceBloque, indiceInodo) // Depuración
        bloque := &FolderBlock{}

        // Deserializar el bloque
        offsetBloque := int64(sb.S_block_start + (indiceBloque * sb.S_block_size))
        err := bloque.Decodificar(archivo, offsetBloque)
        if err != nil {
            return fmt.Errorf("error al deserializar bloque %d: %v", indiceBloque, err)
        }

        // Iterar sobre cada contenido del bloque, desde el índice 2 (evitamos . y ..)
        for indiceContenido := 2; indiceContenido < len(bloque.B_cont); indiceContenido++ {
            contenido := bloque.B_cont[indiceContenido]
            fmt.Printf("Verificando contenido en índice %d del bloque %d\n", indiceContenido, indiceBloque)

            // Si hay más carpetas padres en la ruta, buscar la siguiente carpeta
            if len(directoriosPadre) != 0 {
                // Si el contenido está vacío o no hay más entradas, salir
                if contenido.B_inodo == -1 {
                    fmt.Printf("No se encontró carpeta padre en inodo %d en la posición %d\n", indiceInodo, indiceContenido)
                    break
                }

                directorioPadre, err := Utils.PrimeroEnLista(directoriosPadre)
                if err != nil {
                    return err
                }

                nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")
                nombreDirectorioPadre := strings.Trim(directorioPadre, "\x00 ")
                fmt.Printf("Comparando '%s' con el nombre de la carpeta padre '%s'\n", nombreContenido, nombreDirectorioPadre)

                // Si encontramos la carpeta padre, recursión
                if strings.EqualFold(nombreContenido, nombreDirectorioPadre) {
                    fmt.Printf("Carpeta padre '%s' encontrada en inodo %d\n", nombreDirectorioPadre, contenido.B_inodo)
                    return sb.crearCarpetaEnInodo(archivo, contenido.B_inodo, Utils.EliminarElemento(directoriosPadre, 0), directorioDestino, registrarJournal)
                }
            } else {
                // Estamos en el directorio destino, crear la carpeta

                // Si esta posición ya está ocupada, probar la siguiente
                if contenido.B_inodo != -1 {
                    fmt.Printf("Posición %d ya ocupada, probando siguiente\n", indiceContenido)
                    continue
                }

                fmt.Printf("Creando directorio '%s' en bloque %d posición %d\n", directorioDestino, indiceBloque, indiceContenido)

                // 1. Crear un nuevo inodo para la carpeta
                inodoCarpeta := &INodo{}

                // Inicializar el inodo con valores predeterminados
                inodoCarpeta.I_uid = 1
                inodoCarpeta.I_gid = 1
                inodoCarpeta.I_size = 0
                inodoCarpeta.I_atime = float32(time.Now().Unix())
                inodoCarpeta.I_ctime = float32(time.Now().Unix())
                inodoCarpeta.I_mtime = float32(time.Now().Unix())
                inodoCarpeta.I_type = [1]byte{'0'} // Tipo carpeta
                inodoCarpeta.I_perm = [3]byte{'6', '6', '4'}

                // Inicializar todos los bloques a -1
                for i := range inodoCarpeta.I_block {
                    inodoCarpeta.I_block[i] = -1
                }

                // 2. Asignar y marcar un nuevo inodo en el bitmap
                nuevoIndiceInodo, err := sb.BuscarSiguienteInodoLibre(archivo)
                if err != nil {
                    return fmt.Errorf("error encontrando inodo libre: %v", err)
                }

                if err := sb.ActualizarBitmapInodo(archivo, nuevoIndiceInodo, true); err != nil {
                    return fmt.Errorf("error marcando inodo como usado: %v", err)
                }

                // 3. Asignar un bloque para la carpeta utilizando AgregarBloque
                nuevoIndiceBloque, err := inodoCarpeta.AgregarBloque(archivo, sb)
                if err != nil {
                    // Rollback: liberar el inodo
                    sb.ActualizarBitmapInodo(archivo, nuevoIndiceInodo, false)
                    return fmt.Errorf("error asignando bloque para la carpeta: %v", err)
                }

                // 4. Inicializar el contenido del nuevo bloque de carpeta
                bloqueCarpeta := &FolderBlock{
                    B_cont: [4]FolderContent{
                        {B_name: [12]byte{'.'}, B_inodo: nuevoIndiceInodo},
                        {B_name: [12]byte{'.', '.'}, B_inodo: indiceInodo}, // Apunta al directorio padre
                        {B_name: [12]byte{'-'}, B_inodo: -1},
                        {B_name: [12]byte{'-'}, B_inodo: -1},
                    },
                }

                // 5. Escribir el bloque al disco
                nuevoOffsetBloque := int64(sb.S_block_start + (nuevoIndiceBloque * sb.S_block_size))
                if err := bloqueCarpeta.Codificar(archivo, nuevoOffsetBloque); err != nil {
                    // Rollback: liberar bloque e inodo
                    inodoCarpeta.LiberarBloque(archivo, sb, nuevoIndiceBloque)
                    sb.ActualizarBitmapInodo(archivo, nuevoIndiceInodo, false)
                    return fmt.Errorf("error escribiendo bloque de carpeta: %v", err)
                }

                // 6. Escribir el inodo al disco
                offsetInodo := int64(sb.S_inode_start + (nuevoIndiceInodo * sb.S_inode_size))
                if err := inodoCarpeta.Codificar(archivo, offsetInodo); err != nil {
                    // Rollback: liberar recursos
                    inodoCarpeta.LiberarBloque(archivo, sb, nuevoIndiceBloque)
                    sb.ActualizarBitmapInodo(archivo, nuevoIndiceInodo, false)
                    return fmt.Errorf("error escribiendo inodo de carpeta: %v", err)
                }

                // 7. Actualizar la entrada en el directorio padre
                bytesNombre := [12]byte{}
                copy(bytesNombre[:], directorioDestino)
                bloque.B_cont[indiceContenido] = FolderContent{
                    B_name:  bytesNombre,
                    B_inodo: nuevoIndiceInodo,
                }

                // 8. Guardar el bloque del directorio padre con la nueva entrada
                if err := bloque.Codificar(archivo, offsetBloque); err != nil {
                    // Rollback en caso de error (aunque es poco probable aquí)
                    return fmt.Errorf("error actualizando bloque del directorio padre: %v", err)
                }

                // 9. Journaling si es necesario
                if registrarJournal && sb.S_filesystem_type == 3 {
                    inicioJournaling := int64(sb.InicioJournal())

                    // Construir la ruta completa para el journal de forma más robusta
                    var rutaCompleta string
                    if len(directoriosPadre) > 0 {
                        // Si hay directorios padres, incluirlos en la ruta
                        rutaCompleta = "/" + strings.Join(directoriosPadre, "/")
                        if len(directorioDestino) > 0 {
                            rutaCompleta += "/" + directorioDestino
                        }
                    } else {
                        // Si estamos en la raíz
                        rutaCompleta = "/" + directorioDestino
                    }

                    // Usar AgregarEntradaJournal que maneja automáticamente índices y serialización
                    if err := AgregarEntradaJournal(
                        archivo,
                        inicioJournaling,
                        ENTRADAS_JOURNAL,
                        "mkdir",
                        rutaCompleta,
                        "",
                        sb,
                    ); err != nil {
                        // Solo mostrar advertencia pero continuar con la operación
                        fmt.Printf("Advertencia: error registrando operación en journal: %v\n", err)
                    } else {
                        fmt.Printf("Operación 'mkdir %s' registrada en journal exitosamente\n", rutaCompleta)
                    }
                }

                fmt.Printf("Directorio '%s' creado exitosamente en inodo %d\n", directorioDestino, nuevoIndiceInodo)
                sb.ActualizarSuperblockDespuesAsignacionInodo() // Actualizar contadores en el superbloque

                return nil
            }
        }
    }

    // Si llegamos aquí, no se encontró espacio en los bloques existentes
    // Intentar agregar un nuevo bloque al directorio actual
    if len(directoriosPadre) == 0 { // Solo si estamos buscando crear la carpeta en este nivel
        nuevoIndiceBloque, err := inodo.AgregarBloque(archivo, sb)
        if err != nil {
            return fmt.Errorf("error añadiendo bloque adicional al directorio: %v", err)
        }

        // Inicializar el nuevo bloque
        nuevoBloque := &FolderBlock{
            B_cont: [4]FolderContent{
                {B_name: [12]byte{'-'}, B_inodo: -1},
                {B_name: [12]byte{'-'}, B_inodo: -1},
                {B_name: [12]byte{'-'}, B_inodo: -1},
                {B_name: [12]byte{'-'}, B_inodo: -1},
            },
        }

        offsetBloque := int64(sb.S_block_start + (nuevoIndiceBloque * sb.S_block_size))
        if err := nuevoBloque.Codificar(archivo, offsetBloque); err != nil {
            return fmt.Errorf("error escribiendo nuevo bloque en directorio: %v", err)
        }

        // Actualizar el inodo del directorio con este nuevo bloque
        if err := inodo.Codificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size))); err != nil {
            return fmt.Errorf("error actualizando inodo %d: %v", indiceInodo, err)
        }

        // Recursión para volver a intentar la creación con el nuevo bloque disponible
        fmt.Printf("Añadido nuevo bloque %d al directorio, reintentando creación\n", nuevoIndiceBloque)
        return sb.crearCarpetaEnInodo(archivo, indiceInodo, directoriosPadre, directorioDestino, registrarJournal)
    }

    return fmt.Errorf("no se encontró carpeta para '%s' y no se pudo crear", directorioDestino)
}

// CrearCarpeta genera una carpeta en el sistema de archivos
func (sb *SuperBlock) CrearCarpeta(archivo *os.File, directoriosPadre []string, directorioDestino string, log bool) error {
	// Si directoriosPadre esta vacio trabajar solo con inodo raiz
	if len(directoriosPadre) == 0 {
		return sb.crearCarpetaEnInodo(archivo, 0, directoriosPadre, directorioDestino, log)
	}

	// Recorrer inodo para el inodo padre
	for i := int32(0); i < sb.S_inodes_count; i++ {
		// Desde inodo 0
		err := sb.crearCarpetaEnInodo(archivo, i, directoriosPadre, directorioDestino, log)
		if err != nil {
			return err
		}
	}

	return nil
}

// CrearCarpetaRecursivamente crea carpetas recursivamente asegurando que cada directorio intermedio existe
func (sb *SuperBlock) CrearCarpetaRecursivamente(archivo *os.File, ruta string, registrarJournal bool) error {
    // Fragmentar la ruta en directorios individuales
    directorios := strings.Split(strings.Trim(ruta, "/"), "/")

    if len(directorios) == 0 {
        return fmt.Errorf("ruta no válida: %s", ruta)
    }

    // Invocar la función recursiva iniciando desde el inodo raíz
    return sb.crearCarpetaRecursivamenteEnInodo(archivo, 0, directorios, registrarJournal)
}

// crearCarpetaRecursivamenteEnInodo garantiza que cada carpeta en la lista exista o sea creada
func (sb *SuperBlock) crearCarpetaRecursivamenteEnInodo(archivo *os.File, indiceInodo int32, directorios []string, registrarJournal bool) error {
    if len(directorios) == 0 {
        return nil // No hay más directorios que procesar
    }

    directorioActual := directorios[0]
    directoriosRestantes := directorios[1:]

    // Utilizar la función `crearCarpetaEnInodo` para localizar o crear el directorio actual
    err := sb.crearCarpetaEnInodo(archivo, indiceInodo, nil, directorioActual, registrarJournal)
    if err != nil {
        return fmt.Errorf("error procesando directorio '%s': %v", directorioActual, err)
    }

    // Después de procesar el directorio actual, avanzar al siguiente nivel recursivamente
    return sb.crearCarpetaRecursivamenteEnInodo(archivo, sb.S_inodes_count-1, directoriosRestantes, registrarJournal)
}

// eliminarCarpetaEnInodo elimina recursivamente el contenido de una carpeta en un inodo específico
func (sb *SuperBlock) eliminarCarpetaEnInodo(archivo *os.File, indiceInodo int32, rutaCarpeta ...string) error {
    // 1. Deserializar el inodo del directorio objetivo
    inodoDirectorio := &INodo{}
    err := inodoDirectorio.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al deserializar inodo %d: %w", indiceInodo, err)
    }

    // 2. Validar que el inodo corresponde a un directorio
    if inodoDirectorio.I_type[0] != '0' {
        return fmt.Errorf("el inodo %d no corresponde a una carpeta", indiceInodo)
    }

    // 3. Obtener la ruta completa del directorio para el registro de journal
    rutaCompleta := "/"
    if len(rutaCarpeta) > 0 && rutaCarpeta[0] != "" {
        rutaCompleta = rutaCarpeta[0]
    }

    // Calcular la posición de inicio del journal (disponible para toda la función)
    inicioJournaling := int64(sb.InicioJournal())

    // 4. Registrar en journal si el sistema de archivos es tipo EXT3
    if sb.S_filesystem_type == 3 {
        // Usar el método AgregarEntradaJournal para registrar la operación
        if err := AgregarEntradaJournal(
            archivo,
            inicioJournaling,
            ENTRADAS_JOURNAL,
            "rmdir",
            rutaCompleta,
            "",
            sb,
        ); err != nil {
            // No fallar la operación principal, solo mostrar advertencia
            fmt.Printf("Advertencia: error registrando operación en journal: %v\n", err)
        } else {
            fmt.Printf("Operación 'rmdir %s' registrada en journal exitosamente\n", rutaCompleta)
        }
    }

    // 5. Obtener todos los bloques de datos del directorio
    indicesBloques, err := inodoDirectorio.ObtenerIndicesBloquesDatos(archivo, sb)
    if err != nil {
        return fmt.Errorf("error obteniendo bloques de datos del directorio: %w", err)
    }

    // 6. Procesar cada bloque del directorio
    for _, indiceBloques := range indicesBloques {
        // 6.1 Cargar el bloque de directorio
        bloqueDir := &FolderBlock{}
        offsetBloques := int64(sb.S_block_start + indiceBloques*sb.S_block_size)
        if err := bloqueDir.Decodificar(archivo, offsetBloques); err != nil {
            return fmt.Errorf("error deserializando bloque %d: %w", indiceBloques, err)
        }

        // 6.2 Procesar cada entrada en el bloque
        for _, contenido := range bloqueDir.B_content {
            // Saltar entradas vacías o especiales (. y ..)
            if contenido.B_inodo == -1 ||
                string(contenido.B_name[:1]) == "." ||
                string(contenido.B_name[:2]) == ".." {
                continue
            }

            // 6.3 Obtener el nombre del contenido y construir su ruta
            nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")
            fmt.Printf("Eliminando contenido '%s' en inodo %d\n", nombreContenido, contenido.B_inodo)

            rutaHijo := rutaCompleta
            if !strings.HasSuffix(rutaHijo, "/") {
                rutaHijo += "/"
            }
            rutaHijo += nombreContenido

            // 6.4 Cargar el inodo del contenido
            inodoHijo := &INodo{}
            offsetInodoHijo := int64(sb.S_inode_start + (contenido.B_inodo * sb.S_inode_size))
            if err := inodoHijo.Decodificar(archivo, offsetInodoHijo); err != nil {
                return fmt.Errorf("error deserializando inodo hijo %d: %w", contenido.B_inodo, err)
            }

            // 6.5 Procesar según el tipo de contenido (directorio o archivo)
            if inodoHijo.I_type[0] == '0' { // Es un directorio
                // Registrar eliminación de carpeta en journal
                if sb.S_filesystem_type == 3 {
                    if err := AgregarEntradaJournal(
                        archivo,
                        inicioJournaling,
                        ENTRADAS_JOURNAL,
                        "rmdir",
                        rutaHijo,
                        "",
                        sb,
                    ); err != nil {
                        fmt.Printf("Advertencia: error registrando eliminación de subcarpeta en journal: %v\n", err)
                    }
                }

                // Eliminar recursivamente la subcarpeta
                if err := sb.eliminarCarpetaEnInodo(archivo, contenido.B_inodo, rutaHijo); err != nil {
                    return fmt.Errorf("error eliminando subcarpeta '%s': %w", nombreContenido, err)
                }
            } else { // Es un archivo
                // Registrar eliminación de archivo en journal
                if sb.S_filesystem_type == 3 {
                    // Obtener el contenido del archivo para el journal
                    datosArchivo, err := inodoHijo.LeerDatos(archivo, sb)
                    contenidoArchivo := ""
                    if err == nil {
                        contenidoArchivo = string(datosArchivo)
                    }

                    if err := AgregarEntradaJournal(
                        archivo,
                        inicioJournaling,
                        ENTRADAS_JOURNAL,
                        "rm",
                        rutaHijo,
                        contenidoArchivo,
                        sb,
                    ); err != nil {
                        fmt.Printf("Advertencia: error registrando eliminación de archivo '%s' en journal: %v\n", nombreContenido, err)
                    } else {
                        fmt.Printf("Operación 'rm %s' registrada en journal exitosamente\n", rutaHijo)
                    }
                }

                // Liberar todos los bloques del archivo
                if err := inodoHijo.LiberarTodosLosBloques(archivo, sb); err != nil {
                    return fmt.Errorf("error liberando bloques del archivo '%s': %w", nombreContenido, err)
                }

                // Liberar el inodo del archivo
                if err := sb.ActualizarBitmapInodo(archivo, contenido.B_inodo, false); err != nil {
                    return fmt.Errorf("error liberando inodo %d: %w", contenido.B_inodo, err)
                }
                sb.ActualizarSuperblockDespuesDesasignacionInodo()
                fmt.Printf("Archivo '%s' eliminado exitosamente (inodo %d)\n", nombreContenido, contenido.B_inodo)
            }
        }
    }

    // 7. Liberar todos los bloques del directorio
    if err := inodoDirectorio.LiberarTodosLosBloques(archivo, sb); err != nil {
        return fmt.Errorf("error liberando bloques del directorio: %w", err)
    }

    // 8. Verificar y liberar bloques de apuntadores indirectos vacíos
    if err := inodoDirectorio.VerificarYLiberarBloquesIndirectosVacios(archivo, sb); err != nil {
        fmt.Printf("Advertencia: error al verificar bloques indirectos vacíos: %v\n", err)
    }

    // 9. Liberar el inodo del directorio
    if err := sb.ActualizarBitmapInodo(archivo, indiceInodo, false); err != nil {
        return fmt.Errorf("error liberando inodo del directorio %d: %w", indiceInodo, err)
    }
    sb.ActualizarSuperblockDespuesDesasignacionInodo()

    fmt.Printf("Carpeta en inodo %d eliminada exitosamente.\n", indiceInodo)
    return nil
}

// EliminarCarpeta elimina un directorio y su contenido recursivamente en el sistema de archivos
func (sb *SuperBlock) EliminarCarpeta(archivo *os.File, directoriosPadre []string, nombreCarpeta string) error {
    fmt.Printf("Intentando eliminar carpeta '%s'\n", nombreCarpeta)

    // Construir la ruta completa para el registro de journaling
    var rutaCompleta string
    if len(directoriosPadre) > 0 {
        rutaCompleta = "/" + strings.Join(directoriosPadre, "/") + "/" + nombreCarpeta
    } else {
        rutaCompleta = "/" + nombreCarpeta
    }
    fmt.Printf("Ruta completa para eliminación: %s\n", rutaCompleta)

    // Si no hay directorio padre, eliminar desde el directorio raíz
    if len(directoriosPadre) == 0 {
        return sb.eliminarCarpetaDelDirectorio(archivo, 0, nombreCarpeta, rutaCompleta)
    }

    // Navegar recursivamente por la estructura de directorios
    indiceInodoActual := int32(0) // Comenzamos desde el directorio raíz

    // Recorrer cada nivel de directorio padre
    for _, nombreDir := range directoriosPadre {
        encontrado := false

        // Cargar el inodo del directorio actual
        inodoActual := &INodo{}
        if err := inodoActual.Decodificar(archivo, int64(sb.S_inode_start+indiceInodoActual*sb.S_inode_size)); err != nil {
            return fmt.Errorf("error cargando directorio actual (inodo %d): %w", indiceInodoActual, err)
        }

        // Verificar que sea un directorio
        if inodoActual.I_type[0] != '0' {
            return fmt.Errorf("el inodo %d no corresponde a un directorio", indiceInodoActual)
        }

        // Obtener bloques de datos del directorio
        indicesBloques, err := inodoActual.ObtenerIndicesBloquesDatos(archivo, sb)
        if err != nil {
            return fmt.Errorf("error obteniendo bloques de directorio: %w", err)
        }

        // Buscar el siguiente directorio en la ruta de navegación
        for _, indiceBloques := range indicesBloques {
            if encontrado {
                break
            }

            bloque := &FolderBlock{}
            if err := bloque.Decodificar(archivo, int64(sb.S_block_start+indiceBloques*sb.S_block_size)); err != nil {
                return fmt.Errorf("error deserializando bloque %d: %w", indiceBloques, err)
            }

            // Buscar la carpeta en este bloque
            for _, contenido := range bloque.B_content {
                nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")

                if contenido.B_inodo != -1 && strings.EqualFold(nombreContenido, nombreDir) {
                    // Verificar que sea un directorio
                    inodoSubDir := &INodo{}
                    if err := inodoSubDir.Decodificar(archivo, int64(sb.S_inode_start+contenido.B_inodo*sb.S_inode_size)); err != nil {
                        return fmt.Errorf("error cargando inodo %d: %w", contenido.B_inodo, err)
                    }

                    if inodoSubDir.I_type[0] != '0' {
                        return fmt.Errorf("la entrada '%s' no es un directorio", nombreDir)
                    }

                    // Avanzar al siguiente directorio en la jerarquía
                    indiceInodoActual = contenido.B_inodo
                    encontrado = true
                    break
                }
            }
        }

        if !encontrado {
            return fmt.Errorf("no se encontró el directorio '%s' en la ruta especificada", nombreDir)
        }
    }

    // Llegamos al directorio que debería contener la carpeta a eliminar
    return sb.eliminarCarpetaDelDirectorio(archivo, indiceInodoActual, nombreCarpeta, rutaCompleta)
}

// eliminarCarpetaDelDirectorio método auxiliar para eliminar una carpeta de un directorio específico
func (sb *SuperBlock) eliminarCarpetaDelDirectorio(archivo *os.File, indiceInodoPadre int32, nombreCarpeta string, rutaCompleta string) error {
    // Cargar el inodo del directorio padre
    inodoPadre := &INodo{}
    if err := inodoPadre.Decodificar(archivo, int64(sb.S_inode_start+indiceInodoPadre*sb.S_inode_size)); err != nil {
        return fmt.Errorf("error deserializando inodo del directorio padre %d: %w", indiceInodoPadre, err)
    }

    // Verificar que sea un directorio
    if inodoPadre.I_type[0] != '0' {
        return fmt.Errorf("el inodo %d no corresponde a un directorio", indiceInodoPadre)
    }

    // Obtener bloques de datos del directorio padre
    indicesBloques, err := inodoPadre.ObtenerIndicesBloquesDatos(archivo, sb)
    if err != nil {
        return fmt.Errorf("error obteniendo bloques de datos del directorio: %w", err)
    }

    // Buscar la carpeta objetivo a eliminar
    for _, indiceBloques := range indicesBloques {
        bloque := &FolderBlock{}
        offsetBloques := int64(sb.S_block_start + indiceBloques*sb.S_block_size)

        if err := bloque.Decodificar(archivo, offsetBloques); err != nil {
            return fmt.Errorf("error deserializando bloque %d: %w", indiceBloques, err)
        }

        // Buscar la entrada específica en el directorio
        for i, contenido := range bloque.B_content {
            nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")

            if contenido.B_inodo != -1 && strings.EqualFold(nombreContenido, nombreCarpeta) {
                // Verificar que la entrada corresponde a un directorio
                inodoCarpeta := &INodo{}
                if err := inodoCarpeta.Decodificar(archivo, int64(sb.S_inode_start+contenido.B_inodo*sb.S_inode_size)); err != nil {
                    return fmt.Errorf("error deserializando inodo %d: %w", contenido.B_inodo, err)
                }

                if inodoCarpeta.I_type[0] != '0' {
                    return fmt.Errorf("'%s' no es un directorio válido", nombreCarpeta)
                }

                // Eliminar el directorio recursivamente usando la ruta completa
                if err := sb.eliminarCarpetaEnInodo(archivo, contenido.B_inodo, rutaCompleta); err != nil {
                    return fmt.Errorf("error eliminando carpeta '%s': %w", nombreCarpeta, err)
                }

                // Limpiar la entrada en el directorio padre
                bloque.B_content[i] = FolderContent{
                    B_name:  [12]byte{'-'},
                    B_inodo: -1,
                }

                // Guardar el bloque actualizado con la entrada eliminada
                if err := bloque.Codificar(archivo, offsetBloques); err != nil {
                    return fmt.Errorf("error actualizando bloque de directorio padre: %w", err)
                }

                fmt.Printf("Carpeta '%s' eliminada exitosamente del sistema\n", nombreCarpeta)
                return nil
            }
        }
    }

    return fmt.Errorf("carpeta '%s' no encontrada en el directorio especificado", nombreCarpeta)
}

/* 
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
				sb.ActualizarSuperblockDespuesAsignacionInodo()

				fmt.Printf("Directorio '%s' creado correctamente en inodo %d.\n", directorioDestino, sb.S_inodes_count) // Depuración
				return nil
			}
		}
	}
*/