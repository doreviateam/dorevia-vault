# 🚨 Spécification Alerting & Supervision — Dorevia Vault
**Sprint 4 Phase 4.3 — Alerting & Supervision**

**Date** : Janvier 2025  
**Version** : 1.0

---

## 📋 Vue d'ensemble

Ce document spécifie les règles d'alerte Prometheus, la configuration Alertmanager, et l'export vers Odoo pour Dorevia Vault.

**Objectif** : Détecter et notifier automatiquement les anomalies, erreurs critiques, et problèmes de performance du système.

---

## 🎯 Règles d'Alerte Prometheus

### 1. Taux d'erreur documents élevé

**Alerte** : `HighDocumentErrorRate`  
**Seuil** : > 10% d'erreurs sur 5 minutes  
**Sévérité** : `warning`  
**Durée** : 5 minutes

**Expression PromQL** :
```promql
(
  rate(documents_vaulted_total{status="error"}[5m]) /
  rate(documents_vaulted_total[5m])
) > 0.1
```

**Justification** :
- Un taux d'erreur > 10% indique un problème systémique (DB, stockage, réseau)
- Seuil basé sur l'expérience : < 1% normal, > 10% anormal
- Durée 5min pour éviter les faux positifs sur pics ponctuels

**Actions recommandées** :
- Vérifier les logs d'erreur
- Vérifier la santé de la base de données
- Vérifier l'espace disque disponible

---

### 2. Ledger append lent

**Alerte** : `SlowLedgerAppend`  
**Seuil** : P95 > 2 secondes  
**Sévérité** : `warning`  
**Durée** : 10 minutes

**Expression PromQL** :
```promql
histogram_quantile(0.95, rate(ledger_append_duration_seconds_bucket[5m])) > 2
```

**Justification** :
- Le ledger doit être rapide (< 500ms normalement)
- P95 > 2s indique un problème de performance DB ou de contention
- Durée 10min pour éviter les alertes sur pics temporaires

**Actions recommandées** :
- Vérifier la charge de la base de données
- Vérifier les index sur la table `ledger`
- Vérifier les verrous transactionnels (`SELECT ... FOR UPDATE`)

---

### 3. Erreurs ledger fréquentes

**Alerte** : `FrequentLedgerErrors`  
**Seuil** : > 0.05 erreurs/seconde  
**Sévérité** : `warning`  
**Durée** : 5 minutes

**Expression PromQL** :
```promql
rate(ledger_append_errors_total[5m]) > 0.05
```

**Justification** :
- Les erreurs ledger sont rares (< 0.01/s normalement)
- > 0.05/s indique un problème d'intégrité DB ou de contraintes
- Durée 5min pour détecter rapidement les problèmes

**Actions recommandées** :
- Vérifier les logs d'erreur ledger
- Vérifier l'intégrité de la base de données
- Vérifier les contraintes de la table `ledger`

---

### 4. Stockage presque plein

**Alerte** : `StorageNearlyFull`  
**Seuil** : > 80% de capacité  
**Sévérité** : `critical`  
**Durée** : 1 heure

**Expression PromQL** :
```promql
(storage_size_bytes / system_disk_capacity_bytes) > 0.8
```

**Justification** :
- Le stockage doit rester < 80% pour éviter les problèmes
- > 80% nécessite une action préventive
- Durée 1h pour éviter les alertes sur pics temporaires

**Actions recommandées** :
- Nettoyer les anciens fichiers
- Augmenter la capacité disque
- Configurer la rétention automatique

---

### 5. Stockage critique

**Alerte** : `StorageCritical`  
**Seuil** : > 90% de capacité  
**Sévérité** : `critical`  
**Durée** : 30 minutes

**Expression PromQL** :
```promql
(storage_size_bytes / system_disk_capacity_bytes) > 0.9
```

**Justification** :
- > 90% nécessite une action immédiate
- Risque de panne du système si le disque est plein
- Durée 30min pour alerter rapidement

**Actions recommandées** :
- **Action immédiate** : Nettoyer les fichiers temporaires
- Augmenter la capacité disque
- Arrêter temporairement l'ingestion si nécessaire

---

### 6. Mémoire système élevée

**Alerte** : `HighSystemMemoryUsage`  
**Seuil** : > 90% de mémoire utilisée  
**Sévérité** : `warning`  
**Durée** : 10 minutes

**Expression PromQL** :
```promql
(system_memory_usage_bytes / system_memory_total_bytes) > 0.9
```

