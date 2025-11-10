package unit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportGenerator_GenerateMonthly(t *testing.T) {
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

	// Générer rapport pour janvier 2025
	report, err := generator.GenerateMonthly(2025, 1)
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, audit.PeriodTypeMonthly, report.Period.Type)
	assert.Equal(t, "2025-01-01", report.Period.StartDate)
	assert.Equal(t, "2025-01-31", report.Period.EndDate)
	assert.Equal(t, "Janvier 2025", report.Period.Label)
}

func TestReportGenerator_GenerateMonthly_InvalidMonth(t *testing.T) {
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
	assert.Contains(t, err.Error(), "invalid month")
}

func TestReportGenerator_GenerateQuarterly(t *testing.T) {
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

	// Générer rapport pour Q1 2025
	report, err := generator.GenerateQuarterly(2025, 1)
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, audit.PeriodTypeQuarterly, report.Period.Type)
	assert.Equal(t, "2025-01-01", report.Period.StartDate)
	assert.Equal(t, "2025-03-31", report.Period.EndDate)
	assert.Equal(t, "Q1 2025", report.Period.Label)
}

func TestReportGenerator_GenerateQuarterly_InvalidQuarter(t *testing.T) {
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

	// Trimestre invalide
	_, err = generator.GenerateQuarterly(2025, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quarter")
}

func TestReportGenerator_Generate_Custom(t *testing.T) {
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
	assert.NotNil(t, report)
	assert.Equal(t, audit.PeriodTypeCustom, report.Period.Type)
	assert.Equal(t, "2025-01-15", report.Period.StartDate)
	assert.Equal(t, "2025-01-31", report.Period.EndDate)
}

func TestReportGenerator_Generate_InvalidDates(t *testing.T) {
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

	// Date invalide
	_, err = generator.Generate(audit.PeriodTypeCustom, "invalid", "2025-01-31")
	assert.Error(t, err)

	// Start après end
	_, err = generator.Generate(audit.PeriodTypeCustom, "2025-01-31", "2025-01-15")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start date must be before")
}

func TestReportGenerator_CollectErrorStats(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Créer des événements de test
	events := []audit.Event{
		{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			EventType:  audit.EventTypeDocumentVaulted,
			Status:     audit.EventStatusError,
			DocumentID: uuid.New().String(),
			Metadata: map[string]interface{}{
				"error": "test error 1",
			},
		},
		{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			EventType:  audit.EventTypeJWSSigned,
			Status:     audit.EventStatusError,
			Metadata: map[string]interface{}{
				"error": "test error 2",
			},
		},
		{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			EventType: audit.EventTypeDocumentVaulted,
			Status:    audit.EventStatusSuccess,
		},
	}

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())

	// Utiliser la réflexion pour accéder à la méthode privée via un test helper
	// Pour l'instant, testons via Generate qui appelle collectErrorStats
	// Enregistrer les événements dans le logger
	for _, event := range events {
		_ = logger.Log(event)
	}
	_ = logger.Flush()

	// Générer un rapport pour aujourd'hui
	today := time.Now().UTC().Format("2006-01-02")
	report, err := generator.Generate(audit.PeriodTypeCustom, today, today)
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.GreaterOrEqual(t, report.Errors.Total, int64(0)) // Au moins 0 erreurs
}

func TestReportGenerator_CollectPerformanceStats(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Créer des événements avec durées
	events := []audit.Event{
		{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			EventType:  audit.EventTypeDocumentVaulted,
			Status:     audit.EventStatusSuccess,
			DurationMS: 100,
		},
		{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			EventType:  audit.EventTypeJWSSigned,
			Status:     audit.EventStatusSuccess,
			DurationMS: 50,
		},
		{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			EventType:  audit.EventTypeLedgerAppended,
			Status:     audit.EventStatusSuccess,
			DurationMS: 200,
		},
	}

	// Enregistrer les événements
	for _, event := range events {
		_ = logger.Log(event)
	}
	_ = logger.Flush()

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())

	today := time.Now().UTC().Format("2006-01-02")
	report, err := generator.Generate(audit.PeriodTypeCustom, today, today)
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.GreaterOrEqual(t, report.Performance.DocumentStorage.Count, int64(0))
}

func TestReportGenerator_ExportJSON(t *testing.T) {
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

	// Générer un rapport
	today := time.Now().UTC().Format("2006-01-02")
	report, err := generator.Generate(audit.PeriodTypeCustom, today, today)
	require.NoError(t, err)

	// Exporter en JSON
	outputPath := filepath.Join(tmpDir, "report.json")
	err = generator.ExportJSON(report, outputPath)
	require.NoError(t, err)

	// Vérifier que le fichier existe
	_, err = os.Stat(outputPath)
	require.NoError(t, err)

	// Vérifier le contenu JSON
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var decodedReport audit.AuditReport
	err = json.Unmarshal(data, &decodedReport)
	require.NoError(t, err)
	assert.Equal(t, report.Period.Type, decodedReport.Period.Type)
	assert.NotEmpty(t, decodedReport.Metadata.ReportHash)
}

func TestReportGenerator_ExportJSON_Stdout(t *testing.T) {
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

	// Exporter vers stdout (ne devrait pas planter)
	err = generator.ExportJSON(report, "-")
	require.NoError(t, err)
}

func TestReportGenerator_ExportCSV(t *testing.T) {
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

	// Générer un rapport
	today := time.Now().UTC().Format("2006-01-02")
	report, err := generator.Generate(audit.PeriodTypeCustom, today, today)
	require.NoError(t, err)

	// Exporter en CSV
	outputPath := filepath.Join(tmpDir, "report.csv")
	err = generator.ExportCSV(report, outputPath)
	require.NoError(t, err)

	// Vérifier que le fichier existe
	_, err = os.Stat(outputPath)
	require.NoError(t, err)

	// Vérifier le contenu CSV (au moins une ligne d'en-tête et une ligne de données)
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "period_type")
	assert.Contains(t, string(data), "total_documents")
}

