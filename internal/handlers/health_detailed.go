package handlers

import (
	"context"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/health"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// DetailedHealthHandler gère l'endpoint GET /health/detailed
// Retourne un état de santé détaillé de tous les composants du système
func DetailedHealthHandler(
	db *storage.DB,
	storageDir string,
	jwsService *crypto.Service,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Vérifier l'état de santé de tous les composants
		detailedHealth := health.CheckDetailedHealth(ctx, db, storageDir, jwsService)

		// Déterminer le code de statut HTTP basé sur le statut global
		statusCode := fiber.StatusOK
		if detailedHealth.Status == health.StatusFail {
			statusCode = fiber.StatusServiceUnavailable
		} else if detailedHealth.Status == health.StatusWarn {
			statusCode = fiber.StatusOK // 200 OK même avec warnings
		}

		return c.Status(statusCode).JSON(detailedHealth)
	}
}

