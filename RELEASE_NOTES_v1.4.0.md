# 🚀 Dorevia Vault v1.4.0 — « Ingestion Native Tickets POS »

**Date de publication :** Janvier 2025

**Auteur :** Doreviateam

**Version :** v1.4.0

**État :** Stable — Production ready

---

## 🌟 Aperçu général

Cette version marque la **fin du Sprint 6** et introduit l'**ingestion native des tickets POS** au format JSON avec la même rigueur que pour les factures (3V : **Validé → Vaulté → Vérifiable**).

Elle apporte une **architecture modulaire** avec interfaces abstraites, une **canonicalisation JSON** pour garantir la stabilité des hash, et une **observabilité complète** avec métriques Prometheus et logs structurés.

---

## 🧩 Nouveautés majeures

### 1. Endpoint POS Tickets (Phase 4)

#### Nouvel Endpoint : `POST /api/v1/pos-tickets`

- Ingestion native des tickets POS au format JSON
- Validation complète du payload (taille, champs obligatoires)
- Réponse standardisée avec métadonnées complètes
- Support idempotence métier stricte

**Payload exemple :**
```json
{
  "tenant": "laplatine",
  "source_model": "pos.order",
  "source_id": "POS/2025/0001",
  "currency": "EUR",
  "total_incl_tax": 12.50,
  "total_excl_tax": 10.42,
  "pos_session": "SESSION/2025/01/14-01",
  "cashier": "Verena",
  "location": "La Platine - Boutique",
  "ticket": {
    "lines": [...],
    "payments": [...]
  }
}
```

**Réponse :**
```json
{
  "id": "6b78d57a-3d3c-4b2e-bae3-3a8ee3fdc8d0",
  "tenant": "laplatine",
  "sha256_hex": "ab12cd34...",
  "ledger_hash": "LEDGER:POS:00000123",
  "evidence_jws": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...",
  "created_at": "2025-01-14T10:42:01Z"
}
```

---

### 2. Architecture Modulaire (Phase 0)

#### Interfaces Abstraites

- **`DocumentRepository`** : Abstraction de la couche de stockage
  - `GetDocumentBySHA256()` : Récupération par hash
  - `InsertDocumentWithEvidence()` : Insertion avec ledger et JWS

- **`ledger.Service`** : Abstraction du service ledger
  - `Append()` : Ajout d'entrée avec hash chaîné
  - `ExistsByDocumentID()` : Vérification d'existence

- **`crypto.Signer`** : Abstraction de la signature (HSM-ready)
  - `SignPayload()` : Signature d'un payload Evidence
  - `KeyID()` : Identifiant de la clé actuelle

**Avantages :**
- Testabilité améliorée (mocks faciles)
- Extensibilité (HSM, autres backends)
- Séparation des responsabilités

---

### 3. Canonicalisation JSON (Phase 1)

#### Algorithme de Canonicalisation

- **Tri des clés** : Toutes les clés triées alphabétiquement (récursif)
- **Suppression des null** : Valeurs `null` supprimées
- **Normalisation des nombres** : `10.0` → `10` (si entier)

**Exemple :**
```json
// Input 1
{"b": 2, "a": 1, "c": null}

// Input 2
{"a": 1.0, "b": 2}

// Canonicalisé (identique)
{"a": 1, "b": 2}
// → Même hash SHA256
```

**Garantie :** Deux JSON différents mais canoniquement identiques produisent le **même hash**, assurant l'idempotence même avec des variations de format.

---

### 4. Idempotence Métier Stricte (Phase 3)

#### Stratégie : Option A (Implémentée)

**Hash basé sur** : `ticket + source_id + pos_session`

**Comportement :**
- ✅ Un même ticket avec le même `source_id` et `pos_session` → même document (idempotent)
- ✅ Changement de métadonnées (`cashier`, `location`) → **même document** (idempotent)
- ✅ Changement de `ticket`, `source_id` ou `pos_session` → **nouveau document**

**Avantages :**
- Stabilité métier : un ticket POS est identifié par son contenu + source_id + session
- Métadonnées optionnelles n'affectent pas l'idempotence
- Adapté aux cas d'usage POS (corrections de métadonnées)

---

### 5. Stockage & Migration DB (Phase 1)

#### Nouveaux Champs dans `documents`

- `payload_json JSONB` : JSON brut du ticket POS (stockage DB uniquement)
- `source_id_text TEXT` : ID textuel (pour POS avec IDs string comme "POS/2025/0001")
- `pos_session TEXT` : Session POS
- `cashier TEXT` : Caissier
- `location TEXT` : Localisation

#### Index Optimisés

- **GIN index** sur `payload_json` pour recherche JSON native
- **Index partiels** sur `source_id_text`, `pos_session`, `cashier`, `location` (WHERE source = 'pos')
- **Index composite** sur `(source, odoo_model)` pour recherche POS

