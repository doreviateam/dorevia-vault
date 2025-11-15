package storage

import (
	"context"

	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/doreviateam/dorevia-vault/internal/models"
)

// DocumentRepository définit les opérations de stockage des documents
// Interface pour abstraction de la couche de stockage (Sprint 6)
type DocumentRepository interface {
	// GetDocumentBySHA256 récupère un document par son hash SHA256
	GetDocumentBySHA256(ctx context.Context, sha256 string) (*models.Document, error)

	// InsertDocumentWithEvidence insère un document avec evidence JWS et ledger hash
	// Gère la transaction en interne, inclut l'ajout au ledger
	InsertDocumentWithEvidence(
		ctx context.Context,
		doc *models.Document,
		evidenceJWS string,
		ledgerService ledger.Service, // Service ledger pour ajout dans transaction
	) error
}

