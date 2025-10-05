package Estructuras

import "fmt"

// Usuario define la estructura para los usuarios del sistema
type Usuario struct {
	Id     string //  Identificador único del usuario, si es 0 está eliminado
	Tipo   string //  Tipo de entidad, en este caso "U" para usuarios
	Grupo  string //  Grupo al que pertenece el usuario
	Nombre string //  Nombre del usuario
	Pass   string //  Contraseña del usuario
	Estado bool   //  Indica si el usuario está activo o eliminado
}

func NuevoUsuario(id, rol, nombre, pass string) *Usuario {
	return &Usuario{id, "U", rol, nombre, pass, true} // El usuario se crea como activo
}

func (u *Usuario) ToString() string {
	return fmt.Sprintf("%s,%s,%s,%s,%s", u.Id, u.Tipo, u.Grupo, u.Nombre, u.Pass)
}

//  Elimina y cambia el ID a "0" | desactiva el estado
func (u *Usuario) Eliminar() {
	u.Id = "0"
	u.Estado = false
}
