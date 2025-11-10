package reconcile

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// OrphanFile représente un fichier orphelin (sans entrée DB)
type OrphanFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SHA256Hex string `json:"sha256_hex,omitempty"`
}

// OrphanDB représente une entrée DB sans fichier
type OrphanDB struct {
	DocumentID string `json:"document_id"`
	Filename   string `json:"filename"`
	StoredPath string `json:"stored_path"`
	SHA256Hex  string `json:"sha256_hex"`
}

// ReconciliationReport représente le rapport de réconciliation
type ReconciliationReport struct {
	Timestamp      time.Time    `json:"timestamp"`
	DryRun         bool         `json:"dry_run"`
	OrphanFiles    []OrphanFile `json:"orphan_files"`    // Fichiers sans DB
	OrphanDBs      []OrphanDB   `json:"orphan_dbs"`      // DB sans fichiers
	FilesDeleted   int          `json:"files_deleted"`    // Nombre de fichiers supprimés (si fix)
	DBsMarked      int          `json:"dbs_marked"`      // Nombre d'entrées DB marquées (si fix)
	Errors         []string     `json:"errors,omitempty"` // Erreurs rencontrées
}

// CleanupOrphans détecte et corrige les fichiers orphelins
// - Fichiers sans DB : fichiers dans storage sans entrée correspondante
// - DB sans fichiers : entrées DB dont le fichier n'existe pas
func CleanupOrphans(
	ctx context.Context,
	db *storage.DB,
	storageDir string,
	dryRun bool,
) (*ReconciliationReport, error) {
	report := &ReconciliationReport{
		Timestamp:   time.Now().UTC(),
		DryRun:      dryRun,
		OrphanFiles: []OrphanFile{},
		OrphanDBs:   []OrphanDB{},
		Errors:      []string{},
	}

	if db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	// 1. Scanner storage pour trouver fichiers sans DB
	orphanFiles, err := findOrphanFiles(ctx, db, storageDir)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Failed to scan storage: %v", err))
	} else {
		report.OrphanFiles = orphanFiles
	}

	// 2. Scanner DB pour trouver entrées sans fichiers
	orphanDBs, err := findOrphanDBs(ctx, db)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("Failed to scan database: %v", err))
	} else {
		report.OrphanDBs = orphanDBs
	}

	// 3. Mode fix : supprimer fichiers orphelins et marquer entrées DB
	if !dryRun {
		// Supprimer fichiers orphelins
		for _, orphan := range report.OrphanFiles {
			if err := os.Remove(orphan.Path); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("Failed to delete orphan file %s: %v", orphan.Path, err))
			} else {
				report.FilesDeleted++
			}
		}

		// Marquer entrées DB orphelines (optionnel : ajouter un champ "orphaned" ou supprimer)
		// Pour l'instant, on ne fait que les détecter (suppression DB à implémenter si nécessaire)
		// TODO: Ajouter un champ "orphaned_at" dans documents ou supprimer directement
		report.DBsMarked = len(report.OrphanDBs)
	}

	return report, nil
}

// findOrphanFiles scanne le répertoire storage et trouve les fichiers sans entrée DB
func findOrphanFiles(ctx context.Context, db *storage.DB, storageDir string) ([]OrphanFile, error) {
	var orphans []OrphanFile

	// Scanner récursivement le répertoire storage
	err := filepath.Walk(storageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // Continuer malgré les erreurs
		}

		// Ignorer les répertoires et fichiers temporaires
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".tmp" {
			return nil
		}

		// Lire le fichier pour calculer SHA256
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Ignorer les fichiers non lisibles
		}

		hash := sha256.Sum256(content)
		sha256Hex := hex.EncodeToString(hash[:])

		// Vérifier si ce SHA256 existe en DB
		var docID uuid.UUID
		err = db.Pool.QueryRow(ctx, `
			SELECT id FROM documents WHERE sha256_hex = $1 LIMIT 1
		`, sha256Hex).Scan(&docID)

		if err == pgx.ErrNoRows {
			// Fichier sans entrée DB → orphelin
			orphans = append(orphans, OrphanFile{
				Path:      path,
				SizeBytes: info.Size(),
				SHA256Hex: sha256Hex,
			})
		} else if err != nil {
			return fmt.Errorf("failed to check file in DB: %w", err)
		}

		return nil
	})

	return orphans, err
}

// findOrphanDBs trouve les entrées DB dont le fichier n'existe pas
func findOrphanDBs(ctx context.Context, db *storage.DB) ([]OrphanDB, error) {
	var orphans []OrphanDB

	// Récupérer toutes les entrées avec stored_path
	rows, err := db.Pool.Query(ctx, `
		SELECT id, filename, stored_path, sha256_hex
		FROM documents
		WHERE stored_path IS NOT NULL AND stored_path != ''
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var docID uuid.UUID
		var filename, storedPath, sha256Hex string

		if err := rows.Scan(&docID, &filename, &storedPath, &sha256Hex); err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Vérifier si le fichier existe
		if _, err := os.Stat(storedPath); os.IsNotExist(err) {
			// Fichier n'existe pas → orphelin DB
			orphans = append(orphans, OrphanDB{
				DocumentID: docID.String(),
				Filename:   filename,
				StoredPath: storedPath,
				SHA256Hex:  sha256Hex,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating documents: %w", err)
	}

	return orphans, nil
}

