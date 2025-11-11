package unit

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileKeyManager_GetPrivateKey teste FileKeyManager avec clés valides
func TestFileKeyManager_GetPrivateKey(t *testing.T) {
	tmpDir := t.TempDir()

	// Créer des clés de test
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	ctx := context.Background()
	privateKey, err := keyManager.GetPrivateKey(ctx, "test-kid")
	require.NoError(t, err)
	assert.NotNil(t, privateKey)
	assert.Equal(t, 2048, privateKey.N.BitLen())
}

// TestFileKeyManager_GetPublicKey teste FileKeyManager avec clés publiques
func TestFileKeyManager_GetPublicKey(t *testing.T) {
	tmpDir := t.TempDir()

	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	ctx := context.Background()
	publicKey, err := keyManager.GetPublicKey(ctx, "test-kid")
	require.NoError(t, err)
	assert.NotNil(t, publicKey)
	assert.Equal(t, 2048, publicKey.N.BitLen())
}

// TestFileKeyManager_ListKIDs teste la liste des KIDs
func TestFileKeyManager_ListKIDs(t *testing.T) {
	tmpDir := t.TempDir()

	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	ctx := context.Background()
	kids, err := keyManager.ListKIDs(ctx)
	require.NoError(t, err)
	assert.Contains(t, kids, "default")
}

// TestFileKeyManager_IsAvailable teste la disponibilité
func TestFileKeyManager_IsAvailable(t *testing.T) {
	tmpDir := t.TempDir()

	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	ctx := context.Background()
	available := keyManager.IsAvailable(ctx)
	assert.True(t, available)
}

// TestFileKeyManager_IsAvailable_NoFiles teste avec fichiers manquants
func TestFileKeyManager_IsAvailable_NoFiles(t *testing.T) {
	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager("/nonexistent/private.pem", "/nonexistent/public.pem", log)

	ctx := context.Background()
	available := keyManager.IsAvailable(ctx)
	assert.False(t, available)
}

// TestFileKeyManager_GetPrivateKey_InvalidPath teste avec chemin invalide
func TestFileKeyManager_GetPrivateKey_InvalidPath(t *testing.T) {
	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager("/nonexistent/private.pem", "/nonexistent/public.pem", log)

	ctx := context.Background()
	_, err := keyManager.GetPrivateKey(ctx, "test-kid")
	assert.Error(t, err)
}

// TestVaultKeyManager_Disabled teste VaultKeyManager désactivé
func TestVaultKeyManager_Disabled(t *testing.T) {
	cfg := crypto.VaultConfig{
		Enabled: false,
		Logger:  zerolog.Nop(),
	}

	keyManager, err := crypto.NewVaultKeyManager(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	available := keyManager.IsAvailable(ctx)
	assert.False(t, available)

	_, err = keyManager.GetPrivateKey(ctx, "test-kid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vault is not enabled")
}

// TestVaultKeyManager_InvalidConfig teste avec configuration invalide
func TestVaultKeyManager_InvalidConfig(t *testing.T) {
	cfg := crypto.VaultConfig{
		Enabled:   true,
		Addr:      "http://invalid-vault:8200",
		Token:     "invalid-token",
		MountPath: "secret/dorevia-vault",
		KeyPath:   "keys/jws",
		Logger:    zerolog.Nop(),
	}

	// Ne devrait pas planter même si Vault n'est pas disponible
	// (on teste juste que la création ne crashe pas)
	_, err := crypto.NewVaultKeyManager(cfg)
	// Peut échouer si Vault n'est pas disponible, c'est normal
	if err != nil {
		assert.Contains(t, err.Error(), "vault")
	}
}

// TestNewServiceWithKeyManager teste la création de Service avec KeyManager
func TestNewServiceWithKeyManager(t *testing.T) {
	tmpDir := t.TempDir()

	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	service, err := crypto.NewServiceWithKeyManager(keyManager, "test-kid")
	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, "test-kid", service.GetKID())
}

// TestNewServiceWithKeyManager_InvalidKey teste avec clé invalide
func TestNewServiceWithKeyManager_InvalidKey(t *testing.T) {
	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager("/nonexistent/private.pem", "/nonexistent/public.pem", log)

	_, err := crypto.NewServiceWithKeyManager(keyManager, "test-kid")
	assert.Error(t, err)
}

// Helper functions

// createTestKeys crée une paire de clés RSA de test
func createTestKeys(t *testing.T, dir string) (string, string) {
	// Générer une clé privée RSA (simplifié - en production utiliser cmd/keygen)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Encoder clé privée en PEM
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	privateKeyPath := filepath.Join(dir, "private.pem")
	err = os.WriteFile(privateKeyPath, privateKeyPEM, 0600)
	require.NoError(t, err)

	// Encoder clé publique en PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	publicKeyPath := filepath.Join(dir, "public.pem")
	err = os.WriteFile(publicKeyPath, publicKeyPEM, 0644)
	require.NoError(t, err)

	return privateKeyPath, publicKeyPath
}

