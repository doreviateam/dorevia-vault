package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// UploadHandler gère l'upload de fichiers
func UploadHandler(db *storage.DB, storageDir string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if db == nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": "Database not configured",
			})
		}

		// Récupérer le fichier
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "No file provided",
			})
		}

		// Ouvrir le fichier
		src, err := file.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to open uploaded file",
			})
		}
		defer src.Close()

		// Lire le contenu pour calculer le SHA256
		content, err := io.ReadAll(src)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to read file content",
			})
		}

		// Calculer le SHA256
		hash := sha256.Sum256(content)
		sha256Hex := hex.EncodeToString(hash[:])

		// Vérifier si le fichier existe déjà (par SHA256)
		var existingID uuid.UUID
		err = db.Pool.QueryRow(
			context.Background(),
			"SELECT id FROM documents WHERE sha256_hex = $1 LIMIT 1",
			sha256Hex,
		).Scan(&existingID)

		if err == nil {
			// Fichier déjà existant
			return c.JSON(fiber.Map{
				"id":          existingID.String(),
				"filename":    file.Filename,
				"size_bytes":  file.Size,
				"content_type": file.Header.Get("Content-Type"),
				"sha256_hex":  sha256Hex,
				"message":     "File already exists",
			})
		} else if err != pgx.ErrNoRows {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to check existing file",
			})
		}

		// Générer un UUID pour le document
		docID := uuid.New()

		// Créer le répertoire de stockage par date
		now := time.Now()
		datePath := filepath.Join(
			storageDir,
			fmt.Sprintf("%d", now.Year()),
			fmt.Sprintf("%02d", now.Month()),
			fmt.Sprintf("%02d", now.Day()),
		)

		if err := os.MkdirAll(datePath, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create storage directory",
			})
		}

		// Chemin de stockage
		storedPath := filepath.Join(datePath, fmt.Sprintf("%s-%s", docID.String(), file.Filename))

		// Sauvegarder le fichier
		if err := os.WriteFile(storedPath, content, 0644); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save file",
			})
		}

		// Enregistrer en base de données
		contentType := file.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		_, err = db.Pool.Exec(
			context.Background(),
			`INSERT INTO documents (id, filename, content_type, size_bytes, sha256_hex, stored_path)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			docID, file.Filename, contentType, file.Size, sha256Hex, storedPath,
		)

		if err != nil {
			// Nettoyer le fichier en cas d'erreur
			os.Remove(storedPath)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to save metadata to database",
			})
		}

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"id":          docID.String(),
			"filename":    file.Filename,
			"size_bytes":  file.Size,
			"content_type": contentType,
			"sha256_hex":  sha256Hex,
			"stored_path": storedPath,
			"uploaded_at": now.Format(time.RFC3339),
		})
	}
}

