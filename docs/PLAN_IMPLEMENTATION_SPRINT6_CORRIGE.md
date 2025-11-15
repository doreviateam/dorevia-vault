# 📋 Plan d'Implémentation Sprint 6 — Version Corrigée
**Version** : 1.0 (Corrigée)  
**Date** : 2025-01-14  
**Basé sur** : Analyse experte `ANALYSE_EXPERTE_SPRINT6.md`  
**Cible** : v1.4.0

---

## 🎯 Décisions Architecturales

### ✅ Option Retenue : Réutilisation de la Table `documents`

**Décision** : Utiliser la table `documents` existante au lieu de créer une table `pos_tickets` séparée.

**Justification** :
- ✅ Réutilisation complète du code existant (stockage, hash, JWS, ledger)
- ✅ Ledger unifié (pas de modification de la contrainte FK)
- ✅ Métriques unifiées avec label `source="pos"`
- ✅ Cohérence architecturale (un seul modèle de document)
- ✅ Moins de duplication de code

**Identification des tickets POS** :
- `source = "pos"` (déjà supporté dans la contrainte CHECK)
- `odoo_model = "pos.order"` (déjà présent dans le modèle)
- Champs POS spécifiques ajoutés via migration (voir section 3.1)

---

## 📐 Architecture Corrigée

### 2.1 Composants Impactés

#### Nouveaux Fichiers
- `internal/handlers/pos_tickets_handler.go` — Handler endpoint `/api/v1/pos-tickets`
- `internal/services/pos_tickets_service.go` — Service métier POS
- `internal/crypto/signer.go` — Interface `Signer` (abstraction HSM-ready)
- `internal/crypto/local_signer.go` — Implémentation locale (adaptateur depuis `crypto.Service`)
- `internal/utils/json_canonical.go` — Fonction de canonicalisation JSON
- `migrations/005_add_pos_fields.sql` — Migration pour champs POS dans `documents`

#### Fichiers Modifiés
- `internal/models/document.go` — Ajout champs POS (`PosSession`, `Cashier`, `Location`, `PayloadJSON`)
- `internal/storage/postgres.go` — Ajout migration Sprint 6
- `internal/storage/document_with_evidence.go` — Support JSON (pas de fichier) pour POS
- `internal/metrics/prometheus.go` — Ajout label `document_type` (optionnel) ou réutilisation `source`
- `cmd/vault/main.go` — Enregistrement route `/api/v1/pos-tickets`

#### Fichiers Non Modifiés (Réutilisation)
- `internal/ledger/append.go` — Aucune modification (réutilisé tel quel)
- `internal/crypto/jws.go` — Aucune modification (utilisé via adaptateur `LocalSigner`)

---

## 3. 🗃️ Modèle de Données Corrigé

### 3.1 Migration SQL : Ajout Champs POS à `documents`

**Fichier** : `migrations/005_add_pos_fields.sql`

```sql
-- Migration 005: Ajout des champs POS (Sprint 6)
-- Date: 2025-01-14
-- Description: Ajoute les champs spécifiques aux tickets POS à la table documents

-- Champ pour stocker le JSON brut du ticket POS (pour tickets POS uniquement)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS payload_json JSONB;

-- Champ pour source_id textuel (pour POS avec IDs string comme "POS/2025/0001")
ALTER TABLE documents ADD COLUMN IF NOT EXISTS source_id_text TEXT;

-- Champs métier POS (optionnels, NULL pour les documents non-POS)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS pos_session TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS cashier TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS location TEXT;

-- Index pour recherche rapide sur payload_json (GIN index pour JSONB)
CREATE INDEX IF NOT EXISTS idx_documents_payload_json ON documents USING GIN (payload_json);

-- Index pour recherche POS
CREATE INDEX IF NOT EXISTS idx_documents_source_id_text ON documents(source_id_text) WHERE source = 'pos';
CREATE INDEX IF NOT EXISTS idx_documents_pos_session ON documents(pos_session) WHERE source = 'pos';
CREATE INDEX IF NOT EXISTS idx_documents_cashier ON documents(cashier) WHERE source = 'pos';
CREATE INDEX IF NOT EXISTS idx_documents_location ON documents(location) WHERE source = 'pos';

-- Index composite pour recherche par source + odoo_model (optimisation POS)
CREATE INDEX IF NOT EXISTS idx_documents_source_model ON documents(source, odoo_model) 
    WHERE source = 'pos' AND odoo_model = 'pos.order';
```

**Note** : Les champs `currency`, `total_ht`, `total_ttc` existent déjà dans `documents` (migration 003).

### 3.2 Modèle Go Corrigé

**Fichier** : `internal/models/document.go`

```go
// Document représente un document stocké dans le système
type Document struct {
    // ... champs existants (ID, Filename, ContentType, SizeBytes, SHA256Hex, etc.) ...
    
    // Champs POS (Sprint 6) - optionnels, NULL pour documents non-POS
    SourceIDText *string `json:"source_id_text,omitempty" db:"source_id_text"` // ID textuel (pour POS)
    PayloadJSON  []byte  `json:"payload_json,omitempty" db:"payload_json"`     // JSON brut (pour POS)
    PosSession   *string `json:"pos_session,omitempty" db:"pos_session"`
    Cashier      *string `json:"cashier,omitempty" db:"cashier"`
    Location     *string `json:"location,omitempty" db:"location"`
    
    // Note: currency, total_ht, total_ttc existent déjà dans le modèle
}
```

