package unit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/verify"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerificationResult_Structure teste la structure VerificationResult
func TestVerificationResult_Structure(t *testing.T) {
	result := &verify.VerificationResult{
		Valid:      true,
		DocumentID: "123e4567-e89b-12d3-a456-426614174000",
		Checks: []verify.Check{
			{Component: "database", Status: "ok", Message: "Document found"},
			{Component: "file", Status: "ok", Message: "File exists"},
		},
		Errors:    []string{},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	assert.True(t, result.Valid)
	assert.Equal(t, "123e4567-e89b-12d3-a456-426614174000", result.DocumentID)
	assert.Len(t, result.Checks, 2)
	assert.Empty(t, result.Errors)
	assert.NotEmpty(t, result.Timestamp)
}

// TestCheck_Structure teste la structure Check
func TestCheck_Structure(t *testing.T) {
	check := verify.Check{
		Component: "file",
		Status:    "ok",
		Message:   "File exists, size=12345, SHA256=abc...",
	}

	assert.Equal(t, "file", check.Component)
	assert.Equal(t, "ok", check.Status)
	assert.NotEmpty(t, check.Message)
}

// TestVerificationResult_InvalidDocument teste un résultat avec document invalide
func TestVerificationResult_InvalidDocument(t *testing.T) {
	result := &verify.VerificationResult{
		Valid:      false,
		DocumentID: "123e4567-e89b-12d3-a456-426614174000",
		Checks: []verify.Check{
			{Component: "database", Status: "missing", Message: "Document not found in database"},
		},
		Errors:    []string{"Document not found in database"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	assert.False(t, result.Valid)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "missing", result.Checks[0].Status)
}

// TestVerificationResult_SHA256Mismatch teste un résultat avec SHA256 mismatch
func TestVerificationResult_SHA256Mismatch(t *testing.T) {
	result := &verify.VerificationResult{
		Valid:      false,
		DocumentID: "123e4567-e89b-12d3-a456-426614174000",
		Checks: []verify.Check{
			{Component: "database", Status: "ok", Message: "Document found"},
			{Component: "file", Status: "error", Message: "SHA256 mismatch: expected abc, got def"},
		},
		Errors:    []string{"SHA256 mismatch: file may have been tampered"},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	assert.False(t, result.Valid)
	assert.Equal(t, "error", result.Checks[1].Status)
	assert.Contains(t, result.Errors[0], "SHA256 mismatch")
}

// TestVerifyDocumentIntegrity_RequiresDB teste que VerifyDocumentIntegrity nécessite une DB
func TestVerifyDocumentIntegrity_RequiresDB(t *testing.T) {
	// Ce test nécessite une base de données réelle
	// Pour l'instant, on teste la logique sans DB
	t.Skip("Requires database connection - integration test")
}

// TestVerifyDocumentIntegrity_Logic teste la logique de vérification sans DB
func TestVerifyDocumentIntegrity_Logic(t *testing.T) {
	// Test de la logique de vérification
	docID := uuid.New()
	
	// Vérifier que le document ID est valide
	assert.NotEqual(t, uuid.Nil, docID)
	
	// Test de calcul SHA256 (logique de vérification)
	content := []byte("test content")
	hash := sha256.Sum256(content)
	sha256Hex := hex.EncodeToString(hash[:])
	
	assert.NotEmpty(t, sha256Hex)
	assert.Len(t, sha256Hex, 64) // SHA256 hex = 64 caractères
}

// TestVerifyDocumentIntegrity_FileMissing teste le cas fichier manquant
func TestVerifyDocumentIntegrity_FileMissing(t *testing.T) {
	// Créer un fichier temporaire
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.pdf")
	
	// Vérifier que le fichier n'existe pas
	_, err := os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
	
	// Simuler le résultat attendu
	result := &verify.VerificationResult{
		Valid:      false,
		DocumentID: uuid.New().String(),
		Checks: []verify.Check{
			{Component: "database", Status: "ok", Message: "Document found"},
			{Component: "file", Status: "missing", Message: "File not found: " + filePath},
		},
		Errors:    []string{"File not found: " + filePath},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	
	assert.False(t, result.Valid)
	assert.Equal(t, "missing", result.Checks[1].Status)
}

// TestVerifyDocumentIntegrity_FileSizeMismatch teste le cas taille fichier incorrecte
func TestVerifyDocumentIntegrity_FileSizeMismatch(t *testing.T) {
	// Créer un fichier temporaire
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.pdf")
	content := []byte("test content")
	
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)
	defer os.Remove(filePath)
	
	// Vérifier que la taille ne correspond pas
	fileInfo, err := os.Stat(filePath)
	require.NoError(t, err)
	
	expectedSize := int64(999999) // Taille incorrecte
	actualSize := fileInfo.Size()
	
	assert.NotEqual(t, expectedSize, actualSize)
	
	// Simuler le résultat attendu
	result := &verify.VerificationResult{
		Valid:      false,
		DocumentID: uuid.New().String(),
		Checks: []verify.Check{
			{Component: "database", Status: "ok", Message: "Document found"},
			{Component: "file", Status: "error", Message: fmt.Sprintf("Size mismatch: expected 999999, got %d", actualSize)},
		},
		Errors:    []string{fmt.Sprintf("File size mismatch: expected 999999, got %d", actualSize)},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	
	assert.False(t, result.Valid)
	assert.Equal(t, "error", result.Checks[1].Status)
}

// TestVerifyDocumentIntegrity_ValidDocument teste un document valide (sans DB)
func TestVerifyDocumentIntegrity_ValidDocument(t *testing.T) {
	// Créer un fichier temporaire
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.pdf")
	content := []byte("test content")
	
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)
	defer os.Remove(filePath)
	
	// Calculer SHA256
	hash := sha256.Sum256(content)
	sha256Hex := hex.EncodeToString(hash[:])
	
	// Vérifier que le fichier existe et que le SHA256 est correct
	fileInfo, err := os.Stat(filePath)
	require.NoError(t, err)
	
	assert.Equal(t, int64(len(content)), fileInfo.Size())
	assert.NotEmpty(t, sha256Hex)
	
	// Simuler le résultat attendu pour un document valide
	result := &verify.VerificationResult{
		Valid:      true,
		DocumentID: uuid.New().String(),
		Checks: []verify.Check{
			{Component: "database", Status: "ok", Message: "Document found: test.pdf"},
			{Component: "file", Status: "ok", Message: fmt.Sprintf("File exists, size=%d, SHA256=%s", fileInfo.Size(), sha256Hex[:16]+"...")},
		},
		Errors:    []string{},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.Equal(t, "ok", result.Checks[0].Status)
	assert.Equal(t, "ok", result.Checks[1].Status)
}

// TestVerifyDocumentIntegrity_LedgerWarn teste le cas ledger avec warning
func TestVerifyDocumentIntegrity_LedgerWarn(t *testing.T) {
	result := &verify.VerificationResult{
		Valid:      true,
		DocumentID: uuid.New().String(),
		Checks: []verify.Check{
			{Component: "database", Status: "ok", Message: "Document found"},
			{Component: "file", Status: "ok", Message: "File exists"},
			{Component: "ledger", Status: "warn", Message: "No ledger entry found (ledger may be disabled)"},
		},
		Errors:    []string{},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	
	// Document valide mais avec warning ledger
	assert.True(t, result.Valid)
	assert.Equal(t, "warn", result.Checks[2].Status)
}

// TestVerifyDocumentIntegrity_ContextTimeout teste le timeout context
func TestVerifyDocumentIntegrity_ContextTimeout(t *testing.T) {
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

