package Reports

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	Utils "backend/Utils"
	Estructuras "backend/Estructuras"
)

// Genera un reporte visual de la estructura del disco y lo guarda en la ruta indicada
func ReporteDisco(mbr *Estructuras.MBR, ruta string, rutaDisco string) error {
	// Crear carpetas padre si no existen
	err := Utils.CrearDirectoriosPadre(ruta)
	if err != nil {
		return fmt.Errorf("error al crear directorios: %v", err)
	}

	// Abrir archivo de disco
	archivo, err := os.Open(rutaDisco)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}
	defer archivo.Close()

	// Obtener nombres base de archivo sin extensión
	nombreDot, nombreImagen := Utils.ObtenerNombresArchivos(ruta)

	dot := `digraph G {
		fontname="Helvetica,Arial,sans-serif"
		node [fontname="Helvetica,Arial,sans-serif"]
		edge [fontname="Helvetica,Arial,sans-serif"]
		concentrate=True;
		rankdir=TB;
		node [shape=record];

		titulo [label="Reporte DISCO" shape=plaintext fontname="Helvetica,Arial,sans-serif"];

		disco [label="`

	// Calcular tamaño total y usado
	tamTotal := mbr.MbrSize
	tamUsado := int32(0)

	dot += "{MBR}"

	for _, part := range mbr.MbrPartitions {
		if part.Part_size > 0 {
			porcentaje := (float64(part.Part_size) / float64(tamTotal)) * 100
			tamUsado += part.Part_size

			nombrePart := strings.TrimRight(string(part.Part_name[:]), "\x00")
			if part.Part_type[0] == 'P' {
				// Partición primaria
				dot += fmt.Sprintf("|{Primaria %s\\n%.2f%%}", nombrePart, porcentaje)
			} else if part.Part_type[0] == 'E' {
				// Partición extendida
				dot += fmt.Sprintf("|{Extendida %.2f%%|{", porcentaje)
				inicioEBR := part.Part_start
				contEBR := 0
				tamUsadoEBR := int32(0)

				// Leer EBRs usando método Decode
				for inicioEBR != -1 {
					ebr := &Estructuras.EBR{}
					err := ebr.Decodificar(archivo, int64(inicioEBR))
					if err != nil {
						return fmt.Errorf("error al decodificar EBR: %v", err)
					}

					nombreEBR := strings.TrimRight(string(ebr.Ebr_name[:]), "\x00")
					porcEBR := (float64(ebr.Ebr_size) / float64(tamTotal)) * 100
					tamUsadoEBR += ebr.Ebr_size

					if contEBR > 0 {
						dot += "|"
					}
					dot += fmt.Sprintf("{EBR|Lógica %s\\n%.2f%%}", nombreEBR, porcEBR)

					inicioEBR = ebr.Ebr_next
					contEBR++
				}

				// Espacio libre dentro de extendida
				tamLibreExt := part.Part_size - tamUsadoEBR
				if tamLibreExt > 0 {
					porcLibreExt := (float64(tamLibreExt) / float64(tamTotal)) * 100
					dot += fmt.Sprintf("|Libre %.2f%%", porcLibreExt)
				}

				dot += "}}"
			}
		}
	}

	// Espacio libre restante
	tamLibre := tamTotal - tamUsado
	if tamLibre > 0 {
		porcLibre := (float64(tamLibre) / float64(tamTotal)) * 100
		dot += fmt.Sprintf("|Libre %.2f%%", porcLibre)
	}

	// Cerrar nodo y DOT
	dot += `"];

		titulo -> disco [style=invis];
	}`

	// Crear archivo DOT
	archivoDot, err := os.Create(nombreDot)
	if err != nil {
		return fmt.Errorf("error al crear el archivo DOT: %v", err)
	}
	defer archivoDot.Close()

	_, err = archivoDot.WriteString(dot)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo DOT: %v", err)
	}

	// Generar imagen con Graphviz
	cmd := exec.Command("dot", "-Tpng", nombreDot, "-o", nombreImagen)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}

	fmt.Println("Reporte de disco generado:", nombreImagen)
	return nil
}
