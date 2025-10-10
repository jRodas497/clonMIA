package Forge

import (
    "bytes"
    "errors"
    "fmt"
    "os"
    "regexp"
    "strings"

    Estructuras "backend/Estructuras"
    Global "backend/Global"
    Utils "backend/Utils"
)

// COPY estructura del comando copy con parámetros
type COPY struct {
    path    string // Ruta del archivo o carpeta que se desea copiar
    destino string // Ruta destino donde se va a copiar el contenido
}

// ParserCopy parsea el comando copy y devuelve una instancia de COPY
func ParserCopy(tokens []string) (string, error) {
    cmd := &COPY{}               // Crea una nueva instancia de COPY
    var bufferSalida bytes.Buffer // Buffer para capturar mensajes importantes

    // Expresión regular para capturar los parámetros -path y -destino
    re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-destino="[^"]+"|-destino=[^\s]+`)
    matches := re.FindAllString(strings.Join(tokens, " "), -1)

    // Verificar que se han proporcionado ambos parámetros
    if len(matches) < 2 {
        return "", errors.New("faltan parámetros requeridos: -path y -destino son obligatorios")
    }

    // Iterar sobre cada coincidencia y extraer los valores de -path y -destino
    for _, match := range matches {
        kv := strings.SplitN(match, "=", 2)
        if len(kv) != 2 {
            continue
        }
        key := strings.ToLower(kv[0])
        value := strings.Trim(kv[1], "\"") // Eliminar comillas si existen

        // Asignar los valores de los parámetros
        switch key {
        case "-path":
            cmd.path = value
        case "-destino":
            cmd.destino = value
        }
    }

    // Verificar que ambos parámetros tengan valores
    if cmd.path == "" || cmd.destino == "" {
        return "", errors.New("los parámetros -path y -destino son obligatorios")
    }

    // Ejecutar el comando COPY
    err := comandoCopy(cmd, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

// comandoCopy ejecuta la lógica principal del comando copy
func comandoCopy(comandoCopy *COPY, bufferSalida *bytes.Buffer) error {
    fmt.Fprint(bufferSalida, "======================== COPY ========================\n")

    // Verificar si hay un usuario logueado
    if !Global.VerificarSesionActiva() {
        return fmt.Errorf("no hay un usuario logueado")
    }

    // Obtener el ID de la partición desde el usuario logueado
    idParticion := Global.UsuarioActual.Id

    // Obtener la partición montada asociada al usuario logueado
    superBloqueParticion, _, rutaParticion, err := Global.ObtenerSuperblockParticionMontada(idParticion)
    if err != nil {
        return fmt.Errorf("error al obtener la partición montada: %w", err)
    }

    // Abrir el archivo de partición para operar sobre él
    archivo, err := os.OpenFile(rutaParticion, os.O_RDWR, 0666)
    if err != nil {
        return fmt.Errorf("error al abrir el archivo de partición: %w", err)
    }
    defer archivo.Close() // Cerrar el archivo cuando ya no sea necesario

    fmt.Fprintf(bufferSalida, "Copiando desde: %s\n", comandoCopy.path)
    fmt.Fprintf(bufferSalida, "Hacia destino: %s\n", comandoCopy.destino)

    // Verificar que la ruta origen existe y obtener su tipo
    directoriosPadreOrigen, nombreOrigen := Utils.ObtenerDirectoriosPadre(comandoCopy.path)
    indiceInodoOrigen, esDirectorio, err := buscarElemento(archivo, superBloqueParticion, directoriosPadreOrigen, nombreOrigen)
    if err != nil {
        return fmt.Errorf("error: la ruta origen '%s' no existe: %w", comandoCopy.path, err)
    }

    // Verificar permisos de lectura en el elemento origen
    if !verificarPermisosLectura(archivo, superBloqueParticion, indiceInodoOrigen) {
        return fmt.Errorf("error: no tiene permisos de lectura sobre '%s'", comandoCopy.path)
    }

    // Verificar que el directorio destino existe
    directoriosPadreDestino, _ := Utils.ObtenerDirectoriosPadre(comandoCopy.destino)
    indiceInodoDestino, err := buscarInodoCarpeta(archivo, superBloqueParticion, directoriosPadreDestino)
    if err != nil {
        return fmt.Errorf("error: el directorio destino '%s' no existe: %w", comandoCopy.destino, err)
    }

    // Verificar permisos de escritura en el directorio destino
    if !verificarPermisosEscritura(archivo, superBloqueParticion, indiceInodoDestino) {
        return fmt.Errorf("error: no tiene permisos de escritura sobre el directorio destino '%s'", comandoCopy.destino)
    }

    // Realizar la copia según el tipo de elemento
    if esDirectorio {
        err = copiarDirectorio(archivo, superBloqueParticion, indiceInodoOrigen, indiceInodoDestino, nombreOrigen, bufferSalida)
    } else {
        err = copiarArchivo(archivo, superBloqueParticion, indiceInodoOrigen, indiceInodoDestino, nombreOrigen, bufferSalida)
    }

    if err != nil {
        return fmt.Errorf("error durante la copia: %w", err)
    }

    fmt.Fprintf(bufferSalida, "Copia completada exitosamente\n")
    fmt.Fprint(bufferSalida, "=====================================================\n")

    return nil
}

// buscarElemento busca un archivo o directorio y determina su tipo
func buscarElemento(archivo *os.File, sb *Estructuras.SuperBlock, directoriosPadre []string, nombreElemento string) (int32, bool, error) {
    // Buscar el directorio padre
    indiceInodoPadre, err := buscarInodoCarpeta(archivo, sb, directoriosPadre)
    if err != nil {
        return -1, false, err
    }

    // Buscar el elemento en el directorio padre
    encontrado, indiceInodoElemento, err := directorioExiste(sb, archivo, indiceInodoPadre, nombreElemento)
    if err != nil {
        return -1, false, err
    }
    if !encontrado {
        return -1, false, fmt.Errorf("elemento '%s' no encontrado", nombreElemento)
    }

    // Verificar el tipo del elemento
    inodo := &Estructuras.INodo{}
    err = inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodoElemento*sb.S_inode_size)))
    if err != nil {
        return -1, false, fmt.Errorf("error al leer inodo del elemento: %w", err)
    }

    esDirectorio := inodo.I_type[0] == '0'
    return indiceInodoElemento, esDirectorio, nil
}

