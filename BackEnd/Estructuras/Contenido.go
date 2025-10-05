package Estructuras

type Contenido struct {
	b_name      [12]byte //  Nombre archivo o carpeta
	b_inex_nodo int64    //  Apuntador de inodo del archivo o carpeta
}

// Constructor
func NewContenido() Contenido {
	var cont Contenido
	cont.b_inex_nodo = -1
	return cont
}
