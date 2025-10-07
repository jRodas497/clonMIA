package Estructuras

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "os"
    "strings"
    "time"

	Utils "backend/Utils"
)

// Número máximo de entradas de journal por defecto
const ENTRADAS_JOURNAL = 50

type Journal struct {
    J_count   int32
    J_content Informacion
}

type Informacion struct {
    I_operation [10]byte // 10 bytes - Tipo de operación (mkdir, mkfile, etc)
    I_path      [32]byte // 32 bytes - Ruta del recurso afectado
    I_content   [64]byte // 64 bytes - Contenido afectado
    I_date      uint32   // 4 bytes - Timestamp - Mejor precisión que float32
    // Total: 110 bytes
}

// Codificar guarda el journal en el archivo en la posición indicada
func (journal *Journal) Codificar(file *os.File, offset int64) error {
	// Debug: mostrar offset y conteo
	fmt.Printf("[DEBUG] Journal.Encode: count=%d, offset=%d\n", journal.J_count, offset)
	err := Utils.EscribirAArchivo(file, offset, journal)

	if err != nil {
		return fmt.Errorf("error al escribir el journal en el archivo: %w", err)
	}

	return nil
}

// Decodificar lee el journal desde el archivo en la posición indicada
func (journal *Journal) Decodificar(file *os.File, offset int64) error {
	// Debug: mostrar offset
	fmt.Printf("[DEBUG] Journal.Decode: offset=%d\n", offset)
	err := Utils.LeerDeArchivo(file, offset, journal)

	if err != nil {
		return fmt.Errorf("error al leer el journal del archivo: %w", err)
	}

	return nil
}

// Imprimir imprime la información del journal en formato legible
func (journal *Journal) Imprimir() {
    fecha := time.Unix(int64(journal.J_content.I_date), 0)
    fmt.Println("Journal:")
    fmt.Printf("J_count: %d\n", journal.J_count)
    fmt.Println("Informacion:")
    fmt.Printf("I_operation: %s\n", strings.TrimSpace(string(journal.J_content.I_operation[:])))
    fmt.Printf("I_path: %s\n", strings.TrimSpace(string(journal.J_content.I_path[:])))
    fmt.Printf("I_content: %s\n", strings.TrimSpace(string(journal.J_content.I_content[:])))
    fmt.Printf("I_date: %s\n", fecha.Format(time.RFC3339))
}

// CrearEntradaJournal crea una nueva entrada de journal en memoria
func (j *Journal) CrearEntradaJournal(operacion, ruta, contenido string) {
    *j = Journal{} // limpia todo
    copy(j.J_content.I_operation[:], operacion)
    copy(j.J_content.I_path[:], ruta)
    copy(j.J_content.I_content[:], contenido)
    j.J_content.I_date = uint32(time.Now().Unix())
}

// GenerarTablaJournal genera una tabla en formato dot para el journal
func (journal *Journal) GenerarTablaJournal(indiceJournal int32) string {
    fecha := time.Unix(int64(journal.J_content.I_date), 0).Format(time.RFC3339)
    operacion := strings.TrimSpace(string(journal.J_content.I_operation[:]))
    ruta := strings.TrimSpace(string(journal.J_content.I_path[:]))
    contenido := strings.TrimSpace(string(journal.J_content.I_content[:]))

    tabla := fmt.Sprintf(`journal_table_%d [label=<
        <TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="4">
            <TR>
                <TD COLSPAN="2" BGCOLOR="#4CAF50"><FONT COLOR="#FFFFFF">Entrada Journal %d</FONT></TD>
            </TR>
            <TR>
                <TD BGCOLOR="#FF7043">Operacion:</TD>
                <TD>%s</TD>
            </TR>
            <TR>
                <TD BGCOLOR="#FF7043">Ruta:</TD>
                <TD>%s</TD>
            </TR>
            <TR>
                <TD BGCOLOR="#FF7043">Contenido:</TD>
                <TD>%s</TD>
            </TR>
            <TR>
                <TD BGCOLOR="#FF7043">Fecha:</TD>
                <TD>%s</TD>
            </TR>
        </TABLE>
    >];`, indiceJournal, indiceJournal, operacion, ruta, contenido, fecha)

    return tabla
}

