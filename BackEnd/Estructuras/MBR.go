package Estructuras

import (
    "encoding/binary"
    "errors"
    "fmt"
    "os"
    "strings"
    "time"

    Utils "backend/Utils"
)

type MBR struct {
    MbrSize          int32        // Capacidad total del disco en bytes
    MbrCreacionDate  float32      // Timestamp de creación del disco
    MbrDiskSignature int32        // Identificador único del disco (generado aleatoriamente)
    MbrDiskFit       [1]byte      // Algoritmo de asignación: BF, FF, WF
    MbrPartitions    [4]Particion // Tabla de particiones (máximo 4 entradas)
}

// Serializa la estructura MBR hacia el archivo
func (mbr *MBR) Codificar(archivo *os.File) error {
    return Utils.EscribirAArchivo(archivo, 0, mbr) // Persistir MBR al inicio del archivo
}

// Reconstruye la estructura MBR desde el archivo
func (mbr *MBR) Decodificar(archivo *os.File) error {
    return Utils.LeerDeArchivo(archivo, 0, mbr) // Leer MBR desde el inicio del archivo
}

// Método para obtener la primera partición disponible
func (mbr *MBR) GetFirstAvailablePartition() (*Particion, int, int) {
    offset := binary.Size(mbr)
    for i := 0; i < len(mbr.MbrPartitions); i++ {
        if mbr.MbrPartitions[i].Part_start == -1 {
            return &mbr.MbrPartitions[i], offset, i
        } else {
            offset += int(mbr.MbrPartitions[i].Part_size)
        }
    }
    return nil, -1, -1
}

func (mbr *MBR) ObtenerPrimeraParticionDisponible() (*Particion, int, int) {
    // Calcular desplazamiento inicial considerando el MBR
    desplazamiento := binary.Size(mbr) // Dimensión del MBR en bytes

    // Explorar tabla de particiones
    for i := 0; i < len(mbr.MbrPartitions); i++ {
        if mbr.MbrPartitions[i].Part_start == -1 { // Entrada libre encontrada
            return &mbr.MbrPartitions[i], desplazamiento, i
        } else {
            // Avanzar el desplazamiento sumando la dimensión de la partición actual
            desplazamiento += int(mbr.MbrPartitions[i].Part_size)
        }
    }
    return nil, -1, -1 // No hay particiones disponibles
}

// Método para obtener una partición por nombre
func (mbr *MBR) GetPartitionByName(name string) (*Particion, int) {
    for i, partition := range mbr.MbrPartitions {
        partitionName := strings.Trim(string(partition.Part_name[:]), "\x00 ")
        inputName := strings.Trim(name, "\x00 ")
        if strings.EqualFold(partitionName, inputName) {
            return &partition, i
        }
    }
    return nil, -1
}

func (mbr *MBR) ObtenerParticionPorNombre(nombre string) (*Particion, int) {
    for i, particion := range mbr.MbrPartitions {
        nombreParticion := strings.Trim(string(particion.Part_name[:]), "\x00 ")
        nombreEntrada := strings.Trim(nombre, "\x00 ")

        // Comparación insensible a mayúsculas/minúsculas
        if strings.EqualFold(nombreParticion, nombreEntrada) {
            return &particion, i
        }
    }
    return nil, -1 // Partición no localizada
}

// Función para obtener una partición por ID
func (mbr *MBR) GetPartitionByID(id string) (*Particion, error) {
    for i := 0; i < len(mbr.MbrPartitions); i++ {
        partitionID := strings.Trim(string(mbr.MbrPartitions[i].Part_id[:]), "\x00 ")
        inputID := strings.Trim(id, "\x00 ")
        if strings.EqualFold(partitionID, inputID) {
            return &mbr.MbrPartitions[i], nil
        }
    }
    return nil, errors.New("partición no encontrada")
}

// Localiza partición mediante su identificador único
func (mbr *MBR) ObtenerParticionPorID(id string) (*Particion, error) {
    for i := 0; i < len(mbr.MbrPartitions); i++ {
        idParticion := strings.Trim(string(mbr.MbrPartitions[i].Part_id[:]), "\x00 ")
        idEntrada := strings.Trim(id, "\x00 ")

        // Verificar coincidencia de identificadores
        if strings.EqualFold(idParticion, idEntrada) {
            return &mbr.MbrPartitions[i], nil
        }
    }
    return nil, errors.New("partición con ID especificado no encontrada")
}

