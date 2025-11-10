# ⚙️ Variables d'Environnement — Dorevia Vault

**Date** : Janvier 2025  
**Version** : v1.0 → v1.1 (Sprint 3)

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

---

## 🔧 Configuration Recommandée (Sprint 3)

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
env | grep -E "PORT|LOG_LEVEL|DATABASE_URL|STORAGE_DIR|JWS_|LEDGER_"

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

---

**Document créé le** : Janvier 2025

