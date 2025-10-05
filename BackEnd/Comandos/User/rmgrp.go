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

// Estructura para el comando RMGRP
type RMGRP struct {
	Nombre string
}

// Parseo de argumentos para el comando rmgrp y captura de mensajes importantes
func ParserRmgrp(tokens []string) (string, error) {
	var bufferSalida bytes.Buffer

	// Inicializar el comando RMGRP
	cmd := &RMGRP{}

	// Encontrar el parametro -name
	re := regexp.MustCompile(`-name=[^\s]+`)
	coincidencia := re.FindString(strings.Join(tokens, " "))

	if coincidencia == "" {
		return "", fmt.Errorf("falta el parametro -name")
	}

	// -name
	parametro := strings.SplitN(coincidencia, "=", 2)
	if len(parametro) != 2 {
		return "", fmt.Errorf("formato incorrecto para -name")
	}
	cmd.Nombre = parametro[1]

	err := comandoRmgrp(cmd, &bufferSalida)
	if err != nil {
		return "", err
	}

	return bufferSalida.String(), nil
}

// Captura de mensajes importantes en el buffer
func comandoRmgrp(rmgrp *RMGRP, bufferSalida *bytes.Buffer) error {
	fmt.Fprintln(bufferSalida, "======================= RMGRP =======================")

	// Verificar si hay una sesion activa y si el usuario es root
	if !Global.VerificarSesionActiva() {
		return fmt.Errorf("no hay ninguna sesion activa")
	}
	if Global.UsuarioActual.Nombre != "root" {
		return fmt.Errorf("solo el usuario root puede ejecutar este comando")
	}

	// Verificar que la particion este montada
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

	// Cargar el SuperBlock y la particion
	mbr, sb, _, err := Global.ObtenerParticionMontadaReporte(Global.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se pudo cargar el SuperBlock: %v", err)
	}

	// Obtener la particion montada
	particion, err := mbr.ObtenerParticionPorID(Global.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se pudo obtener la particion: %v", err)
	}

	// Leer el inodo de users.txt
	var inodoUsuarios Estructuras.INodo
	desplazamientoInodo := int64(sb.S_inode_start + int32(binary.Size(inodoUsuarios))) //posicion del inodo de users.txt
	err = inodoUsuarios.Decodificar(archivo, desplazamientoInodo)
	if err != nil {
		return fmt.Errorf("error leyendo el inodo de users.txt: %v", err)
	}

	// Verificar si el grupo existe
	_, err = Global.BuscarEnArchivoUsuarios(archivo, sb, &inodoUsuarios, rmgrp.Nombre, "G")
	if err != nil {
		return fmt.Errorf("el grupo '%s' no existe", rmgrp.Nombre)
	}

	// Cambiar el estado (grupo, usuarios)
	err = ActualizarEstadoEntidadOEliminarUsuarios(archivo, sb, &inodoUsuarios, rmgrp.Nombre, "G", "0")
	if err != nil {
		return fmt.Errorf("error eliminando el grupo y usuarios asociados: %v", err)
	}

	// Actualizar el inodo de users.txt en el archivo
	err = inodoUsuarios.Codificar(archivo, desplazamientoInodo)
	if err != nil {
		return fmt.Errorf("error actualizando inodo de users.txt: %v", err)
	}

	// Guardar SuperBlock usando el Part_start como el offset
	err = sb.Codificar(archivo, int64(particion.Part_start))
	if err != nil {
		return fmt.Errorf("error guardando el SuperBlock: %v", err)
	}

	fmt.Fprintf(bufferSalida, "Grupo '%s' eliminado exitosamente, junto con sus usuarios.\n", rmgrp.Nombre)
	fmt.Fprintln(bufferSalida, "=====================================================")

	return nil
}

// Cambia el estado de un grupo/usuario y elimina usuarios asociados a un grupo
func ActualizarEstadoEntidadOEliminarUsuarios(archivo *os.File, sb *Estructuras.SuperBlock, inodoUsuarios *Estructuras.INodo, nombre string, tipoEntidad string, nuevoEstado string) error {
	// Leer el contenido actual de users.txt
	contenido, err := Global.LeerBloquesArchivo(archivo, sb, inodoUsuarios)
	if err != nil {
		return fmt.Errorf("error leyendo el contenido de users.txt: %v", err)
	}

	lineas := strings.Split(contenido, "\n")
	modificado := false

	var nombreGrupo string
	if tipoEntidad == "G" {
		nombreGrupo = nombre
	}

	for i, linea := range lineas {
		// Eliminar espacios en blanco adicionales
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}

		partes := strings.Split(linea, ",")
		if len(partes) < 3 {
			// Ignorar lineas mal formadas
			continue
		}

		tipo := partes[1]
		nombreEntidad := partes[2]

		// Verificar si coincide el tipo de entidad (usuario o grupo) y el nombre
		if tipo == tipoEntidad && nombreEntidad == nombre {
			partes[0] = nuevoEstado
			lineas[i] = strings.Join(partes, ",")
			modificado = true

			// Si es un grupo, busca y elimina a los usuarios asociados
			if tipoEntidad == "G" {
				// Recorrer de nuevo las lineas para eliminar usuarios del grupo
				for j, lineaUsuario := range lineas {
					lineaUsuario = strings.TrimSpace(lineaUsuario)
					if lineaUsuario == "" {
						continue
					}
					partesUsuario := strings.Split(lineaUsuario, ",")
					if len(partesUsuario) == 5 && partesUsuario[2] == nombreGrupo {
						// Marcar el usuario como eliminado
						partesUsuario[0] = "0"
						lineas[j] = strings.Join(partesUsuario, ",")
					}
				}
			}
			break
		}
	}

	// Si se modifico alguna linea, guardar los cambios en el archivo
	if modificado {
		contenidoActualizado := strings.Join(lineas, "\n")

		for _, indiceBloques := range inodoUsuarios.I_block {
			if indiceBloques == -1 {
				// No hay mas bloques asignados
				break
			}

			desplazamientoBloque := int64(sb.S_block_start + indiceBloques*sb.S_block_size)
			var bloqueArchivo Estructuras.FileBlock

			// Limpiar el contenido del bloque
			bloqueArchivo.LimpiarContenido()

			// Escribir el bloque vacio de nuevo
			err = bloqueArchivo.Codificar(archivo, desplazamientoBloque)
			if err != nil {
				return fmt.Errorf("error escribiendo bloque limpio %d: %w", indiceBloques, err)
			}
		}

		// Reescribir todo el contenido en los bloques despues de limpiar
		err = Global.EscribirBloquesUsuarios(archivo, sb, inodoUsuarios, contenidoActualizado)
		if err != nil {
			return fmt.Errorf("error guardando los cambios en users.txt: %v", err)
		}
	} else {
		return fmt.Errorf("%s '%s' no encontrado en users.txt", tipoEntidad, nombre)
	}

	return nil
}
