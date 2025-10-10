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

// CHOWN estructura del comando chown con parámetros
type CHOWN struct {
    path     string // Ruta del archivo o carpeta a modificar
    recursivo bool  // Indicador para aplicar cambios de forma recursiva
    usuario  string // Nombre del nuevo propietario
}

// ParserChown procesa el comando chown y retorna una instancia de CHOWN
func ParserChown(tokens []string) (string, error) {
    cmd := &CHOWN{}
    var bufferSalida bytes.Buffer

    // Regex para extraer parámetros -path, -r y -usuario
    re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-usuario="[^"]+"|-usuario=[^\s]+|-r`)
    matches := re.FindAllString(strings.Join(tokens, " "), -1)

    // Validar que se proporcionen los parámetros mínimos necesarios
    if len(matches) < 2 {
        return "", errors.New("faltan parámetros requeridos: -path y -usuario son obligatorios")
    }

    // Procesar cada parámetro encontrado
    for _, match := range matches {
        if match == "-r" {
            cmd.recursivo = true
            continue
        }

        kv := strings.SplitN(match, "=", 2)
        if len(kv) != 2 {
            continue
        }
        key := strings.ToLower(kv[0])
        value := strings.Trim(kv[1], "\"") // Quitar comillas del valor

        // Establecer valores según el parámetro
        switch key {
        case "-path":
            cmd.path = value
        case "-usuario":
            cmd.usuario = value
        }
    }

    // Confirmar que se establecieron los parámetros obligatorios
    if cmd.path == "" || cmd.usuario == "" {
        return "", errors.New("los parámetros -path y -usuario son obligatorios")
    }

    // Procesar el comando CHOWN
    err := comandoChown(cmd, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

// comandoChown ejecuta la funcionalidad principal del comando chown
func comandoChown(comandoChown *CHOWN, bufferSalida *bytes.Buffer) error {
    fmt.Fprint(bufferSalida, "======================= CHOWN ======================\n")

    // Confirmar que existe un usuario autenticado
    if !Global.VerificarSesionActiva() {
        return fmt.Errorf("no hay un usuario logueado")
    }

    // Extraer ID de partición del usuario actual
    idParticion := Global.UsuarioActual.Id

    // Recuperar información de la partición montada
    superBloqueParticion, _, rutaParticion, err := Global.ObtenerSuperblockParticionMontada(idParticion)
    if err != nil {
        return fmt.Errorf("error al obtener la partición montada: %w", err)
    }

    // Acceder al archivo de la partición
    archivo, err := os.OpenFile(rutaParticion, os.O_RDWR, 0666)
    if err != nil {
        return fmt.Errorf("error al abrir el archivo de partición: %w", err)
    }
    defer archivo.Close() // Liberar recurso al finalizar

    fmt.Fprintf(bufferSalida, "Cambiando propietario en: %s\n", comandoChown.path)
    fmt.Fprintf(bufferSalida, "Nuevo propietario: %s\n", comandoChown.usuario)
    if comandoChown.recursivo {
        fmt.Fprintf(bufferSalida, "Aplicando recursivamente\n")
    }

    // Validar que el nuevo usuario existe usando funciones existentes de cat.go
    usuarioExiste, err := validarUsuarioExisteOptimizado(archivo, superBloqueParticion, comandoChown.usuario)
    if err != nil {
        return fmt.Errorf("error al verificar usuario: %w", err)
    }
    if !usuarioExiste {
        return fmt.Errorf("el usuario '%s' no existe en el sistema", comandoChown.usuario)
    }

    // Localizar el archivo o directorio usando buscarInodoArchivo de cat.go
    directoriosPadre, nombreElemento := Utils.ObtenerDirectoriosPadre(comandoChown.path)
    indiceInodoElemento, err := buscarInodoArchivo(archivo, superBloqueParticion, directoriosPadre, nombreElemento)
    if err != nil {
        return fmt.Errorf("error: la ruta '%s' no existe: %w", comandoChown.path, err)
    }

    // Validar permisos para realizar el cambio de propietario
    if !validarPermisosChown(archivo, superBloqueParticion, indiceInodoElemento) {
        return fmt.Errorf("error: no tiene permisos para cambiar el propietario de '%s'", comandoChown.path)
    }

    // Ejecutar cambio de propietario
    if comandoChown.recursivo {
        err = cambiarPropietarioRecursivo(archivo, superBloqueParticion, indiceInodoElemento, comandoChown.usuario, comandoChown.path, bufferSalida)
    } else {
        err = cambiarPropietarioElemento(archivo, superBloqueParticion, indiceInodoElemento, comandoChown.usuario, comandoChown.path, bufferSalida)
    }

    if err != nil {
        return fmt.Errorf("error durante el cambio de propietario: %w", err)
    }

    fmt.Fprintf(bufferSalida, "Cambio de propietario completado exitosamente\n")
    fmt.Fprint(bufferSalida, "===================================================\n")

    return nil
}

// validarUsuarioExisteOptimizado usa funciones existentes de cat.go
func validarUsuarioExisteOptimizado(archivo *os.File, sb *Estructuras.SuperBlock, nombreUsuario string) (bool, error) {
    // Usar directorioExiste de cat.go para encontrar users.txt
    encontrado, indiceInodoUsers, err := directorioExiste(sb, archivo, 0, "users.txt")
    if err != nil {
        return false, fmt.Errorf("error al buscar users.txt: %w", err)
    }
    if !encontrado {
        return false, fmt.Errorf("archivo users.txt no encontrado")
    }

    // Usar leerArchivoDesdeInodo de cat.go para obtener contenido
    contenidoUsers, err := leerArchivoDesdeInodo(archivo, sb, indiceInodoUsers)
    if err != nil {
        return false, fmt.Errorf("error al leer users.txt: %w", err)
    }

    // Examinar cada línea buscando el usuario
    lineas := strings.Split(contenidoUsers, "\n")
    for _, linea := range lineas {
        if strings.TrimSpace(linea) == "" {
            continue
        }

        campos := strings.Split(linea, ",")
        if len(campos) >= 4 && strings.TrimSpace(campos[3]) == nombreUsuario {
            return true, nil
        }
    }

    return false, nil
}

// validarPermisosChown verifica si el usuario actual puede cambiar el propietario
func validarPermisosChown(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32) bool {
    // El usuario root puede cambiar cualquier propietario
    if Global.UsuarioActual.Nombre == "root" {
        return true
    }

    // Otros usuarios solo pueden cambiar sus propios archivos
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return false
    }

    // Comparar el propietario actual con el usuario logueado
    propietarioActual := strings.Trim(string(inodo.I_uid[:]), "\x00 ")
    return propietarioActual == Global.UsuarioActual.Nombre
}

