# 🔧 Correction Route `/metrics` — Dorevia Vault

**Date** : 10 Novembre 2025  
**Version** : v1.2-dev (Sprint 4 Phase 4.1)  
**Statut** : ✅ **Résolu**

---

## 🎯 Problème Initial

La route `/metrics` retournait :
- **404 Not Found** ou **"Cannot GET /metrics"**
- **502 Bad Gateway** (après redémarrage)

---

## 🔍 Diagnostic

### Problème 1 : Package Adaptateur Incorrect

**Symptôme** : Route `/metrics` non fonctionnelle, "Empty reply from server"

**Cause** :
- Utilisation de `github.com/gofiber/adaptor/v2` (package externe obsolète)
- Incompatibilité avec Fiber v2.52.9

**Solution** :
```go
// ❌ Ancien
import "github.com/gofiber/adaptor/v2"

// ✅ Nouveau
import fiberadaptor "github.com/gofiber/fiber/v2/middleware/adaptor"
```

**Fichier modifié** : `cmd/vault/main.go`

---

### Problème 2 : Configuration CORS Invalide

**Symptôme** : Service crash au démarrage → 502 Bad Gateway

**Cause** :
- `AllowCredentials: true` + `AllowOrigins: "*"` = Configuration invalide
- Fiber v2.52.9 refuse cette combinaison pour des raisons de sécurité
- Le service redémarrait en boucle (restart=always)

**Logs d'erreur** :
```
panic: [CORS] Insecure setup, 'AllowCredentials' is set to true, and 'AllowOrigins' is set to a wildcard.
```

**Solution** :
```go
// ❌ Ancien
AllowCredentials: true,  // Incompatible avec AllowOrigins: "*"

// ✅ Nouveau
AllowCredentials: false, // Pas besoin de cookies pour /metrics
```

**Fichier modifié** : `internal/middleware/cors.go`

---

## ✅ Corrections Appliquées

### 1. Correction Package Adaptateur

**Fichier** : `cmd/vault/main.go`

```go
// Ligne 19
fiberadaptor "github.com/gofiber/fiber/v2/middleware/adaptor"

// Ligne 118
app.Get("/metrics", fiberadaptor.HTTPHandler(promhttp.Handler()))
```

**Note** : Route montée **AVANT** les blocs conditionnels (DB, JWS) pour être toujours accessible.

---

### 2. Correction Configuration CORS

**Fichier** : `internal/middleware/cors.go`

```go
func CORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: false, // Fix: Cannot use wildcard "*" with AllowCredentials=true
	})
}
```

---

## 📊 Résultats

### Tests de Vérification

```bash
# Test local
curl -s http://localhost:8080/metrics | head -20
# ✅ Fonctionne

# Test via domaine
curl -s https://vault.doreviateam.com/metrics | head -20
# ✅ Fonctionne
```

### Métriques Exposées

**17 métriques actives** :

#### Métriques Métier (Sprint 3)
- `documents_vaulted_total` (counter)
- `jws_signatures_total` (counter)
- `ledger_entries_total` (counter)
- `reconciliation_runs_total` (counter)
- `document_storage_duration_seconds` (histogram)
- `jws_signature_duration_seconds` (histogram)
- `ledger_append_duration_seconds` (histogram)
- `transaction_duration_seconds` (histogram)
- `ledger_size` (gauge)
- `storage_size_bytes` (gauge)
- `active_connections` (gauge)

#### Métriques Système (Sprint 4 Phase 4.1)
- `ledger_append_errors_total` (counter)
- `system_cpu_usage_percent` (gauge)
- `system_memory_usage_bytes` (gauge)
- `system_memory_total_bytes` (gauge)
- `system_disk_usage_bytes` (gauge)
- `system_disk_capacity_bytes` (gauge)

### État du Service

```bash
systemctl status dorevia-vault
# ✅ Active: active (running)
# ✅ Pas de crash
# ✅ Métriques collectées automatiquement (30s)
```

---

## 🧪 Validation

### Checklist de Vérification

- [x] ✅ Route `/metrics` accessible en local (`localhost:8080`)
- [x] ✅ Route `/metrics` accessible via domaine (`vault.doreviateam.com`)
- [x] ✅ Service stable (pas de crash au démarrage)
- [x] ✅ Toutes les métriques exposées (17 métriques)
- [x] ✅ Métriques système collectées automatiquement
- [x] ✅ Format Prometheus valide
- [x] ✅ Compatible avec Prometheus/Grafana

---

## 📝 Fichiers Modifiés

1. **`cmd/vault/main.go`**
   - Correction import adaptateur Prometheus
   - Route `/metrics` montée avant blocs conditionnels

2. **`internal/middleware/cors.go`**
   - Correction configuration CORS (`AllowCredentials: false`)

3. **`go.mod`** / **`go.sum`**
   - Nettoyage dépendances (package `adaptor/v2` retiré)

---

## 🔒 Bonnes Pratiques Appliquées

1. **Route `/metrics` toujours accessible** : Montée avant les blocs conditionnels (DB, JWS)
2. **Configuration CORS sécurisée** : Pas de wildcard avec credentials
3. **Adaptateur officiel Fiber** : Utilisation du middleware intégré
4. **Collecteur automatique** : Métriques système mises à jour toutes les 30s

---

## 🎯 Prochaines Étapes

- [ ] Configuration Prometheus scrape config
- [ ] Dashboard Grafana (Sprint 4 Phase 4.3)
- [ ] Alertes Prometheus (Sprint 4 Phase 4.3)

---

## 📚 Références

- [Fiber v2 Middleware Adaptor](https://github.com/gofiber/fiber/tree/v2/middleware/adaptor)
- [Prometheus Client Go](https://github.com/prometheus/client_golang)
- [Fiber CORS Security](https://docs.gofiber.io/api/middleware/cors)

---

**Document créé le** : 10 Novembre 2025  
**Auteur** : Doreviateam  
**Version** : v1.2-dev (Sprint 4 Phase 4.1)

