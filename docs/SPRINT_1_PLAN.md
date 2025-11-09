# Dorevia Vault — Sprint 1 (Version resserrée)

**Objectif Sprint 1** : obtenir un **MVP "Validé → Vaulté"** connecté à Odoo **sans** JWS, **sans** Ledger, **sans** validation Factur‑X. Focus sur **cohérence transactionnelle**, **ingestion fiable** et **idempotence**.

**Durée cible** : 10–14 jours  
**Version** : 1.0  
**Date** : Janvier 2025

---

## 0) Portée & exclusions

### Inclus (à livrer)

- Migration SQL **`documents`** (métadonnées Odoo + index utiles).  
- **Transaction atomique** fichier ↔ DB (pattern outbox minimal).  
- Endpoint **`POST /api/v1/invoices`** (JSON + base64, et multipart en bonus si temps).  
- **Idempotence** par hash SHA‑256.  
- **Mini‑monitoring** : compteur Prometheus `documents_vaulted_total` + logs structurés.  
- Script/mock d'envoi depuis Odoo (**cURL** ou petit helper Go).

### Exclus (report Sprint 2+)

- ✅ **Validation Factur‑X** (parsing XML, XSD/EN16931).  
- ✅ **Scellement JWS** + `/jwks.json` + rotation de clés.  
- ✅ **Ledger hash‑chaîné** (table, partitions, export).  
- ✅ **Webhooks asynchrones + queue/Redis** (remplacé par test manuel).  
- ✅ **Monitoring avancé** (traces, dashboards complets).

---

## 1) Arborescence cible

```
dorevia-vault/
├─ cmd/
│  └─ vault/
│     └─ main.go                  # serve + migrate (existant)
├─ internal/
│  ├─ models/
│  │  └─ document.go              # struct enrichie
│  ├─ storage/
│  │  └─ postgres.go              # pool, migrations, tx atomique
│  ├─ handlers/
│  │  └─ invoices.go              # POST /api/v1/invoices
│  ├─ hashing/
│  │  └─ sha256.go                # util de hash (optionnel)
│  ├─ odoo/
│  │  └─ mock_push.go             # helper de test (optionnel)
│  └─ metrics/
│     └─ prometheus.go            # registre + compteur
├─ migrations/
│  └─ 003_add_odoo_fields.sql     # migration Sprint 1
├─ docs/
│  └─ sprint_1_plan.md            # ce document
└─ Makefile                       # commandes utiles
```

---

## 2) Checklist détaillée

### 2.1 Extension modèle Document

**Fichier** : `internal/models/document.go`

**Champs à ajouter** :
```go
type Document struct {
    // ... champs existants (ID, Filename, ContentType, SizeBytes, SHA256Hex, StoredPath, CreatedAt) ...
    
    // Métadonnées Odoo
    Source      *string    `json:"source,omitempty"`       // sales|purchase|pos|stock|sale
    OdooModel   *string    `json:"odoo_model,omitempty"`   // account.move, pos.order, etc.
    OdooID      *int       `json:"odoo_id,omitempty"`      // ID dans Odoo
    OdooState   *string    `json:"odoo_state,omitempty"`   // posted, paid, done, etc.
    
    // Routage PDP (pour Sprint 2+)
    PDPRequired *bool      `json:"pdp_required,omitempty"` // Nécessite dispatch PDP ?
    DispatchStatus *string `json:"dispatch_status,omitempty"` // PENDING|SENT|ACK|REJECTED
    
    // Métadonnées facture (pour Sprint 2+ validation Factur-X)
    InvoiceNumber *string  `json:"invoice_number,omitempty"`
    InvoiceDate   *time.Time `json:"invoice_date,omitempty"`
    TotalHT       *float64   `json:"total_ht,omitempty"`
    TotalTTC      *float64   `json:"total_ttc,omitempty"`
    Currency      *string    `json:"currency,omitempty"`
    SellerVAT     *string    `json:"seller_vat,omitempty"`
    BuyerVAT      *string    `json:"buyer_vat,omitempty"`
}
```

**Note** : Utiliser des pointeurs (`*string`, `*int`) pour permettre `NULL` en DB et `omitempty` en JSON.

### 2.2 Migration SQL

**Fichier** : `migrations/003_add_odoo_fields.sql`

