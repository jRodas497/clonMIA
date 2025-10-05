package Global

import (
	"fmt"
	"os"
	"strings"

	Estructuras "backend/Estructuras"
)

// Leer todos los bloques asignados a un archivo y devuelve su contenido completo
func LeerBloquesArchivo(archivo *os.File, sb *Estructuras.SuperBlock, inodo *Estructuras.INodo) (string, error) {
	var contenido string

	for _, indiceBloques := range inodo.I_block {
		if indiceBloques == -1 {
			break
		}

		desplazamientoBloque := int64(sb.S_block_start + indiceBloques*int32(sb.S_block_size))
		var bloqueArchivo Estructuras.FileBlock

		// Leer el bloque desde el archivo
		err := bloqueArchivo.Decodificar(archivo, desplazamientoBloque)
		if err != nil {
			return "", fmt.Errorf("error leyendo bloque %d: %w", indiceBloques, err)
		}

		// Concatenar el contenido del bloque al resultado total
		contenido += string(bloqueArchivo.B_cont[:])
	}

	inodo.ActualizarTiempoAcceso()

	return strings.TrimRight(contenido, "\x00"), nil
}

func EscribirBloquesUsuarios(archivo *os.File, sb *Estructuras.SuperBlock, inodo *Estructuras.INodo, nuevoContenido string) error {
	contenidoExistente, err := LeerBloquesArchivo(archivo, sb, inodo)
	if err != nil {
		return fmt.Errorf("error leyendo contenido existente de users.txt: %w", err)
	}

	// Combinar el contenido existente con el nuevo contenido
	contenidoTotal := contenidoExistente + nuevoContenido

	// Dividir el contenido total en bloques de dimension BlockSize
	bloques, err := Estructuras.DividirContenido(contenidoTotal)
	if err != nil {
		return fmt.Errorf("error al dividir el contenido en bloques: %w", err)
	}

	indice := 0

	for _, bloque := range bloques {
		// Verifica si el indice excede la capacidad del array I_block
		if indice >= len(inodo.I_block) {
			return fmt.Errorf("se alcanzo el limite maximo de bloques del inodo")
		}

		// Si el bloque actual en el inodo esta vacio, asignar uno nuevo
		if inodo.I_block[indice] == -1 {
			nuevoIndiceBloques, err := sb.AsignarNuevoBloque(archivo, inodo, indice)
			if err != nil {
				return fmt.Errorf("error asignando nuevo bloque: %w", err)
			}
			inodo.I_block[indice] = nuevoIndiceBloques
		}

		desplazamientoBloque := int64(sb.S_block_start + inodo.I_block[indice]*int32(sb.S_block_size))

		// Escribir el contenido del bloque en la particion
		err = bloque.Codificar(archivo, desplazamientoBloque)
		if err != nil {
			return fmt.Errorf("error escribiendo el bloque %d: %w", inodo.I_block[indice], err)
		}

		// Mover al siguiente bloque
		indice++
	}

	// Actualizar la dimension del archivo en el inodo (i_size)
	nuevaDimension := len(contenidoTotal)
	inodo.I_size = int32(nuevaDimension)

	// Actualizar los tiempos de modificacion y cambio
	inodo.ActualizarTiempoModificacion()
	inodo.ActualizarTiempoPermisos()

	return nil
}

// Inserta una nueva entrada en el archivo users.txt
func InsertarEnArchivoUsuarios(archivo *os.File, sb *Estructuras.SuperBlock, inodo *Estructuras.INodo, entrada string) error {
	contenidoActual, err := LeerBloquesArchivo(archivo, sb, inodo)
	if err != nil {
		return fmt.Errorf("error leyendo el contenido de users.txt: %w", err)
	}

	// Eliminar lineas vacias o con espacios innecesarios del contenido actual
	lineas := strings.Split(strings.TrimSpace(contenidoActual), "\n")

	// Obtener el grupo desde la nueva entrada
	partesEntrada := strings.Split(entrada, ",")
	if len(partesEntrada) < 4 { // Se espera al menos UID, U, Grupo, Usuario, Contrasena
		return fmt.Errorf("entrada de usuario invalida: %s", entrada)
	}
	grupoUsuario := partesEntrada[2] // El grupo del usuario se encuentra en la tercera posicion

	// Buscar el ID del grupo correspondiente en el contenido actual
	var idGrupo string
	var nuevoContenido []string
	usuarioInsertado := false

	for _, linea := range lineas {
		partes := strings.Split(linea, ",")
		// Agregar la linea actual al nuevo contenido
		nuevoContenido = append(nuevoContenido, strings.TrimSpace(linea))

		// Si encontramos el grupo correcto
		if len(partes) > 2 && partes[1] == "G" && partes[2] == grupoUsuario {
			idGrupo = partes[0] // Obtener el ID del grupo

			// Insertar el usuario justo despues del grupo si no se ha insertado ya
			if idGrupo != "" && !usuarioInsertado {
				usuarioConGrupo := fmt.Sprintf("%s,U,%s,%s,%s", idGrupo, partesEntrada[2], partesEntrada[3], partesEntrada[4])
				nuevoContenido = append(nuevoContenido, usuarioConGrupo)
				usuarioInsertado = true
			}
		}
	}

	// Verificar si el grupo fue encontrado
	if idGrupo == "" {
		return fmt.Errorf("el grupo '%s' no existe", grupoUsuario)
	}

	contenidoNuevo := strings.Join(nuevoContenido, "\n") + "\n"

	// Limpiar los bloques asignados al archivo
	for _, indiceBloques := range inodo.I_block {
		if indiceBloques == -1 {
			break // No hay mas bloques asignados
		}

		desplazamientoBloque := int64(sb.S_block_start + indiceBloques*sb.S_block_size)
		var bloqueArchivo Estructuras.FileBlock

		bloqueArchivo.LimpiarContenido()

		err = bloqueArchivo.Codificar(archivo, desplazamientoBloque)
		if err != nil {
			return fmt.Errorf("error escribiendo bloque limpio %d: %w", indiceBloques, err)
		}
	}

	// Reescribir todo el contenido linea por linea
	err = EscribirBloquesUsuarios(archivo, sb, inodo, contenidoNuevo)
	if err != nil {
		return fmt.Errorf("error escribiendo el nuevo contenido en users.txt: %w", err)
	}

	inodo.I_size = int32(len(contenidoNuevo))

	// Actualizar tiempos de modificacion y cambio
	inodo.ActualizarTiempoModificacion()
	inodo.ActualizarTiempoPermisos()

	return nil
}

