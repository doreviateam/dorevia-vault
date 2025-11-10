# 🚀 Lancement Officiel — Sprint 3 "Expert Edition"

**Projet** : Dorevia Vault  
**Version** : v1.1  
**Date de lancement** : Janvier 2025  
**Période estimée** : Janvier 2025 (15 jours ouvrés)  
**Responsable** : David Baron (Doreviateam)  
**Support technique & audit** : GPT‑5 (expert AMOA & DevOps)

---

## 🎯 Objectif Général

Faire évoluer **Dorevia Vault** du statut de **système vérifiable** (JWS + Ledger) vers un système **supervisable et robuste**, grâce à la mise en place de :

- Health checks avancés (`/health/detailed`)
- Métriques Prometheus (`/metrics`)
- Endpoint de vérification d'intégrité (`/api/v1/ledger/verify/:id`)
- Script de réconciliation automatique (`cmd/reconcile`)
- Optimisations de performance (timeouts, cache JWKS, benchmarks)

---

## ✅ GO / NO GO — Validation de Lancement

| Domaine | Statut | Commentaire |
|:--|:--:|:--|
| 🧱 Environnement VPS | ✅ | IONOS – 8 vCPU / 16 Go RAM / 480 Go SSD |
| 🔑 Clés RSA | ✅ | Générées (key‑2025‑Q1) |
| 💾 Storage | ✅ | `/opt/dorevia-vault/storage` créé |
| 🧰 Build Go | ✅ | Binaire `vault` compilé |
| 🧪 Tests unitaires | ✅ | 38 tests, 100 % réussite |
| 🗄️ PostgreSQL | ⚠️ | DATABASE_URL à configurer (Sprint 3 – J1) |
| 📊 Monitoring | 🚀 | À implémenter dans Sprint 3 |
| 🔍 Endpoint intégrité | 🚀 | À implémenter dans Sprint 3 |

**Verdict** : 🟢 *GO confirmé* — Sprint 3 peut être lancé dès la configuration de la base PostgreSQL.

---

## 🧭 Plan d'Exécution — Sprint 3 (15 jours ouvrés)

| Phase | Jours | Objectif |
|:--|:--|:--|
| **1️⃣ Health & Timeouts** | J1–J3 | `/health/detailed` + timeout transaction (30 s) |
| **2️⃣ Métriques Prometheus** | J4–J6 | `/metrics` avec compteurs + histogrammes |
| **3️⃣ Vérification Intégrité** | J7–J9 | `/api/v1/ledger/verify/:id` signé JWS |
| **4️⃣ Réconciliation** | J10–J12 | `cmd/reconcile` + suppression fichiers orphelins |
| **5️⃣ Optimisations & Tests** | J13–J14 | Benchmarks + cache JWKS + index SQL |
| **6️⃣ Clôture & Documentation** | J15 | Rapport final Sprint 3 + mise à jour README |

---

## 🧰 Étape Initiale (Jour 1)

### 1. Configurer la base PostgreSQL

```bash
export DATABASE_URL="postgres://vault:password@localhost:5432/dorevia_vault?sslmode=disable"
```

### 2. Charger la configuration

```bash
source /opt/dorevia-vault/setup_env.sh
```

### 3. Vérifier la connexion DB

```bash
psql $DATABASE_URL -c "SELECT now();"
```

### 4. Démarrer le service

```bash
go run ./cmd/vault/main.go
```

---

## 🧩 Livrables Attendus

| Catégorie | Livrable | Localisation / Endpoint |
|:--|:--|:--|
| **Monitoring** | Métriques Prometheus | `/metrics` |
| **Santé Système** | Health détaillé | `/health/detailed` |
| **Vérification** | Endpoint intégrité | `/api/v1/ledger/verify/:id` |
| **Maintenance** | Script réconciliation | `cmd/reconcile/main.go` |
| **Performance** | Timeout transaction | `internal/storage/with_timeout.go` |
| **Qualité** | Tests d'intégration | `tests/integration/*` |

---

## 📊 Suivi du Sprint (à compléter au fil du développement)

| Date | Phase | Objectif atteint | Remarques |
|:--|:--|:--:|:--|
| J1–J3 | Health & Timeouts | ✅ **Complété** | Module health (335 lignes), handler, route /health/detailed, timeout 30s, 15 tests unitaires |
| J4–J6 | Métriques Prometheus | ✅ **Complété** | Module metrics (207 lignes), route /metrics, middlewares Helmet/RequestID, 11 métriques actives, intégration handlers/storage |
| J7–J9 | Vérification Intégrité | ✅ **Complété** | Module verify (197 lignes), handler (105 lignes), route /api/v1/ledger/verify/:id, option ?signed=true |
| J10–J12 | Réconciliation | ✅ **Complété** | Module reconcile (180 lignes), CLI cmd/reconcile (120 lignes), flags --dry-run, --fix, --output |
| J13–J14 | Tests & Perf | ⏳ **À venir** | Benchmarks, profiling, optimisations |
| J15 | Clôture & Rapport | ⏳ **À venir** | Documentation finale, mise à jour README |

---

## 🧠 Préconisations Techniques

- Utiliser `context.WithTimeout` sur les transactions DB.  
- Exposer 15 métriques Prometheus minimum (counters + histograms).  
- Tester `VerifyDocumentIntegrity()` avec fichiers et DB.  
- Mettre en cache le `previous_hash` Ledger pour performance.  
- Benchmarker ingestion × 1000 docs (objectif < 300 ms/doc).  
- Documenter tout endpoint dans `/docs/API_SPRINT3.md`.

---

## 🔒 Sécurité

| Élément | État | Recommandation |
|:--|:--|:--|
| Clés RSA | ✅ | Permissions `600/644` vérifiées |
| Variables sensibles | ⚠️ | Chiffrer `.env` ou utiliser Vault (Sprint 4) |
| Auth API | 🚧 | Ajouter Basic Auth (Sprint 4) |
| Sauvegarde DB | 🚀 | Automatiser dump + tar storage hebdomadaire |

---

## 🧮 Indicateurs de Performance Cibles

| Indicateur | Cible Sprint 3 |
|:--|:--|
| Durée moyenne transaction | < 300 ms |
| Latence P95 | < 500 ms |
| Disponibilité service | ≥ 99.9 % |
| Coverage tests | ≥ 85 % |
| Métriques exposées | ≥ 15 |
| Vérification intégrité réussie | 100 % |

---

## 🏁 Clôture Prévue

- **Date cible** : Fin Janvier 2025 (15 jours ouvrés)  
- **Livrable final** : `Dorevia_Vault_Sprint3_Expert_Edition_Report.md`  
- **Version cible** : v1.1 — Stable Supervisable  
- **Critère de réussite** : `/metrics` et `/health/detailed` exposés, tests d'intégrité validés, réconciliation fonctionnelle.

---

## 📚 Références

- **Plan détaillé** : `docs/RESUME_SPRINTS_ET_PLAN_SPRINT3.md`
- **Préparation** : `docs/RESUME_PREPARATION_SPRINT3.md`
- **État pré-Sprint 3** : `docs/Dorevia_Vault_Etat_Pre_Sprint3_ExpertEdition.md`
- **Script configuration** : `setup_env.sh`

---

**Document créé le** : Janvier 2025  
**Version** : 1.0 — Lancement officiel Sprint 3  
**Auteur** : Doreviateam / GPT‑5 (Audit Technique & AMOA)

© 2025 Doreviateam — Document interne confidentiel

