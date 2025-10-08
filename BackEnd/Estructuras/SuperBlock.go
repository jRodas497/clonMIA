package Estructuras

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	Utils "backend/Utils"
)

type SuperBlock struct {
	S_filesystem_type   int32   /*  Numero que identifica el sistema de archivos usado  */
	S_inodes_count      int32   /*  Numero total de inodos creados  */
	S_blocks_count      int32   /*  Numero total de bloques creados  */
	S_free_blocks_count int32   /*  Numero de bloques libres  */
	S_free_inodes_count int32   /*  Numero de inodos libres  */
	S_mtime             float64 /*  Ultima fecha en que el sistema fue montado  */
	S_umtime            float64 /*  Ultima fecha en que el sistema fue desmontado  */
	S_mnt_count         int32   /*  Numero de veces que se ha montado el sistema  */
	S_magic             int32   /*  Valor que identifica el sistema de archivos  */
	S_inode_size        int32   /*  Dimension de la estructura inodo  */
	S_block_size        int32   /*  Dimension de la estructura bloque  */
	S_first_ino         int32   /*  Primer inodo libre  */
	S_first_blo         int32   /*  Primer bloque libre  */
	S_bm_inode_start    int32   /*  Inicio del bitmap de inodos  */
	S_bm_block_start    int32   /*  Inicio del bitmap de bloques  */
	S_inode_start       int32   /*  Inicio de la tabla de inodos  */
	S_block_start       int32   /*  Inicio de la tabla de bloques  */
}

/*  Serializa la estructura SuperBlock en un archivo  */
func (sb *SuperBlock) Codificar(archivo *os.File, desplazamiento int64) error {
	return Utils.EscribirAArchivo(archivo, desplazamiento, sb)
}

/*  Deserializa la estructura SuperBlock desde un archivo  */
func (sb *SuperBlock) Decodificar(archivo *os.File, desplazamiento int64) error {
	return Utils.LeerDeArchivo(archivo, desplazamiento, sb)
}

// InicioJournal retorna el byte donde inicia el journal
func (sb *SuperBlock) InicioJournal() int32 {
	// El journal está justo antes del inicio del bitmap de inodos
	journalSize := int32(binary.Size(Journal{}))
	start := sb.S_bm_inode_start - ENTRADAS_JOURNAL*journalSize
	fmt.Printf("[DEBUG] Superblock.InicioJournal: bm_inode_start=%d, journalSize=%d, entries=%d -> start=%d\n",
		sb.S_bm_inode_start, journalSize, ENTRADAS_JOURNAL, start)
	return start
}

// FinJournal calcula el final del área de journaling
func (sb *SuperBlock) FinJournal() int32 {
	end := sb.S_bm_inode_start
	fmt.Printf("[DEBUG] Superblock.FinJournal: bm_inode_start=%d -> end=%d\n",
		sb.S_bm_inode_start, end)
	return end
}

