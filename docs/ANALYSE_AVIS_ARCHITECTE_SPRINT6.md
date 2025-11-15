# 🔍 Analyse de l'Avis d'Architecte — Sprint 6
**Date** : 2025-01-14  
**Document analysé** : `Avis_Architeque_Team.md`  
**Plan concerné** : `PLAN_IMPLEMENTATION_SPRINT6_CORRIGE.md`

---

## 📋 Résumé Exécutif

L'avis d'architecte identifie **7 points critiques** à corriger avant l'implémentation. Tous sont **actionnables** et amélioreront significativement la qualité architecturale du code.

**Statut** : ⚠️ **Corrections nécessaires avant implémentation**

---

## 🔴 Points Critiques à Corriger

### 1. ❗ Cohérence Repository / Database

**Problème identifié** :
- Le plan utilise `*storage.DB` directement avec manipulation de transactions (`Pool.Begin`)
- Pas d'interface `DocumentRepository` pour abstraire la couche de stockage
- Le service connaît les détails d'implémentation SQL

**Impact** :
- ❌ Service difficilement testable (nécessite une vraie DB)
- ❌ Violation du principe de dépendance inverse
- ❌ Couplage fort avec PostgreSQL

**Correction requise** :

Créer une interface `DocumentRepository` dans `internal/storage/repository.go` :

```go
package storage

import (
    "context"
    "github.com/doreviateam/dorevia-vault/internal/models"
    "github.com/google/uuid"
)

// DocumentRepository définit les opérations de stockage des documents
type DocumentRepository interface {
    // GetDocumentBySHA256 récupère un document par son hash SHA256
    GetDocumentBySHA256(ctx context.Context, sha256 string) (*models.Document, error)
    
    // InsertDocumentWithEvidence insère un document avec evidence JWS et ledger hash
    // Gère la transaction en interne
    InsertDocumentWithEvidence(
        ctx context.Context,
        doc *models.Document,
        evidenceJWS string,
        ledgerHash string,
    ) error
}
```

**Modification du plan** :
- Phase 1 : Ajouter création de l'interface `DocumentRepository`
- Phase 3 : Utiliser `DocumentRepository` au lieu de `*storage.DB` dans `PosTicketsService`

---

### 2. ❗ Ledger : Service vs Fonction Globale

**Problème identifié** :
- Le plan mentionne `ledger.Service` mais le code existant utilise `ledger.AppendLedger()` (fonction globale)
- Incohérence : `ledger.Service` n'existe pas encore

**Impact** :
- ❌ Incohérence architecturale
- ❌ Moins testable (pas mockable)
- ❌ Incompatible avec l'approche `Signer` (interface)

**Correction requise** :

Créer une interface `Service` dans `internal/ledger/service.go` :

```go
package ledger

import (
    "context"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

// Service définit les opérations sur le ledger
type Service interface {
    // Append ajoute une entrée au ledger avec hash chaîné
    // Prend une transaction en paramètre pour garantir l'atomicité
    Append(ctx context.Context, tx pgx.Tx, docID uuid.UUID, shaHex, jws string) (string, error)
    
    // ExistsByDocumentID vérifie si un document existe dans le ledger
    ExistsByDocumentID(ctx context.Context, tx pgx.Tx, docID uuid.UUID) (bool, error)
}
```

Créer une implémentation dans `internal/ledger/service_impl.go` :

```go
package ledger

import (
    "context"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
)

// DefaultService implémente Service avec la logique existante
type DefaultService struct{}

func NewService() Service {
    return &DefaultService{}
}

func (s *DefaultService) Append(ctx context.Context, tx pgx.Tx, docID uuid.UUID, shaHex, jws string) (string, error) {
    return AppendLedger(ctx, tx, docID, shaHex, jws)
}

func (s *DefaultService) ExistsByDocumentID(ctx context.Context, tx pgx.Tx, docID uuid.UUID) (bool, error) {
    return ExistsByDocumentID(ctx, tx, docID)
}
```

**Modification du plan** :
- Phase 2 : Créer l'interface `ledger.Service` et l'implémentation
- Phase 3 : Utiliser `s.ledger.Append()` au lieu de `ledger.AppendLedger()`