**Important** : 
- `PayloadJSON` est de type `[]byte` en Go mais stocké en `JSONB` en PostgreSQL
- Pour les documents POS, `Filename` peut être généré (ex: `pos-ticket-{source_id}.json`)
- `ContentType` sera `"application/json"` pour les tickets POS
- `StoredPath` peut être `NULL` pour les tickets POS (stockage en DB uniquement) OU stocker le JSON en fichier

---

## 4. 🔐 Abstraction Crypto : Interface `Signer`

### 4.1 Interface `Signer` (Corrigée)

**Fichier** : `internal/crypto/signer.go`

```go
package crypto

import (
    "context"
)

// Signature représente une signature JWS avec son KID
type Signature struct {
    JWS string // JWS compact
    KID string // Key ID
}

// Signer est une interface pour signer des payloads (abstraction HSM-ready)
type Signer interface {
    // SignPayload signe un payload Evidence et retourne une Signature
    // Le payload doit être un JSON marshallé contenant {document_id, sha256, timestamp}
    SignPayload(ctx context.Context, payload []byte) (*Signature, error)
    
    // KeyID retourne l'identifiant de la clé actuelle
    KeyID() string
}
```

### 4.2 Implémentation Locale (Adaptateur)

**Fichier** : `internal/crypto/local_signer.go`

```go
package crypto

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
)

// EvidencePayload représente le payload à signer
type EvidencePayload struct {
    DocumentID string    `json:"document_id"`
    Sha256     string    `json:"sha256"`
    Timestamp  time.Time `json:"timestamp"`
}

// LocalSigner implémente Signer en utilisant crypto.Service (implémentation locale)
type LocalSigner struct {
    service *Service
}

// NewLocalSigner crée un LocalSigner depuis un Service existant
func NewLocalSigner(service *Service) *LocalSigner {
    return &LocalSigner{service: service}
}

// SignPayload signe un payload Evidence
func (s *LocalSigner) SignPayload(ctx context.Context, payload []byte) (*Signature, error) {
    // Parser le payload pour extraire document_id, sha256, timestamp
    var evidence EvidencePayload
    if err := json.Unmarshal(payload, &evidence); err != nil {
        return nil, fmt.Errorf("failed to unmarshal evidence payload: %w", err)
    }
    
    // Utiliser le service JWS existant
    jws, err := s.service.SignEvidence(evidence.DocumentID, evidence.Sha256, evidence.Timestamp)
    if err != nil {
        return nil, fmt.Errorf("failed to sign evidence: %w", err)
    }
    
    return &Signature{
        JWS: jws,
        KID: s.service.GetKID(),
    }, nil
}

// KeyID retourne le KID actuel
func (s *LocalSigner) KeyID() string {
    return s.service.GetKID()
}
```

### 4.3 Utilisation dans le Service POS

```go
// Dans PosTicketsService
type PosTicketsService struct {
    repo   storage.DocumentRepository // Réutilise le repository documents
    ledger ledger.Service
    signer crypto.Signer // Interface abstraite
}

func (s *PosTicketsService) Ingest(ctx context.Context, payload PosTicketPayload) (*PosTicketResult, error) {
    // ... calcul hash, création document ...
    
    // Construire le payload Evidence
    evidencePayload := crypto.EvidencePayload{
        DocumentID: docID.String(),
        Sha256:     sha256Hex,
        Timestamp:  time.Now().UTC(),
    }
    evidenceBytes, _ := json.Marshal(evidencePayload)
    
    // Signer via l'interface Signer
    signature, err := s.signer.SignPayload(ctx, evidenceBytes)
    if err != nil {
        return nil, fmt.Errorf("sign evidence: %w", err)
    }
    
    // Utiliser signature.JWS et signature.KID
    // ...
}
```

---

## 5. 📝 Canonicalisation JSON (Spécifiée)

### 5.1 Algorithme de Canonicalisation

**Fichier** : `internal/utils/json_canonical.go`

