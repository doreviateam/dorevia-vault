package unit

import (
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/models"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestDocumentModel teste la structure Document
func TestDocumentModel(t *testing.T) {
	doc := models.Document{
		ID:          uuid.New(),
		Filename:    "test.pdf",
		ContentType: "application/pdf",
		SizeBytes:   12345,
		SHA256Hex:   "abc123",
		StoredPath:  "/path/to/file",
		CreatedAt:   time.Now(),
	}

	assert.NotEqual(t, uuid.Nil, doc.ID)
	assert.Equal(t, "test.pdf", doc.Filename)
	assert.Equal(t, "application/pdf", doc.ContentType)
	assert.Equal(t, int64(12345), doc.SizeBytes)
}

// TestDocumentQuery teste la structure DocumentQuery
func TestDocumentQuery(t *testing.T) {
	now := time.Now()
	query := models.DocumentQuery{
		Page:     1,
		Limit:    20,
		Search:   "test",
		Type:     "application/pdf",
		DateFrom: &now,
		DateTo:   &now,
	}

	assert.Equal(t, 1, query.Page)
	assert.Equal(t, 20, query.Limit)
	assert.Equal(t, "test", query.Search)
	assert.Equal(t, "application/pdf", query.Type)
	assert.NotNil(t, query.DateFrom)
	assert.NotNil(t, query.DateTo)
}

// TestPaginationResponse teste la structure PaginationResponse
func TestPaginationResponse(t *testing.T) {
	pagination := models.PaginationResponse{
		Page:  1,
		Limit: 20,
		Total: 100,
		Pages: 5,
	}

	assert.Equal(t, 1, pagination.Page)
	assert.Equal(t, 20, pagination.Limit)
	assert.Equal(t, 100, pagination.Total)
	assert.Equal(t, 5, pagination.Pages)
}

// TestCalculatePages teste la fonction CalculatePages
func TestCalculatePages(t *testing.T) {
	// Cas normal
	assert.Equal(t, 5, storage.CalculatePages(100, 20))
	assert.Equal(t, 6, storage.CalculatePages(101, 20)) // Arrondi vers le haut
	assert.Equal(t, 1, storage.CalculatePages(0, 20))   // Minimum 1 page
	assert.Equal(t, 1, storage.CalculatePages(10, 0))   // Limit 0 = 1 page
	assert.Equal(t, 1, storage.CalculatePages(0, 0))    // Cas limite
}

