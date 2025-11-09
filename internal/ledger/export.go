package ledger

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ExportLedgerJSON exporte le ledger au format JSON avec pagination
func ExportLedgerJSON(ctx context.Context, pool *pgxpool.Pool, w io.Writer, limit, offset int) error {
	// Protection : limit max 10000
	if limit > 10000 {
		limit = 10000
	}
	if limit <= 0 {
		limit = 100
	}

	// Requête avec pagination
	rows, err := pool.Query(ctx, `
		SELECT id, document_id, hash, previous_hash, timestamp, evidence_jws
		FROM ledger
		ORDER BY timestamp ASC, id ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)

	if err != nil {
		return fmt.Errorf("failed to query ledger: %w", err)
	}
	defer rows.Close()

	// Collecter les entrées
	var entries []map[string]interface{}
	for rows.Next() {
		var id int
		var docID string
		var hash, prevHash, jws *string
		var timestamp string

		if err := rows.Scan(&id, &docID, &hash, &prevHash, &timestamp, &jws); err != nil {
			return fmt.Errorf("failed to scan ledger entry: %w", err)
		}

		entry := map[string]interface{}{
			"id":            id,
			"document_id":   docID,
			"hash":          hash,
			"previous_hash": prevHash,
			"timestamp":     timestamp,
			"evidence_jws":  jws,
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating ledger: %w", err)
	}

	// Encoder en JSON
	response := map[string]interface{}{
		"entries": entries,
		"limit":   limit,
		"offset":  offset,
		"total":   len(entries),
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// ExportLedgerCSV exporte le ledger au format CSV avec pagination
func ExportLedgerCSV(ctx context.Context, pool *pgxpool.Pool, w io.Writer, limit, offset int) error {
	// Protection : limit max 10000
	if limit > 10000 {
		limit = 10000
	}
	if limit <= 0 {
		limit = 100
	}

	// Requête avec pagination
	rows, err := pool.Query(ctx, `
		SELECT id, document_id, hash, previous_hash, timestamp, evidence_jws
		FROM ledger
		ORDER BY timestamp ASC, id ASC
		LIMIT $1 OFFSET $2
	`, limit, offset)

	if err != nil {
		return fmt.Errorf("failed to query ledger: %w", err)
	}
	defer rows.Close()

	// Créer le writer CSV
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// En-têtes CSV
	if err := writer.Write([]string{
		"id", "document_id", "hash", "previous_hash", "timestamp", "evidence_jws",
	}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Lignes de données
	for rows.Next() {
		var id int
		var docID, hash, prevHash, jws string
		var timestamp string

		if err := rows.Scan(&id, &docID, &hash, &prevHash, &timestamp, &jws); err != nil {
			return fmt.Errorf("failed to scan ledger entry: %w", err)
		}

		record := []string{
			fmt.Sprintf("%d", id),
			docID,
			hash,
			prevHash,
			timestamp,
			jws,
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating ledger: %w", err)
	}

	return nil
}

// GetLedgerTotal retourne le nombre total d'entrées dans le ledger
func GetLedgerTotal(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	var total int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM ledger").Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get ledger total: %w", err)
	}
	return total, nil
}

