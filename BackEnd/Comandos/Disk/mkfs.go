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
	fs   string // Tipo de sistema de archivos
}

func ParserMkfs(tokens []string) (string, error) {
	// Buffer para capturar los mensajes importantes para el usuario
	var bufferSalida bytes.Buffer
	cmd := &MKFS{}

	argumentos := strings.Join(tokens, " ")
	re := regexp.MustCompile(`-id=[^\s]+|-type=[^\s]+|-fs=[^\s]+`)
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
		case "-fs":
			if valor != "2fs" && valor != "3fs" {
				return "", errors.New("el sistema de archivos debe ser 2fs o 3fs")
			}
			cmd.fs = valor
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

	if cmd.fs == "" {
		cmd.fs = "2fs"
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
	n := calcularN(particionMontada, mkfs.fs)
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

	if mkfs.fs == "3fs" {
		err = superBloque.CrearArchivoUsuariosExt3(archivo, int64(particionMontada.Part_start+int32(binary.Size(Estructuras.SuperBlock{}))))
	} else {
		err = superBloque.CrearArchivoUsuarios(archivo)
	}
	
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

func calcularN(particion *Estructuras.Particion, fs string) int32 {
	numerador := int(particion.Part_size) - binary.Size(Estructuras.SuperBlock{})
	baseDenominador := 4 + binary.Size(Estructuras.INodo{}) + 3*binary.Size(Estructuras.FileBlock{}) 
	temp := 0
	if fs == "3fs" {
		temp = binary.Size(Estructuras.Journal{})
	}
	denominador := baseDenominador + temp
	n := math.Floor(float64(numerador) / float64(denominador+temp))

	return int32(n)
}

// Calcular punteros de las estructuras
func crearSuperBlock(particion *Estructuras.Particion, n int32, fs string) *Estructuras.SuperBlock {
	InicioJournal, InicioBMInodo, InicioBMBloque, InicioInodo, InicioBloque := calcularInicioPosiciones(particion, fs, n)

	fmt.Println("\nInicio del SuperBlock:", particion.Part_start)
	fmt.Println("\nFin del SuperBlock:", particion.Part_start+int32(binary.Size(Estructuras.SuperBlock{})))
	fmt.Println("\nInicio del Journal:", InicioJournal)
	fmt.Println("\nFin del Journal:", InicioJournal+int32(binary.Size(Estructuras.Journal{})))
	fmt.Println("\nInicio del Bitmap de Inodos:", InicioBMInodo)
	fmt.Println("\nFin del Bitmap de Inodos:", InicioBMInodo+n)
	fmt.Println("\nInicio del Bitmap de Bloques:", InicioBMBloque)
	fmt.Println("\nFin del Bitmap de Bloques:", InicioBMBloque+(3*n))
	fmt.Println("\nInicio de Inodos:", InicioInodo)

	var fsType int32
	if fs == "3fs" {
		fsType = 3
	} else {
		fsType = 2
	}

	// Nuevo superbloque
	superBloque := &Estructuras.SuperBlock{
		S_filesystem_type:   fsType,
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
		S_first_ino:         InicioInodo,
		S_first_blo:         InicioBloque,
		S_bm_inode_start:    InicioBMInodo,
		S_bm_block_start:    InicioBMBloque,
		S_inode_start:       InicioInodo,
		S_block_start:       InicioBloque,
	}
	return superBloque
}

func calcularInicioPosiciones(particion *Estructuras.Particion, fs string, n int32) (int32, int32, int32, int32, int32) {
	tamSuperBlock := int32(binary.Size(Estructuras.SuperBlock{}))
	tamJournal := int32(binary.Size(Estructuras.Journal{}))
	tamInodo := int32(binary.Size(Estructuras.INodo{}))

	InicioJournal := int32(0)
	InicioBMInodo := particion.Part_start + tamSuperBlock
	InicioBMBloque := InicioBMInodo + n
	InicioInodo := InicioBMBloque + (3 * n)
	InicioBloque := InicioInodo + (tamInodo * n)

	if fs == "3fs" {
		InicioJournal = particion.Part_start + tamSuperBlock
		InicioBMInodo += InicioJournal + tamJournal * tamJournal
		InicioBMBloque += InicioBMInodo + n
		InicioInodo += InicioBMBloque + (3 * n)
		InicioBloque += InicioInodo + (tamInodo * n)
	}
	return InicioJournal, InicioBMInodo, InicioBMBloque, InicioInodo, InicioBloque
}