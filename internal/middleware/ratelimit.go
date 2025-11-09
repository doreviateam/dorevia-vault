package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimit configure et retourne le middleware de rate limiting
func RateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        100,                // Nombre maximum de requêtes
		Expiration: 1 * time.Minute,    // Période de temps
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() // Limite par IP
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many requests, please try again later",
			})
		},
	})
}

