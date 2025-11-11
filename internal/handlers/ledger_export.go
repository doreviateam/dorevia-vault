package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// LedgerExportHandler gère l'endpoint GET /api/v1/ledger/export
// Params:
//   - format: "json" (défaut) ou "csv"
//   - limit:  nombre de lignes (1..10000, défaut 100)
//   - offset: décalage (défaut 0)
func LedgerExportHandler(db *storage.DB, log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Database not configured",
			})
		}

		// Parse query params
		format := c.Query("format", "json") // json or csv
		limitStr := c.Query("limit", "100")
		offsetStr := c.Query("offset", "0")

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			limit = 100
		}
		if limit > 10000 {
			limit = 10000
		}

		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			offset = 0
		}

		ctx := context.Background()

		switch format {
		case "json":
			c.Set("Content-Type", "application/json")

			rows, err := db.ExportLedger(ctx, limit, offset)
			if err != nil {
				log.Error().Err(err).Msg("Failed to export ledger JSON")
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to export ledger",
					"details": err.Error(),
				})
			}

			// (simple) renvoyer juste le tableau; si tu veux la pagination, on peut l’ajouter
			return c.JSON(rows)

		case "csv":
			c.Set("Content-Type", "text/csv")
			c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=ledger_%d_%d.csv", limit, offset))

			rows, err := db.ExportLedger(ctx, limit, offset)
			if err != nil {
				log.Error().Err(err).Msg("Failed to export ledger CSV")
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Failed to export ledger",
					"details": err.Error(),
				})
			}

			// Écriture CSV minimaliste
			w := c.Response().BodyWriter()
			_, _ = w.Write([]byte("id,document_id,hash,previous_hash,seq,timestamp\n"))
			for _, r := range rows {
				prev := ""
				if r.PrevHash != nil {
					prev = *r.PrevHash
				}
				ts := r.Timestamp.UTC().Format(time.RFC3339)
				line := fmt.Sprintf("%d,%s,%s,%s,%d,%s\n", r.ID, r.DocumentID, r.Hash, prev, r.Seq, ts)
				_, _ = w.Write([]byte(line))
			}
			return nil

		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid format. Use 'json' or 'csv'",
			})
		}
	}
}
