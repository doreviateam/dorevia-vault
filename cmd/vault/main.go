package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/doreviateam/dorevia-vault/internal/auth"
	"github.com/doreviateam/dorevia-vault/internal/config"
	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/doreviateam/dorevia-vault/internal/metrics"
	"github.com/doreviateam/dorevia-vault/internal/middleware"
	"github.com/doreviateam/dorevia-vault/internal/services"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/doreviateam/dorevia-vault/internal/webhooks"
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

	// Initialisation de l'authentification (Sprint 5 Phase 5.2)
	var authService *auth.AuthService
	var rbacService *auth.RBACService
	if cfg.AuthEnabled {
		// Charger la clé publique JWT
		// Si JWTPublicKeyPath est configuré, on le charge
		// Sinon, on utilise la clé publique JWS si disponible (pour compatibilité)
		var jwtPublicKey *rsa.PublicKey
		if cfg.JWTPublicKeyPath != "" {
			// TODO: Charger depuis fichier ou Vault
			// Pour l'instant, on utilise la clé publique JWS si disponible
			if jwsService != nil {
				// Note: On devrait avoir une méthode GetPublicKey() dans Service
				// Pour l'instant, on peut utiliser la même clé que JWS
			}
		} else if jwsService != nil && cfg.JWTEnabled {
			// Utiliser la clé publique JWS comme clé publique JWT par défaut
			// Cela permet d'utiliser les mêmes clés pour JWS et JWT
			// TODO: Extraire la clé publique depuis jwsService
		}

		// Créer le service RBAC
		rbacService = auth.NewRBACService()

		// Créer le service d'authentification
		authCfg := auth.AuthConfig{
			JWTPublicKey:  jwtPublicKey,
			APIKeys:       make(map[string]*auth.APIKey), // TODO: Charger depuis config/DB
			JWTEnabled:    cfg.JWTEnabled,
			APIKeyEnabled: cfg.APIKeyEnabled,
			Logger:        *log,
		}
		authService = auth.NewAuthService(authCfg)
		log.Info().Msg("Authentication enabled")
	} else {
		log.Info().Msg("Authentication disabled (AUTH_ENABLED=false)")
	}

	// Initialisation des webhooks (Sprint 5 Phase 5.3)
	var webhookManager *webhooks.Manager
	if cfg.WebhooksEnabled {
		// Créer la queue Redis
		queueCfg := webhooks.QueueConfig{
			RedisURL:  cfg.WebhooksRedisURL,
			QueueName: "dorevia:webhooks",
			Logger:    *log,
		}
		queue, err := webhooks.NewQueue(queueCfg)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize webhook queue, webhooks disabled")
		} else {
			// Créer le worker
			workerCfg := webhooks.WorkerConfig{
				Queue:     queue,
				SecretKey: cfg.WebhooksSecretKey,
				Workers:   cfg.WebhooksWorkers,
				Logger:    *log,
			}
			worker := webhooks.NewWorker(workerCfg)

			// Parser les URLs webhooks depuis la config
			webhookURLs := webhooks.ParseWebhookURLs(cfg.WebhooksURLs)

			// Créer le manager
			managerCfg := webhooks.ManagerConfig{
				Queue:       queue,
				Worker:      worker,
				WebhookURLs: webhookURLs,
				Logger:      *log,
			}
			webhookManager = webhooks.NewManager(managerCfg)

			// Démarrer les workers
			ctx := context.Background()
			webhookManager.Start(ctx)
			defer webhookManager.Stop()

			log.Info().
				Int("workers", cfg.WebhooksWorkers).
				Int("events", len(webhookURLs)).
				Msg("Webhooks enabled")
		}
	} else {
		log.Info().Msg("Webhooks disabled (WEBHOOKS_ENABLED=false)")
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
	// 6. Prometheus : métriques HTTP par route/méthode/code
	app.Use(middleware.PrometheusMiddleware())
	// 7. RateLimit : limite en dernier (après métriques)
	app.Use(middleware.RateLimit())

	// Enregistrement des routes de base
	app.Get("/", handlers.Home)
	app.Get("/health", handlers.Health)
	app.Get("/health/detailed", handlers.DetailedHealthHandler(db, cfg.StorageDir, jwsService))
	app.Get("/health/live", handlers.HealthLive)
	app.Get("/health/ready", handlers.DetailedHealthHandler(db, cfg.StorageDir, jwsService)) // Réutilise detailed pour readiness
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
		auditGroup := app.Group("/audit")
		// Protection avec authentification et permission audit:read (Sprint 5 Phase 5.2)
		if authService != nil && rbacService != nil {
			auditGroup.Use(auth.AuthMiddleware(authService, *log))
			auditGroup.Use(auth.RequirePermission(rbacService, auth.PermissionReadAudit, *log))
		}
		auditGroup.Get("/export", handlers.AuditExportHandler(auditLogger, log))
		auditGroup.Get("/dates", handlers.AuditDatesHandler(auditLogger, log))
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
		// Routes publiques (sans authentification)
		app.Get("/dbhealth", handlers.DBHealthHandler(db))

		// Routes protégées (Sprint 5 Phase 5.2)
		apiGroup := app.Group("/api/v1")
		if authService != nil && rbacService != nil {
			apiGroup.Use(auth.AuthMiddleware(authService, *log))
		}

		// Routes documents (permission documents:read)
		docGroup := app.Group("/documents")
		if authService != nil && rbacService != nil {
			docGroup.Use(auth.AuthMiddleware(authService, *log))
			docGroup.Use(auth.RequirePermission(rbacService, auth.PermissionReadDocuments, *log))
		}
		docGroup.Get("", handlers.DocumentsListHandler(db))
		docGroup.Get("/:id", handlers.DocumentByIDHandler(db))

		// Route download (permission documents:read)
		downloadGroup := app.Group("/download")
		if authService != nil && rbacService != nil {
			downloadGroup.Use(auth.AuthMiddleware(authService, *log))
			downloadGroup.Use(auth.RequirePermission(rbacService, auth.PermissionReadDocuments, *log))
		}
		downloadGroup.Get("/:id", handlers.DownloadHandler(db))

		// Route upload (permission documents:write)
		uploadGroup := app.Group("/upload")
		if authService != nil && rbacService != nil {
			uploadGroup.Use(auth.AuthMiddleware(authService, *log))
			uploadGroup.Use(auth.RequirePermission(rbacService, auth.PermissionWriteDocuments, *log))
		}
		uploadGroup.Post("", handlers.UploadHandler(db, cfg.StorageDir))

		// Route Sprint 1 : Endpoint d'ingestion Odoo (permission documents:write)
		invoicesGroup := apiGroup.Group("/invoices")
		if rbacService != nil {
			invoicesGroup.Use(auth.RequirePermission(rbacService, auth.PermissionWriteDocuments, *log))
		}
		invoicesGroup.Post("", handlers.InvoicesHandler(db, cfg.StorageDir, jwsService, &cfg, log, auditLogger, webhookManager))
		invoicesGroup.Get("", handlers.GetInvoice) // 405 Method Not Allowed pour GET

		// Route Sprint 6 : Endpoint POS tickets (permission documents:write)
		posTicketsGroup := apiGroup.Group("/pos-tickets")
		if rbacService != nil {
			posTicketsGroup.Use(auth.RequirePermission(rbacService, auth.PermissionWriteDocuments, *log))
		}
		// Initialiser le service POS si DB et JWS sont disponibles
		if db != nil && jwsService != nil {
			// Créer le repository
			repo := storage.NewPostgresRepository(db.Pool, log)
			// Créer le service ledger
			ledgerService := ledger.NewService()
			// Créer le signer (adaptateur depuis jwsService)
			signer := crypto.NewLocalSigner(jwsService)
			// Créer le service POS
			posTicketsService := services.NewPosTicketsService(repo, ledgerService, signer)
			// Enregistrer les routes
			posTicketsGroup.Post("", handlers.PosTicketsHandler(posTicketsService, &cfg, log))
			posTicketsGroup.Get("", handlers.GetPosTicket) // 405 Method Not Allowed pour GET
			log.Info().Msg("POS tickets endpoint enabled: /api/v1/pos-tickets")
		} else {
			log.Warn().Msg("POS tickets endpoint disabled (requires DB and JWS)")
		}

		// Route Sprint 2 : Export ledger (permission ledger:read)
		ledgerGroup := apiGroup.Group("/ledger")
		if rbacService != nil {
			ledgerGroup.Use(auth.RequirePermission(rbacService, auth.PermissionReadLedger, *log))
		}
		ledgerGroup.Get("/export", handlers.LedgerExportHandler(db, log))

		// Route Sprint 3 Phase 3 : Vérification intégrité (permission documents:verify)
		verifyGroup := apiGroup.Group("/ledger/verify")
		if rbacService != nil {
			verifyGroup.Use(auth.RequirePermission(rbacService, auth.PermissionVerifyDocuments, *log))
		}
		verifyGroup.Get("/:document_id", handlers.VerifyHandler(db, jwsService, log, auditLogger, webhookManager))

		log.Info().Msg("Database routes enabled: /dbhealth, /upload, /documents, /documents/:id, /download/:id, /api/v1/invoices, /api/v1/pos-tickets, /api/v1/ledger/export, /api/v1/ledger/verify/:document_id")
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