func (sb *SuperBlock) CrearArchivoUsuarios(archivo *os.File) error {
	inodoRaiz := &INodo{
		I_uid:   1,
		I_gid:   1,
		I_size:  0,
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{sb.S_blocks_count, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		I_type:  [1]byte{'0'},
		I_perm:  [3]byte{'7', '7', '7'},
	}

	err := Utils.EscribirAArchivo(archivo, int64(sb.S_inode_start), inodoRaiz)
	if err != nil {
		return fmt.Errorf("error al escribir el inodo 0: %w", err)
	}

	err = sb.ActualizarBitmapInodo(archivo, 0, true)
	if err != nil {
		return fmt.Errorf("error al actualizar bitmap de inodos: %w", err)
	}

	// Actualizar el contador de inodos y el puntero al primer inodo libre
	sb.ActualizarSuperblockDespuesAsignacionInodo()

	bloqueRaiz := &FolderBlock{
		B_cont: [4]FolderContent{
			{B_name: [12]byte{'.'}, B_inodo: 0},                                                         /*  Apunta a si mismo  */
			{B_name: [12]byte{'.', '.'}, B_inodo: 0},                                                    /*  Apunta al padre  */
			{B_name: [12]byte{'u', 's', 'e', 'r', 's', '.', 't', 'x', 't'}, B_inodo: sb.S_inodes_count}, /*  Apunta a users.txt  */
			{B_name: [12]byte{'-'}, B_inodo: -1},                                                        /*  Vacio  */
		},
	}

	// Escribir el bloque raiz
	err = Utils.EscribirAArchivo(archivo, int64(sb.S_block_start), bloqueRaiz)
	if err != nil {
		return fmt.Errorf("error al escribir el bloque raiz: %w", err)
	}

	// Actualizar bitmap de bloques (indice 0)
	err = sb.ActualizarBitmapBloque(archivo, 0, true)
	if err != nil {
		return fmt.Errorf("error al actualizar el bitmap de bloques: %w", err)
	}

	// Actualizar el contador de bloques y el puntero al primer bloque libre
	sb.ActualizarSuperblockDespuesAsignacionBloque()

	// ----------- Crear Inodo para /users.txt (inodo 1) -----------
	grupoRaiz := NuevoGrupo("1", "root")
	usuarioRaiz := NuevoUsuario("1", "root", "root", "123")
	textoUsuarios := fmt.Sprintf("%s\n%s\n", grupoRaiz.ToString(), usuarioRaiz.ToString())

	inodoUsuarios := &INodo{
		I_uid:   1,
		I_gid:   1,
		I_size:  int32(len(textoUsuarios)),
		I_atime: float32(time.Now().Unix()),
		I_ctime: float32(time.Now().Unix()),
		I_mtime: float32(time.Now().Unix()),
		I_block: [15]int32{1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}, // Apunta al bloque 1 (users.txt)
		I_type:  [1]byte{'1'},                                                         // Tipo archivo
		I_perm:  [3]byte{'7', '7', '7'},
	}

	// Escribir el inodo de users.txt (inodo 1)
	err = inodoUsuarios.Codificar(archivo, int64(sb.S_inode_start + sb.S_inode_size))
	if err != nil {
		return fmt.Errorf("error al escribir el inodo de users.txt: %w", err)
	}

	// Actualizar bitmap de inodos (indice 1)
	err = sb.ActualizarBitmapInodo(archivo, 1, true)
	if err != nil {
		return fmt.Errorf("error al actualizar bitmap de inodos para users.txt: %w", err)
	}

	// Actualizar el contador de inodos y el puntero al primer inodo libre
	sb.ActualizarSuperblockDespuesAsignacionInodo()

	// ----------- Crear Bloque para users.txt (bloque 1) -----------
	bloqueUsuarios := &FileBlock{}
	copy(bloqueUsuarios.B_cont[:], textoUsuarios)

	// Escribir el bloque de users.txt
	err = Utils.EscribirAArchivo(archivo, int64(sb.S_block_start+int32(binary.Size(bloqueUsuarios))), bloqueUsuarios)
	if err != nil {
		return fmt.Errorf("error al escribir el bloque de users.txt: %w", err)
	}

	// Actualizar el bitmap de bloques (indice 1)
	err = sb.ActualizarBitmapBloque(archivo, 1, true)
	if err != nil {
		return fmt.Errorf("error al actualizar el bitmap de bloques para users.txt: %w", err)
	}

	// Actualizar el contador de bloques y el puntero al primer bloque libre
	sb.ActualizarSuperblockDespuesAsignacionBloque()

	fmt.Println("Archivo users.txt generado correctamente.")
	fmt.Println("SuperBloque despues de la creacion de users.txt:")
	sb.Imprimir()
	fmt.Println("\nBloques:")
	sb.ImprimirBloques(archivo.Name())
	fmt.Println("\nInodos:")
	sb.ImprimirInodos(archivo.Name())
	return nil
}

// Muestra los valores de la estructura SuperBlock
func (sb *SuperBlock) Imprimir() {
	fmt.Printf("%-25s %-10s\n", "Campo", "Valor")
	fmt.Printf("%-25s %-10s\n", "-------------------------", "----------")
	fmt.Printf("%-25s %-10d\n", "Tipo de sistema archivos:", sb.S_filesystem_type)
	fmt.Printf("%-25s %-10d\n", "Total inodos creados:", sb.S_inodes_count)
	fmt.Printf("%-25s %-10d\n", "Total bloques creados:", sb.S_blocks_count)
	fmt.Printf("%-25s %-10d\n", "Bloques libres:", sb.S_free_blocks_count)
	fmt.Printf("%-25s %-10d\n", "Inodos libres:", sb.S_free_inodes_count)
	fmt.Printf("%-25s %-10s\n", "Ultimo montaje:", time.Unix(int64(sb.S_mtime), 0).Format("02/01/2006 15:04"))
	fmt.Printf("%-25s %-10s\n", "Ultimo desmontaje:", time.Unix(int64(sb.S_umtime), 0).Format("02/01/2006 15:04"))
	fmt.Printf("%-25s %-10d\n", "Veces montado:", sb.S_mnt_count)
	fmt.Printf("%-25s %-10x\n", "Numero magico:", sb.S_magic)
	fmt.Printf("%-25s %-10d\n", "Dimension inodo:", sb.S_inode_size)
	fmt.Printf("%-25s %-10d\n", "Dimension bloque:", sb.S_block_size)
	fmt.Printf("%-25s %-10d\n", "Primer inodo libre:", sb.S_first_ino)
	fmt.Printf("%-25s %-10d\n", "Primer bloque libre:", sb.S_first_blo)
	fmt.Printf("%-25s %-10d\n", "Inicio bitmap inodos:", sb.S_bm_inode_start)
	fmt.Printf("%-25s %-10d\n", "Inicio bitmap bloques:", sb.S_bm_block_start)
	fmt.Printf("%-25s %-10d\n", "Inicio tabla inodos:", sb.S_inode_start)
	fmt.Printf("%-25s %-10d\n", "Inicio tabla bloques:", sb.S_block_start)
}

// Muestra los inodos desde el archivo
func (sb *SuperBlock) ImprimirInodos(ruta string) error {
	archivo, err := os.Open(ruta)
	if err != nil {
		return fmt.Errorf("fallo al abrir archivo %s: %w", ruta, err)
	}
	defer archivo.Close()

	fmt.Println("\nInodos\n----------------")
	inodos := make([]INodo, sb.S_inodes_count)

	// Deserializar todos los inodos en memoria
	for i := int32(0); i < sb.S_inodes_count; i++ {
		inodo := &inodos[i]
		err := Utils.LeerDeArchivo(archivo, int64(sb.S_inode_start+(i*int32(binary.Size(INodo{})))), inodo)
		if err != nil {
			return fmt.Errorf("fallo al decodificar inodo %d: %w", i, err)
		}
	}

	// Mostrar los inodos
	for i, inodo := range inodos {
		fmt.Printf("\nInodo %d:\n", i)
		inodo.Imprimir()
	}

	return nil
}

// Muestra los bloques desde el archivo
func (sb *SuperBlock) ImprimirBloques(ruta string) error {
	archivo, err := os.Open(ruta)
	if err != nil {
		return fmt.Errorf("fallo al abrir archivo %s: %w", ruta, err)
	}
	defer archivo.Close()

	fmt.Println("\nBloques\n----------------")
	inodos := make([]INodo, sb.S_inodes_count)

	// Deserializar todos los inodos en memoria
	for i := int32(0); i < sb.S_inodes_count; i++ {
		inodo := &inodos[i]
		err := Utils.LeerDeArchivo(archivo, int64(sb.S_inode_start+(i*int32(binary.Size(INodo{})))), inodo)
		if err != nil {
			return fmt.Errorf("fallo al decodificar inodo %d: %w", i, err)
		}
	}

	// Mostrar los bloques
	for _, inodo := range inodos {
		for _, indiceBloques := range inodo.I_block {
			if indiceBloques == -1 {
				break
			}
			if inodo.I_type[0] == '0' {
				bloque := &FolderBlock{}
				err := Utils.LeerDeArchivo(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)), bloque)
				if err != nil {
					return fmt.Errorf("fallo al decodificar bloque carpeta %d: %w", indiceBloques, err)
				}
				fmt.Printf("\nBloque %d:\n", indiceBloques)
				bloque.Imprimir()
			} else if inodo.I_type[0] == '1' {
				bloque := &FileBlock{}
				err := Utils.LeerDeArchivo(archivo, int64(sb.S_block_start+(indiceBloques*sb.S_block_size)), bloque)
				if err != nil {
					return fmt.Errorf("fallo al decodificar bloque archivo %d: %w", indiceBloques, err)
				}
				fmt.Printf("\nBloque %d:\n", indiceBloques)
				bloque.Imprimir()
			}
		}
	}

	return nil
}

