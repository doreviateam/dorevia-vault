package unit

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestKeys crée une paire de clés RSA temporaire pour les tests
func setupTestKeys(t *testing.T) (privateKeyPath, publicKeyPath string, cleanup func()) {
	tmpDir := t.TempDir()

	// Générer paire de clés RSA
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Sauvegarder clé privée
	privateKeyPath = filepath.Join(tmpDir, "private.pem")
	privateKeyFile, err := os.Create(privateKeyPath)
	require.NoError(t, err)
	defer privateKeyFile.Close()

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	_, err = privateKeyFile.Write(privateKeyPEM)
	require.NoError(t, err)

	// Sauvegarder clé publique
	publicKeyPath = filepath.Join(tmpDir, "public.pem")
	publicKeyFile, err := os.Create(publicKeyPath)
	require.NoError(t, err)
	defer publicKeyFile.Close()

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	_, err = publicKeyFile.Write(publicKeyPEM)
	require.NoError(t, err)

	cleanup = func() {
		os.Remove(privateKeyPath)
		os.Remove(publicKeyPath)
	}

	return privateKeyPath, publicKeyPath, cleanup
}

// TestNewService_Success teste la création d'un service JWS avec succès
func TestNewService_Success(t *testing.T) {
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	kid := "test-key-2025"
	service, err := crypto.NewService(privateKeyPath, publicKeyPath, kid)

	require.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, kid, service.GetKID())
}

// TestNewService_InvalidPrivateKey teste l'échec avec clé privée invalide
func TestNewService_InvalidPrivateKey(t *testing.T) {
	_, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	// Créer un fichier invalide
	invalidKeyPath := filepath.Join(t.TempDir(), "invalid.pem")
	err := os.WriteFile(invalidKeyPath, []byte("invalid key"), 0644)
	require.NoError(t, err)

	service, err := crypto.NewService(invalidKeyPath, publicKeyPath, "test-kid")

	assert.Error(t, err)
	assert.Nil(t, service)
}

// TestNewService_InvalidPublicKey teste l'échec avec clé publique invalide
func TestNewService_InvalidPublicKey(t *testing.T) {
	privateKeyPath, _, cleanup := setupTestKeys(t)
	defer cleanup()

	// Créer un fichier invalide
	invalidKeyPath := filepath.Join(t.TempDir(), "invalid.pem")
	err := os.WriteFile(invalidKeyPath, []byte("invalid key"), 0644)
	require.NoError(t, err)

	service, err := crypto.NewService(privateKeyPath, invalidKeyPath, "test-kid")

	assert.Error(t, err)
	assert.Nil(t, service)
}

// TestNewService_MissingFiles teste l'échec avec fichiers manquants
func TestNewService_MissingFiles(t *testing.T) {
	service, err := crypto.NewService("/nonexistent/private.pem", "/nonexistent/public.pem", "test-kid")

	assert.Error(t, err)
	assert.Nil(t, service)
}

// TestSignEvidence_Success teste la signature JWS avec succès
func TestSignEvidence_Success(t *testing.T) {
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-kid")
	require.NoError(t, err)

	docID := uuid.New().String()
	shaHex := "abc123def456"
	timestamp := time.Now().UTC()

	jws, err := service.SignEvidence(docID, shaHex, timestamp)

	require.NoError(t, err)
	assert.NotEmpty(t, jws)
	// JWS doit avoir 3 parties séparées par des points
	parts := splitJWS(jws)
	assert.Equal(t, 3, len(parts), "JWS should have 3 parts (header.payload.signature)")
}

// TestSignEvidence_WithoutService teste l'échec sans service initialisé
func TestSignEvidence_WithoutService(t *testing.T) {
	service := &crypto.Service{} // Service sans clé privée

	docID := uuid.New().String()
	shaHex := "abc123"
	timestamp := time.Now()

	jws, err := service.SignEvidence(docID, shaHex, timestamp)

	assert.Error(t, err)
	assert.Empty(t, jws)
	assert.Contains(t, err.Error(), "private key not loaded")
}

// TestVerifyEvidence_Success teste la vérification JWS avec succès
func TestVerifyEvidence_Success(t *testing.T) {
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-kid")
	require.NoError(t, err)

	docID := uuid.New().String()
	shaHex := "abc123def456"
	timestamp := time.Now().UTC()

	// Signer
	jws, err := service.SignEvidence(docID, shaHex, timestamp)
	require.NoError(t, err)

	// Vérifier
	evidence, err := service.VerifyEvidence(jws)

	require.NoError(t, err)
	assert.NotNil(t, evidence)
	assert.Equal(t, docID, evidence.DocumentID)
	assert.Equal(t, shaHex, evidence.Sha256)
	// Vérifier que le timestamp est proche (tolérance 1 seconde)
	assert.WithinDuration(t, timestamp, evidence.Timestamp, time.Second)
}

// TestVerifyEvidence_InvalidJWS teste l'échec avec JWS invalide
func TestVerifyEvidence_InvalidJWS(t *testing.T) {
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-kid")
	require.NoError(t, err)

	// JWS invalide
	invalidJWS := "invalid.jws.token"

	evidence, err := service.VerifyEvidence(invalidJWS)

	assert.Error(t, err)
	assert.Nil(t, evidence)
}