// HasExtendedPartition verifica si ya existe una partición extendida en el MBR
func (mbr *MBR) HasExtendedPartition() bool {
    for _, partition := range mbr.MbrPartitions {
        if partition.Part_type[0] == 'E' {
            return true
        }
    }
    return false
}

// VerificarParticionExtendida examina si existe una partición extendida activa
func (mbr *MBR) VerificarParticionExtendida() bool {
    for _, particion := range mbr.MbrPartitions {
        // Examinar tipo de partición
        if particion.Part_type[0] == 'E' {
            return true // Partición extendida detectada
        }
    }
    return false // No se encontró partición extendida
}

// CalculateAvailableSpace calcula el espacio disponible en el disco.
func (mbr *MBR) CalculateAvailableSpace() (int32, error) {
    totalSize := mbr.MbrSize
    usedSpace := int32(binary.Size(MBR{}))

    partitions := mbr.MbrPartitions[:]
    for _, part := range partitions {
        if part.Part_size != 0 {
            usedSpace += part.Part_size
        }
    }

    if usedSpace >= totalSize {
        return 0, fmt.Errorf("there is no available space on the disk")
    }

    return totalSize - usedSpace, nil
}

func (mbr *MBR) CalcularEspacioDisponible() (int32, error) {
    capacidadTotal := mbr.MbrSize
    espacioUsado := int32(binary.Size(*mbr))

    particiones := mbr.MbrPartitions[:]
    for _, part := range particiones {
        if part.Part_start != -1 && part.Part_size > 0 {
            espacioUsado += part.Part_size
        }
    }

    fmt.Printf("Debug: Capacidad total=%d, Espacio usado=%d\n", capacidadTotal, espacioUsado)

    if espacioUsado >= capacidadTotal {
        return 0, fmt.Errorf("no hay espacio disponible en el disco")
    }

    return capacidadTotal - espacioUsado, nil
}

// AplicarAjuste aplica el algoritmo de ajuste definido en el MBR
func (mbr *MBR) AplicarAjuste(tamanoParticion int32) (*Particion, error) {
    espacioDisponible, err := mbr.CalcularEspacioDisponible()
    if err != nil {
        return nil, err
    }
    if espacioDisponible < tamanoParticion {
        return nil, fmt.Errorf("no hay suficiente espacio en el disco")
    }
    switch rune(mbr.MbrDiskFit[0]) {
    case 'F': // First Fit
        return mbr.AplicarPrimerAjuste(tamanoParticion)
    case 'B': // Best Fit
        return mbr.AplicarMejorAjuste(tamanoParticion)
    case 'W': // Worst Fit
        return mbr.AplicarPeorAjuste(tamanoParticion)
    default:
        return nil, fmt.Errorf("tipo de ajuste inválido")
    }
}

// ListPartitions obtiene la información del MBR y sus particiones
func (mbr *MBR) ListPartitions() []map[string]interface{} {
    partitions := []map[string]interface{}{}
    for _, partition := range mbr.MbrPartitions {
        if partition.Part_start != -1 {
            partitionData := map[string]interface{}{
                "name": strings.Trim(string(partition.Part_name[:]), "\x00 "), // Eliminamos los caracteres nulos (\x00)
            }
            partitions = append(partitions, partitionData)
        }
    }

    return partitions
}

// ListarParticiones obtiene la información de las particiones activas del MBR
func (mbr *MBR) ListarParticiones() []map[string]interface{} {
    particiones := []map[string]interface{}{}
    for _, particion := range mbr.MbrPartitions {
        if particion.Part_start != -1 {
            datosParticion := map[string]interface{}{
                "nombre": strings.Trim(string(particion.Part_name[:]), "\x00 "), // Eliminamos los caracteres nulos (\x00)
            }
            particiones = append(particiones, datosParticion)
        }
    }
    return particiones
}

