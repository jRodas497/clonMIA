package User

import (
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"strings"

	Estructuras "backend/Estructuras"
	Global "backend/Global"
)

// Estructura comando CHGRP
type CHGRP struct {
	Usuario string
	Grupo   string
}

// Parseo de argumentos para el comando chgrp
func ParserChgrp(tokens []string) (string, error) {
	// Inicializar el comando CHGRP
	var bufferSalida strings.Builder
	cmd := &CHGRP{}

	reUsuario := regexp.MustCompile(`-usr=[^\s]+`)
	reGrupo := regexp.MustCompile(`-grp=[^\s]+`)

	coincidenciaUsuario := reUsuario.FindString(strings.Join(tokens, " "))
	coincidenciaGrupo := reGrupo.FindString(strings.Join(tokens, " "))

	if coincidenciaUsuario == "" {
		return "", fmt.Errorf("falta el parametro -usr")
	}
	if coincidenciaGrupo == "" {
		return "", fmt.Errorf("falta el parametro -grp")
	}

	// Extraer los valores de los parametros
	cmd.Usuario = strings.SplitN(coincidenciaUsuario, "=", 2)[1]
	cmd.Grupo = strings.SplitN(coincidenciaGrupo, "=", 2)[1]

	// Ejecutar la logica del comando chgrp
	err := comandoChgrp(cmd, &bufferSalida)
	if err != nil {
		return "", err
	}

	return bufferSalida.String(), nil
}

// Ejecucion del comando CHGRP
func comandoChgrp(chgrp *CHGRP, bufferSalida *strings.Builder) error {
	fmt.Fprintln(bufferSalida, "---------------------------- CHGRP ----------------------------")

	if !Global.VerificarSesionActiva() {
		return fmt.Errorf("no hay ninguna sesion activa")
	}
	if Global.UsuarioActual.Nombre != "root" {
		return fmt.Errorf("solo el usuario root puede ejecutar este comando")
	}

	particion, ruta, err := Global.ObtenerParticionMontada(Global.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se puede encontrar la particion montada: %v", err)
	}

	archivo, err := os.OpenFile(ruta, os.O_RDWR, 0755)
	if err != nil {
		return fmt.Errorf("no se puede abrir el archivo de la particion: %v", err)
	}
	defer archivo.Close()

	// Cargar el SuperBlock usando el descriptor de archivo
	_, sb, _, err := Global.ObtenerParticionMontadaReporte(Global.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se pudo cargar el SuperBlock: %v", err)
	}

	var inodoUsuarios Estructuras.INodo
	desplazamientoInodo := int64(sb.S_inode_start + int32(binary.Size(inodoUsuarios))) //ubicacion de los bloques de users.txt
	err = inodoUsuarios.Decodificar(archivo, desplazamientoInodo)
	if err != nil {
		return fmt.Errorf("error leyendo el inodo de users.txt: %v", err)
	}

	err = CambiarGrupoUsuario(archivo, sb, &inodoUsuarios, chgrp.Usuario, chgrp.Grupo)
	if err != nil {
		return fmt.Errorf("error cambiando el grupo del usuario '%s': %v", chgrp.Usuario, err)
	}

	err = sb.Codificar(archivo, int64(particion.Part_start))
	if err != nil {
		return fmt.Errorf("error guardando el superbloque: %v", err)
	}

	fmt.Fprintf(bufferSalida, "El grupo del usuario '%s' ha sido cambiado exitosamente a '%s'\n", chgrp.Usuario, chgrp.Grupo)
	fmt.Fprintln(bufferSalida, "--------------------------------------------")
	return nil
}

