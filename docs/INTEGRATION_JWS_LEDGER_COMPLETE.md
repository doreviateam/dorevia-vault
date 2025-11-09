# ✅ Intégration JWS + Ledger Complète — Sprint 2

**Date** : Janvier 2025  
**Statut** : ✅ **Implémenté et compilé**

---

## 📊 Résumé de l'Intégration

### Modules Créés

| Module | Fichier | Description | Statut |
|:-------|:--------|:------------|:-------|
| **JWS Service** | `internal/crypto/jws.go` | Signature et vérification JWS (RS256) | ✅ |
| **Key Generator** | `cmd/keygen/main.go` | Génération de clés RSA + JWKS | ✅ |
| **JWKS Handler** | `internal/handlers/jwks.go` | Endpoint `/jwks.json` | ✅ |
| **Evidence Storage** | `internal/storage/document_with_evidence.go` | Intégration JWS + Ledger | ✅ |

### Modifications

| Fichier | Modification | Statut |
|:--------|:-------------|:-------|
| `internal/models/document.go` | Ajout `EvidenceJWS` et `LedgerHash` | ✅ |
| `internal/config/config.go` | Ajout config JWS + Ledger | ✅ |
| `internal/handlers/invoices.go` | Intégration JWS + Ledger dans flux | ✅ |
| `internal/storage/queries.go` | Mise à jour `GetDocumentByID` | ✅ |
| `internal/storage/postgres.go` | Mise à jour INSERT avec evidence | ✅ |
| `migrations/003_add_odoo_fields.sql` | Ajout colonnes `evidence_jws`, `ledger_hash` | ✅ |
| `cmd/vault/main.go` | Initialisation JWS + route `/jwks.json` | ✅ |

---

## 🔧 Fonctionnalités Implémentées

### 1. Service JWS (`internal/crypto/jws.go`)

- ✅ **SignEvidence** : Signature JWS (RS256) avec payload `{document_id, sha256, timestamp}`
- ✅ **VerifyEvidence** : Vérification JWS et extraction de l'Evidence
- ✅ **CurrentJWKS** : Génération du JWKS (JSON Web Key Set) pour clés publiques
- ✅ **Chargement clés** : Depuis fichiers PEM ou variables d'environnement (base64)

### 2. Générateur de Clés (`cmd/keygen/main.go`)

- ✅ Génération paire RSA-2048 (configurable)
- ✅ Export PEM (privé + public)
- ✅ Génération JWKS automatique
- ✅ Permissions sécurisées (600 pour privé, 644 pour public)

### 3. Endpoint JWKS (`/jwks.json`)

- ✅ Retourne le JWKS au format JSON standard
- ✅ Cache HTTP (5 minutes)
- ✅ Disponible uniquement si JWS activé

### 4. Intégration dans Flux (`StoreDocumentWithEvidence`)

- ✅ **Transaction atomique** : Fichier → DB → JWS → Ledger → UPDATE evidence
- ✅ **Mode dégradé JWS** : Continue sans JWS si `JWS_REQUIRED=false`
- ✅ **Ledger hash-chaîné** : Intégration avec verrou `FOR UPDATE`
- ✅ **Idempotence renforcée** : Vérification ledger pour documents existants

---

## 🚀 Utilisation

### 1. Générer les Clés RSA

```bash
# Générer paire de clés + JWKS
go run ./cmd/keygen/main.go \
  --out /opt/dorevia-vault/keys \
  --kid key-2025-Q1 \
  --bits 2048

# Sécuriser les permissions
chmod 600 /opt/dorevia-vault/keys/private.pem
chmod 644 /opt/dorevia-vault/keys/public.pem
```

### 2. Configuration Environnement

```bash
# JWS Configuration
export JWS_ENABLED=true
export JWS_REQUIRED=true
export JWS_PRIVATE_KEY_PATH=/opt/dorevia-vault/keys/private.pem
export JWS_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/public.pem
export JWS_KID=key-2025-Q1

# Ledger Configuration
export LEDGER_ENABLED=true

# Database (requis)
export DATABASE_URL="postgres://user:pass@localhost/dorevia_vault"
```

### 3. Démarrer le Serveur

```bash
go run ./cmd/vault
```

