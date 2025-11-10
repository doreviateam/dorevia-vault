# 🚀 Préparation Sprint 3 — Checklist Pratique

**Date** : Janvier 2025  
**Version** : v1.0 → v1.1  
**Objectif** : Préparer l'environnement pour le Sprint 3 "Expert Edition"

---

## ✅ Checklist Prérequis (GO/NO GO)

### 1. 🔑 Génération des Clés RSA

**Statut actuel** : ❌ Clés non générées

**Action requise** :

```bash
# 1. Créer le répertoire
mkdir -p /opt/dorevia-vault/keys

# 2. Générer la paire de clés RSA + JWKS
cd /opt/dorevia-vault
go run ./cmd/keygen/main.go \
  --out /opt/dorevia-vault/keys \
  --kid key-2025-Q1 \
  --bits 2048

# 3. Vérifier la génération
ls -lah /opt/dorevia-vault/keys/
# Doit afficher : private.pem, public.pem, jwks.json

# 4. Sécuriser les permissions
chmod 600 /opt/dorevia-vault/keys/private.pem
chmod 644 /opt/dorevia-vault/keys/public.pem
chmod 644 /opt/dorevia-vault/keys/jwks.json

# 5. Vérifier les permissions
stat -c "%a %n" /opt/dorevia-vault/keys/*.pem
# private.pem doit être 600
# public.pem doit être 644
```

**✅ Validation** : Les 3 fichiers existent avec les bonnes permissions

---

### 2. 💾 Répertoire de Stockage

**Statut actuel** : ❌ Répertoire inexistant

**Action requise** :

```bash
# Créer le répertoire storage
mkdir -p /opt/dorevia-vault/storage

# Vérifier les permissions (doit être 755)
ls -ld /opt/dorevia-vault/storage
# Doit afficher : drwxr-xr-x ... storage
```

**✅ Validation** : Répertoire créé et accessible

---

### 3. 🗄️ Base de Données PostgreSQL

**Statut actuel** : ⚠️ À vérifier

**Actions de vérification** :

```bash
# 1. Vérifier que DATABASE_URL est configuré
echo $DATABASE_URL
# Doit afficher : postgres://user:pass@host:port/database

# 2. Tester la connexion (si psql disponible)
psql $DATABASE_URL -c "SELECT version();"

# 3. Vérifier les migrations appliquées
psql $DATABASE_URL -c "\dt"
# Doit afficher : documents, ledger

# 4. Vérifier la structure de la table documents
psql $DATABASE_URL -c "\d documents" | grep -E "evidence_jws|ledger_hash"
# Doit afficher les colonnes evidence_jws et ledger_hash

# 5. Vérifier la structure de la table ledger
psql $DATABASE_URL -c "\d ledger"
# Doit afficher : id, document_id, hash, previous_hash, timestamp, evidence_jws
```

**✅ Validation** : 
- DATABASE_URL configuré
- Tables `documents` et `ledger` existent
- Colonnes Sprint 2 présentes (evidence_jws, ledger_hash)

---

### 4. 🔨 Build Go

**Action requise** :

```bash
cd /opt/dorevia-vault

# 1. Vérifier la version Go
go version
# Doit être : go1.23.x ou supérieur

# 2. Nettoyer les builds précédents
go clean -cache

# 3. Télécharger les dépendances
go mod download

# 4. Vérifier les dépendances
go mod verify

# 5. Build du binaire principal
go build -o bin/vault ./cmd/vault
# Doit réussir sans erreur

# 6. Build du générateur de clés
go build -o bin/keygen ./cmd/keygen
# Doit réussir sans erreur

# 7. Vérifier que les binaires existent
ls -lh bin/
# Doit afficher : vault, keygen
```

**✅ Validation** : Les 2 binaires compilent sans erreur

---

### 5. 🧪 Tests Unitaires

**Action requise** :

```bash
cd /opt/dorevia-vault

# 1. Exécuter tous les tests unitaires
go test ./tests/unit/... -v

# 2. Vérifier le résultat (doit être 100% réussite)
go test ./tests/unit/... -v | grep -E "PASS|FAIL"
# Doit afficher : PASS pour tous les tests

# 3. Compter les tests
go test ./tests/unit/... -v 2>&1 | grep -c "RUN"
# Doit être : 38 tests minimum

# 4. Tests avec couverture
go test ./tests/unit/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total
# Doit afficher : coverage: ~80% ou plus
```

**✅ Validation** : 38 tests passent à 100%, couverture ≥ 80%

---

### 6. ⚙️ Variables d'Environnement

**Action requise** :

