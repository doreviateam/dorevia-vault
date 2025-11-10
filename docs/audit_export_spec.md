# 📋 Spécification — Export Rapports d'Audit

**Version** : v1.2-dev  
**Date** : Janvier 2025  
**Sprint** : Sprint 4 Phase 4.4  
**Auteur** : Doreviateam

---

## 🎯 Vue d'Ensemble

Les **rapports d'audit** permettent de générer des documents consolidés (mensuels/trimestriels) pour la conformité réglementaire (PDP/PPF 2026). Ils sont disponibles en trois formats : **JSON**, **CSV**, et **PDF**.

### Formats Disponibles

| Format | Description | Usage |
|:-------|:------------|:------|
| **JSON** | Format structuré complet avec toutes les données | Intégration, traitement automatique |
| **CSV** | Format simplifié avec colonnes principales | Analyse Excel, import dans outils |
| **PDF** | Document professionnel signé (8 pages) | Conformité, archivage, présentation |

---

## 📄 Format JSON

### Structure Complète

Le format JSON contient toutes les données du rapport dans une structure hiérarchique :

```json
{
  "period": {
    "type": "monthly",
    "start_date": "2025-01-01",
    "end_date": "2025-01-31",
    "label": "Janvier 2025"
  },
  "summary": {
    "total_documents": 1234,
    "total_errors": 12,
    "error_rate": 0.97,
    "total_ledger_entries": 1234,
    "total_reconciliations": 5,
    "avg_document_size": 45678,
    "total_storage_size": 56789012
  },
  "documents": {
    "total": 1234,
    "by_status": {
      "success": 1200,
      "error": 12,
      "idempotent": 22
    },
    "by_source": {
      "sales": 800,
      "purchase": 300,
      "pos": 100,
      "stock": 34
    },
    "by_content_type": {
      "application/pdf": 1200,
      "application/xml": 34
    },
    "size_distribution": {
      "min": 10000,
      "max": 200000,
      "mean": 45678.5,
      "median": 45000,
      "p95": 150000,
      "p99": 180000
    }
  },
  "errors": {
    "total": 12,
    "by_type": {
      "storage error": 8,
      "JWS signing failed": 4
    },
    "by_event_type": {
      "document_vaulted": 8,
      "jws_signed": 4
    },
    "critical_errors": [
      {
        "timestamp": "2025-01-15T10:30:00Z",
        "event_type": "document_vaulted",
        "document_id": "123e4567-e89b-12d3-a456-426614174000",
        "message": "storage error",
        "count": 5
      }
    ]
  },
  "performance": {
    "document_storage": {
      "count": 1200,
      "mean": 0.150,
      "median": 0.120,
      "p50": 0.120,
      "p95": 0.300,
      "p99": 0.500,
      "min": 0.050,
      "max": 0.800
    },
    "jws_signature": {
      "count": 1200,
      "mean": 0.010,
      "median": 0.008,
      "p50": 0.008,
      "p95": 0.020,
      "p99": 0.030,
      "min": 0.005,
      "max": 0.050
    },
    "ledger_append": {
      "count": 1200,
      "mean": 0.050,
      "median": 0.040,
      "p50": 0.040,
      "p95": 0.100,
      "p99": 0.150,
      "min": 0.020,
      "max": 0.200
    },
    "transaction": {
      "count": 1200,
      "mean": 0.200,
      "median": 0.180,
      "p50": 0.180,
      "p95": 0.400,
      "p99": 0.600,
      "min": 0.100,
      "max": 1.000
    }
  },
  "ledger": {
    "total_entries": 5000,
    "new_entries": 1234,
    "errors": 0,
    "error_rate": 0.0,
    "current_size": 5000,
    "chain_integrity": true,
    "last_hash": "abc123def456ghi789jkl012mno345pqr678"
  },
  "reconciliation": {
    "total_runs": 5,
    "successful_runs": 5,
    "failed_runs": 0,
    "orphan_files_found": 10,
    "orphan_files_fixed": 10,
    "documents_fixed": 5
  },
  "signatures": [
    {
      "date": "2025-01-15",
      "hash": "abc123def456...",
      "jws": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
      "line_count": 150,
      "timestamp": "2025-01-15T23:59:59Z"
    }
  ],
  "metadata": {
    "generated_at": "2025-02-01T10:00:00Z",
    "generated_by": "cli",
    "version": "v1.2-dev",
    "report_id": "123e4567-e89b-12d3-a456-426614174000",
    "report_hash": "6016c6675c71c2c277ddedbba6fed0a809024f708473bddd81bd82889295c3f9",
    "report_jws": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
    "data_sources": ["audit_logs", "database", "daily_signatures"]
  }
}
```

