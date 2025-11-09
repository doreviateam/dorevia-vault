# 📋 Plan de Développement Phase 3
## Dorevia Vault — Proxy d'Intégrité Odoo ↔ PDP/PPF

**Date** : Janvier 2025  
**Basé sur** : FICHE_DE_CONCEPTION_TECHNIQUE_PHASE_3.MD  
**Objectif** : Implémenter la règle des 3V (Validé → Vaulté → Vérifiable)

---

## 🎯 Analyse du document de conception

### Concepts clés identifiés

1. **Règle des 3V** : Validé → Vaulté → Vérifiable
   - **Validé** : Document dans un état juridiquement engageant dans Odoo
   - **Vaulté** : Hash SHA-256, JWS, inscription dans ledger
   - **Vérifiable** : Preuve indépendamment vérifiable (JWKS, ledger, TSA)

2. **Rôle du Vault** : Proxy d'intégrité (pas un PDP)
   - Réception depuis Odoo
   - Scellement (hash, JWS, ledger)
   - Routage vers PDP/PPF si nécessaire
   - Webhooks de statut vers Odoo

3. **Flux unifié** : Même pipeline pour tous les types de documents
   - Factures (ventes, achats, avoirs)
   - Tickets POS
   - Bons de livraison (optionnel)
   - Commandes (optionnel)

---

## 📊 État actuel vs. Besoins

### ✅ Ce qui existe déjà

| Fonctionnalité | État | Notes |
|:---------------|:-----|:------|
| Hash SHA-256 | ✅ | Implémenté dans `/upload` |
| Stockage fichiers | ✅ | Organisé par date |
| Base de données | ✅ | Table `documents` basique |
| Upload multipart | ✅ | Endpoint `/upload` |
| Recherche/listing | ✅ | Endpoint `/documents` |

### ❌ Ce qui manque (selon la fiche)

| Fonctionnalité | Priorité | Complexité |
|:---------------|:---------|:------------|
| Métadonnées enrichies (source, odoo_id, pdp_required) | 🔴 Haute | Faible |
| JWS (Jeton signé) | 🔴 Haute | Moyenne |
| Ledger hash-chaîné | 🔴 Haute | Moyenne |
| Validation Factur-X | 🟡 Moyenne | Moyenne |
| Endpoint `/api/v1/invoices` | 🔴 Haute | Faible |
| Webhooks vers Odoo | 🔴 Haute | Moyenne |
| Dispatch PDP/PPF | 🟡 Moyenne | Élevée |
| JWKS public | 🟡 Moyenne | Moyenne |
| TSA (horodatage) | 🟢 Basse | Élevée |

---

## 🗺️ Plan de développement par itérations

### Itération 1 : Fondations Odoo (Semaine 1-2)

**Objectif** : Préparer l'intégration avec Odoo

#### 1.1 Extension du modèle Document

**Fichier** : `internal/models/document.go`

**Nouveaux champs à ajouter** :
```go
type Document struct {
    // ... champs existants ...
    
    // Métadonnées Odoo
    Source      string    `json:"source"`       // sales|purchase|pos|stock|sale
    OdooModel   string    `json:"odoo_model"`   // account.move, pos.order, etc.
    OdooID      *int      `json:"odoo_id"`      // ID dans Odoo
    OdooState   string    `json:"odoo_state"`   // posted, paid, done, etc.
    
    // Routage PDP
    PDPRequired bool      `json:"pdp_required"` // Nécessite dispatch PDP ?
    DispatchStatus string `json:"dispatch_status"` // PENDING|SENT|ACK|REJECTED
    PDPMessageID *string  `json:"pdp_message_id"`  // ID message PDP
    
    // Preuves d'intégrité
    EvidenceJWS string    `json:"evidence_jws"`    // Jeton JWS signé
    LedgerHash  string    `json:"ledger_hash"`    // Hash dans le ledger
    
    // Métadonnées facture (Factur-X)
    InvoiceNumber string  `json:"invoice_number"`
    InvoiceDate   *time.Time `json:"invoice_date"`
    TotalHT       *float64   `json:"total_ht"`
    TotalTTC      *float64   `json:"total_ttc"`
    Currency      string     `json:"currency"`
    SellerVAT     string     `json:"seller_vat"`
    BuyerVAT      string     `json:"buyer_vat"`
}
```

#### 1.2 Migration base de données

