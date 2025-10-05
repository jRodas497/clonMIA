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
type MKUSR struct {
    Usuario    string
    Contrasena string
    Grupo      string
}

// Valida que los parametros no excedan una longitud maxima
func validarLongitudParametro(param string, longitudMaxima int, nombreParam string) error {
    if len(param) > longitudMaxima {
        return fmt.Errorf("%s debe tener un maximo de %d caracteres", nombreParam, longitudMaxima)
    }
    return nil
}

// Parseo de argumentos para el comando y captura de mensajes
func ParserMkusr(tokens []string) (string, error) {
    var bufferSalida bytes.Buffer

    cmd := &MKUSR{}

    // Expresion regular para encontrar los parametros -user, -pass, -grp
    reUsuario := regexp.MustCompile(`-user=[^\s]+`)
    reContrasena := regexp.MustCompile(`-pass=[^\s]+`)
    reGrupo := regexp.MustCompile(`-grp=[^\s]+`)

    // Buscar parametros
    coincidenciaUsuario := reUsuario.FindString(strings.Join(tokens, " "))
    coincidenciaContrasena := reContrasena.FindString(strings.Join(tokens, " "))
    coincidenciaGrupo := reGrupo.FindString(strings.Join(tokens, " "))

    // Verificar que se proporcionen los parametros
    if coincidenciaUsuario == "" {
        return "", fmt.Errorf("falta el parametro -user")
    }
    if coincidenciaContrasena == "" {
        return "", fmt.Errorf("falta el parametro -pass")
    }
    if coincidenciaGrupo == "" {
        return "", fmt.Errorf("falta el parametro -grp")
    }

    cmd.Usuario = strings.SplitN(coincidenciaUsuario, "=", 2)[1]
    cmd.Contrasena = strings.SplitN(coincidenciaContrasena, "=", 2)[1]
    cmd.Grupo = strings.SplitN(coincidenciaGrupo, "=", 2)[1]

    if err := validarLongitudParametro(cmd.Usuario, 10, "Usuario"); err != nil {
        return "", err
    }
    if err := validarLongitudParametro(cmd.Contrasena, 10, "Contrasena"); err != nil {
        return "", err
    }
    if err := validarLongitudParametro(cmd.Grupo, 10, "Grupo"); err != nil {
        return "", err
    }

    err := comandoMkusr(cmd, &bufferSalida)
    if err != nil {
        return "", err
    }

    return bufferSalida.String(), nil
}

// Ejecuta el comando con captura de mensajes en buffer
func comandoMkusr(mkusr *MKUSR, bufferSalida *bytes.Buffer) error {
    fmt.Fprintln(bufferSalida, "---------------------------- MKUSR ----------------------------")
    
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

    // Cargar SuperBlock y particion utilizando la funcion ObtenerParticionMontadaRep
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
    desplazamientoInodo := int64(sb.S_inode_start + int32(binary.Size(inodoUsuarios))) //ubicacion de los bloques de users.txt
    err = inodoUsuarios.Decodificar(archivo, desplazamientoInodo)
    if err != nil {
        return fmt.Errorf("error leyendo el inodo de users.txt: %v", err)
    }

    _, err = Global.BuscarEnArchivoUsuarios(archivo, sb, &inodoUsuarios, mkusr.Grupo, "G")
    if err != nil {
        return fmt.Errorf("el grupo '%s' no existe", mkusr.Grupo)
    }

    _, err = Global.BuscarEnArchivoUsuarios(archivo, sb, &inodoUsuarios, mkusr.Usuario, "U")
    if err == nil {
        return fmt.Errorf("el usuario '%s' ya existe", mkusr.Usuario)
    }

    usuario := Estructuras.NuevoUsuario(fmt.Sprintf("%d", sb.S_inodes_count+1), mkusr.Grupo, mkusr.Usuario, mkusr.Contrasena)

    // Insertar la nueva entrada en el archivo users.txt
    err = Global.InsertarEnArchivoUsuarios(archivo, sb, &inodoUsuarios, usuario.ToString())
    if err != nil {
        return fmt.Errorf("error insertando el usuario '%s': %v", mkusr.Usuario, err)
    }

    // Actualizar el inodo de users.txt
    err = inodoUsuarios.Codificar(archivo, desplazamientoInodo)
    if err != nil {
        return fmt.Errorf("error actualizando inodo de users.txt: %v", err)
    }

    // Guardar SuperBlock usando Part_start como el offset
    err = sb.Codificar(archivo, int64(particion.Part_start))
    if err != nil {
        return fmt.Errorf("error guardando el SuperBlock: %v", err)
    }

    fmt.Fprintf(bufferSalida, "Usuario '%s' agregado exitosamente al grupo '%s'\n", mkusr.Usuario, mkusr.Grupo)
    fmt.Fprintf(bufferSalida, "--------------------------------------------")

    return nil
}
