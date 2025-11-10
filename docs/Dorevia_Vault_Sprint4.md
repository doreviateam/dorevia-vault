# 🚀 Dorevia Vault — Sprint 4 : Observabilité & Auditabilité Continue

**Version** : v1.2-dev  
**Date de démarrage** : Février 2025  
**Responsable** : Doreviateam (David Baron)  
**Statut** : 🟡 Plan révisé — Corrections appliquées selon analyse experte  
**Durée révisée** : **16 jours** (au lieu de 11 jours initialement prévus)

---

## 🧭 Objectif global

Rendre **Dorevia Vault auditable par conception**.  
Chaque événement (stockage, scellement, réconciliation, vérification) doit être :
- Mesurable (via métriques Prometheus)
- Traçable (via logs signés)
- Corrélable (via RequestID / LedgerID)
- Exportable (via rapports JSON/PDF signés)

---

## 🎯 Buts concrets

1. **Observer** → exposer toutes les métriques du système.  
2. **Alerter** → détecter anomalies, lenteurs ou échecs.  
3. **Auditer** → consigner chaque opération avec preuve JWS.  
4. **Archiver** → exporter les rapports d'audit complets.

---

## 🧱 Structure du Sprint 4

| Phase | Nom | Objectif | Durée révisée |
|:--|:--|:--|:--|
| 4.0 | Corrections document | Harmoniser métriques + détails techniques | 1 jour |
| 4.1 | Observabilité avancée | Étendre Prometheus + Grafana + métriques système | 4 jours |
| 4.2 | Journalisation auditable | Créer logs signés et exportables | 4 jours |
| 4.3 | Alerting & supervision | Configurer alertes + seuils Prometheus | 3 jours |
| 4.4 | Audit & conformité | Générer rapport auditable trimestriel | 4 jours |

**Durée totale** : **16 jours** (au lieu de 11 jours)

---

## 🔧 Phase 4.0 — Corrections Document (J1)

### Objectifs
- Harmoniser les noms de métriques avec le code existant
- Ajouter les détails techniques manquants
- Définir les seuils d'alerte précis
- Préciser les dépendances externes

### Corrections appliquées

**1. Noms de métriques harmonisés** :
- ❌ `vault_documents_total` → ✅ `documents_vaulted_total` (existant Sprint 3)
- ✅ `jws_signatures_total{status="error"}` (existant, pas besoin de `jws_failures_total`)
- ✅ `reconciliation_runs_total{status}` (existant Sprint 3)
- ✅ `ledger_append_errors_total` (à créer)

**2. Dépendances externes précisées** :
- Grafana : v10.0+ (installation Docker recommandée)
- Alertmanager : v0.26+ (installation Docker recommandée)
- `gopsutil` : v3.23+ (compatible Go 1.23+)

**3. Seuils d'alerte définis** :
- Taux d'erreur documents : > 10% sur 5 minutes
- Ledger append lent : P95 > 2s pendant 10 minutes
- Stockage plein : > 80% de capacité pendant 1 heure

---

## 🔍 Phase 4.1 — Observabilité avancée (J2-J5)

### Objectifs
- Compléter les métriques existantes avec `ledger_append_errors_total`
- Ajouter métriques système (`/metrics` avec labels système) : RAM, CPU, disque (via `gopsutil`)
- Intégrer Grafana : latence, erreurs, volumétrie, stockage
- Dashboard complet avec noms de métriques corrects

### Métriques à ajouter

**Nouvelles métriques** :
- `ledger_append_errors_total` (counter) : Nombre total d'erreurs lors de l'ajout au ledger
- `system_cpu_usage_percent` (gauge) : Utilisation CPU en pourcentage
- `system_memory_usage_bytes` (gauge) : Utilisation mémoire en octets
- `system_disk_usage_bytes` (gauge) : Utilisation disque en octets
- `system_disk_capacity_bytes` (gauge) : Capacité disque totale en octets