// verificarPermisosLectura verifica si el usuario actual tiene permisos de lectura
func verificarPermisosLectura(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32) bool {
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return false
    }

    // Verificar permisos según el owner y group
    // Para este ejemplo, asumimos que el permiso '6' incluye lectura (4)
    permisos := string(inodo.I_perm[:])
    return strings.Contains(permisos, "6") || strings.Contains(permisos, "4")
}

// verificarPermisosEscritura verifica si el usuario actual tiene permisos de escritura
func verificarPermisosEscritura(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32) bool {
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return false
    }

    // Verificar permisos según el owner y group
    // Para este ejemplo, asumimos que el permiso '6' incluye escritura (2)
    permisos := string(inodo.I_perm[:])
    return strings.Contains(permisos, "6") || strings.Contains(permisos, "2")
}

// copiarArchivo copia un archivo individual
func copiarArchivo(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodoOrigen int32, indiceInodoDestino int32, nombreArchivo string, bufferSalida *bytes.Buffer) error {
    fmt.Fprintf(bufferSalida, "Copiando archivo: %s\n", nombreArchivo)

    // Leer contenido del archivo origen usando leerArchivoDesdeInodo de cat.go
    contenido, err := leerArchivoDesdeInodo(archivo, sb, indiceInodoOrigen)
    if err != nil {
        return fmt.Errorf("error al leer archivo origen: %w", err)
    }

    // Crear nuevo inodo para el archivo destino usando NuevoInodoVacio
    nuevoInodo := Estructuras.NuevoInodoVacio()
    nuevoInodo.I_type[0] = '1' // Tipo archivo
    nuevoInodo.I_size = int32(len(contenido))
    copy(nuevoInodo.I_perm[:], "664")

    // Buscar inodo libre para el nuevo archivo
    nuevoIndiceInodo, err := sb.BuscarSiguienteInodoLibre(archivo)
    if err != nil {
        return fmt.Errorf("error al buscar inodo libre: %w", err)
    }

    // Escribir contenido en bloques del nuevo archivo
    err = nuevoInodo.EscribirDatos(archivo, sb, []byte(contenido))
    if err != nil {
        return fmt.Errorf("error al escribir datos del archivo: %w", err)
    }

    // Guardar el nuevo inodo
    offsetInodo := int64(sb.S_inode_start + nuevoIndiceInodo*sb.S_inode_size)
    err = nuevoInodo.Codificar(archivo, offsetInodo)
    if err != nil {
        return fmt.Errorf("error al guardar el nuevo inodo: %w", err)
    }

    // Actualizar bitmap de inodos
    err = sb.ActualizarBitmapInodo(archivo, nuevoIndiceInodo, true)
    if err != nil {
        return fmt.Errorf("error al actualizar bitmap de inodos: %w", err)
    }

    // Agregar entrada al directorio destino
    err = agregarEntradaDirectorio(archivo, sb, indiceInodoDestino, nombreArchivo, nuevoIndiceInodo)
    if err != nil {
        return fmt.Errorf("error al agregar entrada al directorio destino: %w", err)
    }

    return nil
}

