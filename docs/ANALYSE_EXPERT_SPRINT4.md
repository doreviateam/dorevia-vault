# 🔍 Analyse Expert — Sprint 4 "Observabilité & Auditabilité Continue"

**Date** : Janvier 2025  
**Document analysé** : `docs/Dorevia_Vault_Sprint4.md`  
**Version code actuel** : v1.1-dev (Sprint 3 Phase 3 complétée)  
**Analyste** : Expert Technique (AMOA & DevOps)

---

## 🎯 Synthèse de l'Analyse

**Verdict global** : 🟡 **GO avec corrections majeures** — Plan solide mais nécessite ajustements techniques et réalistes

### Points Forts ✅
- Vision claire : "auditable par conception"
- Objectifs alignés avec besoins PDP/PPF 2026
- Structure en 4 phases logique
- Intégration Prometheus/Grafana pertinente

### Points d'Attention ⚠️
- **Incohérences de nommage** : Métriques déjà existantes mal nommées
- **Durées optimistes** : 11 jours pour 4 phases complexes
- **Gaps techniques** : Détails manquants sur implémentation
- **Dépendances externes** : Grafana, Alertmanager non précisées
- **Tests insuffisants** : Pas de tests d'intégration détaillés

---

## 📊 Analyse Détaillée par Phase

### Phase 4.1 — Observabilité avancée ⚠️ **CORRECTIONS NÉCESSAIRES**

#### Problèmes Identifiés

**1. Incohérence de nommage des métriques** ❌

Le document propose :
- `vault_documents_total{status,source}`

**Mais le code actuel utilise** :
- `documents_vaulted_total{status,source}` ✅ (déjà implémenté)

**Impact** : Confusion, duplication potentielle, breaking changes

**2. Métriques déjà existantes** ⚠️

Le document propose :
- `reconciliation_runs_total{status}` → ✅ **Déjà implémenté** (Sprint 3)
- `jws_failures_total` → ⚠️ **Partiellement** (`jws_signatures_total{status="error"}` existe)

**3. Métriques système manquantes** ✅

- `/metrics/system` avec `gopsutil` → ✅ **Nouveau, pertinent**
- RAM, CPU, disque → ✅ **Nécessaire pour supervision complète**

#### Préconisations Phase 4.1

**✅ À CORRIGER** :

1. **Harmoniser les noms de métriques** :
   ```go
   // ❌ Document propose : vault_documents_total
   // ✅ Code actuel : documents_vaulted_total
   // → GARDER documents_vaulted_total (cohérence Sprint 3)
   ```

2. **Compléter les métriques existantes** :
   ```go
   // Ajouter compteur d'erreurs ledger (actuellement manquant)
   LedgerAppendErrors = promauto.NewCounter(
       prometheus.CounterOpts{
           Name: "ledger_append_errors_total",
           Help: "Nombre total d'erreurs lors de l'ajout au ledger",
       },
   )
   
   // JWS failures : utiliser jws_signatures_total{status="error"}
   // → Pas besoin de nouvelle métrique, utiliser label existant
   ```

3. **Créer module métriques système** :
   ```go
   // internal/metrics/system.go
   // Utiliser github.com/shirou/gopsutil/v3
   // Exposer : cpu_usage_percent, memory_usage_bytes, disk_usage_bytes
   ```

4. **Endpoint `/metrics/system`** :
   ```go
   // Option 1 : Route séparée (recommandé)
   app.Get("/metrics/system", handlers.SystemMetricsHandler())
   
   // Option 2 : Label dans /metrics (plus standard Prometheus)
   // → Préférer Option 2 (standard Prometheus)
   ```

**📋 Livrables ajustés** :
- ✅ `internal/metrics/system.go` (nouveau)
- ✅ `internal/metrics/prometheus.go` (ajouter `LedgerAppendErrors`)
- ✅ `docs/observability_metrics_spec.md` (corriger noms métriques)
- ✅ `grafana/dashboard_vault.json` (utiliser noms corrects)

**⏱ Durée révisée** : **4 jours** (au lieu de 3)
- J1 : Métriques système (`gopsutil`)
- J2 : Compléter métriques manquantes + tests
- J3 : Dashboard Grafana
- J4 : Documentation + validation

