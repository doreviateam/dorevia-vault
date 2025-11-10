package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLI_ValidateFlags teste la validation des flags
func TestCLI_ValidateFlags(t *testing.T) {
	tests := []struct {
		name    string
		period  string
		from    string
		to      string
		format  string
		wantErr bool
	}{
		{
			name:    "valid monthly",
			period:  "monthly",
			format:  "json",
			wantErr: false,
		},
		{
			name:    "valid quarterly",
			period:  "quarterly",
			format:  "csv",
			wantErr: false,
		},
		{
			name:    "valid custom",
			period:  "custom",
			from:    "2025-01-01",
			to:      "2025-01-31",
			format:  "pdf",
			wantErr: false,
		},
		{
			name:    "missing period",
			period:  "",
			format:  "json",
			wantErr: true,
		},
		{
			name:    "invalid period",
			period:  "invalid",
			format:  "json",
			wantErr: true,
		},
		{
			name:    "custom missing from",
			period:  "custom",
			to:      "2025-01-31",
			format:  "json",
			wantErr: true,
		},
		{
			name:    "custom missing to",
			period:  "custom",
			from:    "2025-01-01",
			format:  "json",
			wantErr: true,
		},
		{
			name:    "invalid date format",
			period:  "custom",
			from:    "2025/01/01",
			to:      "2025-01-31",
			format:  "json",
			wantErr: true,
		},
		{
			name:    "invalid format",
			period:  "monthly",
			format:  "xml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFlagsCLI(tt.period, tt.from, tt.to, tt.format)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// validateFlagsCLI est une copie de la fonction de validation pour les tests
func validateFlagsCLI(periodType, fromDate, toDate, format string) error {
	if periodType == "" {
		return fmt.Errorf("--period is required")
	}

	if periodType != "monthly" && periodType != "quarterly" && periodType != "custom" {
		return fmt.Errorf("invalid period type: %s (must be monthly, quarterly, or custom)", periodType)
	}

	if periodType == "custom" {
		if fromDate == "" {
			return fmt.Errorf("--from is required for custom period")
		}
		if toDate == "" {
			return fmt.Errorf("--to is required for custom period")
		}

		if _, err := time.Parse("2006-01-02", fromDate); err != nil {
			return fmt.Errorf("invalid --from date format: %s (must be YYYY-MM-DD)", fromDate)
		}
		if _, err := time.Parse("2006-01-02", toDate); err != nil {
			return fmt.Errorf("invalid --to date format: %s (must be YYYY-MM-DD)", toDate)
		}
	}

	if format != "json" && format != "csv" && format != "pdf" {
		return fmt.Errorf("invalid format: %s (must be json, csv, or pdf)", format)
	}

	return nil
}

// TestCLI_GenerateMonthly teste la génération mensuelle via CLI
func TestCLI_GenerateMonthly(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())

	// Générer rapport mensuel
	report, err := generator.GenerateMonthly(2025, 1)
	require.NoError(t, err)

	// Exporter en JSON
	outputPath := filepath.Join(tmpDir, "report-monthly.json")
	err = generator.ExportJSON(report, outputPath)
	require.NoError(t, err)

	// Vérifier le fichier
	_, err = os.Stat(outputPath)
	require.NoError(t, err)

	// Vérifier le contenu JSON
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var decodedReport audit.AuditReport
	err = json.Unmarshal(data, &decodedReport)
	require.NoError(t, err)
	assert.Equal(t, audit.PeriodTypeMonthly, decodedReport.Period.Type)
	assert.Equal(t, "2025-01-01", decodedReport.Period.StartDate)
}

// TestCLI_GenerateQuarterly teste la génération trimestrielle via CLI
func TestCLI_GenerateQuarterly(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())

	// Générer rapport trimestriel
	report, err := generator.GenerateQuarterly(2025, 1)
	require.NoError(t, err)

	// Exporter en CSV
	outputPath := filepath.Join(tmpDir, "report-quarterly.csv")
	err = generator.ExportCSV(report, outputPath)
	require.NoError(t, err)

	// Vérifier le fichier
	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

// TestCLI_GenerateCustom teste la génération personnalisée via CLI
func TestCLI_GenerateCustom(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())

	// Générer rapport personnalisé
	report, err := generator.Generate(audit.PeriodTypeCustom, "2025-01-15", "2025-01-31")
	require.NoError(t, err)

	// Exporter en PDF
	outputPath := filepath.Join(tmpDir, "report-custom.pdf")
	pdfGen := audit.NewPDFGenerator(nil, zerolog.Nop())
	err = pdfGen.Generate(report, outputPath)
	require.NoError(t, err)

	// Vérifier le fichier
	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

// TestCLI_ExportFormats teste tous les formats d'export
func TestCLI_ExportFormats(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())

	today := time.Now().UTC().Format("2006-01-02")
	report, err := generator.Generate(audit.PeriodTypeCustom, today, today)
	require.NoError(t, err)

	// Test JSON
	jsonPath := filepath.Join(tmpDir, "report.json")
	err = generator.ExportJSON(report, jsonPath)
	require.NoError(t, err)
	_, err = os.Stat(jsonPath)
	require.NoError(t, err)

	// Test CSV
	csvPath := filepath.Join(tmpDir, "report.csv")
	err = generator.ExportCSV(report, csvPath)
	require.NoError(t, err)
	_, err = os.Stat(csvPath)
	require.NoError(t, err)

	// Test PDF
	pdfPath := filepath.Join(tmpDir, "report.pdf")
	pdfGen := audit.NewPDFGenerator(nil, zerolog.Nop())
	err = pdfGen.Generate(report, pdfPath)
	require.NoError(t, err)
	_, err = os.Stat(pdfPath)
	require.NoError(t, err)
}

// TestCLI_SignReport teste la signature JWS
func TestCLI_SignReport(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)

	// Tester sans JWS service (doit fonctionner)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())
	today := time.Now().UTC().Format("2006-01-02")
	report, err := generator.Generate(audit.PeriodTypeCustom, today, today)
	require.NoError(t, err)

	// Tenter de signer sans JWS service (doit échouer)
	err = generator.Sign(report)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWS service not available")
}

