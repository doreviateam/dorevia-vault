# 📋 Plan Détaillé — Sprint 5 : Sécurité & Interopérabilité

**Version** : v1.3-dev (planification)  
**Date de démarrage prévue** : Mars 2025  
**Responsable** : Doreviateam (David Baron)  
**Statut** : 🟡 Planification  
**Durée prévue** : **20 jours ouvrés** (4 semaines)

---

## 🎯 Objectif Global

Renforcer la **sécurité**, l'**interopérabilité** et la **scalabilité** de Dorevia Vault pour répondre aux besoins de production à grande échelle et aux exigences de conformité avancées.

---

## 📊 Vue d'Ensemble

### Contexte

Le Sprint 5 s'appuie sur les fondations solides des Sprints 1-4 :
- ✅ MVP fonctionnel (Sprint 1)
- ✅ Documents vérifiables (Sprint 2)
- ✅ Supervision & réconciliation (Sprint 3)
- ✅ Observabilité & auditabilité (Sprint 4)

### Objectifs Concrets

1. **Sécurité renforcée** → HSM/Vault pour gestion clés, authentification/autorisation
2. **Interopérabilité** → Validation Factur-X, webhooks asynchrones
3. **Scalabilité** → Rotation multi-KID, partitionnement ledger
4. **Conformité** → Chiffrement au repos, audit renforcé

---

## 🏗️ Architecture Technique

### Phases du Sprint 5

```
Sprint 5
├── Phase 5.1 : Sécurité & Gestion des Clés (6 jours)
│   ├── Intégration HSM/Vault
│   ├── Rotation multi-KID
│   └── Chiffrement au repos
│
├── Phase 5.2 : Authentification & Autorisation (5 jours)
│   ├── JWT / API Keys
│   ├── RBAC (Role-Based Access Control)
│   └── Protection endpoints
│
├── Phase 5.3 : Interopérabilité (5 jours)
│   ├── Validation Factur-X (EN 16931)
│   ├── Webhooks asynchrones (Redis Queue)
│   └── Intégrations externes
│
└── Phase 5.4 : Scalabilité & Performance (4 jours)
    ├── Partitionnement Ledger
    ├── Optimisations base de données
    └── Tests de charge
```

---

## 📦 Phase 5.1 : Sécurité & Gestion des Clés

**Durée** : 6 jours  
**Priorité** : 🔴 Haute

### Objectifs

1. Intégrer HSM/Vault pour gestion sécurisée des clés privées
2. Implémenter rotation multi-KID pour JWKS
3. Ajouter chiffrement au repos pour logs d'audit

### Livrables

#### J1-J2 : Intégration HSM/Vault

**Module** : `internal/crypto/vault.go`

- Support HashiCorp Vault (API v1)
- Support AWS KMS (optionnel)
- Abstraction `KeyManager` interface
- Fallback vers fichiers locaux si Vault indisponible
- Configuration via variables d'environnement

**Configuration** :
```bash
VAULT_ENABLED=true
VAULT_ADDR=https://vault.example.com:8200
VAULT_TOKEN=hvs.xxxxx
VAULT_MOUNT_PATH=secret/dorevia-vault
VAULT_KEY_PATH=keys/jws
```

**Tests** : 10+ tests unitaires

---

#### J3-J4 : Rotation multi-KID

**Module** : `internal/crypto/rotation.go`

- Gestion multiple KID simultanés
- Rotation automatique (cron ou manuel)
- JWKS dynamique avec plusieurs clés
- Support période de transition (2 clés actives)
- Endpoint `/jwks.json` avec toutes les clés valides

**Structure** :
```go
type KeyRotation struct {
    CurrentKID string
    PreviousKID string
    NextRotationDate time.Time
    Keys map[string]*KeyPair
}
```

**Tests** : 8+ tests unitaires

---

#### J5-J6 : Chiffrement au repos

**Module** : `internal/audit/encrypt.go`

