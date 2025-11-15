# 📄 API Tickets POS — Documentation

**Version** : 1.0  
**Date** : 2025-01-14  
**Sprint** : Sprint 6

---

## 🎯 Endpoint

**POST** `/api/v1/pos-tickets`

Ingère un ticket de caisse POS au format JSON avec la même rigueur que pour les factures (3V : **Validé → Vaulté → Vérifiable**).

---

## 📥 Payload — Requête

### Structure

```json
{
  "tenant": "laplatine",
  "source_system": "odoo_pos",
  "source_model": "pos.order",
  "source_id": "POS/2025/0001",
  "currency": "EUR",
  "total_incl_tax": 12.50,
  "total_excl_tax": 10.42,
  "pos_session": "SESSION/2025/01/14-01",
  "cashier": "Verena",
  "location": "La Platine - Boutique",
  "ticket": {
    "lines": [
      {
        "product": "Crêpe Manioc Sucre",
        "quantity": 2,
        "unit_price": 3.50,
        "taxes": [
          {"name": "TVA 8.5%", "amount": 0.55}
        ]
      }
    ],
    "payments": [
      {
        "method": "CB",
        "amount": 12.50
      }
    ],
    "timestamp": "2025-01-14T10:42:00Z"
  }
}
```

### Champs

| Champ | Type | Obligatoire | Description |
|:------|:-----|:------------|:------------|
| `tenant` | string | ✅ | Identifiant du tenant |
| `source_system` | string | ❌ | Système source (défaut: `"odoo_pos"`) |
| `source_model` | string | ✅ | Modèle source (ex: `"pos.order"`) |
| `source_id` | string | ✅ | Identifiant source (ex: `"POS/2025/0001"`) |
| `currency` | string | ❌ | Devise (ex: `"EUR"`) |
| `total_incl_tax` | number | ❌ | Total TTC |
| `total_excl_tax` | number | ❌ | Total HT |
| `pos_session` | string | ❌ | Session POS |
| `cashier` | string | ❌ | Caissier |
| `location` | string | ❌ | Localisation |
| `ticket` | object | ✅ | JSON brut du ticket |

---

## 📤 Réponse — Succès

**HTTP 201 Created**

```json
{
  "id": "6b78d57a-3d3c-4b2e-bae3-3a8ee3fdc8d0",
  "tenant": "laplatine",
  "sha256_hex": "ab12cd34ef5678901234567890abcdef1234567890abcdef1234567890abcdef",
  "ledger_hash": "LEDGER:POS:00000123",
  "evidence_jws": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...",
  "created_at": "2025-01-14T10:42:01Z"
}
```

### Champs de Réponse

| Champ | Type | Description |
|:------|:-----|:------------|
| `id` | string (UUID) | Identifiant unique du document |
| `tenant` | string | Tenant du ticket |
| `sha256_hex` | string | Hash SHA256 pour idempotence |
| `ledger_hash` | string | Hash dans le ledger (si activé) |
| `evidence_jws` | string | Preuve JWS (si activé) |
| `created_at` | string (ISO 8601) | Date de création |

---

## ❌ Réponses — Erreurs

### 400 Bad Request

**JSON invalide** :
```json
{
  "error": "Invalid JSON payload",
  "details": "..."
}
```

**Champ manquant** :
```json
{
  "error": "Missing required field: tenant"
}
```

### 401 Unauthorized

**Authentification requise** :
```json
{
  "error": "Unauthorized"
}
```

### 413 Request Entity Too Large

**Payload trop volumineux** :
```json
{
  "error": "Payload too large",
  "max_size_bytes": 65536
}
```

### 422 Unprocessable Entity

**Validation échouée** :
```json
{
  "error": "validation_error",
  "message": "..."
}
```

### 500 Internal Server Error

**Erreur serveur** :
```json
{
  "error": "Failed to ingest POS ticket"
}
```

---

## 🔐 Stratégie d'Idempotence

### Option A : Idempotence Métier Stricte (Implémentée)

**Hash basé sur** : `ticket + source_id + pos_session`

**Conséquence** :
- ✅ Un même ticket avec le même `source_id` et `pos_session` → même document (idempotent)
- ✅ Changement de métadonnées (`cashier`, `location`) → **même document** (idempotent)
- ✅ Changement de `ticket`, `source_id` ou `pos_session` → **nouveau document**