// cambiarPropietarioElemento modifica el propietario de un elemento específico
func cambiarPropietarioElemento(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32, nuevoPropietario string, rutaElemento string, bufferSalida *bytes.Buffer) error {
    // Cargar información del inodo
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al cargar inodo: %w", err)
    }

    // Registrar propietario anterior
    propietarioAnterior := strings.Trim(string(inodo.I_uid[:]), "\x00 ")
    fmt.Fprintf(bufferSalida, "Cambiando propietario de '%s': %s -> %s\n", rutaElemento, propietarioAnterior, nuevoPropietario)

    // Establecer nuevo propietario
    copy(inodo.I_uid[:], nuevoPropietario)

    // Almacenar cambios en el inodo
    offsetInodo := int64(sb.S_inode_start + indiceInodo*sb.S_inode_size)
    err = inodo.Codificar(archivo, offsetInodo)
    if err != nil {
        return fmt.Errorf("error al guardar cambios del inodo: %w", err)
    }

    return nil
}

// cambiarPropietarioRecursivo modifica el propietario de un directorio y todo su contenido
func cambiarPropietarioRecursivo(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32, nuevoPropietario string, rutaActual string, bufferSalida *bytes.Buffer) error {
    // Aplicar cambio al elemento actual
    err := cambiarPropietarioElemento(archivo, sb, indiceInodo, nuevoPropietario, rutaActual, bufferSalida)
    if err != nil {
        return err
    }

    // Verificar si es un directorio usando lógica inline
    inodo := &Estructuras.INodo{}
    err = inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al cargar inodo del directorio: %w", err)
    }

    // Si no es directorio, terminar aquí
    if inodo.I_type[0] != '0' {
        return nil
    }

    // Recorrer todos los bloques del directorio
    for _, indiceBloques := range inodo.I_block {
        if indiceBloques == -1 {
            break // No hay más bloques asignados
        }

        // Cargar contenido del bloque de directorio
        bloque := &Estructuras.FolderBlock{}
        err := bloque.Decodificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
        if err != nil {
            return fmt.Errorf("error al cargar bloque del directorio: %w", err)
        }

        // Examinar cada entrada del bloque
        for _, contenido := range bloque.B_cont {
            if contenido.B_inodo == -1 {
                continue // Saltar entradas vacías
            }

            nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ")

            // Omitir referencias de directorio especiales
            if nombreContenido == "." || nombreContenido == ".." {
                continue
            }

            // Validar permisos de lectura antes de procesar
            if !validarPermisosLecturaChown(archivo, sb, contenido.B_inodo) {
                fmt.Fprintf(bufferSalida, "Saltando '%s/%s' - sin permisos de lectura\n", rutaActual, nombreContenido)
                continue
            }

            // Aplicar cambio recursivamente
            nuevaRuta := rutaActual + "/" + nombreContenido
            err = cambiarPropietarioRecursivo(archivo, sb, contenido.B_inodo, nuevoPropietario, nuevaRuta, bufferSalida)
            if err != nil {
                fmt.Fprintf(bufferSalida, "Error en '%s': %v\n", nuevaRuta, err)
                continue
            }
        }
    }

    return nil
}

// validarPermisosLecturaChown verifica si se tienen permisos de lectura sobre un elemento
func validarPermisosLecturaChown(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32) bool {
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return false
    }

    // Evaluar permisos según el owner y group
    permisos := string(inodo.I_perm[:])
    return strings.Contains(permisos, "6") || strings.Contains(permisos, "4")
}