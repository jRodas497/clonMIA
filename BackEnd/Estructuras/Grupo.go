package Estructuras

import "fmt"

// Define la estructura para los grupos del sistema
type Grupo struct {
	GID   string //  Identificador unico del grupo, si es 0 esta eliminado
	Tipo  string //  Tipo de entidad, en este caso "G" para grupos
	Grupo string //  Nombre del grupo
}

//  Crea un nuevo grupo
func NuevoGrupo(gid, grupo string) *Grupo {
	return &Grupo{gid, "G", grupo}
}

//  Devuelve una representacion en cadena del grupo
func (g *Grupo) ToString() string {
	return fmt.Sprintf("%s,%s,%s", g.GID, g.Tipo, g.Grupo)
}

//  Borra el grupo (cambia el GID a "0")
func (g *Grupo) Eliminar() {
	g.GID = "0"
}
