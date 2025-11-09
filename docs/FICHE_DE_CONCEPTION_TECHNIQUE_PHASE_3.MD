# FICHE DE CONCEPTION TECHNIQUE
## Interopérabilité **Dorevia Vault** ↔ **OCA eInvoicing (PDP/PPF)**

**Version**: 0.1 (draft)  
**Auteur**: Doreviateam  
**Date**: 09/11/2025  
**Portée**: Spécifier le rôle du *Vault* comme **service intermédiaire d’intégrité/archivage** entre **Odoo CE 18 (modules OCA)** et **PDP/PPF**, sans devenir PDP.

---

## 0bis) Principe fondateur — **Règle des 3 V**
> **Validé → Vaulté → Vérifiable**

- **Validé** : le document atteint un état **juridiquement engageant** dans Odoo (voir déclencheurs ci‑dessous).  
- **Vaulté** : il est immédiatement **transféré, hashé (SHA‑256), jetonné (JWS)** et inscrit dans le **ledger**.  
- **Vérifiable** : la preuve est **indépendamment vérifiable** (JWKS public, export ledger, horodatage option TSA).

Cette règle est **universelle** et s’applique à **tous les flux économiques** (ventes, achats, POS, avoirs, BL optionnel).

---

## 1) Objectifs & périmètre
- Assurer **intégrité**, **traçabilité**, **preuve portable** pour chaque facture sortante Odoo.  
- Normaliser l’ingestion **Factur‑X** / **PDF + XML** et préparer le **dispatch** vers un **connecteur PDP/PPF** (fourni/initié par OCA).  
- Conserver **archivage long terme** (fichier + preuves + statuts).  
- Exposer **API** stables pour Odoo et le connecteur OCA (statuts, webhooks, vérification).

**Hors périmètre** : devenir PDP, supervision réglementaire, responsabilité de transmission légale (déléguée au PDP/PPF choisi).

---

## 2) Hypothèses & dépendances
- Odoo CE 18 génère **Factur‑X** (profil EN16931) ou **PDF + XML embarqué**.  
- Les modules OCA eInvoicing fourniront : formats UBL/EN16931, mapping TVA/SIRET, et un **connecteur PDP/PPF** (ou Peppol).  
- Le Vault agit comme **proxy d’intégrité** : hash, jeton signé (JWS), ledger, archivage, routage.  
- Signature avancée (PAdES/XAdES) via microservice **DSS** (Java) — option activable.

---

## 3) Vue d’architecture
```
Odoo CE 18 (OCA)
   |  Factur-X/PDF + meta (REST)
   v
Dorevia Vault (proxy d’intégrité)
   ├─ Validation Factur-X / meta
   ├─ Hash SHA-256
   ├─ Jeton signé JWS (+ TSA opt.)
   ├─ Ledger hash-chaîné (append-only)
   ├─ (opt) Scellement PAdES/XAdES
   ├─ Dispatch → Adaptateur « OCA PDP/PPF / Peppol »
   v
PDP / PPF / Peppol
   |  ACK/REJECT
   v
Vault → Odoo (webhook statut signé HMAC)
```

---

## 4bis) Déclencheurs Odoo → Vault (états de validation)
La validation opérationnelle côté Odoo devient le **signal d’archivage** côté Vault.

| Module / Modèle Odoo | État « validé » (technique) | Doit être vaulté ? | Notes |
|:--|:--|:--:|:--|
| Facture client (`account.move` type `out_invoice`) | `state = 'posted'` | ✅ | Factur‑X/PDF → pré‑PDP si B2B/B2G |
| Facture fournisseur (`account.move` type `in_invoice`) | `state = 'posted'` | ✅ | Archivage + réconciliation reçus PDP |
| Avoir client/fournisseur (`account.move` type `out_refund`/`in_refund`) | `state = 'posted'` | ✅ | Type **CREDIT_NOTE** si envoi PDP |
| Ticket POS (`pos.order`) | `state = 'paid'` ou `state = 'done'` | ✅ | B2C (NF525‑like) : archivage interne, pas PDP |
| Bon de livraison (`stock.picking`) | `state = 'done'` | ⚙️ Option | GED logistique (preuve livraison) |
| Commande client (`sale.order`) | `state = 'sale'` | ⚙️ Option | Engagement commercial, non PDP |

**Règle fonctionnelle officielle** :  
> *Tout document Odoo passant à l’un des états ci‑dessus « validé » déclenche automatiquement l’API d’ingestion du Vault. Les documents brouillons (`draft`) ne sont jamais vaultés.*

---

## 4ter) Flux multi‑sources unifiés (POS, ventes, achats)
Le même pipeline Vault s’applique, avec une décision de **routage** selon `pdp_required` :

1. **Réception** (`POST /api/v1/invoices` ou `/documents`) avec `source` (`sales|purchase|pos|stock|sale`) et `meta`.  
2. **Scellement** : SHA‑256, JWS, entrée ledger (append‑only).  
3. **Routage** :  
   - `pdp_required = true` → **dispatch PDP/PPF** via adaptateur, suivi `status`.  
   - `false` → **archivage interne** (POS/B2C, BL, devis…).  
4. **Retour statut** → webhook Odoo signé HMAC.

### Exemple de payload standardisé
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

## 6bis) Politique d’idempotence & corrélation
- **Idempotency-Key** : `sha256(file)` ou UUID fourni par Odoo ; en cas de doublon, Vault répond `200` avec les identifiants existants.  
- **Clés de corrélation** : `odoo_move_id`, `pdp_message_id`, `dispatch_id` stockées côté Vault et répliquées en champs techniques Odoo.  
- **Replays** PDP gérés par `dispatch.status` + backoff.

---

## 9bis) Tests fonctionnels — règles 3V
- **TF‑01** : passage `draft→posted` crée `document`, `evidence_jws`, `ledger` (hash‑chaîne).  
- **TF‑02** : doublon fichier → idempotence (mêmes IDs).  
- **TF‑03** : POS `paid/done` vaulté sans dispatch PDP.  
- **TF‑04** : envoi PDP rejeté → `status=REJECTED` + motif remonté à Odoo.  
- **TF‑05** : vérification publique d’un JWS via JWKS (clé rotatée `kid`).

---

**Résumé** : Dorevia Vault devient la **couche de confiance universelle** entre Odoo et PDP/PPF, fondée sur la **règle des 3 V (Validé → Vaulté → Vérifiable)**, garantissant l’intégrité, la traçabilité et la vérifiabilité de tous les documents économiques.
