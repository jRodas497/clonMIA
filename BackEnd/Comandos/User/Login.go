package User

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"strings"

	Estructuras "backend/Estructuras"
	Global "backend/Global"
)

type LOGIN struct {
	Usuario    string
	Contrasena string
	ID         string
}

// Analiza tokens y crea una instancia del comando, devolviendo mensajes importantes
func ParserLogin(tokens []string) (string, error) {
	var bufferSalida bytes.Buffer
	cmd := &LOGIN{}
	argumentos := strings.Join(tokens, " ")

	// Expresion regular para encontrar los parametros del comando login
	re := regexp.MustCompile(`-user=[^\s]+|-pass=[^\s]+|-id=[^\s]+`)
	coincidencias := re.FindAllString(argumentos, -1)

	// Validate tokens: ensure every token contains '=' and is one of the expected keys
	for _, coincidencia := range coincidencias {
		if !strings.Contains(coincidencia, "=") {
			return "", fmt.Errorf("formato de parametro invalido, se esperaba clave=valor: %s", coincidencia)
		}
		clavValor := strings.SplitN(coincidencia, "=", 2)
		if len(clavValor) != 2 || clavValor[1] == "" {
			return "", fmt.Errorf("formato de parametro invalido o valor vacio para: %s", coincidencia)
		}
		clave, valor := strings.ToLower(clavValor[0]), clavValor[1]

		switch clave {
		case "-user":
			cmd.Usuario = valor
		case "-pass":
			cmd.Contrasena = valor
		case "-id":
			cmd.ID = valor
		default:
			return "", fmt.Errorf("parametro desconocido: %s", clave)
		}
	}

	// Validar que se hayan proporcionado todos los parametros
	if cmd.Usuario == "" || cmd.Contrasena == "" || cmd.ID == "" {
		return "", fmt.Errorf("faltan parametros requeridos: -user, -pass, -id")
	}

	err := comandoLogin(cmd, &bufferSalida)
	if err != nil {
		return "", err
	}

	return bufferSalida.String(), nil
}

// Logica para ejecutar el login
func comandoLogin(login *LOGIN, bufferSalida *bytes.Buffer) error {
	fmt.Fprintln(bufferSalida, "----------------------------  LOGIN ----------------------------")
	fmt.Fprintf(bufferSalida, "Intentando iniciar sesion con ID: %s, Usuario: %s\n", login.ID, login.Usuario)

	// Validar si ya hay una sesion activa
	if Global.UsuarioActual != nil && Global.UsuarioActual.Estado {
		return fmt.Errorf("ya hay un usuario activo, debe cerrar sesion primero")
	}

	// Verificar que la particion este montada
	// Mostrar particiones montadas (debug amigable)
	if len(Global.ParticionesMontadas) > 0 {
		fmt.Fprintln(bufferSalida, "Particiones montadas:")
		for id, path := range Global.ParticionesMontadas {
			fmt.Fprintf(bufferSalida, "  ID: %s -> %s\n", id, path)
		}
	}

	_, ruta, err := Global.ObtenerParticionMontada(login.ID)
	if err != nil {
		return fmt.Errorf("no se puede encontrar la particion: %v", err)
	}
	fmt.Fprintf(bufferSalida, "Particion montada en: %s\n", ruta)

	// Cargar el Superblock de la particion montada
	_, sb, _, err := Global.ObtenerParticionMontadaReporte(login.ID)
	if err != nil {
		return fmt.Errorf("no se pudo cargar el SuperBlock: %v", err)
	}
	fmt.Fprintln(bufferSalida, "SuperBlock cargado correctamente")

	// Acceder al inodo del archivo users.txt (inodo 1)
	archivo, err := os.Open(ruta)
	if err != nil {
		return fmt.Errorf("no se puede abrir el archivo de particion: %v", err)
	}
	defer archivo.Close()

	// Leer el inodo 1 (que contiene el archivo users.txt)
	var inodoUsuarios Estructuras.INodo
	desplazamientoInodo := int64(sb.S_inode_start + int32(binary.Size(inodoUsuarios)))

	err = inodoUsuarios.Decodificar(archivo, desplazamientoInodo)
	if err != nil {
		return fmt.Errorf("error leyendo inodo de users.txt: %v", err)
	}

	inodoUsuarios.ActualizarTiempoAcceso()

	// Leer el contenido de los bloques asociados al archivo users.txt
	var contenido string
	for _, indiceBloques := range inodoUsuarios.I_block {
		if indiceBloques == -1 {
			continue
		}

		desplazamientoBloque := int64(sb.S_block_start + indiceBloques*int32(binary.Size(Estructuras.FileBlock{})))

		var bloqueArchivo Estructuras.FileBlock
		err = bloqueArchivo.Decodificar(archivo, desplazamientoBloque)
		if err != nil {
			return fmt.Errorf("error leyendo bloque de users.txt: %v", err)
		}

		contenido += string(bloqueArchivo.B_cont[:])
	}

	// Validar el usuario y contrasena
	encontrado := false
	lineas := strings.Split(strings.TrimSpace(contenido), "\n")
	for _, linea := range lineas {
		if linea == "" {
			continue
		}

		datos := strings.Split(linea, ",")
		if len(datos) == 5 && datos[1] == "U" {
			// Crear un objeto Usuario a partir de la linea
			usuario := Estructuras.NuevoUsuario(datos[0], datos[2], datos[3], datos[4])

			// Comparar usuario y contrasena
			if usuario.Nombre == login.Usuario && usuario.Pass == login.Contrasena {
				encontrado = true
				Global.UsuarioActual = usuario
				Global.UsuarioActual.Estado = true
				Global.UsuarioActual.Id = login.ID
				fmt.Fprintf(bufferSalida, "Bienvenido %s, inicio de sesion exitoso.\n", usuario.Nombre)
				break
			}
		}
	}

	if !encontrado {
		return fmt.Errorf("usuario o contrasena incorrectos")
	}

	fmt.Fprintln(bufferSalida, "--------------------------------------------")
	return nil
}
