package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/doreviateam/dorevia-vault/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ListDocuments récupère une liste paginée de documents avec filtres
func (db *DB) ListDocuments(ctx context.Context, query models.DocumentQuery) ([]models.Document, int, error) {
	// Construction de la requête SQL avec filtres
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1

	// Filtre par recherche textuelle (filename)
	if query.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("filename ILIKE $%d", argIndex))
		args = append(args, "%"+query.Search+"%")
		argIndex++
	}

	// Filtre par type MIME
	if query.Type != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("content_type = $%d", argIndex))
		args = append(args, query.Type)
		argIndex++
	}

	// Filtre par date (from)
	if query.DateFrom != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *query.DateFrom)
		argIndex++
	}

	// Filtre par date (to)
	if query.DateTo != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *query.DateTo)
		argIndex++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Compter le total
	var total int
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM documents WHERE %s", whereSQL)
	err := db.Pool.QueryRow(ctx, countSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %w", err)
	}

	// Récupérer les documents avec pagination
	limit := query.Limit
	if limit <= 0 {
		limit = 20 // Par défaut
	}
	if limit > 100 {
		limit = 100 // Maximum
	}

	offset := (query.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	selectSQL := fmt.Sprintf(`
		SELECT id, filename, content_type, size_bytes, sha256_hex, stored_path, created_at
		FROM documents
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, selectSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var documents []models.Document
	for rows.Next() {
		var doc models.Document
		err := rows.Scan(
			&doc.ID,
			&doc.Filename,
			&doc.ContentType,
			&doc.SizeBytes,
			&doc.SHA256Hex,
			&doc.StoredPath,
			&doc.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan document: %w", err)
		}
		documents = append(documents, doc)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating documents: %w", err)
	}

	return documents, total, nil
}

// GetDocumentByID récupère un document par son ID
func (db *DB) GetDocumentByID(ctx context.Context, id uuid.UUID) (*models.Document, error) {
	var doc models.Document
	err := db.Pool.QueryRow(ctx, `
		SELECT id, filename, content_type, size_bytes, sha256_hex, stored_path, created_at,
		       source, odoo_model, odoo_id, odoo_state, pdp_required, dispatch_status,
		       invoice_number, invoice_date, total_ht, total_ttc, currency, seller_vat, buyer_vat,
		       evidence_jws, ledger_hash
		FROM documents
		WHERE id = $1
	`, id).Scan(
		&doc.ID,
		&doc.Filename,
		&doc.ContentType,
		&doc.SizeBytes,
		&doc.SHA256Hex,
		&doc.StoredPath,
		&doc.CreatedAt,
		&doc.Source,
		&doc.OdooModel,
		&doc.OdooID,
		&doc.OdooState,
		&doc.PDPRequired,
		&doc.DispatchStatus,
		&doc.InvoiceNumber,
		&doc.InvoiceDate,
		&doc.TotalHT,
		&doc.TotalTTC,
		&doc.Currency,
		&doc.SellerVAT,
		&doc.BuyerVAT,
		&doc.EvidenceJWS,
		&doc.LedgerHash,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("document not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return &doc, nil
}

// CalculatePages calcule le nombre de pages total
func CalculatePages(total, limit int) int {
	if limit <= 0 {
		return 1
	}
	pages := total / limit
	if total%limit > 0 {
		pages++
	}
	if pages == 0 {
		pages = 1
	}
	return pages
}