**Métriques existantes à utiliser** (Sprint 3) :
- `documents_vaulted_total{status, source}` ✅
- `jws_signatures_total{status}` ✅
- `ledger_entries_total` ✅
- `reconciliation_runs_total{status}` ✅
- `document_storage_duration_seconds{operation}` ✅
- `jws_signature_duration_seconds` ✅
- `ledger_append_duration_seconds` ✅
- `transaction_duration_seconds` ✅
- `ledger_size` ✅
- `storage_size_bytes` ✅
- `active_connections` ✅

### Livrables
- `internal/metrics/system.go` : Module métriques système avec `gopsutil`
- `internal/metrics/prometheus.go` : Ajout `LedgerAppendErrors` counter
- `docs/observability_metrics_spec.md` : Spécification complète avec noms corrects
- `grafana/dashboard_vault.json` : Dashboard avec toutes les métriques (noms harmonisés)

### Prérequis
- `github.com/shirou/gopsutil/v3` v3.23+
- Grafana v10.0+ (installation Docker recommandée)
- Prometheus déjà configuré (Sprint 3)

---

## 🧾 Phase 4.2 — Journalisation auditable (J6-J9)

### Objectifs
- Nouveau module `internal/audit/`
- Format log : JSONL (1 ligne par événement)
- Signature journalière : hash cumulé incrémental (SHA256 + JWS)
- Rotation automatique : `audit-YYYY-MM-DD.log` à 00:00 UTC
- Endpoint `/audit/export?from=YYYY-MM-DD&to=YYYY-MM-DD&page=1&limit=1000&format=json`
- Rétention : 90 jours par défaut (configurable)

### Architecture de stockage

```
/opt/dorevia-vault/audit/
├── logs/
│   ├── audit-2025-01-15.log
│   ├── audit-2025-01-16.log
│   └── ...
├── signatures/
│   ├── audit-2025-01-15.log.jws
│   └── ...
└── index.json  # Index des logs pour recherche rapide
```

### Format événement audit

```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "event_type": "document_vaulted|jws_signed|ledger_appended|reconciliation_run|verification_run",
  "document_id": "uuid",
  "request_id": "uuid",
  "source": "sales|purchase|pos|stock|sale|unknown",
  "status": "success|error|idempotent",
  "duration_ms": 123,
  "metadata": {
    "filename": "invoice.pdf",
    "size_bytes": 12345,
    "sha256_hex": "abc..."
  }
}
```

### Signature journalière optimisée

```go
// Hash cumulé incrémental (plus performant)
type DailyHash struct {
    Date      string  // YYYY-MM-DD
    Hash      string  // SHA256 cumulé
    JWS       string  // Signature JWS du hash
    LineCount int64   // Nombre de lignes signées
}
```

### Livrables
- `internal/audit/log.go` : Writer JSONL avec buffer
- `internal/audit/sign.go` : Signature journalière optimisée (hash incrémental)
- `internal/audit/export.go` : Export avec pagination
- `internal/audit/rotation.go` : Rotation automatique + rétention
- `docs/audit_log_spec.md` : Format événement + architecture complète

### Performance
- Buffer JSONL : 1000 lignes ou 10s (flush automatique)
- Hash incrémental : Calcul en O(1) par ligne
- Export paginé : Limite 1000 lignes par page (configurable)

---

## 🚨 Phase 4.3 — Alerting & supervision (J10-J12)

### Objectifs
- Définir alertes Prometheus avec seuils précis
- Configurer Alertmanager (Slack ou webhook interne)
- Export possible vers Odoo (`ir.logging`)

### Règles d'alerte détaillées

**1. Taux d'erreur documents élevé** :
```yaml
- alert: HighDocumentErrorRate
  expr: rate(documents_vaulted_total{status="error"}[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Taux d'erreur élevé (>10% sur 5min)"
    description: "{{ $value | humanizePercentage }} des documents échouent"
```

**2. Ledger append lent** :
```yaml
- alert: SlowLedgerAppend
  expr: histogram_quantile(0.95, ledger_append_duration_seconds) > 2
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Ledger append lent (P95 > 2s)"
    description: "P95 = {{ $value }}s"
```