// TestCLI_WithDatabase teste avec base de données
func TestCLI_WithDatabase(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping database test")
	}

	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	log := zerolog.Nop()
	db, err := storage.NewDB(ctx, databaseURL, &log)
	require.NoError(t, err)
	defer db.Close()

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, db, nil, zerolog.Nop())

	today := time.Now().UTC().Format("2006-01-02")
	report, err := generator.Generate(audit.PeriodTypeCustom, today, today)
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Contains(t, report.Metadata.DataSources, "database")
}

// TestCLI_InvalidPeriod teste les périodes invalides
func TestCLI_InvalidPeriod(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())

	// Mois invalide
	_, err = generator.GenerateMonthly(2025, 13)
	assert.Error(t, err)

	// Trimestre invalide
	_, err = generator.GenerateQuarterly(2025, 5)
	assert.Error(t, err)

	// Dates invalides
	_, err = generator.Generate(audit.PeriodTypeCustom, "invalid", "2025-01-31")
	assert.Error(t, err)
}

// TestCLI_DefaultOutput teste les chemins de sortie par défaut
func TestCLI_DefaultOutput(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())

	today := time.Now().UTC().Format("2006-01-02")
	report, err := generator.Generate(audit.PeriodTypeCustom, today, today)
	require.NoError(t, err)

	// Test stdout pour JSON (ne doit pas planter)
	err = generator.ExportJSON(report, "-")
	require.NoError(t, err)

	// Test stdout pour CSV (ne doit pas planter)
	err = generator.ExportCSV(report, "-")
	require.NoError(t, err)
}

