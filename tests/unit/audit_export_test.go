package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExporter_Export_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)

	opts := audit.ExportOptions{
		Format: audit.ExportFormatJSON,
		Page:   1,
		Limit:  100,
	}

	result, err := exporter.Export(opts)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(0), result.Total)
	assert.Equal(t, 0, len(result.Events))
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 100, result.Limit)
	assert.False(t, result.HasNext)
	assert.False(t, result.HasPrevious)
}

func TestExporter_Export_WithEvents(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir:      tmpDir,
		MaxBuffer:     100,
		FlushInterval: 100 * time.Millisecond,
		Logger:        zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Logger quelques événements
	today := time.Now().UTC().Format("2006-01-02")
	for i := 0; i < 5; i++ {
		event := audit.Event{
			EventType: audit.EventTypeDocumentVaulted,
			DocumentID: "doc-" + string(rune(i)),
			Status:    audit.EventStatusSuccess,
		}
		err := logger.Log(event)
		require.NoError(t, err)
	}

	// Flush pour forcer l'écriture
	err = logger.Flush()
	require.NoError(t, err)

	// Exporter
	exporter := audit.NewExporter(logger)
	opts := audit.ExportOptions{
		From:   today,
		To:     today,
		Format: audit.ExportFormatJSON,
		Page:   1,
		Limit:  10,
	}

	result, err := exporter.Export(opts)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(5), result.Total)
	assert.Equal(t, 5, len(result.Events))
	assert.Equal(t, 1, result.Page)
	assert.False(t, result.HasNext)
}

func TestExporter_Export_Pagination(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir:      tmpDir,
		MaxBuffer:     100,
		FlushInterval: 100 * time.Millisecond,
		Logger:        zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Logger 15 événements
	today := time.Now().UTC().Format("2006-01-02")
	for i := 0; i < 15; i++ {
		event := audit.Event{
			EventType: audit.EventTypeDocumentVaulted,
			Status:    audit.EventStatusSuccess,
		}
		err := logger.Log(event)
		require.NoError(t, err)
	}

	// Flush
	err = logger.Flush()
	require.NoError(t, err)

	exporter := audit.NewExporter(logger)

	// Page 1 : 10 événements
	opts1 := audit.ExportOptions{
		From:   today,
		To:     today,
		Format: audit.ExportFormatJSON,
		Page:   1,
		Limit:  10,
	}

	result1, err := exporter.Export(opts1)
	require.NoError(t, err)
	assert.Equal(t, int64(15), result1.Total)
	assert.Equal(t, 10, len(result1.Events))
	assert.True(t, result1.HasNext)
	assert.False(t, result1.HasPrevious)

	// Page 2 : 5 événements restants
	opts2 := audit.ExportOptions{
		From:   today,
		To:     today,
		Format: audit.ExportFormatJSON,
		Page:   2,
		Limit:  10,
	}

	result2, err := exporter.Export(opts2)
	require.NoError(t, err)
	assert.Equal(t, int64(15), result2.Total)
	assert.Equal(t, 5, len(result2.Events))
	assert.False(t, result2.HasNext)
	assert.True(t, result2.HasPrevious)
}

func TestExporter_Export_InvalidDate(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)

	opts := audit.ExportOptions{
		From:   "invalid-date",
		To:     "2025-01-15",
		Format: audit.ExportFormatJSON,
	}

	result, err := exporter.Export(opts)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestExporter_Export_DateRange(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir:      tmpDir,
		MaxBuffer:     100,
		FlushInterval: 100 * time.Millisecond,
		Logger:        zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Logger des événements sur plusieurs jours
	dates := []string{"2025-01-15", "2025-01-16", "2025-01-17"}
	for _, date := range dates {
		// Créer le fichier manuellement pour simuler plusieurs jours
		logPath := logger.GetLogPath(date)
		dir := filepath.Dir(logPath)
		os.MkdirAll(dir, 0755)

		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		require.NoError(t, err)

		event := audit.Event{
			Timestamp: date + "T10:00:00Z",
			EventType: audit.EventTypeDocumentVaulted,
			Status:    audit.EventStatusSuccess,
		}

		data, _ := json.Marshal(event)
		file.Write(data)
		file.WriteString("\n")
		file.Close()
	}

	exporter := audit.NewExporter(logger)

	// Exporter sur 3 jours
	opts := audit.ExportOptions{
		From:   "2025-01-15",
		To:     "2025-01-17",
		Format: audit.ExportFormatJSON,
		Page:   1,
		Limit:  100,
	}

	result, err := exporter.Export(opts)
	require.NoError(t, err)
	assert.Equal(t, int64(3), result.Total)
	assert.Equal(t, 3, len(result.Events))
}

func TestExporter_ExportToCSV(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	exporter := audit.NewExporter(logger)

	result := &audit.ExportResult{
		Events: []audit.Event{
			{
				Timestamp:  "2025-01-15T10:00:00Z",
				EventType:   audit.EventTypeDocumentVaulted,
				DocumentID: "doc-123",
				RequestID:  "req-456",
				Source:     "sales",
				Status:     audit.EventStatusSuccess,
				DurationMS: 123,
			},
		},
		Total: 1,
	}

	csv, err := exporter.ExportToCSV(result)
	require.NoError(t, err)
	assert.Contains(t, csv, "timestamp,event_type,document_id")
	assert.Contains(t, csv, "2025-01-15T10:00:00Z")
	assert.Contains(t, csv, "document_vaulted")
	assert.Contains(t, csv, "doc-123")
}

func TestExporter_ListAvailableDates(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Créer quelques fichiers de log
	dates := []string{"2025-01-15", "2025-01-16", "2025-01-17"}
	for _, date := range dates {
		logPath := logger.GetLogPath(date)
		dir := filepath.Dir(logPath)
		os.MkdirAll(dir, 0755)
		file, err := os.Create(logPath)
		require.NoError(t, err)
		file.Close()
	}

	exporter := audit.NewExporter(logger)
	availableDates, err := exporter.ListAvailableDates()
	require.NoError(t, err)
	assert.Equal(t, 3, len(availableDates))
	assert.Contains(t, availableDates, "2025-01-15")
	assert.Contains(t, availableDates, "2025-01-16")
	assert.Contains(t, availableDates, "2025-01-17")
}

