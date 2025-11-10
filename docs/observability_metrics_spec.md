# 📊 Spécification des Métriques — Dorevia Vault

**Version** : v1.2-dev  
**Date** : Janvier 2025  
**Sprint** : Sprint 4 Phase 4.1  
**Endpoint** : `GET /metrics`

---

## 🎯 Vue d'Ensemble

Dorevia Vault expose des métriques Prometheus via l'endpoint `/metrics`.  
Toutes les métriques suivent les conventions Prometheus :
- Préfixe : `documents_`, `jws_`, `ledger_`, `system_`, `transaction_`
- Suffixe : `_total` (counters), `_seconds` (histograms), `_bytes` (gauges), `_percent` (gauges)
- Labels : Utilisés pour dimensionner les métriques

---

## 📈 Métriques par Catégorie

### 1. Counters (Compteurs d'événements)

Les counters ne peuvent qu'augmenter (ou être réinitialisés à 0).

#### `documents_vaulted_total`

**Type** : Counter (avec labels)  
**Labels** :
- `status` : `"success"` | `"error"` | `"idempotent"`
- `source` : `"sales"` | `"purchase"` | `"pos"` | `"stock"` | `"sale"` | `"unknown"`

**Description** : Nombre total de documents vaultés par statut et source.

**Exemple** :
```
documents_vaulted_total{source="sales",status="success"} 1234
documents_vaulted_total{source="sales",status="error"} 5
documents_vaulted_total{source="purchase",status="idempotent"} 42
```

**Utilisation** :
- Taux d'erreur : `rate(documents_vaulted_total{status="error"}[5m]) / rate(documents_vaulted_total[5m])`
- Volume par source : `sum by (source) (documents_vaulted_total{status="success"})`

---

#### `jws_signatures_total`

**Type** : Counter (avec labels)  
**Labels** :
- `status` : `"success"` | `"error"` | `"degraded"`

**Description** : Nombre total de signatures JWS générées par statut.

**Exemple** :
```
jws_signatures_total{status="success"} 1200
jws_signatures_total{status="error"} 3
jws_signatures_total{status="degraded"} 1
```

**Utilisation** :
- Taux d'erreur JWS : `rate(jws_signatures_total{status="error"}[5m]) / rate(jws_signatures_total[5m])`
- Mode dégradé : `jws_signatures_total{status="degraded"}`

---

#### `ledger_entries_total`

**Type** : Counter  
**Description** : Nombre total d'entrées ajoutées au ledger.

**Exemple** :
```
ledger_entries_total 1200
```

**Utilisation** :
- Taux d'ajout : `rate(ledger_entries_total[5m])`
- Volume total : `ledger_entries_total`

---

#### `ledger_append_errors_total`

**Type** : Counter  
**Description** : Nombre total d'erreurs lors de l'ajout au ledger.

**Exemple** :
```
ledger_append_errors_total 2
```

**Utilisation** :
- Taux d'erreur : `rate(ledger_append_errors_total[5m])`
- Ratio erreurs : `ledger_append_errors_total / ledger_entries_total`

**Sprint** : Sprint 4 Phase 4.1 (nouveau)

---

#### `reconciliation_runs_total`

**Type** : Counter (avec labels)  
**Labels** :
- `status` : `"success"` | `"error"`

**Description** : Nombre total d'exécutions de réconciliation par statut.

**Exemple** :
```
reconciliation_runs_total{status="success"} 10
reconciliation_runs_total{status="error"} 1
```

**Utilisation** :
- Taux d'exécution : `rate(reconciliation_runs_total[1h])`
- Taux d'erreur : `rate(reconciliation_runs_total{status="error"}[1h])`

---

### 2. Histogrammes (Durées d'opérations)

Les histogrammes mesurent la distribution des durées d'opérations.

#### `document_storage_duration_seconds`

**Type** : Histogram (avec labels)  
**Labels** :
- `operation` : `"store"` | `"verify"`

**Buckets** : `0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10` secondes

**Description** : Durée de stockage des documents en secondes.

