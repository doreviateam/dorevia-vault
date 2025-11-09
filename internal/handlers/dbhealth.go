package handlers

import (
	"context"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// DBHealthHandler vérifie l'état de la connexion à la base de données
func DBHealthHandler(db *storage.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "error",
				"message": "Database not configured",
			})
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := db.Health(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "error",
				"message": "Database connection failed",
				"error": err.Error(),
			})
		}

		return c.JSON(fiber.Map{
			"status": "ok",
			"message": "Database connection healthy",
		})
	}
}

