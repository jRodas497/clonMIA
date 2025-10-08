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

// MKDIR estructura del comando mkdir con parametros
type MKDIR struct {
	ruta string // Ruta del directorio
	p    bool   // Opcion -p (crea directorios padres si no existen)
}

func ParserMkdir(tokens []string) (string, error) {
	// Crea una nueva instancia de MKDIR
	cmd := &MKDIR{}
	// Buffer para capturar mensajes importantes
	var bufferSalida bytes.Buffer

	argumentos := strings.Join(tokens, " ")
	// Expresion regular para encontrar los parametros del comando mkdir
	re := regexp.MustCompile(`-path=[^\s]+|-p`)
	coincidencias := re.FindAllString(argumentos, -1)

	// Verificar que todos los tokens fueron reconocidos por la expresion regular
	if len(coincidencias) != len(tokens) {
		for _, token := range tokens {
			if !re.MatchString(token) {
				return "", fmt.Errorf("parametro invalido: %s", token)
			}
		}
	}

	// Itera cada coincidencia encontrada
	for _, coincidencia := range coincidencias {
		clavValor := strings.SplitN(coincidencia, "=", 2)
		clave := strings.ToLower(clavValor[0])

		switch clave {
		case "-path":
			if len(clavValor) != 2 {
				return "", fmt.Errorf("formato de parametro invalido: %s", coincidencia)
			}
			valor := clavValor[1]
			// Remover comillas
			if strings.HasPrefix(valor, "\"") && strings.HasSuffix(valor, "\"") {
				valor = strings.Trim(valor, "\"")
			}
			cmd.ruta = valor
		case "-p":
			cmd.p = true
		default:
			// Parametro no reconocido devuelve un error
			return "", fmt.Errorf("parametro desconocido: %s", clave)
		}
	}

	// Verifica que el parametro -path haya sido proporcionado
	if cmd.ruta == "" {
		return "", errors.New("faltan parametros requeridos: -path")
	}

	// Ejecutar el comando mkdir con captura de mensajes en el buffer
	err := comandoMkdir(cmd, &bufferSalida)
	if err != nil {
		return "", err
	}

	// Retorna el contenido del buffer al usuario
	return bufferSalida.String(), nil
}

func comandoMkdir(mkdir *MKDIR, bufferSalida *bytes.Buffer) error {
	// Verificar si hay un usuario logueado
	if !Global.VerificarSesionActiva() {
		return fmt.Errorf("no hay un usuario logueado")
	}

	// Obtener el ID de la particion desde el usuario logueado
	idParticion := Global.UsuarioActual.Id

	superBloqueParticion, particionMontada, rutaParticion, err := Global.ObtenerSuperblockParticionMontada(idParticion)
	if err != nil {
		return fmt.Errorf("error al obtener la particion montada: %w", err)
	}

	// Abrir el archivo de particion
	archivo, err := os.OpenFile(rutaParticion, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el archivo de particion: %w", err)
	}
	defer archivo.Close()

	// Capturar mensajes importantes en buffer
	fmt.Fprintln(bufferSalida, "---------------------------- MKDIR ----------------------------")
	fmt.Fprintf(bufferSalida, "Creando directorio: %s\n", mkdir.ruta)

	err = crearDirectorio(mkdir.ruta, mkdir.p, superBloqueParticion, archivo, particionMontada)
	if err != nil {
		return fmt.Errorf("error al crear el directorio: %w", err)
	}

	fmt.Fprintf(bufferSalida, "Directorio %s creado exitosamente\n", mkdir.ruta)
	fmt.Fprintln(bufferSalida, "--------------------------------------------")

	return nil
}

func crearDirectorio(rutaDirectorio string, crearPadres bool, sb *Estructuras.SuperBlock, archivo *os.File, particionMontada *Estructuras.Particion) error {
    // Si el parámetro -p está habilitado, crear los directorios intermedios recursivamente
    if crearPadres {
        // Utilizamos `CrearCarpetaRecursivamente` para crear los directorios si no existen
        err := sb.CrearCarpetaRecursivamente(archivo, rutaDirectorio, true)
        if err != nil {
            return fmt.Errorf("error al crear los directorios recursivamente: %w", err)
        }
    } else {
        // Si no se habilita el parámetro -p, asegurarse de que los directorios padres existan
        directoriosPadre, directorioDestino := Utils.ObtenerDirectoriosPadre(rutaDirectorio)
        // / -> ["/"] -> usuarios : /usuarios
        // Verificar que todos los directorios padres existen
        err := verificarDirectoriosPadreExisten(sb, archivo, directoriosPadre)
        if err != nil {
            return err
        }

	// Crear el directorio final
	err := sb.CrearCarpeta(archivo, directoriosPadres, directorioDestino, true)
	if err != nil {
		return fmt.Errorf("error al crear el directorio: %w", err)
	}

	// Serializar el superbloque en el archivo de particion abierto
	err = sb.Codificar(archivo, int64(particionMontada.Part_start))
	if err != nil {
		return fmt.Errorf("error al serializar el superbloque: %w", err)
	}

	// Depuración
	fmt.Println("\nInodos:")
	sb.ImprimirInodos(archivo.Name())
	fmt.Println("\nBloques:")
	sb.ImprimirBloques(archivo.Name())

	return nil
}
