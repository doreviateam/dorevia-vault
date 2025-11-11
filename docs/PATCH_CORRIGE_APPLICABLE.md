# 🔧 Patch Corrigé — Intégration avec Code Existant

**Date** : Janvier 2025  
**Basé sur** : `vault_enhancements_fixed.patch`  
**Adaptations** : Corrections pour intégration avec code existant

---

## ⚠️ Problèmes Identifiés dans le Patch Original

1. **HealthDetailed** : Le patch référence `HealthDetailed` mais le code utilise `DetailedHealthHandler`
2. **VersionPayload** : Syntaxe incorrecte (point-virgule au lieu de retour à la ligne)
3. **DownloadHandler** : Le patch modifie `DownloadDocument` mais le code utilise `DownloadHandler`
4. **InvoicesHandler** : Le patch modifie `PostInvoice` mais le code utilise `InvoicesHandler`

---

## ✅ Patch Corrigé et Adapté

### 1. buildinfo.go (nouveau fichier)

```go
package buildinfo

// Ces valeurs sont surchargées au build via -ldflags.
var (
    Version = "0.0.1"  // -X github.com/doreviateam/dorevia-vault/internal/buildinfo.Version=1.3.0
    Commit  = "dev"    // -X github.com/doreviateam/dorevia-vault/internal/buildinfo.Commit=$(git rev-parse --short HEAD)
    BuiltAt = "unknown" // -X github.com/doreviateam/dorevia-vault/internal/buildinfo.BuiltAt=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    Schema  = "unknown" // -X github.com/doreviateam/dorevia-vault/internal/buildinfo.Schema=2025-11-11_0012
)

type VersionPayload struct {
    Version string `json:"version"`
    Commit  string `json:"commit"`
    BuiltAt string `json:"built_at"`
    Schema  string `json:"schema"`
}
```

### 2. version.go (modifié)

```go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "github.com/doreviateam/dorevia-vault/internal/buildinfo"
)

// Version retourne la version enrichie du service
func Version(c *fiber.Ctx) error {
    return c.JSON(buildinfo.VersionPayload{
        Version: buildinfo.Version,
        Commit:  buildinfo.Commit,
        BuiltAt: buildinfo.BuiltAt,
        Schema:  buildinfo.Schema,
    })
}
```

### 3. health_ready.go (nouveau fichier)

```go
package handlers

import (
    "github.com/gofiber/fiber/v2"
)

// HealthLive gère l'endpoint GET /health/live (liveness probe)
// Vérifie simplement que le processus répond
func HealthLive(c *fiber.Ctx) error {
    return c.SendStatus(fiber.StatusOK)
}

// HealthReady gère l'endpoint GET /health/ready (readiness probe)
// Note: Dans main.go, on utilisera DetailedHealthHandler pour la vraie vérification
// Cette fonction est un wrapper léger si nécessaire
func HealthReady(c *fiber.Ctx) error {
    // Par défaut, on retourne ready
    // Dans main.go, on peut directement utiliser DetailedHealthHandler
    return c.JSON(fiber.Map{
        "status": "ready",
    })
}
```

### 4. download.go (modifié)

```go
// ... code existant jusqu'à la ligne 55 ...

// Définir les headers pour le téléchargement
c.Set("Content-Type", doc.ContentType)
c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, doc.Filename))
c.Set("Content-Length", fmt.Sprintf("%d", doc.SizeBytes))

// Ajouter ETag basé sur SHA256 pour cache HTTP
if doc.SHA256Hex != "" {
    etag := fmt.Sprintf(`"%s"`, doc.SHA256Hex)
    c.Set("ETag", etag)
    
    // Vérifier If-None-Match pour 304 Not Modified
    if match := c.Get("If-None-Match"); match == etag {
        return c.SendStatus(fiber.StatusNotModified)
    }
}

// Envoyer le fichier
return c.SendFile(doc.StoredPath)
```

### 5. invoices.go (modifié)

```go
// ... code existant ...

// GetInvoice gère GET /api/v1/invoices -> 405 Method Not Allowed
// Retourne une erreur claire avec header Allow: POST
func GetInvoice(c *fiber.Ctx) error {
    return c.Status(fiber.StatusMethodNotAllowed).
        Set("Allow", "POST").
        JSON(fiber.Map{
            "error": "Method Not Allowed",
            "message": "Only POST method is allowed for /api/v1/invoices",
        })
}
```

