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

// LOSS estructura del comando loss con parámetros
type LOSS struct {
    id string // Identificador de la partición montada donde simular pérdida
}

// ParserLoss analiza los argumentos del comando loss y ejecuta la simulación de pérdida
func ParserLoss(tokens []string) (string, error) {
    cmd := &LOSS{}
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

    // Ejecutar la simulación de pérdida
    err := comandoLoss(cmd.id, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

// comandoLoss simula pérdida de datos limpiando áreas críticas de una partición montada
func comandoLoss(idParticion string, bufferSalida *bytes.Buffer) error {
    fmt.Fprint(bufferSalida, "======================= LOSS =======================\n")
    fmt.Fprintf(bufferSalida, "Iniciando simulación de pérdida en partición: %s\n", idParticion)

    // Obtener información de la partición montada
    superBloqueParticion, particionMontada, rutaParticion, err := Global.ObtenerSuperblockParticionMontada(idParticion)
    if err != nil {
        return fmt.Errorf("no existe montaje %s: %w", idParticion, err)
    }

    // Acceder al archivo de la partición
    archivo, err := os.OpenFile(rutaParticion, os.O_RDWR, 0666)
    if err != nil {
        return fmt.Errorf("error al abrir archivo %s: %w", rutaParticion, err)
    }
    defer archivo.Close() // Liberar recurso al finalizar

    // Cargar el superbloque desde la partición
    err = superBloqueParticion.Decodificar(archivo, int64(particionMontada.Part_start))
    if err != nil {
        return fmt.Errorf("error al leer superbloque: %w", err)
    }

    fmt.Fprintf(bufferSalida, "Operación LOSS ejecutándose sobre %s (inicio partición=%d)\n", 
        rutaParticion, particionMontada.Part_start)

    // Ejecutar la limpieza de áreas críticas
    err = simularPerdidaDatos(archivo, superBloqueParticion, bufferSalida)
    if err != nil {
        return fmt.Errorf("error durante simulación de pérdida: %w", err)
    }

    fmt.Fprintf(bufferSalida, "Simulación de pérdida completada en partición %s\n", idParticion)
    fmt.Fprint(bufferSalida, "===================================================\n")

    return nil
}

// simularPerdidaDatos limpia selectivamente áreas críticas del sistema de archivos
func simularPerdidaDatos(archivo *os.File, sb *Estructuras.SuperBlock, bufferSalida *bytes.Buffer) error {
    fmt.Fprintf(bufferSalida, "Comenzando limpieza de áreas críticas del sistema...\n")

    // Usar la función optimizada de la estructura Loss.go
    err := Estructuras.EjecutarLimpiezaLoss(archivo, sb)
    if err != nil {
        return fmt.Errorf("error durante limpieza: %w", err)
    }

    fmt.Fprintf(bufferSalida, "Todas las áreas críticas limpiadas con \\0\n")
    fmt.Fprintf(bufferSalida, "Simulación de pérdida completada exitosamente\n")
    return nil
}