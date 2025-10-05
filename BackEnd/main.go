package main

import (
	"fmt"
	"log"
	"strings"

	Analizador "backend/Analizador"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
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

	// Server en puerto 3000
	log.Fatal(app.Listen(":3000"))
}