```go
package utils

import (
    "encoding/json"
    "sort"
)

// CanonicalizeJSON canonicalise un JSON pour garantir un hash stable
// Algorithme :
// 1. Parser le JSON en map[string]interface{}
// 2. Trier récursivement les clés
// 3. Supprimer les champs null (optionnel, configurable)
// 4. Marshal avec json.Marshal (pas d'indentation, pas d'espaces)
func CanonicalizeJSON(data []byte) ([]byte, error) {
    var obj interface{}
    if err := json.Unmarshal(data, &obj); err != nil {
        return nil, err
    }
    
    // Normaliser récursivement
    normalized := normalizeValue(obj)
    
    // Marshal sans indentation (compact)
    return json.Marshal(normalized)
}

// normalizeValue normalise récursivement une valeur
func normalizeValue(v interface{}) interface{} {
    switch val := v.(type) {
    case map[string]interface{}:
        // Créer une map triée
        sorted := make(map[string]interface{})
        keys := make([]string, 0, len(val))
        for k := range val {
            keys = append(keys, k)
        }
        sort.Strings(keys)
        
        // Copier les valeurs normalisées dans l'ordre trié
        for _, k := range keys {
            // Ignorer les valeurs null (optionnel)
            if val[k] != nil {
                sorted[k] = normalizeValue(val[k])
            }
        }
        return sorted
        
    case []interface{}:
        // Normaliser chaque élément du tableau
        normalized := make([]interface{}, len(val))
        for i, item := range val {
            normalized[i] = normalizeValue(item)
        }
        return normalized
        
    case float64:
        // Normaliser les nombres (10.0 -> 10 si entier)
        if val == float64(int64(val)) {
            return int64(val)
        }
        return val
        
    default:
        return val
    }
}
```

### 5.2 Tests de Canonicalisation

**Fichier** : `internal/utils/json_canonical_test.go`

```go
func TestCanonicalizeJSON(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "simple object",
            input:    `{"b":2,"a":1}`,
            expected: `{"a":1,"b":2}`,
        },
        {
            name:     "with null",
            input:    `{"a":1,"b":null,"c":3}`,
            expected: `{"a":1,"c":3}`, // null supprimé
        },
        {
            name:     "nested object",
            input:    `{"z":{"b":2,"a":1},"y":10}`,
            expected: `{"y":10,"z":{"a":1,"b":2}}`,
        },
        {
            name:     "number normalization",
            input:    `{"a":10.0,"b":10.5}`,
            expected: `{"a":10,"b":10.5}`,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := CanonicalizeJSON([]byte(tt.input))
            require.NoError(t, err)
            assert.Equal(t, tt.expected, string(result))
        })
    }
}
```

---

## 6. 🌐 API : Endpoint `/api/v1/pos-tickets`

### 6.1 Handler

**Fichier** : `internal/handlers/pos_tickets_handler.go`

```go
package handlers

import (
    "context"
    "time"
    
    "github.com/doreviateam/dorevia-vault/internal/services"
    "github.com/gofiber/fiber/v2"
)

// PosTicketPayload représente le payload JSON pour l'endpoint /api/v1/pos-tickets
type PosTicketPayload struct {
    Tenant       string                 `json:"tenant"`        // Obligatoire
    SourceSystem string                 `json:"source_system"` // Défaut: "odoo_pos"
    SourceModel  string                 `json:"source_model"`  // Obligatoire (ex: "pos.order")
    SourceID     string                 `json:"source_id"`     // Obligatoire
    Currency     *string                `json:"currency,omitempty"`
    TotalInclTax *float64               `json:"total_incl_tax,omitempty"`
    TotalExclTax *float64               `json:"total_excl_tax,omitempty"`
    PosSession   *string                `json:"pos_session,omitempty"`
    Cashier      *string                `json:"cashier,omitempty"`
    Location     *string                `json:"location,omitempty"`
    Ticket       map[string]interface{} `json:"ticket"`        // Obligatoire (JSON brut)
}

// PosTicketResponse représente la réponse standardisée
type PosTicketResponse struct {
    ID          string    `json:"id"`
    Tenant      string    `json:"tenant"`       // Ajouté pour cohérence
    SHA256Hex   string    `json:"sha256_hex"`
    LedgerHash  *string   `json:"ledger_hash,omitempty"`
    EvidenceJWS *string   `json:"evidence_jws,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}

// PosTicketsHandler gère l'endpoint POST /api/v1/pos-tickets
func PosTicketsHandler(
    service *services.PosTicketsService,
    cfg *config.Config,
    log *zerolog.Logger,
) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Validation taille (configurable)
        maxSize := cfg.PosTicketMaxSizeBytes
        if maxSize == 0 {
            maxSize = 64 * 1024 // 64 KB par défaut
        }
        if len(c.Body()) > maxSize {
            return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
                "error": "Payload too large",
                "max_size_bytes": maxSize,
            })
        }
        
        // Parser le payload
        var payload PosTicketPayload
        if err := c.BodyParser(&payload); err != nil {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Invalid JSON payload",
                "details": err.Error(),
            })
        }
        
        // Validation
        if payload.Tenant == "" {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Missing required field: tenant",
            })
        }
        if payload.SourceModel == "" {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Missing required field: source_model",
            })
        }
        if payload.SourceID == "" {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Missing required field: source_id",
            })
        }
        if payload.Ticket == nil {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Missing required field: ticket",
            })
        }
        
        // Valeur par défaut pour source_system
        if payload.SourceSystem == "" {
            payload.SourceSystem = "odoo_pos"
        }
        
        // Mapper handlers.PosTicketPayload → services.PosTicketInput
        input := services.PosTicketInput{
            Tenant:       payload.Tenant,
            SourceSystem: payload.SourceSystem,
            SourceModel:  payload.SourceModel,
            SourceID:     payload.SourceID,
            Currency:     payload.Currency,
            TotalInclTax: payload.TotalInclTax,
            TotalExclTax: payload.TotalExclTax,
            PosSession:   payload.PosSession,
            Cashier:      payload.Cashier,
            Location:     payload.Location,
            Ticket:       payload.Ticket,
        }
        
        // Appeler le service
        ctx := context.Background()
        result, err := service.Ingest(ctx, input)
        if err != nil {
            // Gérer les erreurs selon le type
            // ...
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to ingest POS ticket",
            })
        }
        
        // Retourner la réponse standardisée
        return c.Status(fiber.StatusCreated).JSON(PosTicketResponse{
            ID:          result.ID.String(),
            Tenant:      result.Tenant,
            SHA256Hex:   result.SHA256Hex,
            LedgerHash:  result.LedgerHash,
            EvidenceJWS: result.EvidenceJWS,
            CreatedAt:   result.CreatedAt,
        })
    }
}
```

### 6.2 Service Métier

**Fichier** : `internal/services/pos_tickets_service.go`

```go
package services

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "time"
    
    "github.com/doreviateam/dorevia-vault/internal/crypto"
    "github.com/doreviateam/dorevia-vault/internal/handlers"
    "github.com/doreviateam/dorevia-vault/internal/ledger"
    "github.com/doreviateam/dorevia-vault/internal/models"
    "github.com/doreviateam/dorevia-vault/internal/storage"
    "github.com/doreviateam/dorevia-vault/internal/utils"
    "github.com/google/uuid"
)

