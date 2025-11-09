# 🚀 Dorevia Vault

**Dorevia Vault** est un microservice écrit en **Go + Fiber**.  
Il constitue la brique "coffre documentaire" du projet **Doreviateam**,  
destiné à héberger, indexer et archiver de manière souveraine  
les documents électroniques (Factur-X, pièces jointes, rapports, etc.)

---

## 🌍 Environnement

| Élément | Détail |
| :-- | :-- |
| **Langage** | Go 1.22+ |
| **Framework HTTP** | [Fiber](https://github.com/gofiber/fiber) v2.52.9 |
| **Base de données** | PostgreSQL (avec pgxpool) |
| **Reverse Proxy** | Caddy (HTTPS automatique via Let's Encrypt) |
| **Logging** | Zerolog (JSON structuré) |
| **Domaine** | [https://vault.doreviateam.com](https://vault.doreviateam.com) |
| **Version actuelle** | v0.1.0 |
| **Auteur / Mainteneur** | [David Baron – Doreviateam](https://doreviateam.com) |

---

## 🔧 Endpoints disponibles (v0.1.0)

### Routes de base (toujours actives)

| Méthode | Route | Description |
| :-- | :-- | :-- |
| `GET` | `/` | Page d'accueil |
| `GET` | `/health` | Vérifie l'état du service |
| `GET` | `/version` | Retourne la version déployée |

### Routes avec base de données (si `DATABASE_URL` configuré)

| Méthode | Route | Description |
| :-- | :-- | :-- |
| `GET` | `/dbhealth` | Vérifie l'état de la connexion PostgreSQL |
| `POST` | `/upload` | Upload de fichier (multipart/form-data) |
| `GET` | `/documents` | Liste paginée des documents (avec recherche et filtres) |
| `GET` | `/documents/:id` | Récupère un document par son ID (UUID) |
| `GET` | `/download/:id` | Télécharge un document par son ID |

Exemples :
```bash
# Version
curl -s https://vault.doreviateam.com/version
# → {"version":"0.1.0"}

# Health check DB
curl -s https://vault.doreviateam.com/dbhealth
# → {"status":"ok","message":"Database connection healthy"}

# Upload fichier
curl -F "file=@document.pdf" https://vault.doreviateam.com/upload

# Liste documents avec recherche
curl "https://vault.doreviateam.com/documents?search=facture&page=1&limit=20"

# Téléchargement
curl -O https://vault.doreviateam.com/download/{uuid}
```

---

## 🧱 Structure

```
/opt/dorevia-vault/
 ├── cmd/vault/main.go          # Point d'entrée de l'application
 ├── internal/
 │   ├── config/                # Configuration centralisée
 │   ├── handlers/              # Handlers HTTP (7 handlers)
 │   ├── middleware/            # Middlewares (CORS, rate limiting, logger)
 │   ├── models/                # Modèles de données
 │   └── storage/               # PostgreSQL + requêtes
 ├── pkg/logger/                # Logger structuré (zerolog)
 ├── tests/unit/                # Tests unitaires (19 tests)
 ├── scripts/deploy.sh          # Script de déploiement
 ├── storage/                   # Stockage fichiers (YYYY/MM/DD/)
 └── docs/                      # Documentation
```

---

## ⚙️ Configuration

Le service utilise des variables d'environnement pour la configuration :

| Variable | Description | Défaut |
| :-- | :-- | :-- |
| `PORT` | Port d'écoute du serveur | `8080` |
| `LOG_LEVEL` | Niveau de log (debug, info, warn, error) | `info` |
| `DATABASE_URL` | URL de connexion PostgreSQL | *(optionnel)* |
| `STORAGE_DIR` | Répertoire de stockage des fichiers | `/opt/dorevia-vault/storage` |

**Exemple de configuration** :
```bash
export PORT=8080
export LOG_LEVEL=info
export DATABASE_URL="postgres://vault:password@localhost:5432/dorevia_vault?sslmode=disable"
export STORAGE_DIR="/opt/dorevia-vault/storage"
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

# Avec couverture
go test ./tests/unit/... -coverprofile=coverage.out
```

**Statistiques** : 19 tests unitaires — 100% de réussite ✅

---

## 🛣️ Roadmap

### ✅ Phase 1 — Fondations (Complétée)
- [x] Architecture modulaire
- [x] Configuration centralisée
- [x] Logging structuré
- [x] Middlewares sécurité (CORS, rate limiting)
- [x] Tests unitaires
- [x] CI/CD GitHub Actions

### ✅ Phase 2 — Fonctionnalités (Complétée)
- [x] Connexion PostgreSQL
- [x] Endpoint `/upload` pour stockage et indexation des fichiers
- [x] Endpoint `/documents` pour recherche et consultation
- [x] Endpoint `/download/:id` pour téléchargement
- [x] Détection de doublons (SHA256)

### 🔄 Phase 3 — Intégrations (À venir)
- [ ] Authentification JWT / API keys
- [ ] Intégration Odoo CE 18 (Factur-X, webhooks)
- [ ] Indexation avancée (métadonnées, full-text)
- [ ] Archivage long terme (S3/MinIO)

---

## 📚 Documentation

- [`docs/plan_A.md`](docs/plan_A.md) — Plan d'action détaillé
- [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) — Guide de déploiement
- [`docs/INTEGRATION_POSTGRESQL_DOREVIA_VAULT_v0.1.md`](docs/INTEGRATION_POSTGRESQL_DOREVIA_VAULT_v0.1.md) — Intégration PostgreSQL
- [`RAPPORT_SITUATION_PHASE2.md`](RAPPORT_SITUATION_PHASE2.md) — Rapport de situation Phase 2

---

## 🔒 Sécurité

- **CORS** : Configuré (actuellement ouvert à toutes les origines)
- **Rate Limiting** : 100 requêtes/minute par IP
- **Authentification** : À venir (Phase 3)

---

## 📊 Statistiques

- **Fichiers Go** : 23 fichiers
- **Tests unitaires** : 19 tests (100% réussite)
- **Endpoints** : 8 endpoints
- **Packages** : 8 packages modulaires

---

© 2025 Doreviateam – Projet sous licence MIT
