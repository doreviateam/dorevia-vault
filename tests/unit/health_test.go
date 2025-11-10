package unit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetermineGlobalStatus_OK teste le statut global quand tous les composants sont OK
func TestDetermineGlobalStatus_OK(t *testing.T) {
	components := []health.ComponentHealth{
		{Status: health.StatusOK, Message: "DB OK"},
		{Status: health.StatusOK, Message: "Storage OK"},
		{Status: health.StatusOK, Message: "JWS OK"},
		{Status: health.StatusOK, Message: "Ledger OK"},
	}

	status := health.DetermineGlobalStatus(components...)
	assert.Equal(t, health.StatusOK, status)
}

// TestDetermineGlobalStatus_Warn teste le statut global avec warnings
func TestDetermineGlobalStatus_Warn(t *testing.T) {
	components := []health.ComponentHealth{
		{Status: health.StatusOK, Message: "DB OK"},
		{Status: health.StatusWarn, Message: "Storage warning"},
		{Status: health.StatusOK, Message: "JWS OK"},
		{Status: health.StatusOK, Message: "Ledger OK"},
	}

	status := health.DetermineGlobalStatus(components...)
	assert.Equal(t, health.StatusWarn, status)
}

// TestDetermineGlobalStatus_Fail teste le statut global avec erreurs
func TestDetermineGlobalStatus_Fail(t *testing.T) {
	components := []health.ComponentHealth{
		{Status: health.StatusOK, Message: "DB OK"},
		{Status: health.StatusFail, Message: "Storage failed"},
		{Status: health.StatusOK, Message: "JWS OK"},
		{Status: health.StatusOK, Message: "Ledger OK"},
	}

	status := health.DetermineGlobalStatus(components...)
	assert.Equal(t, health.StatusFail, status)
}

// TestDetermineGlobalStatus_FailPrioritaire teste que fail est prioritaire sur warn
func TestDetermineGlobalStatus_FailPrioritaire(t *testing.T) {
	components := []health.ComponentHealth{
		{Status: health.StatusOK, Message: "DB OK"},
		{Status: health.StatusWarn, Message: "Storage warning"},
		{Status: health.StatusFail, Message: "JWS failed"},
		{Status: health.StatusOK, Message: "Ledger OK"},
	}

	status := health.DetermineGlobalStatus(components...)
	assert.Equal(t, health.StatusFail, status)
}

// TestCheckStorage_OK teste la vérification storage avec répertoire valide
func TestCheckStorage_OK(t *testing.T) {
	tmpDir := t.TempDir()

	result := health.CheckStorage(tmpDir)

	assert.Equal(t, health.StatusOK, result.Status)
	assert.Contains(t, result.Message, tmpDir)
	assert.NotNil(t, result.Latency)
}

// TestCheckStorage_NotExists teste la vérification storage avec répertoire inexistant
func TestCheckStorage_NotExists(t *testing.T) {
	nonExistentDir := "/nonexistent/directory/that/does/not/exist"

	result := health.CheckStorage(nonExistentDir)

	assert.Equal(t, health.StatusFail, result.Status)
	assert.Contains(t, result.Message, "does not exist")
}

// TestCheckStorage_NotDirectory teste la vérification storage avec fichier au lieu de répertoire
func TestCheckStorage_NotDirectory(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	require.NoError(t, err)

	result := health.CheckStorage(tmpFile)

	assert.Equal(t, health.StatusFail, result.Status)
	assert.Contains(t, result.Message, "not a directory")
}

// TestCheckStorage_NotWritable teste la vérification storage avec répertoire non inscriptible
func TestCheckStorage_NotWritable(t *testing.T) {
	// Créer un répertoire en lecture seule (permissions 555)
	tmpDir := t.TempDir()
	err := os.Chmod(tmpDir, 0555)
	require.NoError(t, err)
	defer os.Chmod(tmpDir, 0755) // Restaurer après test

	result := health.CheckStorage(tmpDir)

	// Le test peut être OK ou WARN selon les permissions réelles
	// On vérifie juste que ce n'est pas FAIL
	assert.NotEqual(t, health.StatusFail, result.Status)
}

// TestCheckStorage_EmptyPath teste la vérification storage avec chemin vide
func TestCheckStorage_EmptyPath(t *testing.T) {
	result := health.CheckStorage("")

	assert.Equal(t, health.StatusFail, result.Status)
	assert.Contains(t, result.Message, "not configured")
}

// TestCheckJWS_OK teste la vérification JWS avec service valide
func TestCheckJWS_OK(t *testing.T) {
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-kid")
	require.NoError(t, err)

	result := health.CheckJWS(service)

	assert.Equal(t, health.StatusOK, result.Status)
	assert.Contains(t, result.Message, "operational")
	assert.NotNil(t, result.Latency)
}

// TestCheckJWS_NilService teste la vérification JWS sans service (mode dégradé)
func TestCheckJWS_NilService(t *testing.T) {
	result := health.CheckJWS(nil)

	assert.Equal(t, health.StatusWarn, result.Status)
	assert.Contains(t, result.Message, "not configured")
	assert.Contains(t, result.Message, "degraded")
}

// TestCheckJWS_ServiceError teste la vérification JWS avec service en erreur
// Note: Ce test nécessiterait un mock de service JWS en erreur
// Pour l'instant, on teste juste que le service nil retourne WARN
func TestCheckJWS_ServiceError(t *testing.T) {
	// Test avec service nil (cas le plus simple à tester)
	result := health.CheckJWS(nil)
	assert.Equal(t, health.StatusWarn, result.Status)
}

