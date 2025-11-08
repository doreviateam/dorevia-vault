package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/version", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"version": "0.0.1",
		})
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("🚀 Dorevia Vault API is running!")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s", port)
	log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
}
