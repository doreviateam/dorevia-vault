package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// VerificationResult représente le résultat d'une vérification d'intégrité
type VerificationResult struct {
	Valid      bool     `json:"valid"`                // true si toutes les vérifications passent
	DocumentID string   `json:"document_id"`         // ID du document vérifié
	Checks     []Check  `json:"checks"`               // Détails des vérifications
	Errors     []string `json:"errors,omitempty"`    // Erreurs rencontrées
	Timestamp  string   `json:"timestamp"`           // Timestamp de la vérification
}

// Check représente une vérification individuelle
type Check struct {
	Component string `json:"component"` // "file", "database", "ledger"
	Status    string `json:"status"`    // "ok", "error", "missing"
	Message   string `json:"message"`   // Message détaillé
}

// VerifyDocumentIntegrity vérifie l'intégrité complète d'un document
// Vérifie la cohérence entre fichier, base de données et ledger
func VerifyDocumentIntegrity(
	ctx context.Context,
	db *storage.DB,
	docID uuid.UUID,
) (*VerificationResult, error) {
	result := &VerificationResult{
		DocumentID: docID.String(),
		Checks:      []Check{},
		Errors:      []string{},
		Valid:       true,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	// 1. Vérifier présence en DB
	doc, err := db.GetDocumentByID(ctx, docID)
	if err != nil {
		if err == pgx.ErrNoRows {
			result.Checks = append(result.Checks, Check{
				Component: "database",
				Status:    "missing",
				Message:   "Document not found in database",
			})
			result.Valid = false
			result.Errors = append(result.Errors, "Document not found in database")
			return result, nil
		}
		return nil, fmt.Errorf("failed to query document: %w", err)
	}

	// Document trouvé en DB
	result.Checks = append(result.Checks, Check{
		Component: "database",
		Status:    "ok",
		Message:   fmt.Sprintf("Document found: %s", doc.Filename),
	})

	// 2. Vérifier fichier sur disque
	if doc.StoredPath == "" {
		result.Checks = append(result.Checks, Check{
			Component: "file",
			Status:    "missing",
			Message:   "No stored_path in database",
		})
		result.Valid = false
		result.Errors = append(result.Errors, "No stored_path in database")
		return result, nil
	}

	// Vérifier existence du fichier
	fileInfo, err := os.Stat(doc.StoredPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.Checks = append(result.Checks, Check{
				Component: "file",
				Status:    "missing",
				Message:   fmt.Sprintf("File not found: %s", doc.StoredPath),
			})
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("File not found: %s", doc.StoredPath))
			return result, nil
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Vérifier taille du fichier
	if fileInfo.Size() != doc.SizeBytes {
		result.Checks = append(result.Checks, Check{
			Component: "file",
			Status:    "error",
			Message:   fmt.Sprintf("Size mismatch: expected %d, got %d", doc.SizeBytes, fileInfo.Size()),
		})
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("File size mismatch: expected %d, got %d", doc.SizeBytes, fileInfo.Size()))
		return result, nil
	}

	// Lire le fichier et calculer SHA256
	fileContent, err := os.ReadFile(doc.StoredPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Calculer SHA256 du fichier
	hash := sha256.Sum256(fileContent)
	calculatedSHA256 := hex.EncodeToString(hash[:])

	// Vérifier cohérence SHA256
	if calculatedSHA256 != doc.SHA256Hex {
		result.Checks = append(result.Checks, Check{
			Component: "file",
			Status:    "error",
			Message:   fmt.Sprintf("SHA256 mismatch: expected %s, got %s", doc.SHA256Hex, calculatedSHA256),
		})
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("SHA256 mismatch: file may have been tampered"))
		return result, nil
	}

	// Fichier OK
	result.Checks = append(result.Checks, Check{
		Component: "file",
		Status:    "ok",
		Message:   fmt.Sprintf("File exists, size=%d, SHA256=%s", fileInfo.Size(), calculatedSHA256),
	})

	// 3. Vérifier présence dans le ledger (si ledger activé)
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	hasLedger, err := ledger.ExistsByDocumentID(ctx, tx, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to check ledger: %w", err)
	}

	if hasLedger {
		// Vérifier que le ledger_hash dans documents correspond à une entrée ledger
		if doc.LedgerHash != nil {
			// Vérifier que l'entrée ledger existe avec ce hash
			var ledgerHash string
			err = tx.QueryRow(ctx, `
				SELECT hash FROM ledger 
				WHERE document_id = $1 AND hash = $2
			`, docID, *doc.LedgerHash).Scan(&ledgerHash)

			if err == nil {
				result.Checks = append(result.Checks, Check{
					Component: "ledger",
					Status:    "ok",
					Message:   fmt.Sprintf("Ledger entry found with hash: %s", *doc.LedgerHash),
				})
			} else if err == pgx.ErrNoRows {
				result.Checks = append(result.Checks, Check{
					Component: "ledger",
					Status:    "error",
					Message:   fmt.Sprintf("Ledger hash mismatch: expected %s but not found in ledger", *doc.LedgerHash),
				})
				result.Valid = false
				result.Errors = append(result.Errors, "Ledger hash mismatch")
			} else {
				return nil, fmt.Errorf("failed to query ledger: %w", err)
			}
		} else {
			result.Checks = append(result.Checks, Check{
				Component: "ledger",
				Status:    "warn",
				Message:   "Ledger entry exists but ledger_hash is NULL in documents table",
			})
		}
	} else {
		result.Checks = append(result.Checks, Check{
			Component: "ledger",
			Status:    "warn",
			Message:   "No ledger entry found (ledger may be disabled)",
		})
	}

	tx.Rollback(ctx) // Rollback car on ne fait que lire

	return result, nil
}