// Localiza el siguiente bloque libre y lo marca como ocupado
func (sb *SuperBlock) BuscarSiguienteBloqueLibre(archivo *os.File) (int32, error) {
	totalBloques := sb.S_blocks_count + sb.S_free_blocks_count // Numero total de bloques

	for posicion := int32(0); posicion < totalBloques; posicion++ {
		estaLibre, err := sb.verificarBloqueLibre(archivo, sb.S_bm_block_start, posicion)
		if err != nil {
			return -1, fmt.Errorf("error buscando bloque libre: %w", err)
		}

		if estaLibre {
			// Marcar el bloque como ocupado
			err = sb.ActualizarBitmapBloque(archivo, posicion, true)
			if err != nil {
				return -1, fmt.Errorf("error actualizando el bitmap del bloque: %w", err)
			}

			// Devolver el indice del bloque libre encontrado
			fmt.Println("Indice encontrado:", posicion)
			return posicion, nil
		}
	}

	// Si no hay bloques disponibles
	return -1, fmt.Errorf("no hay bloques disponibles")
}

// Localiza el siguiente inodo libre en el bitmap y lo marca como ocupado
func (sb *SuperBlock) BuscarSiguienteInodoLibre(archivo *os.File) (int32, error) {
	totalInodos := sb.S_inodes_count + sb.S_free_inodes_count // Numero total de inodos

	// Recorre todos los inodos en el bitmap
	for posicion := int32(0); posicion < totalInodos; posicion++ {
		// Verifica si el inodo esta libre
		estaLibre, err := sb.verificarInodoLibre(archivo, sb.S_bm_inode_start, posicion)
		if err != nil {
			return -1, fmt.Errorf("error buscando inodo libre en la posicion %d: %w", posicion, err)
		}

		// Si encontramos un inodo libre
		if estaLibre {
			// Marcar el inodo como ocupado
			err = sb.ActualizarBitmapInodo(archivo, posicion, true)
			if err != nil {
				return -1, fmt.Errorf("error actualizando el bitmap del inodo en la posicion %d: %w", posicion, err)
			}
			// Devolver la posicion del inodo libre encontrado
			fmt.Printf("Inodo libre encontrado y asignado: %d\n", posicion)
			return posicion, nil
		}
	}

	// Si no hay inodos disponibles
	return -1, fmt.Errorf("no hay inodos disponibles")
}