```bash
# Vérifier les variables essentielles
env | grep -E "PORT|LOG_LEVEL|DATABASE_URL|STORAGE_DIR|JWS_|LEDGER_"

# Variables attendues :
# PORT=8080 (ou autre)
# LOG_LEVEL=info
# DATABASE_URL=postgres://...
# STORAGE_DIR=/opt/dorevia-vault/storage
# JWS_ENABLED=true
# JWS_REQUIRED=true
# JWS_PRIVATE_KEY_PATH=/opt/dorevia-vault/keys/private.pem
# JWS_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/public.pem
# JWS_KID=key-2025-Q1
# LEDGER_ENABLED=true
```

**✅ Validation** : Toutes les variables essentielles sont configurées

---

### 7. 🚀 Test de Démarrage

**Action requise** :

```bash
cd /opt/dorevia-vault

# 1. Démarrer le service en mode test (arrière-plan)
./bin/vault &
VAULT_PID=$!

# 2. Attendre 2 secondes
sleep 2

# 3. Vérifier que le processus tourne
ps aux | grep vault | grep -v grep
# Doit afficher le processus vault

# 4. Tester l'endpoint health
curl -s http://localhost:8080/health
# Doit retourner : {"status":"ok"}

# 5. Tester l'endpoint version
curl -s http://localhost:8080/version
# Doit retourner : {"version":"1.0"}

# 6. Tester l'endpoint dbhealth (si DB configurée)
curl -s http://localhost:8080/dbhealth
# Doit retourner : {"status":"ok","message":"Database connection healthy"}

# 7. Tester l'endpoint JWKS (si clés générées)
curl -s http://localhost:8080/jwks.json
# Doit retourner un JSON avec les clés publiques

# 8. Arrêter le service
kill $VAULT_PID
wait $VAULT_PID 2>/dev/null
```

**✅ Validation** : Le service démarre et répond aux endpoints de base

---

## 📋 Résumé Préparation

### ✅ Checklist Complète

- [ ] **Clés RSA générées** (`/opt/dorevia-vault/keys/` avec 3 fichiers, permissions correctes)
- [ ] **Répertoire storage créé** (`/opt/dorevia-vault/storage/` existe)
- [ ] **PostgreSQL configuré** (DATABASE_URL valide, tables documents + ledger présentes)
- [ ] **Build Go réussi** (bin/vault et bin/keygen compilent)
- [ ] **Tests unitaires passent** (38 tests, 100% réussite, couverture ≥ 80%)
- [ ] **Variables d'environnement configurées** (PORT, DATABASE_URL, JWS_*, LEDGER_*)
- [ ] **Service démarre correctement** (endpoints /health, /version, /dbhealth, /jwks.json répondent)

---

## 🎯 Prochaines Étapes (Sprint 3)

Une fois tous les prérequis validés, démarrer le Sprint 3 selon le plan :

**Phase 1 : Health & Timeouts (J1-J3)**
- Créer `internal/health/detailed.go`
- Implémenter vérifications multi-systèmes
- Ajouter timeout transaction (30s)
- Route `/health/detailed`

**Référence** : `docs/RESUME_SPRINTS_ET_PLAN_SPRINT3.md` (plan détaillé)

---

## 🚨 En Cas de Problème

### Clés RSA non générées
```bash
# Vérifier que cmd/keygen existe
ls -la cmd/keygen/main.go
# Si absent, vérifier le commit Git
```

### PostgreSQL inaccessible
```bash
# Vérifier que PostgreSQL tourne
sudo systemctl status postgresql
# Vérifier les logs
sudo journalctl -u postgresql -n 50
```

### Tests échouent
```bash
# Vérifier les dépendances
go mod tidy
# Relancer les tests avec détails
go test ./tests/unit/... -v -run TestName
```

### Service ne démarre pas
```bash
# Vérifier les logs
./bin/vault 2>&1 | head -20
# Vérifier les ports
netstat -tulpn | grep 8080
```

---

## 🚀 Script de Configuration Automatique

Un script `setup_env.sh` est disponible à la racine du projet pour faciliter la configuration :

```bash
# Utiliser le script
source /opt/dorevia-vault/setup_env.sh
```

Le script :
- ✅ Configure automatiquement toutes les variables d'environnement
- ✅ Vérifie les prérequis (clés RSA, répertoires)
- ✅ Teste la connexion PostgreSQL si configurée
- ✅ Affiche un résumé de la configuration
- ✅ Propose de créer les répertoires manquants

**Voir** : `/opt/dorevia-vault/setup_env.sh`

---

**Document créé le** : Janvier 2025  
**Dernière mise à jour** : Janvier 2025