// copiarDirectorio copia un directorio y todo su contenido recursivamente
func copiarDirectorio(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodoOrigen int32, indiceInodoDestino int32, nombreDirectorio string, bufferSalida *bytes.Buffer) error {
    fmt.Fprintf(bufferSalida, "Copiando directorio: %s\n", nombreDirectorio)

    // Crear nuevo inodo para el directorio destino usando NuevoInodoVacio
    nuevoInodo := Estructuras.NuevoInodoVacio()
    nuevoInodo.I_type[0] = '0' // Tipo directorio
    nuevoInodo.I_size = 0
    copy(nuevoInodo.I_perm[:], "664")

    // Buscar inodo libre para el nuevo directorio
    nuevoIndiceInodo, err := sb.BuscarSiguienteInodoLibre(archivo)
    if err != nil {
        return fmt.Errorf("error al buscar inodo libre: %w", err)
    }

    // Crear bloque inicial para el directorio con entradas . y ..
    err = crearBloqueDirectorioInicial(archivo, sb, nuevoIndiceInodo, indiceInodoDestino)
    if err != nil {
        return fmt.Errorf("error al crear bloque inicial del directorio: %w", err)
    }

    // Guardar el nuevo inodo
    offsetInodo := int64(sb.S_inode_start + nuevoIndiceInodo*sb.S_inode_size)
    err = nuevoInodo.Codificar(archivo, offsetInodo)
    if err != nil {
        return fmt.Errorf("error al guardar el nuevo inodo: %w", err)
    }

    // Actualizar bitmap de inodos
    err = sb.ActualizarBitmapInodo(archivo, nuevoIndiceInodo, true)
    if err != nil {
        return fmt.Errorf("error al actualizar bitmap de inodos: %w", err)
    }

    // Agregar entrada al directorio destino padre
    err = agregarEntradaDirectorio(archivo, sb, indiceInodoDestino, nombreDirectorio, nuevoIndiceInodo)
    if err != nil {
        return fmt.Errorf("error al agregar entrada al directorio destino: %w", err)
    }

    // Copiar contenido del directorio origen recursivamente
    err = copiarContenidoDirectorio(archivo, sb, indiceInodoOrigen, nuevoIndiceInodo, bufferSalida)
    if err != nil {
        return fmt.Errorf("error al copiar contenido del directorio: %w", err)
    }

    return nil
}

// copiarContenidoDirectorio copia recursivamente el contenido de un directorio
func copiarContenidoDirectorio(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodoOrigen int32, indiceInodoDestino int32, bufferSalida *bytes.Buffer) error {
    // Cargar inodo del directorio origen
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodoOrigen*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al cargar inodo origen: %w", err)
    }

    // Recorrer todos los bloques del directorio origen
    for _, indiceBloques := range inodo.I_block {
        if indiceBloques == -1 {
            break
        }

        // Cargar bloque del directorio
        bloque := &Estructuras.FolderBlock{}
        err := bloque.Decodificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
        if err != nil {
            return fmt.Errorf("error al cargar bloque del directorio: %w", err)
        }

        // Procesar cada entrada del bloque (saltar . y ..)
        for i, entrada := range bloque.B_cont {
            if i < 2 || entrada.B_inodo == -1 {
                continue // Saltar entradas . y .. y entradas vacías
            }

            nombreEntrada := strings.Trim(string(entrada.B_name[:]), "\x00 ")
            
            // Verificar permisos de lectura en la entrada
            if !verificarPermisosLectura(archivo, sb, entrada.B_inodo) {
                fmt.Fprintf(bufferSalida, "Saltando '%s' - sin permisos de lectura\n", nombreEntrada)
                continue
            }

            // Determinar si es archivo o directorio
            _, esDirectorio, err := determinarTipoElemento(archivo, sb, entrada.B_inodo)
            if err != nil {
                fmt.Fprintf(bufferSalida, "Error al determinar tipo de '%s': %v\n", nombreEntrada, err)
                continue
            }

            // Copiar elemento recursivamente
            if esDirectorio {
                err = copiarDirectorio(archivo, sb, entrada.B_inodo, indiceInodoDestino, nombreEntrada, bufferSalida)
            } else {
                err = copiarArchivo(archivo, sb, entrada.B_inodo, indiceInodoDestino, nombreEntrada, bufferSalida)
            }

            if err != nil {
                fmt.Fprintf(bufferSalida, "Error al copiar '%s': %v\n", nombreEntrada, err)
                continue
            }
        }
    }

    return nil
}

