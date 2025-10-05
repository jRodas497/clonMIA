package Estructuras

import (
	"fmt"
	"os"
	"time"

	Utils "backend/Utils"
)

type INodo struct {

	/* 88 bytes */

	I_uid   int32     /* UID del usuario propietario del archivo */
	I_gid   int32     /* GID del grupo propietario del archivo */
	I_size  int32     /* Tamaño del archivo en bytes */
	I_atime float32   /* Último acceso al archivo */
	I_ctime float32   /* Último cambio de permisos */
	I_mtime float32   /* Última modificación del archivo */
	I_type  [1]byte   /* Indica si es archivo o carpeta 1=archivo, 0=carpeta */
	I_perm  [3]byte   /* Guarda los permisos del archivo */
	I_block [15]int32 /* 12 bloques directos, 1 indirecto simple, 1 indirecto doble, 1 indirecto triple */
}

func (inodo *INodo) Codificar(archivo *os.File, desplazamiento int64) error {
	err := Utils.EscribirAArchivo(archivo, desplazamiento, inodo)
	if err != nil {
		return fmt.Errorf("error escribiendo INodo al archivo: %w", err)
	}
	return nil
}

func (inodo *INodo) Decodificar(archivo *os.File, desplazamiento int64) error {
	err := Utils.LeerDeArchivo(archivo, desplazamiento, inodo)
	if err != nil {
		return fmt.Errorf("error leyendo INodo desde archivo: %w", err)
	}
	return nil
}

func (inodo *INodo) ActualizarTiempoAcceso() {
	inodo.I_atime = float32(time.Now().Unix())
}

func (inodo *INodo) ActualizarTiempoModificacion() {
	inodo.I_mtime = float32(time.Now().Unix())
}

func (inodo *INodo) ActualizarTiempoPermisos() {
	inodo.I_ctime = float32(time.Now().Unix())
}

// Imprimir atributos del inodo
func (inodo *INodo) Imprimir() {
	tiempoAcceso := time.Unix(int64(inodo.I_atime), 0)
	tiempoPermisos := time.Unix(int64(inodo.I_ctime), 0)
	tiempoModificacion := time.Unix(int64(inodo.I_mtime), 0)

	fmt.Printf("UID propietario: %d\n", inodo.I_uid)
	fmt.Printf("GID grupo: %d\n", inodo.I_gid)
	fmt.Printf("Dimension archivo: %d bytes\n", inodo.I_size)
	fmt.Printf("Ultimo acceso: %s\n", tiempoAcceso.Format(time.RFC3339))
	fmt.Printf("Ultimo cambio de permisos: %s\n", tiempoPermisos.Format(time.RFC3339))
	fmt.Printf("Ultima modificacion: %s\n", tiempoModificacion.Format(time.RFC3339))
	fmt.Printf("Bloques asignados: %v\n", inodo.I_block)
	fmt.Printf("Tipo de elemento: %s\n", string(inodo.I_type[:]))
	fmt.Printf("Permisos: %s\n", string(inodo.I_perm[:]))
}
