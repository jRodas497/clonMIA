package Estructuras

import (
    "fmt"
    "os"
    "strings"
    "time"
)

func (sb *SuperBlock) crearArchivoEnInodo( archivo *os.File, indiceInodo int32,
    directoriosPadre []string, // carpetas que faltan por bajar
    archivoDestino string,     // nombre del archivo a crear
    tamanoArchivo int,         // tamaño solicitado (solo informativo)
    contenidoArchivo []string, // datos divididos en chunks de ≤64 B
    verboso bool,              // prints de depuración
) error {

    if verboso {
        fmt.Printf("<- crear '%s' en inodo %d\n", archivoDestino, indiceInodo)
    }

    inodoDirectorio := &INodo{}
    if err := inodoDirectorio.Decodificar(archivo, int64(sb.S_inode_start+indiceInodo*sb.S_inode_size)); err != nil {
        return fmt.Errorf("decodificar inodo %d: %w", indiceInodo, err)
    }
    if inodoDirectorio.I_type[0] != '0' { // no es carpeta: abortar silencioso
        return nil
    }

    /* Si aún quedan carpetas por bajar, descender recursivamente */
    if len(directoriosPadre) > 0 {
        buscar := strings.Trim(directoriosPadre[0], "\x00 ")
        indicesBloques, _ := inodoDirectorio.ObtenerIndicesBloquesDatos(archivo, sb)

        for _, idxBloque := range indicesBloques {
            bloque := &FolderBlock{}
            if err := bloque.Decodificar(archivo, int64(sb.S_block_start+idxBloque*sb.S_block_size)); err != nil {
                return err
            }
            for _, entrada := range bloque.B_cont {
                nombre := strings.Trim(string(entrada.B_name[:]), "\x00 ")
                if entrada.B_inodo != -1 && strings.EqualFold(nombre, buscar) {
                    return sb.crearArchivoEnInodo(
                        archivo,
                        entrada.B_inodo,
                        directoriosPadre[1:],
                        archivoDestino,
                        tamanoArchivo,
                        contenidoArchivo,
                        verboso,
                    )
                }
            }
        } return fmt.Errorf("no se encontró la carpeta '%s'", buscar)
    }

    /* Estamos en el directorio destino — buscar hueco libre */
    var (
        indiceBloqueLibre int32 = -1
        posicionEntradaLibre int = -1
        offsetBloque     int64
    )

    indicesBloques, _ := inodoDirectorio.ObtenerIndicesBloquesDatos(archivo, sb)
	busqueda:
		for _, idxBloque := range indicesBloques {
			bloque := &FolderBlock{}
			offsetBloque = int64(sb.S_block_start + idxBloque*sb.S_block_size)
			if err := bloque.Decodificar(archivo, offsetBloque); err != nil {
				return err
			}

			// Duplicado
			for _, entrada := range bloque.B_cont {
				if strings.Trim(string(entrada.B_name[:]), "\x00 ") == archivoDestino && entrada.B_inodo != -1 {
					return fmt.Errorf("el archivo '%s' ya existe", archivoDestino)
				}
			}
			// Primer hueco libre (pos≥2)
			for p := 2; p < len(bloque.B_cont); p++ {
				if bloque.B_cont[p].B_inodo == -1 {
					indiceBloqueLibre, posicionEntradaLibre = idxBloque, p
					break busqueda
				}
			}
		}

    	/* Si no hay hueco -> añadir bloque nuevo al directorio */
		if indiceBloqueLibre == -1 {
			var err error
			indiceBloqueLibre, err = inodoDirectorio.AgregarBloque(archivo, sb)
			if err != nil {
				return err
			}
			bloque := &FolderBlock{}
			for i := range bloque.B_cont {
				bloque.B_cont[i] = FolderContent{B_name: [12]byte{'-'}, B_inodo: -1}
			}
			offsetBloque = int64(sb.S_block_start + indiceBloqueLibre*sb.S_block_size)
			if err = bloque.Codificar(archivo, offsetBloque); err != nil {
				return err
			}
			posicionEntradaLibre = 0
		}

		/* Reservar nuevo inodo */
		nuevoIndiceInodo, err := sb.BuscarSiguienteInodoLibre(archivo)
		if err != nil {
			return err
		}
		if err := sb.ActualizarBitmapInodo(archivo, nuevoIndiceInodo, true); err != nil {
			return err
		}

    inodoArchivo := NuevoInodoVacio()
    inodoArchivo.I_type[0] = '1'
    inodoArchivo.I_perm = [3]byte{'6', '6', '4'}
    inodoArchivo.I_uid, inodoArchivo.I_gid = 1, 1
    ahora := float32(time.Now().Unix())
    inodoArchivo.I_atime, inodoArchivo.I_ctime, inodoArchivo.I_mtime = ahora, ahora, ahora

    offsetInodo := int64(sb.S_inode_start + nuevoIndiceInodo*sb.S_inode_size)
    if err := inodoArchivo.Codificar(archivo, offsetInodo); err != nil {
        sb.ActualizarBitmapInodo(archivo, nuevoIndiceInodo, false)
        return err
    }

    /* 5. Escribir datos */
    datos := []byte(strings.Join(contenidoArchivo, ""))
    if err := inodoArchivo.EscribirDatos(archivo, sb, datos); err != nil {
        inodoArchivo.LiberarTodosLosBloques(archivo, sb)
        sb.ActualizarBitmapInodo(archivo, nuevoIndiceInodo, false)
        return err
    }
    inodoArchivo.I_size = int32(len(datos))
    inodoArchivo.ActualizarMtime()
    inodoArchivo.ActualizarCtime()
    if err := inodoArchivo.Codificar(archivo, offsetInodo); err != nil {
        return err
    }

    /* 6. Actualizar el bloque de directorio con la nueva entrada */
    bloqueDirectorio := &FolderBlock{}
    if err := bloqueDirectorio.Decodificar(archivo, offsetBloque); err != nil {
        return err
    }
    copy(bloqueDirectorio.B_cont[posicionEntradaLibre].B_name[:], archivoDestino)
    bloqueDirectorio.B_cont[posicionEntradaLibre].B_inodo = nuevoIndiceInodo
    if err := bloqueDirectorio.Codificar(archivo, offsetBloque); err != nil {
        return err
    }

    /* 7. Metadatos directorio + superbloque */
    inodoDirectorio.I_size++
    inodoDirectorio.ActualizarMtime()
    if err := inodoDirectorio.Codificar(archivo, int64(sb.S_inode_start+indiceInodo*sb.S_inode_size)); err != nil {
        return err
    }
    sb.ActualizarSuperblockDespuesAsignacionInodo()

    if verboso {
        fmt.Printf("'%s' creado (inodo %d  bloque %d:%d)\n",
            archivoDestino, nuevoIndiceInodo, indiceBloqueLibre, posicionEntradaLibre)
    }
    return nil
}

