package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// DB représente le pool de connexions PostgreSQL
type DB struct {
	Pool *pgxpool.Pool
	log  *zerolog.Logger
}

// NewDB crée une nouvelle connexion à PostgreSQL
func NewDB(ctx context.Context, databaseURL string, log *zerolog.Logger) (*DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test de la connexion
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		Pool: pool,
		log:  log,
	}

	// Migration automatique
	if err := db.migrate(ctx); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Info().Msg("PostgreSQL connection established and migrations applied")

	return db, nil
}

// migrate applique les migrations de base de données
func (db *DB) migrate(ctx context.Context) error {
	createExtensionSQL := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS documents (
			id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			filename     TEXT NOT NULL,
			content_type TEXT,
			size_bytes   BIGINT,
			sha256_hex   TEXT NOT NULL,
			stored_path  TEXT NOT NULL,
			created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`

	// Créer l'extension
	if _, err := db.Pool.Exec(ctx, createExtensionSQL); err != nil {
		return fmt.Errorf("failed to create uuid extension: %w", err)
	}

	// Créer la table
	if _, err := db.Pool.Exec(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create documents table: %w", err)
	}

	// Migration Sprint 1 : Ajouter les champs Odoo
	if err := db.migrateSprint1(ctx); err != nil {
		return fmt.Errorf("failed to apply Sprint 1 migration: %w", err)
	}

	// Migration Sprint 2 : Table ledger
	if err := db.migrateSprint2(ctx); err != nil {
		return fmt.Errorf("failed to apply Sprint 2 migration: %w", err)
	}

	// Migration Sprint 6 : Champs POS
	if err := db.migrateSprint6(ctx); err != nil {
		return fmt.Errorf("failed to apply Sprint 6 migration: %w", err)
	}

	db.log.Debug().Msg("Database migrations applied successfully")
	return nil
}

// migrateSprint1 applique la migration Sprint 1 (métadonnées Odoo)
func (db *DB) migrateSprint1(ctx context.Context) error {
	migrationSQL := `
		-- Métadonnées Odoo
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS source TEXT;
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_model TEXT;
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_id INTEGER;
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_state TEXT;

		-- Routage PDP (préparation Sprint 2)
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS pdp_required BOOLEAN DEFAULT false;
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS dispatch_status TEXT DEFAULT 'PENDING';

		-- Métadonnées facture (préparation Sprint 2)
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS invoice_number TEXT;
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS invoice_date DATE;
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS total_ht DECIMAL(10,2);
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS total_ttc DECIMAL(10,2);
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS currency TEXT;
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS seller_vat TEXT;
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS buyer_vat TEXT;

		-- Index pour recherche rapide
		CREATE INDEX IF NOT EXISTS idx_documents_odoo_id ON documents(odoo_id);
		CREATE INDEX IF NOT EXISTS idx_documents_dispatch_status ON documents(dispatch_status);
		CREATE INDEX IF NOT EXISTS idx_documents_source ON documents(source);

		-- Contrainte sur dispatch_status
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_dispatch_status'
			) THEN
				ALTER TABLE documents ADD CONSTRAINT chk_dispatch_status 
					CHECK (dispatch_status IN ('PENDING', 'SENT', 'ACK', 'REJECTED'));
			END IF;
		END $$;
	`

	if _, err := db.Pool.Exec(ctx, migrationSQL); err != nil {
		return fmt.Errorf("failed to apply Sprint 1 migration: %w", err)
	}

	db.log.Debug().Msg("Sprint 1 migration applied successfully")
	return nil
}

// migrateSprint2 applique la migration Sprint 2 (table ledger)
func (db *DB) migrateSprint2(ctx context.Context) error {
	migrationSQL := `
		-- Table ledger
		CREATE TABLE IF NOT EXISTS ledger (
		  id SERIAL PRIMARY KEY,
		  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
		  hash TEXT NOT NULL,
		  previous_hash TEXT,
		  timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
		  evidence_jws TEXT
		);

		-- Index pour recherche rapide
		CREATE INDEX IF NOT EXISTS idx_ledger_document_id ON ledger(document_id);
		CREATE INDEX IF NOT EXISTS idx_ledger_timestamp ON ledger(timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_ledger_hash ON ledger(hash);
		CREATE INDEX IF NOT EXISTS idx_ledger_prev_hash ON ledger(previous_hash);

		-- Index composite pour SELECT previous_hash optimisé
		CREATE INDEX IF NOT EXISTS idx_ledger_ts_id_desc ON ledger(timestamp DESC, id DESC);

		-- Contrainte d'unicité
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'uq_ledger_doc_hash'
			) THEN
				ALTER TABLE ledger ADD CONSTRAINT uq_ledger_doc_hash 
					UNIQUE (document_id, hash);
			END IF;
		END $$;
	`

	if _, err := db.Pool.Exec(ctx, migrationSQL); err != nil {
		return fmt.Errorf("failed to apply Sprint 2 migration: %w", err)
	}

	db.log.Debug().Msg("Sprint 2 migration applied successfully")
	return nil
}

// migrateSprint6 applique la migration Sprint 6 (champs POS)
func (db *DB) migrateSprint6(ctx context.Context) error {
	migrationSQL := `
		-- Champ pour stocker le JSON brut du ticket POS
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS payload_json JSONB;

		-- Champ pour source_id textuel (pour POS avec IDs string)
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS source_id_text TEXT;

		-- Champs métier POS (optionnels, NULL pour les documents non-POS)
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS pos_session TEXT;
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS cashier TEXT;
		ALTER TABLE documents ADD COLUMN IF NOT EXISTS location TEXT;

		-- Index pour recherche rapide sur payload_json (GIN index pour JSONB)
		CREATE INDEX IF NOT EXISTS idx_documents_payload_json ON documents USING GIN (payload_json);

		-- Index pour recherche POS
		CREATE INDEX IF NOT EXISTS idx_documents_source_id_text ON documents(source_id_text) WHERE source = 'pos';
		CREATE INDEX IF NOT EXISTS idx_documents_pos_session ON documents(pos_session) WHERE source = 'pos';
		CREATE INDEX IF NOT EXISTS idx_documents_cashier ON documents(cashier) WHERE source = 'pos';
		CREATE INDEX IF NOT EXISTS idx_documents_location ON documents(location) WHERE source = 'pos';

		-- Index composite pour recherche par source + odoo_model (optimisation POS)
		CREATE INDEX IF NOT EXISTS idx_documents_source_model ON documents(source, odoo_model) 
			WHERE source = 'pos' AND odoo_model = 'pos.order';
	`

	if _, err := db.Pool.Exec(ctx, migrationSQL); err != nil {
		return fmt.Errorf("failed to apply Sprint 6 migration: %w", err)
	}

	db.log.Debug().Msg("Sprint 6 migration applied successfully")
	return nil
}

// Close ferme le pool de connexions
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// Health vérifie l'état de la connexion à la base de données
func (db *DB) Health(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// ErrDocumentExists est retourné quand un document avec le même hash existe déjà
type ErrDocumentExists struct {
	ID uuid.UUID
}

func (e ErrDocumentExists) Error() string {
	return fmt.Sprintf("document already exists with id: %s", e.ID.String())
}

// StoreDocumentWithTransaction stocke un document avec transaction atomique
// Pattern Transaction Outbox : garantit la cohérence fichier ↔ DB
func (db *DB) StoreDocumentWithTransaction(ctx context.Context, doc *models.Document, content []byte, storageDir string) error {
	// 1. Calculer hash avant transaction
	hash := sha256.Sum256(content)
	sha256Hex := hex.EncodeToString(hash[:])

	// 2. Vérifier idempotence (SELECT avant transaction)
	var existingID uuid.UUID
	err := db.Pool.QueryRow(ctx, "SELECT id FROM documents WHERE sha256_hex = $1 LIMIT 1", sha256Hex).Scan(&existingID)
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

	// 4. Créer chemin temporaire puis final (déplacement après COMMIT)
	tmpPath := filepath.Join(datePath, fmt.Sprintf("%s-%s.tmp", docID.String(), doc.Filename))
	finalPath := filepath.Join(datePath, fmt.Sprintf("%s-%s", docID.String(), doc.Filename))

	// 5. BEGIN transaction
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 6. Stocker fichier sur disque (fichier temporaire)
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// 7. INSERT dans documents (utiliser finalPath pour stored_path)
	_, err = tx.Exec(ctx, `
		INSERT INTO documents (
			id, filename, content_type, size_bytes, sha256_hex, stored_path,
			source, odoo_model, odoo_id, odoo_state, pdp_required, dispatch_status,
			invoice_number, invoice_date, total_ht, total_ttc, currency, seller_vat, buyer_vat
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, docID, doc.Filename, doc.ContentType, doc.SizeBytes, sha256Hex, finalPath,
		doc.Source, doc.OdooModel, doc.OdooID, doc.OdooState, doc.PDPRequired, doc.DispatchStatus,
		doc.InvoiceNumber, doc.InvoiceDate, doc.TotalHT, doc.TotalTTC, doc.Currency, doc.SellerVAT, doc.BuyerVAT)

	if err != nil {
		// Nettoyage fichier temporaire en cas d'erreur
		os.Remove(tmpPath)
		return fmt.Errorf("failed to insert document: %w", err)
	}

	// 8. COMMIT
	if err := tx.Commit(ctx); err != nil {
		os.Remove(tmpPath) // Nettoyage fichier temporaire
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 9. Déplacer fichier temporaire vers final (après COMMIT réussi)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		// Si le move échoue après COMMIT, on a une incohérence
		// On log l'erreur mais on ne rollback pas (déjà commité)
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

	db.log.Info().
		Str("document_id", docID.String()).
		Str("sha256", sha256Hex).
		Msg("Document stored successfully with transaction")

	return nil
}

