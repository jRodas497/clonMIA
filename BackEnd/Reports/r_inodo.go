package Reports

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	Estructuras "backend/Estructuras"
	Utils "backend/Utils"
)

// Genera un reporte visual de los inodos
func ReporteInodos(sb *Estructuras.SuperBlock, rutaDisco string, ruta string) error {
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

	dot := iniciarDotInodos()

	if sb.S_inodes_count == 0 {
		return fmt.Errorf("no hay inodos en el sistema")
	}

	dot, err = graficarInodos(dot, sb, archivo)
	if err != nil {
		return err
	}

	dot += "}"

	err = escribirArchivoDot(nombreDot, dot)
	if err != nil {
		return err
	}

	err = generarImagenInodos(nombreDot, nombreImagen)
	if err != nil {
		return err
	}

	fmt.Println("Imagen de inodos generada:", nombreImagen)
	return nil
}

func iniciarDotInodos() string {
	return `digraph G {
		fontname="Helvetica,Arial,sans-serif"
		node [fontname="Helvetica,Arial,sans-serif", shape=plain, fontsize=12];
		edge [fontname="Helvetica,Arial,sans-serif", color="#FF7043", arrowsize=0.8];
		rankdir=LR;
		bgcolor="#FAFAFA";
		node [shape=plaintext];
	`
}

func graficarInodos(dot string, sb *Estructuras.SuperBlock, archivo *os.File) (string, error) {
	for i := int32(0); i < sb.S_inodes_count; i++ {
		inodo := &Estructuras.INodo{}
		err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(i*sb.S_inode_size)))
		if err != nil {
			return "", fmt.Errorf("error al leer inodo %d: %v", i, err)
		}
		if inodo.I_uid == -1 || inodo.I_uid == 0 {
			continue
		}
		dot += tablaInodo(i, inodo)
		if i < sb.S_inodes_count-1 {
			dot += fmt.Sprintf("inodo%d -> inodo%d [color=\"#FF7043\"]\n", i, i+1)
		}
	}
	return dot, nil
}

func tablaInodo(idx int32, inodo *Estructuras.INodo) string {
	atime := time.Unix(int64(inodo.I_atime), 0).Format(time.RFC3339)
	ctime := time.Unix(int64(inodo.I_ctime), 0).Format(time.RFC3339)
	mtime := time.Unix(int64(inodo.I_mtime), 0).Format(time.RFC3339)
	tabla := fmt.Sprintf(`inodo%d [label=<
		<table border="0" cellborder="1" cellspacing="0" cellpadding="4" bgcolor="#FFFDE7" style="rounded">
			<tr><td colspan="2" bgcolor="#4CAF50" align="center"><b>INODO %d</b></td></tr>
			<tr><td><b>uid</b></td><td>%d</td></tr>
			<tr><td><b>gid</b></td><td>%d</td></tr>
			<tr><td><b>size</b></td><td>%d</td></tr>
			<tr><td><b>atime</b></td><td>%s</td></tr>
			<tr><td><b>ctime</b></td><td>%s</td></tr>
			<tr><td><b>mtime</b></td><td>%s</td></tr>
			<tr><td><b>type</b></td><td>%c</td></tr>
			<tr><td><b>perm</b></td><td>%s</td></tr>
			<tr><td colspan="2" bgcolor="#FF9800"><b>BLOQUES DIRECTOS</b></td></tr>
	`, idx, idx, inodo.I_uid, inodo.I_gid, inodo.I_size, atime, ctime, mtime, rune(inodo.I_type[0]), string(inodo.I_perm[:]))
	for j, bloque := range inodo.I_block[:12] {
		if bloque != -1 {
			tabla += fmt.Sprintf("<tr><td><b>%d</b></td><td>%d</td></tr>", j+1, bloque)
		}
	}
	tabla += bloquesIndirectos(inodo)
	tabla += "</table>>];"
	return tabla
}

func bloquesIndirectos(inodo *Estructuras.INodo) string {
	res := ""
	if inodo.I_block[12] != -1 {
		res += fmt.Sprintf(`
			<tr><td colspan="2" bgcolor="#FF9800"><b>INDIRECTO SIMPLE</b></td></tr>
			<tr><td><b>13</b></td><td>%d</td></tr>
		`, inodo.I_block[12])
	}
	if inodo.I_block[13] != -1 {
		res += fmt.Sprintf(`
			<tr><td colspan="2" bgcolor="#FF9800"><b>INDIRECTO DOBLE</b></td></tr>
			<tr><td><b>14</b></td><td>%d</td></tr>
		`, inodo.I_block[13])
	}
	if inodo.I_block[14] != -1 {
		res += fmt.Sprintf(`
			<tr><td colspan="2" bgcolor="#FF9800"><b>INDIRECTO TRIPLE</b></td></tr>
			<tr><td><b>15</b></td><td>%d</td></tr>
		`, inodo.I_block[14])
	}
	return res
}

func escribirArchivoDot(nombreDot string, dot string) error {
	archivo, err := os.Create(nombreDot)
	if err != nil {
		return fmt.Errorf("error al crear el archivo DOT: %v", err)
	}
	defer archivo.Close()
	_, err = archivo.WriteString(dot)
	if err != nil {
		return fmt.Errorf("error al escribir en el archivo DOT: %v", err)
	}
	return nil
}

func generarImagenInodos(nombreDot string, nombreImagen string) error {
	cmd := exec.Command("dot", "-Tpng", nombreDot, "-o", nombreImagen)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz: %v", err)
	}
	return nil
}
