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
)

// RECOVERY estructura del comando recovery con parámetros
type RECOVERY struct {
    id string // Identificador de la partición montada donde ejecutar recuperación
}

// ParserRecovery analiza los argumentos del comando recovery y ejecuta la recuperación
func ParserRecovery(tokens []string) (string, error) {
    cmd := &RECOVERY{}
    var bufferSalida bytes.Buffer

    // Unir todos los tokens en una cadena para procesamiento
    argumentos := strings.Join(tokens, " ")

    // Regex para extraer el parámetro -id
    re := regexp.MustCompile(`-id=[^\s]+`)
    coincidencias := re.FindAllString(argumentos, -1)

    // Validar que todos los tokens sean parámetros válidos
    if len(coincidencias) != len(tokens) {
        for _, token := range tokens {
            if !re.MatchString(token) {
                return "", fmt.Errorf("parámetro inválido: %s", token)
            }
        }
    }

    // Procesar cada coincidencia encontrada
    for _, coincidencia := range coincidencias {
        partes := strings.SplitN(coincidencia, "=", 2)
        if strings.ToLower(partes[0]) == "-id" && len(partes) == 2 {
            cmd.id = strings.Trim(partes[1], `"`)
        }
    }

    // Confirmar que se proporcionó el parámetro obligatorio
    if cmd.id == "" {
        return "", errors.New("falta parámetro requerido: -id")
    }

    // Ejecutar la recuperación del sistema
    err := comandoRecovery(cmd.id, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

// comandoRecovery ejecuta la recuperación del sistema de archivos EXT3 usando journaling
func comandoRecovery(idParticion string, bufferSalida *bytes.Buffer) error {
    fmt.Fprint(bufferSalida, "===================== RECOVERY =====================\n")
    fmt.Fprintf(bufferSalida, "Iniciando recuperación en partición: %s\n", idParticion)

    // Obtener información de la partición montada
    superBloqueParticion, particionMontada, rutaParticion, err := Global.ObtenerSuperblockParticionMontada(idParticion)
    if err != nil {
        return fmt.Errorf("no existe montaje %s: %w", idParticion, err)
    }

    // Verificar que la partición sea EXT3 (con journaling)
    if superBloqueParticion.S_filesystem_type != 3 {
        return fmt.Errorf("la partición no es EXT3 (sin journaling)")
    }

    // Acceder al archivo de la partición
    archivo, err := os.OpenFile(rutaParticion, os.O_RDWR, 0666)
    if err != nil {
        return fmt.Errorf("error al abrir archivo %s: %w", rutaParticion, err)
    }
    defer archivo.Close() // Liberar recurso al finalizar

    fmt.Fprintf(bufferSalida, "Sistema EXT3 detectado - Journaling disponible\n")
    fmt.Fprintf(bufferSalida, "Ejecutando recuperación sobre %s (inicio partición=%d)\n", 
        rutaParticion, particionMontada.Part_start)

    // Ejecutar el proceso de recuperación usando journaling
    err = ejecutarRecuperacion(archivo, superBloqueParticion, particionMontada.Part_start, bufferSalida)
    if err != nil {
        return fmt.Errorf("error durante recuperación: %w", err)
    }

    fmt.Fprintf(bufferSalida, "Recuperación completada exitosamente\n")
    fmt.Fprintf(bufferSalida, "El sistema se restauró usando el journal\n")
    fmt.Fprint(bufferSalida, "===================================================\n")

    return nil
}

// ejecutarRecuperacion realiza la recuperación completa del sistema de archivos
func ejecutarRecuperacion(archivo *os.File, sb *Estructuras.SuperBlock, inicioParticion int32, bufferSalida *bytes.Buffer) error {
    fmt.Fprintf(bufferSalida, "Comenzando proceso de recuperación EXT3...\n")

    // Usar la función optimizada de la estructura Recovery.go
    err := Estructuras.RecuperarSistemaArchivos(archivo, sb, inicioParticion)
    if err != nil {
        return fmt.Errorf("error durante recuperación: %w", err)
    }

    fmt.Fprintf(bufferSalida, "✓ Sistema de archivos recuperado desde journal\n")
    fmt.Fprintf(bufferSalida, "✓ Estructuras del sistema restauradas\n")
    fmt.Fprintf(bufferSalida, "✓ Operaciones del journal reproducidas\n")
    fmt.Fprintf(bufferSalida, "Proceso de recuperación completado exitosamente\n")
    return nil
}