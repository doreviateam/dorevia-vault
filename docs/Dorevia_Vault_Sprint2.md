# Dorevia Vault — Sprint 2 (Vérifiable)

**Objectif Sprint 2** : rendre chaque document **vérifiable** via (1) **scellement JWS** et (2) **ledger hash‑chaîné**.  
**Hors scope** : webhooks asynchrones, monitoring avancé, validation Factur‑X (reportés Sprint 3+).

**Durée cible** : 10–12 jours  
**Version** : 1.0  
**Date** : novembre 2025

---

## 0) Portée & livrables

### Inclus (à livrer)
- Génération et **gestion de clés RSA** (outil `cmd/keygen`).
- **Signature JWS** du triplet `{document_id, sha256, timestamp}` → `evidence_jws`.
- Endpoint public **`/jwks.json`** exposant les clés publiques (JWKS) avec **`kid`**.
- **Table `ledger`** append‑only + fonction de **chaînage** SHA‑256.
- Intégration **transactionnelle** : document vaulté → signé → inscrit dans le ledger.
- Fonctions **`VerifyJWS()`** et **`VerifyLedger()`** + **export ledger** (JSON/CSV minimal).
- Scripts d’ops (Makefile), exemples cURL & commandes de test de perf.

### Exclus / reportés
- Rotation de clés **automatisée** (préparer le champ `kid`, rotation **manuelle** OK en Sprint 2).
- Partitionnement du ledger (à introduire si volumétrie > 100k entrées / an, Sprint 3+).
- File/queue de webhooks & monitoring complet (Sprint 3).

---

## 1) Arborescence cible

```
dorevia-vault/
├─ cmd/
│  ├─ vaultd/
│  │  └─ main.go                  # serve + routes (inclut /jwks.json)
│  └─ keygen/
│     └─ main.go                  # génération paires RSA + JWKS
├─ internal/
│  ├─ crypto/
│  │  └─ jws.go                   # Sign/Verify + JWKS builder
│  ├─ ledger/
│  │  ├─ append.go                # AppendLedger + VerifyLedger
│  │  └─ export.go                # ExportLedger JSON/CSV
│  ├─ handlers/
│  │  ├─ invoices.go              # POST /api/v1/invoices (intègre JWS + ledger)
│  │  └─ jwks.go                  # GET /jwks.json
│  ├─ storage/
│  │  └─ postgres.go              # migrations + tx intégrée
│  ├─ models/
│  │  └─ document.go              # evidence_jws, ledger_hash
│  └─ metrics/
│     └─ prometheus.go            # (optionnel) jws_generation_duration_seconds
├─ migrations/
│  ├─ 003_add_odoo_fields.sql     # Sprint 1 (existant)
│  └─ 004_add_ledger.sql          # Sprint 2 (nouveau)
├─ docs/
│  └─ sprint_2_plan.md            # ce document
└─ Makefile
```

---

## 2) Variables d’environnement (Sprint 2)

```bash
# Chemins de clés (permissions 600, non committés)
JWS_PRIVATE_KEY_PATH=/opt/dorevia-vault/keys/private.pem
JWS_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/public.pem
JWS_KID=key-2025-Q4

# Alternatives (clé en base64, à décoder au démarrage)
JWS_PRIVATE_KEY_BASE64=
JWS_PUBLIC_KEY_BASE64=

# JWKS cache (optionnel)
JWKS_CACHE_TTL=300s

# Ledger
LEDGER_ENABLED=true
```

> Les clés peuvent être fournies en **PEM** (fichiers) ou via **base64** dans les variables d’environnement.
> **Toujours** appliquer `chmod 600` aux fichiers de clés et les sortir du dépôt git.

---

## 3) Migration SQL — `004_add_ledger.sql`

```sql
-- Table ledger
CREATE TABLE IF NOT EXISTS ledger (
  id SERIAL PRIMARY KEY,
  document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  hash TEXT NOT NULL,
  previous_hash TEXT,
  timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
  evidence_jws TEXT
);

-- Index
CREATE INDEX IF NOT EXISTS idx_ledger_document_id ON ledger(document_id);
CREATE INDEX IF NOT EXISTS idx_ledger_timestamp ON ledger(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_hash ON ledger(hash);
CREATE INDEX IF NOT EXISTS idx_ledger_prev_hash ON ledger(previous_hash);
```

**Commande de migration** :
```bash
go run cmd/vaultd/main.go migrate up
```

---

## 4) Crypto JWS — Conception & API

### 4.1 Payload de scellement
```json
{
  "document_id": "uuid",
  "sha256": "hex",
  "timestamp": "2025-11-09T10:30:00Z"
}
```

### 4.2 Algorithme & en-tête
- **Signature** : `RS256` (RSA‑SHA256)
- **En‑tête protégé** :
```json
{ "alg":"RS256", "kid":"key-2025-Q4", "typ":"JWT" }
```

### 4.3 API Go (extraits)
```go
// internal/crypto/jws.go
type Evidence struct {
    DocumentID string    `json:"document_id"`
    Sha256     string    `json:"sha256"`
    Timestamp  time.Time `json:"timestamp"`
}

func SignEvidence(docID, shaHex string, t time.Time) (jws string, err error)
func VerifyEvidence(jws string) (Evidence, error)
func CurrentJWKS() ([]byte, error) // JWKS construit à partir des clés publiques
```

