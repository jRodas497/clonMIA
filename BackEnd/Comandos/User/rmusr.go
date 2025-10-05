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

// Estructura para el comando
type RMUSR struct {
	Usuario string
}

// Parseo de argumentos para el comando rmusr y captura de mensajes importantes
func ParserRmusr(tokens []string) (string, error) {
	var bufferSalida bytes.Buffer

	cmd := &RMUSR{}

	// Expresion regular para parametro -user
	re := regexp.MustCompile(`-user=[^\s]+`)
	coincidencia := re.FindString(strings.Join(tokens, " "))

	if coincidencia == "" {
		return "", fmt.Errorf("falta el parametro -user")
	}

	// Extraer el valor del parametro -usr
	parametro := strings.SplitN(coincidencia, "=", 2)
	if len(parametro) != 2 {
		return "", fmt.Errorf("formato incorrecto para -user")
	}
	cmd.Usuario = parametro[1]

	err := comandoRmusr(cmd, &bufferSalida)
	if err != nil {
		return "", err
	}

	return bufferSalida.String(), nil
}

// RMUSR captura los mensajes importantes en buffer
func comandoRmusr(rmusr *RMUSR, bufferSalida *bytes.Buffer) error {
	fmt.Fprintln(bufferSalida, "---------------------------- RMUSR ----------------------------")

	// Verificar si hay una sesion activa y si el usuario es root
	if !Global.VerificarSesionActiva() {
		return fmt.Errorf("no hay ninguna sesion activa")
	}
	if Global.UsuarioActual.Nombre != "root" {
		return fmt.Errorf("solo el usuario root puede ejecutar este comando")
	}

	// Verificar que la particion esta montada
	_, ruta, err := Global.ObtenerParticionMontada(Global.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se puede encontrar la particion montada: %v", err)
	}

	// Abrir el archivo de la particion
	archivo, err := os.OpenFile(ruta, os.O_RDWR, 0755)
	if err != nil {
		return fmt.Errorf("no se puede abrir el archivo de la particion: %v", err)
	}
	defer archivo.Close()

	mbr, sb, _, err := Global.ObtenerParticionMontadaReporte(Global.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se pudo cargar el SuperBlock: %v", err)
	}

	// Obtener la particion montada
	particion, err := mbr.ObtenerParticionPorID(Global.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se pudo obtener la particion: %v", err)
	}

	var inodoUsuarios Estructuras.INodo
	desplazamientoInodo := int64(sb.S_inode_start + int32(binary.Size(inodoUsuarios))) // Posicion de los bloques de users.txt
	err = inodoUsuarios.Decodificar(archivo, desplazamientoInodo)
	if err != nil {
		return fmt.Errorf("error leyendo el inodo de users.txt: %v", err)
	}

	_, err = Global.BuscarEnArchivoUsuarios(archivo, sb, &inodoUsuarios, rmusr.Usuario, "U")
	if err != nil {
		return fmt.Errorf("el usuario '%s' no existe", rmusr.Usuario)
	}

	// Marcar el usuario como eliminado
	err = ActualizarEstadoUsuario(archivo, sb, &inodoUsuarios, rmusr.Usuario)
	if err != nil {
		return fmt.Errorf("error eliminando el usuario '%s': %v", rmusr.Usuario, err)
	}

	// Actualizar el inodo de users.txt
	err = inodoUsuarios.Codificar(archivo, desplazamientoInodo)
	if err != nil {
		return fmt.Errorf("error actualizando inodo de users.txt: %v", err)
	}

	// Guardar el SuperBlock utilizando el Part_start como el offset
	err = sb.Codificar(archivo, int64(particion.Part_start))
	if err != nil {
		return fmt.Errorf("error guardando el SuperBlock: %v", err)
	}

	fmt.Fprintf(bufferSalida, "Usuario '%s' eliminado exitosamente.\n", rmusr.Usuario)
	fmt.Fprintf(bufferSalida, "--------------------------------------------")

	return nil
}

// Cambia el estado de un usuario a eliminado y actualiza el archivo
func ActualizarEstadoUsuario(archivo *os.File, sb *Estructuras.SuperBlock, inodoUsuarios *Estructuras.INodo, nombreUsuario string) error {
	contenido, err := Global.LeerBloquesArchivo(archivo, sb, inodoUsuarios)
	if err != nil {
		return fmt.Errorf("error leyendo el contenido de users.txt: %v", err)
	}

	lineas := strings.Split(contenido, "\n")
	modificado := false

	// Recorrer las lineas para buscar y modificar el estado del usuario
	for i, linea := range lineas {
		linea = strings.TrimSpace(linea) 
		if linea == "" {
			continue
		}

		// Crear un objeto Usuario a partir de la linea
		usuario := crearUsuarioDesdeLinea(linea)

		if usuario != nil && usuario.Nombre == nombreUsuario {
			// Eliminar el usuario (cambiar ID a "0")
			usuario.Eliminar()

			lineas[i] = usuario.ToString()
			modificado = true
			break
		}
	}

	if !modificado {
		return fmt.Errorf("usuario '%s' no encontrado en users.txt", nombreUsuario)
	}

	contenidoActualizado := limpiarYActualizarContenido(lineas)

	return escribirCambiosEnArchivo(archivo, sb, inodoUsuarios, contenidoActualizado)
}

func crearUsuarioDesdeLinea(linea string) *Estructuras.Usuario {
	partes := strings.Split(linea, ",")
	if len(partes) >= 5 && partes[1] == "U" {
		return Estructuras.NuevoUsuario(partes[0], partes[2], partes[3], partes[4])
	}
	return nil
}

// Elimina lineas vacias y devuelve el contenido
func limpiarYActualizarContenido(lineas []string) string {
	var contenidoActualizado []string
	for _, linea := range lineas {
		if strings.TrimSpace(linea) != "" {
			contenidoActualizado = append(contenidoActualizado, linea)
		}
	}
	return strings.Join(contenidoActualizado, "\n") + "\n"
}

// Limpia los bloques y escribe el contenido actualizado en el archivo
func escribirCambiosEnArchivo(archivo *os.File, sb *Estructuras.SuperBlock, inodoUsuarios *Estructuras.INodo, contenido string) error {
	for _, indiceBloques := range inodoUsuarios.I_block {
		if indiceBloques == -1 {
			break
		}

		desplazamientoBloque := int64(sb.S_block_start + indiceBloques*sb.S_block_size)
		var bloqueArchivo Estructuras.FileBlock

		bloqueArchivo.LimpiarContenido()

		err := bloqueArchivo.Codificar(archivo, desplazamientoBloque)
		if err != nil {
			return fmt.Errorf("error escribiendo bloque limpio %d: %w", indiceBloques, err)
		}
	}

	err := Global.EscribirBloquesUsuarios(archivo, sb, inodoUsuarios, contenido)
	if err != nil {
		return fmt.Errorf("error guardando los cambios en users.txt: %v", err)
	}

	return nil
}