**Migration :** `migrations/005_add_pos_fields.sql`

---

### 6. Observabilité (Phase 5)

#### Métriques Prometheus

- `documents_vaulted_total{status="success|idempotent|error", source="pos"}` : Compteur de documents vaultés
- `document_storage_duration_seconds{operation="pos_ingest"}` : Durée d'ingestion

#### Logs Structurés

**Exemple de log :**
```json
{
  "level": "info",
  "message": "POS ticket ingested",
  "tenant": "laplatine",
  "source_model": "pos.order",
  "source_id": "POS/2025/0001",
  "document_id": "6b78d57a-3d3c-4b2e-bae3-3a8ee3fdc8d0",
  "sha256_hex": "ab12cd34...",
  "ledger_hash": "LEDGER:POS:00000123",
  "evidence_jws": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...",
  "duration_seconds": 0.123
}
```

---

## 📊 Statistiques

### Code

- **Lignes de code ajoutées** : ~2000 lignes
- **Fichiers créés** : 17 fichiers
- **Fichiers modifiés** : 4 fichiers

### Tests

- **Tests unitaires** : 20 tests (100% de réussite)
  - 4 tests canonicalisation JSON
  - 7 tests service POS
  - 8 tests handler API
  - 1 test signer

- **Tests d'intégration** : 5 tests (100% de réussite)
  - Test end-to-end
  - Test idempotence
  - Test idempotence avec changement métadonnées
  - Test canonicalisation
  - Test métriques

### Couverture

- `internal/utils` : 100% (canonicalisation)
- `internal/services` : 85.4% (service POS)
- `internal/handlers` : 10.0% (handler API - partagé avec autres handlers)
- `internal/crypto` : 1.9% (signer - partagé avec autres modules)

---

## 🔧 Configuration

### Nouvelle Variable d'Environnement

```bash
# Taille maximale du payload POS (en bytes)
POS_TICKET_MAX_SIZE_BYTES=65536  # Défaut: 64 KB
```

### Prérequis

- PostgreSQL avec migration 005 appliquée
- Clés JWS configurées (`JWS_PRIVATE_KEY_PATH`, `JWS_PUBLIC_KEY_PATH`)
- Base de données avec table `documents` existante

---

## 📝 Documentation

### Nouveaux Documents

- `docs/POS_TICKETS_API.md` : Documentation complète de l'API POS
- `docs/VALIDATION_SPRINT6.md` : Rapport de validation Sprint 6
- `docs/PLAN_IMPLEMENTATION_SPRINT6_CORRIGE.md` : Plan d'implémentation détaillé

### Documentation Mise à Jour

- `README.md` : Ajout section POS tickets
- `CHANGELOG.md` : Ajout entrée v1.4.0

---

## 🚀 Migration depuis v1.3.0

### Étapes de Migration

1. **Appliquer la migration DB** :
   ```bash
   psql $DATABASE_URL -f migrations/005_add_pos_fields.sql
   ```

2. **Redémarrer le service** :
   ```bash
   sudo systemctl restart dorevia-vault
   ```

3. **Vérifier l'endpoint** :
   ```bash
   curl -X POST http://localhost:8080/api/v1/pos-tickets \
     -H "Content-Type: application/json" \
     -d '{
       "tenant": "test",
       "source_model": "pos.order",
       "source_id": "POS/001",
       "ticket": {"lines": []}
     }'
   ```

### Compatibilité

- ✅ **Rétrocompatibilité totale** : Aucun changement breaking
- ✅ **Endpoints existants** : Tous fonctionnent sans modification
- ✅ **Migration DB** : Non-destructive (ajout de colonnes)

---

## 🐛 Corrections de Bugs

Aucun bug corrigé dans cette version (nouvelle fonctionnalité).

---

## 🔒 Sécurité

- ✅ Validation stricte du payload (taille, champs obligatoires)
- ✅ Idempotence métier pour éviter les doublons
- ✅ Intégration avec le ledger (hash chaîné)
- ✅ Intégration avec le signer (JWS)
- ✅ Logs structurés pour audit

---

## 🎯 Prochaines Étapes

### Sprint 7 (Prévu)

- Recherche avancée dans les tickets POS (filtres JSON)
- Export des tickets POS (CSV, JSON)
- Statistiques POS (revenus, produits, sessions)
- Intégration avec systèmes de paiement

---

## 📚 Références

- [Documentation API POS](./docs/POS_TICKETS_API.md)
- [Rapport de Validation](./docs/VALIDATION_SPRINT6.md)
- [Plan d'Implémentation](./docs/PLAN_IMPLEMENTATION_SPRINT6_CORRIGE.md)

---

**Auteur :** Doreviateam  
**Date :** Janvier 2025  
**Version :** v1.4.0

