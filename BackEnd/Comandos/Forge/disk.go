
package Forge

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// DiskCommand maneja el comando que muestra información del disco
type DiskCommand struct {
	DiskManager *DiskManager
}

// NewDiskCommand crea un nuevo comando de disco
func NewDiskCommand() *DiskCommand {
	return &DiskCommand{
		DiskManager: NewDiskManager(), // Inicializa DiskManager
	}
}

// ShowDisk muestra la información del disco en base a la ruta
func (dc *DiskCommand) ShowDisk(diskPath string) (string, error) {
	// Buffer para capturar la salida
	var outputBuffer bytes.Buffer

	// Cargar el disco usando el DiskManager
	err := dc.DiskManager.LoadDisk(diskPath)
	if err != nil {
		return "", fmt.Errorf("error al cargar el disco: %v", err)
	}

	// Obtener el MBR del disco
	mbr, exists := dc.DiskManager.PartitionMBRs[diskPath]
	if !exists {
		return "", fmt.Errorf("error: no se pudo encontrar el MBR para el disco en la ruta '%s'", diskPath)
	}

	// Convertir la información de las particiones en JSON para mostrar
	partitions := mbr.ListPartitions()
	partitionsJSON, err := json.MarshalIndent(partitions, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error serializando las particiones a JSON: %v", err)
	}

	// Escribir la información en el buffer
	outputBuffer.WriteString(fmt.Sprintf("Disco: %s\n", diskPath))
	outputBuffer.WriteString("Particiones:\n")
	outputBuffer.WriteString(string(partitionsJSON))
	outputBuffer.WriteString("\n")

	return outputBuffer.String(), nil
}
