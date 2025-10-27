package Forge

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	Estructuras "backend/Estructuras"
	Global "backend/Global"
	Utils "backend/Utils"
)

// Estructura del comando mkfile con parametros
type MKFILE struct {
	ruta      string // Ruta del archivo
	r         bool   // Opcion recursiva
	tamaño    int    // Dimension del archivo
	contenido string // Contenido del archivo
}

// Procesa el comando mkfile y devuelve una instancia de MKFILE
func ParserMkfile(tokens []string) (string, error) {
	cmd := &MKFILE{}              // Crear nueva instancia de MKFILE
	var bufferSalida bytes.Buffer // Buffer para capturar mensajes importantes

	argumentos := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-path="[^"]+"|-path=[^\s]+|-r|-size=\d+|-cont="[^"]+"|-cont=[^\s]+`)
	coincidencias := re.FindAllString(argumentos, -1)

	if len(coincidencias) != len(tokens) {
		for _, token := range tokens {
			if !re.MatchString(token) {
				return "", fmt.Errorf("parametro invalido: %s", token)
			}
		}
	}

	for _, coincidencia := range coincidencias {
		clavValor := strings.SplitN(coincidencia, "=", 2)
		clave := strings.ToLower(clavValor[0])
		var valor string
		if len(clavValor) == 2 {
			valor = clavValor[1]
		}

		if strings.HasPrefix(valor, "\"") && strings.HasSuffix(valor, "\"") {
			valor = strings.Trim(valor, "\"")
		}

		// Switch para manejar diferentes parametros
		switch clave {
		case "-path":
			if valor == "" {
				return "", errors.New("la ruta no puede estar vacia")
			}
			cmd.ruta = valor
		case "-r":
			// Habilitar opcion recursiva
			cmd.r = true
		case "-size":
			dimension, err := strconv.Atoi(valor)
			if err != nil || dimension < 0 {
				return "", errors.New("la dimension debe ser un numero entero no negativo")
			}
			cmd.tamaño = dimension
		case "-cont":
			if valor == "" {
				return "", errors.New("el contenido no puede estar vacio")
			}
			cmd.contenido = valor
		default:
			return "", fmt.Errorf("parametro desconocido: %s", clave)
		}
	}

	if cmd.ruta == "" {
		return "", errors.New("faltan parametros requeridos: -path")
	}

	if cmd.tamaño == 0 {
		cmd.tamaño = 0
	}

	if cmd.contenido == "" {
		cmd.contenido = ""
	}

	// Crear archivo con parametros proporcionados
	err := comandoMkfile(cmd, &bufferSalida)
	if err != nil {
		return "", err
	}

	// Retornar contenido del buffer con mensajes importantes
	return bufferSalida.String(), nil
}

// Ejecuta la creacion del archivo
func comandoMkfile(mkfile *MKFILE, bufferSalida *bytes.Buffer) error {
	// Verificar usuario logueado
	if !Global.VerificarSesionActiva() {
		return fmt.Errorf("no hay un usuario logueado")
	}

	// Obtener ID de particion del usuario logueado
	idParticion := Global.UsuarioActual.Id

	// Obtener particion al usuario logueado
	superBloqueParticion, particionMontada, rutaParticion, err := Global.ObtenerSuperblockParticionMontada(idParticion)
	if err != nil {
		return fmt.Errorf("error al obtener la particion montada: %w", err)
	}

	// Generar contenido del archivo si no se proporciono
	if mkfile.contenido == "" {
		mkfile.contenido = generarContenido(mkfile.tamaño)
	}

	// Abrir archivo de particion para operar
	archivo, err := os.OpenFile(rutaParticion, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de particion: %w", err)
	}
	// Cerrar archivo cuando no sea necesario
	defer archivo.Close()

	// Capturar mensajes importantes en buffer
    fmt.Fprintln(bufferSalida, "----------------------- MKFILE -----------------------")
    fmt.Fprintf(bufferSalida, "Creando archivo: %s\n", mkfile.ruta)

    // Obtener directorios padre del archivo
    directoriosPadre, _ := Utils.ObtenerDirectoriosPadre(mkfile.ruta)

    // Verificar si el directorio existe, y si -r está habilitado, crearlo recursivamente
    if mkfile.r {
        fmt.Fprintf(bufferSalida, "Creando directorios intermedios si es necesario: %s\n", strings.Join(directoriosPadre, "/"))
        err = superBloqueParticion.CrearCarpetaRecursivamente(archivo, strings.Join(directoriosPadre, "/"), true)
        if err != nil {
            return fmt.Errorf("error al crear directorios intermedios: %w", err)
        }
    } else {
        fmt.Fprintf(bufferSalida, "Verificando si el directorio '%s' existe...\n", strings.Join(directoriosPadre, "/"))
        _, err := buscarInodoCarpeta(archivo, superBloqueParticion, directoriosPadre)
        if err != nil {
            return fmt.Errorf("el directorio '%s' no existe y no se ha especificado la opcion -r: %w", strings.Join(directoriosPadre, "/"), err)
        }
    }

	err = crearArchivo(mkfile.ruta, mkfile.tamaño, mkfile.contenido, superBloqueParticion, archivo, particionMontada, bufferSalida)
	if err != nil {
		return fmt.Errorf("error al crear el archivo: %w", err)
	}

	fmt.Fprintf(bufferSalida, "Archivo %s creado exitosamente\n", mkfile.ruta)
	fmt.Fprintln(bufferSalida, "----------------------------------------------")

	return nil
}

// Produce una cadena de numeros del 0 al 9 para cumplir la dimension
func generarContenido(dimension int) string {
	contenido := ""
	for len(contenido) < dimension {
		contenido += "0123456789"
	}
	return contenido[:dimension] // Recortar cadena a dimension exacta
}

// crearArchivo utiliza archivo de particion ya abierto
func crearArchivo(rutaArchivo string, dimension int, contenido string, sb *Estructuras.SuperBlock, archivo *os.File, particionMontada *Estructuras.Particion, bufferSalida *bytes.Buffer) error {
	fmt.Fprintf(bufferSalida, "Creando archivo en la ruta: %s\n", rutaArchivo)

	// Obtener directorios padre y destino
	directoriosPadre, directorioDestino := Utils.ObtenerDirectoriosPadre(rutaArchivo)
	// Obtener contenido por fragmentos
	fragmentos := Utils.DividirCadenaEnChunks(contenido)
	fmt.Fprintf(bufferSalida, "Contenido generado: %v\n", fragmentos)

	// Crear archivo en sistema de archivos
	err := sb.CrearArchivo(archivo, directoriosPadre, directorioDestino, dimension, fragmentos, false)
	if err != nil {
		return fmt.Errorf("error al crear el archivo: %w", err)
	}

	// Serializar superbloque
	err = sb.Codificar(archivo, int64(particionMontada.Part_start))
	if err != nil {
		return fmt.Errorf("error al serializar el superbloque: %w", err)
	}

	fmt.Println("\nInodos:")
	sb.ImprimirInodos(archivo.Name())
	fmt.Println("\nBloques de datos:")
	sb.ImprimirBloques(archivo.Name())

	return nil
}

// Devuelve carpeta directorio y nombre del archivo
func ObtenerDirectorioYArchivo(ruta string) (string, string) {
	// Obtener carpeta donde se creara el archivo
	directorio := filepath.Dir(ruta)
	// Obtener nombre del archivo
	archivo := filepath.Base(ruta)
	return directorio, archivo
}