// Entrada al archivo users.txt (ya sea grupo o usuario)
func AgregarEntradaArchivoUsuarios(archivo *os.File, sb *Estructuras.SuperBlock, inodo *Estructuras.INodo, entrada, nombre, tipoEntidad string) error {
	// Leer el contenido actual de users.txt
	contenidoActual, err := LeerBloquesArchivo(archivo, sb, inodo)
	if err != nil {
		return fmt.Errorf("error leyendo bloques de users.txt: %w", err)
	}

	// Verificar si el grupo/usuario ya existe
	_, _, err = buscarLineaEnArchivoUsuarios(contenidoActual, nombre, tipoEntidad)
	if err == nil {
		return nil
	}

	// Escribir solo la nueva entrada al final de los bloques
	err = EscribirBloquesUsuarios(archivo, sb, inodo, entrada+"\n") // Solo el nuevo grupo
	if err != nil {
		return fmt.Errorf("error agregando entrada a users.txt: %w", err)
	}

	return nil
}

// Nuevo grupo en el archivo users.txt
func CrearGrupo(archivo *os.File, sb *Estructuras.SuperBlock, inodo *Estructuras.INodo, nombreGrupo string) error {
	entradaGrupo := fmt.Sprintf("%d,G,%s", sb.S_inodes_count+1, nombreGrupo)
	return AgregarEntradaArchivoUsuarios(archivo, sb, inodo, entradaGrupo, nombreGrupo, "G")
}

// Nuevo usuario en el archivo users.txt
func CrearUsuario(archivo *os.File, sb *Estructuras.SuperBlock, inodo *Estructuras.INodo, nombreUsuario, contrasenaUsuario, nombreGrupo string) error {
	entradaUsuario := fmt.Sprintf("%d,U,%s,%s,%s", sb.S_inodes_count+1, nombreUsuario, nombreGrupo, contrasenaUsuario)
	return AgregarEntradaArchivoUsuarios(archivo, sb, inodo, entradaUsuario, nombreUsuario, "U")
}

// Busca una entrada en el archivo users.txt segun nombre y tipo
func BuscarEnArchivoUsuarios(archivo *os.File, sb *Estructuras.SuperBlock, inodo *Estructuras.INodo, nombre, tipoEntidad string) (string, error) {
	contenido, err := LeerBloquesArchivo(archivo, sb, inodo)
	if err != nil {
		return "", err
	}

	// Usamos la funcion auxiliar para buscar la linea
	linea, _, err := buscarLineaEnArchivoUsuarios(contenido, nombre, tipoEntidad)
	if err != nil {
		return "", err
	}

	return linea, nil
}

// Linea en el archivo users.txt segun nombre y tipo
func buscarLineaEnArchivoUsuarios(contenido string, nombre, tipoEntidad string) (string, int, error) {
	lineas := strings.Split(contenido, "\n")

	for i, linea := range lineas {
		campos := strings.Split(linea, ",")
		if len(campos) < 3 {
			// Ignorar lineas mal formadas
			continue
		}

		// Determinar si es un grupo o un usuario segun el tipoEntidad
		if tipoEntidad == "G" && len(campos) == 3 {
			// Crear instancia de Grupo
			grupo := Estructuras.NuevoGrupo(campos[0], campos[2])
			if grupo.Tipo == tipoEntidad && grupo.Grupo == nombre {
				// Devolver la linea y el indice
				return grupo.ToString(), i, nil
			}
		} else if tipoEntidad == "U" && len(campos) == 5 {
			// Es un usuario
			usuario := Estructuras.NuevoUsuario(campos[0], campos[2], campos[3], campos[4]) // Crear instancia de Usuario
			if usuario.Tipo == tipoEntidad && usuario.Nombre == nombre {
				return usuario.ToString(), i, nil
			}
		}
	}

	return "", -1, fmt.Errorf("%s '%s' no encontrado en users.txt", tipoEntidad, nombre)
}
