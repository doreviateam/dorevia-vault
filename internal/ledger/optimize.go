package ledger

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// OptimizeDatabase crée les index optimisés pour le ledger
func OptimizeDatabase(ctx context.Context, pool *pgxpool.Pool, log zerolog.Logger) error {
	log.Info().Msg("Creating optimized indexes for ledger")
	
	indexes := []struct {
		name string
		sql   string
	}{
		{
			name: "ledger_timestamp_idx",
			sql: `
				CREATE INDEX IF NOT EXISTS ledger_timestamp_idx 
				ON ledger (timestamp DESC, id DESC)
			`,
		},
		{
			name: "ledger_document_id_idx",
			sql: `
				CREATE INDEX IF NOT EXISTS ledger_document_id_idx 
				ON ledger (document_id)
			`,
		},
		{
			name: "ledger_hash_idx",
			sql: `
				CREATE INDEX IF NOT EXISTS ledger_hash_idx 
				ON ledger (hash)
			`,
		},
		{
			name: "ledger_previous_hash_idx",
			sql: `
				CREATE INDEX IF NOT EXISTS ledger_previous_hash_idx 
				ON ledger (previous_hash)
				WHERE previous_hash IS NOT NULL
			`,
		},
		{
			name: "ledger_timestamp_month_idx",
			sql: `
				CREATE INDEX IF NOT EXISTS ledger_timestamp_month_idx 
				ON ledger (DATE_TRUNC('month', timestamp))
			`,
		},
	}
	
	for _, idx := range indexes {
		_, err := pool.Exec(ctx, idx.sql)
		if err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
		log.Debug().Str("index", idx.name).Msg("Index created")
	}
	
	log.Info().Int("indexes_created", len(indexes)).Msg("Database optimization completed")
	return nil
}

// AnalyzeTable exécute ANALYZE sur la table ledger pour optimiser les statistiques
func AnalyzeTable(ctx context.Context, pool *pgxpool.Pool, log zerolog.Logger) error {
	log.Info().Msg("Analyzing ledger table for query optimization")
	
	_, err := pool.Exec(ctx, "ANALYZE ledger")
	if err != nil {
		return fmt.Errorf("failed to analyze ledger table: %w", err)
	}
	
	log.Info().Msg("Ledger table analyzed")
	return nil
}

// VacuumTable exécute VACUUM sur la table ledger pour récupérer l'espace
func VacuumTable(ctx context.Context, pool *pgxpool.Pool, log zerolog.Logger) error {
	log.Info().Msg("Vacuuming ledger table")
	
	_, err := pool.Exec(ctx, "VACUUM ANALYZE ledger")
	if err != nil {
		return fmt.Errorf("failed to vacuum ledger table: %w", err)
	}
	
	log.Info().Msg("Ledger table vacuumed")
	return nil
}

// GetTableStats retourne les statistiques de la table ledger
func GetTableStats(ctx context.Context, pool *pgxpool.Pool) (TableStats, error) {
	var stats TableStats
	
	// Nombre total de lignes
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ledger
	`).Scan(&stats.TotalRows)
	if err != nil {
		return stats, fmt.Errorf("failed to get total rows: %w", err)
	}
	
	// Taille de la table
	err = pool.QueryRow(ctx, `
		SELECT pg_size_pretty(pg_total_relation_size('ledger'))
	`).Scan(&stats.TableSize)
	if err != nil {
		return stats, fmt.Errorf("failed to get table size: %w", err)
	}
	
	// Taille des index
	err = pool.QueryRow(ctx, `
		SELECT pg_size_pretty(pg_indexes_size('ledger'))
	`).Scan(&stats.IndexSize)
	if err != nil {
		return stats, fmt.Errorf("failed to get index size: %w", err)
	}
	
	// Nombre d'index
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM pg_indexes 
		WHERE tablename = 'ledger'
	`).Scan(&stats.IndexCount)
	if err != nil {
		return stats, fmt.Errorf("failed to get index count: %w", err)
	}
	
	return stats, nil
}

// TableStats contient les statistiques de la table ledger
type TableStats struct {
	TotalRows  int64
	TableSize  string
	IndexSize  string
	IndexCount int
}

