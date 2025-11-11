# Dorevia Vault — Sprint 5 Review & Technical Documentation

**Version** : v1.3.0  
**Date** : Janvier 2025  
**Statut** : ✅ Complété  
**Auteur** : David Baron — Doreviateam  
**Type** : Documentation technique & pédagogique

---

## Objectif global

Le Sprint 5 de **Dorevia Vault** consolide la phase de production : sécurité, interopérabilité et scalabilité.  
Ce document présente de manière didactique les évolutions majeures introduites en v1.3.0, ainsi que les préconisations pour une exploitation fiable en environnement souverain (France / DOM).

---

## Table des matières

1. Vue d’ensemble  
2. Phase 5.1 — Sécurité & Gestion des clés  
3. Phase 5.2 — Authentification & Autorisation  
4. Phase 5.3 — Interopérabilité  
5. Phase 5.4 — Scalabilité & Performance  
6. Release Notes v1.3.0  
7. Changelog v1.3.0  
8. Configuration complète  
9. Tests & Validation  
10. Préconisations générales  
11. Roadmap post‑Sprint 5  
12. Références & normes

---

## 1. Vue d’ensemble

Le Sprint 5 a marqué la transition de Dorevia Vault vers une **architecture prête pour la conformité Factur‑X / PDP 2026**.  
Les grands axes sont :

- 🔐 Gestion sécurisée des clés via **HashiCorp Vault** et **rotation multi‑KID**
- 🧾 Authentification JWT + API Keys avec **RBAC granulaire**
- 🔄 Validation **Factur‑X EN 16931** et **webhooks Redis asynchrones**
- 📈 Partitionnement Ledger PostgreSQL pour la **scalabilité**
- 🧱 82 tests unitaires validés — couverture > 80 %

> **Préconisation :** chaque Sprint valide une brique du triptyque 3V — *Validé → Vaulté → Vérifiable* —.  
> Sprint 5 ajoute la couche de *sécurité et d’interopérabilité* nécessaire à la conformité documentaire souveraine.

---

## 2. Phase 5.1 — Sécurité & Gestion des clés

Durée : 6 jours — Priorité : haute — Statut : ✅ complété

### Objectif
Intégrer une gestion centralisée des clés cryptographiques, avec rotation planifiée et chiffrement au repos.

### Fonctionnalités principales
- Intégration complète **HashiCorp Vault API v1**
- Rotation **multi‑KID** (clé courante + clé précédente)
- Chiffrement AES‑256‑GCM pour journaux d’audit
- Fallback local sécurisé pour environnements de développement

### Préconisations
> **Préconisation :** planifier la rotation des clés tous les 90 jours et activer l’alerte si Vault devient indisponible.  
> **Bonne pratique :** ne jamais exporter de clé privée ; seules les clés publiques doivent être sauvegardées et auditées.

---

## 3. Phase 5.2 — Authentification & Autorisation

Durée : 5 jours — Priorité : haute — Statut : ✅ complété

### Fonctionnalités
- Authentification **JWT RS256** et **API Keys SHA‑256**
- Rôles : `admin`, `auditor`, `operator`, `viewer`
- Middleware Fiber avec vérification automatique des permissions

### Points clés
- Gestion expiration / révocation des API Keys  
- Vérification des permissions via `RequirePermission()`  
- Endpoints protégés selon rôle (documents, audit, ledger)

> **Préconisation :** imposer HTTPS et durée de vie courte des tokens (< 24 h).  
> Mettre en place une surveillance Prometheus du taux d’échec JWT → tentatives d’accès anormales.

---

## 4. Phase 5.3 — Interopérabilité

Durée : 5 jours — Statut : ✅ complété

### Validation Factur‑X
- Parsing XML UBL 2.1 (EN 16931)  
- Validation montants et cohérence TVA  
- Extraction automatique des métadonnées (ID, date, TVA, totaux)

### Webhooks Redis
- Queue Redis pour envoi asynchrone (`document.vaulted`, `document.verified`)  
- Retry × 5 avec backoff exponentiel (1 s → 5 min)  
- Signature HMAC‑SHA256 pour intégrité

> **Préconisation :** activer les webhooks uniquement sur endpoints HTTPS et vérifier la signature HMAC côté client.  
> Documenter les événements déclenchés dans chaque intégration Odoo ou externe.

---

## 5. Phase 5.4 — Scalabilité & Performance

Durée : 4 jours — Statut : ✅ complété

### Objectif
Optimiser le Ledger pour des volumes > 100 000 entrées / an.

### Réalisations
- Table Ledger partitionnée mensuellement (`ledger_YYYY_MM`)  
- Indexation optimisée : `timestamp`, `document_id`, `hash`  
- Maintenance automatique (VACUUM / ANALYZE)