type PosTicketsService struct {
    repo   storage.DocumentRepository // Interface, pas *storage.DB
    ledger ledger.Service              // Interface
    signer crypto.Signer
}

func NewPosTicketsService(
    repo storage.DocumentRepository,
    ledger ledger.Service,
    signer crypto.Signer,
) *PosTicketsService {
    return &PosTicketsService{
        repo:   repo,
        ledger: ledger,
        signer: signer,
    }
}

type PosTicketResult struct {
    ID          uuid.UUID
    Tenant      string
    SHA256Hex   string
    LedgerHash  *string
    EvidenceJWS *string
    CreatedAt   time.Time
}

func (s *PosTicketsService) Ingest(ctx context.Context, input PosTicketInput) (*PosTicketResult, error) {
    // 1. Construire le hash input pour idempotence métier stricte (Option A)
    // Hash basé sur ticket + source_id + pos_session (plus stable)
    hashInput := map[string]interface{}{
        "ticket":      input.Ticket,
        "source_id":   input.SourceID,
        "pos_session": input.PosSession,
    }
    
    // 2. Marshal et canonicaliser le hash input
    hashInputBytes, err := json.Marshal(hashInput)
    if err != nil {
        return nil, fmt.Errorf("marshal hash input: %w", err)
    }
    
    canonicalBytes, err := utils.CanonicalizeJSON(hashInputBytes)
    if err != nil {
        return nil, fmt.Errorf("canonicalize JSON: %w", err)
    }
    
    // 3. Calculer SHA256 pour idempotence
    hash := sha256.Sum256(canonicalBytes)
    sha256Hex := hex.EncodeToString(hash[:])
    
    // 4. Vérifier idempotence (par sha256)
    existingDoc, err := s.repo.GetDocumentBySHA256(ctx, sha256Hex)
    if err == nil && existingDoc != nil {
        // Document déjà existant
        return &PosTicketResult{
            ID:          existingDoc.ID,
            Tenant:      input.Tenant,
            SHA256Hex:   existingDoc.SHA256Hex,
            LedgerHash:  existingDoc.LedgerHash,
            EvidenceJWS: existingDoc.EvidenceJWS,
            CreatedAt:   existingDoc.CreatedAt,
        }, nil
    }
    
    // 5. Marshal le payload complet pour stockage
    fullPayload := map[string]interface{}{
        "tenant":        input.Tenant,
        "source_system": input.SourceSystem,
        "source_model":  input.SourceModel,
        "source_id":     input.SourceID,
        "currency":      input.Currency,
        "total_incl_tax": input.TotalInclTax,
        "total_excl_tax": input.TotalExclTax,
        "pos_session":   input.PosSession,
        "cashier":       input.Cashier,
        "location":      input.Location,
        "ticket":        input.Ticket,
    }
    fullPayloadBytes, err := json.Marshal(fullPayload)
    if err != nil {
        return nil, fmt.Errorf("marshal full payload: %w", err)
    }
    fullCanonicalBytes, err := utils.CanonicalizeJSON(fullPayloadBytes)
    if err != nil {
        return nil, fmt.Errorf("canonicalize full payload: %w", err)
    }
    
    // 6. Créer le document
    docID := uuid.New()
    source := "pos"
    now := time.Now()
    
    doc := &models.Document{
        ID:          docID,
        Filename:    fmt.Sprintf("pos-ticket-%s.json", input.SourceID),
        ContentType: "application/json",
        SizeBytes:   int64(len(fullCanonicalBytes)),
        SHA256Hex:   sha256Hex, // Hash pour idempotence (basé sur ticket + source_id + session)
        StoredPath:  "", // Pas de fichier, stockage en DB uniquement
        CreatedAt:   now,
        Source:      &source,
        OdooModel:   &input.SourceModel,
        SourceIDText: &input.SourceID, // Stocker l'ID textuel
        // OdooID reste NULL pour les tickets POS (on utilise source_id_text)
        PayloadJSON: fullCanonicalBytes, // JSON complet canonicalisé pour stockage
        Currency:    input.Currency,
        TotalHT:     input.TotalExclTax,
        TotalTTC:    input.TotalInclTax,
        PosSession:  input.PosSession,
        Cashier:     input.Cashier,
        Location:    input.Location,
    }
    
    // 7. Construire le payload Evidence et signer
    evidencePayload := crypto.EvidencePayload{
        DocumentID: docID.String(),
        Sha256:     sha256Hex,
        Timestamp:  now,
    }
    evidenceBytes, _ := json.Marshal(evidencePayload)
    
    signature, err := s.signer.SignPayload(ctx, evidenceBytes)
    if err != nil {
        return nil, fmt.Errorf("sign evidence: %w", err)
    }
    evidenceJWS := signature.JWS
    
    // 8. Insérer le document avec evidence via le repository
    // Le repository gère la transaction, l'insertion, l'ajout au ledger et la mise à jour
    // Note: L'interface DocumentRepository.InsertDocumentWithEvidence :
    // - Crée une transaction DB
    // - Insère le document
    // - Appelle ledger.Service.Append() dans la transaction
    // - Met à jour le document avec evidence_jws et ledger_hash
    // - Commit la transaction
    err = s.repo.InsertDocumentWithEvidence(ctx, doc, evidenceJWS, s.ledger)
    if err != nil {
        return nil, fmt.Errorf("insert document: %w", err)
    }
    
    // 9. Récupérer le ledger_hash depuis le document (mis à jour par le repository)
    ledgerHash := ""
    if doc.LedgerHash != nil {
        ledgerHash = *doc.LedgerHash
    }
    
    return &PosTicketResult{
        ID:          docID,
        Tenant:      input.Tenant,
        SHA256Hex:   sha256Hex,
        LedgerHash:  &ledgerHash,
        EvidenceJWS: &evidenceJWS,
        CreatedAt:   now,
    }, nil
}
```

---

## 7. 📊 Observabilité

### 7.1 Métriques Prometheus

**Stratégie** : Réutiliser les métriques existantes avec le label `source="pos"`.

**Fichier** : `internal/metrics/prometheus.go` (modifications minimales)

```go
// Aucune nouvelle métrique nécessaire si on réutilise documents_vaulted_total{source="pos"}
// Mais on peut ajouter un label document_type pour plus de granularité (optionnel)