**Exemple** :

```json
// Premier appel
{
  "tenant": "laplatine",
  "source_id": "POS/2025/0001",
  "pos_session": "SESSION/001",
  "cashier": "Verena",
  "ticket": {...}
}
// → Document créé avec hash H1

// Deuxième appel (même ticket, cashier différent)
{
  "tenant": "laplatine",
  "source_id": "POS/2025/0001",
  "pos_session": "SESSION/001",
  "cashier": "Marie",  // ← Changé
  "ticket": {...}      // ← Identique
}
// → Retourne le même document (idempotent, hash H1)
```

**Avantages** :
- ✅ Stabilité métier : un ticket POS est identifié par son contenu + source_id + session
- ✅ Métadonnées optionnelles n'affectent pas l'idempotence
- ✅ Adapté aux cas d'usage POS (corrections de métadonnées)

---

## 📝 Canonicalisation JSON

### Algorithme

Les tickets POS sont **canonicalisés** avant calcul du hash SHA256 pour garantir la stabilité :

1. **Tri des clés** : Toutes les clés sont triées alphabétiquement (récursif)
2. **Suppression des null** : Les valeurs `null` sont supprimées
3. **Normalisation des nombres** : `10.0` → `10` (si entier)

### Exemple

**Input** :
```json
{
  "b": 2,
  "a": 1,
  "c": null,
  "d": 10.0
}
```

**Canonicalisé** :
```json
{
  "a": 1,
  "b": 2,
  "d": 10
}
```

**Conséquence** : Deux JSON différents peuvent produire le **même hash** si leur contenu canonique est identique.

### Exemples de Canonicalisation

#### Exemple 1 : Ordre des clés

```json
// Input 1
{"b": 2, "a": 1}

// Input 2
{"a": 1, "b": 2}

// Canonicalisé (identique)
{"a": 1, "b": 2}
// → Même hash SHA256
```

#### Exemple 2 : Valeurs null

```json
// Input 1
{"a": 1, "b": null, "c": 3}

// Input 2
{"a": 1, "c": 3}

// Canonicalisé (identique)
{"a": 1, "c": 3}
// → Même hash SHA256
```

#### Exemple 3 : Normalisation nombres

```json
// Input 1
{"a": 10.0, "b": 10.5}

// Input 2
{"a": 10, "b": 10.5}

// Canonicalisé (identique)
{"a": 10, "b": 10.5}
// → Même hash SHA256
```

---

## 🔄 Idempotence + Canonicalisation

**Comportement combiné** :

1. Le payload est **canonicalisé** (tri clés, suppression null, normalisation nombres)
2. Le hash est calculé sur `ticket + source_id + pos_session` (idempotence métier stricte)
3. Si le hash existe déjà → retourne le document existant (HTTP 200 OK)
4. Sinon → crée un nouveau document (HTTP 201 Created)

**Exemple** :

```json
// Premier appel
POST /api/v1/pos-tickets
{
  "source_id": "POS/001",
  "pos_session": "SESSION/001",
  "ticket": {"b": 2, "a": 1, "c": null}
}
// → Hash calculé sur canonical({"a": 1, "b": 2}) + "POS/001" + "SESSION/001"
// → Document créé

// Deuxième appel (identique après canonicalisation)
POST /api/v1/pos-tickets
{
  "source_id": "POS/001",
  "pos_session": "SESSION/001",
  "ticket": {"a": 1, "b": 2.0}  // Ordre différent, null supprimé, nombre normalisé
}
// → Hash identique (canonicalisation)
// → Retourne le document existant (idempotent)
```

---

## 📊 Variables d'Environnement

| Variable | Description | Défaut |
|:---------|:------------|:-------|
| `POS_TICKET_MAX_SIZE_BYTES` | Taille max du payload (bytes) | `65536` (64 KB) |

---

## 🔗 Voir Aussi

- [Plan d'Implémentation Sprint 6](../docs/PLAN_IMPLEMENTATION_SPRINT6_CORRIGE.md)
- [Analyse Experte Sprint 6](../docs/ANALYSE_EXPERTE_SPRINT6.md)

---

**Auteur** : Documentation Dorevia Vault  
**Date** : 2025-01-14  
**Version** : 1.0

