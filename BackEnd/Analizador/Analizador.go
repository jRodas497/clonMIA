package Analizador

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	Disk "backend/Comandos/Disk"
	Forge "backend/Comandos/Forge"
	User "backend/Comandos/User"
)

// Funcion principal que procesa las entradas del usuario
func Analizador(entrada string) (string, error) {
	if strings.HasPrefix(strings.TrimSpace(entrada), "#") {
		return fmt.Sprintf("Comentario procesado: %s", entrada), nil
	}

	// Dividir la entrada en tokens individuales
	tokens := strings.Fields(entrada)
	if len(tokens) == 0 {
		return "", errors.New("entrada vacia proporcionada")
	}

	// Buscar el comando en el mapa de funciones
	funcionComando, existe := mapaComandos[tokens[0]]
	if !existe {
		switch tokens[0] {

		case "clear":
			return limpiarTerminal()
		case "exit":
			os.Exit(0)
		case "help":
			return mostrarAyuda(nil)
		}

		return "", fmt.Errorf("comando no reconocido: %s", tokens[0])
	}

	// Invocar la funcion asociada al comando
	return funcionComando(tokens[1:])
}

// Diccionario que asocia comandos con sus funciones correspondientes
var mapaComandos = map[string]func([]string) (string, error){
	"mkdisk": func(argumentos []string) (string, error) {
		resultado, err := Disk.ParserMkdisk(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"rmdisk": func(argumentos []string) (string, error) {
		resultado, err := Disk.ParserRmdisk(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"fdisk": func(argumentos []string) (string, error) {
		resultado, err := Disk.ParserFdisk(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"mount": func(argumentos []string) (string, error) {
		resultado, err := Disk.ParserMount(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"mounted": func(args []string) (string, error) {
		result, err := Disk.ParserMounted(args)
		return fmt.Sprintf("%v", result), err
	},
	"mkfs": func(argumentos []string) (string, error) {
		resultado, err := Disk.ParserMkfs(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"login": func(argumentos []string) (string, error) {
		resultado, err := User.ParserLogin(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"logout": func(argumentos []string) (string, error) {
		resultado, err := User.ParserLogout(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"mkgrp": func(argumentos []string) (string, error) {
		resultado, err := User.ParserMkgrp(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"rmgrp": func(argumentos []string) (string, error) {
		resultado, err := User.ParserRmgrp(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"mkusr": func(argumentos []string) (string, error) {
		resultado, err := User.ParserMkusr(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"rmusr": func(argumentos []string) (string, error) {
		resultado, err := User.ParserRmusr(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"chgrp": func(argumentos []string) (string, error) {
		resultado, err := User.ParserChgrp(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"mkdir": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserMkdir(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"mkfile": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserMkfile(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"cat": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserCat(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"remove": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserRemove(argumentos)
		return fmt.Sprintf("%v", resultado), err
	}
	"unmount": func(argumentos []string) (string, error) {
		resultado, err := Disk.ParserUnmount(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"edit": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserEdit(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"rename": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserRename(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"copy": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserCopy(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"move": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserMove(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"find": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserFind(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"chown": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserChown(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"chmod": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserChmod(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"journaling": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserJournaling(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"loss": func(args []string) (string, error) {
		result, err := Forge.ParserLoss(args)
		return fmt.Sprintf("%v", result), err
	},
	"recovery": func(args []string) (string, error) {
		result, err := Forge.ParserRecovery(args)
		return fmt.Sprintf("%v", result), err
	},
	"rep": func(argumentos []string) (string, error) {
		resultado, err := Forge.ParserRep(argumentos)
		return fmt.Sprintf("%v", resultado), err
	},
	"help": mostrarAyuda,
}

// Muestra informacion de ayuda sobre comandos disponibles
func mostrarAyuda(argumentos []string) (string, error) {
    mensajeAyuda := `
Lista de comandos disponibles en el sistema:

GESTION DE DISCOS:
- mkdisk: Genera un nuevo disco virtual
  Sintaxis: mkdisk -size=100 -unit=M -fit=FF -path="/ruta/archivo.mia"

- rmdisk: Elimina un disco virtual existente
  Sintaxis: rmdisk -path="/ruta/archivo.mia"

- fdisk: Administra particiones en el disco
  Sintaxis: fdisk -size=50 -unit=M -path="/ruta/archivo.mia" -type=P -name="Particion1"

- mount: Monta una particion en el sistema
  Sintaxis: mount -path="/ruta/archivo.mia" -name="Particion1"

- unmount: Desmonta una particion del sistema
  Sintaxis: unmount -id=vd1

- mounted: Lista las particiones montadas
  Sintaxis: mounted

- mkfs: Aplica formato a una particion
  Sintaxis: mkfs -id=vd1 -type=full

ADMINISTRACION DE USUARIOS:
- login: Accede al sistema con credenciales
  Sintaxis: login -user=admin -pass=1234 -id=vd1

- logout: Termina la sesion actual
  Sintaxis: logout

- mkgrp: Registra un nuevo grupo
  Sintaxis: mkgrp -name=usuarios

- rmgrp: Remueve un grupo del sistema
  Sintaxis: rmgrp -name=usuarios

- mkusr: Crea una nueva cuenta de usuario
  Sintaxis: mkusr -user=usuario1 -pass=clave -grp=usuarios

- rmusr: Elimina una cuenta de usuario
  Sintaxis: rmusr -user=usuario1

- chgrp: Modifica el grupo de un usuario
  Sintaxis: chgrp -user=usuario1 -grp=usuarios

MANEJO DE ARCHIVOS Y DIRECTORIOS:
- mkdir: Genera un directorio
  Sintaxis: mkdir -path="/home/carpeta" -p

- mkfile: Crea un nuevo archivo
  Sintaxis: mkfile -path="/home/archivo.txt" -size=100 -cont="contenido"

- cat: Muestra el contenido de archivos
  Sintaxis: cat -file="/home/archivo.txt"

- remove: Elimina archivos o directorios
  Sintaxis: remove -path="/home/archivo.txt"

- edit: Modifica el contenido de un archivo
  Sintaxis: edit -path="/home/archivo.txt" -cont="nuevo contenido"

- rename: Cambia el nombre de archivos o directorios
  Sintaxis: rename -path="/home/archivo.txt" -name="nuevo_nombre.txt"

- copy: Copia archivos o directorios
  Sintaxis: copy -path="/home/origen.txt" -dest="/home/destino.txt"

- move: Mueve archivos o directorios
  Sintaxis: move -path="/home/archivo.txt" -dest="/home/nueva_ubicacion/"

- find: Busca archivos y directorios
  Sintaxis: find -path="/home" -name="archivo.txt"

PERMISOS Y PROPIEDADES:
- chown: Cambia el propietario de archivos/directorios
  Sintaxis: chown -path="/home/archivo.txt" -user=usuario1 -r

- chmod: Modifica permisos de archivos/directorios
  Sintaxis: chmod -path="/home/archivo.txt" -ugo=755 -r

SISTEMA EXT3 Y RECUPERACION:
- journaling: Muestra el historial de transacciones EXT3
  Sintaxis: journaling -id=vd1

- loss: Simula perdida de datos en el sistema
  Sintaxis: loss -id=vd1

- recovery: Recupera el sistema usando journaling
  Sintaxis: recovery -id=vd1

REPORTES Y HERRAMIENTAS:
- rep: Produce reportes del sistema
  Sintaxis: rep -id=vd1 -path="/ruta/reporte.png" -name=mbr

- clear: Limpia la pantalla de la terminal

- exit: Finaliza la ejecucion del programa

- help: Presenta esta informacion de ayuda

PARAMETROS COMUNES:
- -path: Ruta del archivo o directorio
- -id: Identificador de particion montada
- -r: Aplicar recursivamente
- -p: Crear directorios padre si no existen
- -size: Tamaño en bytes
- -unit: Unidad (B, K, M)
- -user: Nombre de usuario
- -pass: Contraseña
- -grp: Nombre de grupo
- -name: Nombre del elemento
- -cont: Contenido del archivo
- -dest: Destino para operaciones de copia/movimiento
- -ugo: Permisos en formato UGO (ej: 755)

`
    return mensajeAyuda, nil
}

// ...existing code...

// Limpia el contenido de la terminal segun el sistema operativo
func limpiarTerminal() (string, error) {
	var args []string
	var cmdName string

	if runtime.GOOS == "windows" {
		cmdName = "cmd"
		args = []string{"/c", "cls"}
	} else {
		cmdName = "clear"
		args = []string{}
	}

	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("no se pudo limpiar la terminal: %v", err)
	}
	return "Terminal limpiada correctamente", nil
}


/*
COMANDOS QUE ESTÁN EN 202100106 Y NO EN TU P2:

1. JOURNALING
   - Comando: journaling
   - Función: Muestra el historial de transacciones del sistema EXT3
   - Sintaxis: journaling -id=vd1

2. LSBLK  
   - Comando: lsblk
   - Función: Lista todas las particiones de un disco con información detallada
   - Sintaxis: lsblk -path="/ruta/disco.mia"

3. FDISK EXTENDIDO
   - Parámetros adicionales que no tienes:
     * -delete (Fast/Full) - Para eliminar particiones
     * -add (positivo/negativo) - Para agregar/quitar espacio a particiones
   - Tu fdisk actual solo crea particiones, no las modifica ni elimina

4. COMANDOS DE SISTEMA DE ARCHIVOS AVANZADOS (mencionados en el enunciado):
   - remove: Eliminar archivos/directorios
   - edit: Editar contenido de archivos
   - copy: Copiar archivos
   - chown: Cambiar propietario de archivos

5. COMANDOS EXT3 ESPECÍFICOS:
   - Versiones de mkdir, mkfile que escriben al journal
   - Comandos que registran operaciones en el sistema de journaling

ESTRUCTURAS QUE FALTAN:
- Journal struct con j_count y j_content
- JournalInfo struct con i_operation, i_path, i_content, i_date
- Soporte EXT3 en SuperBloque (S_filesystem_type = 3)

FUNCIONALIDADES FALTANTES:
- Sistema de journaling integrado en comandos existentes
- Escritura automática al journal en operaciones de archivos
- Lectura y visualización del historial de transacciones
*/