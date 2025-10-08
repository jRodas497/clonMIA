package Forge

import (
    Estructuras "backend/Estructuras"
    Global "backend/Global"
    Utils "backend/Utils"
	
    "bytes"
    "errors"
    "fmt"
    "os"
    "regexp"
    "strings"
)

// EDIT estructura que representa el comando EDIT para modificar archivos
type EDIT struct {
    ruta      string // Ruta completa del archivo a modificar
    contenido string // Ruta del archivo que contiene el nuevo contenido
}

// ParserEdit analiza el comando edit y retorna una instancia de EDIT configurada
func ParserEdit(tokens []string) (string, error) {
    comando := &EDIT{}            // Crear nueva instancia del comando EDIT
    var bufferSalida bytes.Buffer // Buffer para recopilar mensajes de salida

    // Expresion regular para extraer parametros -ruta y -contenido
    expresionRegular := regexp.MustCompile(`-ruta="[^"]+"|-ruta=[^\s]+|-contenido="[^"]+"|-contenido=[^\s]+`)
    coincidencias := expresionRegular.FindAllString(strings.Join(tokens, " "), -1)

    // Validar que se proporcionaron los parametros minimos requeridos
    if len(coincidencias) < 2 {
        return "", errors.New("parametros insuficientes: se requieren -ruta y -contenido")
    }

    // Procesar cada coincidencia encontrada para extraer valores
    for _, coincidencia := range coincidencias {
        parClaveValor := strings.SplitN(coincidencia, "=", 2)
        clave := strings.ToLower(parClaveValor[0])
        valor := strings.Trim(parClaveValor[1], "\"") // Remover comillas si las hay

        // Asignar valores segun el parametro encontrado
        switch clave {
        case "-ruta":
            comando.ruta = valor
        case "-contenido":
            comando.contenido = valor
        }
    }

    // Verificar que ambos parametros fueron proporcionados
    if comando.ruta == "" || comando.contenido == "" {
        return "", errors.New("ambos parametros -ruta y -contenido son requeridos")
    }

    // Ejecutar la operacion de edicion
    err := comandoEdit(comando, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

// comandoEdit ejecuta la logica principal del comando edit
func comandoEdit(cmdEdit *EDIT, bufferSalida *bytes.Buffer) error {
    fmt.Fprint(bufferSalida, "======================= EDIT =======================\n")

    // Verificar estado de sesion del usuario
    if !Global.VerificarSesionActiva() {
        return fmt.Errorf("operacion denegada: no hay usuario autenticado")
    }

    // Obtener identificador de particion del usuario actual
    idParticion := Global.UsuarioActual.Id

    // Recuperar informacion de la particion montada
    superBloqueParticion, _, rutaParticion, err := Global.GetMountedPartitionSuperblock(idParticion)
    if err != nil {
        return fmt.Errorf("error obteniendo particion montada: %w", err)
    }

    // Abrir archivo de particion para operaciones de lectura/escritura
    archivo, err := os.OpenFile(rutaParticion, os.O_RDWR, 0666)
    if err != nil {
        return fmt.Errorf("error abriendo archivo de particion: %w", err)
    }
    defer archivo.Close() // Asegurar cierre del archivo

    // Analizar ruta para obtener directorios padre y nombre del archivo
    directoriosPadre, nombreArchivo := Utils.ObtenerDirectoriosPadre(cmdEdit.ruta)

    // Localizar el inodo del archivo objetivo
    indiceInodo, err := buscarInodoArchivo(archivo, superBloqueParticion, directoriosPadre, nombreArchivo)
    if err != nil {
        return fmt.Errorf("archivo no encontrado: %v", err)
    }

    // Leer contenido del archivo de reemplazo desde el sistema operativo
    contenidoNuevo, err := os.ReadFile(cmdEdit.contenido)
    if err != nil {
        return fmt.Errorf("error leyendo archivo de contenido '%s': %v", cmdEdit.contenido, err)
    }

    // Aplicar modificaciones al archivo en el sistema de archivos simulado
    err = modificarContenidoArchivo(archivo, superBloqueParticion, indiceInodo, contenidoNuevo)
    if err != nil {
        return fmt.Errorf("error modificando contenido del archivo: %v", err)
    }

    fmt.Fprintf(bufferSalida, "Archivo '%s' modificado exitosamente\n", nombreArchivo)
    fmt.Fprint(bufferSalida, "===================================================\n")

    return nil
}

// modificarContenidoArchivo actualiza el contenido de un archivo en el sistema de archivos
func modificarContenidoArchivo(archivo *os.File, sb *Estructuras.SuperBlock, indiceInodo int32, contenidoNuevo []byte) error {
    // Cargar inodo del archivo
    inodo := &Estructuras.INodo{}
    err := inodo.Decodificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error deserializando inodo %d: %v", indiceInodo, err)
    }

    // Validar que el inodo corresponde a un archivo
    if inodo.I_type[0] != '1' {
        return fmt.Errorf("inodo %d no es un archivo valido", indiceInodo)
    }

    // Limpiar bloques actuales del archivo para sobrescribir completamente
    for _, indiceBloques := range inodo.I_block {
        if indiceBloques != -1 {
            bloqueArchivo := &Estructuras.FileBlock{}
            bloqueArchivo.LimpiarContenido() // Vaciar contenido del bloque
            err := bloqueArchivo.Codificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
            if err != nil {
                return fmt.Errorf("error limpiando bloque %d: %v", indiceBloques, err)
            }
        }
    }

    // Fragmentar el nuevo contenido en bloques de 64 bytes
    bloques, err := Estructuras.DividirContenido(string(contenidoNuevo))
    if err != nil {
        return fmt.Errorf("error fragmentando contenido: %v", err)
    }

    // Escribir los nuevos bloques de contenido
    cantidadBloques := len(bloques)
    for i := 0; i < cantidadBloques; i++ {
        if i < len(inodo.I_block) {
            // Usar bloques ya asignados si existen
            indiceBloques := inodo.I_block[i]
            if indiceBloques == -1 {
                // Asignar nuevo bloque si no existe
                indiceBloques, err = sb.AsignarNuevoBloque(archivo, inodo, i)
                if err != nil {
                    return fmt.Errorf("error asignando bloque nuevo: %v", err)
                }
            }

            // Escribir contenido en el bloque actual
            err := bloques[i].Codificar(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)))
            if err != nil {
                return fmt.Errorf("error escribiendo bloque %d: %v", indiceBloques, err)
            }
        } else {
            // Manejar bloques adicionales usando bloques de apuntadores
            indicePointerBlock := inodo.I_block[len(inodo.I_block)-1]
            if indicePointerBlock == -1 {
                // Asignar nuevo bloque de apuntadores si no existe
                indicePointerBlock, err = sb.AsignarNuevoBloque(archivo, inodo, len(inodo.I_block)-1)
                if err != nil {
                    return fmt.Errorf("error asignando bloque de apuntadores: %v", err)
                }
            }

            // Cargar bloque de apuntadores existente
            pointerBlock := &Estructuras.PointerBlock{}
            err := pointerBlock.Decodificar(archivo, int64(sb.S_block_start+(indicePointerBlock*sb.S_block_size)))
            if err != nil {
                return fmt.Errorf("error decodificando bloque de apuntadores: %v", err)
            }

            // Encontrar posicion libre en el bloque de apuntadores
            indiceLibre, err := pointerBlock.BuscarApuntadorLibre()
            if err != nil {
                return fmt.Errorf("sin apuntadores libres disponibles: %v", err)
            }

            // Asignar nuevo bloque para contenido adicional
            nuevoIndiceBloques, err := sb.AsignarNuevoBloque(archivo, inodo, indiceLibre)
            if err != nil {
                return fmt.Errorf("error asignando bloque adicional: %v", err)
            }

            // Actualizar apuntador en el bloque de apuntadores
            err = pointerBlock.EstablecerApuntador(indiceLibre, int64(nuevoIndiceBloques))
            if err != nil {
                return fmt.Errorf("error actualizando apuntador: %v", err)
            }

            // Guardar bloque de apuntadores modificado
            err = pointerBlock.Codificar(archivo, int64(sb.S_block_start+(indicePointerBlock*sb.S_block_size)))
            if err != nil {
                return fmt.Errorf("error guardando bloque de apuntadores: %v", err)
            }

            // Escribir contenido en el nuevo bloque asignado
            err = bloques[i].Codificar(archivo, int64(sb.S_block_start+(nuevoIndiceBloques*sb.S_block_size)))
            if err != nil {
                return fmt.Errorf("error escribiendo nuevo bloque %d: %v", nuevoIndiceBloques, err)
            }
        }
    }

    // Actualizar tamano del archivo en el inodo
    inodo.I_size = int32(len(contenidoNuevo))
    err = inodo.Codificar(archivo, int64(sb.S_inode_start+(indiceInodo*sb.S_inode_size)))
    if err != nil {
        return fmt.Errorf("error actualizando inodo %d: %v", indiceInodo, err)
    }

    return nil
}