### 4.4 Génération de clés (outil)
```bash
# Générer clés RSA 2048 + JWKS minimal
go run cmd/keygen/main.go --out /opt/dorevia-vault/keys --kid key-2025-Q4
chmod 600 /opt/dorevia-vault/keys/*.pem
```

**Notes sécurité :**
- Conserver une **copie chiffrée** de la clé privée (GPG / coffre).
- **Rotation manuelle** : nouvelle paire → ajouter au JWKS avec nouveau `kid`, conserver l’ancienne 30 jours pour vérif.

---

## 5) Ledger — Chaînage & Vérification

### 5.1 Règle de chaînage
```
ledger_hash = SHA256(previous_hash + sha256_document)
```
- **`previous_hash`** : dernier `hash` (par `timestamp DESC, id DESC`).
- **Premier enregistrement** : `previous_hash = NULL` (à gérer explicitement).

### 5.2 API Go (extraits)
```go
// internal/ledger/append.go
func AppendLedger(ctx context.Context, db *pgxpool.Pool, docID, shaHex, jws string) (hash string, err error)

// internal/ledger/export.go
func ExportLedgerJSON(ctx context.Context, w io.Writer) error
func ExportLedgerCSV(ctx context.Context, w io.Writer) error

// internal/ledger/verify.go
func VerifyLedger(ctx context.Context, db *pgxpool.Pool) error // recalcul et comparaison
```

### 5.3 Points d’implémentation
- **Sélection du previous** : `SELECT hash FROM ledger ORDER BY timestamp DESC, id DESC LIMIT 1`.
- **Concurrence** : pour de gros volumes, utiliser **verrou court** ou **séquence** pour garantir l’ordre.  
  (En S2, insertion séquentielle suffit.)
- **Idempotence** : ne pas dupliquer les entrées ledger pour un même `sha256`; protéger via transaction + unique `(document_id, hash)` si souhaité.

---

## 6) Intégration transactionnelle (ingestion → scellement → ledger)

### 6.1 Ordre recommandé
1) **Écriture fichier** → chemin tmp.
2) **BEGIN** transaction.
3) **INSERT** dans `documents` (avec `sha256_hex`, path, meta).
4) **Génération JWS** (hors DB, rapide).
5) **AppendLedger** (INSERT dans `ledger` avec `previous_hash`).
6) **UPDATE documents** pour stocker `evidence_jws` & `ledger_hash`.
7) **COMMIT**.
8) **Move** fichier tmp → chemin final + cleanup.

> Si `AppendLedger` échoue → rollback et suppression du fichier tmp.  
> `evidence_jws` est **persisté** dans `documents` et **copié** dans `ledger`.

### 6.2 Pseudo‑code d’assemblage (extrait réaliste)
```go
// internal/handlers/invoices.go (post-Sprint 1)
doc, err := store.InsertDocumentTx(ctx, meta, raw) // steps 1-3 + move tmp contrôlé par tx
if err != nil { /* handle */ }

jws, err := crypto.SignEvidence(doc.ID, doc.Sha256Hex, time.Now().UTC())
if err != nil { /* rollback & cleanup */ }

hash, err := ledger.AppendLedger(ctx, store.Pool, doc.ID, doc.Sha256Hex, jws)
if err != nil { /* rollback & cleanup */ }

if err := store.AttachEvidence(ctx, doc.ID, jws, hash); err != nil { /* rollback & cleanup */ }

// commit s’effectue dans InsertDocumentTx ou via une tx englobante selon design
```

---

## 7) Endpoint `/jwks.json`

- **Route** : `GET /jwks.json`  
- **Cache** : 5 minutes (configurable `JWKS_CACHE_TTL`).  
- **Contenu** : jeu de clés publiques (RSA) actives, chaque entrée avec `kid`.

Exemple JWKS :
```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-2025-Q4",
      "use": "sig",
      "alg": "RS256",
      "n": "…",
      "e": "AQAB"
    }
  ]
}
```

---

## 8) Sécurité & Exploitation

- **Permissions** des clés : `0600`, propriétaire service `vaultd`.
- **Backup** des clés privées : gpg, coffre externe, double contrôle (4‑yeux).
- **Rotation** : nouvelle paire → **ajouter** au JWKS (nouveau `kid`), garder l’ancienne **30 jours**.
- **Logs** : tracer `doc_id`, `odoo_id`, `kid`, latence `SignEvidence`.
- **Fail‑safe** : si signature indisponible → refuser l’ingestion (`503`), **ne pas** vault‑er sans JWS en S2.
- **Intégrité** : exécuter `VerifyLedger()` quotidiennement (cron) et alerter en cas d’échec.

Checklist sécurité rapide :
- [ ] Clés hors dépôt git, perms 600, propriétaire `vaultd`.
- [ ] Fichiers de clés sur disque chiffré (si possible).
- [ ] Variables d’environnement non loguées.
- [ ] Journaux sans fuite de secrets (redaction).