// GenerarGrafico genera el contenido del grafo de las entradas del Journal en formato DOT
func (journal *Journal) GenerarGrafico(inicioJournal int64, conteoJournal int32, file *os.File) (string, error) {
    contenidoDot := ""
    tamañoEntrada := int64(binary.Size(Journal{}))

    fmt.Println("Generando grafo de Journal...")

    for i := int32(0); i < conteoJournal; i++ {
        offset := inicioJournal + int64(i)*tamañoEntrada
        fmt.Printf("Leyendo entrada del Journal en offset: %d\n", offset)
        err := journal.Decodificar(file, offset)
        if err != nil {
            return "", fmt.Errorf("error al deserializar el journal %d en offset %d: %v", i, offset, err)
        }
        operacion := strings.TrimSpace(string(journal.J_content.I_operation[:]))
        if operacion == "" {
            fmt.Printf("Entrada de Journal vacía encontrada en índice %d, deteniendo la lectura.\n", i)
            break
        }
        fmt.Printf("Generando tabla para la entrada de Journal %d con operación: %s\n", i, operacion)
        contenidoDot += journal.GenerarTablaJournal(i)
    }

    return contenidoDot, nil
}

// GuardarEntradaJournal guarda una nueva entrada en el journal y la serializa en el archivo
func (journal *Journal) GuardarEntradaJournal(file *os.File, inicio_journaling int64, operacion string, ruta string, contenido string) error {
    journal.CrearEntradaJournal(operacion, ruta, contenido)
    // Calcular el offset correcto basado en J_count
    tamañoEntrada := int64(binary.Size(Journal{}))
    offset := inicio_journaling + int64(journal.J_count)*tamañoEntrada
    // Debug: mostrar cálculo de offset
    fmt.Printf("[DEBUG] GuardarEntradaJournal: index=%d, tamañoEntrada=%d, inicio_journaling=%d, offset=%d\n", journal.J_count, tamañoEntrada, inicio_journaling, offset)

    err := journal.Codificar(file, offset)
    if err != nil {
        return fmt.Errorf("error al guardar la entrada de journal: %w", err)
    }
    return nil
}

// CalcularEspacioJournaling calcula el espacio total necesario para el journaling
func CalcularEspacioJournaling(n int32) int64 {
    return int64(n) * int64(binary.Size(Journal{}))
}

// InicializarAreaJournal inicializa el área completa de journaling con entradas vacías
func InicializarAreaJournal(file *os.File, inicioJournal int64, n int32) error {
    fmt.Println("=== Inicializando área de journaling ===")

    tamañoEntrada := int64(binary.Size(Journal{}))
    finJournal := inicioJournal + tamañoEntrada*int64(n)

    fmt.Printf("[DEBUG] Rango Journal  : [%d, %d)  (%d slots · %d bytes c/u)\n",
        inicioJournal, finJournal, n, tamañoEntrada)

    // Plantilla vacía
    journalNulo := &Journal{
        J_content: Informacion{
            I_operation: [10]byte{},
            I_path:      [32]byte{},
            I_content:   [64]byte{},
            I_date:      0,
        },
    }

    for i := int32(0); i < n; i++ {
        journalNulo.J_count = i
        offset := inicioJournal + tamañoEntrada*int64(i)

        err := journalNulo.Codificar(file, offset)
        if err != nil {
            return fmt.Errorf("error inicializando journal slot %d (off %d): %w", i, offset, err)
        }
        fmt.Printf("[DEBUG] slot=%02d | off=%d | ok\n", i, offset)
    }

    fmt.Printf("Journal inicializado correctamente con %d entradas\n", n)
    return nil
}

// EncontrarEntradasJournalValidas busca y devuelve todas las entradas de journal válidas
func EncontrarEntradasJournalValidas(file *os.File, inicioJournal int64, entradasMaximas int32) ([]Journal, error) {
    var entradas []Journal
    var conteoValido int32 = 0
    tamañoEntrada := int64(binary.Size(Journal{}))

    fmt.Println("Buscando entradas válidas de journal...")

    // Definir las operaciones válidas
    operacionesValidas := map[string]bool{
        "mkdir": true, "mkfile": true, "rm": true, "rmdir": true,
        "edit": true, "cat": true, "rename": true, "copy": true,
    }

    // Debug: mostrar todas las entradas
    fmt.Println("Contenido actual del journal:")

    for i := int32(0); i < entradasMaximas; i++ {
        offset := inicioJournal + int64(i)*tamañoEntrada
        journal := &Journal{}

        if err := journal.Decodificar(file, offset); err != nil {
            fmt.Printf("Error leyendo journal en offset %d: %v\n", offset, err)
            break
        }

        // Extraer y limpiar la operación
        operacionRaw := journal.J_content.I_operation[:]
        // Encontrar el primer byte nulo
        posicionNula := 0
        for ; posicionNula < len(operacionRaw); posicionNula++ {
            if operacionRaw[posicionNula] == 0 {
                break
            }
        }
        operacion := string(operacionRaw[:posicionNula])
        operacion = strings.TrimSpace(operacion)

        // Mostrar cada entrada
        fmt.Printf("-- Entrada %d --\n", i)
        journal.Imprimir()

        // Si no hay operación, llegamos al final
        if operacion == "" {
            break
        }

        // fechaUnix := float64(journal.J_content.I_date)
        // fecha := time.Unix(int64(fechaUnix), 0)
        
        fechaUnix := int64(journal.J_content.I_date)
        fecha := time.Unix(fechaUnix, 0)

        // Mostrar detalles de validación
        fmt.Printf("[DEBUG] - Validando entrada %d: operación='%s', fechaUnix=%d, fecha=%s\n",
            i, operacion, fechaUnix, fecha.Format(time.RFC3339))

        // Verificar si la operación es válida - ahora con cadenas correctamente preparadas
        if _, ok := operacionesValidas[operacion]; !ok {
            fmt.Printf("[DEBUG] - Entrada %d rechazada: operación '%s' no válida\n", i, operacion)
            continue
        }

        // La entrada pasó todas las validaciones
        fmt.Printf("[DEBUG] - Entrada %d ACEPTADA ✓\n", i)
        entradas = append(entradas, *journal)
        conteoValido++
    }

    fmt.Printf("Se encontraron %d entradas válidas de journal\n", conteoValido)
    return entradas, nil
}

