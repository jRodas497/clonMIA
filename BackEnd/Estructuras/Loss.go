package Estructuras

import (
    "fmt"
    "os"
)

// MENSAJE emite información de depuración con prefijo específico
func MENSAJE(formato string, args ...any) {
    fmt.Printf("[LOSS] "+formato+"\n", args...)
}

// LimpiarRegion sobrescribe con ceros el área especificada del archivo
func LimpiarRegion(archivo *os.File, posicion, longitud int64) error {
    const fragmento = 4096
    buffer := make([]byte, fragmento)
    var escritos int64
    
    for escritos < longitud {
        cantidad := longitud - escritos
        if cantidad > fragmento {
            cantidad = fragmento
        }
        
        if _, err := archivo.WriteAt(buffer[:cantidad], posicion+escritos); err != nil {
            return fmt.Errorf("error escribiendo ceros en posición=%d: %w", posicion+escritos, err)
        }
        escritos += cantidad
    }
    
    MENSAJE("  ↳ %d bytes limpiados en rango [%d, %d)", longitud, posicion, posicion+longitud)
    return nil
}

// EjecutarLimpiezaLoss borra selectivamente bitmaps, área de inodos y bloques
func EjecutarLimpiezaLoss(archivo *os.File, sb *SuperBlock) error {
    totalInodos := int64(sb.S_inodes_count + sb.S_free_inodes_count)
    totalBloques := int64(sb.S_blocks_count + sb.S_free_blocks_count)
    tamañoInodo := int64(sb.S_inode_size)
    tamañoBloque := int64(sb.S_block_size)

    MENSAJE("Simulando LOSS — inodos=%d  bloques=%d", totalInodos, totalBloques)

    // 1️⃣ Bitmap de inodos
    longitudBitmapInodos := (totalInodos + 7) / 8
    if err := LimpiarRegion(archivo, int64(sb.S_bm_inode_start), longitudBitmapInodos); err != nil {
        return err
    }

    // 2️⃣ Bitmap de bloques
    longitudBitmapBloques := (totalBloques + 7) / 8
    if err := LimpiarRegion(archivo, int64(sb.S_bm_block_start), longitudBitmapBloques); err != nil {
        return err
    }

    // 3️⃣ Tabla completa de inodos
    if err := LimpiarRegion(archivo, int64(sb.S_inode_start), totalInodos*tamañoInodo); err != nil {
        return err
    }

    // 4️⃣ Área completa de bloques de datos
    if err := LimpiarRegion(archivo, int64(sb.S_block_start), totalBloques*tamañoBloque); err != nil {
        return err
    }

    // Sincronizar cambios al disco
    return archivo.Sync()
}