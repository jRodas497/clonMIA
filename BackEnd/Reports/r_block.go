package Reports

import (
	"fmt"
	"html"
	"os"
	"os/exec"
	"strings"

	Estructuras "backend/Estructuras"
	Utils "backend/Utils"
)

// Genera un reporte visual de bloques y conexiones usando Graphviz
func ReporteBloques(sb *Estructuras.SuperBlock, rutaDisco string, ruta string) error {
	err := Utils.CrearDirectoriosPadre(ruta)
	if err != nil {
		return fmt.Errorf("error al crear directorios: %v", err)
	}
	archivo, err := os.Open(rutaDisco)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de disco: %v", err)
	}
	defer archivo.Close()
	nombreDot, nombreImagen := Utils.ObtenerNombresArchivos(ruta)

	dot := iniciarDotBloques()
	dot, conexiones, err := graficarBloques(dot, sb, archivo)
	if err != nil {
		return err
	}
	dot += conexiones
	dot += "}"

	err = escribirArchivoDot(nombreDot, dot)
	if err != nil {
		return err
	}
	err = generarImagenBloques(nombreDot, nombreImagen)
	if err != nil {
		return err
	}

	fmt.Println("Imagen de bloques generada:", nombreImagen)
	return nil
}

func iniciarDotBloques() string {
	return `digraph G {
        fontname="Helvetica,Arial,sans-serif"
        node [fontname="Helvetica,Arial,sans-serif", shape=box, fontsize=12];
        edge [fontname="Helvetica,Arial,sans-serif", color="#FF7043", arrowsize=0.8];
        rankdir=LR;
        bgcolor="#FAFAFA";
    `
}

func graficarBloques(dot string, sb *Estructuras.SuperBlock, archivo *os.File) (string, string, error) {
	visitados := make(map[int32]bool)
	var conexiones string
	for i := int32(0); i < sb.S_inodes_count; i++ {
		inodo := &Estructuras.INodo{}
		err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(i*sb.S_inode_size)))
		if err != nil {
			return "", "", fmt.Errorf("error al leer inodo %d: %v", i, err)
		}
		if inodo.I_uid == -1 || inodo.I_uid == 0 {
			continue
		}
		for _, bloque := range inodo.I_block {
			if bloque != -1 && !visitados[bloque] {
				dot, conexiones, err = etiquetaBloque(dot, conexiones, bloque, inodo, sb, archivo, visitados)
				if err != nil {
					return "", "", err
				}
				visitados[bloque] = true
			}
		}
	}
	return dot, conexiones, nil
}

func etiquetaBloque(dot, conexiones string, idx int32, inodo *Estructuras.INodo, sb *Estructuras.SuperBlock, archivo *os.File, visitados map[int32]bool) (string, string, error) {
	offset := int64(sb.S_block_start + (idx * sb.S_block_size))
	if inodo.I_type[0] == '0' {
		bloqueCarpeta := &Estructuras.FolderBlock{}
		err := bloqueCarpeta.Decodificar(archivo, offset)
		if err != nil {
			return "", "", fmt.Errorf("error al decodificar bloque de carpeta %d: %w", idx, err)
		}
		etiqueta := fmt.Sprintf("BLOQUE CARPETA %d", idx)
		tieneConexiones := false
		for i, contenido := range bloqueCarpeta.B_cont {
			nombre := limpiarNombreBloque(contenido.B_name)
			nombre = html.EscapeString(nombre)
			if contenido.B_inodo != -1 && !(i == 0 || i == 1) {
				etiqueta += fmt.Sprintf("\\nContenido %d: %s (Inodo %d)", i+1, nombre, contenido.B_inodo)
				if contenido.B_inodo != idx {
					conexiones += fmt.Sprintf("bloque%d -> bloque%d [color=\"#FF7043\"]\n", idx, contenido.B_inodo)
				}
				tieneConexiones = true
			} else if i > 1 {
				etiqueta += fmt.Sprintf("\\nContenido %d: %s (Inodo no asignado)", i+1, nombre)
			}
		}
		if tieneConexiones {
			dot += fmt.Sprintf("bloque%d [label=\"%s\", shape=box, style=filled, fillcolor=\"#FFFDE7\", color=\"#EEEEEE\"]\n", idx, etiqueta)
		}
	} else if inodo.I_type[0] == '1' {
		bloqueArchivo := &Estructuras.FileBlock{}
		err := bloqueArchivo.Decodificar(archivo, offset)
		if err != nil {
			return "", "", fmt.Errorf("error al decodificar bloque de archivo %d: %w", idx, err)
		}
		contenido := limpiarContenidoBloque(bloqueArchivo.ObtenerContenido())
		if len(strings.TrimSpace(contenido)) > 0 {
			etiqueta := fmt.Sprintf("BLOQUE ARCHIVO %d\\n%s", idx, contenido)
			dot += fmt.Sprintf("bloque%d [label=\"%s\", shape=box, style=filled, fillcolor=\"#FFFDE7\", color=\"#EEEEEE\"]\n", idx, etiqueta)
			siguiente := buscarSiguienteBloque(inodo, idx)
			if siguiente != -1 {
				conexiones += fmt.Sprintf("bloque%d -> bloque%d [color=\"#FF7043\"]\n", idx, siguiente)
			}
		}
	}
	padre := buscarBloquePadre(inodo, idx)
	if padre != -1 {
		conexiones += fmt.Sprintf("bloque%d -> bloque%d [color=\"#FF7043\"]\n", padre, idx)
	}
	return dot, conexiones, nil
}

func buscarBloquePadre(inodo *Estructuras.INodo, actual int32) int32 {
	for i := 0; i < len(inodo.I_block); i++ {
		if inodo.I_block[i] == actual && i > 0 {
			return inodo.I_block[i-1]
		}
	}
	return -1
}

func buscarSiguienteBloque(inodo *Estructuras.INodo, actual int32) int32 {
	for i := 0; i < len(inodo.I_block); i++ {
		if inodo.I_block[i] == actual {
			for j := i + 1; j < len(inodo.I_block); j++ {
				if inodo.I_block[j] != -1 {
					return inodo.I_block[j]
				}
			}
		}
	}
	return -1
}

func limpiarNombreBloque(nombre [12]byte) string {
	return strings.TrimRight(string(nombre[:]), "\x00")
}

func limpiarContenidoBloque(contenido string) string {
	return strings.ReplaceAll(contenido, "\n", "\\n")
}

func generarImagenBloques(nombreDot string, nombreImagen string) error {
	cmd := exec.Command("dot", "-Tpng", nombreDot, "-o", nombreImagen)

	err := cmd.Run()

	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}
	return nil
}