**Justification** :
- > 90% peut causer des ralentissements ou des OOM
- Durée 10min pour éviter les alertes sur pics temporaires

**Actions recommandées** :
- Vérifier les fuites mémoire
- Vérifier les processus consommateurs
- Redémarrer le service si nécessaire

---

### 7. CPU système élevé

**Alerte** : `HighSystemCPUUsage`  
**Seuil** : > 80% d'utilisation CPU  
**Sévérité** : `warning`  
**Durée** : 15 minutes

**Expression PromQL** :
```promql
system_cpu_usage_percent > 80
```

**Justification** :
- > 80% peut causer des ralentissements
- Durée 15min pour éviter les alertes sur pics temporaires

**Actions recommandées** :
- Vérifier la charge du système
- Vérifier les processus consommateurs
- Optimiser les requêtes DB si nécessaire

---

### 8. Aucun document vaulté récemment

**Alerte** : `NoRecentDocuments`  
**Seuil** : 0 document/heure pendant 2 heures  
**Sévérité** : `info`  
**Durée** : 2 heures

**Expression PromQL** :
```promql
rate(documents_vaulted_total{status="success"}[1h]) == 0
```

**Justification** :
- Aucun document pendant 2h peut indiquer un problème de connectivité Odoo
- Sévérité `info` car peut être normal (weekend, maintenance)

**Actions recommandées** :
- Vérifier la connectivité avec Odoo
- Vérifier les endpoints d'ingestion
- Vérifier les logs Odoo

---

### 9. Réconciliations fréquentes

**Alerte** : `HighReconciliationRate`  
**Seuil** : > 0.1 réconciliations/heure  
**Sévérité** : `warning`  
**Durée** : 30 minutes

**Expression PromQL** :
```promql
rate(reconciliation_runs_total{status="success"}[1h]) > 0.1
```

**Justification** :
- Les réconciliations doivent être rares (< 0.01/h normalement)
- > 0.1/h indique des problèmes d'intégrité fichiers ↔ DB

**Actions recommandées** :
- Vérifier l'intégrité des fichiers
- Vérifier la cohérence de la base de données
- Investiguer les causes des orphelins

---

### 10. Service down

**Alerte** : `ServiceDown`  
**Seuil** : `up{job="dorevia_vault"} == 0`  
**Sévérité** : `critical`  
**Durée** : 1 minute

**Expression PromQL** :
```promql
up{job="dorevia_vault"} == 0
```

**Justification** :
- Le service doit être accessible en permanence
- Durée 1min pour alerter rapidement

**Actions recommandées** :
- Vérifier l'état du service systemd
- Vérifier les logs système
- Redémarrer le service si nécessaire

---

## 🔧 Configuration Alertmanager

### Structure des routes

Les alertes sont routées selon leur sévérité :
- **`critical`** → Récepteur `critical` (Slack + Webhook)
- **`warning`** → Récepteur `warning` (Webhook uniquement)
- **`info`** → Récepteur `info` (Webhook uniquement)

### Groupement

Les alertes sont groupées par :
- `alertname`
- `severity`
- `component`

### Timing

