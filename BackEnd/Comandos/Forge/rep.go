package Forge

import (
	Global "backend/Global"
	Reportes "backend/Reports"
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// REP estructura que representa el comando rep con sus parametros
type REP struct {
	id              string // ID del disco
	ruta            string // Ruta del archivo de salida
	nombre          string // Nombre del reporte
	ruta_archivo_ls string // Ruta del archivo ls (opcional)
}

func ParserRep(tokens []string) (string, error) {
	var bufferSalida bytes.Buffer
	cmd := &REP{}
	argumentos := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-id=[^\s]+|-path="[^"]+"|-path=[^\s]+|-name=[^\s]+|-path_file_ls="[^"]+"|-path_file_ls=[^\s]+`)
	coincidencias := re.FindAllString(argumentos, -1)

	for _, coincidencia := range coincidencias {
		clavValor := strings.SplitN(coincidencia, "=", 2)
		if len(clavValor) != 2 {
			return "", fmt.Errorf("formato de parametro invalido: %s", coincidencia)
		}
		clave, valor := strings.ToLower(clavValor[0]), clavValor[1]
		if strings.HasPrefix(valor, "\"") && strings.HasSuffix(valor, "\"") {
			valor = strings.Trim(valor, "\"")
		}

		switch clave {
		case "-id":
			if valor == "" {
				return "", errors.New("el id no puede estar vacio")
			}
			cmd.id = valor
		case "-path":
			if valor == "" {
				return "", errors.New("la ruta no puede estar vacia")
			}
			cmd.ruta = valor
		case "-name":
			nombresValidos := []string{"mbr", "disk", "inode", "block", "bm_inode", "bm_block", "sb", "file", "ls"}
			if !contiene(nombresValidos, valor) {
				return "", errors.New("nombre invalido, debe ser uno de: mbr, disk, inode, block, bm_inode, bm_block, sb, file, ls")
			}
			cmd.nombre = valor
		case "-path_file_ls":
			cmd.ruta_archivo_ls = valor
		default:
			return "", fmt.Errorf("parametro desconocido: %s", clave)
		}
	}

	if cmd.id == "" || cmd.ruta == "" || cmd.nombre == "" {
		return "", errors.New("faltan parametros requeridos: -id, -path, -name")
	}

	err := comandoRep(cmd, &bufferSalida)
	if err != nil {
		return "", err
	}

	return bufferSalida.String(), nil
}

func contiene(lista []string, valor string) bool {
	for _, v := range lista {
		if v == valor {
			return true
		}
	}
	return false
}

func comandoRep(rep *REP, bufferSalida *bytes.Buffer) error {
	// Obtener datos de la particion montada
	mbrMontado, sbMontado, rutaDisco, err := Global.ObtenerParticionMontadaReporte(rep.id)
	if err != nil {
		return err
	}

	archivo, err := os.Open(rutaDisco)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}
	defer archivo.Close()

	fmt.Fprintf(bufferSalida, "Generando reporte '%s'...\n", rep.nombre)

	switch rep.nombre {
	case "mbr":
		err = Reportes.ReporteMBR(mbrMontado, rep.ruta, archivo)
	case "disk":
		err = Reportes.ReporteDisco(mbrMontado, rep.ruta, rutaDisco)
	case "inode":
		err = Reportes.ReporteInodos(sbMontado, rutaDisco, rep.ruta)
	case "block":
		err = Reportes.ReporteBloques(sbMontado, rutaDisco, rep.ruta)
	case "bm_inode":
		err = Reportes.ReporteBMInodo(sbMontado, rutaDisco, rep.ruta)
	case "bm_block":
		err = Reportes.ReporteBMBloque(sbMontado, rutaDisco, rep.ruta)
	case "sb":
		err = Reportes.ReporteSuperbloque(sbMontado, rutaDisco, rep.ruta)
	case "file":
		err = Reportes.ReporteArchivo(sbMontado, rutaDisco, rep.ruta, rep.ruta_archivo_ls)
    case "ls":
        err = Reportes.ReporteLs(sbMontado, rutaDisco, rep.ruta, rep.ruta_archivo_ls)
    case "tree":
        err = Reportes.ReporteTree(sbMontado, rutaDisco, rep.ruta)
	default:
		return fmt.Errorf("tipo de reporte no soportado: %s", rep.nombre)
	}

	if err != nil {
		fmt.Fprintf(bufferSalida, "Error generando reporte de %s: %v\n", rep.nombre, err)

		fmt.Printf("Error generando reporte de %s: %v\n", rep.nombre, err) // Depuración
	}

	fmt.Fprintf(bufferSalida, "Reporte '%s' generado exitosamente en: %s\n", rep.nombre, rep.ruta)
	return nil
}
