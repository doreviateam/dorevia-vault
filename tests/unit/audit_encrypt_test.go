package unit

import (
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewEncryptionService_Disabled teste EncryptionService désactivé
func TestNewEncryptionService_Disabled(t *testing.T) {
	cfg := audit.EncryptionConfig{
		Enabled: false,
		Logger:  zerolog.Nop(),
	}

	service, err := audit.NewEncryptionService(cfg)
	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.False(t, service.IsEnabled())
}

// TestNewEncryptionService_Enabled teste EncryptionService activé
func TestNewEncryptionService_Enabled(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := audit.EncryptionConfig{
		Enabled:    true,
		KeyManager: keyManager,
		KeyID:      "default",
		Logger:     log,
	}

	service, err := audit.NewEncryptionService(cfg)
	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.True(t, service.IsEnabled())
}

// TestEncryptionService_EncryptDecrypt teste le chiffrement/déchiffrement
func TestEncryptionService_EncryptDecrypt(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := audit.EncryptionConfig{
		Enabled:    true,
		KeyManager: keyManager,
		KeyID:      "default",
		Logger:     log,
	}

	service, err := audit.NewEncryptionService(cfg)
	require.NoError(t, err)

	// Texte à chiffrer
	plaintext := []byte("This is a test log entry with sensitive data")

	// Chiffrer
	ciphertext, err := service.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotNil(t, ciphertext)
	assert.NotEqual(t, plaintext, ciphertext) // Le texte chiffré doit être différent

	// Déchiffrer
	decrypted, err := service.Decrypt(ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// TestEncryptionService_EncryptStringDecryptString teste avec strings
func TestEncryptionService_EncryptStringDecryptString(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := audit.EncryptionConfig{
		Enabled:    true,
		KeyManager: keyManager,
		KeyID:      "default",
		Logger:     log,
	}

	service, err := audit.NewEncryptionService(cfg)
	require.NoError(t, err)

	plaintext := "This is a test log entry with sensitive data"

	// Chiffrer
	ciphertextBase64, err := service.EncryptString(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertextBase64)

	// Déchiffrer
	decrypted, err := service.DecryptString(ciphertextBase64)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

// TestEncryptionService_Encrypt_Disabled teste avec chiffrement désactivé
func TestEncryptionService_Encrypt_Disabled(t *testing.T) {
	cfg := audit.EncryptionConfig{
		Enabled: false,
		Logger:  zerolog.Nop(),
	}

	service, err := audit.NewEncryptionService(cfg)
	require.NoError(t, err)

	plaintext := []byte("test data")

	// Chiffrer (devrait retourner tel quel)
	ciphertext, err := service.Encrypt(plaintext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, ciphertext)
}

// TestEncryptionService_Decrypt_InvalidCiphertext teste avec ciphertext invalide
func TestEncryptionService_Decrypt_InvalidCiphertext(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := audit.EncryptionConfig{
		Enabled:    true,
		KeyManager: keyManager,
		KeyID:      "default",
		Logger:     log,
	}

	service, err := audit.NewEncryptionService(cfg)
	require.NoError(t, err)

	// Ciphertext trop court
	invalidCiphertext := []byte("too short")
	_, err = service.Decrypt(invalidCiphertext)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

// TestEncryptionService_DecryptString_InvalidBase64 teste avec base64 invalide
func TestEncryptionService_DecryptString_InvalidBase64(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath, publicKeyPath := createTestKeys(t, tmpDir)

	log := zerolog.Nop()
	keyManager := crypto.NewFileKeyManager(privateKeyPath, publicKeyPath, log)

	cfg := audit.EncryptionConfig{
		Enabled:    true,
		KeyManager: keyManager,
		KeyID:      "default",
		Logger:     log,
	}

	service, err := audit.NewEncryptionService(cfg)
	require.NoError(t, err)

	// Base64 invalide
	_, err = service.DecryptString("invalid-base64!")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode base64")
}

// Note: createTestKeys est définie dans crypto_vault_test.go