**Fichier** : `internal/storage/postgres.go`

**Nouvelle migration** :
```sql
ALTER TABLE documents ADD COLUMN IF NOT EXISTS source TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_model TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_id INTEGER;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_state TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS pdp_required BOOLEAN DEFAULT false;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS dispatch_status TEXT DEFAULT 'PENDING';
ALTER TABLE documents ADD COLUMN IF NOT EXISTS pdp_message_id TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS evidence_jws TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS ledger_hash TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS invoice_number TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS invoice_date DATE;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS total_ht DECIMAL(10,2);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS total_ttc DECIMAL(10,2);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS currency TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS seller_vat TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS buyer_vat TEXT;
```

#### 1.3 Nouvel endpoint `/api/v1/invoices`

**Fichier** : `internal/handlers/invoices.go`

**Fonctionnalités** :
- Accepte le payload standardisé Odoo (JSON avec base64)
- Validation des métadonnées
- Idempotence via SHA256 ou UUID
- Stockage fichier + métadonnées enrichies
- Retourne les identifiants (id, evidence_jws, ledger_hash)

**Payload attendu** :
```json
{
  "source": "sales",
  "model": "account.move",
  "odoo_id": 12345,
  "state": "posted",
  "pdp_required": true,
  "file": "<base64 PDF Factur-X>",
  "meta": {
    "number": "F2025-00123",
    "invoice_date": "2025-11-09",
    "total_ht": 158.33,
    "total_ttc": 190.00,
    "currency": "EUR",
    "seller_vat": "FRXX...",
    "buyer_vat": "FRYY..."
  }
}
```

---

### Itération 2 : Scellement et Preuves (Semaine 2-3)

**Objectif** : Implémenter la règle des 3V (Vaulté → Vérifiable)

#### 2.1 Génération JWS (Jeton signé)

**Fichier** : `internal/crypto/jws.go`

**Fonctionnalités** :
- Génération de clés RSA/ECDSA
- Signature JWS avec header (kid, alg)
- Rotation des clés (kid)
- Vérification de signature

**Dépendances** :
- `github.com/golang-jwt/jwt/v5`
- `github.com/go-jose/go-jose/v3`

**Structure JWS** :
```json
{
  "protected": {
    "alg": "RS256",
    "kid": "key-2025-01",
    "typ": "JWT"
  },
  "payload": {
    "document_id": "uuid",
    "sha256": "abc123...",
    "timestamp": "2025-01-15T10:30:00Z"
  },
  "signature": "..."
}
```

#### 2.2 Ledger hash-chaîné

**Fichier** : `internal/ledger/ledger.go`

**Fonctionnalités** :
- Ledger append-only (table PostgreSQL)
- Hash chaîné (hash précédent + hash document)
- Export du ledger (vérification)
- Horodatage optionnel (TSA)

**Structure ledger** :
```sql
CREATE TABLE ledger (
  id SERIAL PRIMARY KEY,
  document_id UUID REFERENCES documents(id),
  hash TEXT NOT NULL,
  previous_hash TEXT,
  timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
  evidence_jws TEXT
);
```

**Algorithme** :
```
hash_0 = SHA256(document_0)
hash_1 = SHA256(hash_0 + document_1)
hash_2 = SHA256(hash_1 + document_2)
...
```

#### 2.3 Intégration dans le flux upload

**Modifications** :
- Après stockage fichier → générer JWS
- Après JWS → inscrire dans ledger
- Retourner `evidence_jws` et `ledger_hash` dans la réponse

---

### Itération 3 : Validation Factur-X (Semaine 3-4)

**Objectif** : Valider et extraire les métadonnées Factur-X

#### 3.1 Parser Factur-X

**Fichier** : `internal/validation/facturx.go`

**Fonctionnalités** :
- Extraction XML depuis PDF Factur-X
- Validation schéma EN16931
- Extraction métadonnées (number, date, totals, VAT)
- Mapping vers modèle Document

**Dépendances** :
- Bibliothèque XML Go standard
- Validation XSD (optionnel)

#### 3.2 Validation PDF

**Fichier** : `internal/validation/pdf.go`

**Fonctionnalités** :
- Vérification format PDF valide
- Extraction métadonnées PDF
- Détection Factur-X embarqué

---

### Itération 4 : Webhooks Odoo (Semaine 4)

**Objectif** : Notifier Odoo des changements de statut

#### 4.1 Système de webhooks

