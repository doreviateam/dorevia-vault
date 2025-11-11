# 🚀 Dorevia Vault

**Dorevia Vault** est un **proxy d'intégrité** pour documents électroniques, garantissant la traçabilité et la vérifiabilité selon la **règle des 3V** :
- **Validé** → Document validé dans Odoo
- **Vaulté** → Stocké de manière sécurisée dans Dorevia Vault
- **Vérifiable** → Preuve d'intégrité via JWS + Ledger

Il constitue la brique "coffre documentaire" du projet **Doreviateam**,  
destiné à héberger, indexer et archiver de manière souveraine  
les documents électroniques (Factur-X, pièces jointes, rapports, etc.)

---

## ✨ Fonctionnalités Principales

### Sprint 1 — MVP "Validé → Vaulté"
- ✅ **Ingestion Odoo** : Endpoint `/api/v1/invoices` pour documents Odoo
- ✅ **Transaction atomique** : Garantit cohérence fichier ↔ base de données
- ✅ **Idempotence** : Détection doublons par SHA256
- ✅ **Métadonnées enrichies** : Source, modèle Odoo, état, métadonnées facture

### Sprint 2 — Documents "Vérifiables"
- ✅ **Scellement JWS** : Signature RS256 (RSA-SHA256) conforme RFC 7515
- ✅ **Ledger hash-chaîné** : Traçabilité immuable avec verrou transactionnel
- ✅ **JWKS public** : Endpoint `/jwks.json` pour vérification externe
- ✅ **Export Ledger** : Export JSON/CSV avec pagination
- ✅ **Mode dégradé** : Continuité de service si JWS échoue (optionnel)

### Sprint 3 — "Expert Edition" (Complété)
- ✅ **Health checks avancés** : Endpoint `/health/detailed` avec vérification multi-systèmes
- ✅ **Métriques Prometheus** : 11 métriques actives (counters + histogrammes) via `/metrics`
- ✅ **Sécurité renforcée** : Middlewares Helmet, Recover, RequestID
- ✅ **Vérification intégrité** : Endpoint `/api/v1/ledger/verify/:id` avec preuve JWS signée
- ✅ **Réconciliation automatique** : CLI `bin/reconcile` pour détection et correction fichiers orphelins

### Sprint 4 — "Observabilité & Auditabilité Continue" (Complété — 100%)
- ✅ **Observabilité avancée** : 6 métriques système (CPU, RAM, disque) + `ledger_append_errors_total`
- ✅ **Collecteur automatique** : Mise à jour métriques système toutes les 30s
- ✅ **Journalisation auditable** : Logs signés JSONL avec export paginé (Phase 4.2)
- ✅ **Alerting & supervision** : Alertes Prometheus + Alertmanager + Export Odoo (Phase 4.3)
- ✅ **Audit & conformité** : Rapports signés mensuels/trimestriels (Phase 4.4)

### Sprint 5 — "Sécurité & Interopérabilité" (Complété — 100%)
- ✅ **Sécurité & Key Management** : Intégration HashiCorp Vault, rotation multi-KID, chiffrement au repos (Phase 5.1)
- ✅ **Authentification & Autorisation** : JWT/API Keys, RBAC avec 4 rôles, protection endpoints (Phase 5.2)
- ✅ **Interopérabilité** : Validation Factur-X EN 16931, webhooks asynchrones Redis (Phase 5.3)
- ✅ **Scalabilité** : Partitionnement ledger mensuel, optimisations base de données (Phase 5.4)

---

## 🌍 Environnement

