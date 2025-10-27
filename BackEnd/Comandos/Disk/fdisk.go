package Disk

import (
    Estructuras "backend/Estructuras"
    Utils "backend/Utils"
    "bytes"
    "errors"
    "fmt"
    "os"
    "regexp"
    "strconv"
    "strings"
)

// FDisk representa el comando fDisk con sus parametros
type FDisk struct {
    capacidad int    // Dimension de la particion
    unidad    string // Unidad de medida del dimension (K o M)
    ajuste    string // Tipo de ajuste (BF, FF, WF)
    ruta      string // Ubicacion del archivo del disco
    tipo      string // Categoria de particion (P, E, L)
    nombre    string // Identificador de la particion
    agregar   int    // Espacio a agregar o quitar
    eliminar  string // Metodo de eliminacion (fast o full)
}

// Procesa el comando fDisk y retorna los mensajes generados
func ParserFdisk(tokens []string) (string, error) {
    var bufferSalida bytes.Buffer
    cmd := &FDisk{}

    argumentos := strings.Join(tokens, " ")
    patron := regexp.MustCompile(`-size=\d+|-unit=[bBkKmM]|-fit=[bBfFwfW]{2}|-path="[^"]+"|-path=[^\s]+|-type=[pPeElL]|-name="[^"]+"|-name=[^\s]+|-add=[+-]?\d+|-delete=(fast|full)`)
    coincidencias := patron.FindAllString(argumentos, -1)

    for _, coincidencia := range coincidencias {
        claveValor := strings.SplitN(coincidencia, "=", 2)
        if len(claveValor) != 2 {
            return "", fmt.Errorf("formato de parametro invalido: %s", coincidencia)
        }
        clave, valor := strings.ToLower(claveValor[0]), claveValor[1]
        if strings.HasPrefix(valor, "\"") && strings.HasSuffix(valor, "\"") {
            valor = strings.Trim(valor, "\"")
        }

        switch clave {
        case "-size":
            dimension, err := strconv.Atoi(valor)
            if err != nil || dimension <= 0 {
                return "", errors.New("la dimension debe ser un numero entero positivo")
            }
            cmd.capacidad = dimension
        case "-unit":
            valor = strings.ToUpper(valor)
            if valor != "B" && valor != "K" && valor != "M" {
                return "", errors.New("la unidad debe ser B, K, M")
            }
            cmd.unidad = valor
        case "-fit":
            valor = strings.ToUpper(valor)
            if valor != "BF" && valor != "FF" && valor != "WF" {
                return "", errors.New("el ajuste debe ser BF, FF, WF")
            }
            cmd.ajuste = valor
        case "-path":
            if valor == "" {
                return "", errors.New("la ruta no puede estar vacia")
            }
            cmd.ruta = valor
        case "-type":
            valor = strings.ToUpper(valor)
            if valor != "P" && valor != "E" && valor != "L" {
                return "", errors.New("el tipo debe ser P, E, L")
            }
            cmd.tipo = valor
        case "-name":
            if valor == "" {
                return "", errors.New("el nombre no puede estar vacio")
            }
            cmd.nombre = valor
        case "-add":
            agregar, err := strconv.Atoi(valor)
            if err != nil {
                return "", errors.New("el valor de -add debe ser un numero entero")
            }
            cmd.agregar = agregar
        case "-delete":
            valor = strings.ToLower(valor)
            if valor != "fast" && valor != "full" {
                return "", errors.New("el valor de -delete debe ser 'fast' o 'full'")
            }
            cmd.eliminar = valor
        default:
            return "", fmt.Errorf("parametro desconocido: %s", clave)
        }
    }

    // Identificar el tipo de operacion: add, delete o crear particion
    if cmd.eliminar != "" {
        // Operacion de eliminacion de particion
        if cmd.ruta == "" {
            return "", errors.New("falta el parametro requerido: -path")
        }
        if cmd.nombre == "" {
            return "", errors.New("falta el parametro requerido: -name")
        }
        return procesarEliminarParticion(cmd, &bufferSalida)
    }

    if cmd.agregar != 0 {
        // Operacion de agregar/quitar espacio
        if cmd.ruta == "" {
            return "", errors.New("falta el parametro requerido: -path")
        }
        if cmd.nombre == "" {
            return "", errors.New("falta el parametro requerido: -name")
        }
        return procesarAgregarParticion(cmd, &bufferSalida)
    }

    // Operacion de crear particion (requiere -size, -path, -name)
    if cmd.capacidad == 0 {
        return "", errors.New("faltan parametros requeridos: -size")
    }
    if cmd.ruta == "" {
        return "", errors.New("faltan parametros requeridos: -path")
    }
    if cmd.nombre == "" {
        return "", errors.New("faltan parametros requeridos: -name")
    }

    // Asignar valores predeterminados
    if cmd.unidad == "" {
        cmd.unidad = "K"
    }
    if cmd.ajuste == "" {
        cmd.ajuste = "WF"
    }
    if cmd.tipo == "" {
        cmd.tipo = "P"
    }

    // Ejecutar operacion fdisk y capturar mensajes en el buffer
    err := ejecutarComandoFdisk(cmd, &bufferSalida)
    if err != nil {
        return "", fmt.Errorf("error al crear la particion: %v", err)
    }

    return bufferSalida.String(), nil
}

