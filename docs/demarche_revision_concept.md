# ✅ CHECKLIST TECHNIQUE — PHASE 3
## Dorevia Vault — Passage Phase 2 → Phase 3

**Objectif** : Implémenter la règle des 3V (Validé → Vaulté → Vérifiable) et l’intégration automatique Odoo → Vault.

---

## 🧱 1. Étendre le modèle & la base de données

- [ ] Ajouter les champs suivants dans `internal/models/document.go` :
  - `source`, `odoo_model`, `odoo_id`, `odoo_state`
  - `pdp_required`, `dispatch_status`
  - `evidence_jws`, `ledger_hash`
  - `invoice_number`, `invoice_date`, `total_ht`, `total_ttc`, `currency`, `seller_vat`, `buyer_vat`
- [ ] Ajouter la migration SQL correspondante dans `internal/storage/postgres.go`.

---

## 🔌 2. Créer l’endpoint d’ingestion Odoo

- [ ] Nouveau handler `internal/handlers/invoices.go`
- [ ] Endpoint : `POST /api/v1/invoices`
- [ ] Accepte JSON (base64) **et** multipart
- [ ] Valide le payload (source, meta, file)
- [ ] Calcule `sha256`
- [ ] Stocke le fichier + métadonnées
- [ ] Crée la ligne `documents`
- [ ] Retourne `{doc_id, sha256}`
- [ ] Idempotence : si même hash, retourne le document existant

---

## 🔄 3. Relier Odoo (déclencheur “validé”)

- [ ] Odoo : sur `account.move (state='posted')` et `pos.order (state='paid'|'done')`
- [ ] Envoie un `POST /api/v1/invoices`
- [ ] Payload :
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

## 🔐 4. Ajouter le scellement (JWS)

- [ ] Nouveau package : `internal/crypto/jws.go`
- [ ] Génère une paire de clés RSA (`/opt/dorevia-vault/keys/`)
- [ ] Émet un JWS avec `{doc_id, sha256, timestamp}`
- [ ] Stocke le jeton dans `evidence_jws`
- [ ] Expose `/jwks.json` pour vérification publique

---

## 🔗 5. Ajouter le ledger hash-chaîné

- [ ] Nouvelle table `ledger` :
  ```sql
  CREATE TABLE ledger (
    id SERIAL PRIMARY KEY,
    document_id UUID REFERENCES documents(id),
    hash TEXT NOT NULL,
    previous_hash TEXT,
    timestamp TIMESTAMPTZ DEFAULT now(),
    evidence_jws TEXT
  );
  ```
- [ ] Fonction `AppendLedger(documentID, hash, jws)`
- [ ] Calcul `newHash = SHA256(prevHash + currentHash)`
- [ ] Insertion transactionnelle (documents + ledger dans la même transaction)

---

## 📣 6. (Optionnel) Webhook retour Odoo

- [ ] Nouveau module : `internal/webhooks/webhooks.go`
- [ ] Payload :
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
- [ ] Signature HMAC SHA256 (`X-Vault-Signature`)
- [ ] Retry + backoff

---

## 🧪 7. Tests & monitoring

- [ ] Ajouter tests unitaires pour `/api/v1/invoices`
- [ ] Tests idempotence (doublons)
- [ ] Tests JWS et ledger
- [ ] Ajouter métriques Prometheus :
  - `documents_vaulted_total`
  - `vault_errors_total`
  - `vault_duration_seconds`

---

## 🧭 Ordre d’exécution recommandé

| Étape | Description | Priorité |
|:--|:--|:--:|
| 1 | Étendre modèle Document + migration DB | 🔴 |
| 2 | Créer `/api/v1/invoices` | 🔴 |
| 3 | Lier Odoo (webhook validation) | 🔴 |
| 4 | Implémenter JWS | 🟠 |
| 5 | Implémenter ledger | 🟠 |
| 6 | Ajouter webhook retour Odoo | 🟡 |
| 7 | Monitoring + tests | 🟢 |

---

## ✅ Résumé final

- **Étape 1 à 3** → MVP fonctionnel (Validé → Vaulté)  
- **Étape 4 à 5** → Scellement et traçabilité (Vérifiable)  
- **Étape 6 à 7** → Confort et robustesse (suivi et supervision)

---

**Document généré automatiquement — Phase 3 Checklist Technique**