// Optionnel : Ajouter un label document_type
DocumentsVaulted = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "documents_vaulted_total",
        Help: "Nombre total de documents vaultés par statut, source et type",
    },
    []string{"status", "source", "document_type"}, // Ajout document_type
)

// Dans le handler POS :
metrics.RecordDocumentVaulted("success", "pos", "ticket")
```

**Alternative (si on veut des métriques séparées)** :

```go
// Nouvelle métrique POS (si séparation souhaitée)
PosTicketsIngested = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "dorevia_pos_tickets_ingested_total",
        Help: "Nombre total de tickets POS ingérés",
    },
    []string{"tenant", "source"},
)

PosTicketsFailed = promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "dorevia_pos_tickets_failed_total",
        Help: "Nombre total d'échecs d'ingestion POS",
    },
    []string{"reason"}, // validation, ledger, signer, db
)
```

**Recommandation** : Réutiliser `documents_vaulted_total{source="pos"}` pour cohérence.

### 7.2 Logs Structurés

```go
// Dans le handler/service
log.Info().
    Str("tenant", payload.Tenant).
    Str("source_model", payload.SourceModel).
    Str("source_id", payload.SourceID).
    Str("sha256_hex", sha256Hex).
    Str("ledger_hash", ledgerHash).
    Msg("pos ticket ingested")