### Champs Principaux

#### `period`
- **type** : `"monthly"` | `"quarterly"` | `"custom"`
- **start_date** : Date début au format `YYYY-MM-DD`
- **end_date** : Date fin au format `YYYY-MM-DD`
- **label** : Libellé lisible (ex: "Janvier 2025", "Q1 2025")

#### `summary`
Résumé exécutif avec indicateurs clés :
- **total_documents** : Nombre total de documents vaultés
- **total_errors** : Nombre total d'erreurs
- **error_rate** : Taux d'erreur en pourcentage
- **total_ledger_entries** : Nombre d'entrées ledger
- **total_reconciliations** : Nombre de réconciliations
- **avg_document_size** : Taille moyenne document (bytes)
- **total_storage_size** : Taille totale stockage (bytes)

#### `documents`
Statistiques détaillées sur les documents :
- **total** : Total documents
- **by_status** : Répartition par statut (success, error, idempotent)
- **by_source** : Répartition par source (sales, purchase, pos, etc.)
- **by_content_type** : Répartition par type MIME
- **size_distribution** : Distribution des tailles (min, max, mean, median, P95, P99)

#### `errors`
Statistiques sur les erreurs :
- **total** : Total erreurs
- **by_type** : Répartition par type d'erreur
- **by_event_type** : Répartition par type d'événement
- **critical_errors** : Top 10 erreurs critiques avec détails

#### `performance`
Métriques de performance (durées en secondes) :
- **document_storage** : Stockage documents
- **jws_signature** : Signature JWS
- **ledger_append** : Ajout au ledger
- **transaction** : Transactions complètes

Chaque métrique contient : `count`, `mean`, `median`, `p50`, `p95`, `p99`, `min`, `max`.

#### `ledger`
Statistiques sur le ledger :
- **total_entries** : Total entrées
- **new_entries** : Nouvelles entrées (période)
- **errors** : Erreurs ledger
- **error_rate** : Taux d'erreur (%)
- **current_size** : Taille actuelle
- **chain_integrity** : Intégrité de la chaîne (bool)
- **last_hash** : Dernier hash

#### `reconciliation`
Statistiques sur les réconciliations :
- **total_runs** : Total exécutions
- **successful_runs** : Exécutions réussies
- **failed_runs** : Exécutions échouées
- **orphan_files_found** : Fichiers orphelins trouvés
- **orphan_files_fixed** : Fichiers orphelins corrigés
- **documents_fixed** : Documents corrigés

#### `signatures`
Liste des signatures journalières de la période :
- **date** : Date au format `YYYY-MM-DD`
- **hash** : Hash SHA256 cumulé
- **jws** : Signature JWS du hash
- **line_count** : Nombre de lignes signées
- **timestamp** : Timestamp de la signature (RFC3339)

#### `metadata`
Métadonnées du rapport :
- **generated_at** : Date de génération (RFC3339)
- **generated_by** : `"cli"` ou `"dorevia-vault"`
- **version** : Version du système
- **report_id** : UUID unique du rapport
- **report_hash** : SHA256 du rapport JSON
- **report_jws** : Signature JWS du rapport (si signé)
- **data_sources** : Sources de données utilisées

---

## 📊 Format CSV

### Structure Simplifiée

Le format CSV contient uniquement les indicateurs principaux du résumé exécutif :

```csv
period_type,period_start,period_end,total_documents,total_errors,error_rate,avg_document_size,total_storage_size,total_ledger_entries,total_reconciliations
monthly,2025-01-01,2025-01-31,1234,12,0.97,45678,56789012,1234,5
```

### Colonnes

| Colonne | Description | Type |
|:--------|:------------|:-----|
| `period_type` | Type de période (monthly, quarterly, custom) | String |
| `period_start` | Date début `YYYY-MM-DD` | String |
| `period_end` | Date fin `YYYY-MM-DD` | String |
| `total_documents` | Total documents vaultés | Integer |
| `total_errors` | Total erreurs | Integer |
| `error_rate` | Taux d'erreur (%) | Float |
| `avg_document_size` | Taille moyenne document (bytes) | Integer |
| `total_storage_size` | Taille totale stockage (bytes) | Integer |
| `total_ledger_entries` | Total entrées ledger | Integer |
| `total_reconciliations` | Total réconciliations | Integer |

### Exemple

