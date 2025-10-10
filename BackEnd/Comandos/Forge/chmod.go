package Forge

import (
    "bytes"
    "errors"
    "fmt"
    "os"
    "regexp"
    "strconv"
    "strings"

    Estructuras "backend/Estructuras"
    Global "backend/Global"
    Utils "backend/Utils"
)

// CHMOD estructura del comando chmod con parámetros
type CHMOD struct {
    path      string // Ruta del archivo o carpeta a modificar permisos
    ugo       string // Permisos en formato UGO (ej: "764")
    recursivo bool   // Indicador para aplicar cambios de forma recursiva
}

// ParserChmod procesa el comando chmod y retorna una instancia de CHMOD
func ParserChmod(tokens []string) (string, error) {
    cmd := &CHMOD{}
    var bufferSalida bytes.Buffer

    // Regex para extraer parámetros -path, -r y -ugo
    re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-ugo="[^"]+"|-ugo=[^\s]+|-r`)
    matches := re.FindAllString(strings.Join(tokens, " "), -1)

    // Validar que se proporcionen los parámetros mínimos necesarios
    if len(matches) < 2 {
        return "", errors.New("faltan parámetros requeridos: -path y -ugo son obligatorios")
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
        case "-ugo":
            cmd.ugo = value
        }
    }

    // Confirmar que se establecieron los parámetros obligatorios
    if cmd.path == "" || cmd.ugo == "" {
        return "", errors.New("los parámetros -path y -ugo son obligatorios")
    }

    // Procesar el comando CHMOD
    err := comandoChmod(cmd, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

// comandoChmod ejecuta la funcionalidad principal del comando chmod
func comandoChmod(comandoChmod *CHMOD, bufferSalida *bytes.Buffer) error {
    fmt.Fprint(bufferSalida, "======================= CHMOD ======================\n")

    // Confirmar que existe un usuario autenticado
    if !Global.VerificarSesionActiva() {
        return fmt.Errorf("no hay un usuario logueado")
    }

    // Verificar que solo el usuario root puede ejecutar chmod
    if Global.UsuarioActual.Nombre != "root" {
        return fmt.Errorf("solo el usuario root puede cambiar permisos")
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

    fmt.Fprintf(bufferSalida, "Cambiando permisos en: %s\n", comandoChmod.path)
    fmt.Fprintf(bufferSalida, "Nuevos permisos UGO: %s\n", comandoChmod.ugo)
    if comandoChmod.recursivo {
        fmt.Fprintf(bufferSalida, "Aplicando recursivamente\n")
    }

    // Validar formato de permisos UGO
    if !validarFormatoUGO(comandoChmod.ugo) {
        return fmt.Errorf("formato de permisos inválido: '%s'. Debe ser 3 dígitos del 0-7", comandoChmod.ugo)
    }

    // Localizar el archivo o directorio usando buscarInodoArchivo de cat.go
    directoriosPadre, nombreElemento := Utils.ObtenerDirectoriosPadre(comandoChmod.path)
    indiceInodoElemento, err := buscarInodoArchivo(archivo, superBloqueParticion, directoriosPadre, nombreElemento)
    if err != nil {
        return fmt.Errorf("error: la ruta '%s' no existe: %w", comandoChmod.path, err)
    }

    // Ejecutar cambio de permisos
    if comandoChmod.recursivo {
        err = cambiarPermisosRecursivo(archivo, superBloqueParticion, indiceInodoElemento, comandoChmod.ugo, comandoChmod.path, bufferSalida)
    } else {
        err = cambiarPermisosElemento(archivo, superBloqueParticion, indiceInodoElemento, comandoChmod.ugo, comandoChmod.path, bufferSalida)
    }

    if err != nil {
        return fmt.Errorf("error durante el cambio de permisos: %w", err)
    }

    fmt.Fprintf(bufferSalida, "Cambio de permisos completado exitosamente\n")
    fmt.Fprint(bufferSalida, "===================================================\n")

    return nil
}

// validarFormatoUGO verifica que el formato UGO sea válido (3 dígitos del 0-7)
func validarFormatoUGO(ugo string) bool {
    // Debe tener exactamente 3 caracteres
    if len(ugo) != 3 {
        return false
    }

    // Cada carácter debe ser un dígito del 0 al 7
    for _, char := range ugo {
        digit, err := strconv.Atoi(string(char))
        if err != nil || digit < 0 || digit > 7 {
            return false
        }
    }

    return true
}

// cambiarPermisosElemento modifica los permisos de un elemento específico
func cambiarPermisosElemento(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32, nuevosPermisos string, rutaElemento string, bufferSalida *bytes.Buffer) error {
    // Cargar información del inodo
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al cargar inodo: %w", err)
    }

    // Registrar permisos anteriores
    permisosAnteriores := strings.Trim(string(inodo.I_perm[:]), "\x00 ")
    fmt.Fprintf(bufferSalida, "Cambiando permisos de '%s': %s -> %s\n", rutaElemento, permisosAnteriores, nuevosPermisos)

    // Establecer nuevos permisos
    copy(inodo.I_perm[:], nuevosPermisos)

    // Almacenar cambios en el inodo
    offsetInodo := int64(sb.S_inode_start + indiceInodo*sb.S_inode_size)
    err = inodo.Codificar(archivo, offsetInodo)
    if err != nil {
        return fmt.Errorf("error al guardar cambios del inodo: %w", err)
    }

    return nil
}

// cambiarPermisosRecursivo modifica los permisos de un directorio y todo su contenido
func cambiarPermisosRecursivo(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32, nuevosPermisos string, rutaActual string, bufferSalida *bytes.Buffer) error {
    // Aplicar cambio al elemento actual
    err := cambiarPermisosElemento(archivo, sb, indiceInodo, nuevosPermisos, rutaActual, bufferSalida)
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

            // Verificar si el elemento pertenece al usuario actual antes de procesar
            if !esElementoDelUsuarioActual(archivo, sb, contenido.B_inodo) {
                fmt.Fprintf(bufferSalida, "Saltando '%s/%s' - no pertenece al usuario actual\n", rutaActual, nombreContenido)
                continue
            }

            // Aplicar cambio recursivamente
            nuevaRuta := rutaActual + "/" + nombreContenido
            err = cambiarPermisosRecursivo(archivo, sb, contenido.B_inodo, nuevosPermisos, nuevaRuta, bufferSalida)
            if err != nil {
                fmt.Fprintf(bufferSalida, "Error en '%s': %v\n", nuevaRuta, err)
                continue
            }
        }
    }

    return nil
}

// esElementoDelUsuarioActual verifica si un elemento pertenece al usuario actual
func esElementoDelUsuarioActual(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32) bool {
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return false
    }

    // Comparar el propietario del elemento con el usuario actual
    propietario := strings.Trim(string(inodo.I_uid[:]), "\x00 ")
    return propietario == Global.UsuarioActual.Nombre
}