// Método para imprimir los valores del MBR
func (mbr *MBR) Print() {
    creationTime := time.Unix(int64(mbr.MbrCreacionDate), 0)
    diskFit := rune(mbr.MbrDiskFit[0])
    fmt.Printf("MBR Size: %d | Creation Date: %s | Disk Signature: %d | Disk Fit: %c\n",
        mbr.MbrSize, creationTime.Format(time.RFC3339), mbr.MbrDiskSignature, diskFit)
}

// Imprimir despliega la información principal del MBR
func (mbr *MBR) Imprimir() {
    tiempoCreacion := time.Unix(int64(mbr.MbrCreacionDate), 0)
    algoritmoAjuste := rune(mbr.MbrDiskFit[0])

    fmt.Printf("═══ MASTER BOOT RECORD ═══\n")
    fmt.Printf("Capacidad: %d bytes | Creado: %s\n",
        mbr.MbrSize, tiempoCreacion.Format("2006-01-02 15:04:05"))
    fmt.Printf("Firma: %d | Ajuste: %c\n",
        mbr.MbrDiskSignature, algoritmoAjuste)
    fmt.Printf("═══════════════════════════\n")
}

// Método para imprimir las particiones del MBR
func (mbr *MBR) PrintPartitions() {
    for i, partition := range mbr.MbrPartitions {
        partStatus := rune(partition.Part_status[0])
        partType := rune(partition.Part_type[0])
        partFit := rune(partition.Part_fit[0])
        partName := strings.TrimSpace(string(partition.Part_name[:]))
        partID := strings.TrimSpace(string(partition.Part_id[:]))
        fmt.Printf("Partition %d: Status: %c | Type: %c | Fit: %c | Start: %d | Size: %d | Name: %s | Correlative: %d | ID: %s\n",
            i+1, partStatus, partType, partFit, partition.Part_start, partition.Part_size, partName, partition.Part_correlative, partID)
    }
}

