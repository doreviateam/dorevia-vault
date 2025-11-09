package ledger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AppendLedger ajoute une entrée au ledger avec hash chaîné
// Utilise un verrou exclusif (FOR UPDATE) pour éviter les race conditions
func AppendLedger(ctx context.Context, tx pgx.Tx, docID uuid.UUID, shaHex, jws string) (string, error) {
	// 1. Récupérer le previous_hash avec verrou exclusif
	// Le verrou FOR UPDATE empêche les autres transactions de lire/modifier
	// le dernier enregistrement pendant cette transaction
	var previousHash *string
	err := tx.QueryRow(ctx, `
		SELECT hash FROM ledger 
		ORDER BY timestamp DESC, id DESC 
		LIMIT 1 
		FOR UPDATE
	`).Scan(&previousHash)

	// 2. Calculer le nouveau hash
	var newHash string
	if err == pgx.ErrNoRows || previousHash == nil {
		// Premier enregistrement : hash = SHA256(sha256_document)
		hash := sha256.Sum256([]byte(shaHex))
		newHash = hex.EncodeToString(hash[:])
	} else if err != nil {
		return "", fmt.Errorf("failed to get previous hash: %w", err)
	} else {
		// Chaînage : hash = SHA256(previous_hash + sha256_document)
		combined := *previousHash + shaHex
		hash := sha256.Sum256([]byte(combined))
		newHash = hex.EncodeToString(hash[:])
	}

	// 3. Insérer dans le ledger avec ON CONFLICT pour idempotence
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger (document_id, hash, previous_hash, evidence_jws)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (document_id, hash) DO NOTHING
	`, docID, newHash, previousHash, jws)

	if err != nil {
		return "", fmt.Errorf("failed to insert into ledger: %w", err)
	}

	return newHash, nil
}

// ExistsByDocumentID vérifie si un document existe déjà dans le ledger
func ExistsByDocumentID(ctx context.Context, tx pgx.Tx, docID uuid.UUID) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM ledger WHERE document_id = $1)
	`, docID).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("failed to check ledger existence: %w", err)
	}

	return exists, nil
}

