package storage

import (
	"context"
	"fmt"

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

	db.log.Debug().Msg("Database migrations applied successfully")
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