// Acá | AssignNewBlock
// Asigna un nuevo bloque al inodo en el indice especificado si es necesario
func (sb *SuperBlock) AsignarNuevoBloque(archivo *os.File, inodo *INodo, indice int) (int32, error) {
	fmt.Println("=== Iniciando la asignacion de un nuevo bloque ===")

	// Validar que el indice este dentro del rango de bloques validos
	if indice < 0 || indice >= len(inodo.I_block) {
		return -1, fmt.Errorf("indice de bloque fuera de rango: %d", indice)
	}

	// Verificar si ya hay un bloque asignado en ese indice
	if inodo.I_block[indice] != -1 {
		return -1, fmt.Errorf("bloque en el indice %d ya esta asignado: %d", indice, inodo.I_block[indice])
	}

	// Intentar encontrar un bloque libre
	nuevoBloque, err := sb.BuscarSiguienteBloqueLibre(archivo)
	if err != nil {
		return -1, fmt.Errorf("error buscando nuevo bloque libre: %w", err)
	}

	// Verificar si se encontro un bloque libre
	if nuevoBloque == -1 {
		return -1, fmt.Errorf("no hay bloques libres disponibles")
	}

	// Asignar el nuevo bloque en el indice especificado
	inodo.I_block[indice] = nuevoBloque
	fmt.Printf("Nuevo bloque asignado: %d en I_block[%d]\n", nuevoBloque, indice)

	// Actualizar el SuperBlock despues de asignar el bloque
	sb.ActualizarSuperblockDespuesAsignacionBloque()

	// Retornar el nuevo bloque asignado
	return nuevoBloque, nil
}

