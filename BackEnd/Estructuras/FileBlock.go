package Estructuras

import (
	"fmt"
	"os"
	"strings"

	Utils "backend/Utils"
)

const DimensionBloque = 64

type FileBlock struct {
	B_cont [DimensionBloque]byte
	// Total: 64 bytes
}

// Serializa la estructura FileBlock en un archivo binario en la posicion especificada
func (fb *FileBlock) Codificar(archivo *os.File, desplazamiento int64) error {
	err := Utils.EscribirAArchivo(archivo, desplazamiento, fb.B_cont)
	if err != nil {
		return fmt.Errorf("error escribiendo FileBlock al archivo: %w", err)
	}
	return nil
}

// Deserializa la estructura FileBlock desde un archivo binario en posicion especificada
func (fb *FileBlock) Decodificar(archivo *os.File, desplazamiento int64) error {
	err := Utils.LeerDeArchivo(archivo, desplazamiento, &fb.B_cont)
	if err != nil {
		return fmt.Errorf("error leyendo FileBlock desde archivo: %w", err)
	}
	return nil
}

// Calcula el espacio usado en el bloque en bytes
func (fb *FileBlock) EspacioUsado() int {
	contenido := fb.ObtenerContenido()
	return len(contenido)
}

// Retorna el contenido de B_cont como una cadena, eliminando bytes nulos al final
func (fb *FileBlock) ObtenerContenido() string {
	contenido := string(fb.B_cont[:])

	contenido = strings.TrimRight(contenido, "\x00")
	return contenido
}

// Copia una cadena en B_cont, asegurando que no exceda la dimension maxima
func (fb *FileBlock) EstablecerContenido(contenido string) error {
	if len(contenido) > DimensionBloque {
		return fmt.Errorf("la dimension del contenido excede la dimension del bloque de %d bytes", DimensionBloque)
	}

	fb.LimpiarContenido()

	copy(fb.B_cont[:], contenido)
	return nil
}

// Retorna la cantidad de bytes disponibles en el bloque
func (fb *FileBlock) EspacioDisponible() int {
	return DimensionBloque - fb.EspacioUsado()
}

// Verifica si aun queda espacio en el bloque
func (fb *FileBlock) TieneEspacio() bool {
	return fb.EspacioDisponible() > 0
}

// Muestra el contenido de B_cont como una cadena
func (fb *FileBlock) Imprimir() {
	fmt.Print(fb.ObtenerContenido())
}

// AgregarContenido debe asegurar que el resto del bloque esté limpio
func (fb *FileBlock) AgregarContenido(contenido string) error {
    espacioDisponible := fb.EspacioDisponible()
    if len(contenido) > espacioDisponible {
        return fmt.Errorf("no hay suficiente espacio en el bloque para añadir %d bytes", len(contenido))
    }

    // Encontrar donde termina el contenido actual
    espacioUsado := fb.EspacioUsado()

    // Copiar el nuevo contenido
    copy(fb.B_cont[espacioUsado:], contenido)

    // IMPORTANTE: Limpiar el resto del bloque con bytes nulos
    for i := espacioUsado + len(contenido); i < DimensionBloque; i++ {
        fb.B_cont[i] = 0
    }

    return nil
}

// Limpia el contenido de B_cont
func (fb *FileBlock) LimpiarContenido() {
	for i := range fb.B_cont {
		fb.B_cont[i] = 0
	}
}

// Crea un nuevo FileBlock con contenido opcional
func NuevoFileBlock(contenido string) (*FileBlock, error) {
	fb := &FileBlock{}
	err := fb.EstablecerContenido(contenido)
	if err != nil {
		return nil, err
	}
	return fb, nil
}

// Divide una cadena en bloques de dimension y retorna un slice de FileBlocks
func DividirContenido(contenido string) ([]*FileBlock, error) {
	var bloques []*FileBlock
	for len(contenido) > 0 {
		final := DimensionBloque
		if len(contenido) < DimensionBloque {
			final = len(contenido)
		}
		ba, err := NuevoFileBlock(contenido[:final])
		if err != nil {
			return nil, err
		}
		bloques = append(bloques, ba)
		contenido = contenido[final:]
	}
	return bloques, nil
}