// TestVerifyEvidence_TamperedJWS teste l'échec avec JWS modifié
func TestVerifyEvidence_TamperedJWS(t *testing.T) {
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-kid")
	require.NoError(t, err)

	// Signer un JWS valide
	jws, err := service.SignEvidence(uuid.New().String(), "abc123", time.Now())
	require.NoError(t, err)

	// Modifier le JWS (tamper)
	parts := splitJWS(jws)
	if len(parts) >= 2 {
		// Modifier le payload
		tamperedJWS := parts[0] + "." + "tampered_payload" + "." + parts[2]

		evidence, err := service.VerifyEvidence(tamperedJWS)

		assert.Error(t, err)
		assert.Nil(t, evidence)
	}
}

// TestVerifyEvidence_WithoutService teste l'échec sans service initialisé
func TestVerifyEvidence_WithoutService(t *testing.T) {
	service := &crypto.Service{} // Service sans clé publique

	evidence, err := service.VerifyEvidence("some.jws.token")

	assert.Error(t, err)
	assert.Nil(t, evidence)
	assert.Contains(t, err.Error(), "public key not loaded")
}

// TestCurrentJWKS_Success teste la génération JWKS avec succès
func TestCurrentJWKS_Success(t *testing.T) {
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-kid-2025")
	require.NoError(t, err)

	jwks, err := service.CurrentJWKS()

	require.NoError(t, err)
	assert.NotEmpty(t, jwks)
	// Vérifier que c'est du JSON valide
	assert.Contains(t, string(jwks), "keys")
	assert.Contains(t, string(jwks), "test-kid-2025")
	assert.Contains(t, string(jwks), "RSA")
	assert.Contains(t, string(jwks), "RS256")
}

// TestCurrentJWKS_WithoutService teste l'échec sans service initialisé
func TestCurrentJWKS_WithoutService(t *testing.T) {
	service := &crypto.Service{} // Service sans clé publique

	jwks, err := service.CurrentJWKS()

	assert.Error(t, err)
	assert.Nil(t, jwks)
	assert.Contains(t, err.Error(), "public key not loaded")
}

// TestSignVerify_RoundTrip teste le cycle complet signer → vérifier
func TestSignVerify_RoundTrip(t *testing.T) {
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "roundtrip-kid")
	require.NoError(t, err)

	// Test avec différents documents
	testCases := []struct {
		name     string
		docID    string
		shaHex   string
		timestamp time.Time
	}{
		{
			name:     "document_1",
			docID:    uuid.New().String(),
			shaHex:   "abc123def456",
			timestamp: time.Now().UTC(),
		},
		{
			name:     "document_2",
			docID:    uuid.New().String(),
			shaHex:   "xyz789ghi012",
			timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:     "document_3",
			docID:    uuid.New().String(),
			shaHex:   "very_long_sha256_hash_that_should_work_fine",
			timestamp: time.Now().UTC().Add(-24 * time.Hour),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Signer
			jws, err := service.SignEvidence(tc.docID, tc.shaHex, tc.timestamp)
			require.NoError(t, err)
			assert.NotEmpty(t, jws)

			// Vérifier
			evidence, err := service.VerifyEvidence(jws)
			require.NoError(t, err)
			assert.Equal(t, tc.docID, evidence.DocumentID)
			assert.Equal(t, tc.shaHex, evidence.Sha256)
			assert.WithinDuration(t, tc.timestamp, evidence.Timestamp, time.Second)
		})
	}
}

// TestJWS_Format teste le format JWS (3 parties séparées par des points)
func TestJWS_Format(t *testing.T) {
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "format-test")
	require.NoError(t, err)

	jws, err := service.SignEvidence(uuid.New().String(), "test123", time.Now())
	require.NoError(t, err)

	parts := splitJWS(jws)
	assert.Equal(t, 3, len(parts), "JWS should have exactly 3 parts")

	// Vérifier que chaque partie est base64url valide
	for i, part := range parts {
		_, err := base64.RawURLEncoding.DecodeString(part)
		assert.NoError(t, err, "Part %d should be valid base64url", i+1)
	}
}

// TestJWKS_Structure teste la structure du JWKS généré
func TestJWKS_Structure(t *testing.T) {
	privateKeyPath, publicKeyPath, cleanup := setupTestKeys(t)
	defer cleanup()

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "jwks-test-kid")
	require.NoError(t, err)

	jwks, err := service.CurrentJWKS()
	require.NoError(t, err)

	jwksStr := string(jwks)
	
	// Vérifier les champs requis (avec indentation JSON possible)
	assert.Contains(t, jwksStr, `"keys"`)
	assert.Contains(t, jwksStr, `"kty"`)
	assert.Contains(t, jwksStr, `"RSA"`)
	assert.Contains(t, jwksStr, `"jwks-test-kid"`)
	assert.Contains(t, jwksStr, `"use"`)
	assert.Contains(t, jwksStr, `"sig"`)
	assert.Contains(t, jwksStr, `"alg"`)
	assert.Contains(t, jwksStr, `"RS256"`)
	assert.Contains(t, jwksStr, `"n"`) // modulus
	assert.Contains(t, jwksStr, `"e"`) // exponent
}

// splitJWS divise un JWS en ses 3 parties
func splitJWS(jws string) []string {
	parts := make([]string, 0, 3)
	current := ""
	for _, char := range jws {
		if char == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