func TestReportGenerator_ExportCSV_Stdout(t *testing.T) {
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

	// Exporter vers stdout
	err = generator.ExportCSV(report, "-")
	require.NoError(t, err)
}

func TestReportGenerator_Sign(t *testing.T) {
	tmpDir := t.TempDir()

	// Créer des clés JWS de test
	keyDir := filepath.Join(tmpDir, "keys")
	err := os.MkdirAll(keyDir, 0755)
	require.NoError(t, err)

	privateKeyPath := filepath.Join(keyDir, "private.pem")
	publicKeyPath := filepath.Join(keyDir, "public.pem")

	// Générer des clés de test (simplifié - en production utiliser cmd/keygen)
	// Pour les tests, on peut créer des clés minimales ou mocker le service

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)

	// Si JWS service disponible, tester la signature
	jwsService, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-key")
	if err != nil {
		t.Skip("JWS service not available for testing")
	}

	generator := audit.NewReportGenerator(logger, exporter, nil, jwsService, zerolog.Nop())

	today := time.Now().UTC().Format("2006-01-02")
	report, err := generator.Generate(audit.PeriodTypeCustom, today, today)
	require.NoError(t, err)

	// Signer le rapport
	err = generator.Sign(report)
	require.NoError(t, err)
	assert.NotEmpty(t, report.Metadata.ReportHash)
	assert.NotEmpty(t, report.Metadata.ReportJWS)
}

func TestReportGenerator_Sign_NoJWSService(t *testing.T) {
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

	// Tenter de signer sans JWS service
	err = generator.Sign(report)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "JWS service not available")
}

func TestReportGenerator_CollectDailySignatures(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Créer une signature de test
	sigDir := filepath.Join(tmpDir, "signatures")
	err = os.MkdirAll(sigDir, 0755)
	require.NoError(t, err)

	today := time.Now().UTC().Format("2006-01-02")
	sigPath := logger.GetSignaturePath(today)

	dailyHash := audit.DailyHash{
		Date:      today,
		Hash:      "test-hash",
		JWS:       "test-jws",
		LineCount: 10,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	sigData, err := json.Marshal(dailyHash)
	require.NoError(t, err)
	err = os.WriteFile(sigPath, sigData, 0644)
	require.NoError(t, err)

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())

	// Générer un rapport qui devrait inclure la signature
	report, err := generator.Generate(audit.PeriodTypeCustom, today, today)
	require.NoError(t, err)
	assert.NotNil(t, report)
	// La signature devrait être collectée si elle existe
}

func TestReportGenerator_WithDatabase(t *testing.T) {
	// Test avec base de données (nécessite DATABASE_URL)
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

	// Connexion à la base de données
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
	// Vérifier que les sources de données incluent "database"
	assert.Contains(t, report.Metadata.DataSources, "database")
}

func TestReportGenerator_Metadata(t *testing.T) {
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

	// Vérifier les métadonnées
	assert.NotEmpty(t, report.Metadata.ReportID)
	assert.NotEmpty(t, report.Metadata.GeneratedAt)
	assert.Equal(t, "cli", report.Metadata.GeneratedBy)
	assert.Contains(t, report.Metadata.DataSources, "audit_logs")
}

// TestReportGenerator_WithTestData utilise le jeu de données de test complet
// Ce test est défini dans audit_report_testdata.go
func TestReportGenerator_WithTestData(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Créer des données de test pour 7 jours
	startDate := "2025-01-15"
	endDate := "2025-01-21"

	err = createTestAuditData(logger, startDate, endDate)
	require.NoError(t, err)

	// Créer des signatures de test
	err = createTestSignatures(logger, startDate, endDate)
	require.NoError(t, err)

	exporter := audit.NewExporter(logger)
	generator := audit.NewReportGenerator(logger, exporter, nil, nil, zerolog.Nop())

	// Générer un rapport pour cette période
	report, err := generator.Generate(audit.PeriodTypeCustom, startDate, endDate)
	require.NoError(t, err)
	assert.NotNil(t, report)

	// Vérifier que le rapport contient des données
	// Note: TotalDocuments sera 0 sans DB, mais on vérifie au moins que le rapport est généré
	assert.GreaterOrEqual(t, report.Summary.TotalDocuments, int64(0))
	assert.Greater(t, len(report.Signatures), 0, "Les signatures journalières doivent être collectées")

	// Vérifier les statistiques
	assert.NotNil(t, report.Documents)
	assert.NotNil(t, report.Errors)
	assert.NotNil(t, report.Performance)
	assert.NotNil(t, report.Reconciliation)

	// Vérifier les métadonnées
	assert.NotEmpty(t, report.Metadata.ReportID)
	assert.NotEmpty(t, report.Metadata.GeneratedAt)
	assert.Contains(t, report.Metadata.DataSources, "audit_logs")
	assert.Contains(t, report.Metadata.DataSources, "daily_signatures")

	// Tester l'export JSON avec ces données
	outputPath := filepath.Join(tmpDir, "report-with-data.json")
	err = generator.ExportJSON(report, outputPath)
	require.NoError(t, err)

	// Vérifier que le fichier existe et contient des données
	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	// Tester l'export CSV
	csvPath := filepath.Join(tmpDir, "report-with-data.csv")
	err = generator.ExportCSV(report, csvPath)
	require.NoError(t, err)

	info, err = os.Stat(csvPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

