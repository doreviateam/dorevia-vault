# 📊 Rapport de Vérification — Préparation Sprint 3

**Date** : Janvier 2025  
**Statut** : Vérification prérequis Sprint 3

---

## ✅ Éléments Complétés

### 1. Clés RSA ✅

- **Répertoire** : `/opt/dorevia-vault/keys/` créé
- **Fichiers générés** :
  - `private.pem` (1.7K, permissions 600) ✅
  - `public.pem` (451B, permissions 644) ✅
  - `jwks.json` (496B, permissions 644) ✅
- **KID** : `key-2025-Q1`
- **Bits** : 2048

### 2. Répertoire Storage ✅

- **Répertoire** : `/opt/dorevia-vault/storage/` créé
- **Permissions** : 755 (drwxrwxr-x)
- **Statut** : Prêt pour stockage de documents

---

## ❌ Éléments Manquants

### 1. Variables d'Environnement ❌

**Variables non configurées** :

| Variable | Statut | Action Requise |
|:---------|:-------|:----------------|
| `DATABASE_URL` | ❌ Non configuré | **REQUIS** — Configurer URL PostgreSQL |
| `JWS_PRIVATE_KEY_PATH` | ❌ Non configuré | Configurer : `/opt/dorevia-vault/keys/private.pem` |
| `JWS_PUBLIC_KEY_PATH` | ❌ Non configuré | Configurer : `/opt/dorevia-vault/keys/public.pem` |
| `STORAGE_DIR` | ⚠️ Défaut OK | Optionnel (défaut : `/opt/dorevia-vault/storage`) |

**Variables avec valeurs par défaut** (OK) :

- `PORT=8080` (défaut)
- `LOG_LEVEL=info` (défaut)
- `JWS_ENABLED=true` (défaut)
- `JWS_REQUIRED=true` (défaut)
- `JWS_KID=key-2025-Q1` (défaut)
- `LEDGER_ENABLED=true` (défaut)

### 2. PostgreSQL ❌

**Statut** : Impossible de vérifier (DATABASE_URL manquant)

**Actions requises** :

1. **Configurer DATABASE_URL** :
   ```bash
   export DATABASE_URL="postgres://user:password@localhost:5432/dorevia_vault?sslmode=disable"
   ```

2. **Vérifier la connexion** :
   ```bash
   psql $DATABASE_URL -c "SELECT version();"
   ```

3. **Vérifier les tables** :
   ```bash
   psql $DATABASE_URL -c "\dt"
   # Doit afficher : documents, ledger
   ```

4. **Vérifier les migrations** :
   ```bash
   psql $DATABASE_URL -c "\d documents" | grep -E "evidence_jws|ledger_hash"
   psql $DATABASE_URL -c "\d ledger"
   ```

---

## 🔧 Configuration Recommandée

### Script de Configuration

Créer un fichier `setup_env.sh` :

```bash
#!/bin/bash

# Configuration de base
export PORT=8080
export LOG_LEVEL=info
export STORAGE_DIR=/opt/dorevia-vault/storage

# Configuration PostgreSQL (À ADAPTER)
export DATABASE_URL="postgres://vault:password@localhost:5432/dorevia_vault?sslmode=disable"

# Configuration JWS
export JWS_ENABLED=true
export JWS_REQUIRED=true
export JWS_PRIVATE_KEY_PATH=/opt/dorevia-vault/keys/private.pem
export JWS_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/public.pem
export JWS_KID=key-2025-Q1

# Configuration Ledger
export LEDGER_ENABLED=true

echo "✅ Variables d'environnement configurées"
```

**Utilisation** :

```bash
source setup_env.sh
```

### Vérification Post-Configuration

```bash
# Vérifier toutes les variables
env | grep -E "PORT|LOG_LEVEL|DATABASE_URL|STORAGE_DIR|JWS_|LEDGER_"

# Tester PostgreSQL
psql $DATABASE_URL -c "SELECT 'OK' as status;"

# Vérifier les clés
ls -lh $JWS_PRIVATE_KEY_PATH $JWS_PUBLIC_KEY_PATH
```

---

## 📋 Checklist Finale

### Prérequis Sprint 3

- [x] **Clés RSA générées** (`/opt/dorevia-vault/keys/` avec 3 fichiers, permissions correctes)
- [x] **Répertoire storage créé** (`/opt/dorevia-vault/storage/` existe)
- [ ] **PostgreSQL configuré** (DATABASE_URL valide, tables documents + ledger présentes)
- [ ] **Variables d'environnement configurées** (DATABASE_URL, JWS_PRIVATE_KEY_PATH, JWS_PUBLIC_KEY_PATH)
- [ ] **Build Go réussi** (bin/vault et bin/keygen compilent)
- [ ] **Tests unitaires passent** (38 tests, 100% réussite, couverture ≥ 80%)
- [ ] **Service démarre correctement** (endpoints /health, /version, /dbhealth, /jwks.json répondent)

---

## 🎯 Prochaines Actions

1. **Configurer DATABASE_URL** avec les identifiants PostgreSQL réels
2. **Configurer les variables JWS** (JWS_PRIVATE_KEY_PATH, JWS_PUBLIC_KEY_PATH)
3. **Tester la connexion PostgreSQL** (vérifier tables, migrations)
4. **Vérifier le build Go** (`go build ./cmd/vault`)
5. **Tester le démarrage du service** avec les variables configurées

---

## 📚 Documentation

- `docs/PREPARATION_SPRINT3.md` — Checklist complète de préparation
- `docs/VARIABLES_ENVIRONNEMENT.md` — Guide des variables d'environnement
- `docs/ENVIRONNEMENT_SERVEUR.md` — Spécifications serveur IONOS

---

**Document créé le** : Janvier 2025  
**Dernière vérification** : Janvier 2025

