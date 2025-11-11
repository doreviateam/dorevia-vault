# ⚙️ Variables d'Environnement — Dorevia Vault

**Date** : Janvier 2025  
**Version** : v1.0 → v1.3.0 (Sprint 5)

---

## 📋 Variables Requises

### Configuration de Base

| Variable | Description | Défaut | Requis |
|:---------|:------------|:-------|:-------|
| `PORT` | Port d'écoute HTTP | `8080` | Non |
| `LOG_LEVEL` | Niveau de log (debug, info, warn, error) | `info` | Non |
| `DATABASE_URL` | URL de connexion PostgreSQL | - | **Oui** |
| `STORAGE_DIR` | Répertoire de stockage fichiers | `/opt/dorevia-vault/storage` | Non |

### Configuration JWS (Sprint 2)

| Variable | Description | Défaut | Requis |
|:---------|:------------|:-------|:-------|
| `JWS_ENABLED` | Activer le scellement JWS | `true` | Non |
| `JWS_REQUIRED` | JWS obligatoire (sinon mode dégradé) | `true` | Non |
| `JWS_PRIVATE_KEY_PATH` | Chemin clé privée RSA (PEM) | - | Si `JWS_ENABLED=true` |
| `JWS_PUBLIC_KEY_PATH` | Chemin clé publique RSA (PEM) | - | Si `JWS_ENABLED=true` |
| `JWS_KID` | Key ID pour JWKS | `key-2025-Q1` | Non |

### Configuration Ledger (Sprint 2)

| Variable | Description | Défaut | Requis |
|:---------|:------------|:-------|:-------|
| `LEDGER_ENABLED` | Activer le ledger hash-chaîné | `true` | Non |

### Configuration Audit (Sprint 4 Phase 4.2)

| Variable | Description | Défaut | Requis |
|:---------|:------------|:-------|:-------|
| `AUDIT_DIR` | Répertoire de stockage des logs d'audit | `/opt/dorevia-vault/audit` | Non |

### Configuration Authentification (Sprint 5 Phase 5.2)

| Variable | Description | Défaut | Requis |
|:---------|:------------|:-------|:-------|
| `AUTH_ENABLED` | Activer l'authentification | `false` | Non |
| `AUTH_JWT_ENABLED` | Activer l'authentification JWT | `true` | Si `AUTH_ENABLED=true` |
| `AUTH_APIKEY_ENABLED` | Activer l'authentification API Keys | `true` | Si `AUTH_ENABLED=true` |
| `AUTH_JWT_PUBLIC_KEY_PATH` | Chemin clé publique JWT (PEM) | - | Si `AUTH_JWT_ENABLED=true` |

### Configuration HashiCorp Vault (Sprint 5 Phase 5.1)

| Variable | Description | Défaut | Requis |
|:---------|:------------|:-------|:-------|
| `VAULT_ENABLED` | Activer HashiCorp Vault | `false` | Non |
| `VAULT_ADDR` | Adresse du serveur Vault | - | Si `VAULT_ENABLED=true` |
| `VAULT_TOKEN` | Token d'authentification Vault | - | Si `VAULT_ENABLED=true` |
| `VAULT_KEY_PATH` | Chemin des clés dans Vault | `secret/data/dorevia/keys` | Si `VAULT_ENABLED=true` |
| `VAULT_NAMESPACE` | Namespace Vault (optionnel) | - | Non |

### Configuration Rotation Multi-KID (Sprint 5 Phase 5.1)

| Variable | Description | Défaut | Requis |
|:---------|:------------|:-------|:-------|
| `KEY_ROTATION_ENABLED` | Activer rotation automatique | `false` | Non |
| `KEY_ROTATION_INTERVAL` | Intervalle de rotation | `90d` | Si `KEY_ROTATION_ENABLED=true` |
| `CURRENT_KID` | Key ID actuelle | `key-2025-Q1` | Non |
| `PREVIOUS_KID` | Key ID précédente | - | Non |
| `NEXT_KID` | Key ID suivante | - | Non |

### Configuration Chiffrement Audit (Sprint 5 Phase 5.1)

| Variable | Description | Défaut | Requis |
|:---------|:------------|:-------|:-------|
| `AUDIT_ENCRYPTION_ENABLED` | Activer chiffrement au repos | `false` | Non |
| `AUDIT_ENCRYPTION_KEY_ID` | ID de la clé de chiffrement | - | Si `AUDIT_ENCRYPTION_ENABLED=true` |

### Configuration Validation Factur-X (Sprint 5 Phase 5.3)

| Variable | Description | Défaut | Requis |
|:---------|:------------|:-------|:-------|
| `FACTURX_VALIDATION_ENABLED` | Activer validation Factur-X | `true` | Non |
| `FACTURX_VALIDATION_REQUIRED` | Validation Factur-X obligatoire | `false` | Non |

### Configuration Webhooks (Sprint 5 Phase 5.3)

| Variable | Description | Défaut | Requis |
|:---------|:------------|:-------|:-------|
| `WEBHOOKS_ENABLED` | Activer les webhooks | `false` | Non |
| `WEBHOOKS_REDIS_URL` | URL Redis pour la queue | `redis://localhost:6379/0` | Si `WEBHOOKS_ENABLED=true` |
| `WEBHOOKS_SECRET_KEY` | Clé secrète pour signature HMAC | - | Si `WEBHOOKS_ENABLED=true` |
| `WEBHOOKS_WORKERS` | Nombre de workers parallèles | `3` | Non |
| `WEBHOOKS_URLS` | URLs webhooks par événement | - | Si `WEBHOOKS_ENABLED=true` |