**Migration** :
```sql
-- Métadonnées Odoo
ALTER TABLE documents ADD COLUMN IF NOT EXISTS source TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_model TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_id INTEGER;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_state TEXT;

-- Routage PDP (préparation Sprint 2)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS pdp_required BOOLEAN DEFAULT false;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS dispatch_status TEXT DEFAULT 'PENDING';

-- Métadonnées facture (préparation Sprint 2)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS invoice_number TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS invoice_date DATE;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS total_ht DECIMAL(10,2);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS total_ttc DECIMAL(10,2);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS currency TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS seller_vat TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS buyer_vat TEXT;

-- Index pour recherche rapide
CREATE INDEX IF NOT EXISTS idx_documents_odoo_id ON documents(odoo_id);
CREATE INDEX IF NOT EXISTS idx_documents_dispatch_status ON documents(dispatch_status);
CREATE INDEX IF NOT EXISTS idx_documents_source ON documents(source);

-- Contrainte sur dispatch_status
ALTER TABLE documents ADD CONSTRAINT chk_dispatch_status 
  CHECK (dispatch_status IN ('PENDING', 'SENT', 'ACK', 'REJECTED'));
```

**Intégration** : Modifier `internal/storage/postgres.go` pour exécuter cette migration.

### 2.3 Transaction atomique

**Fichier** : `internal/storage/postgres.go`

**Pattern** : Transaction Outbox minimal

```go
// Pseudo-code
func (db *DB) StoreDocumentWithTransaction(ctx context.Context, doc *models.Document, content []byte, storageDir string) error {
    // 1. Calculer hash avant transaction
    hash := sha256.Sum256(content)
    sha256Hex := hex.EncodeToString(hash[:])
    
    // 2. Vérifier idempotence (SELECT avant transaction)
    var existingID uuid.UUID
    err := db.Pool.QueryRow(ctx, "SELECT id FROM documents WHERE sha256_hex = $1", sha256Hex).Scan(&existingID)
    if err == nil {
        return ErrDocumentExists{ID: existingID} // Document déjà existant
    }
    if err != pgx.ErrNoRows {
        return err
    }
    
    // 3. Générer UUID et chemin
    docID := uuid.New()
    storedPath := generateStoragePath(storageDir, docID, doc.Filename)
    
    // 4. BEGIN transaction
    tx, err := db.Pool.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    // 5. Stocker fichier sur disque
    if err := os.WriteFile(storedPath, content, 0644); err != nil {
        return fmt.Errorf("failed to save file: %w", err)
    }
    
    // 6. INSERT dans documents
    _, err = tx.Exec(ctx, `
        INSERT INTO documents (id, filename, content_type, size_bytes, sha256_hex, stored_path, 
                              source, odoo_model, odoo_id, odoo_state, pdp_required, dispatch_status,
                              invoice_number, invoice_date, total_ht, total_ttc, currency, seller_vat, buyer_vat)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
    `, docID, doc.Filename, doc.ContentType, doc.SizeBytes, sha256Hex, storedPath,
       doc.Source, doc.OdooModel, doc.OdooID, doc.OdooState, doc.PDPRequired, doc.DispatchStatus,
       doc.InvoiceNumber, doc.InvoiceDate, doc.TotalHT, doc.TotalTTC, doc.Currency, doc.SellerVAT, doc.BuyerVAT)
    
    if err != nil {
        // Nettoyage fichier en cas d'erreur
        os.Remove(storedPath)
        return fmt.Errorf("failed to insert document: %w", err)
    }
    
    // 7. COMMIT
    if err := tx.Commit(ctx); err != nil {
        os.Remove(storedPath) // Nettoyage
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    doc.ID = docID
    doc.SHA256Hex = sha256Hex
    doc.StoredPath = storedPath
    
    return nil
}
```

### 2.4 Endpoint `/api/v1/invoices`

**Fichier** : `internal/handlers/invoices.go`

**Fonctionnalités** :
- Accepte JSON avec base64 (priorité)
- Accepte multipart/form-data (bonus)
- Validation payload
- Idempotence par SHA256
- Transaction atomique
- Retourne `{id, sha256_hex, created_at}`

**Structure payload JSON** :
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

**Réponse succès (201)** :
```json
{
  "id": "uuid",
  "sha256_hex": "abc123...",
  "created_at": "2025-01-15T10:30:00Z"
}
```

