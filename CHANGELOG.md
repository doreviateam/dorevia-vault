# 📜 CHANGELOG — Dorevia Vault

Ce fichier suit la convention [Keep a Changelog](https://keepachangelog.com/fr/1.0.0/),  
et respecte la sémantique de versionnage : `MAJEURE.MINEURE.PATCH`.

---

## [1.4.0] — Janvier 2025

### 🎫 Ingestion Native Tickets POS (Sprint 6)

#### Ajouté

**Phase 0 — Architecture Modulaire**
- Interface `DocumentRepository` pour abstraction de la couche de stockage
- Interface `ledger.Service` pour abstraction du service ledger
- Interface `crypto.Signer` pour abstraction de la signature (HSM-ready)
- Type `PosTicketInput` pour séparation handlers/services

**Phase 1 — Préparation**
- Migration DB : Champs POS (`payload_json`, `source_id_text`, `pos_session`, `cashier`, `location`)
- Canonicalisation JSON : Tri des clés, suppression null, normalisation nombres
- Index optimisés : GIN index sur `payload_json`, index partiels pour recherche POS

**Phase 2 — Abstraction Crypto**
- Adaptateur `LocalSigner` : Implémentation `Signer` utilisant `crypto.Service` existant
- Support futur HSM via interface `Signer`

**Phase 3 — Service Métier**
- `PosTicketsService` : Service d'ingestion avec idempotence métier stricte
- Hash basé sur `ticket + source_id + pos_session` (Option A)
- Intégration complète avec ledger et signer

**Phase 4 — Handler API**
- Endpoint `POST /api/v1/pos-tickets` : Ingestion native tickets POS
- Validation complète (taille, champs obligatoires)
- Réponse standardisée avec métadonnées complètes
- Configuration `POS_TICKET_MAX_SIZE_BYTES` (défaut: 64 KB)

**Phase 5 — Observabilité**
- Métriques Prometheus : `documents_vaulted_total{status, source="pos"}`
- Logs structurés avec contexte complet (tenant, source_model, source_id, document_id, sha256, ledger_hash, evidence_jws, duration)
- Gestion code HTTP : 200 OK pour idempotence, 201 Created pour création

**Phase 6 — Tests d'Intégration**
- 5 tests d'intégration : End-to-end, idempotence, canonicalisation, métriques
- 20 tests unitaires : Canonicalisation (4), Service (7), Handler (8), Signer (1)

#### Modifié

- `internal/models/document.go` : Champs POS ajoutés
- `internal/storage/postgres.go` : Fonction `migrateSprint6()` ajoutée
- `internal/config/config.go` : Configuration `PosTicketMaxSizeBytes` ajoutée
- `cmd/vault/main.go` : Route POS enregistrée

#### Documentation

- `docs/POS_TICKETS_API.md` : Documentation complète de l'API POS
- `docs/VALIDATION_SPRINT6.md` : Rapport de validation Sprint 6
- `docs/PLAN_IMPLEMENTATION_SPRINT6_CORRIGE.md` : Plan d'implémentation détaillé
- `RELEASE_NOTES_v1.4.0.md` : Notes de version complètes

---

## [1.3.0] — Janvier 2025

### 🔐 Sécurité & Interopérabilité (Sprint 5)

#### Ajouté

**Phase 5.1 — Sécurité & Key Management**
- Intégration **HashiCorp Vault** pour stockage sécurisé des clés privées
- **Rotation multi-KID** : Support de plusieurs clés actives simultanément
- **Chiffrement au repos** : AES-256-GCM pour logs d'audit sensibles
- Interface `KeyManager` abstraite (Vault / fichiers locaux)
- 24 tests unitaires pour modules crypto

**Phase 5.2 — Authentification & Autorisation**
- **Authentification JWT** (RS256) et **API Keys** avec expiration
- **RBAC** : 4 rôles (admin, auditor, operator, viewer) avec 7 permissions
- **Middleware Fiber** : Protection automatique des endpoints sensibles
- Mapping endpoints → permissions automatique
- 25 tests unitaires pour auth/RBAC

**Phase 5.3 — Interopérabilité**
- **Validation Factur-X** : Parsing XML UBL, validation EN 16931, extraction métadonnées
- **Webhooks asynchrones** : Queue Redis, workers parallèles, retry avec backoff exponentiel
- **Signature HMAC** : Sécurité webhooks avec HMAC-SHA256
- Intégration dans handlers (`document.vaulted`, `document.verified`)
- 23 tests unitaires (validation + webhooks)

**Phase 5.4 — Scalabilité**
- **Partitionnement ledger** : Partitions mensuelles automatiques (PostgreSQL 14+)
- **Optimisations DB** : 5 index optimisés, ANALYZE/VACUUM automatiques
- Migration transparente des données existantes
- 10 tests unitaires pour partitionnement

#### Modifié

- Endpoints protégés : `/audit/export`, `/api/v1/ledger/export`, `/api/v1/invoices`, etc.
- Handler `/api/v1/invoices` : Validation Factur-X automatique, métadonnées enrichies
- Handler `/api/v1/ledger/verify` : Émission webhook `document.verified`
- Configuration : 15+ nouvelles variables d'environnement

#### Documentation

- 6 documents de spécification créés :
  - `docs/security_vault_spec.md`
  - `docs/auth_rbac_spec.md`
  - `docs/facturx_validation_spec.md`
  - `docs/webhooks_spec.md`
  - `docs/partitioning_spec.md`
  - `docs/SPRINT5_PLAN.md`

---

## [1.2.0-rc1] — 28 février 2025  

### 🚀 Audit & Conformité (Phase 4.4)

#### Ajouté

- Génération complète de **rapports d'audit** (JSON, CSV, PDF) avec signature **JWS RS256**.  
- **CLI `audit`** : génération manuelle ou scriptée des rapports mensuels/trimestriels.  
- **PDF 8 pages** avec QR code du hash SHA256 et signature JWS intégrée.  
- Collecte et consolidation des statistiques : documents, erreurs, ledger, réconciliations.  
- 39 nouveaux tests unitaires : 15 (report) + 14 (PDF) + 10 (CLI).  
- Documentation : `docs/audit_export_spec.md` et `SPRINT4_PHASE4.4_PLAN.md`.

#### Modifié

- Harmonisation des noms et seuils des métriques Prometheus.  
- Refonte partielle du module `internal/audit` (logs, export, sign, report, pdf).  
- Amélioration du `health/detailed` : inclusion vérification ledger + stockage.  
- Nettoyage du code CLI (`flag` et validation des périodes).

#### Corrigé

- Blocage aléatoire sur écriture ledger lors de pics I/O.  
- Rotation des logs d'audit maintenant stable < 24 h.  
- Correctifs mineurs : calcul médian document_size et gestion JSON invalides.  

---

## [1.1.0] — 30 janvier 2025  

### 🔍 Supervision & Réconciliation (Sprint 3)

#### Ajouté

- Endpoint `/health/detailed` : vérifications DB, JWS, ledger, stockage.  
- Module Prometheus (11 métriques métier) + export `/metrics`.  
- Endpoint `/api/v1/ledger/verify/:id` + option `signed=true`.  
- CLI `reconcile` : détection et correction des fichiers orphelins.  
- Middleware Helmet + RequestID + timeout transactions 30 s.

#### Corrigé

- Alignement des timestamps ledger ↔ DB.  
- Suppression des doublons dans `ledger_append`.  

---

## [1.0.0] — 15 décembre 2024  

### 🧱 Fondation du Vault (Sprints 1 & 2)

#### Ajouté

- Endpoint `/api/v1/invoices` : ingestion Odoo (Validé → Vaulté).  
- Transaction atomique fichier ↔ base de données.  
- Idempotence par SHA256.  
- Scellement JWS RS256 + Ledger hash-chaîné immuable.  
- Endpoint `/jwks.json` (public key set) et `/ledger/export`.  
- Générateur de clés `cmd/keygen`.  
- 38 tests unitaires initiaux (ingestion + JWS + ledger).

---

## 🧾 Notes

- Ce changelog reflète les livrables certifiés après validation manuelle CI.  
- Les numéros de commit et tags Git sont enregistrés dans le ledger du Vault (`/api/v1/ledger/export`).  
- Chaque version est signée numériquement (JWS RS256 avec KID `key-2025-Q1`).  

---

💙 *Dédié à Antoine Béranger — pour nous avoir rappelé que chaque histoire mérite son changelog.*