**3. Stockage presque plein** :
```yaml
- alert: StorageNearlyFull
  expr: storage_size_bytes / system_disk_capacity_bytes > 0.8
  for: 1h
  labels:
    severity: critical
  annotations:
    summary: "Stockage > 80% de capacité"
    description: "{{ $value | humanizePercentage }} utilisé"
```

**4. Erreurs ledger fréquentes** :
```yaml
- alert: FrequentLedgerErrors
  expr: rate(ledger_append_errors_total[5m]) > 0.05
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Erreurs ledger fréquentes"
```

### Configuration Alertmanager

```yaml
# alertmanager.yml
route:
  receiver: 'slack-default'
  group_by: ['alertname', 'severity']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h

receivers:
  - name: 'slack-default'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/...'
        channel: '#dorevia-vault-alerts'
        title: 'Dorevia Vault Alert'
        text: '{{ .GroupLabels.alertname }}: {{ .CommonAnnotations.summary }}'
```

### Export Odoo

**Format** : `ir.logging` (Odoo)
```go
type OdooLogEntry struct {
    Name     string  // "dorevia.vault"
    Type     string  // "server"
    Level    string  // "error" | "warning" | "info"
    Message  string
    Func     string  // "document_vaulted"
    Line     int
    Path     string  // "dorevia-vault"
}
```

**Endpoint Odoo** : `POST /web/dataset/call_kw/ir.logging/create`

### Livrables
- `alert_rules.yml` : Règles Prometheus détaillées avec seuils
- `alertmanager.yml` : Configuration complète Alertmanager
- `internal/audit/odoo_export.go` : Export vers Odoo
- `docs/alerting_rules_spec.md` : Seuils + justification

### Prérequis
- Alertmanager v0.26+ (installation Docker recommandée)
- Slack webhook ou webhook interne configuré
- Accès Odoo avec authentification

---

## 📑 Phase 4.4 — Audit & conformité (J13-J16)

### Objectifs
- Générer rapport consolidé (mensuel / trimestriel)
- Contenu : documents, erreurs, durées moyennes, ledger, réconciliations
- Export JSON + CSV + PDF signé
- CLI : `cmd/audit/main.go` pour génération manuelle

### Contenu rapport détaillé

```go
type AuditReport struct {
    Period          Period              // Période (mensuel/trimestriel)
    Summary         ReportSummary       // Résumé exécutif
    Documents       DocumentStats       // Statistiques documents
    Errors          ErrorStats          // Statistiques erreurs
    Performance     PerformanceStats    // Durées moyennes (P50, P95, P99)
    Ledger          LedgerStats         // Statistiques ledger
    Reconciliation  ReconciliationStats // Réconciliations
    Signatures      []DailySignature    // Signatures journalières
    Metadata        ReportMetadata      // Métadonnées (généré le, version, etc.)
}
```

### Génération PDF

- Utiliser `github.com/jung-kurt/gofpdf` ou `github.com/signintech/gopdf`
- Template professionnel : en-tête, pied de page, graphiques
- Signature JWS : QR code du hash SHA256 du rapport JSON

### CLI

```bash
# Génération rapport mensuel
./bin/audit --period monthly --format json --output report-2025-01.json

# Génération rapport trimestriel avec PDF
./bin/audit --period quarterly --format pdf --output report-Q1-2025.pdf

# Génération avec signature JWS
./bin/audit --period monthly --format json --sign --output report-2025-01.json
```

### Livrables
- `cmd/audit/main.go` : CLI avec flags (`--period`, `--format`, `--output`, `--sign`)
- `internal/audit/report.go` : Génération rapport (JSON/CSV)
- `internal/audit/pdf.go` : Génération PDF avec template
- `docs/audit_export_spec.md` : Format rapport + exemples

---

## 🧪 Tests & validation

| Type | Description | Résultat attendu |
|:--|:--|:--|
| **Unitaires** | Audit log, métriques système, export | 100% succès |
| **Intégration** | Prometheus + Grafana | OK (scrape 15s) |
| **Performance** | Signature journalière (10k lignes) | < 5s |
| **Charge** | 10 000 documents simulés | < 200ms latence |
| **Vérification** | Hash JWS journaliers cohérents | OK |
| **Export** | Export 30 jours avec pagination | < 10s par page |
| **Rotation** | Rotation automatique + rétention | OK |

