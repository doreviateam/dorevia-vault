package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/doreviateam/dorevia-vault/internal/config"
	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/doreviateam/dorevia-vault/internal/metrics"
	"github.com/doreviateam/dorevia-vault/internal/middleware"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/doreviateam/dorevia-vault/pkg/logger"
	"github.com/gofiber/fiber/v2"
	fiberadaptor "github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Chargement de la configuration
	cfg := config.LoadOrDie()

	// Initialisation du logger structuré
	log := logger.New(cfg.LogLevel)

	// Validation et création du répertoire de stockage
	if cfg.StorageDir == "" {
		log.Fatal().Msg("STORAGE_DIR not configured")
	}
	if err := os.MkdirAll(cfg.StorageDir, 0755); err != nil {
		log.Fatal().Err(err).Str("dir", cfg.StorageDir).Msg("failed to create STORAGE_DIR")
	}
	log.Info().Str("storage_dir", cfg.StorageDir).Msg("storage directory ready")

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

	// Initialisation du service JWS (optionnel)
	var jwsService *crypto.Service
	if cfg.JWSEnabled && (cfg.JWSPrivateKeyPath != "" || cfg.JWSPrivateKeyBase64 != "") {
		var err error
		jwsService, err = crypto.NewService(cfg.JWSPrivateKeyPath, cfg.JWSPublicKeyPath, cfg.JWSKID)
		if err != nil {
			if cfg.JWSRequired {
				log.Fatal().Err(err).Msg("JWS required but initialization failed")
			}
			log.Warn().Err(err).Msg("JWS initialization failed, continuing without JWS")
		} else {
			log.Info().Str("kid", cfg.JWSKID).Msg("JWS service initialized")
		}
	} else if cfg.JWSEnabled {
		log.Warn().Msg("JWS_ENABLED=true but no key path configured → JWS disabled (degraded mode)")
	}

	// Initialisation du logger d'audit (Sprint 4 Phase 4.2)
	var auditLogger *audit.Logger
	if cfg.AuditDir != "" {
		auditCfg := audit.Config{
			AuditDir:      cfg.AuditDir,
			MaxBuffer:     1000,
			FlushInterval: 10 * time.Second,
			Logger:        *log,
		}
		var err error
		auditLogger, err = audit.NewLogger(auditCfg)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize audit logger, continuing without audit")
		} else {
			defer auditLogger.Close()
			log.Info().Str("audit_dir", cfg.AuditDir).Msg("Audit logger initialized")
		}
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
				Str("request_id", c.Get("X-Request-ID")). // Sprint 3 Phase 2 : Traçabilité requêtes
				Msg("Request error")
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middlewares globaux (ordre important)
	// 1. Recover : capture les panic runtime pour éviter crash
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	// 2. Helmet : ajoute headers sécurité HTTP (Sprint 3 Phase 2)
	app.Use(helmet.New())
	// 3. RequestID : génère ID unique par requête (Sprint 3 Phase 2)
	app.Use(requestid.New())
	// 4. Logger : peut maintenant utiliser RequestID
	app.Use(middleware.Logger(log))
	// 5. CORS : gère les en-têtes CORS
	app.Use(middleware.CORS())
	// 6. RateLimit : limite en dernier (après métriques)
	app.Use(middleware.RateLimit())

	// Enregistrement des routes de base
	app.Get("/", handlers.Home)
	app.Get("/health", handlers.Health)
	app.Get("/health/detailed", handlers.DetailedHealthHandler(db, cfg.StorageDir, jwsService))
	app.Get("/version", handlers.Version)

	// Route Prometheus /metrics (Sprint 3 Phase 2)
	// Expose les métriques HTTP standard Prometheus
	// IMPORTANT : Route montée AVANT les blocs conditionnels (DB, JWS) pour être toujours accessible
	app.Get("/metrics", fiberadaptor.HTTPHandler(promhttp.Handler()))

	// Sprint 4 Phase 4.1 : Démarrer le collecteur de métriques système
	// Mise à jour automatique toutes les 30 secondes
	metrics.StartSystemMetricsCollector(30 * time.Second)
	log.Info().Msg("System metrics collector started (30s interval)")

	// Routes audit (Sprint 4 Phase 4.2)
	// Accessibles même sans DB pour permettre l'audit du système
	if auditLogger != nil {
		app.Get("/audit/export", handlers.AuditExportHandler(auditLogger, log))
		app.Get("/audit/dates", handlers.AuditDatesHandler(auditLogger, log))
		log.Info().Msg("Audit endpoints enabled: /audit/export, /audit/dates")
	}

	// Initialisation exporteur Odoo (Sprint 4 Phase 4.3)
	var odooExporter *audit.OdooExporter
	if cfg.OdooURL != "" && cfg.OdooDatabase != "" {
		odooCfg := audit.OdooConfig{
			OdooURL:      cfg.OdooURL,
			OdooDatabase: cfg.OdooDatabase,
			OdooUser:     cfg.OdooUser,
			OdooPassword: cfg.OdooPassword,
			Logger:       *log,
		}
		odooExporter = audit.NewOdooExporter(odooCfg)
		log.Info().Str("odoo_url", cfg.OdooURL).Msg("Odoo exporter initialized")
	}

	// Route webhook alertes (Sprint 4 Phase 4.3)
	// Reçoit les alertes depuis Alertmanager
	// Accessible même sans exporteur Odoo (pour tests et monitoring)
	app.Post("/api/v1/alerts/webhook", handlers.AlertsWebhookHandler(odooExporter, log))
	if odooExporter != nil {
		log.Info().Msg("Alerts webhook endpoint enabled with Odoo export: /api/v1/alerts/webhook")
	} else {
		log.Info().Msg("Alerts webhook endpoint enabled (without Odoo export): /api/v1/alerts/webhook")
	}

	// Route JWKS indépendante de la DB (disponible même si DB down)
	// Sprint 3 : JWKS doit être accessible pour vérification JWS sans DB
	if jwsService != nil {
		app.Get("/jwks.json", handlers.JWKSHandler(jwsService, log))
		log.Info().Msg("JWKS endpoint enabled: /jwks.json")
	} else if cfg.JWSEnabled {
		log.Warn().Msg("JWS enabled but service not initialized → JWKS disabled (degraded)")
	}

	// Routes avec base de données (si configurée)
	if db != nil {
		app.Get("/dbhealth", handlers.DBHealthHandler(db))
		app.Post("/upload", handlers.UploadHandler(db, cfg.StorageDir))
		app.Get("/documents", handlers.DocumentsListHandler(db))
		app.Get("/documents/:id", handlers.DocumentByIDHandler(db))
		app.Get("/download/:id", handlers.DownloadHandler(db))

		// Route Sprint 1 : Endpoint d'ingestion Odoo
		app.Post("/api/v1/invoices", handlers.InvoicesHandler(db, cfg.StorageDir, jwsService, &cfg, log, auditLogger))

		// Route Sprint 2 : Export ledger
		app.Get("/api/v1/ledger/export", handlers.LedgerExportHandler(db, log))

		// Route Sprint 3 Phase 3 : Vérification intégrité
		app.Get("/api/v1/ledger/verify/:document_id", handlers.VerifyHandler(db, jwsService, log, auditLogger))

		log.Info().Msg("Database routes enabled: /dbhealth, /upload, /documents, /documents/:id, /download/:id, /api/v1/invoices, /api/v1/ledger/export, /api/v1/ledger/verify/:document_id")
	}

	// Gestion de l'arrêt propre avec timeout
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

	// Graceful shutdown avec timeout (10 secondes)
	shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Arrêter le serveur Fiber
	if err := app.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	}

	// Fermer la connexion DB proprement avec timeout
	if db != nil {
		done := make(chan struct{})
		go func() {
			db.Close()
			close(done)
		}()
		select {
		case <-done:
			log.Info().Msg("Database connection closed")
		case <-shCtx.Done():
			log.Warn().Msg("Timeout closing database pool")
		}
	}

	log.Info().Msg("Server stopped")
}
