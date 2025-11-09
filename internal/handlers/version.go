package handlers

import "github.com/gofiber/fiber/v2"

// Version retourne la version actuelle du service
func Version(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"version": "0.0.1",
	})
}

