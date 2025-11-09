# 🐘 Intégration PostgreSQL — Dorevia Vault v0.1

**Date :** Janvier 2025  
**Projet :** Dorevia Vault  
**Version concernée :** v0.1.x  
**Auteur :** Doreviateam  

---

## 📋 Objectif

Ce document décrit **le processus complet d’intégration de PostgreSQL** dans le microservice **Dorevia Vault**, incluant :

- installation et configuration PostgreSQL  
- création de l’utilisateur et de la base dédiés  
- intégration Go (`pgxpool`)  
- migration automatique et test `/dbhealth`  
- endpoint `/upload` pour stockage + insertion en base  
- procédure de build, déploiement et tests

---

## 🏗️ Étape 1 — Installation et préparation de PostgreSQL

### 1.1 Installer PostgreSQL

```bash
sudo apt-get update
sudo apt-get install -y postgresql postgresql-contrib
```

### 1.2 Créer la base et l’utilisateur

```bash
sudo -u postgres psql <<'SQL'
CREATE USER vault WITH PASSWORD 'change-me' LOGIN;
CREATE DATABASE dorevia_vault OWNER vault;
\c dorevia_vault
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS documents (
  id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  filename     TEXT NOT NULL,
  content_type TEXT,
  size_bytes   BIGINT,
  sha256_hex   TEXT NOT NULL,
  stored_path  TEXT NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
SQL
```

---

## ⚙️ Étape 2 — Configuration du service

Créer `/etc/dorevia-vault.env` :

```bash
PORT=8080
DATABASE_URL=postgres://vault:change-me@localhost:5432/dorevia_vault?sslmode=disable
STORAGE_DIR=/opt/dorevia-vault/storage
```

Créer le dossier :

```bash
sudo mkdir -p /opt/dorevia-vault/storage
sudo chown -R dorevia:dorevia /opt/dorevia-vault/storage
```

Modifier l’unité Systemd :

```bash
sudo sed -i 's|^ExecStart=.*|EnvironmentFile=-/etc/dorevia-vault.env\nExecStart=/opt/dorevia-vault/bin/vault|' /etc/systemd/system/dorevia-vault.service
sudo systemctl daemon-reload
```

---

## 💻 Étape 3 — Intégration Go

### 3.1 Installer la dépendance

```bash
cd /opt/dorevia-vault
go get github.com/jackc/pgx/v5/pgxpool
```

### 3.2 Remplacer `cmd/vault/main.go`

*(voir section précédente pour code complet)*

> Ce code gère : connexion DB, migration auto, endpoint `/dbhealth`, et `/upload` stockant métadonnées + fichier.

---

## 🧱 Étape 4 — Build & restart

```bash
cd /opt/dorevia-vault
go mod tidy
go build -o bin/vault ./cmd/vault
sudo systemctl restart dorevia-vault
journalctl -u dorevia-vault -n 50 --no-pager
```

---

## 🔍 Étape 5 — Vérification

```bash
curl -s https://vault.doreviateam.com/dbhealth
echo "hello $(date)" > /tmp/test.txt
curl -s -F "file=@/tmp/test.txt" https://vault.doreviateam.com/upload | jq .
sudo -u postgres psql -d dorevia_vault -c "SELECT id, filename, size_bytes, left(sha256_hex,12) sha12, created_at FROM documents ORDER BY created_at DESC LIMIT 5;"
```

---

## 🚀 Étape 6 — Étapes suivantes

- Créer `internal/storage/postgres.go`  
- Ajouter `/documents` (listing, recherche)  
- Ajouter `/download/:id` (récupération)  
- Tests d’intégration Postgres (`pgxmock`)  
- Authentification JWT/API key  
- Indexation Factur‑X / PDF  

---

## ✅ Résumé

| Élément | Statut |
|:--------|:-------|
| PostgreSQL installé | ✅ |
| Base & user créés | ✅ |
| Connexion Go (`pgxpool`) | ✅ |
| Migration auto | ✅ |
| Endpoint `/dbhealth` | ✅ |
| Endpoint `/upload` | ✅ |
| Stockage fichiers | ✅ |
| Tests API | ✅ |

---

**Document validé :** Janvier 2025  
**Version :** 1.0  
**Auteur :** Doreviateam  
© 2025 — Licence MIT
