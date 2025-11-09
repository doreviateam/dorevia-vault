package handlers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// LedgerExportHandler gère l'endpoint GET /api/v1/ledger/export
func LedgerExportHandler(db *storage.DB, log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Database not configured",
			})
		}

		// Parser les paramètres de requête
		format := c.Query("format", "json") // json ou csv
		limitStr := c.Query("limit", "100")
		offsetStr := c.Query("offset", "0")

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			limit = 100
		}

		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			offset = 0
		}

		// Protection : limit max 10000
		if limit > 10000 {
			limit = 10000
		}

		ctx := context.Background()

		// Exporter selon le format
		switch format {
		case "json":
			c.Set("Content-Type", "application/json")
			if err := ledger.ExportLedgerJSON(ctx, db.Pool, c.Response().BodyWriter(), limit, offset); err != nil {
				log.Error().Err(err).Msg("Failed to export ledger JSON")
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to export ledger",
					"details": err.Error(),
				})
			}
			return nil

		case "csv":
			c.Set("Content-Type", "text/csv")
			c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=ledger_%d_%d.csv", limit, offset))
			if err := ledger.ExportLedgerCSV(ctx, db.Pool, c.Response().BodyWriter(), limit, offset); err != nil {
				log.Error().Err(err).Msg("Failed to export ledger CSV")
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Failed to export ledger",
					"details": err.Error(),
				})
			}
			return nil

		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid format. Use 'json' or 'csv'",
			})
		}
	}
}

