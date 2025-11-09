package unit

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// TestHealthHandler teste le handler /health
func TestHealthHandler(t *testing.T) {
	app := fiber.New()
	app.Get("/health", handlers.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	assert.Equal(t, "ok", string(body))
}

// TestVersionHandler teste le handler /version
func TestVersionHandler(t *testing.T) {
	app := fiber.New()
	app.Get("/version", handlers.Version)

	req := httptest.NewRequest("GET", "/version", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "0.0.1", result["version"])
}

// TestHomeHandler teste le handler /
func TestHomeHandler(t *testing.T) {
	app := fiber.New()
	app.Get("/", handlers.Home)

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, resp.ContentLength)
	resp.Body.Read(body)
	assert.Contains(t, string(body), "Dorevia Vault API")
}