---

### 3. ❗ Dépendance Inverse (services → handlers)

**Problème identifié** :
- Le service POS importe `handlers.PosTicketPayload`
- Violation de la hiérarchie : `services` ne doit pas dépendre de `handlers`

**Impact** :
- ❌ Architecture inversée (couches supérieures dépendent des inférieures)
- ❌ Réutilisabilité limitée
- ❌ Tests plus complexes

**Correction requise** :

Définir `PosTicketInput` dans `internal/services/pos_tickets_types.go` :

```go
package services

// PosTicketInput représente l'input pour l'ingestion d'un ticket POS
type PosTicketInput struct {
    Tenant       string
    SourceSystem string
    SourceModel  string
    SourceID     string
    Currency     *string
    TotalInclTax *float64
    TotalExclTax *float64
    PosSession   *string
    Cashier      *string
    Location     *string
    Ticket       map[string]interface{} // JSON brut du ticket
}
```

Le handler mappe `handlers.PosTicketPayload` → `services.PosTicketInput`.

**Modification du plan** :
- Phase 3 : Créer `services/pos_tickets_types.go` avec `PosTicketInput`
- Phase 4 : Le handler mappe vers `PosTicketInput` avant d'appeler le service

---

### 4. ❗ Idempotence : Clarification Stratégique

**Problème identifié** :
- Actuellement : hash = canonical(payload complet)
- Conséquence : changement de métadonnée (ex: `cashier`) → nouveau document

**Impact** :
- ⚠️ Potentiellement non désiré pour POS (métadonnées peuvent changer)
- ⚠️ Pas de décision explicite documentée

**Correction requise** :

**Option A (Recommandée pour POS)** : Idempotence métier stricte
```go
// Hash basé sur ticket + source_id + session (plus stable)
hashInput := map[string]interface{}{
    "ticket": payload.Ticket,
    "source_id": payload.SourceID,
    "pos_session": payload.PosSession,
}
```

**Option B** : Idempotence totale (actuelle)
```go
// Hash basé sur payload complet (actuel)
hashInput := payload
```

**Décision à prendre** : Choisir Option A pour POS (plus stable métier).

**Modification du plan** :
- Phase 1 : Documenter la stratégie d'idempotence dans la spécification
- Phase 3 : Implémenter l'Option A (idempotence métier stricte)

---

### 5. ⚠️ Canonicalisation JSON : Documentation

**Problème identifié** :
- Algorithme de canonicalisation implémenté mais pas documenté dans l'API
- Risque de confusion pour les clients externes

**Correction requise** :

Ajouter dans `docs/API.md` (ou créer `docs/POS_TICKETS_API.md`) :

```markdown
## Canonicalisation JSON

Les tickets POS sont canonicalisés avant calcul du hash SHA256 :

1. **Tri des clés** : Toutes les clés sont triées alphabétiquement
2. **Suppression des null** : Les valeurs `null` sont supprimées
3. **Normalisation des nombres** : `10.0` → `10` (si entier)

**Conséquence** : Deux JSON différents peuvent produire le même hash.

Exemple :
```json
{"b": 2, "a": 1, "c": null}
```
et
```json
{"a": 1, "b": 2.0}
```
produisent le même hash.
```

**Modification du plan** :
- Phase 6 : Ajouter documentation de la canonicalisation dans l'API

---

### 6. ⚠️ Micro Incohérences Go

**Problèmes identifiés** :
- `import {` → `import (`
- Struct tags incorrects (`"json:"id"` → `json:"id"`)
- `fmt` manquant dans les imports
- Erreurs de compilation potentielles

**Correction requise** :

Vérifier tous les exemples de code dans le plan :
- ✅ Imports corrects : `import (`
- ✅ Struct tags corrects : `json:"id"` (sans guillemets autour de `json`)
- ✅ Tous les packages nécessaires importés (`fmt`, `context`, etc.)

**Modification du plan** :
- Phase 0 (avant implémentation) : Review complet du code d'exemple dans le plan

---

### 7. ✅ Stockage POS : Choix Unifié

**Statut** : ✅ **Déjà clarifié dans le plan**

