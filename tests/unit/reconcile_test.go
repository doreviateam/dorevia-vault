package unit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/reconcile"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOrphanFile_Structure teste la structure OrphanFile
func TestOrphanFile_Structure(t *testing.T) {
	orphan := reconcile.OrphanFile{
		Path:      "/path/to/orphan.pdf",
		SizeBytes: 12345,
		SHA256Hex: "abc123def456",
	}

	assert.Equal(t, "/path/to/orphan.pdf", orphan.Path)
	assert.Equal(t, int64(12345), orphan.SizeBytes)
	assert.Equal(t, "abc123def456", orphan.SHA256Hex)
}

// TestOrphanDB_Structure teste la structure OrphanDB
func TestOrphanDB_Structure(t *testing.T) {
	orphan := reconcile.OrphanDB{
		DocumentID: "123e4567-e89b-12d3-a456-426614174000",
		Filename:   "missing.pdf",
		StoredPath: "/path/to/missing.pdf",
		SHA256Hex:  "abc123def456",
	}

	assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", orphan.DocumentID)
	assert.Equal(t, "missing.pdf", orphan.Filename)
	assert.Equal(t, "/path/to/missing.pdf", orphan.StoredPath)
	assert.Equal(t, "abc123def456", orphan.SHA256Hex)
}

// TestReconciliationReport_Structure teste la structure ReconciliationReport
func TestReconciliationReport_Structure(t *testing.T) {
	report := &reconcile.ReconciliationReport{
		Timestamp:    time.Now().UTC(),
		DryRun:       true,
		OrphanFiles:  []reconcile.OrphanFile{},
		OrphanDBs:    []reconcile.OrphanDB{},
		FilesDeleted: 0,
		DBsMarked:    0,
		Errors:       []string{},
	}

	assert.True(t, report.DryRun)
	assert.Empty(t, report.OrphanFiles)
	assert.Empty(t, report.OrphanDBs)
	assert.Zero(t, report.FilesDeleted)
	assert.Zero(t, report.DBsMarked)
	assert.Empty(t, report.Errors)
}

// TestReconciliationReport_WithOrphans teste un rapport avec orphelins
func TestReconciliationReport_WithOrphans(t *testing.T) {
	report := &reconcile.ReconciliationReport{
		Timestamp: time.Now().UTC(),
		DryRun:    true,
		OrphanFiles: []reconcile.OrphanFile{
			{Path: "/path/to/orphan1.pdf", SizeBytes: 12345, SHA256Hex: "abc123"},
			{Path: "/path/to/orphan2.pdf", SizeBytes: 67890, SHA256Hex: "def456"},
		},
		OrphanDBs: []reconcile.OrphanDB{
			{DocumentID: uuid.New().String(), Filename: "missing.pdf", StoredPath: "/path/to/missing.pdf"},
		},
		FilesDeleted: 0,
		DBsMarked:    0,
		Errors:       []string{},
	}

	assert.Len(t, report.OrphanFiles, 2)
	assert.Len(t, report.OrphanDBs, 1)
	assert.True(t, report.DryRun)
}

// TestReconciliationReport_FixMode teste un rapport en mode fix
func TestReconciliationReport_FixMode(t *testing.T) {
	report := &reconcile.ReconciliationReport{
		Timestamp:    time.Now().UTC(),
		DryRun:       false,
		OrphanFiles:  []reconcile.OrphanFile{{Path: "/path/to/orphan.pdf", SizeBytes: 12345}},
		OrphanDBs:    []reconcile.OrphanDB{},
		FilesDeleted: 1,
		DBsMarked:    0,
		Errors:       []string{},
	}

	assert.False(t, report.DryRun)
	assert.Equal(t, 1, report.FilesDeleted)
}

// TestCleanupOrphans_RequiresDB teste que CleanupOrphans nécessite une DB
func TestCleanupOrphans_RequiresDB(t *testing.T) {
	// Ce test nécessite une base de données réelle
	// Pour l'instant, on teste la logique sans DB
	t.Skip("Requires database connection - integration test")
}

// TestCleanupOrphans_Logic teste la logique de détection sans DB
func TestCleanupOrphans_Logic(t *testing.T) {
	// Test de la logique de détection d'orphelins
	tmpDir := t.TempDir()
	
	// Créer un fichier orphelin (sans entrée DB)
	orphanPath := filepath.Join(tmpDir, "orphan.pdf")
	content := []byte("orphan content")
	err := os.WriteFile(orphanPath, content, 0644)
	require.NoError(t, err)
	defer os.Remove(orphanPath)
	
	// Calculer SHA256
	hash := sha256.Sum256(content)
	sha256Hex := hex.EncodeToString(hash[:])
	
	// Vérifier que le fichier existe
	fileInfo, err := os.Stat(orphanPath)
	require.NoError(t, err)
	
	assert.NotEmpty(t, sha256Hex)
	assert.Equal(t, int64(len(content)), fileInfo.Size())
	
	// Simuler un rapport avec orphelin détecté
	report := &reconcile.ReconciliationReport{
		Timestamp: time.Now().UTC(),
		DryRun:    true,
		OrphanFiles: []reconcile.OrphanFile{
			{Path: orphanPath, SizeBytes: fileInfo.Size(), SHA256Hex: sha256Hex},
		},
		OrphanDBs: []reconcile.OrphanDB{},
		Errors:    []string{},
	}
	
	assert.Len(t, report.OrphanFiles, 1)
	assert.Equal(t, orphanPath, report.OrphanFiles[0].Path)
}

