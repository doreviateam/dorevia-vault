package ledger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// PartitionManager gère le partitionnement mensuel du ledger
type PartitionManager struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

// NewPartitionManager crée un nouveau gestionnaire de partitions
func NewPartitionManager(pool *pgxpool.Pool, log zerolog.Logger) *PartitionManager {
	return &PartitionManager{
		pool: pool,
		log:  log,
	}
}

// EnsurePartition crée une partition pour un mois donné si elle n'existe pas
func (p *PartitionManager) EnsurePartition(ctx context.Context, year int, month int) error {
	partitionName := p.getPartitionName(year, month)
	
	// Vérifier si la partition existe déjà
	var exists bool
	err := p.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_class 
			WHERE relname = $1
		)
	`, partitionName).Scan(&exists)
	
	if err != nil {
		return fmt.Errorf("failed to check partition existence: %w", err)
	}
	
	if exists {
		p.log.Debug().
			Str("partition", partitionName).
			Msg("Partition already exists")
		return nil
	}
	
	// Créer la partition
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0) // Premier jour du mois suivant
	
	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s PARTITION OF ledger
		FOR VALUES FROM ('%s') TO ('%s')
	`, partitionName, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	_, err = p.pool.Exec(ctx, createSQL)
	if err != nil {
		return fmt.Errorf("failed to create partition %s: %w", partitionName, err)
	}
	
	p.log.Info().
		Str("partition", partitionName).
		Str("start_date", startDate.Format("2006-01-02")).
		Str("end_date", endDate.Format("2006-01-02")).
		Msg("Partition created")
	
	return nil
}

// EnsureCurrentPartition crée la partition pour le mois actuel
func (p *PartitionManager) EnsureCurrentPartition(ctx context.Context) error {
	now := time.Now()
	return p.EnsurePartition(ctx, now.Year(), int(now.Month()))
}

// EnsureNextPartition crée la partition pour le mois suivant
func (p *PartitionManager) EnsureNextPartition(ctx context.Context) error {
	nextMonth := time.Now().AddDate(0, 1, 0)
	return p.EnsurePartition(ctx, nextMonth.Year(), int(nextMonth.Month()))
}

// MigrateExistingData migre les données existantes vers des partitions
func (p *PartitionManager) MigrateExistingData(ctx context.Context) error {
	p.log.Info().Msg("Starting ledger data migration to partitions")
	
	// Récupérer toutes les dates distinctes dans le ledger
	rows, err := p.pool.Query(ctx, `
		SELECT DISTINCT DATE(timestamp) as date
		FROM ledger
		ORDER BY date ASC
	`)
	if err != nil {
		return fmt.Errorf("failed to query distinct dates: %w", err)
	}
	defer rows.Close()
	
	var dates []time.Time
	for rows.Next() {
		var date time.Time
		if err := rows.Scan(&date); err != nil {
			return fmt.Errorf("failed to scan date: %w", err)
		}
		dates = append(dates, date)
	}
	
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating dates: %w", err)
	}
	
	// Créer les partitions nécessaires
	partitionsCreated := 0
	for _, date := range dates {
		year := date.Year()
		month := int(date.Month())
		
		if err := p.EnsurePartition(ctx, year, month); err != nil {
			return fmt.Errorf("failed to create partition for %d-%02d: %w", year, month, err)
		}
		partitionsCreated++
	}
	
	p.log.Info().
		Int("partitions_created", partitionsCreated).
		Msg("Ledger data migration completed")
	
	return nil
}

// getPartitionName retourne le nom de la partition pour un mois donné
func (p *PartitionManager) getPartitionName(year int, month int) string {
	return fmt.Sprintf("ledger_%d_%02d", year, month)
}

// GetPartitionInfo retourne les informations sur les partitions existantes
func (p *PartitionManager) GetPartitionInfo(ctx context.Context) ([]PartitionInfo, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT 
			relname as partition_name,
			pg_size_pretty(pg_total_relation_size(relname::regclass)) as size,
			(SELECT COUNT(*) FROM pg_inherits WHERE inhrelid = rel.oid) as is_partition
		FROM pg_class rel
		WHERE relname LIKE 'ledger_%' 
		AND relkind = 'r'
		ORDER BY relname ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query partitions: %w", err)
	}
	defer rows.Close()
	
	var partitions []PartitionInfo
	for rows.Next() {
		var info PartitionInfo
		if err := rows.Scan(&info.Name, &info.Size, &info.IsPartition); err != nil {
			return nil, fmt.Errorf("failed to scan partition info: %w", err)
		}
		partitions = append(partitions, info)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating partitions: %w", err)
	}
	
	return partitions, nil
}

// PartitionInfo contient les informations sur une partition
type PartitionInfo struct {
	Name        string
	Size        string
	IsPartition bool
}

