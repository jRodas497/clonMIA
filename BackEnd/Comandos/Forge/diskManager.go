package Forge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	Estructuras "backend/Estructuras"
	Global "backend/Global"
)

// DiskManager maneja el acceso a discos, MBR y particiones (implementación mínima).
type DiskManager struct {
	disks         map[string]*os.File
	PartitionMBRs map[string]*Estructuras.MBR
}

// NewDiskManager crea un nuevo gestor de discos
func NewDiskManager() *DiskManager {
	return &DiskManager{
		disks:         make(map[string]*os.File),
		PartitionMBRs: make(map[string]*Estructuras.MBR),
	}
}

// LoadDisk abre un archivo de disco binario y carga su MBR
func (dm *DiskManager) LoadDisk(diskPath string) error {
	// Intenta abrir el archivo
	file, err := os.OpenFile(diskPath, os.O_RDWR, 0666)
	if err != nil {
		return fmt.Errorf("error al abrir el disco: %w", err)
	}

	// Leer el MBR del disco
	mbr := &Estructuras.MBR{}
	err = mbr.Decodificar(file)
	if err != nil {
		file.Close()
		return fmt.Errorf("error al leer el MBR del disco: %w", err)
	}

	// Guardar el archivo y el MBR en las estructuras del DiskManager
	dm.disks[diskPath] = file
	dm.PartitionMBRs[diskPath] = mbr
	fmt.Printf("Disco '%s' cargado exitosamente.\n", diskPath)
	return nil
}

// CloseDisk cierra un disco abierto
func (dm *DiskManager) CloseDisk(diskPath string) error {
	if file, exists := dm.disks[diskPath]; exists {
		file.Close()
		delete(dm.disks, diskPath)
		delete(dm.PartitionMBRs, diskPath)
		fmt.Printf("Disco '%s' cerrado exitosamente.\n", diskPath)
		return nil
	}
	return fmt.Errorf("disco no encontrado: %s", diskPath)
}

// MountPartition intenta obtener la partición montada por nombre
func (dm *DiskManager) MountPartition(diskPath string, partitionName string) (*Estructuras.Particion, error) {
	partition, path, err := Global.GetMountedPartitionByName(partitionName)
	if err != nil {
		return nil, fmt.Errorf("la partición '%s' no está montada en el disco '%s': %v", partitionName, diskPath, err)
	}
	if path != diskPath {
		return nil, fmt.Errorf("la partición '%s' no está montada en el disco '%s'", partitionName, diskPath)
	}
	return partition, nil
}

// PrintPartitionTree serializa a JSON el árbol de directorios de una partición
func (dm *DiskManager) PrintPartitionTree(diskPath string, partitionName string, outputBuffer *bytes.Buffer) error {
	tree, err := dm.GetPartitionTree(diskPath, partitionName)
	if err != nil {
		return fmt.Errorf("error obteniendo el árbol de directorios: %v", err)
	}

	treeJSON, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return fmt.Errorf("error al serializar el árbol de directorios a JSON: %v", err)
	}

	outputBuffer.WriteString(string(treeJSON))
	return nil
}

// GetPartitionTree genera el árbol de ficheros de una partición
func (dm *DiskManager) GetPartitionTree(diskPath string, partitionName string) (*DirectoryTree, error) {
	_, exists := dm.disks[diskPath]
	if !exists {
		return nil, fmt.Errorf("disco '%s' no está cargado", diskPath)
	}
	partition, err := dm.MountPartition(diskPath, partitionName)
	if err != nil {
		return nil, err
	}

	treeService, err := NewDirectoryTreeService()
	if err != nil {
		return nil, fmt.Errorf("error inicializando el servicio de árbol de directorios: %v", err)
	}
	defer treeService.Close()

	tree, err := treeService.GetDirectoryTree(fmt.Sprintf("/partition/%s", strings.TrimRight(strings.TrimSpace(string(partition.Part_name[:])), "\x00")))
	if err != nil {
		return nil, fmt.Errorf("error obteniendo el árbol de directorios: %v", err)
	}

	return tree, nil
}
