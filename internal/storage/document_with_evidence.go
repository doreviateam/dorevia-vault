package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/doreviateam/dorevia-vault/internal/metrics"
	"github.com/doreviateam/dorevia-vault/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// StoreDocumentWithEvidence stocke un document avec JWS + Ledger (Sprint 2)
// Flux complet : fichier → DB → JWS → Ledger → UPDATE evidence
// Sprint 3 : Ajout timeout transaction (30s par défaut) + métriques Prometheus
func (db *DB) StoreDocumentWithEvidence(
	ctx context.Context,
	doc *models.Document,
	content []byte,
	storageDir string,
	jwsService *crypto.Service,
	jwsEnabled, jwsRequired, ledgerEnabled bool,
) error {
	// Sprint 3 : Ajouter timeout transaction (30s)
	transactionTimeout := 30 * time.Second
	txCtx, cancel := context.WithTimeout(ctx, transactionTimeout)
	defer cancel()
	
	// Sprint 3 Phase 2 : Mesure durée stockage document
	storageStartTime := time.Now()
	defer func() {
		storageDuration := time.Since(storageStartTime).Seconds()
		metrics.RecordDocumentStorageDuration("store", storageDuration)
	}()

	// 1. Calculer hash avant transaction
	hash := sha256.Sum256(content)
	sha256Hex := hex.EncodeToString(hash[:])

	// 2. Vérifier idempotence (SELECT avant transaction)
	var existingID uuid.UUID
	err := db.Pool.QueryRow(txCtx, "SELECT id FROM documents WHERE sha256_hex = $1 LIMIT 1", sha256Hex).Scan(&existingID)
	if err == nil {
		// Document déjà existant
		doc.ID = existingID
		doc.SHA256Hex = sha256Hex
		return ErrDocumentExists{ID: existingID}
	}
	if err != pgx.ErrNoRows {
		return fmt.Errorf("failed to check existing document: %w", err)
	}

	// 3. Générer UUID et chemin
	docID := uuid.New()
	now := time.Now()
	datePath := filepath.Join(
		storageDir,
		fmt.Sprintf("%d", now.Year()),
		fmt.Sprintf("%02d", now.Month()),
		fmt.Sprintf("%02d", now.Day()),
	)

	if err := os.MkdirAll(datePath, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// 4. Créer chemin temporaire puis final
	tmpPath := filepath.Join(datePath, fmt.Sprintf("%s-%s.tmp", docID.String(), doc.Filename))
	finalPath := filepath.Join(datePath, fmt.Sprintf("%s-%s", docID.String(), doc.Filename))

	// 5. BEGIN transaction (avec timeout)
	tx, err := db.Pool.Begin(txCtx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(txCtx)

	// 6. Stocker fichier sur disque (fichier temporaire)
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// 7. INSERT dans documents (sans evidence_jws et ledger_hash pour l'instant)
	_, err = tx.Exec(txCtx, `
		INSERT INTO documents (
			id, filename, content_type, size_bytes, sha256_hex, stored_path,
			source, odoo_model, odoo_id, odoo_state, pdp_required, dispatch_status,
			invoice_number, invoice_date, total_ht, total_ttc, currency, seller_vat, buyer_vat,
			evidence_jws, ledger_hash
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`, docID, doc.Filename, doc.ContentType, doc.SizeBytes, sha256Hex, finalPath,
		doc.Source, doc.OdooModel, doc.OdooID, doc.OdooState, doc.PDPRequired, doc.DispatchStatus,
		doc.InvoiceNumber, doc.InvoiceDate, doc.TotalHT, doc.TotalTTC, doc.Currency, doc.SellerVAT, doc.BuyerVAT,
		nil, nil) // evidence_jws et ledger_hash seront mis à jour après

	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to insert document: %w", err)
	}

	// 8. Générer JWS (hors transaction mais rapide)
	var jws string
	if jwsEnabled && jwsService != nil {
		jwsStartTime := time.Now() // Sprint 3 Phase 2 : Mesure durée JWS
		jws, err = jwsService.SignEvidence(docID.String(), sha256Hex, now)
		jwsDuration := time.Since(jwsStartTime).Seconds()
		
		if err != nil {
			// Métrique : JWS échec (Sprint 3 Phase 2)
			metrics.RecordJWSSignature("error")
			metrics.RecordJWSSignatureDuration(jwsDuration)
			
			if jwsRequired {
				os.Remove(tmpPath)
				return fmt.Errorf("JWS required but generation failed: %w", err)
			}
			// Mode dégradé : continuer sans JWS
			metrics.RecordJWSSignature("degraded")
			db.log.Warn().Err(err).Msg("JWS generation failed, continuing without evidence")
		} else {
			// Métrique : JWS succès (Sprint 3 Phase 2)
			metrics.RecordJWSSignature("success")
			metrics.RecordJWSSignatureDuration(jwsDuration)
		}
	}

	// 9. AppendLedger (dans transaction avec verrou)
	var ledgerHash string
	if ledgerEnabled {
		ledgerStartTime := time.Now() // Sprint 3 Phase 2 : Mesure durée ledger
		ledgerHash, err = ledger.AppendLedger(txCtx, tx, docID, sha256Hex, jws)
		ledgerDuration := time.Since(ledgerStartTime).Seconds()
		
		if err != nil {
			// Métrique : erreur ledger (Sprint 4 Phase 4.1)
			metrics.RecordLedgerAppendError()
			os.Remove(tmpPath)
			return fmt.Errorf("failed to append to ledger: %w", err)
		}
		
		// Métrique : ledger append succès (Sprint 3 Phase 2)
		metrics.RecordLedgerAppendDuration(ledgerDuration)
		metrics.LedgerEntries.Inc() // Incrémenter compteur ledger
	}

	// 10. UPDATE documents avec evidence_jws et ledger_hash
	if jws != "" || ledgerHash != "" {
		_, err = tx.Exec(txCtx, `
			UPDATE documents 
			SET evidence_jws = $1, ledger_hash = $2
			WHERE id = $3
		`, jws, ledgerHash, docID)
		if err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("failed to update evidence: %w", err)
		}
	}

	// 11. COMMIT (avec timeout)
	if err := tx.Commit(txCtx); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 12. Déplacer fichier temporaire vers final (après COMMIT réussi)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		db.log.Error().
			Err(err).
			Str("tmp_path", tmpPath).
			Str("final_path", finalPath).
			Msg("Failed to move file after commit - manual cleanup required")
		return fmt.Errorf("failed to move file after commit: %w", err)
	}

	// Mettre à jour le document avec les valeurs finales
	doc.ID = docID
	doc.SHA256Hex = sha256Hex
	doc.StoredPath = finalPath
	doc.CreatedAt = now
	if jws != "" {
		doc.EvidenceJWS = &jws
	}
	if ledgerHash != "" {
		doc.LedgerHash = &ledgerHash
	}

	db.log.Info().
		Str("document_id", docID.String()).
		Str("sha256", sha256Hex).
		Bool("jws_generated", jws != "").
		Bool("ledger_appended", ledgerHash != "").
		Msg("Document stored successfully with evidence")

	return nil
}