### 6. http_metrics.go (nouveau fichier)

```go
package middleware

import (
    "fmt"
    "github.com/gofiber/fiber/v2"
    "github.com/prometheus/client_golang/prometheus"
)

var httpRequestsTotal = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "vault_http_requests_total",
        Help: "Total des requêtes HTTP par route/méthode/code",
    },
    []string{"route", "method", "code"},
)

// safeRegister évite le panic si la métrique est déjà enregistrée
func safeRegister(c prometheus.Collector) {
    if err := prometheus.Register(c); err != nil {
        if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
            return
        }
        panic(err)
    }
}

func init() {
    safeRegister(httpRequestsTotal)
}

// PrometheusMiddleware comptabilise chaque requête par route/méthode/code
func PrometheusMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        err := c.Next()
        code := c.Response().StatusCode()
        routePath := c.Route().Path
        if routePath == "" {
            routePath = c.Path() // Fallback si Route().Path est vide
        }
        httpRequestsTotal.WithLabelValues(routePath, c.Method(), fmt.Sprint(code)).Inc()
        return err
    }
}
```

### 7. main.go (modifié)

```go
// ... imports existants ...
import (
    // ... autres imports ...
    "github.com/doreviateam/dorevia-vault/internal/middleware"
)

// ... code existant jusqu'aux middlewares ...

// Middlewares globaux (ordre important)
app.Use(recover.New(recover.Config{
    EnableStackTrace: true,
}))
app.Use(helmet.New())
app.Use(requestid.New())
app.Use(middleware.Logger(log))
app.Use(middleware.CORS())
app.Use(middleware.PrometheusMiddleware()) // ← NOUVEAU
app.Use(middleware.RateLimit())

// Enregistrement des routes de base
app.Get("/", handlers.Home)
app.Get("/health", handlers.Health)
app.Get("/health/detailed", handlers.DetailedHealthHandler(db, cfg.StorageDir, jwsService))
app.Get("/health/live", handlers.HealthLive)  // ← NOUVEAU
app.Get("/health/ready", handlers.DetailedHealthHandler(db, cfg.StorageDir, jwsService)) // ← NOUVEAU (réutilise detailed)
app.Get("/version", handlers.Version) // ← MODIFIÉ (utilise buildinfo)

// ... routes existantes ...

// Dans le groupe /api/v1/invoices, ajouter :
if rbacService != nil {
    invoicesGroup.Use(auth.RequirePermission(rbacService, auth.PermissionWriteDocuments, *log))
}
invoicesGroup.Post("", handlers.InvoicesHandler(db, cfg.StorageDir, jwsService, &cfg, log, auditLogger, webhookManager))
invoicesGroup.Get("", handlers.GetInvoice) // ← NOUVEAU (405 Method Not Allowed)
```

---

## 🔨 Commandes de Build

```bash
# Build avec ldflags pour injecter les valeurs
go build -ldflags "\
  -X github.com/doreviateam/dorevia-vault/internal/buildinfo.Version=1.3.0 \
  -X github.com/doreviateam/dorevia-vault/internal/buildinfo.Commit=$(git rev-parse --short HEAD) \
  -X github.com/doreviateam/dorevia-vault/internal/buildinfo.BuiltAt=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -X github.com/doreviateam/dorevia-vault/internal/buildinfo.Schema=$(date -u +%Y%m%d)_0012" \
  -o bin/vault ./cmd/vault
```

---

## ✅ Checklist d'Application

- [ ] Créer `internal/buildinfo/buildinfo.go`
- [ ] Modifier `internal/handlers/version.go`
- [ ] Créer `internal/handlers/health_ready.go`
- [ ] Modifier `internal/handlers/download.go` (ajouter ETag)
- [ ] Modifier `internal/handlers/invoices.go` (ajouter GetInvoice)
- [ ] Créer `internal/middleware/http_metrics.go`
- [ ] Modifier `cmd/vault/main.go` (ajouter routes et middleware)
- [ ] Tester le build avec ldflags
- [ ] Vérifier que `/version` retourne les bonnes valeurs
- [ ] Vérifier que `/health/live` et `/health/ready` fonctionnent
- [ ] Vérifier que GET `/api/v1/invoices` retourne 405
- [ ] Vérifier que download avec ETag fonctionne

---

**Document créé le** : Janvier 2025

