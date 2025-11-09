package handlers

import "github.com/gofiber/fiber/v2"

// Home retourne le message d'accueil
func Home(c *fiber.Ctx) error {
	return c.SendString("🚀 Dorevia Vault API is running!")
}

