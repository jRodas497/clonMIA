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

// RENAME estructura del comando rename con parámetros
type RENAME struct {
    ruta   string // Ruta del archivo
    nombre string // Nuevo nombre del archivo
}

// ParserRename parsea el comando rename y devuelve una instancia de RENAME
func ParserRename(tokens []string) (string, error) {
    cmd := &RENAME{}               // Crea una nueva instancia de RENAME
    var bufferSalida bytes.Buffer  // Buffer para capturar mensajes importantes

    // Expresión regular para capturar los parámetros -path y -name
    re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-name="[^"]+"|-name=[^\s]+`)
    matches := re.FindAllString(strings.Join(tokens, " "), -1)

    // Verificar que se han proporcionado ambos parámetros
    if len(matches) != len(tokens) || len(matches) < 2 {
        return "", errors.New("faltan parámetros requeridos: -path o -name")
    }

    // Iterar sobre cada coincidencia y extraer los valores de -path y -name
    for _, match := range matches {
        kv := strings.SplitN(match, "=", 2)
        key := strings.ToLower(kv[0])
        value := strings.Trim(kv[1], "\"") // Eliminar comillas si existen

        // Asignar los valores de los parámetros
        switch key {
        case "-path":
            cmd.ruta = value
        case "-name":
            cmd.nombre = value
        }
    }

    // Verificar que ambos parámetros tengan valores
    if cmd.ruta == "" || cmd.nombre == "" {
        return "", errors.New("los parámetros -path y -name son obligatorios")
    }

    // Ejecutar el comando RENAME
    err := comandoRename(cmd, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

// comandoRename ejecuta la lógica principal del comando rename
func comandoRename(comandoRename *RENAME, bufferSalida *bytes.Buffer) error {
    fmt.Fprint(bufferSalida, "======================= RENAME =======================\n")

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

    // Desglosar el path en directorios y el archivo/carpeta a renombrar
    directoriosPadre, nombreAntiguo := Utils.ObtenerDirectoriosPadre(comandoRename.ruta)

    // Buscar el inodo del directorio donde está el archivo/carpeta usando las funciones de cat.go
    // Cargar el FolderBlock del directorio padre (asumir primer bloque como en el original)
    indiceInodo, err := buscarInodoCarpeta(archivo, superBloqueParticion, directoriosPadre)
    if err != nil {
        return fmt.Errorf("error al encontrar el directorio padre: %v", err)
    }

    // Cargar el FolderBlock del directorio padre (usando solo el primer bloque como en el original)
    bloqueCarpeta := &Estructuras.FolderBlock{}
    err = bloqueCarpeta.Decodificar(archivo, int64(superBloqueParticion.S_block_start+(indiceInodo*superBloqueParticion.S_block_size)))
    if err != nil {
        return fmt.Errorf("error al deserializar el bloque de carpeta: %v", err)
    }

    // Verificar que no exista un archivo/carpeta con el nuevo nombre
    for _, contenido := range bloqueCarpeta.B_cont {
        if strings.EqualFold(strings.Trim(string(contenido.B_name[:]), "\x00 "), comandoRename.nombre) {
            return fmt.Errorf("ya existe un archivo o carpeta con el nombre '%s'", comandoRename.nombre)
        }
    }

    // Renombrar el archivo/carpeta: buscar entrada y reemplazar nombre
    renombrado := false
    for i, contenido := range bloqueCarpeta.B_cont {
        nombre := strings.Trim(string(contenido.B_name[:]), "\x00 ")
        if strings.EqualFold(nombre, nombreAntiguo) {
            // Reemplazar el nombre en la entrada
            copy(bloqueCarpeta.B_cont[i].B_name[:], comandoRename.nombre)
            renombrado = true
            break
        }
    }
    if !renombrado {
        return fmt.Errorf("archivo o carpeta '%s' no encontrado para renombrar", nombreAntiguo)
    }

    // Guardar el bloque modificado de nuevo en el archivo
    err = bloqueCarpeta.Codificar(archivo, int64(superBloqueParticion.S_block_start+(indiceInodo*superBloqueParticion.S_block_size)))
    if err != nil {
        return fmt.Errorf("error al guardar el bloque de carpeta modificado: %v", err)
    }

    fmt.Fprintf(bufferSalida, "Nombre cambiado exitosamente de '%s' a '%s'\n", nombreAntiguo, comandoRename.nombre)
    fmt.Fprint(bufferSalida, "=====================================================\n")

    return nil
}
