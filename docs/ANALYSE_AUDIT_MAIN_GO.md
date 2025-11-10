# 📊 Analyse de l'Audit `main.go` — Dorevia Vault

**Date** : Janvier 2025  
**Document analysé** : `docs/audit_vault_main.go.md`  
**Version code actuel** : v1.0 (post-Sprint 2, Phase 1 Sprint 3 complétée)

---

## 🎯 Synthèse de l'Audit

L'audit propose **6 améliorations critiques** pour durcir le `main.go` avant Sprint 3 :
1. Durcir le serveur Fiber (recover, helmet, requestid)
2. Ajouter endpoint Prometheus `/metrics`
3. Rendre `/jwks.json` indépendant de la DB
4. Valider/créer `STORAGE_DIR` au boot
5. Implémenter graceful shutdown avec timeout
6. Journaliser explicitement le mode dégradé JWS

---

## ✅ État Actuel vs Recommandations

### 1. Durcissement Serveur ❌ **Non Implémenté**

| Middleware | État Actuel | Recommandation | Priorité |
|:-----------|:------------|:---------------|:---------|
| `recover` | ❌ Absent | ✅ Ajouter | **Haute** |
| `helmet` | ❌ Absent | ✅ Ajouter | **Haute** |
| `requestid` | ❌ Absent | ✅ Ajouter | **Moyenne** |
| `Logger` | ✅ Présent | ✅ Conserver | - |

**Impact** : Risque de crash non géré, en-têtes sécurité faibles, traçabilité limitée.

**Action** : ✅ **À implémenter** (Sprint 3 Phase 1 ou 2)

---

### 2. Endpoint Prometheus ⏳ **Planifié Phase 2**

| Élément | État Actuel | Recommandation | Priorité |
|:--------|:------------|:---------------|:---------|
| `/metrics` | ❌ Absent | ✅ Ajouter | **Haute** |
| `fiberprometheus` | ❌ Non installé | ✅ Installer | **Haute** |

**Impact** : Pas de visibilité performance (HTTP metrics).

**Action** : ✅ **Planifié Phase 2 (J4-J6)** — Cohérent avec le plan Sprint 3

**Note** : L'audit recommande d'ajouter les métriques HTTP de base maintenant, puis les métriques métier en Phase 2. C'est une bonne approche.

---

### 3. JWKS Indépendant de DB ⚠️ **Partiellement Implémenté**

| Élément | État Actuel | Recommandation | Impact |
|:--------|:------------|:---------------|:-------|
| Route `/jwks.json` | ⚠️ Dans bloc `if db != nil` | ✅ Hors bloc DB | **Moyen** |

**Code actuel** :
```go
// Routes avec base de données (si configurée)
if db != nil {
    // ...
    if jwsService != nil {
        app.Get("/jwks.json", handlers.JWKSHandler(jwsService, log))
    }
}
```

**Problème** : Si DB est down, JWKS n'est pas accessible même si JWS fonctionne.

**Action** : ✅ **À corriger** — Déplacer hors du bloc DB

---

### 4. Validation STORAGE_DIR ❌ **Non Implémenté**

| Élément | État Actuel | Recommandation | Impact |
|:--------|:------------|:---------------|:-------|
| Validation | ❌ Aucune | ✅ Vérifier/créer au boot | **Haute** |

**Code actuel** : `STORAGE_DIR` utilisé directement sans validation.

**Problème** : Crash au runtime si répertoire absent.

**Action** : ✅ **À implémenter** — Critique pour robustesse

---

### 5. Graceful Shutdown ⚠️ **Partiellement Implémenté**

| Élément | État Actuel | Recommandation | Impact |
|:--------|:------------|:---------------|:-------|
| `app.Shutdown()` | ✅ Présent | ✅ Conserver | - |
| Timeout shutdown | ❌ Absent | ✅ Ajouter (10s) | **Moyenne** |
| Close DB contrôlé | ⚠️ `defer db.Close()` | ✅ Timeout + goroutine | **Moyenne** |

**Code actuel** :
```go
<-quit
log.Info().Msg("Shutting down server...")
if err := app.Shutdown(); err != nil {
    log.Error().Err(err).Msg("Error during server shutdown")
}
```

**Problème** : Pas de timeout, `db.Close()` peut bloquer indéfiniment.

**Action** : ✅ **À améliorer** — Ajouter timeout et close DB contrôlé

---

### 6. Mode Dégradé JWS ⚠️ **Partiellement Implémenté**

| Élément | État Actuel | Recommandation | Impact |
|:--------|:------------|:---------------|:-------|
| Log JWS disabled | ⚠️ Générique | ✅ Explicite "degraded" | **Faible** |

**Code actuel** :
```go
} else if cfg.JWSEnabled {
    log.Warn().Msg("JWS_ENABLED=true but no key path configured, JWS disabled")
}
```

**Problème** : Message pas assez explicite pour diagnostic.

**Action** : ✅ **À améliorer** — Message plus explicite

---

## 📋 Plan d'Action Recommandé

### Priorité Haute (À faire maintenant)

1. ✅ **Validation STORAGE_DIR** — Critique pour robustesse
2. ✅ **JWKS indépendant de DB** — Améliore disponibilité
3. ✅ **Durcissement serveur** (recover, helmet) — Sécurité

### Priorité Moyenne (Phase 1 ou 2)

4. ✅ **Graceful shutdown amélioré** — Qualité production
5. ✅ **RequestID middleware** — Traçabilité
6. ✅ **Log mode dégradé explicite** — Diagnostic

### Priorité Basse (Phase 2)

7. ✅ **Endpoint Prometheus** — Déjà planifié Phase 2

---

## 🎯 Recommandation Finale

### ✅ **GO — Intégrer les améliorations prioritaires**

**Actions immédiates** (Phase 1 - J3) :
1. Validation/création `STORAGE_DIR` au boot
2. Déplacer `/jwks.json` hors du bloc DB
3. Ajouter middleware `recover` (anti-panic)
4. Améliorer graceful shutdown avec timeout

**Actions Phase 2** (J4-J6) :
5. Ajouter middleware `helmet` (sécurité headers)
6. Ajouter middleware `requestid` (traçabilité)
7. Ajouter endpoint `/metrics` Prometheus (HTTP metrics)
8. Améliorer log mode dégradé JWS

**Cohérence** : ✅ Les recommandations sont cohérentes avec le plan Sprint 3 et améliorent la robustesse sans breaking changes.

---

## 🔧 Implémentation Proposée

Souhaitez-vous que j'implémente ces améliorations maintenant, ou préférez-vous les intégrer progressivement ?

**Ordre recommandé** :
1. **Maintenant** : STORAGE_DIR validation + JWKS indépendant + recover
2. **Phase 2** : Prometheus + helmet + requestid + graceful shutdown amélioré

---

**Document créé le** : Janvier 2025  
**Analyse basée sur** : `docs/audit_vault_main.go.md`

