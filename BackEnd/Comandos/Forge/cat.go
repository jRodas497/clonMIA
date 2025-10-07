package Forge

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	Estructuras "backend/Estructuras"
	Global "backend/Global"
	Utils "backend/Utils"
)

// Estructura del comando CAT con parametros
type CAT struct {
	// Lista de archivos a leer
	archivos []string
}

// Procesa el comando cat y devuelve una instancia de CAT
func ParserCat(tokens []string) (string, error) {
	cmd := &CAT{}
	var bufferSalida bytes.Buffer

	// Expresion regular para capturar archivos pasados como parametros -file1, -file2, etc.
	re := regexp.MustCompile(`-file\d+=("[^"]+"|[^\s]+)`)
	coincidencias := re.FindAllString(strings.Join(tokens, " "), -1)

	// Verificar si no se encontraron archivos
	if len(coincidencias) == 0 {
		return "", errors.New("no se especificaron archivos para leer")
	}

	// Procesar cada coincidencia y extraer archivos
	for _, coincidencia := range coincidencias {
		// Separar parametro en clave y valor (ej. "-file1=/home/user/a.txt")
		clavValor := strings.SplitN(coincidencia, "=", 2)
		if len(clavValor) == 2 {
			rutaArchivo := clavValor[1]
			if strings.HasPrefix(rutaArchivo, "\"") && strings.HasSuffix(rutaArchivo, "\"") {
				rutaArchivo = strings.Trim(rutaArchivo, "\"")
			}
			cmd.archivos = append(cmd.archivos, rutaArchivo)
		}
	}

	err := comandoCat(cmd, &bufferSalida)
	if err != nil {
		return "", err
	}

	return bufferSalida.String(), nil
}

// Ejecuta la lectura de archivos
func comandoCat(cat *CAT, bufferSalida *bytes.Buffer) error {
	fmt.Fprint(bufferSalida, "------------------------ CAT ------------------------\n")

	// Verificar usuario logueado
	if !Global.VerificarSesionActiva() {
		return fmt.Errorf("no hay un usuario logueado")
	}

	// Obtener ID de particion del usuario logueado
	idParticion := Global.UsuarioActual.Id

	// Obtener particion montada asociada al usuario logueado
	_, _, rutaParticion, err := Global.ObtenerSuperblockParticionMontada(idParticion)
	if err != nil {
		return fmt.Errorf("error al obtener la particion montada: %w", err)
	}

	// Abrir archivo de particion para operar
	archivo, err := os.OpenFile(rutaParticion, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de particion: %w", err)
	}
	defer archivo.Close()

	for _, rutaArchivo := range cat.archivos {
		fmt.Fprintf(bufferSalida, "Leyendo archivo: %s\n", rutaArchivo)

		contenido, err := leerContenidoArchivo(rutaArchivo)
		if err != nil {
			fmt.Fprintf(bufferSalida, "Error al leer el archivo %s: %v\n", rutaArchivo, err)
			continue
		}

		bufferSalida.WriteString(contenido)
		bufferSalida.WriteString("\n") // Separar contenido con salto de linea
		fmt.Fprint(bufferSalida, "--------------------------------------------\n")
	}

	return nil
}

// Se busca el archivo en el sistema y lee su contenido
func leerContenidoArchivo(rutaArchivo string) (string, error) {
	// Obtener SuperBlock y particion montada asociada
	idParticion := Global.UsuarioActual.Id
	superBloqueParticion, _, rutaParticion, err := Global.ObtenerSuperblockParticionMontada(idParticion)
	if err != nil {
		return "", fmt.Errorf("error al obtener la particion montada: %v", err)
	}

	// Abrir archivo de particion para lectura
	archivo, err := os.OpenFile(rutaParticion, os.O_RDONLY, 0666)
	if err != nil {
		return "", fmt.Errorf("error al abrir el archivo de particion: %v", err)
	}
	defer archivo.Close()

	// Convertir ruta del archivo en array de carpetas
	directoriosPadre, nombreArchivo := Utils.ObtenerDirectoriosPadre(rutaArchivo)

	// Buscar archivo en sistema de archivos
	indiceInodo, err := buscarInodoArchivo(archivo, superBloqueParticion, directoriosPadre, nombreArchivo)
	if err != nil {
		return "", fmt.Errorf("error al encontrar el archivo: %v", err)
	}

	contenido, err := leerArchivoDesdeInodo(archivo, superBloqueParticion, indiceInodo)
	if err != nil {
		return "", fmt.Errorf("error al leer el contenido del archivo: %v", err)
	}

	return contenido, nil
}

