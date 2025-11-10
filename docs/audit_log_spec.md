# 📋 Spécification — Journalisation Auditable Dorevia Vault

**Version** : v1.2-dev  
**Date** : Novembre 2025  
**Sprint** : Sprint 4 Phase 4.2  
**Statut** : ✅ Implémenté

---

## 🎯 Objectif

Rendre **Dorevia Vault auditable par conception** en journalisant chaque événement critique avec :
- **Format standardisé** : JSONL (1 ligne = 1 événement)
- **Signature journalière** : Hash cumulé SHA256 + JWS
- **Export paginé** : JSON/CSV avec pagination
- **Rotation automatique** : Fichiers quotidiens + rétention configurable

---

## 📁 Architecture de Stockage

```
/opt/dorevia-vault/audit/
├── logs/
│   ├── audit-2025-01-15.log      # Logs JSONL du jour
│   ├── audit-2025-01-16.log
│   └── ...
├── signatures/
│   ├── audit-2025-01-15.log.jws # Signature JWS du hash journalier
│   └── ...
└── index.json                     # Index des logs (futur)
```

**Répertoire par défaut** : `/opt/dorevia-vault/audit` (configurable via `AUDIT_DIR`)

---

## 📝 Format Événement Audit

### Structure JSON

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "event_type": "document_vaulted|jws_signed|ledger_appended|reconciliation_run|verification_run|document_downloaded|error",
  "document_id": "uuid",
  "request_id": "uuid",
  "source": "sales|purchase|pos|stock|sale|unknown",
  "status": "success|error|idempotent",
  "duration_ms": 123,
  "metadata": {
    "filename": "invoice.pdf",
    "size_bytes": 12345,
    "sha256_hex": "abc...",
    "odoo_id": 123,
    "model": "account.move"
  }
}
```

### Champs Obligatoires

| Champ | Type | Description |
|:------|:-----|:------------|
| `timestamp` | string (RFC3339) | Horodatage UTC de l'événement |
| `event_type` | string | Type d'événement (voir ci-dessous) |
| `status` | string | Statut : `success`, `error`, `idempotent` |

### Champs Optionnels

| Champ | Type | Description |
|:------|:-----|:------------|
| `document_id` | string (UUID) | ID du document concerné |
| `request_id` | string (UUID) | ID de la requête HTTP (X-Request-ID) |
| `source` | string | Source Odoo : `sales`, `purchase`, `pos`, `stock`, `sale`, `unknown` |
| `duration_ms` | int64 | Durée de l'opération en millisecondes |
| `metadata` | object | Métadonnées spécifiques à l'événement |

### Types d'Événements

| Type | Description | Champs Métadonnées |
|:------|:------------|:-------------------|
| `document_vaulted` | Document stocké dans le vault | `filename`, `size_bytes`, `sha256_hex`, `odoo_id`, `model`, `evidence_jws`, `ledger_hash` |
| `jws_signed` | Document signé avec JWS | `sha256_hex`, `kid` |
| `ledger_appended` | Entrée ajoutée au ledger | `document_id`, `hash`, `previous_hash` |
| `reconciliation_run` | Exécution réconciliation | `orphans_found`, `orphans_fixed`, `dry_run` |
| `verification_run` | Vérification intégrité | `valid`, `signed_proof`, `checks` |
| `document_downloaded` | Téléchargement document | `document_id`, `filename` |
| `error` | Erreur système | `error`, `component` |

---

## 🔐 Signature Journalière

### Algorithme Hash Cumulé

Pour chaque ligne du fichier JSONL :

```
hash_0 = SHA256(line_1)
hash_1 = SHA256(hash_0 + line_2)
hash_2 = SHA256(hash_1 + line_3)
...
hash_final = SHA256(hash_n-1 + line_n)
```

**Avantage** : Calcul en O(1) par ligne (pas besoin de re-hasher tout le fichier)

### Format Signature

Le fichier `.jws` contient un JSON :

```json
{
  "date": "2025-01-15",
  "hash": "abc123...",
  "jws": "eyJhbGciOiJSUzI1NiIs...",
  "line_count": 1234,
  "timestamp": "2025-01-16T00:00:00Z"
}
```

**Vérification** :
1. Recalculer le hash cumulé du fichier `.log`
2. Comparer avec `hash` dans `.jws`
3. Vérifier la signature JWS avec `/jwks.json`

---

## 📤 Export Paginé

### Endpoint `/audit/export`

**Méthode** : `GET`  
**Format** : JSON ou CSV

**Paramètres de requête** :

| Paramètre | Type | Défaut | Description |
|:---------|:-----|:-------|:------------|
| `from` | string (YYYY-MM-DD) | Aujourd'hui | Date de début |
| `to` | string (YYYY-MM-DD) | Aujourd'hui | Date de fin |
| `page` | int | 1 | Numéro de page |
| `limit` | int | 1000 | Nombre de lignes par page (max: 10000) |
| `format` | string | `json` | Format : `json` ou `csv` |

**Exemple** :

```bash
# Export JSON paginé
curl "https://vault.doreviateam.com/audit/export?from=2025-01-15&to=2025-01-17&page=1&limit=100&format=json"

