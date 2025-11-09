package handlers

import (
	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// JWKSHandler gère l'endpoint GET /jwks.json
func JWKSHandler(jwsService *crypto.Service, log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if jwsService == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "JWS service not configured",
			})
		}

		// Générer le JWKS
		jwks, err := jwsService.CurrentJWKS()
		if err != nil {
			log.Error().Err(err).Msg("Failed to generate JWKS")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate JWKS",
			})
		}

		// Retourner le JWKS avec cache (5 minutes)
		c.Set("Content-Type", "application/json")
		c.Set("Cache-Control", "public, max-age=300") // 5 minutes

		return c.Send(jwks)
	}
}