// Asigna un nuevo inodo pero no lo inicializa
func (sb *SuperBlock) AsignarNuevoInodo(archivo *os.File, inodo *INodo, indice int) (int32, error) {
	fmt.Println("=== Iniciando la asignacion de un nuevo inodo ===")

	// Validar que el indice este dentro del rango de inodos validos
	if indice < 0 || indice >= len(inodo.I_block) {
		return -1, fmt.Errorf("indice de inodo fuera de rango: %d", indice)
	}

	// Verificar si ya hay un inodo asignado en ese indice
	if inodo.I_block[indice] != -1 {
		return -1, fmt.Errorf("el inodo en el indice %d ya esta asignado: %d", indice, inodo.I_block[indice])
	}

	// Encontrar un inodo libre
	nuevoIndiceInodo, err := sb.BuscarSiguienteInodoLibre(archivo)
	if err != nil {
		return -1, fmt.Errorf("error encontrando inodo libre: %w", err)
	}

	// Verificar si se encontro un inodo libre
	if nuevoIndiceInodo == -1 {
		return -1, fmt.Errorf("no hay inodos libres disponibles")
	}

	// Asignar el nuevo inodo en el indice especificado
	inodo.I_block[indice] = nuevoIndiceInodo
	fmt.Printf("Nuevo inodo asignado: %d en I_block[%d]\n", nuevoIndiceInodo, indice)

	// Acá
	// Actualizar el bitmap de inodos
	err = sb.ActualizarBitmapInodo(archivo, nuevoIndiceInodo, true)
	if err != nil {
		return -1, fmt.Errorf("error actualizando el bitmap de inodos: %w", err)
	}

	// Actualizar el estado del superblock
	sb.ActualizarSuperblockDespuesAsignacionInodo()

	// Retornar el indice del inodo asignado
	return nuevoIndiceInodo, nil
}

// Escribe un inodo en la posicion especificada del archivo
func EscribirInodoAArchivo(archivo *os.File, desplazamiento int64, inodo *INodo) error {
	// Mover el puntero al desplazamiento calculado
	_, err := archivo.Seek(desplazamiento, 0)
	if err != nil {
		return fmt.Errorf("error buscando la posicion para escribir el inodo: %w", err)
	}

	// Escribir el inodo en el archivo
	err = binary.Write(archivo, binary.LittleEndian, inodo)
	if err != nil {
		return fmt.Errorf("error escribiendo el inodo en el archivo: %w", err)
	}

	return nil
}

