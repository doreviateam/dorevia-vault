package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/doreviateam/dorevia-vault/internal/config"
	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/doreviateam/dorevia-vault/pkg/logger"
)

var (
	version = "v1.2.0-rc1"
	commit  = "dev"
)

func main() {
	// Parse flags
	periodType := flag.String("period", "", "Type de période (monthly, quarterly, custom) [required]")
	year := flag.Int("year", time.Now().Year(), "Année (pour monthly/quarterly) [default: année actuelle]")
	month := flag.Int("month", int(time.Now().Month()), "Mois 1-12 (pour monthly) [default: mois actuel]")
	quarter := flag.Int("quarter", getCurrentQuarter(), "Trimestre 1-4 (pour quarterly) [default: trimestre actuel]")
	fromDate := flag.String("from", "", "Date début YYYY-MM-DD (pour custom) [required si custom]")
	toDate := flag.String("to", "", "Date fin YYYY-MM-DD (pour custom) [required si custom]")
	format := flag.String("format", "json", "Format d'export (json, csv, pdf) [default: json]")
	outputPath := flag.String("output", "", "Chemin fichier de sortie [default: stdout pour json/csv, report-YYYY-MM-DD.pdf pour pdf]")
	sign := flag.Bool("sign", false, "Signer le rapport avec JWS [default: false]")
	jwsKeyPath := flag.String("jws-key-path", "", "Chemin clé privée JWS [default: JWS_PRIVATE_KEY_PATH env]")
	auditDir := flag.String("audit-dir", "", "Répertoire audit [default: AUDIT_DIR env]")
	databaseURL := flag.String("database-url", "", "URL base de données [default: DATABASE_URL env]")
	verbose := flag.Bool("verbose", false, "Mode verbeux")
	help := flag.Bool("help", false, "Afficher l'aide")

	flag.Parse()

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Validation
	if err := validateFlags(periodType, fromDate, toDate, format); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}

	// Configuration
	cfg := loadConfig(*auditDir, *databaseURL, *jwsKeyPath)
	logLevel := cfg.LogLevel
	if *verbose {
		logLevel = "debug"
	}
	log := logger.New(logLevel)

	// Initialisation modules
	auditLogger, err := audit.NewLogger(audit.Config{
		AuditDir:      cfg.AuditDir,
		MaxBuffer:     1000,
		FlushInterval: 10 * time.Second,
		Logger:        *log,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize audit logger")
	}
	defer auditLogger.Close()

	exporter := audit.NewExporter(auditLogger)

	var db *storage.DB
	if cfg.DatabaseURL != "" {
		ctx := context.Background()
		db, err = storage.NewDB(ctx, cfg.DatabaseURL, log)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to connect to database, continuing without DB stats")
		} else {
			defer db.Close()
		}
	}

	var jwsService *crypto.Service
	if *sign || cfg.JWSPrivateKeyPath != "" {
		jwsService, err = crypto.NewService(cfg.JWSPrivateKeyPath, cfg.JWSPublicKeyPath, cfg.JWSKID)
		if err != nil {
			if *sign {
				log.Fatal().Err(err).Msg("JWS service required for signing but initialization failed")
			}
			log.Warn().Err(err).Msg("JWS service unavailable, signature disabled")
		}
	}

	// Génération rapport
	generator := audit.NewReportGenerator(auditLogger, exporter, db, jwsService, *log)

	var report *audit.AuditReport

	switch *periodType {
	case "monthly":
		report, err = generator.GenerateMonthly(*year, *month)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate monthly report")
		}
	case "quarterly":
		report, err = generator.GenerateQuarterly(*year, *quarter)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate quarterly report")
		}
	case "custom":
		report, err = generator.Generate(audit.PeriodTypeCustom, *fromDate, *toDate)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to generate custom report")
		}
	default:
		log.Fatal().Str("period", *periodType).Msg("Invalid period type")
	}

	// Signature (si demandée)
	if *sign {
		if jwsService == nil {
			log.Fatal().Msg("JWS service not available, cannot sign report")
		}
		if err := generator.Sign(report); err != nil {
			log.Fatal().Err(err).Msg("Failed to sign report")
		}
		log.Info().Msg("Report signed with JWS")
	}

	// Export
	switch *format {
	case "json":
		if *outputPath == "" {
			*outputPath = "-" // stdout
		}
		if err := generator.ExportJSON(report, *outputPath); err != nil {
			log.Fatal().Err(err).Msg("Failed to export JSON")
		}
		if *outputPath != "-" {
			log.Info().Str("path", *outputPath).Msg("Report exported to JSON")
		}

	case "csv":
		if *outputPath == "" {
			*outputPath = "-" // stdout
		}
		if err := generator.ExportCSV(report, *outputPath); err != nil {
			log.Fatal().Err(err).Msg("Failed to export CSV")
		}
		if *outputPath != "-" {
			log.Info().Str("path", *outputPath).Msg("Report exported to CSV")
		}

	case "pdf":
		if *outputPath == "" {
			*outputPath = fmt.Sprintf("report-%s.pdf", report.Period.StartDate)
		}
		pdfGen := audit.NewPDFGenerator(jwsService, *log)
		if err := pdfGen.Generate(report, *outputPath); err != nil {
			log.Fatal().Err(err).Msg("Failed to export PDF")
		}
		log.Info().Str("path", *outputPath).Msg("Report exported to PDF")

	default:
		log.Fatal().Str("format", *format).Msg("Invalid format")
	}
}

