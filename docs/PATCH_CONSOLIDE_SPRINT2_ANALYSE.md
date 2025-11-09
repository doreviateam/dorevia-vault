# 🔍 Analyse du Patch Consolidé Sprint 2
## Plan d'Implémentation des 12 Correctifs

**Date** : Janvier 2025  
**Basé sur** : Patch Consolidé Sprint 2 Hardening  
**Statut** : Analyse et planification

---

## 📊 État Actuel vs. Correctifs Requis

### ✅ Déjà Implémenté

| Correctif | Fichier | État | Notes |
|:----------|:--------|:-----|:------|
| 1️⃣ Statuts HTTP | `invoices.go` | ✅ | Déjà correct (201/200) |
| 3️⃣ Types Go | `document.go` | ⚠️ | Partiel (pointeurs utilisés) |

### ❌ À Implémenter

| Correctif | Fichier | Priorité | Complexité |
|:----------|:--------|:---------|:------------|
| 2️⃣ Handler export | `ledger_export.go` | 🔴 Haute | Moyenne |
| 4️⃣ Premier hash ledger | `ledger/append.go` | 🔴 Haute | Faible |
| 5️⃣ Verrou concurrentiel | `ledger/append.go` | 🔴 Haute | Moyenne |
| 6️⃣ Déplacement fichier | `postgres.go` | 🟡 Moyenne | Faible |
| 7️⃣ Décimales SQL | `003_add_odoo_fields.sql` | 🟡 Moyenne | Faible |
| 8️⃣ Table ledger | `004_add_ledger.sql` | 🔴 Haute | Moyenne |
| 9️⃣ Export paginé | `ledger/export.go` | 🟡 Moyenne | Moyenne |
| 🔟 Mode dégradé JWS | `config.go` + `invoices.go` | 🟡 Moyenne | Moyenne |
| 11️⃣ Harmonisation noms | Documentation | 🟢 Basse | Faible |
| 12️⃣ Idempotence ledger | `invoices.go` + `ledger/append.go` | 🟡 Moyenne | Moyenne |

---

## 🎯 Plan d'Implémentation par Priorité

### Phase 1 : Fondations (Priorité 🔴)

#### 1. Migration 004_add_ledger.sql

**Fichier** : `migrations/004_add_ledger.sql`

**Contenu** :
```sql
-- Table ledger
CREATE TABLE IF NOT EXISTS ledger (
  id SERIAL PRIMARY KEY,
  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  hash TEXT NOT NULL,
  previous_hash TEXT,
  timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
  evidence_jws TEXT
);

-- Index
CREATE INDEX IF NOT EXISTS idx_ledger_document_id ON ledger(document_id);
CREATE INDEX IF NOT EXISTS idx_ledger_timestamp ON ledger(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_hash ON ledger(hash);
CREATE INDEX IF NOT EXISTS idx_ledger_prev_hash ON ledger(previous_hash);

-- Index composite pour SELECT previous_hash optimisé
CREATE INDEX IF NOT EXISTS idx_ledger_ts_id_desc ON ledger(timestamp DESC, id DESC);

-- Contrainte d'unicité (document_id, hash)
ALTER TABLE ledger ADD CONSTRAINT IF NOT EXISTS uq_ledger_doc_hash 
  UNIQUE (document_id, hash);
```

#### 2. Module ledger/append.go

**Fichier** : `internal/ledger/append.go`

**Fonctionnalités** :
- `AppendLedger()` avec verrou `FOR UPDATE`
- Gestion du premier hash (NULL)
- Hash chaîné : `SHA256(previous_hash + sha256_document)`

**Code clé** :
```go
func AppendLedger(ctx context.Context, tx pgx.Tx, docID uuid.UUID, shaHex, jws string) (string, error) {
    // SELECT avec verrou exclusif
    var previousHash *string
    err := tx.QueryRow(ctx, `
        SELECT hash FROM ledger 
        ORDER BY timestamp DESC, id DESC 
        LIMIT 1 
        FOR UPDATE
    `).Scan(&previousHash)
    
    var newHash string
    if err == pgx.ErrNoRows || previousHash == nil {
        // Premier enregistrement
        newHash = hex.EncodeToString(sha256.Sum256([]byte(shaHex)))
    } else if err != nil {
        return "", fmt.Errorf("failed to get previous hash: %w", err)
    } else {
        // Chaînage
        combined := *previousHash + shaHex
        newHash = hex.EncodeToString(sha256.Sum256([]byte(combined)))
    }
    
    // INSERT dans ledger
    _, err = tx.Exec(ctx, `
        INSERT INTO ledger (document_id, hash, previous_hash, evidence_jws)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (document_id, hash) DO NOTHING
    `, docID, newHash, previousHash, jws)
    
    return newHash, err
}
```

#### 3. Module ledger/export.go

**Fichier** : `internal/ledger/export.go`

**Fonctionnalités** :
- Export JSON paginé
- Export CSV paginé
- Protection limit <= 10000

### Phase 2 : Améliorations (Priorité 🟡)

#### 4. Configuration JWS

**Fichier** : `internal/config/config.go`

**Ajouts** :
```go
type Config struct {
    // ... champs existants ...
    JWSEnabled  bool `env:"JWS_ENABLED" envDefault:"true"`
    JWSRequired bool `env:"JWS_REQUIRED" envDefault:"true"`
}
```