**Réponse idempotence (200)** :
```json
{
  "id": "uuid-existant",
  "sha256_hex": "abc123...",
  "created_at": "2025-01-15T09:00:00Z",
  "message": "Document already exists"
}
```

### 2.5 Mini-monitoring Prometheus

**Fichier** : `internal/metrics/prometheus.go`

**Métriques** :
- `documents_vaulted_total` (counter) — nombre de documents vaultés
- `vault_errors_total` (counter, label: `type`) — erreurs par type
- `vault_duration_seconds` (histogram) — durée d'ingestion

**Endpoint** : `/metrics` (standard Prometheus)

### 2.6 Helper de test Odoo

**Fichier** : `internal/odoo/mock_push.go` (optionnel)

**Fonction** : Helper pour tester l'envoi depuis Odoo

```go
func MockPushInvoice(vaultURL string, invoiceData InvoicePayload) error {
    // Envoie POST /api/v1/invoices avec payload JSON
}
```

**Alternative** : Script shell `scripts/test_odoo_push.sh` avec cURL.

---

## 3) Tests à implémenter

### 3.1 Tests unitaires

- [ ] Test extension modèle Document
- [ ] Test migration SQL
- [ ] Test transaction atomique (rollback, cohérence)
- [ ] Test endpoint `/api/v1/invoices` (JSON base64)
- [ ] Test idempotence (doublons)
- [ ] Test validation payload (champs manquants)

### 3.2 Tests d'intégration

- [ ] Test end-to-end : JSON → DB → fichier
- [ ] Test transaction rollback (simuler erreur DB)
- [ ] Test nettoyage fichiers orphelins

---

## 4) Dépendances à ajouter

```bash
# Prometheus
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
```

---

## 5) Variables d'environnement

```bash
# Existant
DATABASE_URL=postgres://...
STORAGE_DIR=/opt/dorevia-vault/storage
PORT=8080
LOG_LEVEL=info

# Nouveau (optionnel)
PROMETHEUS_ENABLED=true
```

---

## 6) Ordre d'exécution

1. **Jour 1-2** : Extension modèle Document + migration SQL
2. **Jour 3-4** : Transaction atomique dans `storage/postgres.go`
3. **Jour 5-7** : Endpoint `/api/v1/invoices` (JSON base64)
4. **Jour 8-9** : Idempotence + validation
5. **Jour 10-11** : Mini-monitoring Prometheus
6. **Jour 12-14** : Tests + helper Odoo + documentation

---

## 7) Critères d'acceptation

### Fonctionnels

- ✅ Endpoint `/api/v1/invoices` accepte JSON base64
- ✅ Document stocké en DB avec métadonnées Odoo
- ✅ Fichier stocké sur disque
- ✅ Transaction atomique (tout ou rien)
- ✅ Idempotence par SHA256 (retour 200 si doublon)
- ✅ Métriques Prometheus disponibles

### Techniques

- ✅ Pas de fichiers orphelins (cohérence fichier/DB)
- ✅ Rollback automatique en cas d'erreur
- ✅ Logs structurés pour debugging
- ✅ Tests unitaires > 80% coverage

---

## 8) Livrables Sprint 1

1. **Code** :
   - Modèle Document enrichi
   - Migration SQL
   - Endpoint `/api/v1/invoices`
   - Transaction atomique
   - Mini-monitoring

2. **Tests** :
   - Tests unitaires
   - Tests d'intégration
   - Helper de test Odoo

3. **Documentation** :
   - API `/api/v1/invoices` (exemples)
   - Guide d'intégration Odoo (basique)

---

## 9) Notes importantes

### Transactions

- **Critique** : Toutes les opérations critiques dans une transaction PostgreSQL
- **Nettoyage** : Supprimer fichier si échec DB
- **Idempotence** : Vérifier avant transaction (SELECT rapide)

### Performance

- **Hash SHA256** : Calculer avant transaction (pas dans transaction)
- **Index** : Créer index sur `odoo_id`, `sha256_hex`, `dispatch_status`
- **Concurrence** : Gérer les race conditions (SELECT FOR UPDATE si nécessaire)

### Sécurité

- **Validation** : Valider tous les champs du payload
- **Base64** : Vérifier format valide
- **Taille** : Limiter taille fichier (ex: 10MB max)

---

**Document créé le** : Janvier 2025  
**Version** : 1.0  
**Prochain sprint** : Sprint 2 (JWS + Ledger)

