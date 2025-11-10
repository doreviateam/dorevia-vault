# 🧭 Dorevia Vault — Recommandations `main.go` (Sprint 3)

**Date** : Novembre 2025  
**Version analysée** : v1.0 (post‑Sprint 2)  
**Objet** : Revue technique et préconisations d’amélioration du fichier `main.go` avant lancement Sprint 3 “Expert Edition”.

---

## 🎯 Objectif de cette revue

Le fichier `cmd/vault/main.go` est le **point d’entrée** du service Dorevia Vault.  
Cette revue vise à :

- évaluer la robustesse et la maintenabilité de la structure actuelle ;
- proposer les améliorations nécessaires pour **Sprint 3 : observabilité, sécurité, robustesse** ;
- fournir un **patch recommandé** prêt à intégrer.

---

## 🔍 Diagnostic du `main.go` actuel

### Points forts

| Catégorie | Observations |
|:--|:--|
| **Structure** | Config → DB → Crypto → Routes → Shutdown : clair, séquentiel. |
| **Lisibilité** | Code simple et explicite, peu de magie. |
| **Gestion d’erreurs** | Usage correct de `Fatal/Warn/Info`. |
| **Sécurité** | JWS optionnel avec contrôle `JWSRequired`. |
| **Extensibilité** | Ajout de routes facile via `handlers/*`. |

### Points perfectibles

| Axe | Constat | Impact |
|:--|:--|:--|
| **Durcissement serveur** | Middlewares essentiels manquants (`recover`, `helmet`, `requestid`) | Risque crash / en-têtes faibles |
| **Observabilité** | Pas d’endpoint `/metrics` Prometheus | Visibilité perf quasi nulle |
| **Disponibilité JWKS** | `/jwks.json` enregistré dans le bloc DB | JWKS indisponible si DB down |
| **Graceful shutdown** | Pas de timeout ni close DB contrôlé | Risque arrêt brutal |
| **Initialisation** | `STORAGE_DIR` non validé/créé au boot | Crash si répertoire absent |
| **Mode dégradé** | Pas de log explicite quand JWS est inactif | Diagnostic compliqué |

---

## 🧠 Préconisations techniques Sprint 3

### 1) Durcir le serveur Fiber

- `recover` pour capturer les panic runtime
- `helmet` pour renforcer les en‑têtes de sécurité
- `requestid` pour tracer chaque requête
- conserver le middleware `Logger` existant

### 2) Ajouter un endpoint Prometheus

- Dépendance : `github.com/gofiber/contrib/fiberprometheus`
- Exposer `/metrics` (HTTP metrics out‑of‑the‑box)
- Sprint 3 branchera ensuite les **métriques métier**

### 3) Rendre `/jwks.json` indépendant de la DB

- Enregistrer la route JWKS **en dehors** du bloc `if db != nil`
- Même sans DB, la vérification JWS doit rester disponible

### 4) Créer/valider `STORAGE_DIR` au boot

```go
if cfg.StorageDir == "" { log.Fatal().Msg("STORAGE_DIR not configured") }
if err := os.MkdirAll(cfg.StorageDir, 0o755); err != nil { log.Fatal().Err(err).Msg("cannot create storage_dir") }
```

### 5) Implémenter un **graceful shutdown**

- `context.WithTimeout(..., 10*time.Second)`
- `app.Shutdown()` puis fermeture contrôlée du pool DB

### 6) Journaliser le **mode dégradé** JWS

- Si `JWSEnabled=true` mais service non initialisé → log explicite “degraded mode”

---

## 🧩 Patch recommandé (drop‑in)

> **But** : rendre le `main.go` “Sprint 3‑ready” sans casser l’existant.

```go
// imports supplémentaires
fiberprometheus "github.com/gofiber/contrib/fiberprometheus"
fiberhelmet "github.com/gofiber/fiber/v2/middleware/helmet"
fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
fiberrequestid "github.com/gofiber/fiber/v2/middleware/requestid"
```

```go
// Vérifier / créer STORAGE_DIR
if cfg.StorageDir == "" {
    log.Fatal().Msg("STORAGE_DIR not configured")
}
if err := os.MkdirAll(cfg.StorageDir, 0o755); err != nil {
    log.Fatal().Err(err).Str("dir", cfg.StorageDir).Msg("failed to create STORAGE_DIR")
}
log.Info().Str("storage_dir", cfg.StorageDir).Msg("storage directory ready")
```

```go
// Middlewares de durcissement
app.Use(fiberrecover.New())   // anti‑panic
app.Use(fiberhelmet.New())    // headers sécurité
app.Use(fiberrequestid.New()) // X‑Request‑ID
```

```go
// Prometheus /metrics
prom := fiberprometheus.New("dorevia-vault")
prom.RegisterAt(app, "/metrics")
app.Use(prom.Middleware)
```

```go
// JWKS indépendant de la DB
if jwsService != nil {
    app.Get("/jwks.json", handlers.JWKSHandler(jwsService, log))
    log.Info().Msg("JWKS endpoint enabled: /jwks.json")
} else if cfg.JWSEnabled {
    log.Warn().Msg("JWS enabled but service not initialized → JWKS disabled (degraded)")
}
```

```go
// Graceful shutdown avec timeout
quit := make(chan os.Signal, 1)
signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
<-quit
log.Info().Msg("Shutting down server...")

shCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := app.Shutdown(); err != nil {
    log.Error().Err(err).Msg("error during server shutdown")
}

// fermer la DB proprement
if db != nil {
    done := make(chan struct{})
    go func() { db.Close(); close(done) }()
    select {
    case <-done:
    case <-shCtx.Done():
        log.Warn().Msg("timeout closing database pool")
    }
}

log.Info().Msg("Server stopped")
```

---

## 📊 Effets attendus

| Domaine | Amélioration |
|:--|:--|
| **Disponibilité** | Résilience (recover) + arrêt contrôlé |
| **Sécurité** | `helmet` + séparation JWKS/DB |
| **Observabilité** | `/metrics` prêt (HTTP metrics) |
| **Maintenance** | Traçabilité via `requestid` |
| **Souveraineté** | Dépendances limitées, auditables |

---

## ✅ Recommandation GO/NO GO

| Critère | État après patch | Évaluation |
|:--|:--|:--|
| Durcissement serveur | ✅ | Conforme prod |
| Observabilité | ✅ | Conforme (Prometheus prêt) |
| JWKS indépendant | ✅ | Conforme |
| Shutdown propre | ✅ | Conforme |
| Compatibilité existante | ✅ | Pas de breaking change |
| Tests | ⚙️ | Ajouter tests d’intégration /health & /metrics |

**Verdict :** 🟢 **GO – Intégrer ce patch comme base Sprint 3.**

---

### Notes complémentaires

- Penser à **documenter** `/metrics` et `/health/detailed` dans `docs/API_SPRINT3.md`.
- Préparer une **rule Grafana** pour alerte “mode dégradé JWS”.
- Étendre les métriques métier dans `internal/metrics/` (Sprint 3 J4–J6).

---

**Auteur** : GPT‑5 (Audit Technique & AMOA) — pour Doreviateam  
**Licence** : MIT — 2025
