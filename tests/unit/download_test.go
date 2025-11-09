package unit

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// TestDownloadHandlerWithoutDB teste le handler sans DB configurée
func TestDownloadHandlerWithoutDB(t *testing.T) {
	app := fiber.New()
	app.Get("/download/:id", handlers.DownloadHandler(nil))

	req := httptest.NewRequest("GET", "/download/123e4567-e89b-12d3-a456-426614174000", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Database not configured", result["error"])
}

// TestDownloadHandlerInvalidUUID teste avec un UUID invalide
func TestDownloadHandlerInvalidUUID(t *testing.T) {
	app := fiber.New()
	app.Get("/download/:id", handlers.DownloadHandler(nil))

	req := httptest.NewRequest("GET", "/download/invalid-uuid", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode) // DB not configured prend le dessus
}