#### 5. Mode dégradé dans invoices.go

**Modifications** :
- Vérifier `JWSEnabled` avant génération JWS
- Si `JWSRequired=false` et échec JWS → continuer sans JWS
- Si `JWSRequired=true` et échec JWS → retourner 503

#### 6. Correction migration 003

**Fichier** : `migrations/003_add_odoo_fields.sql`

**Modifications** :
- `DECIMAL(10,2)` → `DECIMAL(12,2)`
- Ajouter contrainte `chk_source`

#### 7. Déplacement fichier après COMMIT

**Fichier** : `internal/storage/postgres.go`

**Modifications** :
- Stocker fichier dans tmp d'abord
- Move après COMMIT réussi

### Phase 3 : Optimisations (Priorité 🟢)

#### 8. Handler ledger_export.go

**Fichier** : `internal/handlers/ledger_export.go`

**Fonctionnalités** :
- Route `/api/v1/ledger/export`
- Support `format=json|csv`
- Pagination `limit`/`offset`

#### 9. Idempotence ledger renforcée

**Fichier** : `internal/handlers/invoices.go`

**Modifications** :
- Vérifier si ledger existe pour document existant
- Compléter ledger si manquant

---

## 📋 Checklist d'Implémentation

### Phase 1 - Fondations

- [ ] Créer `migrations/004_add_ledger.sql`
- [ ] Créer `internal/ledger/append.go`
- [ ] Intégrer migration dans `postgres.go`
- [ ] Tests unitaires `AppendLedger()`

### Phase 2 - Améliorations

- [ ] Ajouter `JWSEnabled`/`JWSRequired` dans `config.go`
- [ ] Implémenter mode dégradé dans `invoices.go`
- [ ] Corriger `migrations/003_add_odoo_fields.sql` (décimales + contrainte source)
- [ ] Modifier `postgres.go` pour déplacement fichier après COMMIT

### Phase 3 - Optimisations

- [ ] Créer `internal/ledger/export.go`
- [ ] Créer `internal/handlers/ledger_export.go`
- [ ] Ajouter route dans `main.go`
- [ ] Implémenter idempotence ledger dans `invoices.go`

---

## 🔧 Détails Techniques

### Correctif 5️⃣ : Verrou Concurrentiel

**Problème** : Race condition sur `previous_hash`

**Solution** : `SELECT ... FOR UPDATE`

**Impact** : Performance légèrement réduite mais cohérence garantie

### Correctif 6️⃣ : Déplacement Fichier

**Problème** : Fichier créé avant COMMIT → risque d'incohérence

**Solution** :
```go
// 1. Créer fichier tmp
tmpPath := storedPath + ".tmp"
os.WriteFile(tmpPath, content, 0644)

// 2. Transaction
tx.Begin()
// ... INSERT documents ...
tx.Commit()

// 3. Move après COMMIT
os.Rename(tmpPath, storedPath)
```

### Correctif 7️⃣ : Décimales

**Problème** : `DECIMAL(10,2)` peut être insuffisant pour gros montants

**Solution** : `DECIMAL(12,2)` (max 9,999,999,999.99)

### Correctif 8️⃣ : Contrainte Source

**Problème** : Pas de validation des valeurs `source`

**Solution** :
```sql
ALTER TABLE documents ADD CONSTRAINT chk_source
  CHECK (source IN ('sales','purchase','pos','stock','sale') OR source IS NULL);
```

---

## 🧪 Tests Requis

### Tests Unitaires

- [ ] `TestAppendLedger_FirstHash` (premier enregistrement)
- [ ] `TestAppendLedger_Chaining` (chaînage)
- [ ] `TestAppendLedger_Concurrent` (verrou FOR UPDATE)
- [ ] `TestExportLedgerJSON_Pagination`
- [ ] `TestExportLedgerCSV_Pagination`

### Tests d'Intégration

- [ ] Pipeline complet avec JWS + Ledger
- [ ] Mode dégradé JWS (JWSRequired=false)
- [ ] Idempotence avec ledger

---

## 📝 Notes d'Implémentation

### Ordre Recommandé

1. **Migration 004** (base pour tout)
2. **ledger/append.go** (core fonctionnel)
3. **Config JWS** (prérequis pour invoices)
4. **Mode dégradé** (amélioration invoices)
5. **Export ledger** (fonctionnalité complémentaire)
6. **Corrections migrations** (nettoyage)

### Dépendances

- `ledger/append.go` nécessite migration 004
- `invoices.go` mode dégradé nécessite config JWS
- `ledger_export.go` nécessite `ledger/export.go`

---

## 🎯 Critères de Succès

### Fonctionnels

- ✅ Ledger fonctionne avec verrou exclusif
- ✅ Premier hash géré correctement
- ✅ Mode dégradé JWS opérationnel
- ✅ Export ledger paginé fonctionnel
- ✅ Idempotence ledger respectée

### Techniques

- ✅ Pas de race conditions sur ledger
- ✅ Fichiers déplacés après COMMIT
- ✅ Décimales cohérentes (12,2)
- ✅ Contraintes SQL appliquées

---

**Document créé le** : Janvier 2025  
**Version** : 1.0  
**Prochaine étape** : Implémentation Phase 1

