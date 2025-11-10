package handlers

import (
	"strconv"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// AuditExportHandler gère l'endpoint GET /audit/export
// Exporte les logs d'audit avec pagination
func AuditExportHandler(auditLogger *audit.Logger, log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		exporter := audit.NewExporter(auditLogger)

		// Parser les paramètres de requête
		opts := audit.ExportOptions{
			From:   c.Query("from"),
			To:     c.Query("to"),
			Format: audit.ExportFormat(c.Query("format", "json")),
		}

		// Parser page
		if pageStr := c.Query("page"); pageStr != "" {
			page, err := strconv.Atoi(pageStr)
			if err != nil || page < 1 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid page parameter (must be >= 1)",
				})
			}
			opts.Page = page
		}

		// Parser limit
		if limitStr := c.Query("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil || limit < 1 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid limit parameter (must be >= 1)",
				})
			}
			opts.Limit = limit
		}

		// Exporter
		result, err := exporter.Export(opts)
		if err != nil {
			log.Error().Err(err).Interface("opts", opts).Msg("failed to export audit logs")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to export audit logs",
			})
		}

		// Retourner selon le format
		switch opts.Format {
		case audit.ExportFormatCSV:
			csv, err := exporter.ExportToCSV(result)
			if err != nil {
				log.Error().Err(err).Msg("failed to convert to CSV")
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "failed to convert to CSV",
				})
			}
			c.Set("Content-Type", "text/csv")
			c.Set("Content-Disposition", "attachment; filename=audit-export.csv")
			return c.SendString(csv)

		case audit.ExportFormatJSON:
			fallthrough
		default:
			return c.JSON(result)
		}
	}
}

// AuditDatesHandler gère l'endpoint GET /audit/dates
// Liste les dates disponibles dans les logs
func AuditDatesHandler(auditLogger *audit.Logger, log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		exporter := audit.NewExporter(auditLogger)

		dates, err := exporter.ListAvailableDates()
		if err != nil {
			log.Error().Err(err).Msg("failed to list audit dates")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to list audit dates",
			})
		}

		return c.JSON(fiber.Map{
			"dates": dates,
			"count": len(dates),
		})
	}
}

