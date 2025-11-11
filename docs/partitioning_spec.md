# 📊 Spécification — Partitionnement Ledger

**Version** : v1.3.0  
**Date** : Janvier 2025  
**Sprint** : Sprint 5 Phase 5.4  
**Statut** : ✅ Implémenté

---

## 🎯 Vue d'Ensemble

Le partitionnement mensuel du ledger améliore les performances pour les volumes élevés (> 100k entrées/an) en divisant la table en partitions mensuelles.

### Fonctionnalités

- ✅ **Partitions mensuelles** : Automatiques (format `ledger_YYYY_MM`)
- ✅ **Migration transparente** : Données existantes migrées automatiquement
- ✅ **Requêtes optimisées** : Partition pruning par PostgreSQL
- ✅ **Maintenance automatique** : Création partitions courante/suivante

---

## 📋 Architecture

### Structure Partitionnée

```
ledger (table partitionnée)
├── ledger_2024_12 (partition)
├── ledger_2025_01 (partition)
├── ledger_2025_02 (partition)
└── ...
```

### Partitionnement par RANGE

```sql
CREATE TABLE ledger (
    id SERIAL,
    document_id UUID NOT NULL,
    hash TEXT NOT NULL,
    previous_hash TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    evidence_jws TEXT,
    PRIMARY KEY (id, timestamp),
    UNIQUE (document_id, hash)
) PARTITION BY RANGE (timestamp);
```

### Partition Mensuelle

```sql
CREATE TABLE ledger_2025_01 PARTITION OF ledger
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

---

## 🔧 Utilisation

### Initialisation

```go
ctx := context.Background()
err := ledger.SetupPartitionedLedger(ctx, pool, log)
```

**Comportement** :
1. Vérifie si le ledger est déjà partitionné
2. Convertit la table en table partitionnée (si nécessaire)
3. Migre les données existantes
4. Créé les partitions pour le mois actuel et suivant

### Création Automatique

Le `PartitionManager` crée automatiquement les partitions nécessaires :

```go
manager := ledger.NewPartitionManager(pool, log)

// Créer partition pour un mois spécifique
err := manager.EnsurePartition(ctx, 2025, 1)

// Créer partition pour le mois actuel
err := manager.EnsureCurrentPartition(ctx)

// Créer partition pour le mois suivant
err := manager.EnsureNextPartition(ctx)
```

### Migration Données Existantes

```go
err := manager.MigrateExistingData(ctx)
```

**Comportement** :
1. Récupère toutes les dates distinctes
2. Crée les partitions nécessaires
3. Les données sont automatiquement dans les bonnes partitions

---

## 📊 Requêtes Optimisées

### Partition Pruning

PostgreSQL sélectionne automatiquement la bonne partition :

```sql
-- Requête sur janvier 2025 → utilise uniquement ledger_2025_01
SELECT * FROM ledger 
WHERE timestamp >= '2025-01-01' 
  AND timestamp < '2025-02-01';
```

### AppendLedgerPartitioned

Version optimisée de `AppendLedger` pour tables partitionnées :

```go
hash, err := ledger.AppendLedgerPartitioned(ctx, tx, docID, shaHex, jws)
```

**Optimisation** : Cherche d'abord dans le mois actuel avant de chercher dans toutes les partitions.

---

## 🔍 Monitoring

### Informations Partitions

```go
partitions, err := manager.GetPartitionInfo(ctx)
```

**Retourne** :
```go
type PartitionInfo struct {
    Name        string  // ledger_2025_01
    Size        string  // 1.2 MB
    IsPartition bool    // true
}
```

### Statistiques Table

```go
stats, err := ledger.GetTableStats(ctx, pool)
```

**Retourne** :
```go
type TableStats struct {
    TotalRows  int64   // Nombre total de lignes
    TableSize  string  // Taille totale (ex: "150 MB")
    IndexSize  string  // Taille des index (ex: "50 MB")
    IndexCount int     // Nombre d'index
}
```

---

## 🛠️ Maintenance

### Analyse Table

```go
err := ledger.AnalyzeTable(ctx, pool, log)
```

Met à jour les statistiques pour l'optimiseur de requêtes.

### Vacuum

```go
err := ledger.VacuumTable(ctx, pool, log)
```

Récupère l'espace et met à jour les statistiques.

### Création Index Optimisés

```go
err := ledger.OptimizeDatabase(ctx, pool, log)
```

Crée les index suivants :
- `ledger_timestamp_idx` : Tri par timestamp
- `ledger_document_id_idx` : Recherche par document
- `ledger_hash_idx` : Recherche par hash
- `ledger_previous_hash_idx` : Chaînage (partial index)
- `ledger_timestamp_month_idx` : Partitionnement

---

## 📋 Configuration

### Activation

Le partitionnement est **automatique** si :
- PostgreSQL 14+ est utilisé
- La fonction `SetupPartitionedLedger` est appelée

### Condition d'Activation

D'après le plan, le partitionnement est recommandé si :
- **Volume > 100k entrées/an**

---

## 🧪 Tests

### Tests Unitaires

- ✅ `TestPartitionManager_GetPartitionName` : Format noms
- ✅ `TestPartitionDateRange` : Logique dates
- ✅ `TestPartitionInfo` : Structure informations

**Total** : 10 tests (skip si PostgreSQL non disponible)

---

## 📊 Performance

### Avantages

1. **Requêtes plus rapides** : Partition pruning réduit les données scannées
2. **Maintenance facilitée** : VACUUM/ANALYZE par partition
3. **Scalabilité** : Support de millions d'entrées
4. **Archivage facile** : Détacher partitions anciennes

### Benchmarks Attendus

| Volume | Sans Partition | Avec Partition | Amélioration |
|:-------|:--------------|:--------------|:------------|
| 10k entrées | 50ms | 45ms | 10% |
| 100k entrées | 500ms | 200ms | 60% |
| 1M entrées | 5s | 800ms | 84% |

---

## 🔧 Migration

### Migration Automatique

La fonction `SetupPartitionedLedger` gère automatiquement :
1. Conversion table → table partitionnée
2. Migration données existantes
3. Création partitions nécessaires

### Migration Manuelle

Si nécessaire, migration manuelle possible :

```sql
-- 1. Créer table partitionnée
CREATE TABLE ledger_new (...) PARTITION BY RANGE (timestamp);

-- 2. Créer partitions
CREATE TABLE ledger_2025_01 PARTITION OF ledger_new ...;

-- 3. Copier données
INSERT INTO ledger_new SELECT * FROM ledger;

-- 4. Remplacer table
DROP TABLE ledger;
ALTER TABLE ledger_new RENAME TO ledger;
```

---

## 📚 Références

- [PostgreSQL Partitioning](https://www.postgresql.org/docs/current/ddl-partitioning.html)
- [Partition Pruning](https://www.postgresql.org/docs/current/ddl-partitioning.html#DDL-PARTITION-PRUNING)
- [Table Partitioning Best Practices](https://www.postgresql.org/docs/current/ddl-partitioning.html#DDL-PARTITIONING-BEST-PRACTICES)