**Exemple** :
```
document_storage_duration_seconds_bucket{operation="store",le="0.1"} 1000
document_storage_duration_seconds_bucket{operation="store",le="0.5"} 1200
document_storage_duration_seconds_sum{operation="store"} 150.5
document_storage_duration_seconds_count{operation="store"} 1200
```

**Utilisation** :
- P95 : `histogram_quantile(0.95, document_storage_duration_seconds{operation="store"})`
- Moyenne : `rate(document_storage_duration_seconds_sum[5m]) / rate(document_storage_duration_seconds_count[5m])`

---

#### `jws_signature_duration_seconds`

**Type** : Histogram  
**Buckets** : `0.0001, 0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1` seconde

**Description** : Durée de génération des signatures JWS en secondes.

**Utilisation** :
- P99 : `histogram_quantile(0.99, jws_signature_duration_seconds)`
- Moyenne : `rate(jws_signature_duration_seconds_sum[5m]) / rate(jws_signature_duration_seconds_count[5m])`

---

#### `ledger_append_duration_seconds`

**Type** : Histogram  
**Buckets** : `0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10` secondes

**Description** : Durée d'ajout d'entrées au ledger en secondes.

**Utilisation** :
- P95 : `histogram_quantile(0.95, ledger_append_duration_seconds)`
- Alerte lenteur : `histogram_quantile(0.95, ledger_append_duration_seconds) > 2`

---

#### `transaction_duration_seconds`

**Type** : Histogram  
**Buckets** : `0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 30` secondes

**Description** : Durée totale des transactions (end-to-end) en secondes.

**Utilisation** :
- P95 : `histogram_quantile(0.95, transaction_duration_seconds)`
- Objectif : P95 < 300ms (0.3s)

---

### 3. Gauges (Valeurs instantanées)

Les gauges peuvent augmenter ou diminuer.

#### `ledger_size`

**Type** : Gauge  
**Description** : Nombre actuel d'entrées dans le ledger.

**Exemple** :
```
ledger_size 1200
```

**Utilisation** :
- Taille actuelle : `ledger_size`
- Croissance : `increase(ledger_size[1h])`

---

#### `storage_size_bytes`

**Type** : Gauge  
**Description** : Taille totale du stockage de documents en octets.

**Exemple** :
```
storage_size_bytes 1073741824
```

**Utilisation** :
- Taille actuelle : `storage_size_bytes / 1024 / 1024 / 1024` (en GB)
- Alerte stockage : `storage_size_bytes / system_disk_capacity_bytes > 0.8`

---

#### `active_connections`

**Type** : Gauge  
**Description** : Nombre de connexions actives à la base de données.

**Exemple** :
```
active_connections 5
```

**Utilisation** :
- Connexions actives : `active_connections`
- Pool utilisation : `active_connections / max_connections`

---

### 4. Métriques Système (Sprint 4 Phase 4.1)

#### `system_cpu_usage_percent`

**Type** : Gauge  
**Description** : Utilisation CPU en pourcentage (0-100).

**Exemple** :
```
system_cpu_usage_percent 45.2
```

**Utilisation** :
- CPU actuel : `system_cpu_usage_percent`
- Alerte CPU : `system_cpu_usage_percent > 80`

**Mise à jour** : Toutes les 30 secondes (automatique)

---

#### `system_memory_usage_bytes`

**Type** : Gauge  
**Description** : Utilisation mémoire en octets.

**Exemple** :
```
system_memory_usage_bytes 8589934592
```

**Utilisation** :
- RAM utilisée : `system_memory_usage_bytes / 1024 / 1024 / 1024` (en GB)
- Ratio : `system_memory_usage_bytes / system_memory_total_bytes`

---

#### `system_memory_total_bytes`

**Type** : Gauge  
**Description** : Mémoire totale disponible en octets.

**Exemple** :
```
system_memory_total_bytes 17179869184
```

**Utilisation** :
- RAM totale : `system_memory_total_bytes / 1024 / 1024 / 1024` (en GB)
- Ratio utilisation : `system_memory_usage_bytes / system_memory_total_bytes`

---

#### `system_disk_usage_bytes`

**Type** : Gauge  
**Description** : Utilisation disque en octets.

