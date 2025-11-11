package storage

import (
	"context"
	"time"
)

// Ligne de ledger pour l’export (⚠️ timestamp en time.Time, PAS string)
type LedgerRow struct {
	ID         int64     `json:"id"`
	DocumentID string    `json:"document_id"`
	Hash       string    `json:"hash"`
	PrevHash   *string   `json:"prev_hash,omitempty"`
	Seq        int64     `json:"seq"`        // si tu n’as pas de seq, remplace par ID
	Timestamp  time.Time `json:"timestamp"`  // <— fix: time.Time
}

// ExportLedger lit une page du ledger
func (db *DB) ExportLedger(ctx context.Context, limit, offset int) ([]LedgerRow, error) {
	const q = `
		SELECT
			id,               -- SERIAL
			document_id,      -- UUID
			hash,
			previous_hash,
			id AS seq,        -- ou une vraie colonne seq si tu en as une
			timestamp
		FROM ledger
		ORDER BY id
		LIMIT $1 OFFSET $2;
	`

	rows, err := db.Pool.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]LedgerRow, 0, limit)
	for rows.Next() {
		var r LedgerRow
		if err := rows.Scan(&r.ID, &r.DocumentID, &r.Hash, &r.PrevHash, &r.Seq, &r.Timestamp); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountLedger renvoie le total pour paginer
func (db *DB) CountLedger(ctx context.Context) (int, error) {
	const q = `SELECT count(*) FROM ledger`
	var n int
	if err := db.Pool.QueryRow(ctx, q).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}