// ImprimirParticiones muestra el estado detallado de la tabla de particiones
func (mbr *MBR) ImprimirParticiones() {
    fmt.Printf("\n TABLA DE PARTICIONES\n")
    fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

    for i, particion := range mbr.MbrPartitions {
        estadoParticion := rune(particion.Part_status[0])
        tipoParticion := rune(particion.Part_type[0])
        ajusteParticion := rune(particion.Part_fit[0])
        nombreParticion := strings.TrimSpace(string(particion.Part_name[:]))
        idParticion := strings.TrimSpace(string(particion.Part_id[:]))

        // Mostrar información condensada por línea
        fmt.Printf("│ Slot %d │ Estado:%c │ Tipo:%c │ Ajuste:%c │ Inicio:%d │ Dimensión:%d │ Nombre:%s │ Correlativo:%d │ ID:%s │\n",
            i+1, estadoParticion, tipoParticion, ajusteParticion,
            particion.Part_start, particion.Part_size, nombreParticion,
            particion.Part_correlative, idParticion)
    }
    fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// Método que aplica un ajuste a las particiones del MBR (First Fit, Best Fit, Worst Fit)
func (mbr *MBR) ApplyFit(partitionSize int32) (*Particion, error) {
    availableSpace, err := mbr.CalculateAvailableSpace()
    if err != nil {
        return nil, err
    }

    if availableSpace < partitionSize {
        return nil, fmt.Errorf("no hay suficiente espacio en el disco")
    }

    switch rune(mbr.MbrDiskFit[0]) {
    case 'F': // First Fit
        return mbr.AplicarPrimerAjuste(partitionSize)
    case 'B': // Best Fit
        return mbr.AplicarMejorAjuste(partitionSize)
    case 'W': // Worst Fit
        return mbr.AplicarPeorAjuste(partitionSize)
    default:
        return nil, fmt.Errorf("tipo de ajuste inválido")
    }
}

// CalculateAvailableSpaceForPartition calcula el espacio disponible a partir del final de la partición actual
func (mbr *MBR) CalculateAvailableSpaceForPartition(partition *Particion) (int32, error) {
    startOfPartition := partition.Part_start
    endOfPartition := startOfPartition + partition.Part_size
    var nextPartitionStart int32 = -1
    for _, p := range mbr.MbrPartitions {
        if p.Part_start > endOfPartition && (nextPartitionStart == -1 || p.Part_start < nextPartitionStart) {
            nextPartitionStart = p.Part_start
        }
    }
    if nextPartitionStart == -1 {
        nextPartitionStart = mbr.MbrSize
    }

    availableSpace := nextPartitionStart - endOfPartition
    if availableSpace < 0 {
        return 0, fmt.Errorf("el cálculo de espacio disponible resultó en un valor negativo")
    }

    return availableSpace, nil
}

func (mbr *MBR) CalcularEspacioDisponibleParaParticion(particion *Particion) (int32, error) {
    inicioParticion := particion.Part_start
    finParticion := inicioParticion + particion.Part_size
    var siguienteInicioParticion int32 = -1

    for _, p := range mbr.MbrPartitions {
        if p.Part_start > finParticion && (siguienteInicioParticion == -1 || p.Part_start < siguienteInicioParticion) {
            siguienteInicioParticion = p.Part_start
        }
    }

    if siguienteInicioParticion == -1 {
        siguienteInicioParticion = mbr.MbrSize
    }

    espacioDisponible := siguienteInicioParticion - finParticion
    if espacioDisponible < 0 {
        return 0, fmt.Errorf("el cálculo de espacio disponible resultó en un valor negativo")
    }

    return espacioDisponible, nil
}

// AplicarPrimerAjuste: Encuentra el primer espacio disponible que sea mayor o igual al tamaño de la partición
func (mbr *MBR) AplicarPrimerAjuste(tamanoParticion int32) (*Particion, error) {
    fmt.Println("Iniciando First Fit...")
    desplazamiento := binary.Size(*mbr)
    for i := 0; i < len(mbr.MbrPartitions); i++ {
        particion := &mbr.MbrPartitions[i]
        fmt.Printf("Evaluando partición %d: Inicio %d, Tamaño %d, Estado %c\n", i, particion.Part_start, particion.Part_size, particion.Part_status[0])
        if particion.Part_start == -1 {
            fmt.Printf("Partición %d es adecuada para First Fit: Inicio en %d, Tamaño %d\n", i, desplazamiento, tamanoParticion)
            particion.Part_start = int32(desplazamiento)
            particion.Part_size = tamanoParticion
            return particion, nil
        } else {
            desplazamiento += int(particion.Part_size)
        }
    }

    fmt.Println("No se encontró espacio suficiente con First Fit.")
    return nil, fmt.Errorf("no se encontró espacio suficiente con First Fit")
}

// AplicarMejorAjuste: Encuentra el espacio disponible más pequeño que sea mayor o igual al tamaño de la partición
func (mbr *MBR) AplicarMejorAjuste(tamanoParticion int32) (*Particion, error) {
    fmt.Println("Iniciando Best Fit...")
    mejorAjuste := -1
    desplazamiento := binary.Size(*mbr)
    
    for i := 0; i < len(mbr.MbrPartitions); i++ {
        particion := &mbr.MbrPartitions[i]
        fmt.Printf("Evaluando partición %d: Inicio %d, Tamaño %d, Estado %c\n", i, particion.Part_start, particion.Part_size, particion.Part_status[0])
        if particion.Part_start == -1 {
            mejorAjuste = i
            fmt.Printf("Partición %d seleccionada para Best Fit: Inicio en %d, Tamaño %d\n", mejorAjuste, desplazamiento, tamanoParticion)
            break
        } else {
            desplazamiento += int(particion.Part_size)
        }
    }
    
    if mejorAjuste == -1 {
        fmt.Println("No se encontró espacio suficiente con Best Fit.")
        return nil, fmt.Errorf("no se encontró espacio suficiente con Best Fit")
    }
    
    particion := &mbr.MbrPartitions[mejorAjuste]
    particion.Part_start = int32(desplazamiento)
    particion.Part_size = tamanoParticion
    return particion, nil
}

// AplicarPeorAjuste: Encuentra el espacio disponible más grande que sea mayor o igual al tamaño de la partición
func (mbr *MBR) AplicarPeorAjuste(tamanoParticion int32) (*Particion, error) {
    fmt.Println("Iniciando Worst Fit...")
    peorAjuste := -1
    desplazamiento := binary.Size(*mbr)
    
    for i := 0; i < len(mbr.MbrPartitions); i++ {
        particion := &mbr.MbrPartitions[i]
        fmt.Printf("Evaluando partición %d: Inicio %d, Tamaño %d, Estado %c\n", i, particion.Part_start, particion.Part_size, particion.Part_status[0])
        if particion.Part_start == -1 {
            peorAjuste = i
            fmt.Printf("Partición %d seleccionada para Worst Fit: Inicio en %d, Tamaño %d\n", peorAjuste, desplazamiento, tamanoParticion)
            break
        } else {
            desplazamiento += int(particion.Part_size)
        }
    }
    
    if peorAjuste == -1 {
        fmt.Println("No se encontró espacio suficiente con Worst Fit.")
        return nil, fmt.Errorf("no se encontró espacio suficiente con Worst Fit")
    }
    
    particion := &mbr.MbrPartitions[peorAjuste]
    particion.Part_start = int32(desplazamiento)
    particion.Part_size = tamanoParticion
    return particion, nil
}

func (mbr *MBR) CreatePartitionWithFit(partSize int32, partType, partName string) error {
    availableSpace, err := mbr.CalculateAvailableSpace()
    if err != nil {
        return fmt.Errorf("error calculando el espacio disponible: %v", err)
    }
    if availableSpace < partSize {
        return fmt.Errorf("no hay suficiente espacio en el disco para la nueva partición")
    }
    partition, err := mbr.ApplyFit(partSize)
    if err != nil {
        return fmt.Errorf("error al aplicar el ajuste: %v", err)
    }
    partition.Part_status[0] = '1' // Activar partición (1 = Activa)
    partition.Part_size = partSize
    if len(partType) > 0 {
        partition.Part_type[0] = partType[0]
    }

    // Asignar el tipo de ajuste (fit) basado en el MBR
    switch mbr.MbrDiskFit[0] {
    case 'B', 'F', 'W':
        partition.Part_fit[0] = mbr.MbrDiskFit[0]
    default:
        return fmt.Errorf("ajuste inválido en el MBR: %c. Debe ser BF (Best Fit), FF (First Fit) o WF (Worst Fit)", mbr.MbrDiskFit[0])
    }
    copy(partition.Part_name[:], partName)

    fmt.Printf("Partición '%s' creada exitosamente con el ajuste '%c'.\n", partName, mbr.MbrDiskFit[0])
    return nil
}

// Crea una partición aplicando el ajuste definido en el MBR (Best Fit, First Fit, Worst Fit)
func (mbr *MBR) CrearParticionConAjuste(tamanoParticion int32, tipoParticion, nombreParticion string) error {
    espacioDisponible, err := mbr.CalcularEspacioDisponible()
    if err != nil {
        return fmt.Errorf("error calculando el espacio disponible: %v", err)
    }
    if espacioDisponible < tamanoParticion {
        return fmt.Errorf("no hay suficiente espacio en el disco para la nueva partición")
    }
    particion, err := mbr.AplicarAjuste(tamanoParticion)
    if err != nil {
        return fmt.Errorf("error al aplicar el ajuste: %v", err)
    }
    particion.Part_status[0] = '1' // Activar partición (1 = Activa)
    particion.Part_size = tamanoParticion
    if len(tipoParticion) > 0 {
        particion.Part_type[0] = tipoParticion[0]
    }
    // Asignar el tipo de ajuste (fit) basado en el MBR
    switch mbr.MbrDiskFit[0] {
    case 'B', 'F', 'W':
        particion.Part_fit[0] = mbr.MbrDiskFit[0]
    default:
        return fmt.Errorf("ajuste inválido en el MBR: %c. Debe ser BF (Best Fit), FF (First Fit) o WF (Worst Fit)", mbr.MbrDiskFit[0])
    }
    copy(particion.Part_name[:], nombreParticion)
    fmt.Printf("Partición '%s' creada exitosamente con el ajuste '%c'.\n", nombreParticion, mbr.MbrDiskFit[0])
    return nil
}