// procesarEliminarParticion maneja la eliminacion de particiones
func procesarEliminarParticion(cmd *FDisk, bufferSalida *bytes.Buffer) (string, error) {
    fmt.Fprintf(bufferSalida, "========================== ELIMINAR ==========================\n")
    fmt.Fprintf(bufferSalida, "Eliminando particion con nombre '%s' usando el metodo %s...\n", cmd.nombre, cmd.eliminar)

    // Abrir el archivo del disco
    archivo, err := os.OpenFile(cmd.ruta, os.O_RDWR, 0644)
    if err != nil {
        return "", fmt.Errorf("error abriendo el archivo del disco: %v", err)
    }
    defer archivo.Close()

    // Leer el MBR del archivo
    var mbr Estructuras.MBR
    err = mbr.Decodificar(archivo)
    if err != nil {
        return "", fmt.Errorf("error al deserializar el MBR: %v", err)
    }

    // Buscar la particion por nombre y eliminarla
    particion, _ := mbr.ObtenerParticionPorNombre(cmd.nombre)
    if particion == nil {
        return "", fmt.Errorf("la particion '%s' no existe", cmd.nombre)
    }

    // Verificar si es extendida para eliminar particiones logicas
    esExtendida := particion.Part_type[0] == 'E'
    err = particion.Eliminar(cmd.eliminar, archivo, esExtendida)
    if err != nil {
        return "", fmt.Errorf("error al eliminar la particion: %v", err)
    }

    // No limpiar entradas del MBR aquí: la modificación ya se aplica directamente sobre la partición
    // (mbr.ObtenerParticionPorNombre devuelve ahora un puntero a la entrada del MBR)
    // Actualizar el MBR en el archivo despues de la eliminacion
    err = mbr.Codificar(archivo)
    if err != nil {
        return "", fmt.Errorf("error al actualizar el MBR en el disco: %v", err)
    }

    // Mensaje de exito
    fmt.Fprintf(bufferSalida, "Particion '%s' eliminada exitosamente.\n", cmd.nombre)
    fmt.Fprintf(bufferSalida, "===========================================================\n")

    // Imprimir las particiones restantes
    fmt.Fprintf(bufferSalida, "========================== PARTICIONES ==========================\n")
    imprimirParticiones(&mbr, bufferSalida)
    fmt.Fprintf(bufferSalida, "===========================================================\n")

    return bufferSalida.String(), nil
}

