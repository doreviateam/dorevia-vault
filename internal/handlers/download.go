package handlers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// DownloadHandler permet de télécharger un document par son ID
func DownloadHandler(db *storage.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Database not configured",
			})
		}

		// Parser l'ID
		idStr := c.Params("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid document ID",
			})
		}

		// Récupérer le document depuis la base
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		doc, err := db.GetDocumentByID(ctx, id)
		if err != nil {
			if err.Error() == "document not found" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"error": "Document not found",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve document",
			})
		}

		// Vérifier que le fichier existe
		if _, err := os.Stat(doc.StoredPath); os.IsNotExist(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "File not found on disk",
			})
		}

		// Définir les headers pour le téléchargement
		c.Set("Content-Type", doc.ContentType)
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, doc.Filename))
		c.Set("Content-Length", fmt.Sprintf("%d", doc.SizeBytes))

		// Ajouter ETag basé sur SHA256 pour cache HTTP
		if doc.SHA256Hex != "" {
			etag := fmt.Sprintf(`"%s"`, doc.SHA256Hex)
			c.Set("ETag", etag)

			// Vérifier If-None-Match pour 304 Not Modified
			if match := c.Get("If-None-Match"); match == etag {
				return c.SendStatus(fiber.StatusNotModified)
			}
		}

		// Envoyer le fichier
		return c.SendFile(doc.StoredPath)
	}
}

