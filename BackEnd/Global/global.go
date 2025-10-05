package Global

import (
	Estructuras "backend/Estructuras"
	"errors"
	"os"
	"strings"
)

const Carnet string = "89" // 202200389
var (
	UsuarioActual       *Estructuras.Usuario = nil
	ParticionesMontadas map[string]string    = make(map[string]string)
)

// GetMountedPartitionSuperblock obtiene el SuperBlock de la partición montada con el id especificado
func GetMountedPartitionSuperblock(id string) (*Estructuras.SuperBlock, *Estructuras.Particion, string, error) {
	path := ParticionesMontadas[id]
	if path == "" {
		return nil, nil, "", errors.New("la partición no está montada")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	defer file.Close()

	var mbr Estructuras.MBR
	err = mbr.Decodificar(file)
	if err != nil {
		return nil, nil, "", err
	}
	
	partition, err := mbr.ObtenerParticionPorID(id)
	if partition == nil {
		return nil, nil, "", err
	}
	
	var sb Estructuras.SuperBlock
	err = sb.Decodificar(file, int64(partition.Part_start))
	if err != nil {
		return nil, nil, "", err
	}

	return &sb, partition, path, nil
}

// GetMountedPartition obtiene la partición montada con el id especificado
func GetMountedPartition(id string) (*Estructuras.Particion, string, error) {
	path := ParticionesMontadas[id]
	if path == "" {
		return nil, "", errors.New("la partición no está montada")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	var mbr Estructuras.MBR
	err = mbr.Decodificar(file)
	if err != nil {
		return nil, "", err
	}
	
	partition, err := mbr.ObtenerParticionPorID(id)
	if partition == nil {
		return nil, "", err
	}

	return partition, path, nil
}

func GetMountedPartitionByName(name string) (*Estructuras.Particion, string, error) {
	for _, path := range ParticionesMontadas {
		file, err := os.Open(path)
		if err != nil {
			continue
		}

		var mbr Estructuras.MBR
		err = mbr.Decodificar(file)
		file.Close()
		if err != nil {
			continue
		}

		partition, _ := mbr.ObtenerParticionPorNombre(name)
		if partition != nil {
			partitionName := strings.Trim(string(partition.Part_name[:]), "\x00 ")
			if strings.EqualFold(partitionName, name) {
				return partition, path, nil
			}
		}
	}
	return nil, "", errors.New("la partición con nombre '" + name + "' no está montada")
}

// GetMountedPartitionRep obtiene el MBR y el SuperBlock de la partición montada con el id especificado
func GetMountedPartitionRep(id string) (*Estructuras.MBR, *Estructuras.SuperBlock, string, error) {
	path := ParticionesMontadas[id]
	if path == "" {
		return nil, nil, "", errors.New("la partición no está montada")
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	defer file.Close()

	var mbr Estructuras.MBR
	err = mbr.Decodificar(file)
	if err != nil {
		return nil, nil, "", err
	}
	
	partition, err := mbr.ObtenerParticionPorID(id)
	if err != nil {
		return nil, nil, "", err
	}
	
	var sb Estructuras.SuperBlock
	err = sb.Decodificar(file, int64(partition.Part_start))
	if err != nil {
		return nil, nil, "", err
	}

	return &mbr, &sb, path, nil
}

// IsLoggedIn verifica si hay un usuario logueado actualmente
func IsLoggedIn() bool {
	return UsuarioActual != nil && UsuarioActual.Estado
}

func Logout() {
	if UsuarioActual != nil {
		UsuarioActual.Estado = false
		UsuarioActual = nil
	}
}

func ValidateAccess(partitionId string) error {
	if !IsLoggedIn() {
		return errors.New("no hay un usuario logueado")
	}
	_, _, err := GetMountedPartition(partitionId)
	if err != nil {
		return errors.New("la partición no está montada")
	}
	return nil
}

// Alias para compatibilidad con código existente
func ObtenerParticionMontada(id string) (*Estructuras.Particion, string, error) {
	return GetMountedPartition(id)
}

func ObtenerParticionMontadaReporte(id string) (*Estructuras.MBR, *Estructuras.SuperBlock, string, error) {
	return GetMountedPartitionRep(id)
}

func ObtenerSuperblockParticionMontada(id string) (*Estructuras.SuperBlock, *Estructuras.Particion, string, error) {
	return GetMountedPartitionSuperblock(id)
}

func VerificarSesionActiva() bool {
	return IsLoggedIn()
}

func CerrarSesion() {
	Logout()
}
