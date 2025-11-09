package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLedger_AppendFirstHash teste l'ajout du premier hash dans le ledger
func TestLedger_AppendFirstHash(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Nettoyer la table ledger pour le test
	_, err = pool.Exec(ctx, "DELETE FROM ledger")
	require.NoError(t, err)

	// Créer une transaction
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	// Test : Premier hash
	docID := uuid.New()
	shaHex := "abc123def456789"
	jws := "test.jws.token"

	hash, err := ledger.AppendLedger(ctx, tx, docID, shaHex, jws)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Vérifier que le hash est SHA256(shaHex) pour le premier
	expectedHash := sha256.Sum256([]byte(shaHex))
	expectedHashHex := hex.EncodeToString(expectedHash[:])
	assert.Equal(t, expectedHashHex, hash)

	// Commit pour valider
	err = tx.Commit(ctx)
	require.NoError(t, err)
}

// TestLedger_AppendChaining teste le chaînage des hash
func TestLedger_AppendChaining(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Nettoyer la table ledger
	_, err = pool.Exec(ctx, "DELETE FROM ledger")
	require.NoError(t, err)

	// Premier hash
	tx1, err := pool.Begin(ctx)
	require.NoError(t, err)
	docID1 := uuid.New()
	shaHex1 := "abc123"
	jws1 := "jws1"
	hash1, err := ledger.AppendLedger(ctx, tx1, docID1, shaHex1, jws1)
	require.NoError(t, err)
	err = tx1.Commit(ctx)
	require.NoError(t, err)

	// Deuxième hash (chaîné)
	tx2, err := pool.Begin(ctx)
	require.NoError(t, err)
	docID2 := uuid.New()
	shaHex2 := "def456"
	jws2 := "jws2"
	hash2, err := ledger.AppendLedger(ctx, tx2, docID2, shaHex2, jws2)
	require.NoError(t, err)
	err = tx2.Commit(ctx)
	require.NoError(t, err)

	// Vérifier que hash2 = SHA256(hash1 + shaHex2)
	combined := hash1 + shaHex2
	expectedHash2 := sha256.Sum256([]byte(combined))
	expectedHash2Hex := hex.EncodeToString(expectedHash2[:])
	assert.Equal(t, expectedHash2Hex, hash2)
}

// TestLedger_ExistsByDocumentID teste ExistsByDocumentID
func TestLedger_ExistsByDocumentID(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	// Nettoyer
	_, err = pool.Exec(ctx, "DELETE FROM ledger")
	require.NoError(t, err)

	// Ajouter un document
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	docID := uuid.New()
	hash, err := ledger.AppendLedger(ctx, tx, docID, "test123", "jws")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	err = tx.Commit(ctx)
	require.NoError(t, err)

	// Vérifier existence
	tx2, err := pool.Begin(ctx)
	require.NoError(t, err)
	exists, err := ledger.ExistsByDocumentID(ctx, tx2, docID)
	require.NoError(t, err)
	assert.True(t, exists)
	tx2.Rollback(ctx)

	// Vérifier non-existence
	tx3, err := pool.Begin(ctx)
	require.NoError(t, err)
	nonExistentID := uuid.New()
	exists, err = ledger.ExistsByDocumentID(ctx, tx3, nonExistentID)
	require.NoError(t, err)
	assert.False(t, exists)
	tx3.Rollback(ctx)
}

