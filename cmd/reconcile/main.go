package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/config"
	"github.com/doreviateam/dorevia-vault/internal/reconcile"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/doreviateam/dorevia-vault/pkg/logger"
)

func main() {
	// Flags
	dryRun := flag.Bool("dry-run", false, "Mode dry-run : détecte les orphelins sans les supprimer")
	fix := flag.Bool("fix", false, "Mode fix : supprime les fichiers orphelins et marque les entrées DB")
	output := flag.String("output", "", "Fichier de sortie pour le rapport JSON (optionnel)")
	flag.Parse()

	// Validation des flags
	if *dryRun && *fix {
		fmt.Fprintf(os.Stderr, "Error: --dry-run and --fix cannot be used together\n")
		os.Exit(1)
	}

	// Mode par défaut : dry-run si aucun flag
	dryRunMode := true
	if *fix {
		dryRunMode = false
	}

	// Charger la configuration
	cfg := config.LoadOrDie()

	// Initialiser le logger
	log := logger.New(cfg.LogLevel)

	// Vérifier que DATABASE_URL est configuré
	if cfg.DatabaseURL == "" {
		log.Fatal().Msg("DATABASE_URL not configured")
	}

	// Vérifier que STORAGE_DIR est configuré
	if cfg.StorageDir == "" {
		log.Fatal().Msg("STORAGE_DIR not configured")
	}

	// Initialiser la connexion à la base de données
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := storage.NewDB(ctx, cfg.DatabaseURL, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	log.Info().
		Bool("dry_run", dryRunMode).
		Str("storage_dir", cfg.StorageDir).
		Msg("Starting reconciliation")

	// Exécuter la réconciliation
	report, err := reconcile.CleanupOrphans(ctx, db, cfg.StorageDir, dryRunMode)
	if err != nil {
		log.Fatal().Err(err).Msg("Reconciliation failed")
	}

	// Afficher le rapport
	fmt.Printf("\n=== Rapport de Réconciliation ===\n\n")
	fmt.Printf("Timestamp: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Printf("Mode: %s\n", map[bool]string{true: "DRY-RUN", false: "FIX"}[report.DryRun])
	fmt.Printf("\nFichiers orphelins (sans DB): %d\n", len(report.OrphanFiles))
	if len(report.OrphanFiles) > 0 {
		for i, orphan := range report.OrphanFiles {
			if i < 10 { // Afficher les 10 premiers
				fmt.Printf("  - %s (SHA256: %s, Size: %d bytes)\n", orphan.Path, orphan.SHA256Hex, orphan.SizeBytes)
			} else if i == 10 {
				fmt.Printf("  ... et %d autres\n", len(report.OrphanFiles)-10)
				break
			}
			}
	}

	fmt.Printf("\nEntrées DB orphelines (sans fichier): %d\n", len(report.OrphanDBs))
	if len(report.OrphanDBs) > 0 {
		for i, orphan := range report.OrphanDBs {
			if i < 10 { // Afficher les 10 premiers
				fmt.Printf("  - Document ID: %s, Path: %s\n", orphan.DocumentID, orphan.StoredPath)
			} else if i == 10 {
				fmt.Printf("  ... et %d autres\n", len(report.OrphanDBs)-10)
				break
			}
		}
	}

	if !dryRunMode {
		fmt.Printf("\nActions effectuées:\n")
		fmt.Printf("  - Fichiers supprimés: %d\n", report.FilesDeleted)
		fmt.Printf("  - Entrées DB marquées: %d\n", report.DBsMarked)
	}

	if len(report.Errors) > 0 {
		fmt.Printf("\nErreurs rencontrées: %d\n", len(report.Errors))
		for _, errMsg := range report.Errors {
			fmt.Printf("  - %s\n", errMsg)
		}
	}

	fmt.Printf("\n")

	// Exporter le rapport JSON si demandé
	if *output != "" {
		reportJSON, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal report to JSON")
			os.Exit(1)
		}

		if err := os.WriteFile(*output, reportJSON, 0644); err != nil {
			log.Error().Err(err).Str("output", *output).Msg("Failed to write report file")
			os.Exit(1)
		}

		log.Info().Str("output", *output).Msg("Report exported to JSON file")
	}

	// Code de sortie
	if len(report.OrphanFiles) > 0 || len(report.OrphanDBs) > 0 {
		if dryRunMode {
			os.Exit(1) // Code d'erreur si orphelins détectés en dry-run
		} else {
			os.Exit(0) // Succès si fix appliqué
		}
	} else {
		os.Exit(0) // Aucun orphelin détecté
	}
}

