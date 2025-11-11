package unit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewKeyRotation teste la création d'un KeyRotation
func TestNewKeyRotation(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := crypto.RotationConfig{
		KeyManager:     keyManager,
		CurrentKID:     "default",
		RotationPeriod: 90 * 24 * time.Hour,
		Logger:         log,
	}

	rotation, err := crypto.NewKeyRotation(cfg)
	require.NoError(t, err)
	assert.NotNil(t, rotation)
	assert.Equal(t, "default", rotation.GetCurrentKID())
}

// TestKeyRotation_GetCurrentKID teste GetCurrentKID
func TestKeyRotation_GetCurrentKID(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := crypto.RotationConfig{
		KeyManager: keyManager,
		CurrentKID: "test-kid",
		Logger:     log,
	}

	rotation, err := crypto.NewKeyRotation(cfg)
	require.NoError(t, err)

	assert.Equal(t, "test-kid", rotation.GetCurrentKID())
}

// TestKeyRotation_GetKeyPair teste GetKeyPair
func TestKeyRotation_GetKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := crypto.RotationConfig{
		KeyManager: keyManager,
		CurrentKID: "default",
		Logger:     log,
	}

	rotation, err := crypto.NewKeyRotation(cfg)
	require.NoError(t, err)

	keyPair, err := rotation.GetKeyPair("default")
	require.NoError(t, err)
	assert.NotNil(t, keyPair)
	assert.Equal(t, "default", keyPair.KID)
	assert.NotNil(t, keyPair.PrivateKey)
	assert.NotNil(t, keyPair.PublicKey)
}

// TestKeyRotation_GetCurrentKeyPair teste GetCurrentKeyPair
func TestKeyRotation_GetCurrentKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := crypto.RotationConfig{
		KeyManager: keyManager,
		CurrentKID: "default",
		Logger:     log,
	}

	rotation, err := crypto.NewKeyRotation(cfg)
	require.NoError(t, err)

	keyPair, err := rotation.GetCurrentKeyPair()
	require.NoError(t, err)
	assert.NotNil(t, keyPair)
	assert.Equal(t, "default", keyPair.KID)
}

// TestKeyRotation_GetAllActiveKeys teste GetAllActiveKeys
func TestKeyRotation_GetAllActiveKeys(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := crypto.RotationConfig{
		KeyManager: keyManager,
		CurrentKID: "default",
		Logger:     log,
	}

	rotation, err := crypto.NewKeyRotation(cfg)
	require.NoError(t, err)

	activeKeys := rotation.GetAllActiveKeys()
	assert.GreaterOrEqual(t, len(activeKeys), 1)
	assert.Equal(t, "default", activeKeys[0].KID)
}

// TestKeyRotation_ShouldRotate teste ShouldRotate
func TestKeyRotation_ShouldRotate(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := crypto.RotationConfig{
		KeyManager:     keyManager,
		CurrentKID:     "default",
		RotationPeriod: 1 * time.Hour, // Court pour test
		Logger:         log,
	}

	rotation, err := crypto.NewKeyRotation(cfg)
	require.NoError(t, err)

	// Ne devrait pas nécessiter de rotation immédiatement
	shouldRotate := rotation.ShouldRotate()
	assert.False(t, shouldRotate)
}

// TestKeyRotation_GetNextRotationDate teste GetNextRotationDate
func TestKeyRotation_GetNextRotationDate(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := crypto.RotationConfig{
		KeyManager: keyManager,
		CurrentKID: "default",
		Logger:     log,
	}

	rotation, err := crypto.NewKeyRotation(cfg)
	require.NoError(t, err)

	nextRotation := rotation.GetNextRotationDate()
	assert.True(t, nextRotation.After(time.Now()))
}

// TestKeyRotation_GetJWKS teste GetJWKS avec clés actives
func TestKeyRotation_GetJWKS(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := crypto.RotationConfig{
		KeyManager: keyManager,
		CurrentKID: "default",
		Logger:     log,
	}

	rotation, err := crypto.NewKeyRotation(cfg)
	require.NoError(t, err)

	jwksBytes, err := rotation.GetJWKS()
	require.NoError(t, err)

	// Vérifier que c'est du JSON valide
	var jwks map[string]interface{}
	err = json.Unmarshal(jwksBytes, &jwks)
	require.NoError(t, err)

	// Vérifier la structure JWKS
	keys, ok := jwks["keys"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(keys), 1)

	// Vérifier que la clé a les champs requis
	key, ok := keys[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "RSA", key["kty"])
	assert.Equal(t, "default", key["kid"])
	assert.Equal(t, "sig", key["use"])
	assert.Equal(t, "RS256", key["alg"])
}

// Note: createTestKeys est définie dans crypto_vault_test.go

