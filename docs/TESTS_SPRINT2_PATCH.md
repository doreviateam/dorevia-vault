# 🧪 Tests — Patch Consolidé Sprint 2

**Date** : Janvier 2025  
**Statut** : Tests unitaires créés, tests d'intégration prêts

---

## 📊 Résumé des Tests

### Tests Unitaires (sans DB)

| Test | Fichier | Statut | Description |
|:-----|:--------|:-------|:------------|
| `TestAppendLedger_Logic` | `ledger_append_test.go` | ✅ | Logique de calcul de hash |
| `TestExportLedgerJSON_Format` | `ledger_export_test.go` | ✅ | Format JSON |
| `TestExportLedgerCSV_Format` | `ledger_export_test.go` | ✅ | Format CSV |
| `TestExportLedger_LimitProtection` | `ledger_export_test.go` | ✅ | Protection limit <= 10000 |

### Tests d'Intégration (avec DB)

| Test | Fichier | Statut | Description |
|:-----|:--------|:-------|:------------|
| `TestLedger_AppendFirstHash` | `ledger_test.go` | ⏳ | Premier hash (previous=NULL) |
| `TestLedger_AppendChaining` | `ledger_test.go` | ⏳ | Chaînage des hash |
| `TestLedger_ExistsByDocumentID` | `ledger_test.go` | ⏳ | Vérification existence |

**Note** : Les tests d'intégration nécessitent `TEST_DATABASE_URL` et sont skip par défaut.

---

## 🚀 Exécution des Tests

### Tests Unitaires (sans DB)

```bash
# Tous les tests unitaires
go test ./tests/unit/... -v

# Tests spécifiques
go test ./tests/unit/... -run TestExport -v
go test ./tests/unit/... -run TestAppend -v
```

### Tests d'Intégration (avec DB)

```bash
# Configurer la base de test
export TEST_DATABASE_URL="postgres://user:pass@localhost/dorevia_vault_test"

# Exécuter les tests d'intégration
go test ./tests/integration/... -v
```

### Tests Rapides (short)

```bash
# Tests sans intégration
go test ./... -short
```

---

## ✅ Résultats Attendus

### Tests Unitaires

- ✅ `TestAppendLedger_Logic` : Valide la logique de calcul
- ✅ `TestExportLedgerJSON_Format` : Valide le format JSON
- ✅ `TestExportLedgerCSV_Format` : Valide le format CSV
- ✅ `TestExportLedger_LimitProtection` : Valide limit <= 10000

### Tests d'Intégration

- ⏳ `TestLedger_AppendFirstHash` : Premier hash = SHA256(shaHex)
- ⏳ `TestLedger_AppendChaining` : Hash2 = SHA256(hash1 + shaHex2)
- ⏳ `TestLedger_ExistsByDocumentID` : Vérification existence correcte

---

## 🔧 Configuration pour Tests d'Intégration

### Prérequis

1. PostgreSQL installé et démarré
2. Base de données de test créée :
   ```sql
   CREATE DATABASE dorevia_vault_test;
   ```

3. Variable d'environnement :
   ```bash
   export TEST_DATABASE_URL="postgres://user:pass@localhost/dorevia_vault_test"
   ```

### Exécution

```bash
# Lancer tous les tests d'intégration
go test ./tests/integration/... -v

# Lancer un test spécifique
go test ./tests/integration/... -run TestLedger_AppendFirstHash -v
```

---

## 📝 Tests Manuels

### Test Export Ledger JSON

```bash
# Démarrer le serveur
go run ./cmd/vault

# Dans un autre terminal
curl "http://localhost:8080/api/v1/ledger/export?format=json&limit=10&offset=0"
```

### Test Export Ledger CSV

```bash
curl "http://localhost:8080/api/v1/ledger/export?format=csv&limit=10&offset=0" -o ledger.csv
```

### Test Protection Limit

```bash
# Test avec limit > 10000 (doit être réduit à 10000)
curl "http://localhost:8080/api/v1/ledger/export?format=json&limit=20000"
```

---

## 🎯 Coverage Cible

- **Ledger/append.go** : > 80%
- **Ledger/export.go** : > 70%
- **Handlers/ledger_export.go** : > 70%

---

**Document créé le** : Janvier 2025  
**Version** : 1.0