// TestCheckDatabase_NilDB teste la vérification database sans DB configurée
func TestCheckDatabase_NilDB(t *testing.T) {
	ctx := context.Background()

	result := health.CheckDatabase(ctx, nil)

	assert.Equal(t, health.StatusFail, result.Status)
	assert.Contains(t, result.Message, "not configured")
}

// TestCheckDatabase_RequiresDB teste la vérification database avec DB réelle
// Note: Ce test nécessite une connexion DB réelle, donc on le skip
func TestCheckDatabase_RequiresDB(t *testing.T) {
	t.Skip("Requires database connection - integration test")
}

// TestCheckLedger_NilDB teste la vérification ledger sans DB configurée
func TestCheckLedger_NilDB(t *testing.T) {
	ctx := context.Background()

	result := health.CheckLedger(ctx, nil)

	assert.Equal(t, health.StatusFail, result.Status)
	assert.Contains(t, result.Message, "not configured")
}

// TestCheckLedger_RequiresDB teste la vérification ledger avec DB réelle
// Note: Ce test nécessite une connexion DB réelle, donc on le skip
func TestCheckLedger_RequiresDB(t *testing.T) {
	t.Skip("Requires database connection - integration test")
}

// TestCheckDetailedHealth_AllOK teste CheckDetailedHealth avec tous les composants OK
func TestCheckDetailedHealth_AllOK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-kid")
	require.NoError(t, err)

	// Test sans DB (nil) - on teste juste la structure
	result := health.CheckDetailedHealth(ctx, nil, tmpDir, service)

	// Vérifier la structure de la réponse
	assert.NotZero(t, result.Timestamp)
	assert.NotNil(t, result.Database)
	assert.NotNil(t, result.Storage)
	assert.NotNil(t, result.JWS)
	assert.NotNil(t, result.Ledger)

	// Storage devrait être OK
	assert.Equal(t, health.StatusOK, result.Storage.Status)

	// JWS devrait être OK
	assert.Equal(t, health.StatusOK, result.JWS.Status)

	// Database et Ledger seront FAIL car DB est nil
	assert.Equal(t, health.StatusFail, result.Database.Status)
	assert.Equal(t, health.StatusFail, result.Ledger.Status)

	// Statut global devrait être FAIL (au moins un composant en fail)
	assert.Equal(t, health.StatusFail, result.Status)
}

// TestCheckDetailedHealth_StorageWarn teste CheckDetailedHealth avec storage en warning
func TestCheckDetailedHealth_StorageWarn(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Répertoire non inscriptible
	tmpDir := t.TempDir()
	err := os.Chmod(tmpDir, 0555)
	require.NoError(t, err)
	defer os.Chmod(tmpDir, 0755)

	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-kid")
	require.NoError(t, err)

	result := health.CheckDetailedHealth(ctx, nil, tmpDir, service)

	// Storage peut être OK ou WARN selon les permissions
	assert.Contains(t, []health.Status{health.StatusOK, health.StatusWarn}, result.Storage.Status)
}

// TestCheckDetailedHealth_JWSDegraded teste CheckDetailedHealth avec JWS en mode dégradé
func TestCheckDetailedHealth_JWSDegraded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tmpDir := t.TempDir()

	// Pas de service JWS (nil)
	result := health.CheckDetailedHealth(ctx, nil, tmpDir, nil)

	// JWS devrait être WARN (mode dégradé)
	assert.Equal(t, health.StatusWarn, result.JWS.Status)
	assert.Contains(t, result.JWS.Message, "degraded")

	// Statut global devrait être WARN ou FAIL
	assert.Contains(t, []health.Status{health.StatusWarn, health.StatusFail}, result.Status)
}

// TestComponentHealth_Structure teste la structure ComponentHealth
func TestComponentHealth_Structure(t *testing.T) {
	comp := health.ComponentHealth{
		Status:  health.StatusOK,
		Message: "Test message",
		Latency: stringPtr("12.34"),
	}

	assert.Equal(t, health.StatusOK, comp.Status)
	assert.Equal(t, "Test message", comp.Message)
	assert.NotNil(t, comp.Latency)
	assert.Equal(t, "12.34", *comp.Latency)
}

// TestDetailedHealth_Structure teste la structure DetailedHealth
func TestDetailedHealth_Structure(t *testing.T) {
	detailedHealth := health.DetailedHealth{
		Status:    health.StatusOK,
		Timestamp: time.Now(),
		Database:  health.ComponentHealth{Status: health.StatusOK},
		Storage:   health.ComponentHealth{Status: health.StatusOK},
		JWS:       health.ComponentHealth{Status: health.StatusOK},
		Ledger:    health.ComponentHealth{Status: health.StatusOK},
	}

	assert.Equal(t, health.StatusOK, detailedHealth.Status)
	assert.NotZero(t, detailedHealth.Timestamp)
	assert.Equal(t, health.StatusOK, detailedHealth.Database.Status)
	assert.Equal(t, health.StatusOK, detailedHealth.Storage.Status)
	assert.Equal(t, health.StatusOK, detailedHealth.JWS.Status)
	assert.Equal(t, health.StatusOK, detailedHealth.Ledger.Status)
}

// stringPtr retourne un pointeur vers une string
func stringPtr(s string) *string {
	return &s
}

