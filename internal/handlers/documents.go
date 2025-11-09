package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/models"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// DocumentsListHandler gère le listing et la recherche de documents
func DocumentsListHandler(db *storage.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Database not configured",
			})
		}

		// Parse des paramètres de requête
		query := models.DocumentQuery{
			Page:  1,
			Limit: 20,
		}

		// Page
		if pageStr := c.Query("page"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
				query.Page = page
			}
		}

		// Limit
		if limitStr := c.Query("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
				query.Limit = limit
			}
		}

		// Search
		if search := c.Query("search"); search != "" {
			query.Search = search
		}

		// Type (content_type)
		if docType := c.Query("type"); docType != "" {
			query.Type = docType
		}

		// Date from
		if dateFromStr := c.Query("date_from"); dateFromStr != "" {
			if dateFrom, err := time.Parse(time.RFC3339, dateFromStr); err == nil {
				query.DateFrom = &dateFrom
			}
		}

		// Date to
		if dateToStr := c.Query("date_to"); dateToStr != "" {
			if dateTo, err := time.Parse(time.RFC3339, dateToStr); err == nil {
				query.DateTo = &dateTo
			}
		}

		// Récupérer les documents
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		documents, total, err := db.ListDocuments(ctx, query)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to retrieve documents",
			})
		}

		// Calculer le nombre de pages
		pages := storage.CalculatePages(total, query.Limit)

		// Construire la réponse
		response := models.DocumentListResponse{
			Data: documents,
			Pagination: models.PaginationResponse{
				Page:  query.Page,
				Limit: query.Limit,
				Total: total,
				Pages: pages,
			},
		}

		return c.JSON(response)
	}
}

// DocumentByIDHandler récupère un document par son ID
func DocumentByIDHandler(db *storage.DB) fiber.Handler {
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

		// Récupérer le document
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

		return c.JSON(doc)
	}
}

