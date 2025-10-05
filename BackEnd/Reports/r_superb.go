package Reports

import (
	"fmt"
	"os/exec"
	"time"

	Estructuras "backend/Estructuras"
	Utils "backend/Utils"
)

// Genera un reporte visual del superbloque como tabla
func ReporteSuperbloque(sb *Estructuras.SuperBlock, rutaDisco string, ruta string) error {
	err := Utils.CrearDirectoriosPadre(ruta)
	if err != nil {
		return fmt.Errorf("error al crear directorios: %v", err)
	}

	nombreDot, nombreImagen := Utils.ObtenerNombresArchivos(ruta)
	dot := dotSuperbloque(sb)

	err = escribirArchivoDot(nombreDot, dot)
	if err != nil {
		return err
	}

	err = generarImagenSuperbloque(nombreDot, nombreImagen)
	if err != nil {
		return err
	}

	fmt.Println("Imagen del superbloque generada:", nombreImagen)
	return nil
}

func generarImagenSuperbloque(nombreDot string, nombreImagen string) error {
	cmd := exec.Command("dot", "-Tpng", nombreDot, "-o", nombreImagen)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error al ejecutar Graphviz para superbloque: %v", err)
	}
	return nil
}

func dotSuperbloque(sb *Estructuras.SuperBlock) string {
	mtime := time.Unix(int64(sb.S_mtime), 0).Format(time.RFC3339)
	umtime := time.Unix(int64(sb.S_umtime), 0).Format(time.RFC3339)
	dot := `
		digraph G {
			fontname="Helvetica,Arial,sans-serif"
			node [fontname="Helvetica,Arial,sans-serif", shape=plain, fontsize=12];
			edge [fontname="Helvetica,Arial,sans-serif", color="#FF7043", arrowsize=0.8];
			bgcolor="#FAFAFA";
			rankdir=TB;

			superbloque [label=<
				<table border="0" cellborder="1" cellspacing="0" cellpadding="10" bgcolor="#FFF9C4" style="rounded">
					<tr><td colspan="2" bgcolor="#4CAF50" align="center"><b>SUPERBLOQUE</b></td></tr>
					<tr><td><b>Tipo de Sistema</b></td><td>%d</td></tr>
					<tr><td><b>Cantidad de Inodos</b></td><td>%d</td></tr>
					<tr><td><b>Cantidad de Bloques</b></td><td>%d</td></tr>
					<tr><td><b>Inodos Libres</b></td><td>%d</td></tr>
					<tr><td><b>Bloques Libres</b></td><td>%d</td></tr>
					<tr><td><b>Tamaño de Inodo</b></td><td>%d bytes</td></tr>
					<tr><td><b>Tamaño de Bloque</b></td><td>%d bytes</td></tr>
					<tr><td><b>Primer Inodo Libre</b></td><td>%d</td></tr>
					<tr><td><b>Primer Bloque Libre</b></td><td>%d</td></tr>
					<tr><td><b>Inicio Bitmap Inodos</b></td><td>%d</td></tr>
					<tr><td><b>Inicio Bitmap Bloques</b></td><td>%d</td></tr>
					<tr><td><b>Inicio Tabla Inodos</b></td><td>%d</td></tr>
					<tr><td><b>Inicio Tabla Bloques</b></td><td>%d</td></tr>
					<tr><td><b>Última Modificación</b></td><td>%s</td></tr>
					<tr><td><b>Último Montaje</b></td><td>%s</td></tr>
					<tr><td><b>Número de Montajes</b></td><td>%d</td></tr>
					<tr><td><b>Valor M</b></td><td>0x%x</td></tr>
				</table>>];
		}
	`
	return fmt.Sprintf(dot,
		sb.S_filesystem_type,
		sb.S_inodes_count,
		sb.S_blocks_count,
		sb.S_free_inodes_count,
		sb.S_free_blocks_count,
		sb.S_inode_size,
		sb.S_block_size,
		sb.S_first_ino,
		sb.S_first_blo,
		sb.S_bm_inode_start,
		sb.S_bm_block_start,
		sb.S_inode_start,
		sb.S_block_start,
		mtime,
		umtime,
		sb.S_mnt_count,
		sb.S_magic,
	)
}