// Verifica si un directorio o archivo existe en el inodo dado
func directorioExiste(sb *Estructuras.SuperBlock, archivo *os.File, indiceInodo int32, nombreDirectorio string) (bool, int32, error) {
	// Deserializar inodo correspondiente
	inodo := &Estructuras.INodo{}
	err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
	if err != nil {
		return false, -1, fmt.Errorf("error al deserializar inodo %d: %v", indiceInodo, err)
	}

	// Verificar si el inodo es de tipo carpeta (I_type == '0') para continuar
	if inodo.I_type[0] != '0' {
		return false, -1, fmt.Errorf("el inodo %d no es una carpeta", indiceInodo)
	}

	// Recorrer bloques del inodo para buscar directorio o archivo
	for _, indiceBloques := range inodo.I_block {
		if indiceBloques == -1 {
			break // Si no hay mas bloques asignados, terminar busqueda
		}

		// Deserializar bloque de directorio
		bloque := &Estructuras.FolderBlock{}
		err := bloque.Decodificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
		if err != nil {
			return false, -1, fmt.Errorf("error al deserializar bloque %d: %v", indiceBloques, err)
		}

		// Recorrer contenidos del bloque para verificar coincidencia de nombre
		for _, contenido := range bloque.B_cont {
			nombreContenido := strings.Trim(string(contenido.B_name[:]), "\x00 ") // Convertir nombre y quitar caracteres nulos
			if strings.EqualFold(nombreContenido, nombreDirectorio) && contenido.B_inodo != -1 {
				return true, contenido.B_inodo, nil // Devolver true si directorio/archivo fue encontrado
			}
		}
	}

	// No se encontro directorio/archivo
	fmt.Printf("Directorio o archivo '%s' no encontrado en inodo %d\n", nombreDirectorio, indiceInodo)
	return false, -1, nil
}

// buscarInodoArchivo busca el inodo de un archivo dado su ruta
func buscarInodoArchivo(archivo *os.File, sb *Estructuras.SuperBlock, directoriosPadre []string, nombreArchivo string) (int32, error) {
	// Empezar busqueda en inodo raiz
	indiceInodo := int32(0)

	// Navegar por directorios padre para llegar al archivo
	for len(directoriosPadre) > 0 {
		nombreDirectorio := directoriosPadre[0]
		encontrado, nuevoIndiceInodo, err := directorioExiste(sb, archivo, indiceInodo, nombreDirectorio)
		if err != nil {
			return -1, err
		}
		if !encontrado {
			return -1, fmt.Errorf("directorio '%s' no encontrado", nombreDirectorio)
		}
		indiceInodo = nuevoIndiceInodo
		directoriosPadre = directoriosPadre[1:]
	}

	// Buscar archivo en ultimo directorio
	encontrado, indiceInodoArchivo, err := directorioExiste(sb, archivo, indiceInodo, nombreArchivo)
	if err != nil {
		return -1, err
	}
	if !encontrado {
		return -1, fmt.Errorf("archivo '%s' no encontrado", nombreArchivo)
	}

	return indiceInodoArchivo, nil
}

// leerArchivoDesdeInodo lee contenido de un archivo desde su inodo
func leerArchivoDesdeInodo(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32) (string, error) {
	inodo := &Estructuras.INodo{}
	err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
	if err != nil {
		return "", fmt.Errorf("error al deserializar el inodo %d: %v", indiceInodo, err)
	}

	if inodo.I_type[0] != '1' {
		return "", fmt.Errorf("el inodo %d no corresponde a un archivo", indiceInodo)
	}

	// Concatenar bloques de contenido del archivo
	var constructorContenido strings.Builder
	for _, indiceBloques := range inodo.I_block {
		if indiceBloques == -1 {
			break
		}

		bloqueArchivo := &Estructuras.FileBlock{}
		err := bloqueArchivo.Decodificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
		if err != nil {
			return "", fmt.Errorf("error al deserializar el bloque %d: %v", indiceBloques, err)
		}

		constructorContenido.WriteString(string(bloqueArchivo.B_cont[:]))
	}

	return constructorContenido.String(), nil
}

// buscarInodoCarpeta busca el inodo de una carpeta dado su ruta
func buscarInodoCarpeta(archivo *os.File, sb *Estructuras.SuperBlock, directoriosPadre []string) (int32, error) {
    // Si no hay directorios padre, retornar el inodo raiz (directorio root)
    if len(directoriosPadre) == 0 {
        return 0, nil
    }
    
    // Empezar busqueda en inodo raiz
    indiceInodo := int32(0)

    // Navegar por cada directorio en la ruta
    for _, nombreDirectorio := range directoriosPadre {
        encontrado, nuevoIndiceInodo, err := directorioExiste(sb, archivo, indiceInodo, nombreDirectorio)
        if err != nil {
            return -1, err
        }
        if !encontrado {
            return -1, fmt.Errorf("directorio '%s' no encontrado", nombreDirectorio)
        }
        
        // Verificar que efectivamente sea un directorio
        inodo := &Estructuras.INodo{}
        err = inodo.Decodificar(archivo, int64(sb.S_inode_start+(nuevoIndiceInodo*sb.S_inode_size)))
        if err != nil {
            return -1, fmt.Errorf("error al leer inodo %d: %v", nuevoIndiceInodo, err)
        }
        
        if inodo.I_type[0] != '0' {
            return -1, fmt.Errorf("'%s' no es un directorio", nombreDirectorio)
        }
        
        indiceInodo = nuevoIndiceInodo
    }

    return indiceInodo, nil
}