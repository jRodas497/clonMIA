package Disk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	Estructuras "backend/Estructuras"
	Global "backend/Global"
)

type MKFS struct {
	id   string // ID disco
	tipo string // Formato
}

func ParserMkfs(tokens []string) (string, error) {
	// Buffer para capturar los mensajes importantes para el usuario
	var bufferSalida bytes.Buffer
	cmd := &MKFS{}

	argumentos := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-id=[^\s]+|-type=[^\s]+`)
	coincidencias := re.FindAllString(argumentos, -1)

	for _, coincidencia := range coincidencias {
		clavValor := strings.SplitN(coincidencia, "=", 2)
		if len(clavValor) != 2 {
			return "", fmt.Errorf("formato de parametro invalido: %s", coincidencia)
		}
		clave, valor := strings.ToLower(clavValor[0]), clavValor[1]
		if strings.HasPrefix(valor, "\"") && strings.HasSuffix(valor, "\"") {
			valor = strings.Trim(valor, "\"")
		}

		switch clave {
		case "-id":
			if valor == "" {
				return "", errors.New("el id no puede estar vacio")
			}
			cmd.id = valor
		case "-type":
			if valor != "full" {
				return "", errors.New("el tipo debe ser full")
			}
			cmd.tipo = valor
		default:
			return "", fmt.Errorf("parametro desconocido: %s", clave)
		}
	}

	if cmd.id == "" {
		return "", errors.New("faltan parametros requeridos: -id")
	}

	if cmd.tipo == "" {
		cmd.tipo = "full"
	}

	err := comandoMkfs(cmd, &bufferSalida)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	return bufferSalida.String(), nil
}

func comandoMkfs(mkfs *MKFS, bufferSalida *bytes.Buffer) error {
	fmt.Fprintf(bufferSalida, "----------------------------  MKFS ----------------------------\n")

	// Obtener la particion montada
	particionMontada, rutaParticion, err := Global.ObtenerParticionMontada(mkfs.id)
	if err != nil {
		return fmt.Errorf("error al obtener la particion montada con ID %s: %v", mkfs.id, err)
	}

	// Abrir el archivo de la particion
	archivo, err := os.OpenFile(rutaParticion, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo el archivo de la particion en %s: %v", rutaParticion, err)
	}
	defer archivo.Close()

	fmt.Fprintf(bufferSalida, "Particion montada correctamente en %s.\n", rutaParticion)
	// Mensaje de depuracion
	fmt.Println("\nParticion montada:")
	particionMontada.Imprimir()

	// Calcular el valor de n
	n := calcularN(particionMontada)
	fmt.Println("\nValor de n:", n)

	// Crear el superblock
	superBloque := crearSuperBlock(particionMontada, n)
	fmt.Println("\nSuperBlock:")
	superBloque.Imprimir()

	// Crear bitmaps
	err = superBloque.CrearBitMaps(archivo)
	if err != nil {
		return fmt.Errorf("error generando bitmaps: %v", err)
	}
	fmt.Fprintln(bufferSalida, "Bitmaps generados correctamente.")

	// Archivo users.txt
	err = superBloque.CrearArchivoUsuarios(archivo)
	if err != nil {
		return fmt.Errorf("error generando el archivo users.txt: %v", err)
	}
	fmt.Fprintln(bufferSalida, "Archivo users.txt generado correctamente.")

	// Serializar el superbloque
	err = superBloque.Codificar(archivo, int64(particionMontada.Part_start))
	if err != nil {
		return fmt.Errorf("error al escribir el superbloque en la particion: %v", err)
	}
	fmt.Fprintln(bufferSalida, "SuperBloque escrito correctamente en el disco.")
	fmt.Fprintln(bufferSalida, "--------------------------------------------")

	return nil
}

func calcularN(particion *Estructuras.Particion) int32 {
	numerador := int(particion.Part_size) - binary.Size(Estructuras.SuperBlock{})
	denominador := 4 + binary.Size(Estructuras.INodo{}) + 3*binary.Size(Estructuras.FileBlock{}) // No importa que bloque poner, ya que todos tienen la misma dimension
	n := math.Floor(float64(numerador) / float64(denominador))

	return int32(n)
}

// Calcular punteros de las estructuras
func crearSuperBlock(particion *Estructuras.Particion, n int32) *Estructuras.SuperBlock {
	// Bitmaps
	bm_inicio_inodo := particion.Part_start + int32(binary.Size(Estructuras.SuperBlock{}))
	bm_inicio_bloque := bm_inicio_inodo + n // n indica la cantidad de inodos, solo la cantidad para ser representada en un bitmap
	// Inodos
	inicio_inodo := bm_inicio_bloque + (3 * n) // 3*n indica la cantidad de bloques, se multiplica por 3 porque se tienen 3 tipos de bloques
	// Bloques
	inicio_bloque := inicio_inodo + (int32(binary.Size(Estructuras.INodo{})) * n) // n indica la cantidad de inodos, solo que aqui indica la cantidad de estructuras INodo

	// Nuevo superbloque
	superBloque := &Estructuras.SuperBlock{
		S_filesystem_type:   2,
		S_inodes_count:      0,
		S_blocks_count:      0,
		S_free_inodes_count: int32(n),
		S_free_blocks_count: int32(n * 3),
		S_mtime:             float64(time.Now().Unix()),
		S_umtime:            float64(time.Now().Unix()),
		S_mnt_count:         1,
		S_magic:             0xEF53,
		S_inode_size:        int32(binary.Size(Estructuras.INodo{})),
		S_block_size:        int32(binary.Size(Estructuras.FileBlock{})),
		S_first_ino:         inicio_inodo,
		S_first_blo:         inicio_bloque,
		S_bm_inode_start:    bm_inicio_inodo,
		S_bm_block_start:    bm_inicio_bloque,
		S_inode_start:       inicio_inodo,
		S_block_start:       inicio_bloque,
	}
	return superBloque
}
