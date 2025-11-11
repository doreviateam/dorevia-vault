package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// HealthLive gère l'endpoint GET /health/live (liveness probe)
// Vérifie simplement que le processus répond
func HealthLive(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}

// HealthReady gère l'endpoint GET /health/ready (readiness probe)
// Note: Dans main.go, on utilisera DetailedHealthHandler pour la vraie vérification
// Cette fonction est un wrapper léger si nécessaire
func HealthReady(c *fiber.Ctx) error {
	// Par défaut, on retourne ready
	// Dans main.go, on peut directement utiliser DetailedHealthHandler
	return c.JSON(fiber.Map{
		"status": "ready",
	})
}

