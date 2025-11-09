package unit

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// TestDocumentsListHandlerWithoutDB teste le handler sans DB configurée
func TestDocumentsListHandlerWithoutDB(t *testing.T) {
	app := fiber.New()
	app.Get("/documents", handlers.DocumentsListHandler(nil))

	req := httptest.NewRequest("GET", "/documents", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Database not configured", result["error"])
}

// TestDocumentByIDHandlerWithoutDB teste le handler sans DB configurée
func TestDocumentByIDHandlerWithoutDB(t *testing.T) {
	app := fiber.New()
	app.Get("/documents/:id", handlers.DocumentByIDHandler(nil))

	req := httptest.NewRequest("GET", "/documents/123e4567-e89b-12d3-a456-426614174000", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Database not configured", result["error"])
}

// TestDocumentByIDHandlerInvalidUUID teste avec un UUID invalide
func TestDocumentByIDHandlerInvalidUUID(t *testing.T) {
	app := fiber.New()
	app.Get("/documents/:id", handlers.DocumentByIDHandler(nil))

	req := httptest.NewRequest("GET", "/documents/invalid-uuid", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode) // DB not configured prend le dessus
}


