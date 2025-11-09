# ✅ CHECKLIST TECHNIQUE — PHASE 3 (VERSION AMÉLIORÉE)
## Dorevia Vault — Passage Phase 2 → Phase 3

**Objectif** : Implémenter la règle des 3V (Validé → Vaulté → Vérifiable) et l'intégration automatique Odoo → Vault.

**Version** : 2.0 (enrichie avec recommandations expert)  
**Date** : Janvier 2025  
**Basé sur** : `demarche_revision_concept.md` + `AVIS_EXPERT_PHASE3.md`

---

## 🎯 Approche recommandée

**Principe** : **MVP d'abord, raffinement ensuite**

- **Sprint 1** (2 semaines) : MVP fonctionnel (étapes 1-3) — **Sans JWS ni ledger**
- **Sprint 2** (2 semaines) : Scellement et traçabilité (étapes 4-5) — **JWS + Ledger**
- **Sprint 3** (2 semaines) : Production-ready (étapes 6-7) — **Webhooks + Monitoring**

---

## 🧱 1. Étendre le modèle & la base de données

**Priorité** : 🔴 Haute  
**Durée estimée** : 1-2 jours

### Modèle Document

- [ ] Ajouter les champs suivants dans `internal/models/document.go` :
  - **Métadonnées Odoo** : `source`, `odoo_model`, `odoo_id`, `odoo_state`
  - **Routage PDP** : `pdp_required`, `dispatch_status`, `pdp_message_id`
  - **Preuves d'intégrité** : `evidence_jws`, `ledger_hash`
  - **Métadonnées facture** : `invoice_number`, `invoice_date`, `total_ht`, `total_ttc`, `currency`, `seller_vat`, `buyer_vat`

### Migration SQL

- [ ] Ajouter la migration SQL dans `internal/storage/postgres.go`
- [ ] Utiliser `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` pour compatibilité
- [ ] **Index sur `odoo_id`** pour recherche rapide
- [ ] **Index sur `sha256_hex`** (déjà présent mais vérifier)
- [ ] **Index sur `dispatch_status`** pour filtrage
- [ ] **Contraintes d'intégrité** (CHECK sur `dispatch_status`, `source`)

### Exemple de migration

```sql
-- Métadonnées Odoo
ALTER TABLE documents ADD COLUMN IF NOT EXISTS source TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_model TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_id INTEGER;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS odoo_state TEXT;

-- Routage PDP
ALTER TABLE documents ADD COLUMN IF NOT EXISTS pdp_required BOOLEAN DEFAULT false;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS dispatch_status TEXT DEFAULT 'PENDING';
ALTER TABLE documents ADD COLUMN IF NOT EXISTS pdp_message_id TEXT;

-- Preuves
ALTER TABLE documents ADD COLUMN IF NOT EXISTS evidence_jws TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS ledger_hash TEXT;

-- Métadonnées facture
ALTER TABLE documents ADD COLUMN IF NOT EXISTS invoice_number TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS invoice_date DATE;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS total_ht DECIMAL(10,2);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS total_ttc DECIMAL(10,2);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS currency TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS seller_vat TEXT;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS buyer_vat TEXT;

-- Index
CREATE INDEX IF NOT EXISTS idx_documents_odoo_id ON documents(odoo_id);
CREATE INDEX IF NOT EXISTS idx_documents_dispatch_status ON documents(dispatch_status);
CREATE INDEX IF NOT EXISTS idx_documents_source ON documents(source);

-- Contraintes
ALTER TABLE documents ADD CONSTRAINT chk_dispatch_status 
  CHECK (dispatch_status IN ('PENDING', 'SENT', 'ACK', 'REJECTED'));
```

### Tests

- [ ] Tests unitaires de migration (rollback, compatibilité)
- [ ] Tests de cohérence des index
- [ ] Vérification des contraintes

---

## 🔒 1bis. Gestion des transactions atomiques ⚠️ CRITIQUE

**Priorité** : 🔴 Haute  
**Durée estimée** : 1 jour

**Objectif** : Garantir la cohérence entre fichier, DB et ledger.

### Pattern Transaction Outbox

