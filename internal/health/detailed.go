package health

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/storage"
)

// Status représente le statut d'un composant
type Status string

const (
	StatusOK    Status = "ok"
	StatusWarn  Status = "warn"
	StatusFail  Status = "fail"
)

// ComponentHealth représente l'état de santé d'un composant
type ComponentHealth struct {
	Status  Status  `json:"status"`
	Message string  `json:"message,omitempty"`
	Latency *string `json:"latency,omitempty"` // en millisecondes
}

// DetailedHealth représente l'état de santé complet du système
type DetailedHealth struct {
	Status    Status                    `json:"status"`    // Statut global (ok si tous ok, warn si au moins un warn, fail si au moins un fail)
	Timestamp time.Time                 `json:"timestamp"`
	Database  ComponentHealth           `json:"database"`
	Storage   ComponentHealth           `json:"storage"`
	JWS       ComponentHealth           `json:"jws"`
	Ledger    ComponentHealth           `json:"ledger"`
}

// CheckDetailedHealth vérifie l'état de santé de tous les composants
func CheckDetailedHealth(
	ctx context.Context,
	db *storage.DB,
	storageDir string,
	jwsService *crypto.Service,
) DetailedHealth {
	health := DetailedHealth{
		Timestamp: time.Now(),
	}

	// Vérifier chaque composant avec timeout
	timeout := 5 * time.Second
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 1. Vérification Database
	health.Database = CheckDatabase(checkCtx, db)

	// 2. Vérification Storage
	health.Storage = CheckStorage(storageDir)

	// 3. Vérification JWS
	health.JWS = CheckJWS(jwsService)

	// 4. Vérification Ledger
	health.Ledger = CheckLedger(checkCtx, db)

	// Déterminer le statut global
	health.Status = DetermineGlobalStatus(health.Database, health.Storage, health.JWS, health.Ledger)

	return health
}

// CheckDatabase vérifie l'état de la base de données
// Exportée pour les tests unitaires
func CheckDatabase(ctx context.Context, db *storage.DB) ComponentHealth {
	start := time.Now()

	if db == nil || db.Pool == nil {
		return ComponentHealth{
			Status:  StatusFail,
			Message: "Database not configured",
		}
	}

	// Ping de la base de données
	if err := db.Pool.Ping(ctx); err != nil {
		latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
		return ComponentHealth{
			Status:  StatusFail,
			Message: fmt.Sprintf("Database ping failed: %v", err),
			Latency: &latency,
		}
	}

	// Test de requête simple
	var result int
	err := db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
		return ComponentHealth{
			Status:  StatusFail,
			Message: fmt.Sprintf("Database query failed: %v", err),
			Latency: &latency,
		}
	}

	latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
	return ComponentHealth{
		Status:  StatusOK,
		Message: "Database connection healthy",
		Latency: &latency,
	}
}

// CheckStorage vérifie l'état du stockage de fichiers
// Exportée pour les tests unitaires
func CheckStorage(storageDir string) ComponentHealth {
	start := time.Now()

	if storageDir == "" {
		return ComponentHealth{
			Status:  StatusFail,
			Message: "Storage directory not configured",
		}
	}

	// Vérifier que le répertoire existe
	info, err := os.Stat(storageDir)
	if err != nil {
		if os.IsNotExist(err) {
			return ComponentHealth{
				Status:  StatusFail,
				Message: fmt.Sprintf("Storage directory does not exist: %s", storageDir),
			}
		}
		latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
		return ComponentHealth{
			Status:  StatusFail,
			Message: fmt.Sprintf("Storage directory access error: %v", err),
			Latency: &latency,
		}
	}

	// Vérifier que c'est un répertoire
	if !info.IsDir() {
		latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
		return ComponentHealth{
			Status:  StatusFail,
			Message: fmt.Sprintf("Storage path is not a directory: %s", storageDir),
			Latency: &latency,
		}
	}

	// Vérifier les permissions (doit être lisible et inscriptible)
	if info.Mode().Perm()&0200 == 0 {
		latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
		return ComponentHealth{
			Status:  StatusWarn,
			Message: fmt.Sprintf("Storage directory not writable: %s", storageDir),
			Latency: &latency,
		}
	}

	latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
	return ComponentHealth{
		Status:  StatusOK,
		Message: fmt.Sprintf("Storage directory accessible: %s", storageDir),
		Latency: &latency,
	}
}

// CheckJWS vérifie l'état du service JWS
// Exportée pour les tests unitaires
func CheckJWS(jwsService *crypto.Service) ComponentHealth {
	start := time.Now()

	if jwsService == nil {
		return ComponentHealth{
			Status:  StatusWarn,
			Message: "JWS service not configured (degraded mode)",
		}
	}

	// Vérifier que le service peut générer un JWKS (test de disponibilité)
	_, err := jwsService.CurrentJWKS()
	if err != nil {
		latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
		return ComponentHealth{
			Status:  StatusFail,
			Message: fmt.Sprintf("JWS service error: %v", err),
			Latency: &latency,
		}
	}

	latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
	return ComponentHealth{
		Status:  StatusOK,
		Message: "JWS service operational",
		Latency: &latency,
	}
}

// CheckLedger vérifie l'état de la table ledger
// Exportée pour les tests unitaires
func CheckLedger(ctx context.Context, db *storage.DB) ComponentHealth {
	start := time.Now()

	if db == nil || db.Pool == nil {
		return ComponentHealth{
			Status:  StatusFail,
			Message: "Database not configured",
		}
	}

	// Vérifier que la table ledger existe
	var tableExists bool
	err := db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'ledger'
		)
	`).Scan(&tableExists)

	if err != nil {
		latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
		return ComponentHealth{
			Status:  StatusFail,
			Message: fmt.Sprintf("Failed to check ledger table: %v", err),
			Latency: &latency,
		}
	}

	if !tableExists {
		latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
		return ComponentHealth{
			Status:  StatusFail,
			Message: "Ledger table does not exist (migration required)",
			Latency: &latency,
		}
	}

	// Vérifier que les index critiques existent
	var indexExists bool
	err = db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM pg_indexes 
			WHERE tablename = 'ledger' 
			AND indexname = 'idx_ledger_ts_id_desc'
		)
	`).Scan(&indexExists)

	if err != nil {
		latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
		return ComponentHealth{
			Status:  StatusWarn,
			Message: fmt.Sprintf("Failed to check ledger indexes: %v", err),
			Latency: &latency,
		}
	}

	if !indexExists {
		latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
		return ComponentHealth{
			Status:  StatusWarn,
			Message: "Ledger index idx_ledger_ts_id_desc missing (performance may be degraded)",
			Latency: &latency,
		}
	}

	latency := fmt.Sprintf("%.2f", time.Since(start).Seconds()*1000)
	return ComponentHealth{
		Status:  StatusOK,
		Message: "Ledger table and indexes present",
		Latency: &latency,
	}
}

// DetermineGlobalStatus détermine le statut global basé sur les statuts des composants
// Exportée pour les tests unitaires
func DetermineGlobalStatus(components ...ComponentHealth) Status {
	hasFail := false
	hasWarn := false

	for _, comp := range components {
		switch comp.Status {
		case StatusFail:
			hasFail = true
		case StatusWarn:
			hasWarn = true
		}
	}

	if hasFail {
		return StatusFail
	}
	if hasWarn {
		return StatusWarn
	}
	return StatusOK
}