// CrearArchivo – API pública
func (sb *SuperBlock) CrearArchivo(
    archivo *os.File,
    directoriosPadre []string,
    archivoDestino string,
    tamano int,
    contenido []string,
    log bool,
) error {

    // Crear el archivo partiendo SIEMPRE del inodo 0 (raíz).
    if err := sb.crearArchivoEnInodo(
        archivo,
        0,                 	// inodo raíz
        directoriosPadre,  	// por descender
        archivoDestino,
        tamano,
        contenido,
        false, 				// verboso interno
    ); err != nil {
        return err
    }

    // Registrar en journal (una única vez y con la ruta completa).
    if log && sb.S_filesystem_type == 3 {
        rutaCompleta := "/" + strings.Join(append(directoriosPadre, archivoDestino), "/")
        if err := AgregarEntradaJournal(
            archivo,
            int64(sb.InicioJournal()),
            ENTRADAS_JOURNAL,
            "mkfile",
            rutaCompleta,
            strings.Join(contenido, ""),
            sb); err != nil {

            fmt.Printf("WARN journal: %v\n", err) // no abortamos la creación
        }
    }

    return nil
}

// BuscarInodoDirectorio busca y retorna el índice del inodo correspondiente a un directorio específico
func (sb *SuperBlock) BuscarInodoDirectorio(archivo *os.File, directoriosPadre []string) (int32, error) {
    if len(directoriosPadre) == 0 {
        return 0, nil // Si no hay ruta, devuelve el directorio raíz (inodo 0)
    }

    // Comenzamos la búsqueda desde el directorio raíz
    indiceInodoActual := int32(0)

    // Recorrer cada nivel de directorio en la ruta
    for i, nombreDirectorio := range directoriosPadre {
        encontrado := false

        // Cargar el inodo del directorio actual
        inodoActual := &INodo{}
        err := inodoActual.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodoActual*sb.S_inode_size)))
        if err != nil {
            return -1, fmt.Errorf("error al deserializar inodo %d: %w", indiceInodoActual, err)
        }

        // Verificar que sea un directorio
        if inodoActual.I_type[0] != '0' {
            return -1, fmt.Errorf("el inodo %d no es un directorio", indiceInodoActual)
        }

        // Obtener los bloques de datos del directorio
        indicesBloques, err := inodoActual.ObtenerIndicesBloquesDatos(archivo, sb)
        if err != nil {
            return -1, fmt.Errorf("error obteniendo bloques de datos: %w", err)
        }

        if len(indicesBloques) == 0 {
            return -1, fmt.Errorf("directorio en inodo %d no tiene bloques asignados", indiceInodoActual)
        }

        // Buscar el subdirectorio en cada bloque
        for _, indiceBloque := range indicesBloques {
            if encontrado {
                break
            }

            // Deserializar el bloque
            bloque := &FolderBlock{}
            offsetBloque := int64(sb.S_block_start + indiceBloque*sb.S_block_size)

            if err := bloque.Decodificar(archivo, offsetBloque); err != nil {
                return -1, fmt.Errorf("error deserializando bloque %d: %w", indiceBloque, err)
            }

            // Buscar el nombre del directorio en este bloque
            for _, contenido := range bloque.B_cont {
                if contenido.B_inodo == -1 {
                    continue // Saltar entradas vacías
                }

                // Obtener el nombre normalizado
                nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")
                nombreDirectorioLimpio := strings.Trim(nombreDirectorio, "\x00 ")

                // Comprobar coincidencia exacta o con truncamiento a 12 caracteres
                if strings.EqualFold(nombreContenido, nombreDirectorioLimpio) ||
                    (len(nombreDirectorioLimpio) > 12 && strings.EqualFold(nombreContenido, nombreDirectorioLimpio[:12])) {

                    // Verificar que sea un directorio (excepto para el último elemento si es un archivo)
                    inodoEntrada := &INodo{}
                    offsetInodoEntrada := int64(sb.S_inode_start + (contenido.B_inodo * sb.S_inode_size))

                    if err := inodoEntrada.Decodificar(archivo, offsetInodoEntrada); err != nil {
                        return -1, fmt.Errorf("error deserializando inodo %d: %w", contenido.B_inodo, err)
                    }

                    // Si no es el último elemento de la ruta, debe ser un directorio
                    if i < len(directoriosPadre)-1 && inodoEntrada.I_type[0] != '0' {
                        return -1, fmt.Errorf("'%s' no es un directorio", nombreDirectorio)
                    }

                    // Avanzar al siguiente inodo
                    indiceInodoActual = contenido.B_inodo
                    encontrado = true
                    fmt.Printf("Directorio '%s' encontrado en inodo %d\n", nombreDirectorio, indiceInodoActual)
                    break
                }
            }
        }

        if !encontrado {
            return -1, fmt.Errorf("directorio '%s' no encontrado en la ruta", nombreDirectorio)
        }
    }

    return indiceInodoActual, nil
}