| Élément | Détail |
| :-- | :-- |
| **Langage** | Go 1.23+ |
| **Framework HTTP** | [Fiber](https://github.com/gofiber/fiber) v2.52.9 |
| **Base de données** | PostgreSQL (avec pgxpool) |
| **Reverse Proxy** | Caddy (HTTPS automatique via Let's Encrypt) |
| **Logging** | Zerolog (JSON structuré) |
| **Domaine** | [https://vault.doreviateam.com](https://vault.doreviateam.com) |
| **Version actuelle** | **v1.3.0** (Sprint 5 complété) |
| **Auteur / Mainteneur** | [David Baron – Doreviateam](https://doreviateam.com) |

---

## 🔧 Endpoints disponibles (v1.2.0-rc1)

### Routes de base (toujours actives)

| Méthode | Route | Description |
| :-- | :-- | :-- |
| `GET` | `/` | Page d'accueil |
| `GET` | `/health` | Vérifie l'état du service |
| `GET` | `/health/detailed` | Health check détaillé multi-systèmes (Sprint 3) |
| `GET` | `/version` | Retourne la version déployée |
| `GET` | `/metrics` | Métriques Prometheus (17 métriques actives - Sprint 3+4) |
| `GET` | `/audit/export` | Export logs d'audit paginé (JSON/CSV) (Sprint 4 Phase 4.2) |
| `GET` | `/audit/dates` | Liste des dates disponibles dans les logs (Sprint 4 Phase 4.2) |

### Routes avec base de données (si `DATABASE_URL` configuré)

| Méthode | Route | Description |
| :-- | :-- | :-- |
| `GET` | `/dbhealth` | Vérifie l'état de la connexion PostgreSQL |
| `POST` | `/upload` | Upload de fichier (multipart/form-data) |
| `GET` | `/documents` | Liste paginée des documents (avec recherche et filtres) |
| `GET` | `/documents/:id` | Récupère un document par son ID (UUID) |
| `GET` | `/download/:id` | Télécharge un document par son ID |

### Routes Sprint 1 — Ingestion Odoo

| Méthode | Route | Description |
| :-- | :-- | :-- |
| `POST` | `/api/v1/invoices` | Ingestion documents Odoo (JSON + base64) avec JWS + Ledger |

### Routes Sprint 2 — Vérification & Export

| Méthode | Route | Description |
| :-- | :-- | :-- |
| `GET` | `/jwks.json` | JWKS (JSON Web Key Set) pour vérification JWS |
| `GET` | `/api/v1/ledger/export` | Export ledger (JSON/CSV) avec pagination |

### Routes Sprint 3 — Supervision & Vérification

| Méthode | Route | Description |
| :-- | :-- | :-- |
| `GET` | `/health/detailed` | Health check détaillé (Database, Storage, JWS, Ledger) |
| `GET` | `/metrics` | Métriques Prometheus (17 métriques : métier + système) |
| `GET` | `/api/v1/ledger/verify/:document_id` | Vérification intégrité (fichier ↔ DB ↔ Ledger) |
| `GET` | `/api/v1/ledger/verify/:document_id?signed=true` | Vérification avec preuve JWS signée |

### Routes Sprint 4 — Audit & Observabilité

| Méthode | Route | Description |
| :-- | :-- | :-- |
| `GET` | `/audit/export` | Export logs d'audit paginé (JSON/CSV) avec filtres date |
| `GET` | `/audit/dates` | Liste des dates disponibles dans les logs |

### Routes Sprint 5 — Sécurité & Interopérabilité

| Méthode | Route | Description | Authentification |
| :-- | :-- | :-- | :-- |
| `POST` | `/api/v1/invoices` | Ingestion avec validation Factur-X (Phase 5.3) | `documents:write` |
| `GET` | `/api/v1/ledger/verify/:id` | Vérification intégrité (webhook émis) | `documents:verify` |
| `GET` | `/audit/export` | Export audit (protégé) | `audit:read` |
| `GET` | `/api/v1/ledger/export` | Export ledger (protégé) | `ledger:read` |

**Exemples** :
```bash
# Version
curl -s https://vault.doreviateam.com/version
# → {"version":"1.0"}

# Health check DB
curl -s https://vault.doreviateam.com/dbhealth
# → {"status":"ok","message":"Database connection healthy"}

# Upload fichier
curl -F "file=@document.pdf" https://vault.doreviateam.com/upload

# Ingestion Odoo (Sprint 1)
curl -X POST https://vault.doreviateam.com/api/v1/invoices \
  -H "Content-Type: application/json" \
  -d '{
    "source": "sales",
    "model": "account.move",
    "odoo_id": 123,
    "state": "posted",
    "file": "base64_encoded_content",
    "filename": "invoice_001.pdf"
  }'
# → {"id":"uuid","sha256_hex":"...","evidence_jws":"...","ledger_hash":"..."}

# JWKS (Sprint 2)
curl https://vault.doreviateam.com/jwks.json
# → {"keys":[{"kty":"RSA","kid":"key-2025-Q1",...}]}

# Export Ledger (Sprint 2)
curl "https://vault.doreviateam.com/api/v1/ledger/export?format=json&limit=10"

# Health détaillé (Sprint 3)
curl https://vault.doreviateam.com/health/detailed

# Métriques Prometheus (Sprint 3+4)
curl https://vault.doreviateam.com/metrics
# → Expose 17 métriques : métier (Sprint 3) + système (Sprint 4)

# Export logs d'audit (Sprint 4 Phase 4.2)
curl "https://vault.doreviateam.com/audit/export?from=2025-01-15&to=2025-01-17&page=1&limit=100&format=json"
# → Export paginé des logs d'audit

# Liste dates disponibles (Sprint 4 Phase 4.2)
curl https://vault.doreviateam.com/audit/dates
# → {"dates":["2025-01-15","2025-01-16"],"count":2}

# Génération rapport d'audit (Sprint 4 Phase 4.4)
./bin/audit --period monthly --year 2025 --month 1 --format json --sign --output report-2025-01.json
# → Rapport mensuel JSON signé

./bin/audit --period quarterly --year 2025 --quarter 1 --format pdf --sign --output report-Q1-2025.pdf
# → Rapport trimestriel PDF signé (8 pages)

# Vérification intégrité (Sprint 3)
curl https://vault.doreviateam.com/api/v1/ledger/verify/123e4567-e89b-12d3-a456-426614174000

# Vérification avec preuve JWS (Sprint 3)
curl "https://vault.doreviateam.com/api/v1/ledger/verify/123e4567-e89b-12d3-a456-426614174000?signed=true"

# Liste documents avec recherche
curl "https://vault.doreviateam.com/documents?search=facture&page=1&limit=20"

# Téléchargement
curl -O https://vault.doreviateam.com/download/{uuid}
```

---

## 🧱 Structure

```
/opt/dorevia-vault/
 ├── cmd/
 │   ├── vault/main.go          # Point d'entrée de l'application
 │   ├── keygen/main.go         # Générateur de clés RSA + JWKS (Sprint 2)
 │   ├── reconcile/main.go      # Script réconciliation fichiers orphelins (Sprint 3)
 │   └── audit/main.go          # CLI génération rapports d'audit (Sprint 4 Phase 4.4)
 ├── internal/
 │   ├── config/                # Configuration centralisée
 │   ├── handlers/              # Handlers HTTP (12+ handlers)
 │   ├── middleware/            # Middlewares (CORS, rate limiting, logger)
 │   ├── models/                # Modèles de données
 │   ├── storage/               # PostgreSQL + requêtes + transactions
 │   ├── crypto/                # Module JWS (Sprint 2)
 │   ├── ledger/                # Module Ledger hash-chaîné (Sprint 2)
 │   ├── health/                # Health checks avancés (Sprint 3)
 │   ├── metrics/               # Métriques Prometheus (Sprint 3+4)
 │   ├── verify/                # Vérification intégrité (Sprint 3)
 │   ├── reconcile/             # Réconciliation fichiers orphelins (Sprint 3)
 │   └── audit/                 # Journalisation auditable + rapports (Sprint 4 Phase 4.2+4.4)
 │       ├── log.go             # Logger audit JSONL signé (Phase 4.2)
 │       ├── export.go          # Export logs paginé (Phase 4.2)
 │       ├── sign.go            # Signature journalière (Phase 4.2)
 │       ├── report.go          # Génération rapports JSON/CSV (Phase 4.4)
 │       └── pdf.go             # Génération rapports PDF (Phase 4.4)
 ├── pkg/logger/                # Logger structuré (zerolog)
 ├── tests/
 │   ├── unit/                  # Tests unitaires (115 tests)
 │   └── integration/           # Tests d'intégration (Sprint 2)
 ├── migrations/                # Migrations SQL (003, 004)
 ├── scripts/deploy.sh          # Script de déploiement
 ├── storage/                   # Stockage fichiers (YYYY/MM/DD/)
 └── docs/                      # Documentation complète
```

---

## ⚙️ Configuration

Le service utilise des variables d'environnement pour la configuration :

### Configuration de base

| Variable | Description | Défaut |
| :-- | :-- | :-- |
| `PORT` | Port d'écoute du serveur | `8080` |
| `LOG_LEVEL` | Niveau de log (debug, info, warn, error) | `info` |
| `DATABASE_URL` | URL de connexion PostgreSQL | *(optionnel)* |
| `STORAGE_DIR` | Répertoire de stockage des fichiers | `/opt/dorevia-vault/storage` |
| `AUDIT_DIR` | Répertoire de stockage des logs d'audit | `/opt/dorevia-vault/audit` |

### Configuration JWS (Sprint 2)

| Variable | Description | Défaut |
| :-- | :-- | :-- |
| `JWS_ENABLED` | Activer le scellement JWS | `true` |
| `JWS_REQUIRED` | JWS obligatoire (sinon mode dégradé) | `true` |
| `JWS_PRIVATE_KEY_PATH` | Chemin clé privée RSA (PEM) | *(optionnel)* |
| `JWS_PUBLIC_KEY_PATH` | Chemin clé publique RSA (PEM) | *(optionnel)* |
| `JWS_KID` | Key ID pour JWKS | `key-2025-Q1` |

### Configuration Ledger (Sprint 2)

| Variable | Description | Défaut |
| :-- | :-- | :-- |
| `LEDGER_ENABLED` | Activer le ledger hash-chaîné | `true` |

**Exemple de configuration complète** :
```bash
# Configuration de base
export PORT=8080
export LOG_LEVEL=info
export DATABASE_URL="postgres://vault:password@localhost:5432/dorevia_vault?sslmode=disable"
export STORAGE_DIR="/opt/dorevia-vault/storage"

# Configuration JWS (Sprint 2)
export JWS_ENABLED=true
export JWS_REQUIRED=true
export JWS_PRIVATE_KEY_PATH="/opt/dorevia-vault/keys/private.pem"
export JWS_PUBLIC_KEY_PATH="/opt/dorevia-vault/keys/public.pem"
export JWS_KID="key-2025-Q1"

# Configuration Ledger (Sprint 2)
export LEDGER_ENABLED=true

# Configuration Audit (Sprint 4 Phase 4.2)
export AUDIT_DIR="/opt/dorevia-vault/audit"

# Configuration Authentification (Sprint 5 Phase 5.2)
export AUTH_ENABLED=true
export AUTH_JWT_ENABLED=true
export AUTH_APIKEY_ENABLED=true
export AUTH_JWT_PUBLIC_KEY_PATH="/opt/dorevia-vault/keys/jwt-public.pem"

# Configuration HashiCorp Vault (Sprint 5 Phase 5.1 - optionnel)
export VAULT_ENABLED=false
# export VAULT_ADDR="https://vault.example.com:8200"
# export VAULT_TOKEN="hvs.xxxxx"
# export VAULT_KEY_PATH="secret/data/dorevia/keys"

# Configuration Factur-X (Sprint 5 Phase 5.3)
export FACTURX_VALIDATION_ENABLED=true
export FACTURX_VALIDATION_REQUIRED=false

# Configuration Webhooks (Sprint 5 Phase 5.3 - optionnel)
export WEBHOOKS_ENABLED=false
# export WEBHOOKS_REDIS_URL="redis://localhost:6379/0"
# export WEBHOOKS_SECRET_KEY="$(openssl rand -hex 32)"
# export WEBHOOKS_WORKERS=3
# export WEBHOOKS_URLS="document.vaulted:https://example.com/webhook/vaulted"
```

**Génération des clés RSA** :
```bash
# Générer paire de clés + JWKS
go run ./cmd/keygen/main.go \
  --out /opt/dorevia-vault/keys \
  --kid key-2025-Q1 \
  --bits 2048

# Sécuriser les permissions
chmod 600 /opt/dorevia-vault/keys/private.pem
chmod 644 /opt/dorevia-vault/keys/public.pem
```

**Configuration rapide** :
```bash
# Utiliser le script de configuration automatique
source /opt/dorevia-vault/setup_env.sh

# Le script configure toutes les variables d'environnement
# et vérifie les prérequis (clés RSA, PostgreSQL, etc.)
# Inclut maintenant les variables Sprint 5 (Auth, Vault, Factur-X, Webhooks)
```

---

## 🚀 Déploiement

Voir la documentation complète :  
👉 [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md)  
👉 [`docs/INTEGRATION_POSTGRESQL_DOREVIA_VAULT_v0.1.md`](docs/INTEGRATION_POSTGRESQL_DOREVIA_VAULT_v0.1.md)

Pour un déploiement rapide :
```bash
./scripts/deploy.sh
```

---

## 🧪 Tests

Le projet inclut une suite de tests unitaires complète :

```bash
# Exécuter tous les tests
go test ./tests/unit/... -v

# Tests spécifiques
go test ./tests/unit/... -run TestJWS -v      # Tests JWS (15 tests)
go test ./tests/unit/... -run TestLedger -v   # Tests Ledger (4 tests)

# Avec couverture
go test ./tests/unit/... -coverprofile=coverage.out

# Tests d'intégration (nécessitent DATABASE_URL)
export TEST_DATABASE_URL="postgres://user:pass@localhost/dorevia_vault_test"
go test ./tests/integration/... -v
```

**Statistiques** :
- ✅ **145+ tests unitaires** — 100% de réussite
  - 19 tests existants (Sprint 1)
  - 15 tests JWS (Sprint 2)
  - 4 tests Ledger (Sprint 2)
  - 15 tests Health (Sprint 3 Phase 1)
  - 22 tests Verify/Reconcile (Sprint 3 Phase 3)
  - 11 tests Metrics System (Sprint 4 Phase 4.1)
  - 16 tests Audit (Sprint 4 Phase 4.2)
  - 15+ tests Report (Sprint 4 Phase 4.4)
  - 14 tests PDF (Sprint 4 Phase 4.4)
  - 10 tests CLI (Sprint 4 Phase 4.4)
  - 13 tests autres
- ⏳ **Tests d'intégration** — Prêts (nécessitent DB)

---

## 📊 Génération Rapports d'Audit

**Dorevia Vault** permet de générer des **rapports d'audit** consolidés (mensuels/trimestriels) pour la conformité réglementaire (PDP/PPF 2026).

### Formats Disponibles

| Format | Description | Usage |
|:-------|:------------|:------|
| **JSON** | Format structuré complet avec toutes les données | Intégration, traitement automatique |
| **CSV** | Format simplifié avec colonnes principales | Analyse Excel, import dans outils |
| **PDF** | Document professionnel signé (8 pages) | Conformité, archivage, présentation |

### Installation CLI

```bash
# Compiler le binaire
go build -o bin/audit ./cmd/audit

# Ou avec version/commit
go build -ldflags "-X main.Version=$(git describe --tags) -X main.Commit=$(git rev-parse HEAD)" -o bin/audit ./cmd/audit
```

### Exemples d'Utilisation

#### Rapport mensuel JSON

```bash
./bin/audit --period monthly --year 2025 --month 1 --format json --output report-2025-01.json
```

#### Rapport trimestriel PDF signé

```bash
./bin/audit --period quarterly --year 2025 --quarter 1 --format pdf --sign --output report-Q1-2025.pdf
```

#### Rapport personnalisé CSV

```bash
./bin/audit --period custom --from 2025-01-15 --to 2025-01-31 --format csv --output report-custom.csv
```

#### Rapport mensuel JSON signé (mois actuel)

```bash
./bin/audit --period monthly --format json --sign --output report-current.json
```

### Flags Disponibles

| Flag | Description | Défaut | Requis |
|:-----|:------------|:-------|:-------|
| `--period` | Type de période (monthly, quarterly, custom) | - | ✅ |
| `--year` | Année (pour monthly/quarterly) | Année actuelle | - |
| `--month` | Mois 1-12 (pour monthly) | Mois actuel | - |
| `--quarter` | Trimestre 1-4 (pour quarterly) | Trimestre actuel | - |
| `--from` | Date début YYYY-MM-DD (pour custom) | - | Si custom |
| `--to` | Date fin YYYY-MM-DD (pour custom) | - | Si custom |
| `--format` | Format (json, csv, pdf) | json | - |
| `--output` | Chemin fichier de sortie | stdout (json/csv) ou report-YYYY-MM-DD.pdf | - |
| `--sign` | Signer le rapport avec JWS | false | - |
| `--jws-key-path` | Chemin clé privée JWS | JWS_PRIVATE_KEY_PATH env | - |
| `--audit-dir` | Répertoire audit | AUDIT_DIR env | - |
| `--database-url` | URL base de données | DATABASE_URL env | - |
| `--verbose` | Mode verbeux | false | - |
| `--help` | Afficher l'aide | - | - |

### Contenu des Rapports

Les rapports incluent :

- **Résumé exécutif** : Total documents, taux d'erreur, taille stockage
- **Statistiques documents** : Répartition par statut, source, type MIME, distribution tailles
- **Statistiques erreurs** : Top 10 erreurs critiques avec détails
- **Performance** : Durées moyennes (P50, P95, P99) pour stockage, JWS, ledger, transactions
- **Ledger** : Statistiques ledger (entrées, erreurs, intégrité)
- **Réconciliation** : Statistiques réconciliations (runs, fichiers orphelins)
- **Signatures journalières** : Liste des signatures JWS de la période
- **Métadonnées** : Version, date génération, hash SHA256, signature JWS

### Structure PDF

Le PDF contient **8 pages** :

1. **Page de garde** : Titre, période, QR code du hash SHA256
2. **Résumé exécutif** : Tableau récapitulatif avec indicateurs clés
3. **Statistiques Documents** : Répartition par statut, source, distribution tailles
4. **Statistiques Erreurs** : Top 10 erreurs critiques
5. **Performance** : Durées moyennes (P50, P95, P99)
6. **Ledger & Réconciliation** : Statistiques ledger et réconciliations
7. **Signatures Journalières** : Tableau des signatures JWS
8. **Métadonnées** : Informations système, signature JWS complète

### Configuration Requise

- **Logs d'audit** : Doivent être disponibles dans `AUDIT_DIR/logs/`
- **Base de données** : Optionnelle, mais recommandée pour statistiques complètes
- **Clés JWS** : Requises uniquement si `--sign` est utilisé

### Documentation Complète

Pour plus de détails sur les formats, la structure et la vérification des signatures :

👉 [`docs/audit_export_spec.md`](docs/audit_export_spec.md)

---

## 🛣️ Roadmap

### ✅ Sprint 1 — MVP "Validé → Vaulté" (Complété)
- [x] Extension modèle Document (métadonnées Odoo)
- [x] Migration SQL (003_add_odoo_fields.sql)
- [x] Transaction atomique (fichier ↔ DB)
- [x] Endpoint `/api/v1/invoices` (ingestion Odoo)
- [x] Idempotence par SHA256
- [x] Tests unitaires (19 tests)

### ✅ Sprint 2 — Documents "Vérifiables" (Complété)
- [x] Module JWS (signature RS256, vérification, JWKS)
- [x] Module Ledger (hash-chaîné avec verrou FOR UPDATE)
- [x] Intégration transactionnelle (JWS + Ledger)
- [x] Endpoint `/jwks.json` (JWKS public)
- [x] Endpoint `/api/v1/ledger/export` (export JSON/CSV)
- [x] Générateur de clés (`cmd/keygen`)
- [x] Tests unitaires JWS (15 tests) + Ledger (4 tests)

### ✅ Sprint 3 — "Expert Edition" — De Vérifiable à Supervisable (Complété)
**Durée** : 15 jours ouvrés (Janvier 2025)

**Phase 1 : Health & Timeouts** ✅
- [x] Health checks avancés (`/health/detailed`)
- [x] Timeout transaction 30s
- [x] Tests unitaires health (15 tests)

**Phase 2 : Métriques Prometheus** ✅
- [x] Module métriques Prometheus (11 métriques actives)
- [x] Route `/metrics` opérationnelle
- [x] Middlewares Helmet, RequestID
- [x] Intégration métriques dans handlers et storage

**Phase 3 : Vérification & Réconciliation** ✅
- [x] Endpoint vérification (`/api/v1/ledger/verify/:id` avec option `?signed=true`)
- [x] Script réconciliation (`cmd/reconcile` avec --dry-run, --fix, --output)

### ✅ Sprint 4 — "Observabilité & Auditabilité Continue" (Complété — 100%)
**Durée** : 16 jours ouvrés (Février 2025)

**Phase 4.0 : Corrections Document** ✅
- [x] Harmonisation noms métriques
- [x] Définition seuils d'alerte
- [x] Documentation technique complétée

**Phase 4.1 : Observabilité avancée** ✅
- [x] Métriques système (CPU, RAM, disque) via `gopsutil`
- [x] Métrique `ledger_append_errors_total`
- [x] Collecteur automatique (30s)
- [x] Tests unitaires métriques système (11 tests)
- [x] Documentation `observability_metrics_spec.md`

**Phase 4.2 : Journalisation auditable** ✅
- [x] Module audit/log.go (JSONL writer avec buffer)
- [x] Module audit/sign.go (signature journalière optimisée)
- [x] Module audit/export.go (export paginé JSON/CSV)
- [x] Module audit/rotation.go (rotation automatique + rétention)
- [x] Endpoints `/audit/export` et `/audit/dates`
- [x] Intégration dans handlers (invoices, verify)
- [x] Tests unitaires (16 tests)
- [x] Documentation `audit_log_spec.md`

**Phase 4.3 : Alerting & supervision** ⏳
- [ ] Règles Prometheus détaillées
- [ ] Configuration Alertmanager
- [ ] Export Odoo

**Phase 4.4 : Audit & conformité** ✅
- [x] Module report.go (génération JSON/CSV avec statistiques complètes)
- [x] Module pdf.go (génération PDF 8 pages avec QR code)
- [x] CLI cmd/audit/main.go (tous les flags, validation, signature JWS)
- [x] Tests unitaires (39 tests : 15 report + 14 PDF + 10 CLI)
- [x] Documentation `audit_export_spec.md`

### 🔄 Sprint 5+ — Sécurité & Interopérabilité (À venir)
- [ ] Intégration HSM/Vault (HashiCorp Vault / AWS KMS)
- [ ] Rotation multi-KID pour JWKS
- [ ] Webhooks asynchrones (Queue Redis)
- [ ] Validation Factur-X (EN 16931)
- [ ] Partitionnement Ledger (si volume > 100k/an)

---

## 📚 Documentation

### Documentation Générale
- [`CHANGELOG.md`](CHANGELOG.md) — **Historique des versions**
- [`RELEASE_NOTES_v1.2.0-rc1.md`](RELEASE_NOTES_v1.2.0-rc1.md) — **Notes de version v1.2.0-rc1**
- [`docs/RESUME_SPRINTS_ET_PLAN_SPRINT3.md`](docs/RESUME_SPRINTS_ET_PLAN_SPRINT3.md) — **Résumé Sprints 1 & 2 + Plan Sprint 3**
- [`docs/plan_A.md`](docs/plan_A.md) — Plan d'action détaillé initial
- [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) — Guide de déploiement

### Documentation Sprint 1
- [`docs/SPRINT_1_PLAN.md`](docs/SPRINT_1_PLAN.md) — Plan détaillé Sprint 1
- [`docs/RESUME_SPRINT_1.md`](docs/RESUME_SPRINT_1.md) — Résumé Sprint 1

### Documentation Sprint 2
- [`docs/Dorevia_Vault_Sprint2.md`](docs/Dorevia_Vault_Sprint2.md) — Plan détaillé Sprint 2
- [`docs/INTEGRATION_JWS_LEDGER_COMPLETE.md`](docs/INTEGRATION_JWS_LEDGER_COMPLETE.md) — Intégration JWS + Ledger
- [`docs/AVIS_EXPERT_SPRINT2_RESUME.md`](docs/AVIS_EXPERT_SPRINT2_RESUME.md) — Avis expert Sprint 2
- [`docs/TESTS_JWS_UNITAIRES.md`](docs/TESTS_JWS_UNITAIRES.md) — Tests JWS unitaires

### Documentation Sprint 3
- [`docs/FICHE_DE_CONCEPTION_TECHNIQUE_PHASE_3.MD`](docs/FICHE_DE_CONCEPTION_TECHNIQUE_PHASE_3.MD) — Conception Phase 3
- [`docs/CHECKLIST_PHASE3_AMELIOREE.md`](docs/CHECKLIST_PHASE3_AMELIOREE.md) — Checklist améliorée
- [`docs/PHASE3_VERIFICATION_RECONCILIATION_RESUME.md`](docs/PHASE3_VERIFICATION_RECONCILIATION_RESUME.md) — Résumé Phase 3

### Documentation Sprint 4
- [`docs/Dorevia_Vault_Sprint4.md`](docs/Dorevia_Vault_Sprint4.md) — Plan détaillé Sprint 4 (révisé)
- [`docs/ANALYSE_EXPERT_SPRINT4.md`](docs/ANALYSE_EXPERT_SPRINT4.md) — Analyse experte Sprint 4
- [`docs/SPRINT4_PHASE4.4_PLAN.md`](docs/SPRINT4_PHASE4.4_PLAN.md) — Plan détaillé Phase 4.4 (Audit & conformité)
- [`docs/observability_metrics_spec.md`](docs/observability_metrics_spec.md) — Spécification métriques Prometheus
- [`docs/audit_log_spec.md`](docs/audit_log_spec.md) — Spécification journalisation auditable (Phase 4.2)
- [`docs/audit_export_spec.md`](docs/audit_export_spec.md) — Spécification export rapports d'audit (Phase 4.4)
- [`docs/CORRECTION_ROUTE_METRICS.md`](docs/CORRECTION_ROUTE_METRICS.md) — Correction route `/metrics`

### Documentation Sprint 5

- [`docs/SPRINT5_PLAN.md`](docs/SPRINT5_PLAN.md) — Plan détaillé Sprint 5 (Sécurité & Interopérabilité)
- [`docs/security_vault_spec.md`](docs/security_vault_spec.md) — Spécification HSM/Vault & Key Management
- [`docs/auth_rbac_spec.md`](docs/auth_rbac_spec.md) — Spécification authentification & autorisation
- [`docs/facturx_validation_spec.md`](docs/facturx_validation_spec.md) — Spécification validation Factur-X
- [`docs/webhooks_spec.md`](docs/webhooks_spec.md) — Spécification webhooks asynchrones
- [`docs/partitioning_spec.md`](docs/partitioning_spec.md) — Spécification partitionnement ledger

---

## 🔒 Sécurité

- **CORS** : Configuré (actuellement ouvert à toutes les origines)
- **Rate Limiting** : 100 requêtes/minute par IP
- **JWS** : Signature RS256 (RSA-SHA256) conforme RFC 7515
- **Ledger** : Hash-chaînage immuable avec verrou transactionnel
- **Clés privées** : Permissions 600 (lecture/écriture propriétaire uniquement)
- **Mode dégradé** : Continuité de service si JWS échoue (si `JWS_REQUIRED=false`)
- **Authentification** : ✅ JWT/API Keys + RBAC (Sprint 5)
- **Key Management** : ✅ HashiCorp Vault / fichiers locaux (Sprint 5)
- **Chiffrement au repos** : ✅ AES-256-GCM pour audit (Sprint 5)

---

## 📊 Statistiques

- **Fichiers Go** : 49 fichiers
- **Tests unitaires** : 115 tests (100% réussite)
  - 19 tests Sprint 1
  - 15 tests JWS (Sprint 2)
  - 4 tests Ledger (Sprint 2)
  - 15 tests Health (Sprint 3)
  - 22 tests Verify/Reconcile (Sprint 3)
  - 11 tests Metrics System (Sprint 4 Phase 4.1)
  - 16 tests Audit (Sprint 4 Phase 4.2)
  - 13 tests autres
- **Endpoints** : 16 endpoints
  - 5 routes de base (/, /health, /health/detailed, /version, /metrics)
  - 5 routes DB (Sprint 1)
  - 4 routes Sprint 2+3 (invoices, jwks, ledger/export, ledger/verify)
  - 2 routes Sprint 4 (audit/export, audit/dates)
- **Métriques Prometheus** : 17 métriques actives
  - 11 métriques métier (Sprint 3)
  - 6 métriques système (Sprint 4 Phase 4.1)
- **Modules** : 12 packages modulaires
  - `internal/crypto` (JWS)
  - `internal/ledger` (hash-chaîné)
  - `internal/health` (health checks)
  - `internal/metrics` (Prometheus + système)
  - `internal/verify` (vérification intégrité)
  - `internal/reconcile` (réconciliation)
  - `internal/audit` (journalisation auditable)
  - `cmd/keygen` (génération clés)
  - `cmd/reconcile` (CLI réconciliation)
- **Migrations SQL** : 4 migrations (001, 002, 003, 004)
- **Binaires** : 2 (vault 22M, reconcile 17M)
- **Version** : v1.2.0-rc1 (Sprint 4 Phase 4.4 complétée)

---

© 2025 Doreviateam – Projet sous licence MIT
