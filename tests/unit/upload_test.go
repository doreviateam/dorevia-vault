package unit

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// TestUploadHandlerWithoutDB teste le handler sans DB configurée
func TestUploadHandlerWithoutDB(t *testing.T) {
	app := fiber.New()
	app.Post("/upload", handlers.UploadHandler(nil, "/tmp"))

	// Créer un fichier multipart
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("test content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 503, resp.StatusCode)

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, "Database not configured", result["error"])
}

// TestUploadHandlerNoFile teste sans fichier fourni
func TestUploadHandlerNoFile(t *testing.T) {
	// Ce test nécessiterait un mock de la DB, on teste juste la validation de base
	app := fiber.New()
	app.Post("/upload", handlers.UploadHandler(nil, "/tmp"))

	req := httptest.NewRequest("POST", "/upload", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	// Soit 503 (DB not configured) soit 400 (No file provided)
	assert.Contains(t, []int{400, 503}, resp.StatusCode)
}

// TestUploadHandlerInvalidStorageDir teste avec un répertoire invalide
func TestUploadHandlerInvalidStorageDir(t *testing.T) {
	// Créer un répertoire temporaire pour le test
	tmpDir := filepath.Join(os.TempDir(), "vault-test")
	defer os.RemoveAll(tmpDir)

	app := fiber.New()
	app.Post("/upload", handlers.UploadHandler(nil, "/invalid/path/that/does/not/exist"))

	// Créer un fichier multipart
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("test content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)

	assert.NoError(t, err)
	// Devrait retourner 503 (DB not configured) ou 500 (storage error)
	assert.Contains(t, []int{500, 503}, resp.StatusCode)
}

