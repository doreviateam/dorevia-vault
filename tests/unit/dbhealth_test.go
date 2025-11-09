package unit

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// TestDBHealthHandlerWithoutDB teste le handler sans DB configurée
func TestDBHealthHandlerWithoutDB(t *testing.T) {
	app := fiber.New()
	app.Get("/dbhealth", handlers.DBHealthHandler(nil))

	req := httptest.NewRequest("GET", "/dbhealth", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "error", result["status"])
	assert.Equal(t, "Database not configured", result["message"])
}

