package Reports

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	Estructuras "backend/Estructuras"
	Utils "backend/Utils"
)

// Genera un reporte del bitmap de inodos
func ReporteBMInodo(sb *Estructuras.SuperBlock, rutaDisco string, ruta string) error {
	err := Utils.CrearDirectoriosPadre(ruta)
	if err != nil {
		return fmt.Errorf("error creando carpetas padre: %v", err)
	}

	archivo, err := os.Open(rutaDisco)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}
	defer archivo.Close()

	totalInodos := sb.S_inodes_count + sb.S_free_inodes_count
	byteCount := (totalInodos + 7) / 8

	var contenido strings.Builder

	for byteIndex := int32(0); byteIndex < byteCount; byteIndex++ {
		_, err := archivo.Seek(int64(sb.S_bm_inode_start+byteIndex), 0)
		if err != nil {
			return fmt.Errorf("error al posicionar el archivo: %v", err)
		}

		var byteVal byte
		err = binary.Read(archivo, binary.LittleEndian, &byteVal)
		if err != nil {
			return fmt.Errorf("error al leer el byte del bitmap: %v", err)
		}

		for bitOffset := 0; bitOffset < 8; bitOffset++ {
			if byteIndex*8+int32(bitOffset) >= totalInodos {
				break
			}
			if (byteVal & (1 << bitOffset)) != 0 {
				contenido.WriteByte('1') } else {
				contenido.WriteByte('0')
			}
			if (byteIndex*8+int32(bitOffset)+1)%20 == 0 {
				contenido.WriteString("\n")
			}
		}
	}

	txtFile, err := os.Create(ruta)
	if err != nil {
		return fmt.Errorf("error al crear el archivo de reporte: %v", err)
	}
	defer txtFile.Close()

	_, err = txtFile.WriteString(contenido.String())
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo de reporte: %v", err)
	}

	fmt.Println("Reporte del bitmap de inodos generado:", ruta)
	return nil
}