- **`group_wait`** : 10s (attendre 10s avant d'envoyer le premier groupe)
- **`group_interval`** : 10s (intervalle entre groupes)
- **`repeat_interval`** : 12h (répéter l'alerte toutes les 12h si non résolue)

### Inhibition

Si le service est down (`ServiceDown`), les autres alertes sont inhibées pour éviter le spam.

---

## 📤 Export vers Odoo

### Format

Les alertes sont exportées vers Odoo via `ir.logging` :

```go
type OdooLogEntry struct {
    Name    string // "dorevia.vault"
    Type    string // "server"
    Level   string // "error" | "warning" | "info"
    Message string // Message formaté
    Func    string // Nom de l'alerte
    Path    string // "dorevia-vault"
}
```

### Mapping Sévérité

- **`critical`** → `level: "error"`
- **`warning`** → `level: "warning"`
- **`info`** → `level: "info"`

### Endpoint Odoo

**URL** : `{ODOO_URL}/jsonrpc`  
**Méthode** : `POST`  
**Service** : `object.execute_kw`  
**Modèle** : `ir.logging`  
**Méthode** : `create`

### Configuration

Variables d'environnement :
- `ODOO_URL` : URL Odoo (ex: `https://odoo.doreviateam.com`)
- `ODOO_DATABASE` : Base de données Odoo
- `ODOO_USER` : Utilisateur Odoo
- `ODOO_PASSWORD` : Mot de passe Odoo

---

## 🔌 Webhook Handler

### Endpoint

**URL** : `POST /api/v1/alerts/webhook`  
**Content-Type** : `application/json`

### Payload Alertmanager

Le handler reçoit le payload standard Alertmanager :

```json
{
  "version": "4",
  "groupKey": "...",
  "status": "firing",
  "receiver": "default",
  "groupLabels": {...},
  "commonLabels": {...},
  "commonAnnotations": {...},
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "HighDocumentErrorRate",
        "severity": "warning"
      },
      "annotations": {
        "summary": "...",
        "description": "..."
      }
    }
  ]
}
```

### Traitement

1. **Parsing** : Parse le payload JSON
2. **Logging** : Log chaque alerte reçue
3. **Export Odoo** : Exporte vers Odoo si configuré et si `status == "firing"`
4. **Réponse** : Retourne `{"status": "ok", "message": "Processed N alerts"}`

---

## 📊 Fichiers de Configuration

### Prometheus

- **`prometheus/alert_rules.yml`** : 10 règles d'alerte détaillées
- **`prometheus/prometheus.yml`** : Configuration scrape + règles

### Alertmanager

- **`alertmanager/alertmanager.yml`** : Configuration routes + récepteurs

---

## 🧪 Tests

### Tests manuels

1. **Tester une alerte** :
   ```bash
   # Simuler une alerte via curl
   curl -X POST http://localhost:8080/api/v1/alerts/webhook \
     -H "Content-Type: application/json" \
     -d @test_alert.json
   ```

2. **Vérifier l'export Odoo** :
   - Vérifier les logs Odoo (`ir.logging`)
   - Vérifier que l'alerte apparaît dans Odoo

### Tests automatisés

À venir (Phase 4.4) :
- Tests unitaires pour `OdooExporter`
- Tests d'intégration pour le webhook handler

---

## 📝 Notes d'Implémentation

### Limitations actuelles

1. **Odoo JSON-RPC** : Utilise JSON-RPC simplifié (peut nécessiter ajustements selon version Odoo)
2. **Authentification Odoo** : Utilise user/password (peut être amélioré avec API keys)
3. **Retry** : Pas de retry automatique en cas d'échec Odoo (à ajouter si nécessaire)

### Améliorations futures

1. **Retry avec backoff** : Retry automatique en cas d'échec Odoo
2. **Queue** : Queue Redis pour les alertes (éviter les pertes)
3. **Templates** : Templates personnalisés pour les messages Odoo
4. **Métriques** : Métriques Prometheus pour les exports Odoo (succès/échecs)

---

## ✅ Checklist de Déploiement

### 1. Configuration Prometheus

- [ ] Copier `prometheus/alert_rules.yml` vers `/etc/prometheus/alert_rules.yml`
- [ ] Copier `prometheus/prometheus.yml` vers `/etc/prometheus/prometheus.yml`
- [ ] Vérifier que Prometheus scrape `/metrics` de Dorevia Vault
- [ ] Vérifier que les règles sont chargées : `curl http://localhost:9090/api/v1/rules`

### 2. Configuration Alertmanager

- [ ] Copier `alertmanager/alertmanager.yml` vers `/etc/alertmanager/alertmanager.yml`
- [ ] Configurer Slack webhook (optionnel) : décommenter `slack_configs`
- [ ] Vérifier que Alertmanager pointe vers le webhook : `http://vault.doreviateam.com/api/v1/alerts/webhook`
- [ ] Tester : `curl http://localhost:9093/api/v1/alerts`

### 3. Configuration Dorevia Vault

- [ ] Configurer variables d'environnement Odoo :
  ```bash
  export ODOO_URL="https://odoo.doreviateam.com"
  export ODOO_DATABASE="dorevia"
  export ODOO_USER="vault_user"
  export ODOO_PASSWORD="..."
  ```
- [ ] Redémarrer le service : `sudo systemctl restart dorevia-vault`
- [ ] Vérifier que le webhook est actif : `curl http://localhost:8080/api/v1/alerts/webhook`

### 4. Tests

- [ ] Tester une alerte manuelle (simuler via Prometheus)
- [ ] Vérifier que l'alerte arrive dans Alertmanager
- [ ] Vérifier que l'alerte est exportée vers Odoo
- [ ] Vérifier les logs Odoo (`ir.logging`)

---

**Document créé le** : Janvier 2025  
**Version** : 1.0  
**Auteur** : Doreviateam

