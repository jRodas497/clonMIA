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

// MOVE estructura del comando move con parámetros
type MOVE struct {
    path    string // Ruta del archivo o carpeta que se desea mover
    destino string // Ruta destino donde se va a mover el contenido
}

// ParserMove parsea el comando move y devuelve una instancia de MOVE
func ParserMove(tokens []string) (string, error) {
    cmd := &MOVE{}               // Crea una nueva instancia de MOVE
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

    // Ejecutar el comando MOVE
    err := comandoMove(cmd, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

// comandoMove ejecuta la lógica principal del comando move
func comandoMove(comandoMove *MOVE, bufferSalida *bytes.Buffer) error {
    fmt.Fprint(bufferSalida, "======================== MOVE ========================\n")

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

    fmt.Fprintf(bufferSalida, "Moviendo desde: %s\n", comandoMove.path)
    fmt.Fprintf(bufferSalida, "Hacia destino: %s\n", comandoMove.destino)

    // Verificar que la ruta origen existe y obtener información
    directoriosPadreOrigen, nombreOrigen := Utils.ObtenerDirectoriosPadre(comandoMove.path)
    indiceInodoOrigen, err := buscarElementoParaMove(archivo, superBloqueParticion, directoriosPadreOrigen, nombreOrigen)
    if err != nil {
        return fmt.Errorf("error: la ruta origen '%s' no existe: %w", comandoMove.path, err)
    }

    // Verificar permisos de escritura en el elemento origen
    if !verificarPermisosEscrituraMove(archivo, superBloqueParticion, indiceInodoOrigen) {
        return fmt.Errorf("error: no tiene permisos de escritura sobre '%s'", comandoMove.path)
    }

    // Obtener el inodo del directorio padre origen
    indiceInodoPadreOrigen, err := buscarInodoCarpeta(archivo, superBloqueParticion, directoriosPadreOrigen)
    if err != nil {
        return fmt.Errorf("error al encontrar directorio padre origen: %w", err)
    }

    // Verificar que el directorio destino existe
    directoriosPadreDestino, _ := Utils.ObtenerDirectoriosPadre(comandoMove.destino)
    indiceInodoDestino, err := buscarInodoCarpeta(archivo, superBloqueParticion, directoriosPadreDestino)
    if err != nil {
        return fmt.Errorf("error: el directorio destino '%s' no existe: %w", comandoMove.destino, err)
    }

    // Verificar permisos de escritura en el directorio destino
    if !verificarPermisosEscrituraMove(archivo, superBloqueParticion, indiceInodoDestino) {
        return fmt.Errorf("error: no tiene permisos de escritura sobre el directorio destino '%s'", comandoMove.destino)
    }

    // Verificar que no exista un elemento con el mismo nombre en el destino
    encontrado, _, err := directorioExiste(superBloqueParticion, archivo, indiceInodoDestino, nombreOrigen)
    if err != nil {
        return fmt.Errorf("error al verificar destino: %w", err)
    }
    if encontrado {
        return fmt.Errorf("ya existe un elemento con el nombre '%s' en el directorio destino", nombreOrigen)
    }

    // Realizar el movimiento (cambiar referencias)
    err = moverElemento(archivo, superBloqueParticion, indiceInodoPadreOrigen, indiceInodoDestino, nombreOrigen, indiceInodoOrigen, bufferSalida)
    if err != nil {
        return fmt.Errorf("error durante el movimiento: %w", err)
    }

    fmt.Fprintf(bufferSalida, "Movimiento completado exitosamente\n")
    fmt.Fprint(bufferSalida, "=====================================================\n")

    return nil
}

// buscarElementoParaMove busca un archivo o directorio y retorna su inodo
func buscarElementoParaMove(archivo *os.File, sb *Estructuras.SuperBlock, directoriosPadre []string, nombreElemento string) (int32, error) {
    // Buscar el directorio padre
    indiceInodoPadre, err := buscarInodoCarpeta(archivo, sb, directoriosPadre)
    if err != nil {
        return -1, err
    }

    // Buscar el elemento en el directorio padre
    encontrado, indiceInodoElemento, err := directorioExiste(sb, archivo, indiceInodoPadre, nombreElemento)
    if err != nil {
        return -1, err
    }
    if !encontrado {
        return -1, fmt.Errorf("elemento '%s' no encontrado", nombreElemento)
    }

    return indiceInodoElemento, nil
}

// verificarPermisosEscrituraMove verifica si el usuario actual tiene permisos de escritura
func verificarPermisosEscrituraMove(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32) bool {
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return false
    }

    // Verificar permisos según el owner y group
    // Para este ejemplo, asumimos que el permiso '6' y '2' incluyen escritura
    permisos := string(inodo.I_perm[:])
    return strings.Contains(permisos, "6") || strings.Contains(permisos, "2")
}

// moverElemento realiza el movimiento cambiando las referencias entre directorios
func moverElemento(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodoPadreOrigen int32, indiceInodoDestino int32, nombreElemento string, indiceInodoElemento int32, bufferSalida *bytes.Buffer) error {
    fmt.Fprintf(bufferSalida, "Moviendo elemento: %s\n", nombreElemento)

    // Paso 1: Eliminar entrada del directorio origen
    err := eliminarEntradaDirectorio(archivo, sb, indiceInodoPadreOrigen, nombreElemento)
    if err != nil {
        return fmt.Errorf("error al eliminar entrada del directorio origen: %w", err)
    }

    // Paso 2: Agregar entrada al directorio destino
    err = agregarEntradaDirectorioMove(archivo, sb, indiceInodoDestino, nombreElemento, indiceInodoElemento)
    if err != nil {
        // Si falla agregar al destino, intentar restaurar en origen
        _ = agregarEntradaDirectorioMove(archivo, sb, indiceInodoPadreOrigen, nombreElemento, indiceInodoElemento)
        return fmt.Errorf("error al agregar entrada al directorio destino: %w", err)
    }

    // Paso 3: Si es un directorio, actualizar entrada ".." para que apunte al nuevo padre
    if esDirectorio(archivo, sb, indiceInodoElemento) {
        err = actualizarEntradaPadreDotDot(archivo, sb, indiceInodoElemento, indiceInodoDestino)
        if err != nil {
            fmt.Fprintf(bufferSalida, "Advertencia: error al actualizar entrada padre: %v\n", err)
        }
    }

    return nil
}

// eliminarEntradaDirectorio elimina una entrada específica de un directorio
func eliminarEntradaDirectorio(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodoDirectorio int32, nombreEntrada string) error {
    // Cargar inodo del directorio
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodoDirectorio*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al cargar inodo del directorio: %w", err)
    }

    // Buscar la entrada en los bloques del directorio
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

        // Buscar la entrada a eliminar
        for i := range bloque.B_cont {
            nombreContenido := strings.Trim(string(bloque.B_cont[i].B_name[:]), "\x00 ")
            if strings.EqualFold(nombreContenido, nombreEntrada) && bloque.B_cont[i].B_inodo != -1 {
                // Eliminar entrada (marcar como libre)
                bloque.B_cont[i].B_inodo = -1
                for j := range bloque.B_cont[i].B_name {
                    bloque.B_cont[i].B_name[j] = 0
                }

                // Guardar bloque modificado
                err = bloque.Codificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
                if err != nil {
                    return fmt.Errorf("error al guardar bloque modificado: %w", err)
                }

                return nil
            }
        }
    }

    return fmt.Errorf("entrada '%s' no encontrada para eliminar", nombreEntrada)
}

