package handlers

import (
	"context"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/config"
	"github.com/doreviateam/dorevia-vault/internal/metrics"
	"github.com/doreviateam/dorevia-vault/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// PosTicketPayload représente le payload JSON pour l'endpoint /api/v1/pos-tickets
// Note: Ce type est utilisé uniquement dans le handler. Le service utilise services.PosTicketInput
type PosTicketPayload struct {
	Tenant       string                 `json:"tenant"`        // Obligatoire
	SourceSystem string                 `json:"source_system"` // Défaut: "odoo_pos"
	SourceModel  string                 `json:"source_model"`  // Obligatoire (ex: "pos.order")
	SourceID     string                 `json:"source_id"`     // Obligatoire
	Currency     *string                `json:"currency,omitempty"`
	TotalInclTax *float64               `json:"total_incl_tax,omitempty"`
	TotalExclTax *float64               `json:"total_excl_tax,omitempty"`
	PosSession   *string                `json:"pos_session,omitempty"`
	Cashier      *string                `json:"cashier,omitempty"`
	Location     *string                `json:"location,omitempty"`
	Ticket       map[string]interface{} `json:"ticket"` // Obligatoire (JSON brut)
}

// PosTicketResponse représente la réponse standardisée
type PosTicketResponse struct {
	ID          string    `json:"id"`
	Tenant      string    `json:"tenant"`       // Ajouté pour cohérence
	SHA256Hex   string    `json:"sha256_hex"`
	LedgerHash  *string   `json:"ledger_hash,omitempty"`
	EvidenceJWS *string   `json:"evidence_jws,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// PosTicketsHandler gère l'endpoint POST /api/v1/pos-tickets
func PosTicketsHandler(
	service services.PosTicketsServiceInterface,
	cfg *config.Config,
	log *zerolog.Logger,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Validation taille (configurable)
		maxSize := cfg.PosTicketMaxSizeBytes
		if maxSize == 0 {
			maxSize = 64 * 1024 // 64 KB par défaut
		}
		if len(c.Body()) > maxSize {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
				"error":         "Payload too large",
				"max_size_bytes": maxSize,
			})
		}

		// Parser le payload
		var payload PosTicketPayload
		if err := c.BodyParser(&payload); err != nil {
			log.Error().Err(err).Msg("Failed to parse POS ticket payload")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "Invalid JSON payload",
				"details": err.Error(),
			})
		}

		// Validation
		if payload.Tenant == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing required field: tenant",
			})
		}
		if payload.SourceModel == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing required field: source_model",
			})
		}
		if payload.SourceID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing required field: source_id",
			})
		}
		if payload.Ticket == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing required field: ticket",
			})
		}

		// Valeur par défaut pour source_system
		if payload.SourceSystem == "" {
			payload.SourceSystem = "odoo_pos"
		}

		// Mapper handlers.PosTicketPayload → services.PosTicketInput
		input := services.PosTicketInput{
			Tenant:       payload.Tenant,
			SourceSystem: payload.SourceSystem,
			SourceModel:  payload.SourceModel,
			SourceID:     payload.SourceID,
			Currency:     payload.Currency,
			TotalInclTax: payload.TotalInclTax,
			TotalExclTax: payload.TotalExclTax,
			PosSession:   payload.PosSession,
			Cashier:      payload.Cashier,
			Location:     payload.Location,
			Ticket:       payload.Ticket,
		}

		// Appeler le service
		ctx := context.Background()
		startTime := time.Now()
		result, err := service.Ingest(ctx, input)
		duration := time.Since(startTime).Seconds()

		if err != nil {
			// Gérer les erreurs selon le type
			log.Error().
				Err(err).
				Str("tenant", payload.Tenant).
				Str("source_model", payload.SourceModel).
				Str("source_id", payload.SourceID).
				Float64("duration_seconds", duration).
				Msg("Failed to ingest POS ticket")

			// Métrique d'erreur
			metrics.RecordDocumentVaulted("error", "pos")
			metrics.RecordDocumentStorageDuration("pos_ingest", duration)

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to ingest POS ticket",
			})
		}

		// Déterminer le statut (idempotent si le document existait déjà)
		status := "success"
		if result.CreatedAt.Before(startTime.Add(-100 * time.Millisecond)) {
			// Si le document a été créé avant notre appel, c'est un cas idempotent
			status = "idempotent"
		}

		// Logs structurés
		logEntry := log.Info().
			Str("tenant", result.Tenant).
			Str("source_model", payload.SourceModel).
			Str("source_id", payload.SourceID).
			Str("document_id", result.ID.String()).
			Str("sha256_hex", result.SHA256Hex).
			Float64("duration_seconds", duration)

		if result.LedgerHash != nil {
			logEntry = logEntry.Str("ledger_hash", *result.LedgerHash)
		}
		if result.EvidenceJWS != nil {
			logEntry = logEntry.Str("evidence_jws", *result.EvidenceJWS)
		}

		logEntry.Msg("POS ticket ingested")

		// Métriques Prometheus
		metrics.RecordDocumentVaulted(status, "pos")
		metrics.RecordDocumentStorageDuration("pos_ingest", duration)

		// Retourner la réponse standardisée
		statusCode := fiber.StatusCreated
		if status == "idempotent" {
			// Pour idempotence, on retourne 200 OK au lieu de 201 Created
			statusCode = fiber.StatusOK
		}

		return c.Status(statusCode).JSON(PosTicketResponse{
			ID:          result.ID.String(),
			Tenant:      result.Tenant,
			SHA256Hex:   result.SHA256Hex,
			LedgerHash:  result.LedgerHash,
			EvidenceJWS: result.EvidenceJWS,
			CreatedAt:   result.CreatedAt,
		})
	}
}

// GetPosTicket gère GET /api/v1/pos-tickets -> 405 Method Not Allowed
// Retourne une erreur claire avec header Allow: POST
func GetPosTicket(c *fiber.Ctx) error {
	c.Set("Allow", "POST")
	return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{
		"error":   "Method Not Allowed",
		"message": "Only POST method is allowed for /api/v1/pos-tickets",
	})
}