// EsJournalVacio verifica si una entrada de journal está vacía
func EsJournalVacio(j *Journal) bool {
    // Sólo chequear si la operación está vacía
    op := strings.TrimSpace(
        string(bytes.TrimRight(j.J_content.I_operation[:], "\x00")),
    )
    return op == ""
}

func AgregarEntradaJournal(file *os.File, inicioJournal int64, entradasMaximas int32, operacion string, ruta string, contenido string, sb *SuperBlock) error {
    // NUEVO: Verificación de consistencia usando el superbloque
    inicioEsperado := int64(sb.InicioJournal())
    if inicioJournal != inicioEsperado {
        fmt.Printf("⚠️ Advertencia: Ajustando inicio del journal de %d a %d\n",
            inicioJournal, inicioEsperado)
        inicioJournal = inicioEsperado
    }

    // Resto del código como estaba...
    siguienteIndice, err := ObtenerSiguienteIndiceJournalVacio(file, inicioJournal, entradasMaximas)
    if err != nil {
        return fmt.Errorf("error buscando el siguiente índice disponible: %w", err)
    }

    if siguienteIndice >= entradasMaximas {
        fmt.Printf("Journal lleno, sobreescribiendo desde el principio (índice 0)\n")
        siguienteIndice = 0
    }

    journal := &Journal{
        J_count: siguienteIndice,
    }

    journal.CrearEntradaJournal(operacion, ruta, contenido)
    offset := inicioJournal + int64(siguienteIndice)*int64(binary.Size(Journal{}))

    // Verificar límites usando el superbloque
    finJournal := int64(sb.FinJournal())
    if offset >= finJournal {
        return fmt.Errorf("error: intento de escritura fuera del área de journal (%d >= %d)",
            offset, finJournal)
    }

    // Usar Codificar para ser consistentes
    if err := journal.Codificar(file, offset); err != nil {
        return fmt.Errorf("error escribiendo nueva entrada de journal: %w", err)
    }

    if err := file.Sync(); err != nil {
        return fmt.Errorf("error sincronizando archivo: %w", err)
    }

    fmt.Printf("Nueva entrada de journal agregada en índice %d: %s %s\n",
        siguienteIndice, operacion, ruta)

    return nil
}

// Nueva función para encontrar el próximo índice realmente vacío
func ObtenerSiguienteIndiceJournalVacio(file *os.File, inicioJournal int64, entradasMaximas int32) (int32, error) {
    tamañoEntrada := int64(binary.Size(Journal{}))

    fmt.Printf("[DEBUG] ⟶ Escanear Journal | inicio=%d | slots=%d | tamañoEntrada=%d bytes\n",
        inicioJournal, entradasMaximas, tamañoEntrada)

    for i := int32(0); i < entradasMaximas; i++ {
        offset := inicioJournal + tamañoEntrada*int64(i)

        j := &Journal{}
        if err := j.Decodificar(file, offset); err != nil {
            return -1, fmt.Errorf("leer journal[%d] en off %d: %w", i, offset, err)
        }

        op := strings.TrimSpace(string(j.J_content.I_operation[:]))
        fecha := j.J_content.I_date

        fmt.Printf("[DEBUG] slot=%02d | off=%d | op='%s' | fecha=%d\n", i, offset, op, fecha)

        // Reutilizamos la lógica centralizada:
        if EsJournalVacio(j) {
            fmt.Printf("[DEBUG] ⇒ slot libre encontrado en índice %d\n", i)
            return i, nil
        }
    }

    fmt.Println("[DEBUG] Journal lleno: se usará sobreescritura circular (idx 0)")
    return 0, nil
}