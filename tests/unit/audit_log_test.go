package unit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLogger_NewLogger(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir:      tmpDir,
		MaxBuffer:     100,
		FlushInterval: 1 * time.Second,
		Logger:        zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	require.NotNil(t, logger)
	defer logger.Close()

	// Vérifier que les répertoires sont créés
	assert.DirExists(t, filepath.Join(tmpDir, "logs"))
	assert.DirExists(t, filepath.Join(tmpDir, "signatures"))
}

func TestAuditLogger_NewLogger_InvalidDir(t *testing.T) {
	cfg := audit.Config{
		AuditDir: "",
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	assert.Error(t, err)
	assert.Nil(t, logger)
}

func TestAuditLogger_Log(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir:      tmpDir,
		MaxBuffer:     10,
		FlushInterval: 100 * time.Millisecond,
		Logger:        zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Logger un événement
	event := audit.Event{
		EventType:  audit.EventTypeDocumentVaulted,
		DocumentID: "test-doc-123",
		RequestID:  "req-456",
		Source:     "sales",
		Status:     audit.EventStatusSuccess,
		DurationMS: 123,
		Metadata: map[string]interface{}{
			"filename": "test.pdf",
		},
	}

	err = logger.Log(event)
	assert.NoError(t, err)

	// Flush pour forcer l'écriture
	err = logger.Flush()
	assert.NoError(t, err)

	// Vérifier que le fichier existe
	today := time.Now().UTC().Format("2006-01-02")
	logPath := logger.GetLogPath(today)
	assert.FileExists(t, logPath)
}

func TestAuditLogger_Log_AutoFlush(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir:      tmpDir,
		MaxBuffer:     5, // Buffer petit pour tester le flush automatique
		FlushInterval: 100 * time.Millisecond,
		Logger:        zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	// Logger plus d'événements que le buffer
	for i := 0; i < 10; i++ {
		event := audit.Event{
			EventType: audit.EventTypeDocumentVaulted,
			Status:    audit.EventStatusSuccess,
		}
		err := logger.Log(event)
		assert.NoError(t, err)
	}

	// Attendre un peu pour le flush
	time.Sleep(200 * time.Millisecond)

	// Vérifier que le fichier existe et contient des données
	today := time.Now().UTC().Format("2006-01-02")
	logPath := logger.GetLogPath(today)
	assert.FileExists(t, logPath)

	// Vérifier la taille du fichier (doit contenir des données)
	info, err := os.Stat(logPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestAuditLogger_GetLogPath(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	date := "2025-01-15"
	path := logger.GetLogPath(date)
	expected := filepath.Join(tmpDir, "logs", "audit-2025-01-15.log")
	assert.Equal(t, expected, path)
}

func TestAuditLogger_GetSignaturePath(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	date := "2025-01-15"
	path := logger.GetSignaturePath(date)
	expected := filepath.Join(tmpDir, "signatures", "audit-2025-01-15.log.jws")
	assert.Equal(t, expected, path)
}

func TestAuditLogger_Close(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := audit.Config{
		AuditDir:      tmpDir,
		MaxBuffer:     100,
		FlushInterval: 1 * time.Second,
		Logger:        zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)

	// Logger un événement
	event := audit.Event{
		EventType: audit.EventTypeDocumentVaulted,
		Status:    audit.EventStatusSuccess,
	}
	err = logger.Log(event)
	assert.NoError(t, err)

	// Fermer le logger (doit flush automatiquement)
	err = logger.Close()
	assert.NoError(t, err)

	// Vérifier que le fichier existe
	today := time.Now().UTC().Format("2006-01-02")
	logPath := logger.GetLogPath(today)
	assert.FileExists(t, logPath)
}

func TestAuditEvent_Timestamp(t *testing.T) {
	event := audit.Event{
		EventType: audit.EventTypeDocumentVaulted,
		Status:    audit.EventStatusSuccess,
	}

	// Timestamp doit être ajouté automatiquement si absent
	assert.Empty(t, event.Timestamp)

	// Après Log, le timestamp devrait être défini
	tmpDir := t.TempDir()
	cfg := audit.Config{
		AuditDir: tmpDir,
		Logger:   zerolog.Nop(),
	}

	logger, err := audit.NewLogger(cfg)
	require.NoError(t, err)
	defer logger.Close()

	err = logger.Log(event)
	assert.NoError(t, err)

	// Le timestamp est ajouté dans Log, mais on ne peut pas le vérifier directement
	// car l'événement est copié dans le buffer
	// On vérifie juste que Log ne retourne pas d'erreur
}

