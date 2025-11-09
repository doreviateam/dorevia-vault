package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/config"
	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/doreviateam/dorevia-vault/internal/models"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// InvoicePayload représente le payload JSON pour l'endpoint /api/v1/invoices
type InvoicePayload struct {
	Source      string                 `json:"source"`       // sales|purchase|pos|stock|sale
	Model       string                 `json:"model"`         // account.move, pos.order, etc.
	OdooID      int                    `json:"odoo_id"`       // ID dans Odoo
	State       string                 `json:"state"`         // posted, paid, done, etc.
	PDPRequired bool                   `json:"pdp_required"`  // Nécessite dispatch PDP ?
	File        string                 `json:"file"`          // Base64 encoded file
	Meta        map[string]interface{} `json:"meta,omitempty"` // Métadonnées facture
}

// InvoiceResponse représente la réponse de l'endpoint /api/v1/invoices
type InvoiceResponse struct {
	ID          string    `json:"id"`
	SHA256Hex   string    `json:"sha256_hex"`
	CreatedAt   time.Time `json:"created_at"`
	EvidenceJWS *string   `json:"evidence_jws,omitempty"` // JWS si disponible
	LedgerHash  *string   `json:"ledger_hash,omitempty"`  // Hash ledger si disponible
	Message     string    `json:"message,omitempty"`       // Pour idempotence
}

// InvoicesHandler gère l'endpoint POST /api/v1/invoices
// Intègre JWS + Ledger si configurés
func InvoicesHandler(db *storage.DB, storageDir string, jwsService *crypto.Service, cfg *config.Config, log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Database not configured",
			})
		}

		// Parser le payload JSON
		var payload InvoicePayload
		if err := c.BodyParser(&payload); err != nil {
			log.Error().Err(err).Msg("Failed to parse invoice payload")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid JSON payload",
				"details": err.Error(),
			})
		}

		// Validation des champs obligatoires
		if payload.Source == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing required field: source",
			})
		}
		if payload.Model == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing required field: model",
			})
		}
		if payload.File == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing required field: file",
			})
		}

		// Décoder le fichier base64
		fileContent, err := base64.StdEncoding.DecodeString(payload.File)
		if err != nil {
			log.Error().Err(err).Msg("Failed to decode base64 file")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid base64 file encoding",
				"details": err.Error(),
			})
		}

		// Extraire le nom de fichier depuis meta ou utiliser un nom par défaut
		filename := "document.pdf"
		if payload.Meta != nil {
			if number, ok := payload.Meta["number"].(string); ok && number != "" {
				filename = fmt.Sprintf("%s.pdf", number)
			}
		}

		// Construire le document
		doc := &models.Document{
			Filename:    filename,
			ContentType: "application/pdf", // Par défaut, peut être amélioré avec détection MIME
			SizeBytes:   int64(len(fileContent)),
			Source:     &payload.Source,
			OdooModel:    &payload.Model,
			OdooID:       &payload.OdooID,
			OdooState:    &payload.State,
			PDPRequired:  &payload.PDPRequired,
		}

		// Définir dispatch_status par défaut
		defaultStatus := "PENDING"
		doc.DispatchStatus = &defaultStatus

		// Extraire les métadonnées facture si présentes
		if payload.Meta != nil {
			if number, ok := payload.Meta["number"].(string); ok {
				doc.InvoiceNumber = &number
			}
			if dateStr, ok := payload.Meta["invoice_date"].(string); ok {
				if date, err := time.Parse("2006-01-02", dateStr); err == nil {
					doc.InvoiceDate = &date
				}
			}
			if totalHT, ok := payload.Meta["total_ht"].(float64); ok {
				doc.TotalHT = &totalHT
			}
			if totalTTC, ok := payload.Meta["total_ttc"].(float64); ok {
				doc.TotalTTC = &totalTTC
			}
			if currency, ok := payload.Meta["currency"].(string); ok {
				doc.Currency = &currency
			}
			if sellerVAT, ok := payload.Meta["seller_vat"].(string); ok {
				doc.SellerVAT = &sellerVAT
			}
			if buyerVAT, ok := payload.Meta["buyer_vat"].(string); ok {
				doc.BuyerVAT = &buyerVAT
			}
		}

		// Stocker le document avec JWS + Ledger (si configurés)
		ctx := context.Background()
		
		// Utiliser StoreDocumentWithEvidence si JWS ou Ledger activés
		if (cfg.JWSEnabled && jwsService != nil) || cfg.LedgerEnabled {
			err = db.StoreDocumentWithEvidence(ctx, doc, fileContent, storageDir, jwsService, cfg.JWSEnabled, cfg.JWSRequired, cfg.LedgerEnabled)
		} else {
			// Fallback vers méthode simple (Sprint 1)
			err = db.StoreDocumentWithTransaction(ctx, doc, fileContent, storageDir)
		}

		// Gérer l'idempotence (document déjà existant)
		if err != nil {
			if docExistsErr, ok := err.(storage.ErrDocumentExists); ok {
				// Récupérer les informations du document existant
				existingDoc, err := db.GetDocumentByID(ctx, docExistsErr.ID)
				if err != nil {
					log.Error().Err(err).Msg("Failed to retrieve existing document")
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": "Failed to retrieve existing document",
					})
				}

				// Document déjà existant - retourner 200 OK avec les infos existantes
				log.Info().
					Str("document_id", existingDoc.ID.String()).
					Str("sha256", existingDoc.SHA256Hex).
					Msg("Document already exists (idempotence)")

				// Vérifier si ledger existe pour document existant (idempotence renforcée)
				var hasLedger bool
				if cfg.LedgerEnabled && db != nil {
					tx, _ := db.Pool.Begin(ctx)
					if tx != nil {
						hasLedger, _ = ledger.ExistsByDocumentID(ctx, tx, existingDoc.ID)
						tx.Rollback(ctx)
					}
					// Si document existe mais pas de ledger, compléter le ledger
					if !hasLedger && jwsService != nil {
						// TODO: Compléter le ledger si nécessaire (optionnel)
					}
				}

				return c.Status(fiber.StatusOK).JSON(InvoiceResponse{
					ID:          existingDoc.ID.String(),
					SHA256Hex:   existingDoc.SHA256Hex,
					CreatedAt:   existingDoc.CreatedAt,
					EvidenceJWS: existingDoc.EvidenceJWS,
					LedgerHash:  existingDoc.LedgerHash,
					Message:     "Document already exists",
				})
			}

			log.Error().Err(err).Msg("Failed to store document")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to store document",
				"details": err.Error(),
			})
		}

		// Succès - retourner 201 Created
		log.Info().
			Str("document_id", doc.ID.String()).
			Str("sha256", doc.SHA256Hex).
			Int("odoo_id", payload.OdooID).
			Msg("Document vaulted successfully")

		return c.Status(fiber.StatusCreated).JSON(InvoiceResponse{
			ID:          doc.ID.String(),
			SHA256Hex:   doc.SHA256Hex,
			CreatedAt:   doc.CreatedAt,
			EvidenceJWS: doc.EvidenceJWS,
			LedgerHash:  doc.LedgerHash,
		})
	}
}
