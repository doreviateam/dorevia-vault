package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// Logger crée un middleware de logging pour Fiber
func Logger(log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Exécution de la requête
		err := c.Next()

		// Calcul de la durée
		duration := time.Since(start)

		// Logging de la requête
		event := log.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("duration", duration).
			Str("ip", c.IP())

		if err != nil {
			event.Err(err)
		}

		event.Msg("HTTP request")

		return err
	}
}

