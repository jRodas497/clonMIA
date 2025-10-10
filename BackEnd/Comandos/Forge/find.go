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

// FIND estructura del comando find con parámetros
type FIND struct {
    path string // Ruta donde iniciar la búsqueda
    name string // Patrón del archivo o carpeta a buscar
}

// ParserFind analiza el comando find y retorna una instancia de FIND
func ParserFind(tokens []string) (string, error) {
    cmd := &FIND{}
    var bufferSalida bytes.Buffer

    // Regex para extraer parámetros -path y -name
    re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-name="[^"]+"|-name=[^\s]+`)
    matches := re.FindAllString(strings.Join(tokens, " "), -1)

    // Validar que se proporcionen los parámetros necesarios
    if len(matches) != len(tokens) || len(matches) < 2 {
        return "", errors.New("faltan parámetros requeridos: -path o -name")
    }

    // Procesar cada parámetro encontrado
    for _, match := range matches {
        kv := strings.SplitN(match, "=", 2)
        key := strings.ToLower(kv[0])
        value := strings.Trim(kv[1], "\"") // Quitar comillas del valor

        // Establecer valores según el parámetro
        switch key {
        case "-path":
            cmd.path = value
        case "-name":
            cmd.name = value
        }
    }

    // Confirmar que se establecieron ambos parámetros
    if cmd.path == "" || cmd.name == "" {
        return "", errors.New("los parámetros -path y -name son obligatorios")
    }

    // Procesar el comando FIND
    err := comandoFind(cmd, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

func comandoFind(comandoFind *FIND, bufferSalida *bytes.Buffer) error {
    fmt.Fprint(bufferSalida, "======================= FIND =======================\n")

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

    // Determinar inodo de inicio según el path
    var indiceInodoRaiz int32
    if comandoFind.path == "/" {
        indiceInodoRaiz = 0 // Inodo raíz del sistema
    } else {
        // Analizar ruta y localizar el inodo correspondiente
        directoriosPadre, nombreDirectorio := Utils.ObtenerDirectoriosPadre(comandoFind.path)
        indiceInodoRaiz, err = buscarInodoArchivo(archivo, superBloqueParticion, directoriosPadre, nombreDirectorio)
        if err != nil {
            return fmt.Errorf("error al encontrar el directorio inicial: %v", err)
        }
    }

    // Transformar patrón de búsqueda a expresión regular
    patron, err := comodinARegex(comandoFind.name)
    if err != nil {
        return fmt.Errorf("error al convertir el patrón de búsqueda: %v", err)
    }

    // Ejecutar búsqueda recursiva en el sistema de archivos
    err = busquedaRecursiva(archivo, superBloqueParticion, indiceInodoRaiz, patron, comandoFind.path, bufferSalida)
    if err != nil {
        return fmt.Errorf("error durante la búsqueda: %v", err)
    }

    fmt.Fprint(bufferSalida, "=================================================\n")
    return nil
}

func busquedaRecursiva(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32, patron *regexp.Regexp, rutaActual string, bufferSalida *bytes.Buffer) error {
    // Cargar información del inodo actual
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error al deserializar el inodo %d: %v", indiceInodo, err)
    }

    // Solo procesar si es un directorio
    if inodo.I_type[0] != '0' {
        return nil // Salir si no es directorio
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
            return fmt.Errorf("error al deserializar el bloque %d: %v", indiceBloques, err)
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

            // Evaluar si el nombre cumple con el patrón
            if patron.MatchString(nombreContenido) {
                fmt.Fprintf(bufferSalida, "%s/%s\n", rutaActual, nombreContenido)
            }

            // Continuar búsqueda en subdirectorios
            nuevoIndiceInodo := contenido.B_inodo
            err = busquedaRecursiva(archivo, sb, nuevoIndiceInodo, patron, rutaActual+"/"+nombreContenido, bufferSalida)
            if err != nil {
                return err
            }
        }
    }

    return nil
}

func comodinARegex(patron string) (*regexp.Regexp, error) {
    // Convertir caracteres especiales a regex válido
    patron = strings.ReplaceAll(patron, ".", "\\.")
    patron = strings.ReplaceAll(patron, "?", ".")  // ? representa un carácter
    patron = strings.ReplaceAll(patron, "*", ".*") // * representa cualquier cantidad

    // Generar expresión regular compilada
    return regexp.Compile("^" + patron + "$")
}