// SetupPartitionedLedger configure le ledger avec partitionnement
// Cette fonction doit être appelée une fois pour initialiser le partitionnement
func SetupPartitionedLedger(ctx context.Context, pool *pgxpool.Pool, log zerolog.Logger) error {
	manager := NewPartitionManager(pool, log)
	
	// 1. Vérifier si le ledger est déjà partitionné
	var isPartitioned bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_inherits 
			WHERE inhparent = 'ledger'::regclass
		)
	`).Scan(&isPartitioned)
	
	if err != nil {
		return fmt.Errorf("failed to check partition status: %w", err)
	}
	
	if isPartitioned {
		log.Info().Msg("Ledger is already partitioned")
		// Migrer les données existantes si nécessaire
		return manager.MigrateExistingData(ctx)
	}
	
	// 2. Convertir la table ledger en table partitionnée
	// Note: Cette opération nécessite que la table soit vide ou utilise ALTER TABLE
	log.Info().Msg("Converting ledger to partitioned table")
	
	// Créer une nouvelle table partitionnée
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS ledger_new (
			id SERIAL,
			document_id UUID NOT NULL,
			hash TEXT NOT NULL,
			previous_hash TEXT,
			timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
			evidence_jws TEXT,
			PRIMARY KEY (id, timestamp),
			UNIQUE (document_id, hash)
		) PARTITION BY RANGE (timestamp)
	`)
	if err != nil {
		return fmt.Errorf("failed to create partitioned ledger: %w", err)
	}
	
	// Migrer les données (si la table existe déjà)
	var tableExists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'ledger'
		)
	`).Scan(&tableExists)
	
	if err == nil && tableExists {
		// Migrer les données vers la nouvelle table partitionnée
		log.Info().Msg("Migrating existing ledger data")
		
		// Créer les partitions pour les mois existants
		if err := manager.MigrateExistingData(ctx); err != nil {
			return fmt.Errorf("failed to migrate data: %w", err)
		}
		
		// Copier les données (dans une transaction)
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback(ctx)
		
		_, err = tx.Exec(ctx, `
			INSERT INTO ledger_new (document_id, hash, previous_hash, timestamp, evidence_jws)
			SELECT document_id, hash, previous_hash, timestamp, evidence_jws
			FROM ledger
		`)
		if err != nil {
			return fmt.Errorf("failed to copy data: %w", err)
		}
		
		// Remplacer l'ancienne table
		_, err = tx.Exec(ctx, `
			DROP TABLE IF EXISTS ledger CASCADE;
			ALTER TABLE ledger_new RENAME TO ledger;
		`)
		if err != nil {
			return fmt.Errorf("failed to replace table: %w", err)
		}
		
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration: %w", err)
		}
		
		log.Info().Msg("Ledger migration completed")
	} else {
		// Pas de données existantes, renommer directement
		_, err = pool.Exec(ctx, `ALTER TABLE ledger_new RENAME TO ledger`)
		if err != nil {
			return fmt.Errorf("failed to rename table: %w", err)
		}
	}
	
	// 3. Créer les partitions pour le mois actuel et suivant
	if err := manager.EnsureCurrentPartition(ctx); err != nil {
		return fmt.Errorf("failed to create current partition: %w", err)
	}
	
	if err := manager.EnsureNextPartition(ctx); err != nil {
		return fmt.Errorf("failed to create next partition: %w", err)
	}
	
	log.Info().Msg("Partitioned ledger setup completed")
	return nil
}

// AppendLedgerPartitioned ajoute une entrée au ledger partitionné
// Cette fonction est une version optimisée de AppendLedger pour les tables partitionnées
func AppendLedgerPartitioned(ctx context.Context, tx pgx.Tx, docID uuid.UUID, shaHex, jws string) (string, error) {
	// Utiliser la même logique que AppendLedger mais optimisée pour les partitions
	// La partition sera automatiquement sélectionnée par PostgreSQL basée sur timestamp
	
	// 1. Récupérer le previous_hash avec verrou exclusif
	// Pour les tables partitionnées, on peut optimiser en cherchant dans la partition actuelle
	var previousHash *string
	err := tx.QueryRow(ctx, `
		SELECT hash FROM ledger 
		WHERE timestamp >= DATE_TRUNC('month', NOW())
		ORDER BY timestamp DESC, id DESC 
		LIMIT 1 
		FOR UPDATE
	`).Scan(&previousHash)
	
	// Si pas trouvé dans le mois actuel, chercher dans toutes les partitions
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, `
			SELECT hash FROM ledger 
			ORDER BY timestamp DESC, id DESC 
			LIMIT 1 
			FOR UPDATE
		`).Scan(&previousHash)
	}
	
	// 2. Calculer le nouveau hash (même logique que AppendLedger)
	var newHash string
	if err == pgx.ErrNoRows || previousHash == nil {
		hash := sha256.Sum256([]byte(shaHex))
		newHash = hex.EncodeToString(hash[:])
	} else if err != nil {
		return "", fmt.Errorf("failed to get previous hash: %w", err)
	} else {
		combined := *previousHash + shaHex
		hash := sha256.Sum256([]byte(combined))
		newHash = hex.EncodeToString(hash[:])
	}
	
	// 3. Insérer dans le ledger (PostgreSQL sélectionnera automatiquement la bonne partition)
	_, err = tx.Exec(ctx, `
		INSERT INTO ledger (document_id, hash, previous_hash, evidence_jws, timestamp)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (document_id, hash) DO NOTHING
	`, docID, newHash, previousHash, jws)
	
	if err != nil {
		return "", fmt.Errorf("failed to insert into ledger: %w", err)
	}
	
	return newHash, nil
}

