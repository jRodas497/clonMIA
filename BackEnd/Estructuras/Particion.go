package Estructuras

import (
    "fmt"
    "os"
    "strings"
)

// Constantes para los valores de ajuste
const (
    AjusteBF = 'B' // Best Fit
    AjusteFF = 'F' // First Fit
    AjusteWF = 'W' // Worst Fit
)

// Estructura que representa una particion
type Particion struct {
    Part_status      [1]byte  // indica si la partición está activa (1) o inactiva (0)
    Part_type        [1]byte  // P para primaria, E para extendida
    Part_fit         [1]byte  // BF para Best Fit, FF para First Fit, WF para Worst Fit
    Part_start       int32    // posición inicial de la partición en bytes
    Part_size        int32    // tamaño de la partición en bytes
    Part_name        [16]byte // nombre asignado a la partición
    Part_correlative int32    // número correlativo, se asigna al montar la partición
    Part_id          [4]byte  // identificador único, se asigna al montar la partición
    // Estructura de 32 bytes en total
}

// Metodo que crea una particion
func (p *Particion) CrearParticion(inicioParticion, capacidadParticion int, tipoParticion, ajusteParticion, nombreParticion string) {
    // Asignamos el valor status de la particion
    p.Part_status[0] = '0' // 0 = Inactiva, 1 = Activa
    p.Part_start = int32(inicioParticion)
    p.Part_size = int32(capacidadParticion)

    if len(tipoParticion) > 0 {
        p.Part_type[0] = tipoParticion[0]
    }

    if len(ajusteParticion) > 0 {
        p.Part_fit[0] = ajusteParticion[0]
    }

    copy(p.Part_name[:], nombreParticion)
}

// Metodo que modifica el tamaño de una particion
func (p *Particion) ModificarTamano(cambioTamano int32, espacioDisponible int32) error {
    nuevoTamano := p.Part_size + cambioTamano

    if nuevoTamano < 0 {
        return fmt.Errorf("el tamaño de la partición no puede ser negativo")
    }

    if cambioTamano > 0 && espacioDisponible < cambioTamano {
        return fmt.Errorf("no hay suficiente espacio disponible para agregar a la partición")
    }
    p.Part_size = nuevoTamano

    fmt.Printf("El tamaño de la partición '%s' ha sido modificado. Nuevo tamaño: %d bytes.\n", string(p.Part_name[:]), p.Part_size)
    return nil
}

func (p *Particion) Eliminar(tipoEliminacion string, archivo *os.File, esExtendida bool) error {
    if esExtendida {
        err := p.eliminarParticionesLogicas(archivo)
        if err != nil {
            return fmt.Errorf("error al eliminar las particiones lógicas dentro de la partición extendida: %v", err)
        }
    }
    p.Part_start = -1
    p.Part_size = -1
    p.Part_name = [16]byte{}
    if tipoEliminacion == "full" {
        err := p.Sobrescribir(archivo)
        if err != nil {
            return fmt.Errorf("error al sobrescribir la partición: %v", err)
        }
    }

    fmt.Printf("La partición '%s' ha sido eliminada (%s).\n", strings.TrimSpace(string(p.Part_name[:])), tipoEliminacion)
    return nil
}

// Metodo que sobrescribe el espacio de la particion con \0 (para eliminacion Full)
func (p *Particion) Sobrescribir(archivo *os.File) error {
    _, err := archivo.Seek(int64(p.Part_start), 0)
    if err != nil {
        return err
    }
    ceros := make([]byte, p.Part_size)
    _, err = archivo.Write(ceros)
    if err != nil {
        return fmt.Errorf("error al sobrescribir el espacio de la partición: %v", err)
    }

    fmt.Printf("Espacio de la partición sobrescrito con ceros.\n")
    return nil
}

// Metodo para eliminar todas las particiones logicas dentro de una particion extendida
func (p *Particion) eliminarParticionesLogicas(archivo *os.File) error {
    fmt.Println("Eliminando particiones lógicas dentro de la partición extendida...")
    var ebrActual EBR
    inicio := p.Part_start
    for {
        err := ebrActual.Decodificar(archivo, int64(inicio))
        if err != nil {
            return fmt.Errorf("error al leer el EBR: %v", err)
        }
        if ebrActual.Ebr_start == -1 {
            break
        }
        ebrActual.Ebr_start = -1
        ebrActual.Ebr_size = -1
        copy(ebrActual.Ebr_name[:], "")

        err = ebrActual.Sobrescribir(archivo)
        if err != nil {
            return fmt.Errorf("error al sobrescribir el EBR: %v", err)
        }

        inicio = ebrActual.Ebr_next
    }

    fmt.Println("Particiones lógicas eliminadas exitosamente.")
    return nil
}

// Metodo que monta una particion
func (p *Particion) MontarParticion(correlativo int, id string) error {
    p.Part_correlative = int32(correlativo)
    copy(p.Part_id[:], id)
    return nil
}

// Imprimir los valores de la partición en una sola línea
func (p *Particion) Imprimir() {
    fmt.Printf("Estado: %c | Tipo: %c | Ajuste: %c | Inicio: %d | Capacidad: %d | Nombre: %s | Correlativo: %d | ID: %s\n",
        p.Part_status[0], p.Part_type[0], p.Part_fit[0], p.Part_start, p.Part_size,
        string(p.Part_name[:]), p.Part_correlative, string(p.Part_id[:]))
}