**Exemple** :
```
system_disk_usage_bytes 429496729600
```

**Utilisation** :
- Disque utilisé : `system_disk_usage_bytes / 1024 / 1024 / 1024` (en GB)
- Ratio : `system_disk_usage_bytes / system_disk_capacity_bytes`

---

#### `system_disk_capacity_bytes`

**Type** : Gauge  
**Description** : Capacité disque totale en octets.

**Exemple** :
```
system_disk_capacity_bytes 515396075520
```

**Utilisation** :
- Capacité totale : `system_disk_capacity_bytes / 1024 / 1024 / 1024` (en GB)
- Alerte stockage : `system_disk_usage_bytes / system_disk_capacity_bytes > 0.8`

---

## 📊 Requêtes Prometheus Utiles

### Taux d'erreur documents

```promql
rate(documents_vaulted_total{status="error"}[5m]) / 
rate(documents_vaulted_total[5m]) * 100
```

### Latence P95 transaction

```promql
histogram_quantile(0.95, transaction_duration_seconds)
```

### Utilisation stockage (%)

```promql
(storage_size_bytes / system_disk_capacity_bytes) * 100
```

### Taux d'erreur ledger

```promql
rate(ledger_append_errors_total[5m]) / 
rate(ledger_entries_total[5m]) * 100
```

### Throughput documents (doc/s)

```promql
rate(documents_vaulted_total{status="success"}[1m])
```

---

## 🚨 Alertes Recommandées

### Taux d'erreur documents élevé

```yaml
- alert: HighDocumentErrorRate
  expr: rate(documents_vaulted_total{status="error"}[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
```

### Ledger append lent

```yaml
- alert: SlowLedgerAppend
  expr: histogram_quantile(0.95, ledger_append_duration_seconds) > 2
  for: 10m
  labels:
    severity: warning
```

### Stockage presque plein

```yaml
- alert: StorageNearlyFull
  expr: storage_size_bytes / system_disk_capacity_bytes > 0.8
  for: 1h
  labels:
    severity: critical
```

### Erreurs ledger fréquentes

```yaml
- alert: FrequentLedgerErrors
  expr: rate(ledger_append_errors_total[5m]) > 0.05
  for: 5m
  labels:
    severity: warning
```

---

## 🔧 Configuration

### Collecte automatique

Les métriques système sont mises à jour automatiquement toutes les **30 secondes** via un collecteur en arrière-plan.

### Endpoint

Toutes les métriques sont exposées via :
```
GET /metrics
```

Format : Prometheus text format (standard)

---

## 📋 Historique des Métriques

| Métrique | Sprint | Statut |
|:---------|:-------|:-------|
| `documents_vaulted_total` | Sprint 3 | ✅ Actif |
| `jws_signatures_total` | Sprint 3 | ✅ Actif |
| `ledger_entries_total` | Sprint 3 | ✅ Actif |
| `reconciliation_runs_total` | Sprint 3 | ✅ Actif |
| `document_storage_duration_seconds` | Sprint 3 | ✅ Actif |
| `jws_signature_duration_seconds` | Sprint 3 | ✅ Actif |
| `ledger_append_duration_seconds` | Sprint 3 | ✅ Actif |
| `transaction_duration_seconds` | Sprint 3 | ✅ Actif |
| `ledger_size` | Sprint 3 | ✅ Actif |
| `storage_size_bytes` | Sprint 3 | ✅ Actif |
| `active_connections` | Sprint 3 | ✅ Actif |
| `ledger_append_errors_total` | Sprint 4.1 | ✅ Actif |
| `system_cpu_usage_percent` | Sprint 4.1 | ✅ Actif |
| `system_memory_usage_bytes` | Sprint 4.1 | ✅ Actif |
| `system_memory_total_bytes` | Sprint 4.1 | ✅ Actif |
| `system_disk_usage_bytes` | Sprint 4.1 | ✅ Actif |
| `system_disk_capacity_bytes` | Sprint 4.1 | ✅ Actif |

---

**Document créé le** : Janvier 2025  
**Version** : 1.0  
**Auteur** : Doreviateam

© 2025 Doreviateam — Projet Dorevia Vault