### Tests détaillés par phase

**Phase 4.1** :
- Tests métriques système (CPU, RAM, disque)
- Tests intégration Grafana (dashboard fonctionnel)

**Phase 4.2** :
- Tests signature journalière (intégrité hash)
- Tests rotation logs (automatique à 00:00 UTC)
- Tests export avec pagination (gros volumes)

**Phase 4.3** :
- Tests alertes Prometheus (déclenchement correct)
- Tests Alertmanager (routing, grouping)
- Tests export Odoo (format correct)

**Phase 4.4** :
- Tests génération rapport (JSON, CSV, PDF)
- Tests signature rapport (JWS valide)
- Tests CLI (tous les flags)

---

## 🧰 Livrables techniques

| Type | Fichier | Description |
|:--|:--|:--|
| Go pkg | `internal/audit/` | Gestion du journal d'audit (log, sign, export, rotation) |
| Go pkg | `internal/metrics/system.go` | Métriques système (CPU, RAM, disque) |
| Go pkg | `internal/metrics/prometheus.go` | Ajout `LedgerAppendErrors` counter |
| CLI | `cmd/audit/main.go` | Génération rapport d'audit |
| Config | `alert_rules.yml` | Règles Prometheus avec seuils |
| Config | `alertmanager.yml` | Configuration Alertmanager |
| Docs | `docs/Dorevia_Vault_Sprint4_Observabilite_Auditabilite.md` | Suivi complet Sprint 4 |
| Dashboards | `grafana/dashboard_vault.json` | Visualisation Prometheus (noms harmonisés) |

---

## 🔒 Sécurité

### Audit logs
- Permissions : `600` (lecture/écriture propriétaire uniquement)
- Chiffrement au repos : Optionnel (Sprint 5)
- Signature JWS : Clé privée stockée sécurisée (`/opt/dorevia-vault/keys/`)

### Export audit
- Authentification : À venir (Sprint 5)
- Rate limiting : 10 exports/heure par IP
- Pagination obligatoire : Max 1000 lignes par page

### Rapports
- Signature JWS : Obligatoire pour PDF
- Stockage : `/opt/dorevia-vault/reports/` (permissions 600)

---

## 🚀 Résultat attendu

À la fin du Sprint 4, **Dorevia Vault** devient :
- ✅ Supervisé en temps réel (Prometheus + Grafana avec métriques système)
- ✅ Auditabilité complète (journaux signés avec rotation automatique)
- ✅ Alerte automatique sur anomalies (seuils précis configurés)
- ✅ Conforme aux standards PDP/PPF 2026 (rapports signés exportables)

---

## 📋 Checklist de validation

### Phase 4.0 — Corrections
- [x] Noms métriques harmonisés
- [x] Dépendances externes précisées
- [x] Seuils d'alerte définis
- [x] Durées révisées (16 jours)

### Phase 4.1 — Observabilité
- [ ] Métriques système implémentées
- [ ] `ledger_append_errors_total` ajouté
- [ ] Dashboard Grafana fonctionnel
- [ ] Tests métriques système OK

### Phase 4.2 — Journalisation
- [ ] Module audit/log.go créé
- [ ] Signature journalière optimisée
- [ ] Endpoint export avec pagination
- [ ] Rotation automatique fonctionnelle

### Phase 4.3 — Alerting
- [ ] Règles Prometheus détaillées
- [ ] Alertmanager configuré
- [ ] Export Odoo fonctionnel

### Phase 4.4 — Audit
- [ ] Module report.go créé
- [ ] Génération PDF avec template
- [ ] CLI cmd/audit/main.go fonctionnel

---

**Document révisé le** : Janvier 2025  
**Basé sur** : Analyse experte `docs/ANALYSE_EXPERT_SPRINT4.md`  
**Version** : v1.2-dev (Sprint 4 révisé)

© 2025 Doreviateam | Projet Dorevia Vault — v1.2-dev
