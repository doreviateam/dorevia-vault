# 💡 Avis d'Expert — Sprint 2
## Analyse Critique du Plan Sprint 2 (Vérifiable)

**Date** : Janvier 2025  
**Analyste** : Analyse technique approfondie  
**Document analysé** : `Dorevia_Vault_Sprint2.md`

---

## 📋 Table des matières

1. [Synthèse exécutive](#synthèse-exécutive)
2. [Points forts](#points-forts)
3. [Risques et défis techniques](#risques-et-défis-techniques)
4. [Points d'attention critiques](#points-dattention-critiques)
5. [Recommandations d'amélioration](#recommandations-damélioration)
6. [Alternatives et bonnes pratiques](#alternatives-et-bonnes-pratiques)
7. [Plan d'action recommandé](#plan-daction-recommandé)

---

## 🎯 Synthèse exécutive

### Verdict global

**✅ Plan solide** avec une vision claire du scellement JWS et du ledger hash-chaîné.  
**⚠️ Complexité technique élevée** nécessitant une attention particulière sur les transactions et la performance.  
**🔴 Risques identifiés** sur la gestion des transactions multi-étapes, la concurrence du ledger, et la sécurité des clés.

### Score de faisabilité

| Critère | Score | Commentaire |
|:--------|:------|:------------|
| **Clarté des objectifs** | 9/10 | JWS + Ledger bien définis |
| **Faisabilité technique** | 7/10 | Réalisable mais complexe |
| **Gestion des transactions** | 6/10 | Multi-étapes critiques |
| **Performance** | 7/10 | Ledger peut être un goulot |
| **Sécurité** | 8/10 | Bonnes pratiques identifiées |
| **Tests** | 7/10 | Plan de tests complet |

**Score global** : **7.3/10** — **Faisable avec précautions**

---

## ✅ Points forts

### 1. Vision claire : JWS + Ledger

**Excellent** : La séparation JWS (preuve de scellement) et Ledger (chaîne d'intégrité) est bien pensée.

- ✅ **JWS** : Preuve portable et vérifiable indépendamment
- ✅ **Ledger** : Chaîne d'intégrité pour l'ordre chronologique
- ✅ **Complémentarité** : Les deux systèmes se renforcent mutuellement

**Impact** : Architecture solide pour la vérifiabilité.

### 2. Sécurité des clés bien pensée

**Bon** : Les recommandations de sécurité sont pertinentes.

- ✅ Clés hors dépôt git
- ✅ Permissions 600
- ✅ Backup chiffré
- ✅ Rotation manuelle (30 jours)

### 3. Plan de tests complet

**Bon** : Tests unitaires, intégration et non-régression prévus.

- ✅ Tests JWS (signature, vérification, corruption)
- ✅ Tests Ledger (chaînage, vérification)
- ✅ Tests d'intégration (1000 docs)

### 4. Plan jour-par-jour réaliste

**Bon** : Découpage en 10-12 jours avec buffer.

- ✅ Tâches bien délimitées
- ✅ Ordre logique
- ✅ Buffer de stabilisation

---

## ⚠️ Risques et défis techniques

### 🔴 CRITIQUE : Transaction multi-étapes complexe

**Problème** : Le flux proposé (section 6.1) est **trop complexe** pour une seule transaction.

**Flux actuel proposé** :
```
1. Écriture fichier tmp
2. BEGIN transaction
3. INSERT documents
4. Génération JWS (hors DB)
5. AppendLedger
6. UPDATE documents (evidence_jws, ledger_hash)
7. COMMIT
8. Move fichier tmp → final
```

**Risques** :
- **Étape 4 (JWS)** : Si échec après INSERT → rollback nécessaire mais fichier tmp déjà créé
- **Étape 5 (Ledger)** : Si échec après JWS → rollback mais JWS déjà généré
- **Étape 8 (Move)** : Si échec après COMMIT → incohérence fichier/DB

**Solution recommandée** : **Simplifier le flux** :

```go
// Option 1 : Transaction englobante
tx, err := db.Begin(ctx)
defer tx.Rollback(ctx)

// 1. Stocker fichier (dans transaction, avec rollback)
// 2. INSERT documents
// 3. Générer JWS (hors transaction mais rapide)
// 4. AppendLedger (dans transaction)
// 5. UPDATE documents (evidence_jws, ledger_hash)
// 6. COMMIT

// Option 2 : Pattern Saga avec compensation
// Si JWS échoue → rollback + nettoyage
// Si Ledger échoue → rollback + nettoyage
```

### 🔴 CRITIQUE : Concurrence sur le ledger

**Problème** : La sélection du `previous_hash` peut créer des **race conditions**.

**Scénario problématique** :
```
Thread 1: SELECT previous_hash → hash_A
Thread 2: SELECT previous_hash → hash_A (même valeur)
Thread 1: INSERT avec hash_B = SHA256(hash_A + doc1)
Thread 2: INSERT avec hash_C = SHA256(hash_A + doc2)
→ Chaîne cassée ! hash_B et hash_C pointent vers hash_A
```

**Solution recommandée** :
- **Verrou exclusif** : `SELECT ... FOR UPDATE` sur le dernier enregistrement
- **Séquence PostgreSQL** : Garantir l'ordre d'insertion
- **Optimistic locking** : Version sur le dernier hash

**Exemple** :
```sql
-- Verrou exclusif sur le dernier hash
SELECT hash FROM ledger 
ORDER BY timestamp DESC, id DESC 
LIMIT 1 
FOR UPDATE;
```

### 🟡 IMPORTANT : Performance du ledger avec SELECT previous_hash

**Problème** : Le SELECT du `previous_hash` peut être lent avec beaucoup d'enregistrements.

**Risques** :
- Table `ledger` qui croît indéfiniment
- SELECT sans index optimal (même avec index sur timestamp)
- Contention sur le dernier enregistrement

**Solutions recommandées** :
- **Index composite** : `(timestamp DESC, id DESC)` pour le SELECT
- **Cache du previous_hash** : En mémoire (Redis) pour éviter SELECT à chaque fois
- **Séquence dédiée** : Pour garantir l'ordre sans SELECT

### 🟡 IMPORTANT : Gestion des erreurs JWS

**Problème** : Le document propose de **refuser l'ingestion** si JWS indisponible (503).

**Risques** :
- **Disponibilité** : Si clés corrompues → service complètement indisponible
- **Récupération** : Difficile de reprendre après correction des clés

**Solution recommandée** : **Mode dégradé** avec flag de configuration.

```go
// Configuration
JWS_REQUIRED=true  // Par défaut true, mais peut être false en mode dégradé

// Comportement
if JWS_REQUIRED && jwsError {
    return 503 // Service unavailable
} else {
    log.Warn().Msg("JWS unavailable, storing without evidence")
    // Continuer sans JWS mais logger l'alerte
}
```

### 🟡 IMPORTANT : Idempotence avec Ledger

**Problème** : L'idempotence doit éviter la **duplication dans le ledger**.

**Scénario problématique** :
```
1. Document A uploadé → JWS généré → Ledger inscrit
2. Document A re-uploadé (même hash) → Idempotence détectée
3. Mais si on ne vérifie pas le ledger → risque de duplication
```

**Solution recommandée** : Vérifier le ledger dans la logique d'idempotence.

```go
// Vérifier idempotence
if documentExists {
    // Vérifier aussi dans le ledger
    ledgerEntry, err := db.GetLedgerByDocumentID(docID)
    if err == nil && ledgerEntry != nil {
        return existingDoc // Document déjà complet
    }
    // Sinon, compléter le ledger (cas de migration)
}
```

### 🟢 MINEUR : Export Ledger

**Problème** : Export JSON/CSV peut être **lent** avec beaucoup d'enregistrements.

**Solution** : Streaming avec pagination.

```go
func ExportLedgerJSON(ctx context.Context, w io.Writer, limit, offset int) error
```

---

## 🔍 Points d'attention critiques

### 1. Ordre d'exécution dans la transaction

**Recommandation** : Réorganiser le flux pour simplifier.

**Flux recommandé** :
```go
// 1. Calculer hash (hors transaction)
hash := sha256.Sum256(content)

// 2. Vérifier idempotence (hors transaction)
if exists { return existing }

// 3. BEGIN transaction
tx, err := db.Begin(ctx)
defer tx.Rollback(ctx)

// 4. Stocker fichier (dans transaction)
// 5. INSERT documents
// 6. Générer JWS (hors transaction, rapide)
// 7. AppendLedger (dans transaction, avec verrou)
// 8. UPDATE documents (evidence_jws, ledger_hash)
// 9. COMMIT
```

### 2. Gestion du premier hash du ledger

**Problème** : Le premier enregistrement n'a pas de `previous_hash`.

**Solution** : Gérer explicitement le cas NULL.

```go
var previousHash *string
err := tx.QueryRow(ctx, 
    "SELECT hash FROM ledger ORDER BY timestamp DESC, id DESC LIMIT 1 FOR UPDATE"
).Scan(&previousHash)

var newHash string
if previousHash == nil {
    // Premier enregistrement
    newHash = hex.EncodeToString(sha256.Sum256([]byte(shaHex)))
} else {
    // Chaînage
    newHash = hex.EncodeToString(sha256.Sum256([]byte(*previousHash + shaHex)))
}
```

### 3. Sécurité des clés en production

**Recommandation** : Utiliser un **HSM** ou **Vault (HashiCorp)** pour les clés privées.

**Alternatives** :
- Variables d'environnement chiffrées (AWS Secrets Manager, etc.)
- Service de gestion de clés dédié
- Rotation automatique (Sprint 3+)

### 4. Performance avec gros volumes

**Recommandation** : Prévoir des **tests de charge** dès le Sprint 2.

**Métriques à surveiller** :
- Temps d'insertion ledger (cible < 50ms)
- Temps de génération JWS (cible < 10ms)
- Temps total d'ingestion (cible < 200ms)

### 5. Vérification périodique du ledger

**Recommandation** : Implémenter un **job de vérification** dès le Sprint 2.

```go
// Job quotidien (cron)
func VerifyLedgerDaily(ctx context.Context) error {
    if err := ledger.VerifyLedger(ctx, db); err != nil {
        // Alerter (email, Slack, etc.)
        return err
    }
    return nil
}
```

---

## 💡 Recommandations d'amélioration

### 1. Simplifier le flux transactionnel

**Problème actuel** : Trop d'étapes dans la transaction.

**Amélioration** : Regrouper les opérations critiques.

```go
// Fonction unique pour le flux complet
func (db *DB) StoreDocumentWithEvidence(ctx context.Context, 
    doc *models.Document, 
    content []byte, 
    storageDir string,
    cryptoService *crypto.Service,
    ledgerService *ledger.Service,
) error {
    // 1. Hash + idempotence (hors transaction)
    // 2. BEGIN transaction
    // 3. Stocker fichier + INSERT documents
    // 4. Générer JWS (hors transaction mais rapide)
    // 5. AppendLedger (dans transaction, avec verrou)
    // 6. UPDATE documents
    // 7. COMMIT
}
```

### 2. Ajouter un cache pour previous_hash

**Amélioration** : Éviter le SELECT à chaque insertion.

```go
type LedgerCache struct {
    mu          sync.RWMutex
    previousHash string
    lastUpdate  time.Time
    ttl         time.Duration
}

func (c *LedgerCache) GetPreviousHash(ctx context.Context, db *pgxpool.Pool) (string, error) {
    c.mu.RLock()
    if time.Since(c.lastUpdate) < c.ttl {
        hash := c.previousHash
        c.mu.RUnlock()
        return hash, nil
    }
    c.mu.RUnlock()

    // Refresh depuis DB
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Double-check
    if time.Since(c.lastUpdate) < c.ttl {
        return c.previousHash, nil
    }
    
    // SELECT avec verrou
    var hash string
    err := db.QueryRow(ctx, 
        "SELECT hash FROM ledger ORDER BY timestamp DESC, id DESC LIMIT 1 FOR UPDATE"
    ).Scan(&hash)
    
    if err == pgx.ErrNoRows {
        return "", nil // Premier enregistrement
    }
    
    c.previousHash = hash
    c.lastUpdate = time.Now()
    return hash, nil
}
```

### 3. Mode dégradé pour JWS

**Amélioration** : Permettre le fonctionnement sans JWS en cas d'urgence.

```go
// Configuration
type Config struct {
    JWSRequired bool `env:"JWS_REQUIRED" envDefault:"true"`
    JWSEnabled  bool `env:"JWS_ENABLED" envDefault:"true"`
}

// Comportement
if !cfg.JWSEnabled {
    log.Warn().Msg("JWS disabled, storing without evidence")
    // Continuer sans JWS
} else if err := generateJWS(); err != nil {
    if cfg.JWSRequired {
        return fmt.Errorf("JWS required but unavailable: %w", err)
    }
    log.Warn().Err(err).Msg("JWS generation failed, continuing without evidence")
}
```

### 4. Améliorer la gestion d'erreurs

**Amélioration** : Types d'erreurs spécifiques pour chaque étape.

```go
type ErrJWSGeneration struct {
    Cause error
}

func (e ErrJWSGeneration) Error() string {
    return fmt.Sprintf("JWS generation failed: %v", e.Cause)
}

type ErrLedgerAppend struct {
    Cause error
}

func (e ErrLedgerAppend) Error() string {
    return fmt.Sprintf("Ledger append failed: %v", e.Cause)
}
```

### 5. Tests de performance dès le Sprint 2

**Amélioration** : Ajouter des benchmarks.

```go
// tests/benchmark/ledger_test.go
func BenchmarkAppendLedger(b *testing.B) {
    // Setup
    for i := 0; i < b.N; i++ {
        // AppendLedger
    }
}

func BenchmarkVerifyLedger(b *testing.B) {
    // Setup avec 1000 entrées
    for i := 0; i < b.N; i++ {
        // VerifyLedger
    }
}
```

---

## 🔄 Alternatives et bonnes pratiques

### Alternative 1 : Ledger avec séquence PostgreSQL

**Option** : Utiliser une séquence pour garantir l'ordre.

```sql
CREATE SEQUENCE ledger_sequence;

CREATE TABLE ledger (
    id SERIAL PRIMARY KEY,
    sequence_id BIGINT DEFAULT nextval('ledger_sequence'),
    document_id UUID NOT NULL,
    hash TEXT NOT NULL,
    previous_hash TEXT,
    ...
);

CREATE INDEX idx_ledger_sequence ON ledger(sequence_id DESC);
```

**Avantages** :
- Ordre garanti sans SELECT
- Performance meilleure
- Pas de race condition

**Inconvénients** :
- Séquences PostgreSQL peuvent avoir des gaps
- Moins flexible pour la vérification

### Alternative 2 : JWS avec ECDSA au lieu de RSA

**Question** : Pourquoi RSA et pas ECDSA ?

**Réflexion** :
- **RSA** : Plus standard, supporté partout, mais plus lent
- **ECDSA** : Plus rapide, clés plus petites, mais moins standardisé

**Recommandation** : **RSA pour Sprint 2** (standard), envisager ECDSA en Sprint 3 si performance critique.

### Alternative 3 : Ledger externalisé

**Option** : Utiliser un service dédié (blockchain privée, service ledger).

**Avantages** :
- Séparation des responsabilités
- Performance dédiée
- Conformité renforcée

**Inconvénients** :
- Dépendance externe
- Coût additionnel
- Complexité d'intégration

**Recommandation** : **Ledger interne pour Sprint 2**, externaliser si volumétrie > 1M/an.

---

## 📊 Plan d'action recommandé (révisé)

### Jour 1-2 : Générateur de clés + JWS basique

**Tâches** :
1. ✅ Outil `cmd/keygen` (RSA 2048)
2. Module `internal/crypto/jws.go` (Sign/Verify)
3. Tests unitaires JWS

**Attention** : Gérer les erreurs de chargement de clés.

### Jour 3 : Endpoint `/jwks.json`

**Tâches** :
1. Handler `internal/handlers/jwks.go`
2. Cache en mémoire (5 minutes)
3. Tests

### Jour 4-5 : Migration + Ledger basique

**Tâches** :
1. Migration `004_add_ledger.sql`
2. Fonction `AppendLedger()` avec verrou
3. Gestion du premier hash (NULL)
4. Tests unitaires

**Attention** : **Verrou exclusif** sur le SELECT previous_hash.

### Jour 6 : Vérification + Export

**Tâches** :
1. Fonction `VerifyLedger()`
2. Export JSON/CSV (avec pagination)
3. Tests

### Jour 7-8 : Intégration transactionnelle

**Tâches** :
1. Modifier `StoreDocumentWithTransaction()` pour inclure JWS + Ledger
2. Gestion d'erreurs complète
3. Tests d'intégration

**Attention** : **Simplifier le flux** (voir recommandations).

### Jour 9-10 : Tests + Performance

**Tâches** :
1. Tests d'intégration (1000 docs)
2. Benchmarks performance
3. Tests de concurrence (race conditions)

### Jour 11-12 : Documentation + Polish

**Tâches** :
1. Documentation API
2. Exemples d'utilisation
3. Bugfix et optimisations

---

## 🎯 Recommandations finales

### Priorités absolues

1. **🔴 Verrou exclusif sur previous_hash** : Critique pour éviter les race conditions
2. **🔴 Simplifier le flux transactionnel** : Réduire la complexité
3. **🟡 Cache previous_hash** : Améliorer la performance
4. **🟡 Mode dégradé JWS** : Éviter l'indisponibilité totale
5. **🟡 Tests de performance** : Valider dès le Sprint 2

### Approche recommandée

**Principe** : **Simplicité d'abord, optimisation ensuite**

1. **Sprint 2** : Implémentation basique avec verrous
2. **Sprint 2.5** (si temps) : Optimisations (cache, séquence)
3. **Sprint 3** : Monitoring et alerting

### Points de vigilance

- ⚠️ **Ne pas sous-estimer la complexité** des transactions multi-étapes
- ⚠️ **Tester la concurrence** dès le début (race conditions)
- ⚠️ **Sécuriser les clés** dès le départ (HSM si possible)
- ⚠️ **Prévoir la scalabilité** (cache, partitionnement futur)
- ⚠️ **Documenter les décisions** techniques (ADR)

---

## 📝 Conclusion

Le plan Sprint 2 présente une **vision solide** pour rendre les documents vérifiables via JWS et Ledger. La séparation des responsabilités est bien pensée.

**Points forts** :
- ✅ Architecture JWS + Ledger claire
- ✅ Sécurité des clés bien pensée
- ✅ Plan de tests complet

**Points d'attention** :
- ⚠️ Complexité des transactions multi-étapes
- ⚠️ Concurrence sur le ledger (race conditions)
- ⚠️ Performance avec gros volumes

**Recommandation principale** : **Simplifier le flux transactionnel** et **ajouter des verrous exclusifs** sur le ledger dès le départ.

---

**Document créé le** : Janvier 2025  
**Version** : 1.0  
**Prochaine révision** : Après validation de l'équipe

