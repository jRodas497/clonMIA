package Reports

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	Estructuras "backend/Estructuras"
	Utils "backend/Utils"
)

// Genera el reporte visual del MBR
func ReporteMBR(mbr *Estructuras.MBR, ruta string, archivo *os.File) error {
	// Crea carpetas si faltan
	err := Utils.CrearDirectoriosPadre(ruta)
	if err != nil {
		return err
	}

	// Nombres de archivos dot y png
	dotFileName, outputImage := Utils.ObtenerNombresArchivos(ruta)

	colorPrimaria := "#B0B8C1"   // Primarias
	colorExtendida := "#A7A9AC"  // Extendidas
	colorLogica := "#5B7FA3"     // Lógicas
	colorEBR := "#D9D9D9"        // EBR
	colorNoAsignado := "#F2F2F2" // No asignados

	// Arma el DOT
	dotContent := fmt.Sprintf(`digraph G {
        node [shape=plaintext]
        tabla [label=<
            <table border="0" cellborder="1" cellspacing="0">
                <tr><td colspan="2" bgcolor="#F8D7DA"><b>REPORTE MBR</b></td></tr>
                <tr><td bgcolor="#F5B7B1">mbr_tamano</td><td bgcolor="#F5B7B1">%d</td></tr>
                <tr><td bgcolor="#F5B7B1">mbr_fecha_creacion</td><td bgcolor="#F5B7B1">%s</td></tr>
                <tr><td bgcolor="#F5B7B1">mbr_disk_signature</td><td bgcolor="#F5B7B1">%d</td></tr>
            `, mbr.MbrSize, time.Unix(int64(mbr.MbrCreacionDate), 0), mbr.MbrDiskSignature)

	tamanoTotal := mbr.MbrSize
	tamanoAsignado := int32(0)

	// Recorre particiones
	for i, part := range mbr.MbrPartitions {
		if part.Part_size > 0 && part.Part_start > 0 {
			// Espacio libre antes
			if part.Part_start > tamanoAsignado {
				tamanoNoAsignado := part.Part_start - tamanoAsignado
				dotContent += fmt.Sprintf(`
                    <tr><td colspan="2" bgcolor="%s"><b>ESPACIO NO ASIGNADO (Tamaño: %d bytes)</b></td></tr>
                `, colorNoAsignado, tamanoNoAsignado)
				tamanoAsignado += tamanoNoAsignado
			}

			// Datos de la partición
			nombrePart := strings.TrimRight(string(part.Part_name[:]), "\x00")
			estadoPart := rune(part.Part_status[0])
			tipoPart := rune(part.Part_type[0])
			ajustePart := rune(part.Part_fit[0])

			// Color según tipo
			colorFila := ""
			switch tipoPart {
			case 'P':
				colorFila = colorPrimaria
			case 'E':
				colorFila = colorExtendida
			}

			// Agrega datos de partición
			dotContent += fmt.Sprintf(`
                <tr><td colspan="2" bgcolor="%s"><b>PARTICIÓN %d</b></td></tr>
                <tr><td bgcolor="%s">part_status</td><td bgcolor="%s">%c</td></tr>
                <tr><td bgcolor="%s">part_type</td><td bgcolor="%s">%c</td></tr>
                <tr><td bgcolor="%s">part_fit</td><td bgcolor="%s">%c</td></tr>
                <tr><td bgcolor="%s">part_start</td><td bgcolor="%s">%d</td></tr>
                <tr><td bgcolor="%s">part_size</td><td bgcolor="%s">%d</td></tr>
                <tr><td bgcolor="%s">part_name</td><td bgcolor="%s">%s</td></tr>
            `, colorFila, i+1,
				colorFila, colorFila, estadoPart,
				colorFila, colorFila, tipoPart,
				colorFila, colorFila, ajustePart,
				colorFila, colorFila, part.Part_start,
				colorFila, colorFila, part.Part_size,
				colorFila, colorFila, nombrePart)

			tamanoAsignado += part.Part_size

			// Si es extendida, recorre EBRs
			if tipoPart == 'E' {
				inicioEBR := part.Part_start
				dotContent += fmt.Sprintf(`
                    <tr><td colspan="2" bgcolor="%s"><b>PART. EXTENDIDA (Inicio: %d)</b></td></tr>
                `, colorExtendida, inicioEBR)

				// Recorre EBRs
				for inicioEBR != -1 {

					ebr := &Estructuras.EBR{}
					err := ebr.Decodificar(archivo, int64(inicioEBR))

					if err != nil {
						return fmt.Errorf("error al leer EBR: %v", err)
					}
					nombreEBR := strings.TrimRight(string(ebr.Ebr_name[:]), "\x00")
					ajusteEBR := rune(ebr.Ebr_fit[0])

					// Datos del EBR
					dotContent += fmt.Sprintf(`
                        <tr><td colspan="2" bgcolor="%s"><b>EBR (Inicio: %d)</b></td></tr>
                        <tr><td bgcolor="%s">ebr_fit</td><td bgcolor="%s">%c</td></tr>
                        <tr><td bgcolor="%s">ebr_start</td><td bgcolor="%s">%d</td></tr>
                        <tr><td bgcolor="%s">ebr_size</td><td bgcolor="%s">%d</td></tr>
                        <tr><td bgcolor="%s">ebr_next</td><td bgcolor="%s">%d</td></tr>
                        <tr><td bgcolor="%s">ebr_name</td><td bgcolor="%s">%s</td></tr>
                    `, colorEBR, inicioEBR,
						colorEBR, colorEBR, ajusteEBR,
						colorEBR, colorEBR, ebr.Ebr_start,
						colorEBR, colorEBR, ebr.Ebr_size,
						colorEBR, colorEBR, ebr.Ebr_next,
						colorEBR, colorEBR, nombreEBR)

					// Si hay lógica tras EBR
					if ebr.Ebr_size > 0 {
						dotContent += fmt.Sprintf(`
                            <tr><td colspan="2" bgcolor="%s"><b>PART. LÓGICA (Inicio: %d)</b></td></tr>
                        `, colorLogica, ebr.Ebr_start)
					}
					tamanoAsignado += ebr.Ebr_size
					inicioEBR = int32(ebr.Ebr_next)
				}
			}
		}
	}

	// Espacio libre al final
	if tamanoAsignado < tamanoTotal {
		tamanoNoAsignado := tamanoTotal - tamanoAsignado
		dotContent += fmt.Sprintf(`
            <tr><td colspan="2" bgcolor="%s"><b>ESPACIO NO ASIGNADO (Tamaño: %d bytes)</b></td></tr>
        `, colorNoAsignado, tamanoNoAsignado)
	}

	dotContent += "</table>>] }"

	// Guarda dot
	archivo, err = os.Create(dotFileName)
	if err != nil {
		return fmt.Errorf("error al crear el archivo: %v", err)
	}
	defer archivo.Close()

	_, err = archivo.WriteString(dotContent)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo: %v", err)
	}

	// Ejecuta Graphviz
	cmd := exec.Command("dot", "-Tpng", dotFileName, "-o", outputImage)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}

	fmt.Println("Imagen de la tabla generada:", outputImage)
	return nil
}