```csv
period_type,period_start,period_end,total_documents,total_errors,error_rate,avg_document_size,total_storage_size,total_ledger_entries,total_reconciliations
monthly,2025-01-01,2025-01-31,1234,12,0.97,45678,56789012,1234,5
quarterly,2025-01-01,2025-03-31,5000,50,1.00,50000,250000000,5000,20
custom,2025-01-15,2025-01-31,500,5,1.00,45000,22500000,500,2
```

---

## 📑 Format PDF

### Structure du Document

Le PDF contient **8 pages** avec un template professionnel :

#### Page 1 : Page de garde
- Logo Doreviateam (optionnel)
- Titre : "Rapport d'Audit Dorevia Vault"
- Période : "Janvier 2025" ou "Q1 2025"
- Date de génération
- **QR code** du hash SHA256 du rapport

#### Page 2 : Résumé exécutif
- Tableau récapitulatif avec indicateurs clés
- Total documents, taux d'erreur, taille stockage
- Intégrité ledger

#### Page 3 : Statistiques Documents
- Tableau par statut (success, error, idempotent)
- Tableau par source (sales, purchase, pos, etc.)
- Distribution des tailles (min, max, mean, median, P95, P99)

#### Page 4 : Statistiques Erreurs
- Tableau Top 10 erreurs critiques
- Détails par type d'erreur

#### Page 5 : Performance
- Tableau des durées moyennes (P50, P95, P99)
- Stockage documents, Signature JWS, Ajout ledger, Transactions

#### Page 6 : Ledger & Réconciliation
- Statistiques ledger (entrées, erreurs, intégrité)
- Statistiques réconciliation (runs, fichiers orphelins)

#### Page 7 : Signatures Journalières
- Tableau des signatures (date, hash, JWS, lignes)
- Vérification intégrité

#### Page 8 : Métadonnées
- Informations système (version, généré par, date)
- Sources de données
- **Signature JWS complète** (texte)

### Caractéristiques