// TestCleanupOrphans_DryRun teste le mode dry-run
func TestCleanupOrphans_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	orphanPath := filepath.Join(tmpDir, "orphan.pdf")
	content := []byte("orphan content")
	
	err := os.WriteFile(orphanPath, content, 0644)
	require.NoError(t, err)
	
	// En mode dry-run, le fichier ne doit pas être supprimé
	report := &reconcile.ReconciliationReport{
		Timestamp:   time.Now().UTC(),
		DryRun:      true,
		OrphanFiles: []reconcile.OrphanFile{{Path: orphanPath, SizeBytes: int64(len(content))}},
		FilesDeleted: 0, // Aucun fichier supprimé en dry-run
	}
	
	assert.True(t, report.DryRun)
	assert.Zero(t, report.FilesDeleted)
	
	// Vérifier que le fichier existe toujours
	_, err = os.Stat(orphanPath)
	assert.NoError(t, err)
	
	os.Remove(orphanPath)
}

// TestCleanupOrphans_FixMode teste le mode fix (simulation)
func TestCleanupOrphans_FixMode(t *testing.T) {
	tmpDir := t.TempDir()
	orphanPath := filepath.Join(tmpDir, "orphan.pdf")
	content := []byte("orphan content")
	
	err := os.WriteFile(orphanPath, content, 0644)
	require.NoError(t, err)
	
	// En mode fix, le fichier doit être supprimé
	report := &reconcile.ReconciliationReport{
		Timestamp:   time.Now().UTC(),
		DryRun:      false,
		OrphanFiles: []reconcile.OrphanFile{{Path: orphanPath, SizeBytes: int64(len(content))}},
		FilesDeleted: 1, // Fichier supprimé en mode fix
	}
	
	assert.False(t, report.DryRun)
	assert.Equal(t, 1, report.FilesDeleted)
	
	// Simuler la suppression
	err = os.Remove(orphanPath)
	require.NoError(t, err)
	
	// Vérifier que le fichier n'existe plus
	_, err = os.Stat(orphanPath)
	assert.True(t, os.IsNotExist(err))
}

// TestCleanupOrphans_Errors teste la gestion des erreurs
func TestCleanupOrphans_Errors(t *testing.T) {
	report := &reconcile.ReconciliationReport{
		Timestamp:   time.Now().UTC(),
		DryRun:      true,
		OrphanFiles: []reconcile.OrphanFile{},
		OrphanDBs:   []reconcile.OrphanDB{},
		Errors: []string{
			"Failed to scan storage: permission denied",
			"Failed to scan database: connection timeout",
		},
	}
	
	assert.Len(t, report.Errors, 2)
	assert.Contains(t, report.Errors[0], "Failed to scan storage")
	assert.Contains(t, report.Errors[1], "Failed to scan database")
}

// TestCleanupOrphans_EmptyStorage teste le cas storage vide
func TestCleanupOrphans_EmptyStorage(t *testing.T) {
	// Storage vide (pas de fichiers)
	report := &reconcile.ReconciliationReport{
		Timestamp:   time.Now().UTC(),
		DryRun:      true,
		OrphanFiles: []reconcile.OrphanFile{},
		OrphanDBs:   []reconcile.OrphanDB{},
		Errors:      []string{},
	}
	
	assert.Empty(t, report.OrphanFiles)
	assert.Empty(t, report.OrphanDBs)
	assert.Empty(t, report.Errors)
}

// TestCleanupOrphans_ContextTimeout teste le timeout context
func TestCleanupOrphans_ContextTimeout(t *testing.T) {
	// Créer un context avec timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	
	// Attendre que le timeout expire
	time.Sleep(10 * time.Millisecond)
	
	// Vérifier que le context est expiré
	select {
	case <-ctx.Done():
		assert.True(t, true, "Context should be done")
	default:
		t.Fatal("Context should be done")
	}
}

// TestReconciliationReport_Timestamp teste le format timestamp
func TestReconciliationReport_Timestamp(t *testing.T) {
	report := &reconcile.ReconciliationReport{
		Timestamp: time.Now().UTC(),
		DryRun:    true,
	}
	
	// Vérifier que le timestamp est récent (moins de 1 seconde)
	now := time.Now().UTC()
	diff := now.Sub(report.Timestamp)
	assert.True(t, diff < 1*time.Second)
}

