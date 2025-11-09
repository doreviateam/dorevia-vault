package handlers

import "github.com/gofiber/fiber/v2"

// Health retourne l'état de santé du service
func Health(c *fiber.Ctx) error {
	return c.SendString("ok")
}