- **Format** : A4 (210 × 297 mm)
- **Police** : Arial/Helvetica
- **Couleurs** :
  - Bleu Dorevia (#0066CC) pour en-têtes
  - Gris (#666666) pour texte secondaire
  - Rouge (#CC0000) pour erreurs
  - Vert (#00CC00) pour succès
- **Marges** : 20mm (haut/bas), 15mm (gauche/droite)
- **QR Code** : 40mm × 40mm sur page de garde

---

## 🔒 Signature JWS

### Format de Signature

La signature JWS suit le format **RFC 7515** (JWS Compact Serialization) :

```
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJkb2N1bWVudF9pZCI6IjEyM2U0NTY3LWU4OWItMTJkMy1hNDU2LTQyNjYxNDE3NDAwMCIsInNoYTI1Nl9oZXgiOiI2MDE2YzY2NzVjNzFjMmMyNzdkZGVkYmJhNmZlZDBhODA5MDI0ZjcwODQ3M2JkZGQ4MWJkODI4ODkyOTVjM2Y5IiwidGltZXN0YW1wIjoiMjAyNS0wMi0wMVQxMDowMDowMFoifQ.SIGNATURE
```

### Structure du Payload JWS

```json
{
  "document_id": "123e4567-e89b-12d3-a456-426614174000",
  "sha256_hex": "6016c6675c71c2c277ddedbba6fed0a809024f708473bddd81bd82889295c3f9",
  "timestamp": "2025-02-01T10:00:00Z"
}
```

### Vérification

La signature peut être vérifiée via :

1. **JWKS public** : `GET /jwks.json`
2. **Hash SHA256** : Vérifier que `report.metadata.report_hash` correspond au hash du JSON
3. **Signature JWS** : Décoder et vérifier avec la clé publique JWKS

### Exemple de Vérification

```bash
# Récupérer le JWKS
curl https://vault.doreviateam.com/jwks.json

# Vérifier la signature (avec un outil JWT)
jwt verify --jwks-url=https://vault.doreviateam.com/jwks.json <report_jws>
```

---

## 🛠️ Utilisation CLI

### Installation

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

#### Export vers stdout

```bash
# JSON vers stdout
./bin/audit --period monthly --format json --output -

# CSV vers stdout
./bin/audit --period monthly --format csv --output -
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

---

## 📊 Sources de Données

Les rapports sont générés à partir de :

1. **Logs d'audit** (`audit/logs/`) : Événements journaliers signés
2. **Base de données PostgreSQL** : Documents, ledger, statistiques
3. **Signatures journalières** (`audit/signatures/`) : Preuves d'intégrité des logs

### Mode Dégradé

Si la base de données n'est pas disponible :
- Les statistiques documents seront vides (0)
- Les statistiques ledger seront vides (0)
- Les logs d'audit seront toujours collectés
- Le rapport sera généré avec les données disponibles

---

## ✅ Validation & Vérification

### Vérification du Hash

Le hash SHA256 du rapport est calculé sur le JSON complet (avant signature) :

```bash
# Calculer le hash d'un rapport JSON
sha256sum report-2025-01.json

# Comparer avec report.metadata.report_hash
cat report-2025-01.json | jq -r '.metadata.report_hash'
```

### Vérification de la Signature JWS

```bash
# Décoder le JWS (sans vérification)
echo "<report_jws>" | base64 -d | jq .

# Vérifier avec JWKS
curl https://vault.doreviateam.com/jwks.json | jq .
```

### Vérification du PDF

Le PDF contient un **QR code** sur la page de garde qui encode le hash SHA256 du rapport. Ce QR code peut être scanné pour vérifier l'intégrité du rapport.

---

## 🔧 Configuration

### Variables d'Environnement

| Variable | Description | Défaut |
|:---------|:------------|:-------|
| `AUDIT_DIR` | Répertoire audit | `/opt/dorevia-vault/audit` |
| `DATABASE_URL` | URL PostgreSQL | *(optionnel)* |
| `JWS_PRIVATE_KEY_PATH` | Chemin clé privée JWS | *(optionnel)* |
| `JWS_PUBLIC_KEY_PATH` | Chemin clé publique JWS | *(optionnel)* |
| `JWS_KID` | Key ID pour JWKS | `key-2025-Q1` |

### Prérequis

- **Logs d'audit** : Doivent être disponibles dans `AUDIT_DIR/logs/`
- **Base de données** : Optionnelle, mais recommandée pour statistiques complètes
- **Clés JWS** : Requises uniquement si `--sign` est utilisé

---

## 📝 Exemples Complets

### Exemple 1 : Rapport Mensuel JSON

```bash
./bin/audit --period monthly --year 2025 --month 1 --format json --output report-2025-01.json
```

**Résultat** : Fichier `report-2025-01.json` avec toutes les statistiques de janvier 2025.

### Exemple 2 : Rapport Trimestriel PDF Signé

```bash
./bin/audit \
  --period quarterly \
  --year 2025 \
  --quarter 1 \
  --format pdf \
  --sign \
  --output report-Q1-2025.pdf
```

**Résultat** : Fichier `report-Q1-2025.pdf` (8 pages) avec QR code et signature JWS.

### Exemple 3 : Rapport Personnalisé CSV

```bash
./bin/audit \
  --period custom \
  --from 2025-01-15 \
  --to 2025-01-31 \
  --format csv \
  --output report-custom.csv
```

**Résultat** : Fichier CSV avec une ligne de données pour la période du 15 au 31 janvier 2025.

### Exemple 4 : Pipeline Automatisé

```bash
#!/bin/bash
# Générer rapport mensuel automatiquement

YEAR=$(date +%Y)
MONTH=$(date +%m)
OUTPUT_DIR="/opt/dorevia-vault/reports"

# JSON
./bin/audit \
  --period monthly \
  --year $YEAR \
  --month $MONTH \
  --format json \
  --sign \
  --output "$OUTPUT_DIR/report-$YEAR-$MONTH.json"

# PDF
./bin/audit \
  --period monthly \
  --year $YEAR \
  --month $MONTH \
  --format pdf \
  --sign \
  --output "$OUTPUT_DIR/report-$YEAR-$MONTH.pdf"
```

---

## 🚨 Gestion d'Erreurs

### Erreurs Communes

| Erreur | Cause | Solution |
|:-------|:------|:---------|
| `--period is required` | Flag `--period` manquant | Ajouter `--period monthly|quarterly|custom` |
| `--from is required for custom period` | Date début manquante | Ajouter `--from YYYY-MM-DD` |
| `invalid --from date format` | Format date invalide | Utiliser `YYYY-MM-DD` |
| `JWS service not available` | Clés JWS manquantes | Configurer `JWS_PRIVATE_KEY_PATH` |
| `Failed to connect to database` | DB indisponible | Vérifier `DATABASE_URL` ou continuer sans DB |

### Codes de Sortie

- **0** : Succès
- **1** : Erreur (validation, génération, export)

---

## 📚 Références

- **RFC 7515** : JSON Web Signature (JWS)
- **RFC 7517** : JSON Web Key (JWK)
- **Sprint 4 Phase 4.2** : Journalisation auditable
- **Sprint 4 Phase 4.4** : Audit & conformité

---

**Document créé le** : Janvier 2025  
**Version** : 1.0  
**Auteur** : Doreviateam

© 2025 Doreviateam | Projet Dorevia Vault — v1.2-dev