- [ ] Implémenter transactions PostgreSQL pour opérations critiques
- [ ] **Ordre d'exécution dans transaction** :
  1. Stocker fichier sur disque
  2. BEGIN transaction
  3. INSERT dans `documents`
  4. Générer JWS (hors transaction si long)
  5. Calculer hash ledger
  6. INSERT dans `ledger` (si étape 5 activée)
  7. COMMIT (tout ou rien)
- [ ] **Rollback automatique** en cas d'erreur
- [ ] **Nettoyage des fichiers orphelins** si échec DB
- [ ] **Tests de cohérence** :
  - Fichier sans DB → détecter et nettoyer
  - DB sans fichier → détecter et signaler
  - Ledger sans document → détecter et corriger

### Exemple de code

```go
// Pseudo-code
tx, err := db.Pool.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx)

// 1. Stocker fichier
storedPath := saveFile(content)

// 2. INSERT documents
_, err = tx.Exec(ctx, "INSERT INTO documents ...", ...)
if err != nil {
    os.Remove(storedPath) // Nettoyage
    return err
}

// 3. INSERT ledger (si activé)
if ledgerEnabled {
    _, err = tx.Exec(ctx, "INSERT INTO ledger ...", ...)
    if err != nil {
        return err
    }
}

// 4. COMMIT
if err := tx.Commit(ctx); err != nil {
    os.Remove(storedPath) // Nettoyage
    return err
}
```

### Tests

- [ ] Test rollback (simuler erreur DB)
- [ ] Test cohérence fichier/DB
- [ ] Test nettoyage fichiers orphelins

---

## 🔌 2. Créer l'endpoint d'ingestion Odoo

**Priorité** : 🔴 Haute  
**Durée estimée** : 2-3 jours

### Handler et endpoint

- [ ] Nouveau handler `internal/handlers/invoices.go`
- [ ] Endpoint : `POST /api/v1/invoices`
- [ ] **Accepte JSON (base64) ET multipart/form-data**
- [ ] Validation complète du payload :
  - Champs obligatoires (`source`, `model`, `odoo_id`, `file`)
  - Format base64 valide (si JSON)
  - Métadonnées facture (si présentes)
- [ ] Calcul SHA256 du fichier
- [ ] **Idempotence** : si même hash, retourne le document existant (200 OK)
- [ ] Stockage fichier + métadonnées enrichies
- [ ] **Transaction atomique** (fichier + DB)
- [ ] Retourne `{id, sha256_hex, evidence_jws?, ledger_hash?}`

### Structure du payload

**Format JSON (base64)** :
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

**Format multipart** (alternative) :
```
POST /api/v1/invoices
Content-Type: multipart/form-data

source=sales
model=account.move
odoo_id=12345
state=posted
pdp_required=true
file=<fichier PDF>
meta={"number":"F2025-00123",...}
```

### Gestion d'erreurs

- [ ] Erreurs de validation (400 Bad Request)
- [ ] Erreurs de stockage (500 Internal Server Error)
- [ ] Erreurs de transaction (rollback + nettoyage)
- [ ] Messages d'erreur détaillés et structurés

### Tests