# Export CSV
curl "https://vault.doreviateam.com/audit/export?format=csv" > audit.csv
```

**Réponse JSON** :

```json
{
  "events": [...],
  "total": 1234,
  "page": 1,
  "limit": 100,
  "total_pages": 13,
  "has_next": true,
  "has_previous": false
}
```

### Endpoint `/audit/dates`

**Méthode** : `GET`  
**Description** : Liste les dates disponibles dans les logs

**Réponse** :

```json
{
  "dates": ["2025-01-15", "2025-01-16", "2025-01-17"],
  "count": 3
}
```

---

## 🔄 Rotation & Rétention

### Rotation Automatique

- **Fichier quotidien** : `audit-YYYY-MM-DD.log`
- **Signature quotidienne** : `audit-YYYY-MM-DD.log.jws` (à 00:00 UTC)
- **Buffer** : Flush automatique toutes les 10s ou 1000 lignes

### Rétention

- **Par défaut** : 90 jours
- **Configurable** : Via `Rotator.CleanupOldLogs()`
- **Suppression** : Logs + signatures plus anciens que la période de rétention

### Statistiques de Rétention

```go
stats, err := rotator.GetRetentionStats()
// Retourne :
// - TotalLogs : Nombre total de fichiers de log
// - OldestLogDate : Date du log le plus ancien
// - NewestLogDate : Date du log le plus récent
// - RetentionDays : Période de rétention configurée
// - LogsToDelete : Nombre de logs à supprimer au prochain cleanup
```

---

## ⚙️ Configuration

### Variables d'Environnement

| Variable | Défaut | Description |
|:---------|:-------|:------------|
| `AUDIT_DIR` | `/opt/dorevia-vault/audit` | Répertoire racine audit |

### Configuration Programme

```go
cfg := audit.Config{
    AuditDir:      "/opt/dorevia-vault/audit",
    MaxBuffer:     1000,              // Nombre max de lignes avant flush
    FlushInterval: 10 * time.Second,  // Intervalle max avant flush
    Logger:        log,                // Logger pour logs internes
}
```

---

## 🔧 Utilisation

### Initialisation

```go
import "github.com/doreviateam/dorevia-vault/internal/audit"

// Créer le logger
auditLogger, err := audit.NewLogger(audit.Config{
    AuditDir:      "/opt/dorevia-vault/audit",
    MaxBuffer:     1000,
    FlushInterval: 10 * time.Second,
    Logger:        log,
})
if err != nil {
    log.Fatal().Err(err).Msg("Failed to initialize audit logger")
}
defer auditLogger.Close()
```

### Logger un Événement

```go
auditLogger.Log(audit.Event{
    EventType:  audit.EventTypeDocumentVaulted,
    DocumentID: docID.String(),
    RequestID:  requestID,
    Source:     "sales",
    Status:     audit.EventStatusSuccess,
    DurationMS: int64(duration.Milliseconds()),
    Metadata: map[string]interface{}{
        "filename":   "invoice.pdf",
        "size_bytes": 12345,
        "sha256_hex": sha256Hex,
    },
})
```

### Signature Journalière

```go
signer := audit.NewSigner(auditLogger, jwsService, log)
dailyHash, err := signer.SignDailyLog("2025-01-15")
// Génère audit-2025-01-15.log.jws
```

### Export

```go
exporter := audit.NewExporter(auditLogger)
result, err := exporter.Export(audit.ExportOptions{
    From:   "2025-01-15",
    To:     "2025-01-17",
    Page:   1,
    Limit:  1000,
    Format: audit.ExportFormatJSON,
})
```

### Rotation & Rétention

```go
rotator := audit.NewRotator(auditLogger, audit.RotationConfig{
    RetentionDays: 90,
    SignDaily:     true,
    Signer:        signer,
    Logger:        log,
})

// Rotation quotidienne (à appeler via cron à 00:00 UTC)
err := rotator.RotateDaily()

// Nettoyage des anciens logs
err := rotator.CleanupOldLogs()
```

---

## 🧪 Tests

**16 tests unitaires** couvrent :
- Création et configuration du logger
- Écriture JSONL avec buffer
- Flush automatique
- Export paginé
- Export CSV
- Liste des dates disponibles
- Gestion des erreurs

**Exécution** :

```bash
go test ./tests/unit/... -run TestAudit -v
go test ./tests/unit/... -run TestExporter -v
```

---

## 📊 Performance

- **Buffer** : 1000 lignes ou 10s (configurable)
- **Hash incrémental** : O(1) par ligne
- **Export paginé** : Limite 10000 lignes par page
- **Rotation** : Automatique à 00:00 UTC

---

## 🔒 Sécurité

- **Permissions fichiers** : 0644 (logs), 0644 (signatures)
- **Permissions répertoires** : 0755
- **Signature JWS** : RS256 (RSA-SHA256)
- **Hash** : SHA256 cumulé
- **Vérification** : Via `/jwks.json` public

---

## 📚 Références

- **Module** : `internal/audit/`
- **Handlers** : `internal/handlers/audit.go`
- **Tests** : `tests/unit/audit_*.go`
- **Plan Sprint 4** : `docs/Dorevia_Vault_Sprint4.md`

---

**Document créé le** : Novembre 2025  
**Auteur** : Doreviateam  
**Version** : v1.2-dev (Sprint 4 Phase 4.2)

