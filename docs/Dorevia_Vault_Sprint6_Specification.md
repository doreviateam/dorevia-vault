# 🚀 Dorevia Vault — Sprint 6 Specification
**Version** : 1.0 (Draft)  
**Date** : 2025-01-14  
**Produit** : `dorevia-vault`  
**Cible** : v1.4.0 (après Sprint 5 v1.3.0)  

---

## 1. 🎯 Objectifs du Sprint 6

### 1.1 Objectif général

Apporter à Dorevia Vault la **capacité native de vaultériser des tickets de caisse POS** au format **JSON**, avec la même rigueur que pour les factures (3V : **Validé → Vaulté → Vérifiable**), et préparer la future intégration **TSE-like / PDP / HSM**.

### 1.2 Objectifs spécifiques

1. **Ajouter un endpoint dédié POS** :  
   `POST /api/v1/pos-tickets`  
   permettant d’ingérer des tickets POS au format JSON, provenant notamment d’Odoo (module `dorevia_vault_pos_connector`).

2. **Étendre le modèle de données** pour gérer un nouveau type de document : `pos_ticket`, avec :
   - Hash d’intégrité (SHA256)
   - Entrée dans le ledger
   - JWS de preuve (evidence)

3. **Normaliser la réponse API** pour que les connecteurs (factures & POS) reçoivent un contrat homogène :
   - `id`
   - `sha256_hex`
   - `ledger_hash`
   - `evidence_jws`
   - `created_at`

4. **Préparer l’abstraction crypto/HSM** :
   - Introduire une interface de provider crypto (`Signer`) permettant, plus tard, de déléguer la signature à un HSM ou service externe sans casser l’API.

5. **Renforcer l’observabilité POS** :
   - Compteurs & métriques Prometheus spécifiques POS
   - Logs structurés pour l’ingestion POS

---

## 2. 🧱 Vue d’Ensemble Architecture Sprint 6

### 2.1 Composants impactés

- `internal/handlers/pos_tickets_handler.go` (nouveau)
- `internal/services/pos_tickets_service.go` (nouveau)
- `internal/storage/pos_tickets_repository.go` (nouveau)
- `internal/crypto/signer.go` (nouveau — abstraction HSM-ready)
- `internal/ledger/ledger_service.go` (adapté pour accepter le type `pos_ticket`)
- `internal/metrics/metrics.go` (ajout métriques POS)
- `internal/router/router.go` (enregistrement du nouveau endpoint)
- `migrations/20250114_add_pos_tickets_table.sql` (nouvelle migration DB)

### 2.2 Flux POS (niveau macro)

1. **Appel HTTP** sur `POST /api/v1/pos-tickets` avec un JSON issu d’Odoo.
2. **Handler** :
   - Valide l’authentification, les en-têtes, la taille de requête.
   - Parse le JSON en structure interne `PosTicketPayload`.
   - Transmet au service métier `PosTicketsService`.

3. **Service métier** :
   - Canonicalise le JSON (string stable).
   - Calcule `sha256_hex`.
   - Crée l’entrée DB `pos_ticket` avec état initial.
   - Demande au **LedgerService** de créer une entrée ledger (`document_type = "pos_ticket"`).
   - Demande au **Signer** (interface crypto) de générer un `evidence_jws`.
   - Met à jour l’enregistrement `pos_ticket` avec `ledger_hash` & `evidence_jws`.
   - Retourne un objet `PosTicketResult` au handler.

4. **Handler** :
   - Transforme `PosTicketResult` en réponse API standardisée.
   - Retourne HTTP 201 + payload JSON.

---

## 3. 🗃️ Modèle de Données — POS Tickets

### 3.1 Nouvelle table : `pos_tickets`

Migration SQL proposée : `migrations/20250114_add_pos_tickets_table.sql`

```sql
CREATE TABLE pos_tickets (
    id              UUID PRIMARY KEY,
    tenant          VARCHAR(255) NOT NULL,
    source_system   VARCHAR(64)  NOT NULL DEFAULT 'odoo_pos',
    source_model    VARCHAR(64)  NOT NULL,          -- ex: 'pos.order'
    source_id       VARCHAR(128) NOT NULL,          -- ex: '42' ou 'POS/2025/0001'
    payload_json    JSONB        NOT NULL,          -- JSON brut reçu
    sha256_hex      VARCHAR(64)  NOT NULL,
    ledger_hash     VARCHAR(256),
    evidence_jws    TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    sealed_at       TIMESTAMPTZ,
    -- champs optionnels de contexte
    currency        VARCHAR(8),
    total_incl_tax  NUMERIC(16, 2),
    total_excl_tax  NUMERIC(16, 2),
    pos_session     VARCHAR(128),
    cashier         VARCHAR(255),
    location        VARCHAR(255)
);

CREATE INDEX idx_pos_tickets_tenant ON pos_tickets (tenant);
CREATE INDEX idx_pos_tickets_source ON pos_tickets (source_system, source_model, source_id);
CREATE INDEX idx_pos_tickets_sha256 ON pos_tickets (sha256_hex);
```