---

### Phase 4.2 — Journalisation auditable ⚠️ **DÉTAILS MANQUANTS**

#### Points Forts ✅
- Format JSONL : Standard, lisible, traçable
- Signature journalière : Sécurité renforcée
- Rotation automatique : Gestion propre des logs

#### Problèmes Identifiés

**1. Performance et scalabilité** ⚠️

- **JSONL volumineux** : 10k documents/jour = ~50-100 MB/jour
- **Signature journalière** : Calcul SHA256 cumulé peut être lent
- **Stockage** : Où stocker les logs ? (`/var/log/dorevia-vault/` ?)

**2. Intégrité des logs** ⚠️

- **Hash cumulé** : Comment gérer les logs partiels en cas de crash ?
- **JWS journalier** : Nécessite clé privée accessible → Risque sécurité
- **Vérification** : Comment vérifier l'intégrité d'un log ancien ?

**3. Endpoint `/audit/export`** ⚠️

- **Performance** : Export de 30 jours = plusieurs GB → Timeout ?
- **Pagination** : Non mentionnée
- **Filtrage** : Par type d'événement ? Par source ?

#### Préconisations Phase 4.2

**✅ À AJOUTER** :

1. **Architecture de stockage** :
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

2. **Format événement audit** :
   ```json
   {
     "timestamp": "2025-01-15T10:30:00Z",
     "event_type": "document_vaulted|jws_signed|ledger_appended|reconciliation_run",
     "document_id": "uuid",
     "request_id": "uuid",
     "source": "sales|purchase|...",
     "status": "success|error",
     "duration_ms": 123,
     "metadata": {...}
   }
   ```

3. **Signature journalière optimisée** :
   ```go
   // Hash cumulé incrémental (plus performant)
   type DailyHash struct {
       Date     string
       Hash     string  // SHA256 cumulé
       JWS      string  // Signature JWS du hash
       LineCount int64  // Nombre de lignes signées
   }
   ```

4. **Endpoint export avec pagination** :
   ```go
   GET /audit/export?from=2025-01-01&to=2025-01-31&page=1&limit=1000&format=json
   ```

5. **Rotation et rétention** :
   - Rotation quotidienne à 00:00 UTC
   - Rétention : 90 jours par défaut (configurable)
   - Archivage automatique après 90 jours (tar.gz)

**📋 Livrables ajustés** :
- ✅ `internal/audit/log.go` (writer JSONL avec buffer)
- ✅ `internal/audit/sign.go` (signature journalière optimisée)
- ✅ `internal/audit/export.go` (export avec pagination)
- ✅ `internal/audit/rotation.go` (rotation automatique)
- ✅ `docs/audit_log_spec.md` (format événement + architecture)

**⏱ Durée révisée** : **4 jours** (au lieu de 3)
- J1 : Module audit/log.go (writer JSONL)
- J2 : Signature journalière + rotation
- J3 : Endpoint export avec pagination
- J4 : Tests + documentation

---

### Phase 4.3 — Alerting & supervision ⚠️ **CONFIGURATION MANQUANTE**

#### Points Forts ✅
- Alertes Prometheus : Standard, bien intégré
- Export Odoo : Intégration pertinente

#### Problèmes Identifiés

**1. Seuils non définis** ⚠️

- `vault_document_errors_total > 5` → Sur quelle période ? (5min ? 1h ?)
- `ledger_append_duration_seconds > 2` → P95 ? P99 ? Moyenne ?
- `storage_size_bytes > threshold` → Quelle valeur ? (80% disque ?)

**2. Alertmanager non précisé** ⚠️

- Installation ? Configuration ? Intégration Slack/Email ?
- Routes d'alerte ? Grouping ? Inhibition ?

**3. Export Odoo vague** ⚠️

- Format ? Endpoint Odoo ? Authentification ?
- Fréquence ? (Temps réel ? Batch ?)

#### Préconisations Phase 4.3

**✅ À PRÉCISER** :

