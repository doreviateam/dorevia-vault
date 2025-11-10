# 🧩 Dorevia Vault — Sprint 3 Phase 2 (Helmet, RequestID, Metrics)

**Date** : Janvier 2025  
**Version** : v1.1‑dev (Sprint 3 Phase 2)  
**Basé sur** : `main.go` durci (Phase 1)  
**Statut** : ✅ Implémentation planifiée et validée

---

## 🎯 Objectif

Renforcer la **sécurité**, la **traçabilité** et l’**observabilité** du service Dorevia Vault :  
1. Ajout des middlewares **Helmet** et **RequestID**.  
2. Mise en place d’un endpoint `/metrics` compatible **Prometheus**.  
3. Préparation des métriques métier internes (phase suivante).

---

## 🔐 1. Middleware Helmet

### Problème
En‑têtes HTTP de sécurité non appliqués : risque d’injection ou de framing.

### Solution
```go
import fiberhelmet "github.com/gofiber/fiber/v2/middleware/helmet"

// Protection des en‑têtes
app.Use(fiberhelmet.New())
```

### Vérification
```bash
curl -i http://localhost:8080/health | grep -E "X-Frame-Options|X-Content-Type-Options"
```
→ Doit retourner :
```
X-Frame-Options: SAMEORIGIN
X-Content-Type-Options: nosniff
```

---

## 🧾 2. Middleware RequestID

### Problème
Aucune corrélation entre logs et requêtes.

### Solution
```go
import fiberrequestid "github.com/gofiber/fiber/v2/middleware/requestid"

// Ajoute X-Request-ID unique par requête
app.Use(fiberrequestid.New())
```

### Vérification
```bash
curl -i http://localhost:8080/health | grep X-Request-ID
```
→ Doit retourner un UUID unique.

### Intégration logger (optionnel)
Ajouter dans le middleware `Logger` ou directement dans le `ErrorHandler` :
```go
Str("request_id", c.Get("X-Request-ID"))
```

---

## 📊 3. Endpoint `/metrics` (Prometheus)

### Problème
Aucune métrique exposée pour le monitoring.

### Solution
```go
import fiberprometheus "github.com/gofiber/contrib/fiberprometheus"

prom := fiberprometheus.New("dorevia-vault")
prom.RegisterAt(app, "/metrics")
app.Use(prom.Middleware)
```

### Vérification
```bash
curl -s http://localhost:8080/metrics | head
```
→ Doit retourner :
```
# HELP fiber_http_requests_total Number of HTTP requests
fiber_http_requests_total{code="200",method="GET",path="/health"} 1
```

---

## 🧪 Tests et Contrôles

| Test | Commande | Résultat attendu |
|:--|:--|:--|
| **Build** | `go build ./cmd/vault` | ✅ Aucun warning |
| **Headers sécurité** | `curl -i /health` | ✅ Présence Helmet |
| **RequestID** | `curl -i /health` | ✅ UUID dans headers |
| **Metrics** | `curl -s /metrics` | ✅ Exposition Prometheus |
| **Logs corrélés** | `journalctl -u dorevia-vault` | ✅ Présence RequestID |

---

## 🧱 Structure du code

```go
// Middlewares globaux
app.Use(recover.New())
app.Use(fiberhelmet.New())
app.Use(fiberrequestid.New())

// Prometheus metrics
prom := fiberprometheus.New("dorevia-vault")
prom.RegisterAt(app, "/metrics")
app.Use(prom.Middleware)
```

---

## 📊 Métriques métier (Phase 2+)

Préparer un package `internal/metrics/` avec :

- `documents_vaulted_total{source, status}` → counter  
- `ledger_append_duration_seconds` → histogram  
- `jws_signature_duration_seconds` → histogram  
- `reconciliation_runs_total{status}` → counter

### Exemple d’initialisation
```go
package metrics

var (
    DocumentsVaulted = promauto.NewCounterVec(
        prometheus.CounterOpts{Name: "documents_vaulted_total", Help: "Documents vaultés"},
        []string{"source", "status"},
    )
)
```

---

## 🧩 Roadmap Phase 2

| Étape | Description | Durée | Statut |
|:--|:--|:--|:--|
| **J4 matin** | Ajout Helmet + RequestID + tests headers | ½ jour | ⏳ |
| **J4 après‑midi** | Intégration `/metrics` | ½ jour | ⏳ |
| **J5 matin** | Ajout métriques métier | ½ jour | ⏳ |
| **J5 après‑midi** | Tests, validation, doc API | ½ jour | ⏳ |

Durée totale : **2 jours ouvrés**.

---

## ✅ Validation GO/NO GO

| Critère | État après patch | Évaluation |
|:--|:--|:--|
| Sécurité HTTP | ✅ Helmet actif | Conforme |
| Traçabilité requêtes | ✅ RequestID actif | Conforme |
| Observabilité | ✅ /metrics exposé | Conforme |
| Compatibilité existante | ✅ Aucun breaking change | Conforme |

**Verdict :** 🟢 GO — prêt à déployer Phase 2 sur environnement test.

---

**Document créé par :** GPT‑5 (AMOA Tech & Infra Doreviateam)  
**Date :** Janvier 2025  
**Licence :** MIT
