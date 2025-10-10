package Estructuras

import (
    "backend/Utils"
    "bytes"
    "encoding/binary"
    "fmt"
    "os"
    "strings"
    "time"
)

// limpiarCadenaC extrae texto limpio eliminando caracteres nulos y espacios
func limpiarCadenaC(buffer []byte) string {
    return strings.TrimSpace(string(bytes.TrimRight(buffer, "\x00")))
}

// dividirRuta separa una ruta en directorios padre y nombre final
func dividirRuta(ruta string) ([]string, string) {
    ruta = strings.Trim(ruta, "/")
    if ruta == "" {
        return []string{}, ""
    }
    partes := strings.Split(ruta, "/")
    return partes[:len(partes)-1], partes[len(partes)-1]
}

// garantizarDirectorioRaiz verifica o crea el inodo raíz del sistema
func garantizarDirectorioRaiz(archivo *os.File, sb *SuperBlock) error {
    inodo0 := &INodo{}
    if err := inodo0.Decodificar(archivo, int64(sb.S_inode_start)); err == nil &&
        inodo0.I_type[0] == '0' {
        fmt.Println("[RECUPERACION]  Directorio raíz ya inicializado")
        return nil
    }

    fmt.Println("[RECUPERACION]  Inicializando inodo 0 como directorio raíz")

    /* actualizar bitmaps */
    if err := sb.ActualizarBitmapInodo(archivo, 0, true); err != nil {
        return err
    }
    if err := sb.ActualizarBitmapBloque(archivo, 0, true); err != nil {
        return err
    }

    /* configurar inodo raíz */
    raiz := NuevoInodoVacio()
    raiz.I_type[0] = '0'
    raiz.I_perm = [3]byte{'7', '7', '7'}
    raiz.I_block[0] = 0
    if err := raiz.Codificar(archivo, int64(sb.S_inode_start)); err != nil {
        return err
    }

    /* configurar bloque raíz */
    bloque0 := NuevoBloqueDirectorio(0, 0, map[string]int32{})
    if err := bloque0.Codificar(archivo, int64(sb.S_block_start)); err != nil {
        return err
    }

    sb.S_inodes_count = 1
    sb.S_blocks_count = 1
    sb.S_free_inodes_count--
    sb.S_free_blocks_count--
    sb.S_first_ino += sb.S_inode_size
    sb.S_first_blo += sb.S_block_size
    return nil
}

// borrarEstructurasSistema limpia bitmaps, inodos y bloques del sistema
func borrarEstructurasSistema(archivo *os.File, sb *SuperBlock) error {
    fmt.Println("[RECUPERACION]  Limpieza de bitmaps, inodos y bloques…")
    return EjecutarLimpiezaLoss(archivo, sb) // ya imprime información de rangos
}

// reproducirJournal aplica las operaciones registradas en el journal
func reproducirJournal(archivo *os.File, sb *SuperBlock, inicioParticion int32) error {
    inicioJournal := int64(inicioParticion) + int64(binary.Size(SuperBlock{}))
    fmt.Printf("[RECUPERACION]  Leyendo journal en posición=%d\n", inicioJournal)

    entradas, err := BuscarEntradasJournalValidas(archivo, inicioJournal, ENTRADAS_JOURNAL)
    if err != nil {
        return err
    }
    fmt.Printf("[RECUPERACION]  %d entrada(s) válida(s) encontradas\n", len(entradas))

    for indice, entrada := range entradas {
        operacion := limpiarCadenaC(entrada.J_content.I_operation[:])
        ruta := limpiarCadenaC(entrada.J_content.I_path[:])
        datos := limpiarCadenaC(entrada.J_content.I_content[:])

        if ruta == "" {
            continue // omitir entradas vacías
        }
        fmt.Printf("[RECUPERACION:%02d] %-6s  %s\n", indice+1, operacion, ruta)

        if operacion == "mkdir" && (ruta == "/" || ruta == "") {
            fmt.Printf("[RECUPERACION:%02d]   • mkdir / → ignorado (raíz ya existe)\n", indice+1)
            continue
        }

        directoriosPadre, nombreElemento := dividirRuta(ruta)

        switch operacion {
        case "mkdir":
            if err := sb.CrearCarpeta(archivo, directoriosPadre, nombreElemento, false); err != nil {
                return fmt.Errorf("reproducir mkdir %s: %w", ruta, err)
            }
            fmt.Printf("[RECUPERACION:%02d]   ✓ carpeta creada\n", indice+1)

        case "mkfile":
            fragmentos := Utils.DividirCadenaEnFragmentos(datos)
            if err := sb.CrearArchivo(archivo, directoriosPadre, nombreElemento,
                len(datos), fragmentos, false); err != nil {
                return fmt.Errorf("reproducir mkfile %s: %w", ruta, err)
            }
            fmt.Printf("[RECUPERACION:%02d]   ✓ archivo (%d bytes) creado\n",
                indice+1, len(datos))

        default:
            fmt.Printf("[RECUPERACION:%02d]   • operación %s no soportada (omitida)\n",
                indice+1, operacion)
        }
    }
    return nil
}

// RecuperarSistemaArchivos ejecuta el proceso completo de recuperación EXT3
func RecuperarSistemaArchivos(archivo *os.File, sb *SuperBlock, inicioParticion int32) error {
    fmt.Println("[RECUPERACION]  Iniciando recuperación EXT3…")

    /* limpiar áreas volátiles del sistema */
    if err := borrarEstructurasSistema(archivo, sb); err != nil {
        return err
    }

    /* recrear inodo raíz si es necesario */
    if err := garantizarDirectorioRaiz(archivo, sb); err != nil {
        return err
    }

    /* reproducir operaciones del journal */
    if err := reproducirJournal(archivo, sb, inicioParticion); err != nil {
        return err
    }

    /* persistir cambios del superbloque */
    sb.S_mtime = float64(time.Now().Unix())
    if err := sb.Codificar(archivo, int64(inicioParticion)); err != nil {
        return err
    }

    fmt.Println("[RECUPERACION]  Superbloque actualizado y escrito")
    return archivo.Sync()
}