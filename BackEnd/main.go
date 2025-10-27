package main

import (
	"fmt"
	"log"
	"strings"

	Analizador "backend/Analizador"
	usercmds "backend/Comandos/User"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	partitions "backend/partitions"
)

func main() {
	// Crear una nueva instancia de Fiber
	app := fiber.New()

	// Configurar el middleware CORS
	app.Use(cors.New())

	// Definir la ruta POST para recibir el comando del usuario
	app.Post("/mia", func(c *fiber.Ctx) error {

		// ESTRUCTURA DE LA SOLICITUD DEL JSON A TRAVES DE API
		type Solicitud struct {
			Comando string `json:"comando"`
		}

		var sol Solicitud

		// Parsear la solicitud mediante el body como JSON
		if err := c.BodyParser(&sol); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "JSON invalido",
			})
		}

		entrada := sol.Comando
		fmt.Println("Entrada: ", entrada)

		// Separar el comando en lineas
		lineas := strings.Split(entrada, "\n")

		// Lista para acumular las salidas
		var resultados []string

		// Analizar cada linea
		for _, linea := range lineas {
			if strings.TrimSpace(linea) == "" {
				continue
			}

			resultado, err := Analizador.Analizador(linea)
			if err != nil {
				resultado = fmt.Sprintf("Error: %s", err.Error())
			}

			resultados = append(resultados, resultado)
		}

		return c.JSON(fiber.Map{
			"resultados": resultados,
		})
	})

	// Ruta para login desde el frontend
	app.Post("/users/login", func(c *fiber.Ctx) error {
		type LoginRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
			ID       string `json:"id"`
		}

		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": "JSON invalido"})
		}

		// Construir comando tal como el analizador espera
		loginCmd := fmt.Sprintf("login -user=%s -pass=%s -id=%s", req.Username, req.Password, req.ID)
		res, err := usercmds.ParserLogin(strings.Split(loginCmd, " "))
		if err != nil {
			// devolver mensaje amigable
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"status": "error", "message": err.Error()})
		}

		// Respuesta de éxito: mandamos el texto que genera el parser
		return c.JSON(fiber.Map{"status": "success", "message": res})
	})

	// Ruta para logout desde el frontend
	app.Post("/users/logout", func(c *fiber.Ctx) error {
		// Ejecutar el comando logout del analizador/usuario
		res, err := usercmds.ParserLogout([]string{"logout"})
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"status": "error", "message": err.Error()})
		}
		// Además invocar cierre global de sesión si aplica
		// Global.CerrarSesion() // ParserLogout already resets Global.UsuarioActual
		return c.JSON(fiber.Map{"status": "success", "message": res})
	})

	// Server en puerto 3000
	// register partition routes
	partitions.RegisterRoutes(app)

	log.Fatal(app.Listen(":3000"))
}