### Résultats
| Volume | Sans partition | Avec partition | Gain |
|:--|--:|--:|--:|
| 10k | 50 ms | 45 ms |  +10 % |
| 100k | 500 ms | 200 ms |  +60 % |
| 1M | 5 s | 0.8 s |  +84 % |

> **Préconisation :** activer le partitionnement dès que le Ledger dépasse 100 000 entrées / an.  
> Surveiller la croissance des partitions via Prometheus (taille > 500 Mo → alerte).

---

## 6. Release Notes v1.3.0

- Gestion sécurisée des clés (Vault / rotation multi‑KID)  
- Authentification JWT + API Keys + RBAC complet  
- Validation Factur‑X et webhooks Redis  
- Partitionnement Ledger + optimisations DB  
- 82 tests unitaires (100 % succès)  
- Documentation complète (6 fichiers .md ajoutés)

---

## 7. Changelog v1.3.0

**Ajouts :**  
- Modules : `internal/crypto/vault.go`, `crypto/rotation.go`, `audit/encrypt.go`, `auth/auth.go`, `auth/rbac.go`, `auth/middleware.go`, `validation/facturx.go`, `webhooks/config.go`, `webhooks/manager.go`, `webhooks/queue.go`, `webhooks/worker.go`, `ledger/partition.go`, `ledger/optimize.go`
- 13 nouveaux modules, 82 tests unitaires  
- Nouvelles vars ENV : Vault, Auth, Factur‑X, Webhooks

**Modifications :**  
- Endpoints `/api/v1/invoices`, `/audit/export`, `/ledger/export` désormais sécurisés  
- `/api/v1/invoices` : validation Factur‑X intégrée  
- `/api/v1/ledger/verify` : émission webhook `document.verified`

---

## 8. Configuration complète

Exemple minimal :

```bash
# Authentification
AUTH_ENABLED=true
AUTH_JWT_ENABLED=true
AUTH_APIKEY_ENABLED=true
AUTH_JWT_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/jwt-public.pem

# HashiCorp Vault (optionnel)
VAULT_ENABLED=true
VAULT_ADDR=https://vault.example.com:8200
VAULT_TOKEN=hvs.xxxxx
VAULT_KEY_PATH=secret/data/dorevia/keys

# Factur-X
FACTURX_VALIDATION_ENABLED=true
FACTURX_VALIDATION_REQUIRED=false

# Webhooks
WEBHOOKS_ENABLED=true
WEBHOOKS_REDIS_URL=redis://localhost:6379/0
WEBHOOKS_SECRET_KEY=$(openssl rand -hex 32)
WEBHOOKS_WORKERS=3
WEBHOOKS_URLS=document.vaulted:https://example.com/webhook/vaulted|document.verified:https://example.com/webhook/verified
```

> **Préconisation :** stocker ces variables dans un fichier `.env` ou utiliser `source setup_env.sh` pour cohérence des déploiements.

---

## 9. Tests & Validation

| Phase | Tests | Résultat |
|:------|:------|:---------|
| 5.1 | 24 | ✅ PASS |
| 5.2 | 25 | ✅ PASS |
| 5.3 | 23 | ✅ PASS |
| 5.4 | 10 | ✅ PASS |
| **Total** | **82** | **✅ 100 % succès** |

- Couverture moyenne : > 80 %  
- Latence P95 : < 200 ms  
- 0 vulnérabilité critique détectée

---

## 10. Préconisations générales

1. **Vault / KMS** : activé en production, rotation trimestrielle obligatoire.  
2. **Auth** : clé JWT 2048 bits ; tokens < 24 h ; surveiller logs 401.  
3. **Interop** : activer validation Factur‑X obligatoire à partir de 2026.  
4. **Scalabilité** : activer partitionnement Ledger > 100 k entrées/an.  
5. **Monitoring** : export Prometheus obligatoire pour supervision.

---

## 11. Roadmap post‑Sprint 5

### Sprint 6 (prévision)
- Support AWS KMS (alternative Vault)
- Cache Redis → accélération lectures Ledger
- Gestion dynamique des API Keys via DB
- Tests de charge automatisés (> 10 k documents)
- API GraphQL et mode multi‑tenant

---

## 12. Références & normes

- **RFC 7515 / 7517 / 7519** : JWS, JWK, JWT  
- **EN 16931** : Factur‑X (UBL 2.1)  
- **AES‑256‑GCM** : NIST SP 800‑38D  
- **OWASP** : Authentication / RBAC best practices  
- **PostgreSQL 14+** : Partition by Range  
- **HashiCorp Vault API v1** : Key Management

---

© 2025 Doreviateam — Projet Dorevia Vault v1.3.0  
Document : `Dorevia_Vault_Sprint5_Review.md`  
Licence MIT
