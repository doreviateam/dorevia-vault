# 🖥️ Environnement Serveur — Dorevia Vault

**Date** : Janvier 2025  
**Statut** : Environnement de référence  
**Hostname** : `doreviateam`

---

## 💻 Serveur VPS

### Spécifications Techniques

| Composant | Détail |
|:----------|:-------|
| **CPU** | **8 vCPU** (AMD EPYC-Milan Processor) |
| **Architecture** | 4 cores × 2 threads = 8 vCPU |
| **RAM** | **16 Go** (15.6 Gi utilisable) |
| **Stockage** | **480 Go SSD** (464 Go utilisable, 31 Go utilisés, 434 Go libres) |
| **OS** | Ubuntu 24.04.3 LTS |
| **Kernel** | Linux 6.8.0-86-generic |
| **Virtualisation** | Microsoft Hyper-V (QEMU) |
| **Architecture** | x86-64 |

### Détails Techniques

```bash
# CPU
CPU(s): 8
Model name: AMD EPYC-Milan Processor
Thread(s) per core: 2
Core(s) per socket: 4
Socket(s): 1

# RAM
Total: 15.6 Gi
Available: 13.0 Gi
Used: 2.4 Gi

# Disque
/dev/vda1: 464G total, 31G used, 434G available (7% used)
```

### Provider

**IONOS** — VPS avec virtualisation Microsoft Hyper-V

---

## 🔑 Localisation des Clés RSA

### Répertoire

**Chemin** : `/opt/dorevia-vault/keys/`

### État Actuel

✅ **Répertoire `keys/` créé** — Clés RSA générées le 9 novembre 2025.

### Permissions Attendues

| Fichier | Permissions | Description |
|:--------|:------------|:------------|
| `private.pem` | `600` (rw-------) | Clé privée RSA (lecture/écriture propriétaire uniquement) |
| `public.pem` | `644` (rw-r--r--) | Clé publique RSA (lecture publique, écriture propriétaire) |
| `jwks.json` | `644` (rw-r--r--) | JWKS (JSON Web Key Set) |

### Génération des Clés

```bash
# Créer le répertoire
mkdir -p /opt/dorevia-vault/keys

# Générer paire de clés + JWKS
go run ./cmd/keygen/main.go \
  --out /opt/dorevia-vault/keys \
  --kid key-2025-Q1 \
  --bits 2048

# Sécuriser les permissions
chmod 600 /opt/dorevia-vault/keys/private.pem
chmod 644 /opt/dorevia-vault/keys/public.pem
chmod 644 /opt/dorevia-vault/keys/jwks.json

# Vérifier
ls -lah /opt/dorevia-vault/keys/
stat -c "%a %n" /opt/dorevia-vault/keys/*.pem
```

---

## 💾 Volume de Documents

### Répertoire de Stockage

**Chemin** : `/opt/dorevia-vault/storage/`  
**Structure** : `YYYY/MM/DD/uuid-filename`

### État Actuel

✅ **Répertoire `storage/` créé** — Prêt pour le stockage de documents (vide actuellement).

### Statistiques

| Métrique | Valeur Actuelle |
|:---------|:----------------|
| **Nombre de fichiers** | **0** (répertoire vide, prêt pour documents) |
| **Taille totale** | **0** (répertoire vide) |
| **Répartition par date** | **N/A** |

### Commandes de Vérification

```bash
# Compter fichiers
find /opt/dorevia-vault/storage -type f 2>/dev/null | wc -l

# Taille totale
du -sh /opt/dorevia-vault/storage 2>/dev/null

# Répartition par année/mois
ls -d /opt/dorevia-vault/storage/*/* 2>/dev/null | head -10

# Créer le répertoire si nécessaire
mkdir -p /opt/dorevia-vault/storage
```

---

## 📊 Base de Données

### PostgreSQL

| Élément | Détail |
|:--------|:-------|
| **Version** | À vérifier (psql non disponible en ligne de commande) |
| **Base de données** | `dorevia_vault` (à confirmer) |
| **Tables** | `documents`, `ledger` |
| **Migrations** | 001, 002, 003, 004 |

### Vérification

```bash
# Compter documents en DB (via DATABASE_URL)
psql $DATABASE_URL -c "SELECT COUNT(*) FROM documents;"

# Compter entrées ledger
psql $DATABASE_URL -c "SELECT COUNT(*) FROM ledger;"

# Taille base de données
psql $DATABASE_URL -c "SELECT pg_size_pretty(pg_database_size(current_database()));"

# Lister tables
psql $DATABASE_URL -c "\dt"
```

---

## 🔧 Configuration Actuelle

### Variables d'Environnement

```bash
# Vérifier configuration
env | grep -E "PORT|LOG_LEVEL|DATABASE_URL|STORAGE_DIR|JWS_|LEDGER_"

# Ou via systemd
systemctl show dorevia-vault | grep -E "Environment"
```

### Services

```bash
# Vérifier service systemd
systemctl status dorevia-vault

# Vérifier processus
ps aux | grep vault

# Logs
journalctl -u dorevia-vault -f
```

---

## 📝 Résumé

### Environnement de Référence

- **VPS** : 8 vCPU / 16 Go RAM / 480 Go SSD
- **OS** : Ubuntu 24.04.3 LTS
- **Provider** : **IONOS** (virtualisation Microsoft Hyper-V)

### État Actuel

- ✅ **Serveur** : Opérationnel
- ✅ **Clés RSA** : Générées (`/opt/dorevia-vault/keys/` avec private.pem, public.pem, jwks.json)
- ✅ **Stockage** : Répertoire créé (`/opt/dorevia-vault/storage/` prêt, vide actuellement)
- ❓ **Base de données** : À vérifier (PostgreSQL configuré ?)

### Actions Recommandées

1. ✅ **Clés RSA générées** (9 novembre 2025) — Plus d'action requise

2. ✅ **Répertoire storage créé** — Plus d'action requise

3. **Vérifier PostgreSQL** :
   ```bash
   # Vérifier si DATABASE_URL est configuré
   echo $DATABASE_URL
   
   # Tester la connexion
   psql $DATABASE_URL -c "SELECT version();"
   ```

4. **Configurer les variables d'environnement** :
   ```bash
   export JWS_PRIVATE_KEY_PATH=/opt/dorevia-vault/keys/private.pem
   export JWS_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/public.pem
   export JWS_KID=key-2025-Q1
   export STORAGE_DIR=/opt/dorevia-vault/storage
   ```

---

**Document mis à jour le** : Janvier 2025
