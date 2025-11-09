package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/doreviateam/dorevia-vault/internal/config"
	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/doreviateam/dorevia-vault/internal/middleware"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/doreviateam/dorevia-vault/pkg/logger"
	"github.com/gofiber/fiber/v2"
)

func main() {
	// Chargement de la configuration
	cfg := config.LoadOrDie()

	// Initialisation du logger structuré
	log := logger.New(cfg.LogLevel)

	// Initialisation de la connexion PostgreSQL (optionnelle)
	var db *storage.DB
	if cfg.DatabaseURL != "" {
		ctx := context.Background()
		var err error
		db, err = storage.NewDB(ctx, cfg.DatabaseURL, log)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to connect to database")
		}
		defer db.Close()
		log.Info().Msg("PostgreSQL connection established")
	} else {
		log.Warn().Msg("DATABASE_URL not configured, database features disabled")
	}

	// Initialisation de l'application Fiber
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			log.Error().
				Err(err).
				Int("status", code).
				Str("path", c.Path()).
				Msg("Request error")
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middlewares globaux
	app.Use(middleware.Logger(log))
	app.Use(middleware.CORS())
	app.Use(middleware.RateLimit())

	// Enregistrement des routes de base
	app.Get("/", handlers.Home)
	app.Get("/health", handlers.Health)
	app.Get("/version", handlers.Version)

	// Routes avec base de données (si configurée)
	if db != nil {
		app.Get("/dbhealth", handlers.DBHealthHandler(db))
		app.Post("/upload", handlers.UploadHandler(db, cfg.StorageDir))
		app.Get("/documents", handlers.DocumentsListHandler(db))
		app.Get("/documents/:id", handlers.DocumentByIDHandler(db))
		app.Get("/download/:id", handlers.DownloadHandler(db))
		log.Info().Msg("Database routes enabled: /dbhealth, /upload, /documents, /documents/:id, /download/:id")
	}

	// Gestion de l'arrêt propre
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		port := cfg.Port
		log.Info().
			Str("port", port).
			Str("log_level", cfg.LogLevel).
			Str("storage_dir", cfg.StorageDir).
			Bool("database_enabled", db != nil).
			Msg("Starting Dorevia Vault API server")

		if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Attendre le signal d'arrêt
	<-quit
	log.Info().Msg("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	}
	log.Info().Msg("Server stopped")
}