1. **Règles d'alerte détaillées** :
   ```yaml
   # alert_rules.yml
   groups:
     - name: dorevia_vault
       interval: 30s
       rules:
         - alert: HighDocumentErrorRate
           expr: rate(documents_vaulted_total{status="error"}[5m]) > 0.1
           for: 5m
           labels:
             severity: warning
           annotations:
             summary: "Taux d'erreur élevé (>10% sur 5min)"
         
         - alert: SlowLedgerAppend
           expr: histogram_quantile(0.95, ledger_append_duration_seconds) > 2
           for: 10m
           labels:
             severity: warning
           annotations:
             summary: "Ledger append lent (P95 > 2s)"
         
         - alert: StorageNearlyFull
           expr: storage_size_bytes / storage_capacity_bytes > 0.8
           for: 1h
           labels:
             severity: critical
           annotations:
             summary: "Stockage > 80% de capacité"
   ```

2. **Configuration Alertmanager** :
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
   ```

3. **Export Odoo** :
   ```go
   // Format : ir.logging (Odoo)
   type OdooLogEntry struct {
       Name     string  // "dorevia.vault"
       Type     string  // "server"
       Level    string  // "error" | "warning" | "info"
       Message  string
       Func     string  // "document_vaulted"
       Line     int
       Path     string  // "dorevia-vault"
   }
   
   // Endpoint Odoo : POST /web/dataset/call_kw/ir.logging/create
   ```

**📋 Livrables ajustés** :
- ✅ `alert_rules.yml` (règles détaillées avec seuils)
- ✅ `alertmanager.yml` (configuration complète)
- ✅ `internal/audit/odoo_export.go` (export vers Odoo)
- ✅ `docs/alerting_rules_spec.md` (seuils + justification)

**⏱ Durée révisée** : **3 jours** (au lieu de 2)
- J1 : Règles Prometheus détaillées
- J2 : Configuration Alertmanager + tests
- J3 : Export Odoo + documentation

---

### Phase 4.4 — Audit & conformité ✅ **BIEN DÉFINIE**

#### Points Forts ✅
- Rapport consolidé : Nécessaire pour conformité
- Export multi-format : JSON, CSV, PDF
- CLI : Génération manuelle utile

#### Préconisations Phase 4.4

**✅ À AJOUTER** :

1. **Contenu rapport détaillé** :
   ```go
   type AuditReport struct {
       Period       Period          // Période (mensuel/trimestriel)
       Summary      ReportSummary   // Résumé exécutif
       Documents    DocumentStats   // Statistiques documents
       Errors       ErrorStats      // Statistiques erreurs
       Performance  PerformanceStats // Durées moyennes
       Ledger       LedgerStats     // Statistiques ledger
       Reconciliation ReconciliationStats // Réconciliations
       Signatures   []DailySignature // Signatures journalières
   }
   ```

2. **Génération PDF** :
   - Utiliser `github.com/jung-kurt/gofpdf` ou `github.com/signintech/gopdf`
   - Template professionnel (en-tête, pied de page, graphiques)

3. **Signature rapport** :
   - JWS du hash SHA256 du rapport JSON
   - Inclure dans PDF (QR code ?)

**📋 Livrables ajustés** :
- ✅ `cmd/audit/main.go` (CLI avec flags : --period, --format, --output)
- ✅ `internal/audit/report.go` (génération rapport)
- ✅ `internal/audit/pdf.go` (génération PDF)
- ✅ `docs/audit_export_spec.md` (format rapport + exemples)

**⏱ Durée révisée** : **4 jours** (au lieu de 3)
- J1 : Module report.go (génération JSON/CSV)
- J2 : Génération PDF + signature
- J3 : CLI cmd/audit/main.go
- J4 : Tests + documentation

---

## 🔍 Points d'Attention Globaux

### 1. Durée Totale ⚠️ **OPTIMISTE**

**Document propose** : 11 jours (3+3+2+3)  
**Réaliste** : **15 jours** (4+4+3+4)

**Justification** :
- Phase 4.1 : Métriques système + Grafana = 4 jours
- Phase 4.2 : Audit logs + signature + export = 4 jours
- Phase 4.3 : Alertes + Alertmanager + Odoo = 3 jours
- Phase 4.4 : Rapport + PDF + CLI = 4 jours

### 2. Tests Insuffisants ⚠️

**Document mentionne** :
- Tests unitaires : Audit log, métriques, export
- Tests intégration : Prometheus + Grafana
- Tests charge : 10 000 documents

**Manque** :
- Tests de signature journalière (intégrité)
- Tests de rotation de logs
- Tests d'export avec gros volumes
- Tests de performance (signature JWS journalière)

**Préconisation** : Ajouter section "Tests détaillés" dans chaque phase

### 3. Dépendances Externes ⚠️

**Non précisées** :
- Grafana : Installation ? Version ? Configuration ?
- Alertmanager : Installation ? Version ?
- `gopsutil` : Version ? Compatibilité Go ?

**Préconisation** : Ajouter section "Prérequis" avec versions précises

### 4. Sécurité ⚠️

**Points à clarifier** :
- Clés JWS pour signature journalière : Stockage sécurisé ?
- Logs audit : Permissions ? Chiffrement au repos ?
- Export audit : Authentification requise ?

**Préconisation** : Ajouter section "Sécurité" dans chaque phase

---

## 📋 Préconisations Techniques Prioritaires

### Priorité 1 : Corrections Critiques 🔴

1. **Harmoniser noms métriques** :
   - Utiliser `documents_vaulted_total` (existant)
   - Ne pas créer `vault_documents_total` (duplication)

2. **Compléter métriques manquantes** :
   - `ledger_append_errors_total` (nouveau compteur)
   - Utiliser `jws_signatures_total{status="error"}` (existant)

3. **Définir seuils d'alerte précis** :
   - Périodes (5min, 1h)
   - Quantiles (P95, P99)
   - Valeurs absolues (80% disque)

### Priorité 2 : Améliorations Techniques 🟡

1. **Architecture audit logs** :
   - Structure répertoires
   - Format événement standardisé
   - Rotation + rétention

2. **Performance** :
   - Hash cumulé incrémental
   - Pagination export
   - Buffer pour JSONL writer

3. **Intégration** :
   - Configuration Alertmanager complète
   - Format export Odoo précis
   - Template PDF professionnel

### Priorité 3 : Documentation & Tests 🟢

1. **Documentation technique** :
   - Format événement audit
   - Architecture stockage logs
   - Seuils d'alerte justifiés

2. **Tests complets** :
   - Tests signature journalière
   - Tests rotation logs
   - Tests export gros volumes
   - Tests performance

---

## ✅ Plan d'Action Recommandé

### Étape 1 : Corrections Document (1 jour)

- [ ] Corriger noms métriques dans document
- [ ] Ajouter détails techniques manquants
- [ ] Définir seuils d'alerte précis
- [ ] Préciser dépendances externes

### Étape 2 : Phase 4.1 Révisée (4 jours)

- [ ] Créer `internal/metrics/system.go`
- [ ] Ajouter `ledger_append_errors_total`
- [ ] Dashboard Grafana avec noms corrects
- [ ] Tests métriques système

### Étape 3 : Phase 4.2 Révisée (4 jours)

- [ ] Module audit/log.go (JSONL writer)
- [ ] Signature journalière optimisée
- [ ] Endpoint export avec pagination
- [ ] Rotation automatique

### Étape 4 : Phase 4.3 Révisée (3 jours)

- [ ] Règles Prometheus détaillées
- [ ] Configuration Alertmanager
- [ ] Export Odoo

### Étape 5 : Phase 4.4 Révisée (4 jours)

- [ ] Module report.go
- [ ] Génération PDF
- [ ] CLI cmd/audit/main.go

**Durée totale révisée** : **16 jours** (1 correction + 15 implémentation)

---

## 🎯 Conclusion

**Verdict** : 🟡 **GO avec corrections majeures**

Le plan Sprint 4 est **solide** et **aligné** avec les objectifs, mais nécessite :
- ✅ **Corrections techniques** : Noms métriques, seuils, détails
- ✅ **Ajustements réalistes** : Durées, dépendances, tests
- ✅ **Améliorations techniques** : Performance, sécurité, architecture

**Recommandation** : Valider les corrections avant démarrage, puis exécuter selon plan révisé (16 jours).

---

**Document créé le** : Janvier 2025  
**Analyse basée sur** : `docs/Dorevia_Vault_Sprint4.md`  
**Version code analysée** : v1.1-dev (Sprint 3 Phase 3 complétée)

© 2025 Doreviateam — Analyse Expert Technique

