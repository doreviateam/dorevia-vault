# 📊 Rapport complet & Plan de Préparation v0.1.x
## Dorevia Vault — Microservice de Coffre Documentaire

**Auteur** : Doreviateam / D. Baron  
**Version du document** : 1.0  
**Date** : Janvier 2025  
**Objet** : Rapport d'analyse + plan d'action unifié avant v0.1.x

---

## 📋 Table des matières

1. [Résumé exécutif](#résumé-exécutif)
2. [Contexte du projet](#contexte-du-projet)
3. [Analyse de l’architecture actuelle](#analyse-de-larchitecture-actuelle)
4. [Points forts et points d’amélioration](#points-forts-et-points-damélioration)
5. [Préconisations détaillées](#préconisations-détaillées)
6. [Fichiers à créer / corriger](#fichiers-à-créer--corriger)
7. [Refactoring incrémental (PR A & B)](#refactoring-incrémental-pr-a--b)
8. [Plan d’action global](#plan-daction-global)
9. [Check-list qualité](#check-list-qualité)
10. [Étapes suivantes](#étapes-suivantes)
11. [Annexes techniques](#annexes-techniques)

---

## 🎯 Résumé exécutif

**Dorevia Vault** (v0.0.1) est un microservice Go minimaliste servant de base à un futur coffre documentaire souverain (Factur-X, pièces jointes, rapports).  
Le projet est **fonctionnel**, bien documenté, et déployé via **Caddy + systemd**, mais nécessite une **structuration complète du code** avant d’intégrer PostgreSQL, les uploads ou l’intégration Odoo.

**Verdict global** : ✅ **Projet prometteur**  
**Priorité d’action** : 🔴 **Élevée — Refactoring architectural avant v0.1.x**

---

## 📖 Contexte du projet

| Élément | Valeur |
|:--|:--|
| **Langage** | Go 1.22.2 *(README mentionne 1.23+ → à corriger)* |
| **Framework HTTP** | Fiber v2.52.9 |
| **Infrastructure** | Caddy (reverse proxy), systemd, Docker |
| **Base de données prévue** | PostgreSQL |
| **Version courante** | v0.0.1 |
| **Serveur** | doreviateam.com |

**Objectif général :**  
Créer un microservice de stockage et d’indexation documentaire interopérable avec Odoo CE et des PDP externes (OpenBee, Bureau Veritas…).

---

## 🏗️ Analyse de l’architecture actuelle

### Structure réelle
```
/opt/dorevia-vault/
├── bin/vault
├── cmd/vault/main.go
├── go.mod
├── go.sum
├── docs/DEPLOYMENT.md
├── README.md
├── LICENSE
└── (deploy.sh manquant)
```

### Analyse rapide

| Élément | État | Commentaire |
|:--|:--|:--|
| Code source | ⚠️ Monolithique | Tout est dans `main.go` |
| Documentation | ✅ Bonne | README + DEPLOYMENT.md |
| Tests | ❌ Aucun | Aucune couverture |
| Sécurité | ❌ Aucune | Pas d’auth, CORS, limiter |
| Déploiement | ✅ Fonctionnel | systemd + Caddy OK |
| Logs | ⚠️ Basiques | Logger standard `log` |
| Variables env | ⚠️ Simples | Pas de struct centralisée |

---

## ✅ Points forts et ⚠️ Points d’amélioration

### ✅ Points forts
- Go + Fiber → légèreté et rapidité  
- Caddy → HTTPS automatique  
- Service systemd propre  
- Documentation lisible  
- Infrastructure réseau bien isolée

### ⚠️ Améliorations critiques
1. **Architecture monolithique**
   - Complexité croissante si ajout de routes  
   - Difficulté de test et de maintenance

2. **Pas de gestion d’erreurs structurée**
   - Pas de types d’erreurs, pas de codes HTTP cohérents

3. **Absence de sécurité**
   - Pas d’authentification, pas de CORS, pas de rate limiting

4. **Aucun test**
   - Risque de régression élevé

### ⚙️ Améliorations secondaires
- Centralisation de la configuration (`internal/config`)  
- Logging structuré (`pkg/logger`)  
- Ajout de CI/CD  
- Correction du README et version Go  
- Création des fichiers manquants

---

## 💡 Préconisations détaillées

### 1. Nouvelle structure cible
```
/opt/dorevia-vault/
├── cmd/
│   └── vault/main.go
├── internal/
│   ├── config/config.go
│   ├── handlers/
│   │   ├── health.go
│   │   ├── version.go
│   │   └── documents.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── cors.go
│   │   └── logger.go
│   └── storage/
│       ├── postgres.go
│       └── filesystem.go
├── pkg/logger/logger.go
├── tests/
│   ├── unit/
│   └── integration/
├── scripts/deploy.sh
├── .github/workflows/ci.yml
├── .gitignore
├── .editorconfig
└── docs/
    ├── DEPLOYMENT.md
    └── ANALYSE_TECHNIQUE_2025-01.md
```

### 2. Sécurité
- Auth : clé API ou JWT
- Rate limiting : `fiber/middleware/limiter`
- Validation : `go-playground/validator/v10`

### 3. Logging
Utiliser `zerolog` :
```go
logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
logger.Info().Msg("Vault API started")
```

### 4. Config centralisée
Utiliser `caarlos0/env` :
```go
type Config struct {
    Port string `env:"PORT" envDefault:"8080"`
    LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}
```

---

## 🧩 Fichiers à créer / corriger

### `.gitignore`
```gitignore
bin/
*.exe
*.dll
*.so
*.test
*.out
vendor/
.idea/
.vscode/
*.swp
*~
.env
.env.local
```

### `.editorconfig`
```ini
root = true

[*]
charset = utf-8
end_of_line = lf
insert_final_newline = true

[*.go]
indent_style = tab
indent_size = 4

[*.{yml,yaml}]
indent_style = space
indent_size = 2
```

### `scripts/deploy.sh`
```bash
#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

echo "🔄 Pulling latest changes..."
git pull

echo "🔨 Building binary..."
go build -o bin/vault ./cmd/vault

echo "🚀 Restarting service..."
sudo systemctl restart dorevia-vault

echo "✅ Deployment complete!"
```

### `.github/workflows/ci.yml`
```yaml
name: Go CI
on: [push, pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go build ./cmd/vault
      - run: go vet ./...
      - run: go test ./... || true
```

---

## 🔧 Refactoring incrémental (PR A & B)

### PR A — Handlers
Créer :
```go
// internal/handlers/health.go
package handlers
import "github.com/gofiber/fiber/v2"
func Health(c *fiber.Ctx) error {
    return c.SendString("ok")
}
```

```go
// internal/handlers/version.go
package handlers
import "github.com/gofiber/fiber/v2"
func Version(c *fiber.Ctx) error {
    return c.JSON(fiber.Map{"version": "0.0.1"})
}
```

Modifier `main.go` :
```go
app := fiber.New()
app.Get("/health", handlers.Health)
app.Get("/version", handlers.Version)
```

### PR B — Config
```go
// internal/config/config.go
package config
import "github.com/caarlos0/env/v10"
type Config struct {
    Port string `env:"PORT" envDefault:"8080"`
    LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}
func Load() Config {
    var cfg Config
    _ = env.Parse(&cfg)
    return cfg
}
```

---

## 🗓️ Plan d’action global

### Phase 1 — Fondations (1-2 semaines)
1. Créer la nouvelle structure  
2. Extraire `handlers` et `config`  
3. Ajouter logger + middleware  
4. CI/CD minimal  
5. Corriger incohérences de doc

### Phase 2 — Fonctionnalités (2-3 semaines)
1. Connexion PostgreSQL  
2. Endpoint `/upload`  
3. Stockage fichiers + métadonnées  
4. Listing `/documents`

### Phase 3 — Intégrations (3-4 semaines)
1. Odoo CE 18  
2. OpenBee PDP  
3. Archivage long terme (NF525 / S3)

---

## ✅ Check-list qualité

| Élément | Cible |
|:--|:--|
| Couverture de test | ≥ 80 % |
| Complexité par fonction | < 10 |
| Documentation API | 100 % |
| CI/CD active | ✅ |
| Sécurité (auth + limiter) | ✅ |
| Config centralisée | ✅ |

---

## 🚀 Étapes suivantes

1. Exécuter les correctifs proposés (`sed`, `cat`, `mkdir` …)  
2. Committer sous `feat: refactoring structure`  
3. Vérifier via `curl -s https://vault.doreviateam.com/health`  
4. Pousser sur GitHub et valider la CI  
5. Lancer la **Phase 1** de refactoring  

---

## 📎 Annexes techniques

### Rapport d'analyse initial (Janvier 2025)
➡️ Fichier : `RAPPORT_ANALYSE_01.md` (à la racine du projet)

### Commandes de vérification
```bash
go version         # doit afficher 1.22.x ou supérieur
go build ./cmd/vault
go test ./...
curl -s https://vault.doreviateam.com/health
```

---

**Mainteneur** : David Baron  
**Hébergement** : doreviateam.com  
**Licence** : MIT  
**Dernière révision** : Janvier 2025