// eliminarArchivoEnInodo elimina un archivo en un inodo específico utilizando las funciones avanzadas
func (sb *SuperBlock) eliminarArchivoEnInodo(archivo *os.File, indiceInodo int32, nombreArchivo string, rutaPadre ...string) error {
    // 1. Cargar el inodo del directorio
    inodoDirectorio := &INodo{}
    err := inodoDirectorio.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al deserializar inodo %d: %w", indiceInodo, err)
    }

    // Verificar que el inodo sea un directorio
    if inodoDirectorio.I_type[0] != '0' {
        return fmt.Errorf("el inodo %d no es una carpeta", indiceInodo)
    }

    // Obtener solo los bloques de datos del directorio (no apuntadores)
    indicesBloques, err := inodoDirectorio.ObtenerIndicesBloquesDatos(archivo, sb)
    if err != nil {
        return fmt.Errorf("error obteniendo bloques de datos del directorio: %w", err)
    }

    // Procesar cada bloque del directorio
    for _, indiceBloque := range indicesBloques {
        bloque := &FolderBlock{}
        offsetBloque := int64(sb.S_block_start + indiceBloque*sb.S_block_size)

        if err := bloque.Decodificar(archivo, offsetBloque); err != nil {
            return fmt.Errorf("error deserializando bloque %d: %w", indiceBloque, err)
        }

        // Buscar el archivo en el bloque
        for i, contenido := range bloque.B_cont {
            nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")

            if contenido.B_inodo != -1 && strings.EqualFold(nombreContenido, nombreArchivo) {
                indiceInodoArchivo := contenido.B_inodo
                fmt.Printf("Archivo '%s' encontrado en inodo %d, eliminando.\n", nombreArchivo, indiceInodoArchivo)

                // Cargar el inodo del archivo
                inodoArchivo := &INodo{}
                offsetInodoArchivo := int64(sb.S_inode_start + (indiceInodoArchivo * sb.S_inode_size))
                if err := inodoArchivo.Decodificar(archivo, offsetInodoArchivo); err != nil {
                    return fmt.Errorf("error deserializando inodo del archivo %d: %w", indiceInodoArchivo, err)
                }

                // Verificar que sea efectivamente un archivo
                if inodoArchivo.I_type[0] != '1' {
                    return fmt.Errorf("el inodo %d no es un archivo sino de tipo %c", indiceInodoArchivo, inodoArchivo.I_type[0])
                }

                // Registrar en journal si es necesario
                if sb.S_filesystem_type == 3 {
                    var rutaCompleta string

                    // Si se proporciona una ruta de directorio padre
                    if len(rutaPadre) > 0 && rutaPadre[0] != "" {
                        rutaCompleta = rutaPadre[0] + "/" + nombreArchivo
                    } else {
                        rutaCompleta = "/" + nombreArchivo
                    }

                    inicioJournaling := int64(sb.InicioJournal())

                    // Cargar los contenidos del archivo para el journal
                    datosArchivo, err := inodoArchivo.LeerDatos(archivo, sb)
                    contenidoArchivo := ""
                    if err != nil {
                        // No fallamos aquí, solo log
                        fmt.Printf("Error leyendo contenido del archivo para journal: %v\n", err)
                    } else {
                        contenidoArchivo = string(datosArchivo)
                    }

                    // Usar AgregarEntradaJournal que maneja automáticamente índices y serialización
                    if err := AgregarEntradaJournal(
                        archivo,
                        inicioJournaling,
                        ENTRADAS_JOURNAL,
                        "rm",
                        rutaCompleta,
                        contenidoArchivo,
                        sb,
                    ); err != nil {
                        fmt.Printf("Advertencia: error registrando operación en journal: %v\n", err)
                        // Continuamos a pesar del error en el journal
                    } else {
                        fmt.Printf("Operación 'rm %s' registrada en journal correctamente\n", rutaCompleta)
                    }
                }

                // Liberar todos los bloques del archivo usando LiberarTodosLosBloques
                if err := inodoArchivo.LiberarTodosLosBloques(archivo, sb); err != nil {
                    return fmt.Errorf("error liberando bloques del archivo: %w", err)
                }

                // Liberar el inodo
                if err := sb.ActualizarBitmapInodo(archivo, indiceInodoArchivo, false); err != nil {
                    return fmt.Errorf("error liberando inodo %d: %w", indiceInodoArchivo, err)
                }
                sb.ActualizarSuperblockDespuesDesasignacionInodo()

                // Limpiar la entrada en el directorio
                bloque.B_cont[i] = FolderContent{B_name: [12]byte{'-'}, B_inodo: -1}
                if err := bloque.Codificar(archivo, offsetBloque); err != nil {
                    return fmt.Errorf("error actualizando bloque de directorio: %w", err)
                }

                // Verificar y liberar bloques de apuntadores vacíos
                if err := inodoDirectorio.VerificarYLiberarBloquesIndirectosVacios(archivo, sb); err != nil {
                    fmt.Printf("Advertencia: error al verificar bloques indirectos vacíos: %v\n", err)
                }

                fmt.Printf("Archivo '%s' eliminado correctamente.\n", nombreArchivo)
                return nil
            }
        }
    }

    return fmt.Errorf("archivo '%s' no encontrado en directorio (inodo %d)", nombreArchivo, indiceInodo)
}

