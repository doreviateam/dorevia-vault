# 📜 CHANGELOG — Dorevia Vault

Ce fichier suit la convention [Keep a Changelog](https://keepachangelog.com/fr/1.0.0/),  
et respecte la sémantique de versionnage : `MAJEURE.MINEURE.PATCH`.

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