// procesarAgregarParticion maneja el agregar o quitar espacio a particiones
func procesarAgregarParticion(cmd *FDisk, bufferSalida *bytes.Buffer) (string, error) {
    fmt.Fprintf(bufferSalida, "========================== AGREGAR ==========================\n")
    fmt.Fprintf(bufferSalida, "Modificando particion '%s', ajustando %d unidades...\n", cmd.nombre, cmd.agregar)

    // Abrir el archivo del disco
    archivo, err := os.OpenFile(cmd.ruta, os.O_RDWR, 0644)
    if err != nil {
        return "", fmt.Errorf("error abriendo el archivo del disco: %v", err)
    }
    defer archivo.Close()

    // Leer el MBR del archivo
    var mbr Estructuras.MBR
    err = mbr.Decodificar(archivo)
    if err != nil {
        return "", fmt.Errorf("error al deserializar el MBR: %v", err)
    }

    // Buscar la particion por nombre
    particion, _ := mbr.ObtenerParticionPorNombre(cmd.nombre)
    if particion == nil {
        return "", fmt.Errorf("la particion '%s' no existe", cmd.nombre)
    }

    // Convertir cmd.agregar a bytes segun la unidad especificada
    bytesAgregar, err := Utils.ConvertirABytes(cmd.agregar, cmd.unidad)
    if err != nil {
        return "", fmt.Errorf("error al convertir las unidades de -add: %v", err)
    }

    // Calcular espacio disponible si se esta agregando espacio
    var espacioDisponible int32 = 0
    if bytesAgregar > 0 {
        espacioDisponible, err = mbr.CalcularEspacioDisponibleParaParticion(particion)
        if err != nil {
            return "", fmt.Errorf("error al calcular el espacio disponible para la particion '%s': %v", cmd.nombre, err)
        }
    }

    // Modificar el tamaño de la particion
    err = particion.ModificarTamano(int32(bytesAgregar), espacioDisponible)
    if err != nil {
        return "", fmt.Errorf("error al modificar el tamaño de la particion: %v", err)
    }

    // Actualizar el MBR en el archivo despues de la modificacion
    err = mbr.Codificar(archivo)
    if err != nil {
        return "", fmt.Errorf("error al actualizar el MBR en el disco: %v", err)
    }

    // Mensaje de exito
    fmt.Fprintf(bufferSalida, "Espacio en la particion '%s' modificado exitosamente.\n", cmd.nombre)
    fmt.Fprintf(bufferSalida, "===========================================================\n")

    // Imprimir las particiones despues de modificar el espacio
    fmt.Fprintf(bufferSalida, "========================== PARTICIONES ==========================\n")
    imprimirParticiones(&mbr, bufferSalida)
    fmt.Fprintf(bufferSalida, "===========================================================\n")

    return bufferSalida.String(), nil
}

// imprimirParticiones imprime las particiones actuales del MBR
func imprimirParticiones(mbr *Estructuras.MBR, bufferSalida *bytes.Buffer) {
    for i, particion := range mbr.MbrPartitions {
        if particion.Part_start != -1 {
            fmt.Fprintf(bufferSalida, "Particion %d: Nombre: %s | Inicio: %d | Tamaño: %d bytes | Tipo: %c | Estado: %c\n",
                i+1,
                strings.TrimSpace(string(particion.Part_name[:])),
                particion.Part_start,
                particion.Part_size,
                particion.Part_type[0],
                particion.Part_status[0],
            )
        } else {
            fmt.Fprintf(bufferSalida, "Particion %d: (Vacia)\n", i+1)
        }
    }
}

