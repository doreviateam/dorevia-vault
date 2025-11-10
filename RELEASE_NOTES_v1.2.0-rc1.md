# 🚀 Dorevia Vault v1.2.0-rc1 — « Audit & Conformité »

**Date de publication :** 28 février 2025  

**Auteur :** Doreviateam (David Baron)  

**Version :** v1.2.0-rc1  

**État :** Release candidate — stable pour production interne  

---

## 🌟 Aperçu général

Cette version marque la **fin du Sprint 4** et l'entrée de Dorevia Vault dans une phase de **traçabilité certifiable**, alignée sur les futures exigences PDP/PPF 2026.

Elle introduit la **génération automatique de rapports d'audit consolidés**, signés électroniquement, et renforce la chaîne d'intégrité à tous les niveaux du système.

---

## 🧩 Nouveautés majeures

### 1. Rapports d'audit consolidés

- Génération **mensuelle ou trimestrielle** à partir des logs, métriques et données SQL.  
- Export multi-format : **JSON**, **CSV**, **PDF 8 pages**.  
- **Signature JWS RS256** et QR code du hash SHA256 inclus dans les PDF.  
- Résumé exécutif, statistiques, erreurs, performances, ledger, réconciliations et métadonnées.

### 2. CLI `audit`

- Nouvelle commande autonome `cmd/audit/main.go`.  
- Flags : `--period`, `--format`, `--sign`, `--output`, `--database-url`, `--audit-dir`.  
- Permet la génération manuelle ou automatisée de rapports planifiés.

### 3. Observabilité étendue

- 17 métriques Prometheus : 11 métier + 6 système (CPU, RAM, disque, latence ledger).  
- Logs audités en format JSONL signés quotidiennement.  
- Export complet `/audit/export` et `/audit/dates`.

---

## 🧱 Améliorations techniques

- **Refonte partielle du module `internal/audit/`** : logs, export, signature, rapport, PDF.  
- **Harmonisation des métriques Prometheus** (nommage + seuils d'alerte).  
- **Health Check détaillé** : ajout vérification ledger et stockage.  
- **Refactoring CLI** : gestion robuste des flags et périodes.  

---

## 🧩 Correctifs

- Suppression des blocages aléatoires sur l'écriture ledger lors de pics I/O.  
- Rotation automatique des logs audit stabilisée (intervalle < 24 h).  
- Ajustement calcul médian `document_size` + gestion JSON invalide.  

---

## 🧪 Tests & Qualité

| Type | Tests | Résultats |
|:--|:--:|:--:|
| Unitaires | 145 tests (dont 39 nouveaux) | ✅ 100 % |
| Intégration | Ledger + Audit | ✅ Validé |
| Linter | `golangci-lint` | ✅ 0 erreur |
| Performance | Rapport 30 jours < 10 s | ✅ |

---

## ⚙️ Documentation mise à jour

- `docs/audit_export_spec.md` — format et structure des rapports  
- `docs/SPRINT4_PHASE4.4_PLAN.md` — plan détaillé du sprint  
- `docs/Dorevia_Vault_Sprint4.md` — plan global Sprint 4  
- `README.md` — mise à jour avec section rapports d'audit  

---

## 🔐 Conformité & sécurité

- Signature JWS RS256 avec KID `key-2025-Q1`.  
- Export JWKS public : `/jwks.json`.  
- Ledger hash-chaîné vérifiable via `/api/v1/ledger/verify/:id`.  
- Permissions Unix strictes (600 / 644).  
- Mode dégradé possible si `JWS_REQUIRED=false`.  

---

## 🔮 Prochaines étapes — Sprint 5

- Intégration **HSM/Vault** pour gestion sécurisée des clés privées.  
- Webhooks asynchrones (Queue Redis).  
- Validation **Factur-X (EN 16931)**.  
- Rotation multi-KID (JWKS dynamique).  

---

## 🧾 Synthèse de version

| Élément | Statut |
|:--|:--|
| **Phase couverte** | 4.4 – Audit & Conformité |
| **Durée sprint** | 4 jours (J1 → J4) |
| **Livrables** | 8 modules, 3 binaries, 2 docs principales |
| **Version tag Git** | `v1.2.0-rc1` |
| **Hash Ledger** | Disponible via `/api/v1/ledger/export` |
| **Compatibilité** | PostgreSQL ≥ 14, Go 1.23+, Fiber v2.52+ |

---

## 💬 Remerciements

- **Équipe Doreviateam** pour la persévérance dans la conception du cœur Vault.  
- **Veréna & Ethel** pour la rigueur des tests terrain.  
- **Antoine Béranger**, ex-collègue chez Enki Technologies —  
  *pour nous avoir rappelé que chaque histoire mérite son changelog.*

---

© 2025 Doreviateam | Projet Dorevia Vault v1.2.0-rc1  

Licence MIT — Système auditable et souverain 🇫🇷