**Fichier** : `internal/webhooks/webhooks.go`

**Fonctionnalités** :
- Configuration URL webhook Odoo
- Signature HMAC des webhooks
- Retry avec backoff exponentiel
- Logging des tentatives

**Payload webhook** :
```json
{
  "event": "document.vaulted",
  "document_id": "uuid",
  "odoo_id": 12345,
  "status": "VAULTED",
  "evidence_jws": "...",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

**Signature HMAC** :
```
HMAC-SHA256(payload, secret_key)
Header: X-Vault-Signature
```

#### 4.2 Endpoint de réception statut PDP

**Fichier** : `internal/handlers/pdp_status.go`

**Endpoint** : `POST /api/v1/pdp/status`

**Fonctionnalités** :
- Réception statut depuis adaptateur PDP
- Mise à jour `dispatch_status`
- Déclenchement webhook vers Odoo

---

### Itération 5 : Dispatch PDP/PPF (Semaine 5-6)

**Objectif** : Routage vers PDP/PPF via adaptateur OCA

#### 5.1 Adaptateur PDP

**Fichier** : `internal/integrations/pdp/adapter.go`

**Fonctionnalités** :
- Interface pour différents PDP/PPF
- Format standardisé (UBL, EN16931)
- Gestion des erreurs et retry
- Suivi des statuts

**Note** : L'adaptateur OCA sera fourni séparément, le Vault expose juste l'interface.

---

## 📝 Checklist de développement

### Itération 1 — Fondations Odoo
- [ ] Étendre modèle Document avec métadonnées Odoo
- [ ] Migration base de données (nouveaux champs)
- [ ] Endpoint `/api/v1/invoices` avec payload enrichi
- [ ] Validation payload et idempotence
- [ ] Tests unitaires

### Itération 2 — Scellement
- [ ] Génération JWS avec clés rotatives
- [ ] Ledger hash-chaîné (table + logique)
- [ ] Intégration dans flux upload
- [ ] Endpoint de vérification JWS
- [ ] Tests unitaires

### Itération 3 — Validation Factur-X
- [ ] Parser XML Factur-X
- [ ] Validation schéma EN16931
- [ ] Extraction métadonnées
- [ ] Tests unitaires

### Itération 4 — Webhooks
- [ ] Système de webhooks avec HMAC
- [ ] Retry et gestion d'erreurs
- [ ] Endpoint réception statut PDP
- [ ] Tests unitaires

### Itération 5 — Dispatch PDP
- [ ] Interface adaptateur PDP
- [ ] Routage conditionnel (pdp_required)
- [ ] Suivi des statuts
- [ ] Tests d'intégration

---

## 🔧 Dépendances à ajouter

```bash
# JWT/JWS
go get github.com/golang-jwt/jwt/v5
go get github.com/go-jose/go-jose/v3

# Validation XML
go get github.com/lestrrat-go/libxml2

# PDF (optionnel)
go get github.com/gen2brain/go-fitz
```

---

## 🎯 Priorisation recommandée

### Sprint 1 (2 semaines) : Fondations
1. Extension modèle Document
2. Migration base de données
3. Endpoint `/api/v1/invoices`
4. Tests

### Sprint 2 (2 semaines) : Scellement
1. Génération JWS
2. Ledger hash-chaîné
3. Intégration dans flux
4. Tests

### Sprint 3 (2 semaines) : Validation & Webhooks
1. Validation Factur-X
2. Système webhooks
3. Tests

### Sprint 4 (2 semaines) : Dispatch PDP
1. Interface adaptateur
2. Routage conditionnel
3. Tests d'intégration

---

## 📊 Estimation globale

- **Durée totale** : 8 semaines (2 mois)
- **Complexité** : Moyenne à Élevée
- **Risques** :
  - Complexité JWS et ledger
  - Validation Factur-X (schémas XML)
  - Intégration avec adaptateur OCA (dépendance externe)

---

## ✅ Prochaines actions immédiates

1. **Valider ce plan** avec l'équipe
2. **Commencer Itération 1** : Extension modèle Document
3. **Créer les issues GitHub** pour chaque itération
4. **Mettre à jour la roadmap** dans README

---

**Document créé le** : Janvier 2025  
**Version** : 1.0  
**Auteur** : Analyse technique basée sur FICHE_DE_CONCEPTION_TECHNIQUE_PHASE_3.MD