// EliminarArchivo elimina un archivo del sistema de archivos
func (sb *SuperBlock) EliminarArchivo(archivo *os.File, directoriosPadre []string, nombreArchivo string) error {
    fmt.Printf("Intentando eliminar archivo '%s'\n", nombreArchivo)

    // Si no hay directorio padre, eliminar desde el directorio raíz
    if len(directoriosPadre) == 0 {
        return sb.eliminarArchivoEnInodo(archivo, 0, nombreArchivo)
    }

    // Navegar recursivamente por la estructura de directorios
    indiceInodoActual := int32(0) // Comenzamos desde el directorio raíz

    // Recorrer cada nivel de directorio padre
    for _, nombreDirectorio := range directoriosPadre {
        encontrado := false

        // Cargar el inodo del directorio actual
        inodoActual := &INodo{}
        if err := inodoActual.Decodificar(archivo, int64(sb.S_inode_start+indiceInodoActual*sb.S_inode_size)); err != nil {
            return fmt.Errorf("error cargando directorio actual (inodo %d): %w", indiceInodoActual, err)
        }

        // Obtener bloques de datos del directorio
        indicesBloques, err := inodoActual.ObtenerIndicesBloquesDatos(archivo, sb)
        if err != nil {
            return fmt.Errorf("error obteniendo bloques de directorio: %w", err)
        }

        // Buscar el siguiente directorio en la ruta
        for _, indiceBloque := range indicesBloques {
            if encontrado {
                break
            }

            bloque := &FolderBlock{}
            if err := bloque.Decodificar(archivo, int64(sb.S_block_start+indiceBloque*sb.S_block_size)); err != nil {
                return fmt.Errorf("error deserializando bloque %d: %w", indiceBloque, err)
            }

            // Buscar la carpeta en este bloque
            for _, contenido := range bloque.B_cont {
                nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")

                if contenido.B_inodo != -1 && strings.EqualFold(nombreContenido, nombreDirectorio) {
                    // Verificar que sea un directorio
                    inodoSubDirectorio := &INodo{}
                    if err := inodoSubDirectorio.Decodificar(archivo, int64(sb.S_inode_start+contenido.B_inodo*sb.S_inode_size)); err != nil {
                        return fmt.Errorf("error cargando inodo %d: %w", contenido.B_inodo, err)
                    }

                    if inodoSubDirectorio.I_type[0] != '0' {
                        return fmt.Errorf("la entrada '%s' no es un directorio", nombreDirectorio)
                    }

                    // Avanzar al siguiente directorio
                    indiceInodoActual = contenido.B_inodo
                    encontrado = true
                    break
                }
            }
        }

        if !encontrado {
            return fmt.Errorf("no se encontró el directorio '%s' en la ruta", nombreDirectorio)
        }
    }

    // Llegamos al directorio que debería contener el archivo
    return sb.eliminarArchivoEnInodo(archivo, indiceInodoActual, nombreArchivo)
}