- Chiffrement AES-256-GCM pour logs d'audit
- Clés de chiffrement depuis HSM/Vault
- Décryptage à la volée pour export
- Intégration transparente avec `audit/log.go`

**Tests** : 6+ tests unitaires

---

## 🔐 Phase 5.2 : Authentification & Autorisation

**Durée** : 5 jours  
**Priorité** : 🔴 Haute

### Objectifs

1. Implémenter authentification JWT ou API Keys
2. Système RBAC (Role-Based Access Control)
3. Protéger endpoints sensibles

### Livrables

#### J7-J8 : Authentification

**Module** : `internal/auth/`

- Support JWT (RS256) et API Keys
- Middleware d'authentification
- Validation tokens
- Gestion sessions (optionnel)

**Endpoints protégés** :
- `/audit/export` → Requiert rôle `auditor`
- `/api/v1/ledger/export` → Requiert rôle `admin`
- `/api/v1/invoices` → Requiert rôle `operator`

**Tests** : 12+ tests unitaires

---

#### J9-J10 : Autorisation RBAC

**Module** : `internal/auth/rbac.go`

- Rôles : `admin`, `auditor`, `operator`, `viewer`
- Permissions par endpoint
- Middleware d'autorisation
- Configuration via config ou DB

**Tests** : 8+ tests unitaires

---

#### J11 : Protection endpoints

- Intégration middleware dans routes
- Tests d'intégration
- Documentation

---

## 🔗 Phase 5.3 : Interopérabilité

**Durée** : 5 jours  
**Priorité** : 🟡 Moyenne

### Objectifs

1. Validation Factur-X (EN 16931)
2. Webhooks asynchrones avec Redis Queue
3. Intégrations externes

### Livrables

#### J12-J13 : Validation Factur-X

**Module** : `internal/validation/facturx.go`

- Parsing XML Factur-X
- Validation XSD (EN 16931)
- Extraction métadonnées
- Intégration dans `/api/v1/invoices`

**Dépendances** :
- `github.com/lestrrat-go/libxml2` (parsing XML)
- Schémas XSD EN 16931

**Tests** : 10+ tests unitaires

---

#### J14-J15 : Webhooks asynchrones

**Module** : `internal/webhooks/`

- Queue Redis pour webhooks
- Workers asynchrones
- Retry avec backoff exponentiel
- Configuration webhooks par événement

**Événements** :
- `document.vaulted`
- `document.verified`
- `ledger.appended`
- `error.critical`

**Tests** : 8+ tests unitaires

---

#### J16 : Intégrations externes

- Documentation API webhooks
- Exemples d'intégration
- Tests d'intégration

---

## 📈 Phase 5.4 : Scalabilité & Performance

**Durée** : 4 jours  
**Priorité** : 🟢 Basse (si volume < 100k/an)

### Objectifs

1. Partitionnement Ledger (si nécessaire)
2. Optimisations base de données
3. Tests de charge

### Livrables

#### J17-J18 : Partitionnement Ledger

**Module** : `internal/ledger/partition.go`

- Partitions mensuelles automatiques
- Migration données existantes
- Requêtes transparentes (union all)
- Maintenance automatique

**Condition** : Si volume > 100k entrées/an

**Tests** : 6+ tests unitaires

---

#### J19 : Optimisations

- Index base de données
- Requêtes optimisées
- Cache Redis (optionnel)

---

#### J20 : Tests de charge

- Tests avec 10k+ documents
- Benchmarks performance
- Documentation résultats

---

## 🧪 Tests & Validation

### Tests Unitaires

| Phase | Tests prévus | Couverture |
|:------|:-------------|:-----------|
| 5.1 | 24 tests | HSM/Vault, rotation, chiffrement |
| 5.2 | 20 tests | Auth, RBAC, middleware |
| 5.3 | 18 tests | Factur-X, webhooks |
| 5.4 | 6 tests | Partitionnement, optimisations |
| **Total** | **68 tests** | **100% réussite attendue** |