Le plan sélectionne **Option A** (stockage DB uniquement) :
- Stockage dans `payload_json JSONB`
- Pas de fichier `.json` dans le filesystem

**Action** : Ajouter une note explicite :
> "Le mode fichier pourra être considéré dans une future release."

---

## 📝 Plan d'Action Corrigé

### Phase 0 : Préparation Architecturale (Avant Phase 1)

- [ ] **Créer interface `DocumentRepository`**
  - [ ] Fichier : `internal/storage/repository.go`
  - [ ] Interface avec méthodes nécessaires
  - [ ] Implémentation dans `internal/storage/postgres_repository.go`

- [ ] **Créer interface `ledger.Service`**
  - [ ] Fichier : `internal/ledger/service.go`
  - [ ] Interface avec méthode `Append()`
  - [ ] Implémentation dans `internal/ledger/service_impl.go`

- [ ] **Créer types services**
  - [ ] Fichier : `internal/services/pos_tickets_types.go`
  - [ ] Type `PosTicketInput` (sans dépendance handlers)

- [ ] **Documenter stratégie d'idempotence**
  - [ ] Ajouter section dans `docs/API.md`
  - [ ] Choisir Option A (idempotence métier stricte)

- [ ] **Review code d'exemple**
  - [ ] Vérifier imports Go
  - [ ] Vérifier struct tags
  - [ ] Corriger micro-incohérences

### Modifications des Phases Existantes

**Phase 1** : Ajouter tests de l'interface `DocumentRepository`

**Phase 2** : Ajouter tests de l'interface `ledger.Service`

**Phase 3** : 
- Utiliser `DocumentRepository` au lieu de `*storage.DB`
- Utiliser `s.ledger.Append()` au lieu de `ledger.AppendLedger()`
- Utiliser `services.PosTicketInput` au lieu de `handlers.PosTicketPayload`
- Implémenter idempotence métier stricte (Option A)

**Phase 6** : Ajouter documentation canonicalisation JSON

---

## ✅ Checklist de Validation

Avant de commencer l'implémentation, vérifier :

- [ ] Interface `DocumentRepository` créée et testée
- [ ] Interface `ledger.Service` créée et testée
- [ ] Type `PosTicketInput` créé (sans dépendance handlers)
- [ ] Stratégie d'idempotence documentée et choisie
- [ ] Code d'exemple du plan corrigé (imports, struct tags)
- [ ] Documentation canonicalisation JSON ajoutée

---

## 🎯 Impact des Corrections

### Avant Corrections
- ❌ Service couplé à PostgreSQL
- ❌ Tests nécessitent vraie DB
- ❌ Architecture inversée
- ❌ Idempotence non documentée

### Après Corrections
- ✅ Service testable avec mocks
- ✅ Architecture propre (dépendances correctes)
- ✅ Idempotence claire et documentée
- ✅ Code prêt pour certification

---

## 📊 Évaluation Finale

| Critère | Avant | Après |
|:--------|:------|:------|
| **Testabilité** | ⭐⭐ (2/5) | ⭐⭐⭐⭐⭐ (5/5) |
| **Architecture** | ⭐⭐⭐ (3/5) | ⭐⭐⭐⭐⭐ (5/5) |
| **Maintenabilité** | ⭐⭐⭐⭐ (4/5) | ⭐⭐⭐⭐⭐ (5/5) |
| **Documentation** | ⭐⭐⭐ (3/5) | ⭐⭐⭐⭐⭐ (5/5) |

**Note globale** : **4.5/5 → 5/5** après corrections

---

## 🏁 Conclusion

Les **7 points identifiés** par l'architecte sont **tous valides** et doivent être corrigés avant l'implémentation. Les corrections sont **actionnables** et amélioreront significativement :

- ✅ Testabilité du code
- ✅ Qualité architecturale
- ✅ Maintenabilité
- ✅ Prêt pour certification

**Recommandation** : **Créer une Phase 0** pour implémenter les interfaces et types avant de commencer la Phase 1.

---

**Auteur** : Analyse technique Dorevia Vault  
**Date** : 2025-01-14  
**Version** : 1.0

