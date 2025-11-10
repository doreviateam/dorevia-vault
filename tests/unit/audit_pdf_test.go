package unit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPDFGenerator_NewPDFGenerator(t *testing.T) {
	generator := audit.NewPDFGenerator(nil, zerolog.Nop())
	assert.NotNil(t, generator)
}

func TestPDFGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()

	// Créer un rapport de test
	report := createTestReport()

	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-report.pdf")
	err := generator.Generate(report, outputPath)
	require.NoError(t, err)

	// Vérifier que le fichier existe
	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))

	// Vérifier que c'est un PDF (les PDF commencent par %PDF)
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(data[:10]), "%PDF")
}

func TestPDFGenerator_Generate_WithJWSService(t *testing.T) {
	tmpDir := t.TempDir()

	// Créer des clés JWS de test
	keyDir := filepath.Join(tmpDir, "keys")
	err := os.MkdirAll(keyDir, 0755)
	require.NoError(t, err)

	privateKeyPath := filepath.Join(keyDir, "private.pem")
	publicKeyPath := filepath.Join(keyDir, "public.pem")

	jwsService, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-key")
	if err != nil {
		t.Skip("JWS service not available for testing")
	}

	report := createTestReport()
	report.Metadata.ReportHash = "test-hash-1234567890abcdef"
	report.Metadata.ReportJWS = "test-jws-signature"

	generator := audit.NewPDFGenerator(jwsService, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-report-signed.pdf")
	err = generator.Generate(report, outputPath)
	require.NoError(t, err)

	// Vérifier que le fichier existe
	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

func TestPDFGenerator_Generate_EmptyReport(t *testing.T) {
	tmpDir := t.TempDir()

	// Rapport vide
	report := &audit.AuditReport{
		Period: audit.Period{
			Type:      audit.PeriodTypeCustom,
			StartDate: "2025-01-01",
			EndDate:   "2025-01-31",
			Label:     "Test",
		},
		Summary:        audit.ReportSummary{},
		Documents:      audit.DocumentStats{},
		Errors:         audit.ErrorStats{},
		Performance:    audit.PerformanceStats{},
		Ledger:         audit.LedgerStats{},
		Reconciliation: audit.ReconciliationStats{},
		Signatures:     []audit.DailyHash{},
		Metadata: audit.ReportMetadata{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			GeneratedBy: "test",
			Version:     "v1.2.0-rc1",
			ReportID:    "test-id",
			ReportHash:  "test-hash",
		},
	}

	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-empty.pdf")
	err := generator.Generate(report, outputPath)
	require.NoError(t, err)

	// Vérifier que le fichier existe
	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

func TestPDFGenerator_Generate_WithData(t *testing.T) {
	tmpDir := t.TempDir()

	report := createTestReportWithData()

	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-with-data.pdf")
	err := generator.Generate(report, outputPath)
	require.NoError(t, err)

	// Vérifier que le fichier existe et a une taille raisonnable
	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(1000)) // Au moins 1KB
}

func TestPDFGenerator_Generate_WithSignatures(t *testing.T) {
	tmpDir := t.TempDir()

	report := createTestReport()
	report.Signatures = []audit.DailyHash{
		{
			Date:      "2025-01-15",
			Hash:      "test-hash-1",
			JWS:       "test-jws-1",
			LineCount: 100,
			Timestamp: "2025-01-15T23:59:59Z",
		},
		{
			Date:      "2025-01-16",
			Hash:      "test-hash-2",
			JWS:       "test-jws-2",
			LineCount: 150,
			Timestamp: "2025-01-16T23:59:59Z",
		},
	}

	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-with-sigs.pdf")
	err := generator.Generate(report, outputPath)
	require.NoError(t, err)

	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

func TestPDFGenerator_Generate_WithErrors(t *testing.T) {
	tmpDir := t.TempDir()

	report := createTestReport()
	report.Errors.Total = 5
	report.Errors.CriticalErrors = []audit.CriticalError{
		{
			Timestamp:  "2025-01-15T10:00:00Z",
			EventType:  "document_vaulted",
			DocumentID: "doc-123",
			Message:    "Storage error",
			Count:      3,
		},
		{
			Timestamp:  "2025-01-15T11:00:00Z",
			EventType:  "jws_signed",
			DocumentID: "doc-456",
			Message:    "JWS signing failed",
			Count:      2,
		},
	}

	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-with-errors.pdf")
	err := generator.Generate(report, outputPath)
	require.NoError(t, err)

	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

func TestPDFGenerator_Generate_WithPerformance(t *testing.T) {
	tmpDir := t.TempDir()

	report := createTestReport()
	report.Performance.DocumentStorage = audit.PerformanceMetric{
		Count:  100,
		Mean:   0.150,
		Median: 0.120,
		P50:    0.120,
		P95:    0.300,
		P99:    0.500,
		Min:    0.050,
		Max:    0.800,
	}
	report.Performance.JWSSignature = audit.PerformanceMetric{
		Count:  100,
		Mean:   0.010,
		P50:    0.008,
		P95:    0.020,
		P99:    0.030,
	}

	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-with-perf.pdf")
	err := generator.Generate(report, outputPath)
	require.NoError(t, err)

	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

func TestPDFGenerator_Generate_InvalidPath(t *testing.T) {
	report := createTestReport()
	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	// Chemin invalide (répertoire inexistant)
	outputPath := "/nonexistent/directory/report.pdf"
	err := generator.Generate(report, outputPath)
	assert.Error(t, err)
}

func TestPDFGenerator_Generate_QRCode(t *testing.T) {
	tmpDir := t.TempDir()

	report := createTestReport()
	report.Metadata.ReportHash = "test-hash-1234567890abcdef1234567890abcdef" // Hash long pour QR code

	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-qrcode.pdf")
	err := generator.Generate(report, outputPath)
	require.NoError(t, err)

	// Vérifier que le PDF contient le QR code (en vérifiant la taille du fichier)
	// Un PDF avec QR code devrait être plus grand qu'un PDF sans
	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(5000)) // Au moins 5KB avec QR code
}

func TestPDFGenerator_Generate_AllPages(t *testing.T) {
	tmpDir := t.TempDir()

	// Rapport complet avec toutes les données pour tester toutes les pages
	report := createTestReportWithData()
	report.Signatures = []audit.DailyHash{
		{Date: "2025-01-15", Hash: "hash1", JWS: "jws1", LineCount: 100, Timestamp: "2025-01-15T23:59:59Z"},
		{Date: "2025-01-16", Hash: "hash2", JWS: "jws2", LineCount: 150, Timestamp: "2025-01-16T23:59:59Z"},
	}
	report.Metadata.ReportHash = "test-hash-for-qrcode"
	report.Metadata.ReportJWS = "test-jws-signature-complete"

	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-all-pages.pdf")
	err := generator.Generate(report, outputPath)
	require.NoError(t, err)

	// Vérifier que le PDF existe et a une taille raisonnable (8 pages)
	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(5000)) // Au moins 5KB pour 8 pages

	// Vérifier que c'est un PDF valide
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Contains(t, string(data[:10]), "%PDF")
}

func TestPDFGenerator_Generate_WithLedgerData(t *testing.T) {
	tmpDir := t.TempDir()

	report := createTestReport()
	report.Ledger = audit.LedgerStats{
		TotalEntries:  500,
		NewEntries:    100,
		Errors:        5,
		ErrorRate:     1.0,
		CurrentSize:   500,
		ChainIntegrity: true,
		LastHash:      "abc123def456ghi789jkl012mno345pqr678",
	}

	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-ledger.pdf")
	err := generator.Generate(report, outputPath)
	require.NoError(t, err)

	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

func TestPDFGenerator_Generate_WithReconciliationData(t *testing.T) {
	tmpDir := t.TempDir()

	report := createTestReport()
	report.Reconciliation = audit.ReconciliationStats{
		TotalRuns:        10,
		SuccessfulRuns:   8,
		FailedRuns:       2,
		OrphanFilesFound: 25,
		OrphanFilesFixed: 20,
		DocumentsFixed:   15,
	}

	generator := audit.NewPDFGenerator(nil, zerolog.Nop())

	outputPath := filepath.Join(tmpDir, "test-reconciliation.pdf")
	err := generator.Generate(report, outputPath)
	require.NoError(t, err)

	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

// Helper functions

func createTestReport() *audit.AuditReport {
	return &audit.AuditReport{
		Period: audit.Period{
			Type:      audit.PeriodTypeMonthly,
			StartDate: "2025-01-01",
			EndDate:   "2025-01-31",
			Label:     "Janvier 2025",
		},
		Summary: audit.ReportSummary{
			TotalDocuments:      100,
			TotalErrors:         5,
			ErrorRate:           5.0,
			TotalLedgerEntries:  100,
			TotalReconciliations: 2,
			AvgDocumentSize:     50000,
			TotalStorageSize:    5000000,
		},
		Documents: audit.DocumentStats{
			Total:         100,
			ByStatus:      map[string]int64{"success": 95, "error": 5},
			BySource:      map[string]int64{"sales": 60, "purchase": 40},
			ByContentType: map[string]int64{"application/pdf": 100},
			SizeDistribution: audit.SizeDistribution{
				Min:    10000,
				Max:    200000,
				Mean:   50000,
				Median: 45000,
				P95:    150000,
				P99:    180000,
			},
		},
		Errors: audit.ErrorStats{
			Total:       5,
			ByType:      map[string]int64{"storage": 3, "jws": 2},
			ByEventType: map[string]int64{"document_vaulted": 3, "jws_signed": 2},
			CriticalErrors: []audit.CriticalError{},
		},
		Performance: audit.PerformanceStats{
			DocumentStorage: audit.PerformanceMetric{Count: 100, Mean: 0.150},
			JWSSignature:     audit.PerformanceMetric{Count: 100, Mean: 0.010},
			LedgerAppend:     audit.PerformanceMetric{Count: 100, Mean: 0.050},
			Transaction:      audit.PerformanceMetric{Count: 100, Mean: 0.200},
		},
		Ledger: audit.LedgerStats{
			TotalEntries:  100,
			NewEntries:    100,
			Errors:        0,
			ErrorRate:     0,
			CurrentSize:   100,
			ChainIntegrity: true,
			LastHash:      "abc123def456",
		},
		Reconciliation: audit.ReconciliationStats{
			TotalRuns:        2,
			SuccessfulRuns:   2,
			FailedRuns:       0,
			OrphanFilesFound: 5,
			OrphanFilesFixed: 5,
			DocumentsFixed:   2,
		},
		Signatures: []audit.DailyHash{},
		Metadata: audit.ReportMetadata{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			GeneratedBy: "test",
			Version:     "v1.2.0-rc1",
			ReportID:   "test-report-id",
			ReportHash:  "test-hash-1234567890abcdef",
			DataSources: []string{"audit_logs"},
		},
	}
}

func createTestReportWithData() *audit.AuditReport {
	report := createTestReport()
	report.Documents.ByStatus = map[string]int64{
		"success":    90,
		"error":      5,
		"idempotent": 5,
	}
	report.Documents.BySource = map[string]int64{
		"sales":    50,
		"purchase": 30,
		"pos":      15,
		"stock":    5,
	}
	report.Errors.CriticalErrors = []audit.CriticalError{
		{EventType: "document_vaulted", Count: 3, Message: "Error 1"},
		{EventType: "jws_signed", Count: 2, Message: "Error 2"},
	}
	return report
}

