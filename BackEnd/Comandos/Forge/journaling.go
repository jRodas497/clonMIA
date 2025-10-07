package Forge

import (
	Estructuras "backend/Estructuras"
	Global "backend/Global"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type ComandoJournaling struct {
	Id string `json:"id"`
}

type EntradaJournal struct {
	Operacion string `json:"operacion"`
	Ruta      string `json:"ruta"`
	Contenido string `json:"contenido"`
	Fecha     string `json:"fecha"`
}

// limpiarCadenaC elimina bytes nulos al final de un arreglo (cadena estilo C)
func limpiarCadenaC(buf []byte) string {
	return strings.TrimSpace(
		string(bytes.TrimRight(buf, "\x00")),
	)
}

// Execute ejecuta el comando journaling que muestra todas las transacciones realizadas
func (cmd *ComandoJournaling) Execute() (interface{}, error) {
	// Validar que el ID sea proporcionado
	if cmd.Id == "" {
		return nil, errors.New("el parámetro id es obligatorio")
	}

	// Obtener el superbloque de la partición
	sb, particion, ruta, err := Global.GetMountedPartitionSuperblock(cmd.Id)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo la partición: %w", err)
	}

	// Verificar que la partición sea de tipo EXT3 (con journaling)
	if sb.S_filesystem_type != 3 {
		return nil, errors.New("la partición no es de tipo EXT3, no tiene journaling")
	}

	// Abrir el archivo para lectura
	archivo, err := os.OpenFile(ruta, os.O_RDONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("error abriendo el archivo: %w", err)
	}
	defer archivo.Close()

	// Calcular la posición de inicio del journal
	inicioJournal := int64(particion.Part_start) + int64(binary.Size(Estructuras.SuperBlock{}))

	fmt.Printf("Leyendo journal desde posición %d\n", inicioJournal)

	// Buscar todas las entradas válidas del journal
	entradas, err := Estructuras.EncontrarEntradasJournalValidas(archivo, inicioJournal, Estructuras.ENTRADAS_JOURNAL)
	if err != nil {
		return nil, fmt.Errorf("error buscando entradas de journal: %w", err)
	}

	// Si no hay entradas, devolver un mensaje
	if len(entradas) == 0 {
		return "No hay entradas de journal para mostrar", nil
	}

	// Convertir las entradas a un formato más amigable para la interfaz
	var resultado []EntradaJournal
	for _, entrada := range entradas {
		operacion := limpiarCadenaC(entrada.J_content.I_operation[:])
		ruta := limpiarCadenaC(entrada.J_content.I_path[:])
		contenido := limpiarCadenaC(entrada.J_content.I_content[:])

		fecha := time.Unix(int64(entrada.J_content.I_date), 0)
		fechaStr := fecha.Format(time.RFC3339)

		resultado = append(resultado, EntradaJournal{
			Operacion: operacion,
			Ruta:      ruta,
			Contenido: contenido,
			Fecha:     fechaStr,
		})
	}

	fmt.Printf("Se encontraron %d entradas válidas de journal\n", len(resultado))

	// Generar y devolver tabla de texto
	return cmd.GenerarTablaJournaling(resultado)
}

// ParserJournaling procesa los argumentos del comando journaling y ejecuta la acción
func ParserJournaling(argumentos []string) (interface{}, error) {
	// Inicializar el comando
	cmd := &ComandoJournaling{}

	// Verificar que haya argumentos
	if len(argumentos) == 0 {
		return nil, errors.New("no se proporcionaron parámetros para el comando journaling")
	}

	// Procesar argumentos
	for _, argumento := range argumentos {
		if !strings.HasPrefix(argumento, "-") {
			continue // Ignorar argumentos que no empiecen con -
		}

		partes := strings.SplitN(strings.TrimPrefix(argumento, "-"), "=", 2)
		if len(partes) != 2 {
			return nil, fmt.Errorf("formato de parámetro incorrecto: %s", argumento)
		}

		parametro, valor := strings.ToLower(partes[0]), partes[1]

		// Eliminar comillas si existen
		if strings.HasPrefix(valor, "\"") && strings.HasSuffix(valor, "\"") {
			valor = valor[1 : len(valor)-1]
		}

		switch parametro {
		case "id":
			cmd.Id = valor
		default:
			return nil, fmt.Errorf("parámetro desconocido: %s", parametro)
		}
	}

	// Validar parámetros obligatorios
	if cmd.Id == "" {
		return nil, errors.New("el parámetro id es obligatorio")
	}

	// Ejecutar el comando
	return cmd.Execute()
}

// GenerarTablaJournaling genera una representación legible del journal
func (cmd *ComandoJournaling) GenerarTablaJournaling(entradas []EntradaJournal) (string, error) {
	if len(entradas) == 0 {
		return "No hay entradas válidas de journal para mostrar", nil
	}

	// Anchuras de columna
	const (
		anchoRuta  = 28
		anchoFecha = 19
	)

	// Cabecera y divisor
	encabezado := fmt.Sprintf("%-5s | %-10s | %-*s | %-*s | %s\n",
		"NO.", "OPERACIÓN", anchoRuta, "RUTA", anchoFecha, "FECHA", "CONTENIDO")
	divisor := strings.Repeat("-", len(encabezado)-1) + "\n"

	// Construir tabla
	var tabla strings.Builder
	tabla.WriteString("\nREGISTRO DE TRANSACCIONES (JOURNAL)\n")
	tabla.WriteString(strings.Repeat("=", 34) + "\n\n")
	tabla.WriteString(encabezado)
	tabla.WriteString(divisor)

	for i, e := range entradas {
		// Truncar contenido largo
		contenido := e.Contenido
		if len(contenido) > 40 {
			contenido = contenido[:37] + "..."
		}

		// Formatear fecha
		fecha := e.Fecha
		if t, err := time.Parse(time.RFC3339, fecha); err == nil {
			fecha = t.Format("02/01/2006 15:04:05")
		}

		fila := fmt.Sprintf("%-5d | %-10s | %-*s | %-*s | %s\n",
			i+1,                        // NO.
			e.Operacion,                // OPERACIÓN
			anchoRuta, e.Ruta,          // RUTA
			anchoFecha, fecha,          // FECHA
			contenido)                  // CONTENIDO

		tabla.WriteString(fila)
	}

	return tabla.String(), nil
}
