package Forge

import (
	Estructuras "backend/Estructuras"
	Global "backend/Global"
	Utils "backend/Utils"
	"bytes"
	"errors"
	"regexp"
	"fmt"
	"os"
	"strings"
)


// REMOVE estructura que representa el comando REMOVE con sus parametros
type REMOVE struct {
    ruta string // Ruta del archivo o carpeta a eliminar
}

func ParserRemove(tokens []string) (string, error) {
    // Crear nueva instancia de REMOVE
	cmd := &REMOVE{}               
    // Buffer para capturar mensajes importantes
	var bufferSalida bytes.Buffer  

    // Expresion regular para capturar el parametro -path="ruta"
    expresionRegular := regexp.MustCompile(`-path=("[^"]+"|[^\s]+)`)
    coincidencias := expresionRegular.FindAllString(strings.Join(tokens, " "), -1)

    // Verificar si no se encontro la ruta
    if len(coincidencias) == 0 {
        return "", errors.New("no se especifico una ruta para eliminar")
    }

    // Extraer el valor de la ruta
    valorClave := strings.SplitN(coincidencias[0], "=", 2)
    if len(valorClave) == 2 {
        cmd.ruta = valorClave[1]
        // Ruta esta entre comillas eliminarlas
        if strings.HasPrefix(cmd.ruta, "\"") && strings.HasSuffix(cmd.ruta, "\"") {
            cmd.ruta = strings.Trim(cmd.ruta, "\"")
        }
    }

    // Ejecutar el comando REMOVE
    err := comandoRemove(cmd, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

func comandoRemove(cmdRemover *REMOVE, bufferSalida *bytes.Buffer) error {
    fmt.Fprint(bufferSalida, "====================== REMOVE ======================\n")

    // Verificar si hay un usuario logueado
    if !Global.VerificarSesionActiva() {
        return fmt.Errorf("no hay un usuario logueado")
    }

    // Obtener la particion montada asociada al usuario logueado
    idParticion := Global.UsuarioActual.Id
    superBloqueParticion, particionMontada, rutaParticion, err := Global.GetMountedPartitionSuperblock(idParticion)
    if err != nil {
        return fmt.Errorf("error al obtener la particion montada: %w", err)
    }

    // Abrir el archivo de particion para operar sobre el
    archivo, err := os.OpenFile(rutaParticion, os.O_RDWR, 0666)
    if err != nil {
        return fmt.Errorf("error al abrir el archivo de particion: %w", err)
    }
    defer archivo.Close() // Cerrar el archivo cuando ya no sea necesario

    // Llamar a la funcion refactorizada para eliminar archivo/carpeta
    err = eliminarArchivoOCarpeta(cmdRemover.ruta, superBloqueParticion, archivo)
    if err != nil {
        return fmt.Errorf("error al eliminar archivo o carpeta: %v", err)
    }

    // Serializar el superbloque para guardar los cambios
    err = superBloqueParticion.Codificar(archivo, int64(particionMontada.Part_start))
    if err != nil {
        return fmt.Errorf("error al serializar el superbloque despues de la eliminacion: %v", err)
    }

    fmt.Fprintf(bufferSalida, "Archivo o carpeta '%s' eliminado exitosamente.\n", cmdRemover.ruta)
    fmt.Fprint(bufferSalida, "====================================================\n")
    return nil
}

// eliminarArchivoOCarpeta elimina un archivo o carpeta dada la ruta
func eliminarArchivoOCarpeta(rutaCompleta string, sb *Estructuras.SuperBlock, archivo *os.File) error {
    // Convertir el path del archivo o carpeta en un arreglo de carpetas
    directoriosPadre, nombreArchivo := Utils.ObtenerDirectoriosPadre(rutaCompleta)

    // Intentar eliminar el archivo
    err := eliminarArchivo(sb, archivo, directoriosPadre, nombreArchivo)
    if err == nil {
        // Si el archivo se elimino correctamente, regresar
        return nil
    }

    // Si no es un archivo, intentar eliminarlo como carpeta
    err = eliminarDirectorio(sb, archivo, directoriosPadre, nombreArchivo)
    if err != nil {
        return fmt.Errorf("error al eliminar archivo o carpeta '%s': %v", rutaCompleta, err)
    }

    return nil
}

// eliminarArchivo intenta eliminar un archivo dado su path
func eliminarArchivo(sb *Estructuras.SuperBlock, archivo *os.File, directoriosPadre []string, nombreArchivo string) error {
    // Buscar el inodo del archivo
    _, err := buscarInodoArchivo(archivo, sb, directoriosPadre, nombreArchivo)
    if err != nil {
        // No se encontro el archivo
        return fmt.Errorf("archivo '%s' no encontrado: %v", nombreArchivo, err)
    }

    // Llamar a la funcion que elimina el archivo
    err = sb.EliminarArchivo(archivo, directoriosPadre, nombreArchivo)
    if err != nil {
        return fmt.Errorf("error al eliminar el archivo '%s': %v", nombreArchivo, err)
    }

    fmt.Printf("Archivo '%s' eliminado correctamente.\n", nombreArchivo)
    return nil
}

// eliminarDirectorio intenta eliminar una carpeta dada su path
func eliminarDirectorio(sb *Estructuras.SuperBlock, archivo *os.File, directoriosPadre []string, nombreDirectorio string) error {
	rutaCarpetaCompleta := append(directoriosPadre, nombreDirectorio)

	// Buscar el inodo de la carpeta
    _, err := buscarInodoCarpeta(archivo, sb, rutaCarpetaCompleta)
    if err != nil {
        // No se encontro la carpeta
        return fmt.Errorf("carpeta '%s' no encontrada: %v", nombreDirectorio, err)
    }

    // Llamar a la funcion que elimina la carpeta
    err = sb.EliminarCarpeta(archivo, directoriosPadre, nombreDirectorio)
    if err != nil {
        return fmt.Errorf("error al eliminar la carpeta '%s': %v", nombreDirectorio, err)
    }

    fmt.Printf("Carpeta '%s' eliminada correctamente.\n", nombreDirectorio)
    return nil
}

