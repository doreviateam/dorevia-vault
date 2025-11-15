package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/doreviateam/dorevia-vault/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// PostgresRepository implémente DocumentRepository pour PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
	log  *zerolog.Logger
}

// NewPostgresRepository crée un nouveau repository PostgreSQL
func NewPostgresRepository(pool *pgxpool.Pool, log *zerolog.Logger) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
		log:  log,
	}
}

// GetDocumentBySHA256 récupère un document par son hash SHA256
func (r *PostgresRepository) GetDocumentBySHA256(ctx context.Context, sha256Hex string) (*models.Document, error) {
	var doc models.Document
	var payloadJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT 
			id, filename, content_type, size_bytes, sha256_hex, stored_path, created_at,
			source, odoo_model, odoo_id, odoo_state, pdp_required, dispatch_status,
			invoice_number, invoice_date, total_ht, total_ttc, currency, seller_vat, buyer_vat,
			evidence_jws, ledger_hash,
			source_id_text, payload_json, pos_session, cashier, location
		FROM documents
		WHERE sha256_hex = $1
		LIMIT 1
	`, sha256Hex).Scan(
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
		// Champs POS (Sprint 6)
		&doc.SourceIDText,
		&payloadJSON,
		&doc.PosSession,
		&doc.Cashier,
		&doc.Location,
	)

	if err == pgx.ErrNoRows {
		return nil, nil // Document non trouvé (pas une erreur pour idempotence)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document by SHA256: %w", err)
	}

	// Désérialiser payload_json si présent
	if len(payloadJSON) > 0 {
		var payload map[string]interface{}
		if err := json.Unmarshal(payloadJSON, &payload); err == nil {
			// Stocker dans un champ si nécessaire (pour l'instant on garde juste les bytes)
			// doc.PayloadJSON = payloadJSON
		}
	}

	return &doc, nil
}

// InsertDocumentWithEvidence insère un document avec evidence JWS et ledger hash
// Gère la transaction en interne, inclut l'ajout au ledger
func (r *PostgresRepository) InsertDocumentWithEvidence(
	ctx context.Context,
	doc *models.Document,
	evidenceJWS string,
	ledgerService ledger.Service,
) error {
	// Timeout transaction (30s)
	transactionTimeout := 30 * time.Second
	txCtx, cancel := context.WithTimeout(ctx, transactionTimeout)
	defer cancel()

	// BEGIN transaction
	tx, err := r.pool.Begin(txCtx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(txCtx)

	// 1. INSERT dans documents (sans evidence_jws et ledger_hash pour l'instant)
	_, err = tx.Exec(txCtx, `
		INSERT INTO documents (
			id, filename, content_type, size_bytes, sha256_hex, stored_path,
			source, odoo_model, odoo_id, odoo_state, pdp_required, dispatch_status,
			invoice_number, invoice_date, total_ht, total_ttc, currency, seller_vat, buyer_vat,
			source_id_text, payload_json, pos_session, cashier, location,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
	`, doc.ID, doc.Filename, doc.ContentType, doc.SizeBytes, doc.SHA256Hex, doc.StoredPath,
		doc.Source, doc.OdooModel, doc.OdooID, doc.OdooState, doc.PDPRequired, doc.DispatchStatus,
		doc.InvoiceNumber, doc.InvoiceDate, doc.TotalHT, doc.TotalTTC, doc.Currency, doc.SellerVAT, doc.BuyerVAT,
		doc.SourceIDText, doc.PayloadJSON, doc.PosSession, doc.Cashier, doc.Location,
		doc.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	// 2. Ajouter au ledger (via interface)
	var ledgerHash string
	if ledgerService != nil {
		ledgerHash, err = ledgerService.Append(txCtx, tx, doc.ID, doc.SHA256Hex, evidenceJWS)
		if err != nil {
			return fmt.Errorf("failed to append to ledger: %w", err)
		}
	}

	// 3. UPDATE documents avec evidence_jws et ledger_hash
	if evidenceJWS != "" || ledgerHash != "" {
		_, err = tx.Exec(txCtx, `
			UPDATE documents 
			SET evidence_jws = $1, ledger_hash = $2
			WHERE id = $3
		`, evidenceJWS, ledgerHash, doc.ID)
		if err != nil {
			return fmt.Errorf("failed to update evidence: %w", err)
		}

		// Mettre à jour le document en mémoire
		if evidenceJWS != "" {
			doc.EvidenceJWS = &evidenceJWS
		}
		if ledgerHash != "" {
			doc.LedgerHash = &ledgerHash
		}
	}

	// 4. COMMIT
	if err := tx.Commit(txCtx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info().
		Str("document_id", doc.ID.String()).
		Str("sha256", doc.SHA256Hex).
		Bool("jws_generated", evidenceJWS != "").
		Bool("ledger_appended", ledgerHash != "").
		Msg("Document inserted with evidence via repository")

	return nil
}