### 3.2 Modèle Go associé

Fichier : `internal/storage/pos_tickets_model.go` (exemple)

```go
type PosTicket struct {
    ID           uuid.UUID      `db:"id"`
    Tenant       string         `db:"tenant"`
    SourceSystem string         `db:"source_system"`
    SourceModel  string         `db:"source_model"`
    SourceID     string         `db:"source_id"`
    PayloadJSON  []byte         `db:"payload_json"`
    Sha256Hex    string         `db:"sha256_hex"`
    LedgerHash   *string        `db:"ledger_hash"`
    EvidenceJWS  *string        `db:"evidence_jws"`
    CreatedAt    time.Time      `db:"created_at"`
    SealedAt     *time.Time     `db:"sealed_at"`
    Currency     *string        `db:"currency"`
    TotalInclTax *decimal.Decimal `db:"total_incl_tax"`
    TotalExclTax *decimal.Decimal `db:"total_excl_tax"`
    PosSession   *string        `db:"pos_session"`
    Cashier      *string        `db:"cashier"`
    Location     *string        `db:"location"`
}
```

> Remarque : types exacts (`decimal.Decimal`, etc.) à adapter selon les conventions déjà utilisées dans le code existant.

---

## 4. 🌐 API — Endpoint `/api/v1/pos-tickets`

### 4.1 Signature

- **Méthode** : `POST`
- **URL** : `/api/v1/pos-tickets`
- **Auth** : Même mécanisme que pour `/api/v1/invoices` (header API key / bearer token, à confirmer avec la config actuelle)
- **Content-Type** : `application/json`

### 4.2 Payload — Requête

Exemple de payload attendu (provenant d’Odoo POS Connector) :

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

### 4.3 Règles de validation

- `tenant` : obligatoire, non vide
- `source_model` : obligatoire (`"pos.order"` au minimum)
- `source_id` : obligatoire
- `ticket` : objet JSON obligatoire
- Taille max du corps (configurable) : ex. 64 KB

En cas de violation :
- Erreur HTTP 400 avec un corps JSON de type :

```json
{
  "error": "validation_error",
  "message": "missing field: ticket"
}
```

### 4.4 Réponse — Succès

HTTP **201 Created** :

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

### 4.5 Réponse — Erreurs

- **401 Unauthorized** : Auth manquante / invalide.
- **413 Payload Too Large** : Body > limite.
- **422 Unprocessable Entity** : JSON invalide / incohérent.
- **500 Internal Server Error** : Problème interne (DB, ledger, signer…).
  - Le message retourné doit être générique, détails en log seulement.

---

## 5. 🔐 Abstraction Crypto — HSM Ready

### 5.1 Interface `Signer`

Fichier : `internal/crypto/signer.go`

```go
type Signer interface {
    // SignPayload signe un payload arbitraire et retourne un JWS compact.
    SignPayload(ctx context.Context, payload []byte) (string, error)

    // KeyID retourne l'identifiant de la clé utilisée (utile pour la traçabilité).
    KeyID() string
}
```

### 5.2 Implémentation par défaut

Fichier : `internal/crypto/local_signer.go`

- Utilise la clé locale actuelle (comme en Sprint 5)
- Implémente `Signer`

### 5.3 Intégration dans le Ledger / Service POS

Dans `PosTicketsService` :

```go
type PosTicketsService struct {
    repo   storage.PosTicketsRepository
    ledger ledger.Service
    signer crypto.Signer
}

func (s *PosTicketsService) Ingest(ctx context.Context, payload PosTicketPayload) (*PosTicketResult, error) {
    // ... création pos_ticket, calcul hash, appel ledger ...
    evidencePayload := buildEvidencePayload(ticket, ledgerEntry)

    evidenceJWS, err := s.signer.SignPayload(ctx, evidencePayload)
    if err != nil {
        return nil, fmt.Errorf("sign evidence: %w", err)
    }

    // mise à jour pos_ticket avec evidenceJWS
}
```

