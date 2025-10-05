package Estructuras

import (
	Utils "backend/Utils"
	"fmt"
	"os"
)

type EBR struct {
	Ebr_mount [1]byte  // Estado de montaje de la partición
	Ebr_fit   [1]byte  // Algoritmo de ajuste: BF, FF, WF
	Ebr_start int32    // Posición inicial en bytes
	Ebr_size  int32    // Dimensión total en bytes
	Ebr_next  int32    // Puntero al próximo EBR (-1 si es el último)
	Ebr_name  [16]byte // Identificador de la partición
}

func (e *EBR) EstablecerEBR(ajuste byte, capacidad int32, inicio int32, siguiente int32, nombre string) {
	fmt.Println("=== Configurando nuevo EBR ===")
	fmt.Printf(" Fit: %c\n Capacidad asignada: %d bytes\n Posición de inicio: %d\n Enlace siguiente: %d\n Identificador: %s\n",
		ajuste, capacidad, inicio, siguiente, nombre)

	e.Ebr_mount[0] = '1' // 1 indica que está montado
	e.Ebr_fit[0] = ajuste
	e.Ebr_start = inicio
	e.Ebr_size = capacidad
	e.Ebr_next = siguiente

	// Transferir nombre al buffer y completar con ceros
	copy(e.Ebr_name[:], nombre)
	for i := len(nombre); i < len(e.Ebr_name); i++ {
		e.Ebr_name[i] = 0
	}
}

// Serialización del EBR hacia archivo en ubicación específica
func (e *EBR) Codificar(archivo *os.File, posicion int64) error {
	return Utils.EscribirAArchivo(archivo, posicion, e)
}

func (e *EBR) CalcularInicioSiguienteEBR(inicioParticionExtendida int32, capacidadParticionExtendida int32) (int32, error) {
	fmt.Printf(">>> Procesando cálculo del siguiente EBR <<<\n")
	fmt.Printf("EBR actual -> Posición: %d | Dimensión: %d | Siguiente: %d\n",
		e.Ebr_start, e.Ebr_size, e.Ebr_next)

	// Validación de dimensión válida
	if e.Ebr_size <= 0 {
		return -1, fmt.Errorf("dimensión del EBR no válida o nula")
	}

	// Verificación de posición dentro de límites
	if e.Ebr_start < inicioParticionExtendida {
		return -1, fmt.Errorf("posición inicial del EBR fuera de rango")
	}

	siguienteInicio := e.Ebr_start + e.Ebr_size

	// Verificar que la nueva posición no exceda los límites
	if siguienteInicio <= e.Ebr_start || siguienteInicio >= inicioParticionExtendida+capacidadParticionExtendida {
		return -1, fmt.Errorf("el próximo EBR excedería los límites de la partición extendida")
	}

	fmt.Printf("Posición calculada exitosamente: %d\n", siguienteInicio)
	return siguienteInicio, nil
}

// Actualiza el puntero hacia el próximo EBR en la cadena enlazada
func (e *EBR) EstablecerSiguienteEBR(nuevoSiguiente int32) {
	fmt.Printf("Actualizando enlace EBR: %d -> %d\n",
		e.Ebr_start, nuevoSiguiente)
	e.Ebr_next = nuevoSiguiente
}

// Visualización completa de los datos del EBR en formato compacto
func (e *EBR) Imprimir() {
	fmt.Printf("[EBR] Estado:%c | Ajuste:%c | Pos:%d | Dim:%d | Sig:%d | ID:%s\n",
		e.Ebr_mount[0], e.Ebr_fit[0], e.Ebr_start, e.Ebr_size, e.Ebr_next, string(e.Ebr_name[:]))
}

func (ebr *EBR) Decodificar(archivo *os.File, posicion int64) error {

	// Obtener metadatos del archivo para validaciones
	infoArchivo, err := archivo.Stat()
	if err != nil {
		return fmt.Errorf("fallo al acceder a metadatos del archivo: %v", err)
	}

	// Validar que la posición sea accesible
	if posicion < 0 || posicion >= infoArchivo.Size() {
		return fmt.Errorf("ubicación %d inaccesible para lectura de EBR", posicion)
	}

	err = Utils.LeerDeArchivo(archivo, posicion, ebr)
	if err != nil {
		return err
	}

	fmt.Printf("EBR recuperado correctamente desde posición %d\n", posicion)
	return nil
}

// Extrae un EBR específico desde una ubicación determinada del archivo
func LeerEBR(inicio int32, archivo *os.File) (*EBR, error) {
	fmt.Printf("Extrayendo EBR desde posición: %d\n", inicio)
	ebr := &EBR{}
	err := ebr.Decodificar(archivo, int64(inicio))
	if err != nil {
		return nil, err
	}
	return ebr, nil
}

func BuscarUltimoEBR(inicio int32, archivo *os.File) (*EBR, error) {
	fmt.Printf("Iniciando búsqueda del último EBR desde: %d\n", inicio)

	ebrActual, err := LeerEBR(inicio, archivo)
	if err != nil {
		return nil, err
	}

	// Recorrer la cadena hasta encontrar el final
	for ebrActual.Ebr_next != -1 {
		if ebrActual.Ebr_next < 0 {
			// Protección contra punteros inválidos
			return ebrActual, nil
		}
		fmt.Printf("Navegando EBR: Pos:%d | Siguiente:%d\n",
			ebrActual.Ebr_start, ebrActual.Ebr_next)

		siguienteEBR, err := LeerEBR(ebrActual.Ebr_next, archivo)
		if err != nil {
			return nil, err
		}
		ebrActual = siguienteEBR
	}

	fmt.Printf("Último EBR localizado en posición: %d\n", ebrActual.Ebr_start)
	return ebrActual, nil
}

func CrearYEscribirEBR(inicio int32, capacidad int32, ajuste byte, nombre string, archivo *os.File) error {
	fmt.Printf("Construyendo y persistiendo EBR en posición: %d\n", inicio)

	ebr := &EBR{}
	ebr.EstablecerEBR(ajuste, capacidad, inicio, -1, nombre)

	return ebr.Codificar(archivo, int64(inicio))
}

// Sobrescribir sobrescribe el espacio de la partición lógica (EBR) con ceros
func (e *EBR) Sobrescribir(archivo *os.File) error {
	// Verificar si el EBR tiene un tamaño válido
	if e.Ebr_size <= 0 {
		return fmt.Errorf("el tamaño del EBR es inválido o cero")
	}

	// Posicionarse en el inicio del EBR (donde comienza la partición lógica)
	_, err := archivo.Seek(int64(e.Ebr_start), 0)
	if err != nil {
		return fmt.Errorf("error al mover el puntero del archivo a la posición del EBR: %v", err)
	}

	// Crear un buffer de ceros del tamaño de la partición lógica
	ceros := make([]byte, e.Ebr_size)

	// Escribir los ceros en el archivo
	_, err = archivo.Write(ceros)
	if err != nil {
		return fmt.Errorf("error al sobrescribir el espacio del EBR: %v", err)
	}

	fmt.Printf("Espacio de la partición lógica (EBR) en posición %d sobrescrito con ceros.\n", e.Ebr_start)
	return nil
}