// determinarTipoElemento determina si un inodo es archivo o directorio
func determinarTipoElemento(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32) (string, bool, error) {
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return "", false, err
    }

    esDirectorio := inodo.I_type[0] == '0'
    if esDirectorio {
        return "directorio", true, nil
    }
    return "archivo", false, nil
}

// crearBloqueDirectorioInicial crea el bloque inicial de un directorio con entradas . y ..
func crearBloqueDirectorioInicial(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodoActual int32, indiceInodoPadre int32) error {
    // Buscar bloque libre
    indiceBloque, err := sb.BuscarSiguienteBloqueLibre(archivo)
    if err != nil {
        return fmt.Errorf("error al buscar bloque libre: %w", err)
    }

    // Crear bloque de directorio
    bloque := &Estructuras.FolderBlock{}
    
    // Entrada . (directorio actual)
    copy(bloque.B_cont[0].B_name[:], ".")
    bloque.B_cont[0].B_inodo = indiceInodoActual

    // Entrada .. (directorio padre)
    copy(bloque.B_cont[1].B_name[:], "..")
    bloque.B_cont[1].B_inodo = indiceInodoPadre

    // Inicializar entradas restantes como vacías
    for i := 2; i < len(bloque.B_cont); i++ {
        bloque.B_cont[i].B_inodo = -1
    }

    // Guardar bloque
    offsetBloque := int64(sb.S_block_start + indiceBloque*sb.S_block_size)
    err = bloque.Codificar(archivo, offsetBloque)
    if err != nil {
        return fmt.Errorf("error al guardar bloque de directorio: %w", err)
    }

    // Actualizar bitmap de bloques
    err = sb.ActualizarBitmapBloque(archivo, indiceBloque, true)
    if err != nil {
        return fmt.Errorf("error al actualizar bitmap de bloques: %w", err)
    }

    // Asignar bloque al inodo
    inodo := &Estructuras.INodo{}
    offsetInodo := int64(sb.S_inode_start + indiceInodoActual*sb.S_inode_size)
    err = inodo.Decodificar(archivo, offsetInodo)
    if err != nil {
        return fmt.Errorf("error al cargar inodo: %w", err)
    }

    inodo.I_block[0] = indiceBloque
    err = inodo.Codificar(archivo, offsetInodo)
    if err != nil {
        return fmt.Errorf("error al actualizar inodo: %w", err)
    }

    return nil
}

// agregarEntradaDirectorio agrega una nueva entrada a un directorio
func agregarEntradaDirectorio(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodoDirectorio int32, nombreEntrada string, indiceInodoEntrada int32) error {
    // Cargar inodo del directorio
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodoDirectorio*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al cargar inodo del directorio: %w", err)
    }

    // Buscar espacio libre en los bloques existentes
    for _, indiceBloques := range inodo.I_block {
        if indiceBloques == -1 {
            break
        }

        // Cargar bloque
        bloque := &Estructuras.FolderBlock{}
        err := bloque.Decodificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
        if err != nil {
            continue
        }

        // Buscar entrada libre
        for i := range bloque.B_cont {
            if bloque.B_cont[i].B_inodo == -1 {
                // Entrada libre encontrada
                copy(bloque.B_cont[i].B_name[:], nombreEntrada)
                bloque.B_cont[i].B_inodo = indiceInodoEntrada

                // Guardar bloque modificado
                err = bloque.Codificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
                if err != nil {
                    return fmt.Errorf("error al guardar bloque modificado: %w", err)
                }

                return nil
            }
        }
    }

    // Si llegamos aquí, necesitamos un nuevo bloque
    return fmt.Errorf("no hay espacio libre en el directorio para agregar la entrada")
}