// Cambia el grupo de un usuario en el archivo users.txt
func CambiarGrupoUsuario(archivo *os.File, sb *Estructuras.SuperBlock, inodoUsuarios *Estructuras.INodo, nombreUsuario, nuevoGrupo string) error {
	// Leer el contenido actual de users.txt
	contenidoActual, err := Global.LeerBloquesArchivo(archivo, sb, inodoUsuarios)
	if err != nil {
		return fmt.Errorf("error leyendo el contenido de users.txt: %w", err)
	}

	// Eliminar lineas vacias
	lineas := strings.Split(strings.TrimSpace(contenidoActual), "\n")
	var nuevoContenido []string
	var usuarioModificado bool
	var grupoEncontrado bool

	// Procesar el contenido del archivo
	var usuarios []Estructuras.Usuario
	var grupos []Estructuras.Grupo

	// Separar usuarios y grupos
	for _, linea := range lineas {
		partes := strings.Split(linea, ",")
		if len(partes) < 3 {
			continue // Saltar lineas mal formadas
		}

		// Identificar si es un grupo o un usuario
		tipo := strings.TrimSpace(partes[1])
		if tipo == "G" {
			// Crear un objeto de tipo Grupo
			grupo := Estructuras.NuevoGrupo(partes[0], partes[2])
			grupos = append(grupos, *grupo)
		} else if tipo == "U" && len(partes) >= 5 {
			// Crear un objeto de tipo Usuario
			usuario := Estructuras.NuevoUsuario(partes[0], partes[2], partes[3], partes[4])
			usuarios = append(usuarios, *usuario)
		}
	}

	// Verificar si el nuevo grupo existe y no esta eliminado
	var nuevoIDGrupo string
	for _, grupo := range grupos {
		if grupo.Grupo == nuevoGrupo && grupo.GID != "0" {
			nuevoIDGrupo = grupo.GID
			grupoEncontrado = true
			break
		}
	}

	if !grupoEncontrado {
		return fmt.Errorf("el grupo '%s' no existe o esta eliminado", nuevoGrupo)
	}

	// Modificar el grupo del usuario si existe
	for i, usuario := range usuarios {
		if usuario.Nombre == nombreUsuario && usuario.Id != "0" { // Verificar que el usuario no este eliminado
			// Cambiar el grupo del usuario y actualizar su ID al ID del nuevo grupo
			usuarios[i].Grupo = nuevoGrupo
			usuarios[i].Id = nuevoIDGrupo // Cambiar el ID del usuario al ID del grupo destino
			usuarioModificado = true
		}
	}

	if !usuarioModificado {
		return fmt.Errorf("el usuario '%s' no existe o esta eliminado", nombreUsuario)
	}

	for _, grupo := range grupos {
		nuevoContenido = append(nuevoContenido, grupo.ToString()) // Agregar grupo al contenido

		// Agregar usuarios asociados al grupo
		for _, usuario := range usuarios {
			if usuario.Grupo == grupo.Grupo {
				nuevoContenido = append(nuevoContenido, usuario.ToString())
			}
		}
	}

	// Limpiar los bloques asignados antes de escribir el nuevo contenido
	for _, indiceBloques := range inodoUsuarios.I_block {
		if indiceBloques == -1 {
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

	err = EscribirContenidoEnBloques(archivo, sb, inodoUsuarios, nuevoContenido)
	if err != nil {
		return fmt.Errorf("error guardando los cambios en users.txt: %v", err)
	}

	inodoUsuarios.I_size = int32(len(strings.Join(nuevoContenido, "\n")))

	inodoUsuarios.ActualizarTiempoModificacion()
	inodoUsuarios.ActualizarTiempoPermisos()

	// Guardar el inodo actualizado en el archivo
	desplazamientoInodo := int64(sb.S_inode_start + int32(binary.Size(*inodoUsuarios)))
	err = inodoUsuarios.Codificar(archivo, desplazamientoInodo)
	if err != nil {
		return fmt.Errorf("error actualizando inodo de users.txt: %w", err)
	}

	return nil
}

// Escribe el contenido de users.txt dividiendolo
func EscribirContenidoEnBloques(archivo *os.File, sb *Estructuras.SuperBlock, inodoUsuarios *Estructuras.INodo, contenido []string) error {
	// Convertir el contenido en una cadena
	contenidoFinal := strings.Join(contenido, "\n") + "\n"
	datos := []byte(contenidoFinal)

	// Dimension maxima del bloque
	dimensionBloque := int(sb.S_block_size)

	// Escribir el contenido por bloques
	for i, indiceBloques := range inodoUsuarios.I_block {
		if indiceBloques == -1 {
			break // No hay mas bloques asignados
		}

		// Dividir los datos en bloques de dimension maxima
		inicio := i * dimensionBloque
		fin := inicio + dimensionBloque
		if fin > len(datos) {
			fin = len(datos)
		}

		// Crear un bloque con el contenido correspondiente
		var bloqueArchivo Estructuras.FileBlock
		copy(bloqueArchivo.B_cont[:], datos[inicio:fin])

		desplazamientoBloque := int64(sb.S_block_start + indiceBloques*sb.S_block_size)

		err := bloqueArchivo.Codificar(archivo, desplazamientoBloque)
		if err != nil {
			return fmt.Errorf("error escribiendo bloque %d: %w", indiceBloques, err)
		}
	}

	return nil
}