func ejecutarComandoFdisk(fdisk *FDisk, bufferSalida *bytes.Buffer) error {
    fmt.Fprintf(bufferSalida, "---------------------------- FDisk ----------------------------\n")
    fmt.Fprintf(bufferSalida, "Generando particion '%s' con dimension %d %s\n",
        fdisk.nombre, fdisk.capacidad, fdisk.unidad)

    fmt.Printf("Detalles internos: dimension=%d, unidad=%s, ajuste=%s, ubicacion=%s, categoria=%s, nombre=%s\n",
        fdisk.capacidad, fdisk.unidad, fdisk.ajuste, fdisk.ruta, fdisk.tipo, fdisk.nombre)

    // Acceder al archivo del disco
    archivo, err := os.OpenFile(fdisk.ruta, os.O_RDWR, 0644)
    if err != nil {
        return fmt.Errorf("error accediendo al archivo del disco: %v", err)
    }
    defer archivo.Close()

    bytesCapacidad, err := Utils.ConvertirABytes(fdisk.capacidad, fdisk.unidad)
    if err != nil {
        fmt.Println("Error convirtiendo dimension:", err)
        return err
    }

    switch fdisk.tipo {
    case "P":
        err = crearParticionPrimaria(archivo, fdisk, bytesCapacidad, bufferSalida)
        if err != nil {
            fmt.Println("Error generando particion primaria:", err)
            return err
        }
    case "E":
        fmt.Println("Generando particion extendida...")
        err = crearParticionExtendida(archivo, fdisk, bytesCapacidad, bufferSalida)
        if err != nil {
            fmt.Println("Error generando particion extendida:", err)
            return err
        }
    case "L":
        fmt.Println("Generando particion logica...")
        err = crearParticionLogica(archivo, fdisk, bytesCapacidad, bufferSalida)
        if err != nil {
            fmt.Println("Error generando particion logica:", err)
            return err
        }
    }

    fmt.Fprintln(bufferSalida, "Particion generada correctamente.")
    fmt.Fprintln(bufferSalida, "--------------------------------------------")
    return nil
}

// Generar una particion primaria
func crearParticionPrimaria(archivo *os.File, fdisk *FDisk, bytesCapacidad int, bufferSalida *bytes.Buffer) error {
    fmt.Fprintf(bufferSalida, "Construyendo particion primaria con dimension %d %s\n", fdisk.capacidad, fdisk.unidad)

    var mbr Estructuras.MBR
    err := mbr.Decodificar(archivo)
    if err != nil {
        return fmt.Errorf("error al deserializar el MBR: %v", err)
    }
    espacioDisponible, err := mbr.CalcularEspacioDisponible()
    if err != nil {
        fmt.Println("Error calculando el espacio disponible:", err)
    } else {
        fmt.Println("Espacio disponible en el disco:", espacioDisponible)
    }

    // Llamar al metodo del MBR para crear la particion con el ajuste correspondiente
    err = mbr.CrearParticionConAjuste(int32(bytesCapacidad), fdisk.tipo, fdisk.nombre)
    if err != nil {
        return fmt.Errorf("error al crear la particion primaria: %v", err)
    }

    // Actualizar el MBR en el archivo del disco
    err = mbr.Codificar(archivo)
    if err != nil {
        return fmt.Errorf("error al actualizar el MBR en el disco: %v", err)
    }

    fmt.Fprintln(bufferSalida, "Particion primaria construida correctamente.")
    return nil
}

// Generar una particion extendida
func crearParticionExtendida(archivo *os.File, fdisk *FDisk, bytesCapacidad int, bufferSalida *bytes.Buffer) error {
    fmt.Fprintf(bufferSalida, "Construyendo particion extendida con dimension %d %s\n", fdisk.capacidad, fdisk.unidad)
    var mbr Estructuras.MBR

    // Deserializar la estructura MBR desde el archivo
    err := mbr.Decodificar(archivo)
    if err != nil {
        return fmt.Errorf("error al deserializar el MBR: %v", err)
    }

    // Verificar si ya existe una particion extendida
    if mbr.VerificarParticionExtendida() {
        return errors.New("ya existe una particion extendida en este disco")
    }

    // Usar el metodo del MBR para crear la particion con el ajuste correspondiente
    err = mbr.CrearParticionConAjuste(int32(bytesCapacidad), "E", fdisk.nombre)
    if err != nil {
        return fmt.Errorf("error al crear la particion extendida: %v", err)
    }

    // Crear el primer EBR dentro de la particion extendida
    particionExtendida, _ := mbr.ObtenerParticionPorNombre(fdisk.nombre)
    err = Estructuras.CrearYEscribirEBR(particionExtendida.Part_start, 0, fdisk.ajuste[0], fdisk.nombre, archivo)
    if err != nil {
        return fmt.Errorf("error al crear el primer EBR en la particion extendida: %v", err)
    }

    // Actualizar el MBR
    err = mbr.Codificar(archivo)
    if err != nil {
        return fmt.Errorf("error al actualizar el MBR en el disco: %v", err)
    }

    fmt.Fprintln(bufferSalida, "Particion extendida construida correctamente.")
    return nil
}

