package ledger

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// DefaultService implémente Service avec la logique existante
type DefaultService struct{}

// NewService crée un nouveau service ledger par défaut
func NewService() Service {
	return &DefaultService{}
}

// Append ajoute une entrée au ledger avec hash chaîné
func (s *DefaultService) Append(ctx context.Context, tx pgx.Tx, docID uuid.UUID, shaHex, jws string) (string, error) {
	return AppendLedger(ctx, tx, docID, shaHex, jws)
}

// ExistsByDocumentID vérifie si un document existe dans le ledger
func (s *DefaultService) ExistsByDocumentID(ctx context.Context, tx pgx.Tx, docID uuid.UUID) (bool, error) {
	return ExistsByDocumentID(ctx, tx, docID)
}

