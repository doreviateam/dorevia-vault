package ledger

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Service définit les opérations sur le ledger
// Interface pour abstraction de la couche ledger (Sprint 6)
type Service interface {
	// Append ajoute une entrée au ledger avec hash chaîné
	// Prend une transaction en paramètre pour garantir l'atomicité
	Append(ctx context.Context, tx pgx.Tx, docID uuid.UUID, shaHex, jws string) (string, error)

	// ExistsByDocumentID vérifie si un document existe dans le ledger
	ExistsByDocumentID(ctx context.Context, tx pgx.Tx, docID uuid.UUID) (bool, error)
}

