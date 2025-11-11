# 📋 Scripts de Configuration — Dorevia Vault

## 🔧 configure_service.sh

Script interactif pour configurer le service systemd Dorevia Vault.

### Fonctionnalités

- ✅ **Vérification** : Analyse la configuration actuelle
- ✅ **Sauvegarde** : Crée une sauvegarde automatique avant modification
- ✅ **Configuration DATABASE_URL** : Ajoute la connexion PostgreSQL
- ✅ **Configuration AUTH_ENABLED** : Active/désactive l'authentification
- ✅ **Redémarrage automatique** : Recharge systemd et redémarre le service
- ✅ **Vérification** : Affiche le statut final et des commandes de test

### Usage

```bash
sudo ./scripts/configure_service.sh
```

### Exemple d'utilisation

```bash
$ sudo ./scripts/configure_service.sh

🔧 Configuration du service Dorevia Vault
==========================================

📋 Création d'une sauvegarde...
✅ Sauvegarde créée: /etc/systemd/system/dorevia-vault.service.backup

📊 Configuration actuelle:
⚠️  DATABASE_URL non configuré
⚠️  AUTH_ENABLED non configuré

🔐 Configuration DATABASE_URL
----------------------------
Entrez votre DATABASE_URL PostgreSQL: postgres://user:pass@localhost:5432/dorevia_vault?sslmode=disable
✅ DATABASE_URL ajouté

🔒 Configuration Authentification
--------------------------------
Souhaitez-vous activer l'authentification ?
  - false : Accès libre aux endpoints (développement)
  - true  : Authentification JWT/API Key requise (production)
AUTH_ENABLED [false]: false
✅ AUTH_ENABLED=false ajouté

🔄 Recharger et redémarrer le service ?
Continuer [O/n]: O

🔄 Rechargement de systemd...
✅ systemd rechargé

🔄 Redémarrage du service...
✅ Service redémarré

📊 Statut du service:
● dorevia-vault.service - Dorevia Vault API
     Active: active (running)

✅ Configuration terminée !

🧪 Tests recommandés:
   curl https://vault.doreviateam.com/health
   curl https://vault.doreviateam.com/dbhealth
   curl https://vault.doreviateam.com/documents
```

### Variables configurées

- **DATABASE_URL** : URL de connexion PostgreSQL
  - Format : `postgres://user:password@host:port/database?sslmode=disable`
  - Requis pour : `/documents`, `/dbhealth`, `/api/v1/invoices`, etc.

- **AUTH_ENABLED** : Active/désactive l'authentification
  - `false` : Accès libre (développement)
  - `true` : Authentification requise (production)
  - Si `true`, configurer aussi `AUTH_JWT_PUBLIC_KEY_PATH` ou utiliser des API Keys

### Sauvegarde

Le script crée automatiquement une sauvegarde :
- **Fichier** : `/etc/systemd/system/dorevia-vault.service.backup`
- **Restauration** : `sudo cp /etc/systemd/system/dorevia-vault.service.backup /etc/systemd/system/dorevia-vault.service`

### Dépannage

#### Le service ne démarre pas

```bash
# Vérifier les logs
sudo journalctl -u dorevia-vault -n 50

# Vérifier la configuration
sudo systemctl cat dorevia-vault
```

#### Restaurer la configuration précédente

```bash
sudo cp /etc/systemd/system/dorevia-vault.service.backup \
       /etc/systemd/system/dorevia-vault.service
sudo systemctl daemon-reload
sudo systemctl restart dorevia-vault
```

#### Vérifier que les endpoints fonctionnent

```bash
# Health check (toujours disponible)
curl https://vault.doreviateam.com/health

# Database health (nécessite DATABASE_URL)
curl https://vault.doreviateam.com/dbhealth

# Documents (nécessite DATABASE_URL + AUTH_ENABLED=false ou token)
curl https://vault.doreviateam.com/documents
```

---

## 🚀 deploy.sh

Script de déploiement rapide (recompilation + redémarrage).

### Usage

```bash
./scripts/deploy.sh
```

---

## 🔨 build.sh

Script de build avec injection de métadonnées via ldflags.

### Usage

```bash
# Build avec version automatique (détectée depuis git)
./scripts/build.sh

# Build avec version spécifique
./scripts/build.sh 1.3.0

# Build avec version et output personnalisés
./scripts/build.sh 1.3.0 bin/vault-custom
```

### Fonctionnalités

- ✅ **Détection automatique de version** : Depuis git tag ou version par défaut
- ✅ **Injection de métadonnées** : Version, Commit, BuiltAt, Schema
- ✅ **Build optimisé** : Utilise ldflags pour injecter les valeurs au build time
- ✅ **Informations détaillées** : Affiche toutes les valeurs injectées

### Exemple de sortie

```bash
$ ./scripts/build.sh 1.3.0

🔨 Build Dorevia Vault
======================

📦 Version    : 1.3.0
🔖 Commit     : a1b2c3d
📅 Built At   : 2025-01-11T12:00:00Z
📋 Schema     : 20250111_1200
📁 Output     : bin/vault

🔨 Compilation en cours...

✅ Build réussi !

📊 Informations du binaire :
   Taille : 24M
   Chemin : bin/vault

📋 Valeurs injectées :
   Version: 1.3.0
   Commit:  a1b2c3d
   BuiltAt:  2025-01-11T12:00:00Z
   Schema:   20250111_1200
```

### Métadonnées injectées

Les valeurs suivantes sont injectées dans le binaire via ldflags :

- **Version** : Version du projet (détectée depuis git tag ou paramètre)
- **Commit** : Hash court du commit Git (ex: `a1b2c3d`)
- **BuiltAt** : Date/heure de build en UTC (format ISO 8601)
- **Schema** : Schema de version (format: `YYYYMMDD_HHMM`)

Ces valeurs sont accessibles via l'endpoint `/version` :

```json
{
  "version": "1.3.0",
  "commit": "a1b2c3d",
  "built_at": "2025-01-11T12:00:00Z",
  "schema": "20250111_1200"
}
```

---

**Document créé le** : Janvier 2025