func (sb *SuperBlock) CalcularDesplazamientoInodo(indiceInodo int32) int64 {
	// Calcula el desplazamiento en el archivo basado en el indice del inodo
	return int64(sb.S_inode_start) + int64(indiceInodo)*int64(sb.S_inode_size)
}

// Actualiza el SuperBlock despues de asignar un bloque
func (sb *SuperBlock) ActualizarSuperblockDespuesAsignacionBloque() {
	// Incrementa el contador de bloques asignados
	sb.S_blocks_count++

	// Decrementa el contador de bloques libres
	sb.S_free_blocks_count--

	// Actualiza el puntero al primer bloque libre
	sb.S_first_blo += sb.S_block_size
}

// Actualiza el SuperBlock despues de asignar un inodo
func (sb *SuperBlock) ActualizarSuperblockDespuesAsignacionInodo() {
	// Incrementa el contador de inodos asignados
	sb.S_inodes_count++

	// Decrementa el contador de inodos libres
	sb.S_free_inodes_count--

	// Actualiza el puntero al primer inodo libre
	sb.S_first_ino += sb.S_inode_size
}

// CrearArchivoUsuariosExt3 inicializa el sistema de archivos EXT3 con journaling
func (sb *SuperBlock) CrearArchivoUsuariosExt3(archivo *os.File, inicioJournaling int64) error {
    // 1. Inicializar el área de journaling para la partición si es necesario
    fmt.Println("Inicializando área de journaling para EXT3...")
    err := InicializarAreaJournal(archivo, inicioJournaling, ENTRADAS_JOURNAL)
    if err != nil {
        return fmt.Errorf("error al inicializar el área de journaling: %w", err)
    }

    // 2. Obtener el siguiente índice de journal disponible
    siguienteIndiceJournal, err := ObtenerSiguienteIndiceJournalVacio(archivo, inicioJournaling, ENTRADAS_JOURNAL)
    if err != nil {
        return fmt.Errorf("error obteniendo el siguiente índice de journal: %w", err)
    }
    fmt.Printf("Siguiente índice de journal disponible: %d\n", siguienteIndiceJournal)

    // 3. Crear entrada de journal para el directorio raíz
    err = AgregarEntradaJournal(
        archivo,
        inicioJournaling,
        ENTRADAS_JOURNAL,
        "mkdir",
        "/",
        "",
        sb,
    )
    if err != nil {
        return fmt.Errorf("error al guardar la entrada de la raíz en el journal: %w", err)
    }

    // 4. Crear el inodo y bloque para la raíz
    indiceBloqueRaiz, err := sb.BuscarSiguienteBloqueLibre(archivo)
    if err != nil {
        return fmt.Errorf("error al encontrar el primer bloque libre para la raíz: %w", err)
    }

    bloquesRaiz := [15]int32{indiceBloqueRaiz, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}

    inodoRaiz := &Inodo{}
    err = inodoRaiz.CrearInodo(
        archivo,
        sb,
        '0',
        0,
        bloquesRaiz,
        [3]byte{'7', '7', '7'},
    )
    if err != nil {
        return fmt.Errorf("error al crear el inodo raíz: %w", err)
    }

    bloqueRaiz := &BloqueCarpeta{
        B_content: [4]ContenidoCarpeta{
            {B_name: [12]byte{'.'}, B_inodo: 0},
            {B_name: [12]byte{'.', '.'}, B_inodo: 0},
            {B_name: [12]byte{'u', 's', 'e', 'r', 's', '.', 't', 'x', 't'}, B_inodo: sb.S_inodes_count},
            {B_name: [12]byte{'-'}, B_inodo: -1},
        },
    }

    err = sb.ActualizarBitmapBloque(archivo, indiceBloqueRaiz, true)
    if err != nil {
        return fmt.Errorf("error actualizando el bitmap de bloques: %w", err)
    }

    err = bloqueRaiz.Codificar(archivo, int64(sb.S_first_blo))
    if err != nil {
        return fmt.Errorf("error serializando el bloque raíz: %w", err)
    }

    sb.ActualizarSuperblockDespuesAsignacionBloque()

    // 5. Crear el contenido del archivo de usuarios
    grupoRaiz := NuevoGrupo("1", "root")
    usuarioRaiz := NuevoUsuario("1", "root", "root", "123")
    textoUsuarios := fmt.Sprintf("%s\n%s\n", grupoRaiz.ToString(), usuarioRaiz.ToString())

    // 6. Crear una segunda entrada en el journal para el archivo users.txt
    err = AgregarEntradaJournal(
        archivo,
        inicioJournaling,
        ENTRADAS_JOURNAL,
        "mkfile",
        "/users.txt",
        textoUsuarios,
        sb,
    )
    if err != nil {
        return fmt.Errorf("error al guardar la entrada del archivo /users.txt en el journal: %w", err)
    }

    // 7. Resto del código para crear users.txt
    indiceBloqueUsuarios, err := sb.BuscarSiguienteBloqueLibre(archivo)
    if err != nil {
        return fmt.Errorf("error al encontrar el primer bloque libre para /users.txt: %w", err)
    }
    bloquesArchivo := [15]int32{indiceBloqueUsuarios, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1}

    inodoUsuarios := &Inodo{}
    err = inodoUsuarios.CrearInodo(
        archivo,
        sb,
        '1',
        int32(len(textoUsuarios)),
        bloquesArchivo,
        [3]byte{'7', '7', '7'},
    )
    if err != nil {
        return fmt.Errorf("error al crear el inodo de /users.txt: %w", err)
    }

    bloqueUsuarios := &BloqueArchivo{
        B_content: [64]byte{},
    }
    bloqueUsuarios.AgregarContenido(textoUsuarios)
    err = bloqueUsuarios.Codificar(archivo, int64(sb.S_first_blo))
    if err != nil {
        return fmt.Errorf("error serializando el bloque de /users.txt: %w", err)
    }
    err = sb.ActualizarBitmapBloque(archivo, indiceBloqueUsuarios, true)
    if err != nil {
        return fmt.Errorf("error actualizando el bitmap de bloques para /users.txt: %w", err)
    }

    sb.ActualizarSuperblockDespuesAsignacionBloque()

    // 8. Mostrar estado del sistema de archivos
    fmt.Println("Bloques")
    sb.ImprimirBloques(archivo.Name())

    // 9. Mostrar las entradas de journal usando los nuevos métodos
    fmt.Println("Entradas del Journal:")
    entradas, err := EncontrarEntradasJournalValidas(archivo, inicioJournaling, ENTRADAS_JOURNAL)
    if err != nil {
        fmt.Printf("Error leyendo entradas de journal: %v\n", err)
    } else {
        for i, entrada := range entradas {
            fmt.Printf("-- Entrada %d --\n", i)
            entrada.Imprimir()
        }
    }

    fmt.Println("Sistema de archivos EXT3 inicializado correctamente con journaling")
    return nil
}

// ActualizarSuperblockDespuesDesasignacionBloque actualiza el SuperBlock después de liberar un bloque
func (sb *SuperBlock) ActualizarSuperblockDespuesDesasignacionBloque() {
    // Decrementar el contador de bloques asignados
    sb.S_blocks_count--

    // Incrementar el contador de bloques libres
    sb.S_free_blocks_count++

    // Retroceder el puntero al primer bloque libre
    sb.S_first_blo -= sb.S_block_size
}

// ActualizarSuperblockDespuesDesasignacionInodo actualiza el SuperBlock después de liberar un inodo
func (sb *SuperBlock) ActualizarSuperblockDespuesDesasignacionInodo() {
    // Decrementar el contador de inodos asignados
    sb.S_inodes_count--

    // Incrementar el contador de inodos libres
    sb.S_free_inodes_count++

    // Retroceder el puntero al primer inodo libre
    sb.S_first_ino -= sb.S_inode_size
}