// validateFlags valide les flags
func validateFlags(periodType *string, fromDate *string, toDate *string, format *string) error {
	if *periodType == "" {
		return fmt.Errorf("--period is required")
	}

	if *periodType != "monthly" && *periodType != "quarterly" && *periodType != "custom" {
		return fmt.Errorf("invalid period type: %s (must be monthly, quarterly, or custom)", *periodType)
	}

	if *periodType == "custom" {
		if *fromDate == "" {
			return fmt.Errorf("--from is required for custom period")
		}
		if *toDate == "" {
			return fmt.Errorf("--to is required for custom period")
		}

		// Valider format des dates
		if _, err := time.Parse("2006-01-02", *fromDate); err != nil {
			return fmt.Errorf("invalid --from date format: %s (must be YYYY-MM-DD)", *fromDate)
		}
		if _, err := time.Parse("2006-01-02", *toDate); err != nil {
			return fmt.Errorf("invalid --to date format: %s (must be YYYY-MM-DD)", *toDate)
		}
	}

	if *format != "json" && *format != "csv" && *format != "pdf" {
		return fmt.Errorf("invalid format: %s (must be json, csv, or pdf)", *format)
	}

	return nil
}

// loadConfig charge la configuration
func loadConfig(auditDir, databaseURL, jwsKeyPath string) config.Config {
	cfg := config.LoadOrDie()

	// Override avec les flags si fournis
	if auditDir != "" {
		cfg.AuditDir = auditDir
	}
	if databaseURL != "" {
		cfg.DatabaseURL = databaseURL
	}
	if jwsKeyPath != "" {
		cfg.JWSPrivateKeyPath = jwsKeyPath
	}

	return cfg
}

// getCurrentQuarter retourne le trimestre actuel (1-4)
func getCurrentQuarter() int {
	month := int(time.Now().Month())
	return (month-1)/3 + 1
}

// printHelp affiche l'aide
func printHelp() {
	fmt.Printf(`Dorevia Vault - Audit Report Generator
Version: %s (commit: %s)

Usage: ./bin/audit [OPTIONS]

Options:
  --period TYPE          Type de période (monthly, quarterly, custom) [required]
  --year YEAR            Année (pour monthly/quarterly) [default: année actuelle]
  --month MONTH          Mois 1-12 (pour monthly) [default: mois actuel]
  --quarter QUARTER      Trimestre 1-4 (pour quarterly) [default: trimestre actuel]
  --from DATE            Date début YYYY-MM-DD (pour custom) [required si custom]
  --to DATE              Date fin YYYY-MM-DD (pour custom) [required si custom]
  --format FORMAT        Format d'export (json, csv, pdf) [default: json]
  --output PATH          Chemin fichier de sortie [default: stdout pour json/csv, report-YYYY-MM-DD.pdf pour pdf]
  --sign                 Signer le rapport avec JWS [default: false]
  --jws-key-path PATH    Chemin clé privée JWS [default: JWS_PRIVATE_KEY_PATH env]
  --audit-dir PATH       Répertoire audit [default: AUDIT_DIR env]
  --database-url URL      URL base de données [default: DATABASE_URL env]
  --verbose              Mode verbeux
  --help                 Afficher cette aide

Exemples:
  # Rapport mensuel JSON (Janvier 2025)
  ./bin/audit --period monthly --year 2025 --month 1 --format json --output report-2025-01.json

  # Rapport trimestriel PDF signé (Q1 2025)
  ./bin/audit --period quarterly --year 2025 --quarter 1 --format pdf --sign --output report-Q1-2025.pdf

  # Rapport personnalisé CSV (15 jours)
  ./bin/audit --period custom --from 2025-01-15 --to 2025-01-31 --format csv --output report-custom.csv

  # Rapport mensuel JSON signé (mois actuel)
  ./bin/audit --period monthly --format json --sign --output report-current.json

`, version, commit)
}