```

---

## 8. 🧪 Plan de Tests

### 8.1 Tests Unitaires

#### Tests de Canonicalisation JSON
- [ ] `TestCanonicalizeJSON` — Vérifier tri des clés, suppression null, normalisation nombres
- [ ] `TestCanonicalizeJSON_Stability` — Même JSON → même hash

#### Tests du Service POS
- [ ] `TestPosTicketsService_Ingest_Success` — Ingestion réussie
- [ ] `TestPosTicketsService_Ingest_Idempotence` — Même payload → même document
- [ ] `TestPosTicketsService_Ingest_LedgerError` — Gestion erreur ledger
- [ ] `TestPosTicketsService_Ingest_SignerError` — Gestion erreur signer

#### Tests du Handler
- [ ] `TestPosTicketsHandler_Success` — 201 Created
- [ ] `TestPosTicketsHandler_InvalidJSON` — 400 Bad Request
- [ ] `TestPosTicketsHandler_MissingFields` — 400 Bad Request
- [ ] `TestPosTicketsHandler_PayloadTooLarge` — 413 Request Entity Too Large
- [ ] `TestPosTicketsHandler_ServiceError` — 500 Internal Server Error

### 8.2 Tests d'Intégration

- [ ] `TestPosTickets_EndToEnd` — Appel HTTP complet, vérification DB, ledger, JWS
- [ ] `TestPosTickets_Idempotence` — Deux appels identiques → même résultat
- [ ] `TestPosTickets_Metrics` — Vérification métriques Prometheus

---

## 9. 📝 Checklist d'Implémentation

> **Note** : Les phases doivent être réalisées **dans l'ordre croissant** (Phase 1 → Phase 2 → ... → Phase 7), sans chevauchement.

### 🎯 Approche "Mini Sprints"

Chaque phase constitue un **mini sprint d'un jour** avec :
- ✅ **Objectif clair** : Livrable spécifique et mesurable
- ✅ **Critères de fin** : Validation avant passage à la phase suivante
- ✅ **Prérequis explicites** : Dépendances clairement identifiées
- ✅ **Livraison incrémentale** : Progression visible jour après jour

**Avantages** :
- 🎯 Visibilité quotidienne sur l'avancement
- 🔄 Feedback rapide et ajustements possibles
- ⚠️ Détection précoce des risques
- 📊 Progression mesurable et motivation renforcée

**Rituel recommandé** : Validation en fin de journée de chaque phase avant de passer à la suivante.

---

## ⚠️ Phase 0 : Préparation Architecturale (Avant Phase 1)

> **Important** : Cette phase doit être complétée **avant** de commencer la Phase 1. Elle corrige les incohérences architecturales identifiées par l'avis d'architecte.

### Phase 0 : Interfaces & Types (Jour 0 - Préparation)

**Objectif** : Créer les interfaces et types nécessaires pour une architecture propre et testable.

- [ ] **Interface `DocumentRepository`**
  - [ ] Créer `internal/storage/repository.go` avec interface `DocumentRepository`
    ```go
    type DocumentRepository interface {
        GetDocumentBySHA256(ctx context.Context, sha256 string) (*models.Document, error)
        InsertDocumentWithEvidence(
            ctx context.Context,
            doc *models.Document,
            evidenceJWS string,
            ledgerService ledger.Service, // Service ledger pour ajout dans transaction
        ) error
    }
    ```
  - [ ] Créer `internal/storage/postgres_repository.go` avec implémentation
    - [ ] L'implémentation doit gérer la transaction, l'insertion, l'ajout au ledger et la mise à jour
  - [ ] **Tests** : Tests unitaires de l'implémentation (avec mock de ledger.Service)

- [ ] **Interface `ledger.Service`**
  - [ ] Créer `internal/ledger/service.go` avec interface `Service`
    ```go
    type Service interface {
        Append(ctx context.Context, tx pgx.Tx, docID uuid.UUID, shaHex, jws string) (string, error)
        ExistsByDocumentID(ctx context.Context, tx pgx.Tx, docID uuid.UUID) (bool, error)
    }
    ```
  - [ ] Créer `internal/ledger/service_impl.go` avec implémentation `DefaultService`
  - [ ] **Tests** : Tests unitaires de l'implémentation

- [ ] **Types Services (sans dépendance handlers)**
  - [ ] Créer `internal/services/pos_tickets_types.go` avec type `PosTicketInput`
    ```go
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
        Ticket       map[string]interface{}
    }
    ```

- [ ] **Documentation Stratégie d'Idempotence**
  - [ ] Ajouter section dans `docs/API.md` ou `docs/POS_TICKETS_API.md`
  - [ ] Documenter : **Option A (Idempotence métier stricte)** choisie
    - Hash basé sur `ticket + source_id + pos_session` (plus stable)
    - Métadonnées optionnelles (`cashier`, `location`) n'affectent pas l'idempotence

- [ ] **Review Code d'Exemple**
  - [ ] Vérifier tous les imports Go (`import (` pas `import {`)
  - [ ] Vérifier struct tags (`json:"id"` pas `"json:"id"`)
  - [ ] Vérifier présence de `fmt`, `context`, etc. dans imports

**✅ Critère de fin de Phase 0** : Toutes les interfaces créées, implémentées, testées. Types services créés. Documentation idempotence ajoutée. Code d'exemple corrigé.

---

### Phase 1 : Préparation (Jour 1)

- [ ] **Migration DB**
  - [ ] Créer `migrations/005_add_pos_fields.sql`
  - [ ] Tester la migration (up/down)
  - [ ] Ajouter migration dans `internal/storage/postgres.go`
  - [ ] **Tests** : Vérifier création des colonnes, index, contraintes

- [ ] **Modèle de Données**
  - [ ] Ajouter champs POS dans `internal/models/document.go`
  - [ ] Mettre à jour les tags DB
  - [ ] **Tests** : Vérifier sérialisation/désérialisation JSON et DB

- [ ] **Canonicalisation JSON**
  - [ ] Implémenter `internal/utils/json_canonical.go`
  - [ ] **Tests unitaires** : `internal/utils/json_canonical_test.go`
    - [ ] `TestCanonicalizeJSON` — Tri des clés, suppression null, normalisation nombres
    - [ ] `TestCanonicalizeJSON_Stability` — Même JSON → même hash
    - [ ] `TestCanonicalizeJSON_EdgeCases` — Tableaux, objets imbriqués, valeurs nulles
  - [ ] Valider stabilité du hash

**✅ Critère de fin de Phase 1** : Migration testée, modèle mis à jour, canonicalisation JSON fonctionnelle avec **tous les tests verts**.

---

### Phase 2 : Abstraction Crypto (Jour 2)

**Prérequis** : Phase 1 terminée

- [ ] **Interface Signer**
  - [ ] Créer `internal/crypto/signer.go`
  - [ ] Créer `internal/crypto/local_signer.go` (adaptateur)
  - [ ] **Tests unitaires** : `internal/crypto/local_signer_test.go`
    - [ ] `TestLocalSigner_SignPayload` — Signature réussie
    - [ ] `TestLocalSigner_SignPayload_InvalidPayload` — Gestion erreur payload invalide
    - [ ] `TestLocalSigner_KeyID` — Retourne le KID correct
    - [ ] `TestLocalSigner_IntegrationWithJWSService` — Intégration avec crypto.Service

**✅ Critère de fin de Phase 2** : Interface `Signer` implémentée, adaptateur `LocalSigner` fonctionnel avec **tous les tests verts**.

---

### Phase 3 : Service Métier (Jour 3)

**Prérequis** : Phase 2 terminée

- [ ] **Service POS**
  - [ ] Créer `internal/services/pos_tickets_service.go`
  - [ ] Utiliser `storage.DocumentRepository` (interface) au lieu de `*storage.DB`
  - [ ] Utiliser `ledger.Service` (interface) avec `s.ledger.Append()`
  - [ ] Utiliser `PosTicketInput` (type service) au lieu de `handlers.PosTicketPayload`
  - [ ] Implémenter **idempotence métier stricte** (Option A) : hash basé sur `ticket + source_id + pos_session`
  - [ ] Intégration ledger + signer
  - [ ] **Tests unitaires** : `internal/services/pos_tickets_service_test.go`
    - [ ] `TestPosTicketsService_Ingest_Success` — Ingestion réussie
    - [ ] `TestPosTicketsService_Ingest_Idempotence` — Même ticket + source_id + session → même document
    - [ ] `TestPosTicketsService_Ingest_Idempotence_MetadataChange` — Changement métadonnées (cashier) → même document
    - [ ] `TestPosTicketsService_Ingest_LedgerError` — Gestion erreur ledger
    - [ ] `TestPosTicketsService_Ingest_SignerError` — Gestion erreur signer
    - [ ] `TestPosTicketsService_Ingest_RepositoryError` — Gestion erreur repository
    - [ ] `TestPosTicketsService_Canonicalization` — Vérifier canonicalisation JSON

**✅ Critère de fin de Phase 3** : Service POS fonctionnel, intégration ledger + signer validée, **tous les tests unitaires verts**.

---

### Phase 4 : Handler API (Jour 4)

**Prérequis** : Phase 3 terminée

- [ ] **Handler POS**
  - [ ] Créer `internal/handlers/pos_tickets_handler.go`
  - [ ] **Mapping** : `handlers.PosTicketPayload` → `services.PosTicketInput` (pas de dépendance inverse)
  - [ ] Validation payload
  - [ ] Gestion erreurs
  - [ ] **Tests unitaires** : `internal/handlers/pos_tickets_handler_test.go`
    - [ ] `TestPosTicketsHandler_Success` — 201 Created avec payload valide
    - [ ] `TestPosTicketsHandler_InvalidJSON` — 400 Bad Request (JSON invalide)
    - [ ] `TestPosTicketsHandler_MissingFields` — 400 Bad Request (champs manquants)
    - [ ] `TestPosTicketsHandler_PayloadTooLarge` — 413 Request Entity Too Large
    - [ ] `TestPosTicketsHandler_ServiceError` — 500 Internal Server Error
    - [ ] `TestPosTicketsHandler_AuthRequired` — 401 Unauthorized (si auth activé)
    - [ ] `TestPosTicketsHandler_Mapping` — Vérifier mapping Payload → Input

- [ ] **Route**
  - [ ] Enregistrer route dans `cmd/vault/main.go`
  - [ ] Ajouter middleware auth (si activé)

**✅ Critère de fin de Phase 4** : Endpoint `/api/v1/pos-tickets` accessible, validation complète, **tous les tests unitaires verts**.

---

### Phase 5 : Observabilité (Jour 5)

**Prérequis** : Phase 4 terminée

- [ ] **Métriques**
  - [ ] Intégrer métriques dans handler/service
  - [ ] Vérifier exposition Prometheus
  - [ ] **Tests** : Vérifier incrémentation des compteurs, histogrammes

- [ ] **Logs**
  - [ ] Ajouter logs structurés
  - [ ] Vérifier format JSON
  - [ ] **Tests** : Vérifier présence des champs dans les logs

**✅ Critère de fin de Phase 5** : Métriques exposées sur `/metrics`, logs structurés fonctionnels, **tests d'observabilité validés**.

---

### Phase 6 : Tests d'Intégration & Documentation (Jour 6)

**Prérequis** : Phase 5 terminée

- [ ] **Tests d'Intégration** : `tests/integration/pos_tickets_test.go`
  - [ ] `TestPosTickets_EndToEnd` — Appel HTTP complet, vérification DB, ledger, JWS
  - [ ] `TestPosTickets_Idempotence` — Deux appels identiques → même résultat
  - [ ] `TestPosTickets_Metrics` — Vérification métriques Prometheus
  - [ ] `TestPosTickets_LedgerIntegration` — Vérification entrée ledger
  - [ ] `TestPosTickets_JWSIntegration` — Vérification signature JWS
  - [ ] Tests de performance (optionnel) — Benchmark canonicalisation JSON

- [ ] **Documentation**
  - [ ] Mettre à jour `README.md`
  - [ ] Documenter endpoint dans `docs/API.md` ou créer `docs/POS_TICKETS_API.md`
  - [ ] **Documenter canonicalisation JSON** : Algorithme (tri clés, suppression null, normalisation nombres)
  - [ ] **Documenter stratégie d'idempotence** : Option A (métier stricte) - hash basé sur `ticket + source_id + pos_session`
  - [ ] Ajouter exemples de payloads
  - [ ] Documenter variables d'environnement POS

**✅ Critère de fin de Phase 6** : **Tous les tests d'intégration verts**, documentation complète.

---

### Phase 7 : Validation & Déploiement (Jour 7)

**Prérequis** : Phase 6 terminée

- [ ] **Validation**
  - [ ] Review code (couverture de tests >80%)
  - [ ] **Tests complets** : Exécuter tous les tests (unitaires + intégration)
    - [ ] Tests unitaires : `go test ./internal/... -v`
    - [ ] Tests d'intégration : `go test ./tests/integration/... -v`
    - [ ] Couverture : `go test ./... -cover`
  - [ ] Validation avec connecteur Odoo (si disponible)
  - [ ] Tests de non-régression (vérifier endpoints existants)

- [ ] **Déploiement**
  - [ ] Tag Git `v1.4.0`
  - [ ] Changelog
  - [ ] Release notes

**✅ Critère de fin de Phase 7** : Code validé, **tous les tests verts avec couverture >80%**, tag `v1.4.0` créé, release notes publiées.

---

**Durée totale estimée : 8 jours** (Phase 0 + 7 phases, une phase par jour, en ordre séquentiel)

> **Note** : La Phase 0 peut être réalisée en parallèle de la finalisation du plan, ou le Jour 0 avant de commencer la Phase 1.

---

## 10. 🔧 Configuration

### 10.1 Variables d'Environnement

**Fichier** : `internal/config/config.go`

```go
type Config struct {
    // ... champs existants ...
    
    // Configuration POS (Sprint 6)
    PosTicketMaxSizeBytes int `env:"POS_TICKET_MAX_SIZE_BYTES" envDefault:"65536"` // 64 KB
}
```

---

## 11. 📌 Points d'Attention

### 11.1 Stockage des Tickets POS

**Question** : Stocker le JSON en DB uniquement (`payload_json`) ou aussi en fichier ?

**Recommandation** :
- **Option A (Recommandée)** : Stockage DB uniquement (`payload_json JSONB`)
  - Avantages : Pas de gestion de fichiers, recherche JSON native, cohérent avec le modèle
  - Inconvénient : Taille DB plus importante
- **Option B** : Stockage fichier + référence en DB
  - Avantages : DB plus légère
  - Inconvénients : Gestion fichiers, moins de recherche native

**Décision** : Option A (stockage DB uniquement) pour Sprint 6.

### 11.2 Gestion de `source_id` (String vs Int)

**Problème** : `source_id` peut être un string (`"POS/2025/0001"`) mais `odoo_id` dans `documents` est `INTEGER`.

**Solution Retenue** : Ajouter un champ `source_id_text TEXT` dans `documents` (voir migration 005).

**Implémentation** :
- Pour les tickets POS : `source_id_text` contient l'ID textuel, `odoo_id` reste `NULL`
- Pour les factures : `odoo_id` contient l'ID numérique, `source_id_text` reste `NULL`
- Index partiel sur `source_id_text` pour recherche rapide POS

---

## 12. ✅ Critères de Finition (Definition of Done)

Pour considérer le Sprint 6 comme **terminé** :

1. ✅ Endpoint `POST /api/v1/pos-tickets` disponible et fonctionnel
2. ✅ Champs POS ajoutés à `documents` via migration
3. ✅ Canonicalisation JSON implémentée et testée
4. ✅ Interface `Signer` implémentée avec adaptateur local
5. ✅ Intégration complète avec le ledger & signer
6. ✅ Réponse API standardisée (id, tenant, sha256_hex, ledger_hash, evidence_jws, created_at)
7. ✅ Métriques Prometheus intégrées (réutilisation ou nouvelles)
8. ✅ Tests unitaires & d'intégration verts (>80% couverture)
9. ✅ Documentation mise à jour (`README`, `CHANGELOG`)
10. ✅ Tag `v1.4.0` poussé et déployé

---

**Auteur** : Plan d'implémentation Dorevia Vault  
**Date** : 2025-01-14  
**Version** : 1.0 (Corrigée)

