package User

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	Estructuras "backend/Estructuras"
	Global "backend/Global"
)

// MKGRP : Estructura para el comando MKGRP
type MKGRP struct {
	Nombre string
}

// AnalizarMkgrp : Parseo de argumentos para el comando mkgrp y captura de los mensajes importantes
func ParserMkgrp(tokens []string) (string, error) {
	var bufferSalida bytes.Buffer

	// Inicializar el comando MKGRP
	cmd := &MKGRP{}

	// Expresion regular para encontrar el parametro -name
	re := regexp.MustCompile(`-name=[^\s]+`)
	coincidencia := re.FindString(strings.Join(tokens, " "))

	if coincidencia == "" {
		return "", fmt.Errorf("falta el parametro -name")
	}

	// Extraer el valor del parametro -name
	parametro := strings.SplitN(coincidencia, "=", 2)
	if len(parametro) != 2 {
		return "", fmt.Errorf("formato incorrecto para -name")
	}
	cmd.Nombre = parametro[1]

	err := comandoMkgrp(cmd, &bufferSalida)
	if err != nil {
		return "", err
	}

	return bufferSalida.String(), nil
}

func comandoMkgrp(mkgrp *MKGRP, bufferSalida *bytes.Buffer) error {
	fmt.Fprintln(bufferSalida, "---------------------------- MKGRP ----------------------------")

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

	// Obtener la particion asociada al id
	particion, err := mbr.ObtenerParticionPorID(Global.UsuarioActual.Id)
	if err != nil {
		return fmt.Errorf("no se pudo obtener la particion: %v", err)
	}

	// Leer el inodo de users.txt
	var inodoUsuarios Estructuras.INodo
	// Calcular el offset del inodo de users.txt, esta en el inodo 1
	desplazamientoInodo := int64(sb.S_inode_start + int32(binary.Size(inodoUsuarios)))
	// Decodificar el inodo de users.txt
	err = inodoUsuarios.Decodificar(archivo, desplazamientoInodo)
	inodoUsuarios.ActualizarTiempoAcceso()
	if err != nil {
		return fmt.Errorf("error leyendo el inodo de users.txt: %v", err)
	}

	// Verificar si el grupo ya existe en users.txt
	_, err = Global.BuscarEnArchivoUsuarios(archivo, sb, &inodoUsuarios, mkgrp.Nombre, "G")
	if err == nil {
		return fmt.Errorf("el grupo '%s' ya existe", mkgrp.Nombre)
	}

	// Obtener el siguiente ID disponible para el nuevo grupo
	siguienteIDGrupo, err := calcularSiguienteID(archivo, sb, &inodoUsuarios)
	if err != nil {
		return fmt.Errorf("error calculando el siguiente ID: %v", err)
	}

	// Crear la nueva entrada de grupo con el siguiente ID
	nuevaEntradaGrupo := fmt.Sprintf("%d,G,%s", siguienteIDGrupo, mkgrp.Nombre)

	// Usar la funcion modular para crear el grupo en users.txt
	err = Global.AgregarEntradaArchivoUsuarios(archivo, sb, &inodoUsuarios, nuevaEntradaGrupo, mkgrp.Nombre, "G")
	if err != nil {
		return fmt.Errorf("error creando el grupo '%s': %v", mkgrp.Nombre, err)
	}

	// Actualizar el inodo de users.txt
	err = inodoUsuarios.Codificar(archivo, desplazamientoInodo)
	inodoUsuarios.ActualizarTiempoAcceso()
	if err != nil {
		return fmt.Errorf("error actualizando inodo de users.txt: %v", err)
	}

	// Guardar el SuperBlock utilizando el Part_start como el offset
	err = sb.Codificar(archivo, int64(particion.Part_start))
	if err != nil {
		return fmt.Errorf("error guardando el SuperBlock: %v", err)
	}

	fmt.Fprintf(bufferSalida, "Grupo creado exitosamente: %s\n", mkgrp.Nombre)
	fmt.Fprintf(bufferSalida, "--------------------------------------------")
	return nil
}

// Calcula el siguiente ID disponible para un grupo o usuario en users.txt
func calcularSiguienteID(archivo *os.File, sb *Estructuras.SuperBlock, inodo *Estructuras.INodo) (int, error) {
	// Leer el contenido de users.txt
	contenido, err := Global.LeerBloquesArchivo(archivo, sb, inodo)
	if err != nil {
		return -1, fmt.Errorf("error leyendo el contenido de users.txt: %v", err)
	}

	lineas := strings.Split(contenido, "\n")
	maxID := 0
	for _, linea := range lineas {
		if linea == "" {
			continue
		}

		campos := strings.Split(linea, ",")
		if len(campos) < 3 {
			// Ignorar lineas mal formadas
			continue
		}

		// Convertir el primer campo (ID) a entero
		id, err := strconv.Atoi(campos[0])
		if err != nil {
			// Ignorar IDs mal formados
			continue
		}

		// Actualizar el maxID si encontramos uno mayor
		if id > maxID {
			maxID = id
		}
	}

	// Devolver el siguiente ID disponible
	return maxID + 1, nil
}