// Generar una particion logica
func crearParticionLogica(archivo *os.File, fdisk *FDisk, bytesCapacidad int, bufferSalida *bytes.Buffer) error {
    fmt.Fprintf(bufferSalida, "Construyendo particion logica con dimension %d %s\n", fdisk.capacidad, fdisk.unidad)
    var mbr Estructuras.MBR

    err := mbr.Decodificar(archivo)
    if err != nil {
        return fmt.Errorf("error al deserializar el MBR: %v", err)
    }

    // Verificar si existe una particion extendida utilizando VerificarParticionExtendida
    if !mbr.VerificarParticionExtendida() {
        return errors.New("no se encontro una particion extendida en el disco")
    }

    // Identificar la particion extendida especifica
    var particionExtendida *Estructuras.Particion
    for i := range mbr.MbrPartitions {
        if mbr.MbrPartitions[i].Part_type[0] == 'E' {
            particionExtendida = &mbr.MbrPartitions[i]
            break
        }
    }

    // Buscar el ultimo EBR en la particion extendida
    ultimoEBR, err := Estructuras.BuscarUltimoEBR(particionExtendida.Part_start, archivo)
    if err != nil {
        return fmt.Errorf("error al buscar el ultimo EBR: %v", err)
    }

    // Verificar si es el primer EBR
    if ultimoEBR.Ebr_size == 0 {
        fmt.Println("Detectado EBR inicial vacio, asignando dimension a la nueva particion logica.")
        ultimoEBR.Ebr_size = int32(bytesCapacidad)
        copy(ultimoEBR.Ebr_name[:], fdisk.nombre)

        err = ultimoEBR.Codificar(archivo, int64(ultimoEBR.Ebr_start))
        if err != nil {
            return fmt.Errorf("error al escribir el primer EBR con la nueva particion logica: %v", err)
        }

        fmt.Fprintln(bufferSalida, "Primera particion logica construida correctamente.")
        return nil
    }

    // Calcular el inicio del nuevo EBR
    nuevoInicioEBR, err := ultimoEBR.CalcularInicioSiguienteEBR(particionExtendida.Part_start, particionExtendida.Part_size)
    if err != nil {
        return fmt.Errorf("error calculando el inicio del nuevo EBR: %v", err)
    }

    dimensionDisponible := particionExtendida.Part_size - (nuevoInicioEBR - particionExtendida.Part_start)
    if dimensionDisponible < int32(bytesCapacidad) {
        return errors.New("no hay suficiente espacio en la particion extendida para una nueva particion logica")
    }

    // Crear el nuevo EBR
    nuevoEBR := Estructuras.EBR{}
    nuevoEBR.EstablecerEBR(fdisk.ajuste[0], int32(bytesCapacidad), nuevoInicioEBR, -1, fdisk.nombre)

    // Escribir el nuevo EBR en el disco
    err = nuevoEBR.Codificar(archivo, int64(nuevoInicioEBR))
    if err != nil {
        return fmt.Errorf("error al escribir el nuevo EBR en el disco: %v", err)
    }

    // Actualizar el ultimo EBR para que apunte al nuevo
    ultimoEBR.EstablecerSiguienteEBR(nuevoInicioEBR)
    err = ultimoEBR.Codificar(archivo, int64(ultimoEBR.Ebr_start))
    if err != nil {
        return fmt.Errorf("error al actualizar el EBR anterior: %v", err)
    }

    fmt.Fprintln(bufferSalida, "Particion logica construida correctamente.")
    return nil
}