---

## 9) Tests

### 9.1 Unitaires
- **JWS** : `SignEvidence()` + `VerifyEvidence()` (doc corrompu, horodatage invalide, kid inconnu).
- **Ledger** : `AppendLedger()` (premier, enchaînement, corruption volontaire), `VerifyLedger()`.
- **Handlers** : `/api/v1/invoices` → happy‑path, idempotence, erreur signature, erreur ledger.

### 9.2 Intégration
- Pipeline complet Odoo → Vault → JWS → Ledger → Export.
- `VerifyLedger()` sur un corpus de 1 000 docs (temps cible < 2 s local).

### 9.3 Non‑régression (depuis S1)
- Idempotence conservée (même hash → même doc, **pas** de duplication ledger).

### 9.4 Commandes de test (exemples)
```bash
# Construire le binaire et lancer le service
go build -o bin/vaultd ./cmd/vaultd && ./bin/vaultd

# Générer une paire de clés pour S2
go run cmd/keygen/main.go --out /opt/dorevia-vault/keys --kid key-2025-Q4

# Vérifier JWKS
curl -s http://localhost:8080/jwks.json | jq

# Ingestion d’une facture (JSON + base64)
curl -sS -X POST http://localhost:8080/api/v1/invoices   -H 'Content-Type: application/json'   -d '{
    "source": "sales",
    "model": "account.move",
    "odoo_id": 12345,
    "state": "posted",
    "pdp_required": false,
    "file": "'$(base64 -w0 document.pdf)'",
    "meta": {
      "number": "F2025-00123",
      "invoice_date": "2025-11-09",
      "total_ht": 158.33,
      "total_ttc": 190.00,
      "currency": "EUR"
    }
  }' | jq

# Export ledger
curl -s http://localhost:8080/api/v1/ledger/export?format=json | jq
```

---

## 10) Métriques (léger)

- `jws_generation_duration_seconds` (histogram).
- `ledger_append_duration_seconds` (histogram).
- `vault_errors_total{type="jws|ledger"}` (counter).

> L’export `/metrics` existe déjà ; ajouter ces instruments reste optionnel en Sprint 2 si le temps manque.

---

## 11) Makefile (exemple)

```makefile
run:
	go run cmd/vaultd/main.go serve

migrate:
	go run cmd/vaultd/main.go migrate up

keygen:
	go run cmd/keygen/main.go --out /opt/dorevia-vault/keys --kid $${KID}

test:
	go test ./...

lint:
	golangci-lint run
```

---

## 12) Plan jour‑par‑jour (10–12 j)

| Jour | Tâches | Livrables |
|---:|---|---|
| J1 | Générateur de clés RSA + `kid` | `cmd/keygen/` + PEM + JWKS local |
| J2 | Module `internal/crypto/jws.go` (Sign/Verify) | tests unitaires JWS |
| J3 | Endpoint `/jwks.json` + cache | `handlers/jwks.go` |
| J4 | Migration `004_add_ledger.sql` | table + index OK |
| J5 | `AppendLedger()` + tests | hash chaîné fiable |
| J6 | `VerifyLedger()` + export JSON/CSV | `export.go` + CLI si besoin |
| J7 | Intégration dans `POST /invoices` (flux complet) | doc signé + ledger inscrit |
| J8 | Gestion d’erreurs & rollback renforcé | cas d’échec couverts |
| J9 | Tests d’intégration (1000 docs) | perf & cohérence |
| J10 | Documentation + exemples | `docs/sprint_2_plan.md` + README |
| J11‑J12 | Buffer de stabilisation | bugfix, polish |

---

## 13) Definition of Done (DoD)

- ✅ `/api/v1/invoices` signe chaque document (JWS) et l’inscrit au ledger.
- ✅ Clés RSA gérées hors dépôt, `kid` visible dans JWS et JWKS.
- ✅ `VerifyEvidence()` valide toute signature émise.
- ✅ `VerifyLedger()` détecte toute corruption de chaîne.
- ✅ Export ledger disponible (JSON ou CSV).
- ✅ Idempotence préservée (aucune duplication ledger pour un même `sha256`).

---

## 14) Risques & Parades

| Risque | Impact | Parade |
|---|---|---|
| Perte/compromission clé privée | Preuves invalidées | Backups chiffrés, rotation, séparation des rôles |
| Rupture du chaînage | Impossibilité de prouver l’ordre | `VerifyLedger()` périodique + alerting (S3) |
| Contention DB sur ledger | Latence | Index + insertion séquentielle, partitionnement S3 |
| Débit limité par RSA | Latence ingestion | Pool de signeurs (goroutines) si >500 docs/j |

---

## 15) Étapes suivantes (pré‑Sprint 3)

- Préparer la **queue webhooks** (Redis Streams ou PG LISTEN/NOTIFY).
- Définir **payload de retour Odoo** + signature HMAC.
- Renforcer **Prometheus** + tracing **OpenTelemetry**.
- Écrire la **procédure d’audit** (comment vérifier un doc et la chaîne en externe).

---

**Fin du document — Sprint 2 (Vérifiable)** ✅
