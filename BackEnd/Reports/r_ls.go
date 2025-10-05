package Reports

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	Estructuras "backend/Estructuras"
	Global "backend/Global"
	Utils "backend/Utils"
)

// Obtiene el nombre de usuario a partir del UID buscando en users.txt
func obtenerNombreUsuarioPorUID(sb *Estructuras.SuperBlock, archivo *os.File, uid int32) string {
	var inodoUsuarios Estructuras.INodo
	desplazamientoInodo := int64(sb.S_inode_start + int32(binary.Size(inodoUsuarios)))
	err := inodoUsuarios.Decodificar(archivo, desplazamientoInodo)
	if err != nil {
		return "root"
	}
	contenido, err := Global.LeerBloquesArchivo(archivo, sb, &inodoUsuarios)
	if err != nil {
		return "root"
	}
	lineas := strings.Split(contenido, "\n")
	for _, linea := range lineas {
		campos := strings.Split(linea, ",")
		if len(campos) == 5 && campos[1] == "U" {
			id, _ := strconv.Atoi(campos[0])
			if int32(id) == uid {
				return campos[3] // Nombre de usuario
			}
		}
	}
	return "root"
}

// Obtiene el nombre del grupo a partir del GID buscando en users.txt
func obtenerNombreGrupoPorGID(sb *Estructuras.SuperBlock, archivo *os.File, gid int32) string {
	var inodoUsuarios Estructuras.INodo
	desplazamientoInodo := int64(sb.S_inode_start + int32(binary.Size(inodoUsuarios)))
	err := inodoUsuarios.Decodificar(archivo, desplazamientoInodo)
	if err != nil {
		return "root"
	}
	contenido, err := Global.LeerBloquesArchivo(archivo, sb, &inodoUsuarios)
	if err != nil {
		return "root"
	}
	lineas := strings.Split(contenido, "\n")
	for _, linea := range lineas {
		campos := strings.Split(linea, ",")
		if len(campos) == 3 && campos[1] == "G" {
			id, _ := strconv.Atoi(campos[0])
			if int32(id) == gid {
				return campos[2] // Nombre del grupo
			}
		}
	}
	return "root"
}

// Genera un reporte tipo 'ls' mostrando archivos y carpetas con detalles
func ReporteLs(sb *Estructuras.SuperBlock, rutaDisco string, ruta string, rutaLs string) error {
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
	dot := iniciarDotLs()

	// Aquí deberías obtener la lista de archivos/carpetas en la rutaLs
	filas, err := obtenerFilasLs(sb, archivo, rutaLs)
	if err != nil {
		return err
	}
	dot += filas
	dot += "</table>>];\n}"

	err = escribirArchivoDot(nombreDot, dot)
	if err != nil {
		return err
	}
	err = generarImagenSuperbloque(nombreDot, nombreImagen)
	if err != nil {
		return err
	}

	fmt.Println("Reporte LS generado:", nombreImagen)
	return nil
}

func iniciarDotLs() string {
	return `digraph G {
        fontname="Helvetica,Arial,sans-serif"
        node [fontname="Helvetica,Arial,sans-serif", shape=plain, fontsize=12];
        edge [fontname="Helvetica,Arial,sans-serif", color="#FF7043", arrowsize=0.8];
        bgcolor="#FAFAFA";
        rankdir=TB;

        lsTable [label=<
            <table border="0" cellborder="1" cellspacing="0" cellpadding="6" bgcolor="#FFF9C4" style="rounded">
                <tr>
                    <td bgcolor="#4CAF50"><b>Permisos</b></td>
                    <td bgcolor="#4CAF50"><b>Owner</b></td>
                    <td bgcolor="#4CAF50"><b>Grupo</b></td>
                    <td bgcolor="#4CAF50"><b>Size (bytes)</b></td>
                    <td bgcolor="#4CAF50"><b>Fecha</b></td>
                    <td bgcolor="#4CAF50"><b>Hora</b></td>
                    <td bgcolor="#4CAF50"><b>Tipo</b></td>
                    <td bgcolor="#4CAF50"><b>Name</b></td>
                </tr>
    `
}

// Debes implementar esta función para recorrer la ruta y obtener los datos de cada archivo/carpeta
func obtenerFilasLs(sb *Estructuras.SuperBlock, archivo *os.File, rutaLs string) (string, error) {
	// Buscar el inodo raíz o el correspondiente a rutaLs
	indiceInodo := int32(0)
	if rutaLs != "/" && rutaLs != "" {
		var err error
		indiceInodo, err = buscarInodoArchivo(sb, archivo, rutaLs)
		if err != nil {
			return "", fmt.Errorf("no se encontró el directorio: %v", err)
		}
	}

	inodo, err := leerInodo(sb, archivo, indiceInodo)
	if err != nil {
		return "", fmt.Errorf("error al leer inodo: %v", err)
	}

	var filas string
	for _, idxBloque := range inodo.I_block {
		if idxBloque == -1 {
			continue
		}
		bloque := &Estructuras.FolderBlock{}
		offset := int64(sb.S_block_start + idxBloque*sb.S_block_size)
		err := bloque.Decodificar(archivo, offset)
		if err != nil {
			continue
		}
		for _, contenido := range bloque.B_cont {
			nombre := strings.Trim(string(contenido.B_name[:]), "\x00 ")
			if nombre == "" || nombre == "." || nombre == ".." {
				continue
			}
			inodoHijo, err := leerInodo(sb, archivo, contenido.B_inodo)
			if err != nil {
				continue
			}
			permisos := string(inodoHijo.I_perm[:])

			owner := obtenerNombreUsuarioPorUID(sb, archivo, inodoHijo.I_uid)
			grupo := obtenerNombreGrupoPorGID(sb, archivo, inodoHijo.I_gid)
			
			size := inodoHijo.I_size
			fecha := time.Unix(int64(inodoHijo.I_ctime), 0).Format("2006-01-02")
			hora := time.Unix(int64(inodoHijo.I_ctime), 0).Format("15:04:05")
			tipo := "Archivo"
			if inodoHijo.I_type[0] == '0' {
				tipo = "Carpeta"
			}
			filas += fmt.Sprintf(
				"<tr><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
				permisos, owner, grupo, size, fecha, hora, tipo, nombre,
			)
		}
	}
	return filas, nil
}