**Logs attendus** :
```
{"level":"info","message":"JWS service initialized","kid":"key-2025-Q1"}
{"level":"info","message":"JWKS endpoint enabled: /jwks.json"}
{"level":"info","message":"PostgreSQL connection established"}
```

### 4. Tester l'Intégration

#### Test Ingestion avec JWS + Ledger

```bash
curl -X POST http://localhost:8080/api/v1/invoices \
  -H "Content-Type: application/json" \
  -d '{
    "source": "sales",
    "model": "account.move",
    "odoo_id": 123,
    "odoo_state": "posted",
    "file": "base64_encoded_file_content",
    "filename": "invoice_001.pdf"
  }'
```

**Réponse attendue** :
```json
{
  "id": "uuid",
  "sha256_hex": "abc123...",
  "created_at": "2025-01-09T10:30:00Z",
  "evidence_jws": "eyJhbGciOiJSUzI1NiIs...",
  "ledger_hash": "def456..."
}
```

#### Test JWKS Endpoint

```bash
curl http://localhost:8080/jwks.json
```

**Réponse attendue** :
```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-2025-Q1",
      "use": "sig",
      "alg": "RS256",
      "n": "base64_modulus",
      "e": "AQAB"
    }
  ]
}
```

#### Test Export Ledger

```bash
curl "http://localhost:8080/api/v1/ledger/export?format=json&limit=10"
```

---

## 🔒 Sécurité

### Mode Dégradé JWS

- **JWS_ENABLED=true, JWS_REQUIRED=false** : Continue sans JWS si génération échoue
- **JWS_ENABLED=true, JWS_REQUIRED=true** : Échoue si JWS ne peut pas être généré
- **JWS_ENABLED=false** : JWS complètement désactivé

### Gestion des Clés

- ✅ Clé privée : Permissions 600 (lecture/écriture propriétaire uniquement)
- ✅ Clé publique : Permissions 644 (lecture publique, écriture propriétaire)
- ✅ Support base64 : Variables d'environnement pour déploiement conteneurisé

---

## 📝 Flux Complet

### Ingestion Document avec JWS + Ledger

```
1. POST /api/v1/invoices
   ↓
2. Validation payload + décodage base64
   ↓
3. StoreDocumentWithEvidence()
   ├─ Calcul SHA256
   ├─ Vérification idempotence
   ├─ BEGIN transaction
   ├─ Stockage fichier (tmp)
   ├─ INSERT documents
   ├─ SignEvidence() → JWS
   ├─ AppendLedger() → Ledger hash
   ├─ UPDATE documents (evidence_jws, ledger_hash)
   ├─ COMMIT
   └─ Rename tmp → final
   ↓
4. Réponse 201 Created avec evidence_jws + ledger_hash
```

### Idempotence Renforcée

```
Document existant (même SHA256)
   ↓
Récupération document existant
   ↓
Vérification ledger (ExistsByDocumentID)
   ↓
Si pas de ledger → TODO: Compléter (optionnel)
   ↓
Réponse 200 OK avec evidence_jws + ledger_hash existants
```

---

## ✅ Tests

### Compilation

- ✅ `go build ./cmd/vault` — **OK**
- ✅ `go build ./cmd/keygen` — **OK**
- ✅ Aucune erreur de linter

### Tests Unitaires

- ⏳ Tests JWS à créer (pending)
- ✅ Tests ledger existants (4/4 passent)
- ✅ Tests unitaires existants (19/19 passent)

---

## 🎯 Prochaines Étapes

1. **Tests Unitaires JWS** : Créer tests pour `SignEvidence` et `VerifyEvidence`
2. **Tests d'Intégration** : Valider flux complet avec DB réelle
3. **Documentation API** : Swagger/OpenAPI pour endpoints
4. **Rotation Clés** : Implémenter support multi-clés dans JWKS

---

## 📚 Références

- **JWT/JWS** : [RFC 7519](https://tools.ietf.org/html/rfc7519), [RFC 7515](https://tools.ietf.org/html/rfc7515)
- **JWKS** : [RFC 7517](https://tools.ietf.org/html/rfc7517)
- **Plan Sprint 2** : `docs/Dorevia_Vault_Sprint2.md`
- **Patch Consolidé** : `docs/PATCH_CONSOLIDE_SPRINT2_ANALYSE.md`

---

**Document créé le** : Janvier 2025  
**Version** : 1.0  
**Statut** : ✅ **Intégration complète**