> L’abstraction `Signer` permet plus tard d’introduire un `HsmSigner` sans toucher l’API ni la logique métier.

---

## 6. 📊 Observabilité & Métriques

### 6.1 Nouvelles métriques Prometheus

Dans `internal/metrics/metrics.go` :

- `dorevia_pos_tickets_ingested_total{tenant="...",source="odoo_pos"}`
- `dorevia_pos_tickets_failed_total{reason="validation|ledger|signer|db"}`
- `dorevia_pos_tickets_duration_seconds` (histogramme, latence par requête)

### 6.2 Logs structurés

Pour chaque requête POS réussie :

```json
{
  "level": "info",
  "msg": "pos ticket ingested",
  "tenant": "laplatine",
  "source_model": "pos.order",
  "source_id": "POS/2025/0001",
  "sha256_hex": "ab12cd34...",
  "ledger_hash": "LEDGER:POS:00000123"
}
```

Pour chaque erreur :
- `reason` explicite (`validation`, `ledger`, `signer`, `db`)
- `tenant` si connu

---

## 7. 🧪 Tests à Réaliser (Sprint 6)

### 7.1 Tests unitaires — Handler

Fichier : `internal/handlers/pos_tickets_handler_test.go`

Cas à couvrir :
1. Requête valide → 201 + payload complet
2. JSON invalide → 400
3. Champ obligatoire manquant → 400
4. Taille > limite → 413
5. Service renvoie erreur validation → 422
6. Service renvoie erreur interne → 500

### 7.2 Tests unitaires — Service POS

Fichier : `internal/services/pos_tickets_service_test.go`

Cas à couvrir :
1. Calcul correct de `sha256_hex` à partir du JSON canonicalisé
2. Idempotence : même payload → même hash & détection de doublon (optionnel)
3. Erreur ledger → erreur remontée
4. Erreur signer → erreur remontée
5. Persistance correcte en DB (mock repository)

### 7.3 Tests d’intégration

- Démarrer Vault avec DB test
- Appeler `POST /api/v1/pos-tickets` avec un payload Odoo complet
- Vérifier :
  - Ligne créée en table `pos_tickets`
  - Entrée ledger créée
  - `evidence_jws` non vide
  - Métriques incrémentées

---

## 8. 🔁 Compatibilité & Migration

### 8.1 Compatibilité API

- Tous les endpoints existants (`/api/v1/invoices`, etc.) restent **inchangés**.
- Aucun impact sur les clients actuels.

### 8.2 Migration DB

- Exécuter la migration `20250114_add_pos_tickets_table.sql` lors du déploiement v1.4.0.
- Pas de donnée legacy à migrer (nouvelle fonctionnalité).

### 8.3 Versionning

- Tag Git recommandé : `v1.4.0`
- Changelog : ajouter section Sprint 6 (POS + HSM abstraction).

---

## 9. ✅ Critères de Finition Sprint 6 (Definition of Done)

Pour considérer le Sprint 6 comme **terminé** :

1. ✅ Endpoint `POST /api/v1/pos-tickets` disponible et documenté
2. ✅ Table `pos_tickets` créée en DB, accessible via repository
3. ✅ Intégration complète avec le ledger & signer
4. ✅ Réponse API standardisée (id, sha256_hex, ledger_hash, evidence_jws, created_at)
5. ✅ Métriques Prometheus POS exposées
6. ✅ Tests unitaires & d’intégration verts
7. ✅ Documentation mise à jour (`README`, `API.md`, `CHANGELOG`)
8. ✅ Tag `v1.4.0` poussé et image Docker buildée

---

## 10. 📌 Synthèse pour le Connecteur Odoo POS

Une fois le Sprint 6 livré :

- Le module Odoo `dorevia_vault_pos_connector` pourra :  
  - Construire un JSON POS à partir de `pos.order`
  - Appeler `POST /api/v1/pos-tickets`
  - Récupérer `id`, `sha256_hex`, `ledger_hash`, `evidence_jws`
  - Stocker ces informations dans les champs `vault_*` de `pos.order`

En clair : **le Sprint 6 ouvre officiellement la voie au “TSE-like souverain” basé sur Dorevia Vault + Odoo POS.**

---

**Auteur** : Assistant technique Dorevia Vault  
**Statut** : Draft validé pour implémentation Sprint 6