- [ ] Tests unitaires pour `/api/v1/invoices`
- [ ] Tests idempotence (doublons avec même hash)
- [ ] Tests validation payload (champs manquants, formats invalides)
- [ ] Tests transaction (rollback en cas d'erreur)
- [ ] Tests multipart vs JSON base64

---

## 🔍 2bis. Validation et extraction Factur-X (Optionnel MVP)

**Priorité** : 🟡 Moyenne  
**Durée estimée** : 2-3 jours  
**Note** : Peut être reporté après MVP

### Extraction XML

- [ ] Extraction XML depuis PDF Factur-X
- [ ] Parsing basique (structure, champs obligatoires)
- [ ] Extraction automatique des métadonnées :
  - `invoice_number`
  - `invoice_date`
  - `total_ht`, `total_ttc`
  - `currency`
  - `seller_vat`, `buyer_vat`
- [ ] **Mode strict/lenient** selon environnement

### Validation

- [ ] Validation structure XML (sans XSD pour MVP)
- [ ] Validation basique des champs obligatoires
- [ ] Gestion des erreurs de validation (retour détaillé)
- [ ] **Validation XSD EN16931** (niveau 2, après MVP)

### Tests

- [ ] Tests extraction XML
- [ ] Tests parsing métadonnées
- [ ] Tests validation (cas valides/invalides)

---

## 🔄 3. Relier Odoo (déclencheur "validé")

**Priorité** : 🟡 Moyenne (après validation endpoint)  
**Durée estimée** : 1-2 jours

**Note** : Cette étape doit être faite **après** que l'endpoint `/api/v1/invoices` soit testé et validé.

### Configuration Odoo

- [ ] Odoo : configurer webhook sur `account.move (state='posted')`
- [ ] Odoo : configurer webhook sur `pos.order (state='paid'|'done')`
- [ ] Odoo : envoie `POST /api/v1/invoices` avec payload standardisé
- [ ] **Gestion des erreurs côté Odoo** (retry, logging)

### Documentation intégration

- [ ] Documentation pour configurer Odoo
- [ ] Exemples de payload
- [ ] Gestion des erreurs et retry
- [ ] Tests d'intégration (ou mock Odoo)

### Tests

- [ ] Tests avec mock Odoo
- [ ] Tests de payloads réels
- [ ] Tests de gestion d'erreurs

---

## 🔐 4. Ajouter le scellement (JWS)

**Priorité** : 🟠 Moyenne-Haute  
**Durée estimée** : 3-4 jours

### Package crypto

- [ ] Nouveau package : `internal/crypto/jws.go`
- [ ] Génération de paire de clés RSA (2048 bits minimum)
- [ ] **Sécurité des clés** :
  - [ ] Clés privées **hors du code source**
  - [ ] Variables d'environnement chiffrées ou HSM
  - [ ] **Permissions restrictives** (600) sur fichiers de clés
  - [ ] **Backup sécurisé** des clés privées
  - [ ] **Rotation des clés** (kid avec timestamp)
- [ ] Émission JWS avec payload `{doc_id, sha256, timestamp}`
- [ ] Stockage du jeton dans `evidence_jws`
- [ ] **Endpoint `/jwks.json`** pour vérification publique
- [ ] **Vérification de signature** (fonction utilitaire)

### Structure JWS

```json
{
  "protected": {
    "alg": "RS256",
    "kid": "key-2025-01-15",
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

### Gestion des clés

- [ ] **Génération initiale** : script ou commande dédiée
- [ ] **Stockage sécurisé** : variables d'environnement ou fichier chiffré
- [ ] **Rotation** : nouvelle clé + ancienne valide 30 jours
- [ ] **JWKS endpoint** : `/jwks.json` avec clés publiques

### Tests

- [ ] Tests génération JWS
- [ ] Tests vérification signature
- [ ] Tests rotation des clés
- [ ] Tests endpoint `/jwks.json`

---

## 🔗 5. Ajouter le ledger hash-chaîné

**Priorité** : 🟠 Moyenne-Haute  
**Durée estimée** : 3-4 jours

### Table ledger

- [ ] Nouvelle table `ledger` :
  ```sql
  CREATE TABLE ledger (
    id SERIAL PRIMARY KEY,
    document_id UUID REFERENCES documents(id) ON DELETE CASCADE,
    hash TEXT NOT NULL,
    previous_hash TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    evidence_jws TEXT
  );
  ```

### Optimisation performance ⚠️ CRITIQUE

- [ ] **Partitionnement mensuel** (prévoir croissance) :
  ```sql
  -- Table principale
  CREATE TABLE ledger (
    id SERIAL,
    document_id UUID,
    hash TEXT NOT NULL,
    previous_hash TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    evidence_jws TEXT,
    PRIMARY KEY (id, timestamp)
  ) PARTITION BY RANGE (timestamp);

  -- Partition mensuelle
  CREATE TABLE ledger_2025_01 PARTITION OF ledger
  FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
  ```

- [ ] **Index optimisés** :
  ```sql
  CREATE INDEX idx_ledger_document_id ON ledger(document_id);
  CREATE INDEX idx_ledger_timestamp ON ledger(timestamp DESC);
  CREATE INDEX idx_ledger_hash ON ledger(hash);
  CREATE INDEX idx_ledger_previous_hash ON ledger(previous_hash);
  ```

- [ ] **Stratégie d'archivage** (après X années)

### Fonction AppendLedger

- [ ] Fonction `AppendLedger(documentID, hash, jws)` dans `internal/ledger/ledger.go`
- [ ] Calcul `newHash = SHA256(prevHash + currentHash)`
- [ ] **Récupération `previous_hash`** : SELECT optimisé avec index
- [ ] **Insertion transactionnelle** (documents + ledger dans la même transaction)
- [ ] Gestion du premier hash (pas de previous_hash)

### Export et vérification

- [ ] Fonction `ExportLedger()` pour vérification externe
- [ ] Fonction `VerifyLedger()` pour vérifier l'intégrité
- [ ] Endpoint `/api/v1/ledger/export` (optionnel)

### Tests

- [ ] Tests hash-chaîné (cohérence)
- [ ] Tests performance (10K+ insertions)
- [ ] Tests transaction (rollback)
- [ ] Tests vérification intégrité

---

## 📣 6. Webhook retour Odoo (avec queue)

**Priorité** : 🟡 Moyenne  
**Durée estimée** : 3-4 jours

**Note** : Ne pas appeler Odoo directement, utiliser une queue.

### Queue de messages

- [ ] **Queue de messages** : Redis Streams ou PostgreSQL LISTEN/NOTIFY
- [ ] **Worker asynchrone** pour envoi webhooks
- [ ] **Retry avec backoff exponentiel** (3 tentatives max)
- [ ] **Dead Letter Queue** pour échecs définitifs
- [ ] **Monitoring** des webhooks (succès/échecs)

### Module webhooks

- [ ] Nouveau module : `internal/webhooks/webhooks.go`
- [ ] Payload standardisé :
  ```json
  {
    "event": "document.vaulted",
    "document_id": "uuid",
    "odoo_id": 12345,
    "status": "VAULTED",
    "evidence_jws": "...",
    "ledger_hash": "...",
    "timestamp": "2025-01-15T10:30:00Z"
  }
  ```
- [ ] **Signature HMAC SHA256** (`X-Vault-Signature`)
- [ ] **Configuration URL webhook** (variable d'environnement)
- [ ] **Endpoint de statut** pour polling de secours (`/api/v1/webhooks/status/:odoo_id`)

### Gestion des erreurs

- [ ] Logging des tentatives
- [ ] Alerting sur échecs répétés
- [ ] Interface de consultation des webhooks en échec

### Tests

- [ ] Tests envoi webhook (mock Odoo)
- [ ] Tests retry et backoff
- [ ] Tests Dead Letter Queue
- [ ] Tests signature HMAC

---

## 🧪 7. Tests, monitoring & observabilité

**Priorité** : 🟢 Basse (mais importante)  
**Durée estimée** : 2-3 jours

### Tests complets

- [ ] Tests unitaires pour `/api/v1/invoices`
- [ ] Tests idempotence (doublons)
- [ ] Tests JWS (génération, vérification)
- [ ] Tests ledger (hash-chaîné, cohérence)
- [ ] Tests transactions (rollback, cohérence)
- [ ] **Tests de charge** :
  - Ledger avec 10K+ entrées
  - Validation Factur-X en concurrence
  - Webhooks (retry, backoff)

### Métriques Prometheus

- [ ] `documents_vaulted_total` (counter)
- [ ] `vault_errors_total` (counter, par type)
- [ ] `vault_duration_seconds` (histogram)
- [ ] `ledger_size_total` (gauge)
- [ ] `jws_generation_duration_seconds` (histogram)
- [ ] `webhook_attempts_total` (counter)
- [ ] `webhook_duration_seconds` (histogram)

### Tracing

- [ ] **OpenTelemetry** pour suivre le flux complet
- [ ] Spans pour chaque étape (upload, JWS, ledger, webhook)
- [ ] Corrélation avec `document_id` et `odoo_id`

### Alerting

- [ ] Alertes sur échecs critiques (DB, stockage)
- [ ] Alertes sur performance (latence élevée)
- [ ] Alertes sur webhooks (taux d'échec > 5%)

### Dashboard (optionnel)

- [ ] Dashboard Grafana avec métriques clés
- [ ] Vue documents vaultés/jour
- [ ] Vue erreurs et latence

---

## 🧭 Ordre d'exécution recommandé (révisé)

| Étape | Description | Priorité | Sprint |
|:--|:--|:--:|:--:|
| 1 | Étendre modèle Document + migration DB | 🔴 | 1 |
| 1bis | Gestion transactions atomiques | 🔴 | 1 |
| 2 | Créer `/api/v1/invoices` | 🔴 | 1 |
| 2bis | Validation Factur-X | 🟡 | 2 (optionnel) |
| 3 | Lier Odoo (webhook validation) | 🟡 | 1 (après étape 2) |
| 4 | Implémenter JWS | 🟠 | 2 |
| 5 | Implémenter ledger | 🟠 | 2 |
| 6 | Ajouter webhook retour Odoo | 🟡 | 3 |
| 7 | Monitoring + tests | 🟢 | 3 |

---

## ✅ Résumé final

### Sprint 1 (2 semaines) — MVP fonctionnel

**Objectif** : Validé → Vaulté (sans JWS ni ledger)

- ✅ Étape 1 : Modèle + Migration
- ✅ Étape 1bis : Transactions atomiques
- ✅ Étape 2 : Endpoint `/api/v1/invoices`
- ✅ Étape 3 : Intégration Odoo (après validation endpoint)

**Livrable** : MVP fonctionnel avec intégration Odoo basique

### Sprint 2 (2 semaines) — Scellement et traçabilité

**Objectif** : Vérifiable (JWS + Ledger)

- ✅ Étape 4 : JWS avec sécurité des clés
- ✅ Étape 5 : Ledger avec optimisations
- ✅ Étape 2bis : Validation Factur-X (optionnel)

**Livrable** : Système complet avec preuves d'intégrité

### Sprint 3 (2 semaines) — Production-ready

**Objectif** : Confort et robustesse

- ✅ Étape 6 : Webhooks avec queue
- ✅ Étape 7 : Monitoring et observabilité

**Livrable** : Système production-ready avec monitoring

---

## ⚠️ Points de vigilance

### Critiques

- 🔴 **Transactions atomiques** : Ne pas oublier, critique pour cohérence
- 🔴 **Sécurité des clés JWS** : Ne pas stocker en clair
- 🔴 **Performance ledger** : Prévoir partitionnement dès le départ

### Importants

- 🟡 **Queue de webhooks** : Ne pas appeler Odoo directement
- 🟡 **Tests intermédiaires** : Valider après chaque étape majeure
- 🟡 **Monitoring** : Mettre en place dès le début

---

## 📝 Notes d'implémentation

### Dépendances à ajouter

```bash
# JWT/JWS
go get github.com/golang-jwt/jwt/v5
go get github.com/go-jose/go-jose/v3

# Queue (optionnel)
go get github.com/redis/go-redis/v9

# Prometheus
go get github.com/prometheus/client_golang
```

### Variables d'environnement

```bash
# Base
DATABASE_URL=postgres://...
STORAGE_DIR=/opt/dorevia-vault/storage

# JWS
JWS_PRIVATE_KEY_PATH=/opt/dorevia-vault/keys/private.pem
JWS_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/public.pem
# OU
JWS_PRIVATE_KEY_BASE64=<base64>
JWS_PUBLIC_KEY_BASE64=<base64>

# Webhooks
ODOO_WEBHOOK_URL=https://odoo.example.com/api/v1/webhooks/vault
ODOO_WEBHOOK_SECRET=<secret HMAC>

# Queue (optionnel)
REDIS_URL=redis://localhost:6379
```

---

**Document créé le** : Janvier 2025  
**Version** : 2.0 (enrichie)  
**Auteur** : Basé sur `demarche_revision_concept.md` + `AVIS_EXPERT_PHASE3.md`