// agregarEntradaDirectorioMove agrega una nueva entrada a un directorio (específica para move)
func agregarEntradaDirectorioMove(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodoDirectorio int32, nombreEntrada string, indiceInodoEntrada int32) error {
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

    // Si llegamos aquí, necesitamos un nuevo bloque (simplificado: error)
    return fmt.Errorf("no hay espacio libre en el directorio destino")
}

// esDirectorio verifica si un inodo corresponde a un directorio
func esDirectorio(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32) bool {
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return false
    }

    return inodo.I_type[0] == '0'
}

// actualizarEntradaPadreDotDot actualiza la entrada ".." de un directorio para que apunte al nuevo padre
func actualizarEntradaPadreDotDot(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodoDirectorio int32, nuevoIndiceInodoPadre int32) error {
    // Cargar inodo del directorio
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodoDirectorio*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al cargar inodo del directorio: %w", err)
    }

    // Buscar en el primer bloque (donde debería estar "..")
    if inodo.I_block[0] != -1 {
        // Cargar primer bloque
        bloque := &Estructuras.FolderBlock{}
        err := bloque.Decodificar(archivo, int64(sb.S_block_start+(inodo.I_block[0]*sb.S_block_size)))
        if err != nil {
            return fmt.Errorf("error al cargar primer bloque: %w", err)
        }

        // Actualizar entrada ".." (debería estar en índice 1)
        if len(bloque.B_cont) > 1 {
            nombreSegundaEntrada := strings.Trim(string(bloque.B_cont[1].B_name[:]), "\x00 ")
            if nombreSegundaEntrada == ".." {
                bloque.B_cont[1].B_inodo = nuevoIndiceInodoPadre

                // Guardar bloque modificado
                err = bloque.Codificar(archivo, int64(sb.S_block_start+(inodo.I_block[0]*sb.S_block_size)))
                if err != nil {
                    return fmt.Errorf("error al guardar bloque con entrada padre actualizada: %w", err)
                }

                return nil
            }
        }
    }

    return fmt.Errorf("no se pudo encontrar la entrada '..' para actualizar")
}