**Format WEBHOOKS_URLS** : `event1:url1,url2|event2:url3`

---

## 🔧 Configuration Recommandée (Sprint 5)

### Fichier `.env` (optionnel)

```bash
# Configuration de base
PORT=8080
LOG_LEVEL=info
DATABASE_URL=postgres://vault:password@localhost:5432/dorevia_vault?sslmode=disable
STORAGE_DIR=/opt/dorevia-vault/storage

# Configuration JWS
JWS_ENABLED=true
JWS_REQUIRED=true
JWS_PRIVATE_KEY_PATH=/opt/dorevia-vault/keys/private.pem
JWS_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/public.pem
JWS_KID=key-2025-Q1

# Configuration Ledger
LEDGER_ENABLED=true

# Configuration Audit (Sprint 4)
AUDIT_DIR=/opt/dorevia-vault/audit

# Configuration Authentification (Sprint 5)
AUTH_ENABLED=true
AUTH_JWT_ENABLED=true
AUTH_APIKEY_ENABLED=true
AUTH_JWT_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/jwt-public.pem

# Configuration HashiCorp Vault (Sprint 5 - optionnel)
VAULT_ENABLED=false
# VAULT_ADDR=https://vault.example.com:8200
# VAULT_TOKEN=hvs.xxxxx
# VAULT_KEY_PATH=secret/data/dorevia/keys

# Configuration Factur-X (Sprint 5)
FACTURX_VALIDATION_ENABLED=true
FACTURX_VALIDATION_REQUIRED=false

# Configuration Webhooks (Sprint 5 - optionnel)
WEBHOOKS_ENABLED=false
# WEBHOOKS_REDIS_URL=redis://localhost:6379/0
# WEBHOOKS_SECRET_KEY=$(openssl rand -hex 32)
# WEBHOOKS_WORKERS=3
# WEBHOOKS_URLS=document.vaulted:https://example.com/webhook/vaulted
```

### Chargement via systemd

Si le service est géré par systemd, créer `/etc/systemd/system/dorevia-vault.service` :

```ini
[Unit]
Description=Dorevia Vault Service
After=network.target postgresql.service

[Service]
Type=simple
User=dorevia
WorkingDirectory=/opt/dorevia-vault
ExecStart=/opt/dorevia-vault/bin/vault
Restart=always
RestartSec=5

# Variables d'environnement
Environment="PORT=8080"
Environment="LOG_LEVEL=info"
Environment="DATABASE_URL=postgres://vault:password@localhost:5432/dorevia_vault?sslmode=disable"
Environment="STORAGE_DIR=/opt/dorevia-vault/storage"
Environment="JWS_ENABLED=true"
Environment="JWS_REQUIRED=true"
Environment="JWS_PRIVATE_KEY_PATH=/opt/dorevia-vault/keys/private.pem"
Environment="JWS_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/public.pem"
Environment="JWS_KID=key-2025-Q1"
Environment="LEDGER_ENABLED=true"

[Install]
WantedBy=multi-user.target
```

---

## ✅ Vérification

### Commandes de Vérification

```bash
# Vérifier toutes les variables
env | grep -E "PORT|LOG_LEVEL|DATABASE_URL|STORAGE_DIR|JWS_|LEDGER_|AUTH_|VAULT_|FACTURX_|WEBHOOKS_"

# Vérifier DATABASE_URL (masquer le mot de passe)
echo $DATABASE_URL | sed 's/:[^:@]*@/:***@/g'

# Tester connexion PostgreSQL
psql $DATABASE_URL -c "SELECT version();"

# Vérifier que les clés existent
ls -lh $JWS_PRIVATE_KEY_PATH $JWS_PUBLIC_KEY_PATH

# Vérifier que le répertoire storage existe
ls -ld $STORAGE_DIR
```

---

## 🔒 Sécurité

### Bonnes Pratiques

1. **Ne jamais commiter** `.env` ou fichiers contenant des mots de passe
2. **Utiliser des secrets managers** en production (HashiCorp Vault, AWS Secrets Manager)
3. **Restreindre les permissions** sur les fichiers de configuration
4. **Chiffrer** les variables sensibles en transit

### Variables Sensibles

- `DATABASE_URL` : Contient mot de passe PostgreSQL
- `JWS_PRIVATE_KEY_PATH` : Chemin vers clé privée RSA
- `VAULT_TOKEN` : Token d'authentification HashiCorp Vault
- `WEBHOOKS_SECRET_KEY` : Clé secrète pour signature HMAC des webhooks
- `AUTH_JWT_PUBLIC_KEY_PATH` : Clé publique pour validation JWT

---

---

## 🚀 Configuration Rapide avec setup_env.sh

Le script `setup_env.sh` configure automatiquement toutes les variables :

```bash
source /opt/dorevia-vault/setup_env.sh
```

Le script gère :
- ✅ Variables de base (PORT, LOG_LEVEL, STORAGE_DIR)
- ✅ Configuration PostgreSQL
- ✅ Configuration JWS (détection automatique des clés)
- ✅ Configuration Ledger
- ✅ **Configuration Sprint 5** (Auth, Vault, Factur-X, Webhooks)

**Voir** : `/opt/dorevia-vault/setup_env.sh`

---

**Document créé le** : Janvier 2025  
**Dernière mise à jour** : Janvier 2025 (Sprint 5)