### Tests d'Intégration

- Intégration HashiCorp Vault réel
- Tests avec Redis Queue
- Validation Factur-X avec fichiers réels
- Tests de charge (10k+ documents)

---

## 📚 Documentation

### Documents à créer

1. `docs/SPRINT5_PLAN.md` — Ce document
2. `docs/security_vault_spec.md` — Spécification HSM/Vault
3. `docs/auth_rbac_spec.md` — Spécification authentification/autorisation
4. `docs/facturx_validation_spec.md` — Spécification validation Factur-X
5. `docs/webhooks_spec.md` — Spécification webhooks asynchrones
6. `docs/partitioning_spec.md` — Spécification partitionnement ledger

### Mise à jour

- `README.md` — Section Sprint 5
- `CHANGELOG.md` — Entrée v1.3.0
- `RELEASE_NOTES_v1.3.0.md` — Notes de version

---

## 🔧 Dépendances & Prérequis

### Nouvelles dépendances

```go
// HSM/Vault
github.com/hashicorp/vault/api v1.13.0

// Redis Queue
github.com/go-redis/redis/v8 v8.11.5

// Validation XML
github.com/lestrrat-go/libxml2 v0.0.0-2023100101...

// Chiffrement
golang.org/x/crypto v0.17.0
```

### Infrastructure requise

- HashiCorp Vault (ou AWS KMS)
- Redis (pour webhooks queue)
- PostgreSQL 14+ (pour partitionnement)

---

## 📊 Métriques de Succès

| Métrique | Cible |
|:---------|:------|
| **Tests unitaires** | 68 tests, 100% réussite |
| **Couverture code** | ≥ 85% |
| **Performance** | Latence < 200ms (P95) |
| **Sécurité** | 0 vulnérabilité critique |
| **Documentation** | 6 documents créés |

---

## 🚨 Risques & Mitigation

| Risque | Impact | Probabilité | Mitigation |
|:-------|:-------|:------------|:-----------|
| Complexité HSM/Vault | 🔴 Haute | 🟡 Moyenne | Abstraction + fallback fichiers |
| Performance partitionnement | 🟡 Moyenne | 🟢 Basse | Tests de charge précoces |
| Validation Factur-X complexe | 🟡 Moyenne | 🟡 Moyenne | Parser XML robuste + tests |
| Redis indisponible | 🟡 Moyenne | 🟢 Basse | Mode dégradé (webhooks synchrones) |

---

## 🛣️ Roadmap Post-Sprint 5

### Sprint 6 (Optionnel)

- Archivage long terme (NF525 / MinIO / S3)
- Multi-tenant
- API GraphQL
- Dashboard web

---

## 📝 Checklist de Validation

### Phase 5.1
- [ ] HSM/Vault intégré et testé
- [ ] Rotation multi-KID fonctionnelle
- [ ] Chiffrement au repos opérationnel
- [ ] Tests unitaires (24 tests)

### Phase 5.2
- [ ] Authentification JWT/API Keys
- [ ] RBAC implémenté
- [ ] Endpoints protégés
- [ ] Tests unitaires (20 tests)

### Phase 5.3
- [ ] Validation Factur-X fonctionnelle
- [ ] Webhooks asynchrones avec Redis
- [ ] Documentation intégrations
- [ ] Tests unitaires (18 tests)

### Phase 5.4
- [ ] Partitionnement ledger (si nécessaire)
- [ ] Optimisations base de données
- [ ] Tests de charge validés
- [ ] Tests unitaires (6 tests)

---

## 💬 Notes

- **Priorisation** : Phases 5.1 et 5.2 sont critiques pour production
- **Phase 5.4** : Optionnelle si volume < 100k/an
- **Flexibilité** : Chaque phase peut être ajustée selon besoins

---

**Document créé le** : Février 2025  
**Version** : 1.0  
**Auteur** : Doreviateam

© 2025 Doreviateam | Projet Dorevia Vault — Sprint 5 Planification

