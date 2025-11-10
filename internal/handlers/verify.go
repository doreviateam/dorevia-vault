package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/doreviateam/dorevia-vault/internal/verify"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// VerifyResponse représente la réponse de l'endpoint de vérification
type VerifyResponse struct {
	*verify.VerificationResult
	SignedProof *string `json:"signed_proof,omitempty"` // JWS signé du résultat si ?signed=true
}

// VerifyHandler gère l'endpoint GET /api/v1/ledger/verify/:document_id
// Option ?signed=true : Génère un JWS signé du résultat (preuve auditable)
func VerifyHandler(db *storage.DB, jwsService *crypto.Service, log *zerolog.Logger, auditLogger *audit.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Database not configured",
			})
		}

		// Parser le document_id depuis les paramètres
		docIDStr := c.Params("document_id")
		if docIDStr == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Missing document_id parameter",
			})
		}

		// Valider UUID
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid document_id format (must be UUID)",
			})
		}

		// Vérifier l'intégrité du document
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		startTime := time.Now()
		result, err := verify.VerifyDocumentIntegrity(ctx, db, docID)
		if err != nil {
			log.Error().Err(err).Str("document_id", docIDStr).Msg("Failed to verify document integrity")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to verify document integrity",
				"details": err.Error(),
			})
		}

		// Construire la réponse
		response := &VerifyResponse{
			VerificationResult: result,
		}

		// Option ?signed=true : Générer JWS signé du résultat
		if c.Query("signed") == "true" {
			if jwsService == nil {
				log.Warn().Msg("JWS service not available, cannot sign verification result")
				response.SignedProof = nil
			} else {
				// Sérialiser le résultat en JSON pour le signer
				resultJSON, err := json.Marshal(result)
				if err != nil {
					log.Error().Err(err).Msg("Failed to marshal verification result")
					response.SignedProof = nil
				} else {
					// Calculer le SHA256 du JSON pour la signature
					hash := sha256.Sum256(resultJSON)
					resultSHA256 := hex.EncodeToString(hash[:])
					
					// Signer le résultat (utiliser le SHA256 du JSON comme "document")
					jws, err := jwsService.SignEvidence(
						result.DocumentID,
						resultSHA256,
						time.Now(),
					)
					if err != nil {
						log.Error().Err(err).Msg("Failed to sign verification result")
						response.SignedProof = nil
					} else {
						response.SignedProof = &jws
					}
				}
			}
		}

		// Audit : vérification intégrité (Sprint 4 Phase 4.2)
		if auditLogger != nil {
			requestID := c.Get("X-Request-ID")
			status := audit.EventStatusSuccess
			if !result.Valid {
				status = audit.EventStatusError
			}
			auditLogger.Log(audit.Event{
				EventType:  audit.EventTypeVerificationRun,
				DocumentID: docIDStr,
				RequestID:  requestID,
				Status:     status,
				DurationMS: int64(time.Since(startTime).Milliseconds()),
				Metadata: map[string]interface{}{
					"valid":        result.Valid,
					"signed_proof": c.Query("signed") == "true",
					"checks":       len(result.Checks),
				},
			})
		}

		// Déterminer le code de statut HTTP
		statusCode := fiber.StatusOK
		if !result.Valid {
			statusCode = fiber.StatusConflict // 409 Conflict pour incohérence détectée
		}

		return c.Status(statusCode).JSON(response)
	}
}

