# 🔍 Analyse de la Checklist Technique Phase 3
## Comparaison avec l'Avis d'Expert

**Date** : Janvier 2025  
**Document analysé** : `demarche_revision_concept.md`  
**Référence** : `AVIS_EXPERT_PHASE3.md`

---

## 📊 Synthèse

La checklist est **bien structurée et actionnable**, mais elle manque de **précautions critiques** identifiées dans l'avis d'expert. Elle suit une approche séquentielle logique mais **sous-estime certains risques techniques**.

**Verdict** : ✅ **Bonne base** mais nécessite des **ajustements critiques** avant implémentation.

---

## ✅ Points forts de la checklist

### 1. Structure claire et progressive

- ✅ Ordre logique (modèle → endpoint → intégration → scellement)
- ✅ Priorités bien définies (🔴 🟠 🟡 🟢)
- ✅ Séparation MVP / Scellement / Confort

### 2. Détails techniques précis

- ✅ Champs exacts à ajouter
- ✅ Structure SQL du ledger
- ✅ Format payload JSON
- ✅ Endpoints à créer

### 3. Approche incrémentale

- ✅ MVP d'abord (étapes 1-3)
- ✅ Enrichissement progressif (étapes 4-7)

---

## ⚠️ Points manquants critiques

### 🔴 CRITIQUE : Transactions atomiques

**Manquant dans la checklist** : Pas de mention explicite des transactions.

**Problème** : L'étape 5 mentionne "Insertion transactionnelle" mais c'est insuffisant.

**Recommandation** : Ajouter une section dédiée :

```markdown
## 🔒 1bis. Gestion des transactions atomiques

- [ ] Toutes les opérations critiques dans une transaction PostgreSQL
- [ ] Pattern : BEGIN → stockage fichier → INSERT documents → INSERT ledger → COMMIT
- [ ] Rollback automatique en cas d'erreur
- [ ] Nettoyage des fichiers orphelins si échec DB
- [ ] Tests de cohérence (fichier sans DB, DB sans fichier)
```

### 🔴 CRITIQUE : Sécurité des clés JWS

**Manquant dans la checklist** : Stockage des clés dans `/opt/dorevia-vault/keys/` est **dangereux**.

**Problème** : Clés privées en clair sur le système de fichiers.

**Recommandation** : Ajouter une section :

```markdown
## 🔐 4bis. Sécurité des clés JWS

- [ ] Clés privées **hors du code source**
- [ ] Variables d'environnement chiffrées ou HSM
- [ ] Permissions restrictives (600) sur les fichiers de clés
- [ ] Rotation des clés (kid avec timestamp)
- [ ] Backup sécurisé des clés privées
- [ ] JWKS endpoint public (`/jwks.json`)
```

### 🟡 IMPORTANT : Performance du ledger

**Manquant dans la checklist** : Pas d'optimisation prévue.

**Problème** : Le ledger va croître indéfiniment sans partitionnement.

**Recommandation** : Ajouter dans l'étape 5 :

```markdown
- [ ] Partitionnement mensuel du ledger
- [ ] Index sur `document_id`, `timestamp`, `hash`
- [ ] Stratégie d'archivage (après X années)
- [ ] Tests de performance (10K+ insertions)
```

### 🟡 IMPORTANT : Queue de webhooks

**Manquant dans la checklist** : Webhooks directs sans queue.

**Problème** : Si Odoo est indisponible, perte de notification.

**Recommandation** : Modifier l'étape 6 :

```markdown
## 📣 6. Webhook retour Odoo (avec queue)

- [ ] Queue de messages (Redis Streams ou PostgreSQL LISTEN/NOTIFY)
- [ ] Worker asynchrone pour envoi webhooks
- [ ] Retry avec backoff exponentiel
- [ ] Dead Letter Queue pour échecs définitifs
- [ ] Signature HMAC SHA256 (`X-Vault-Signature`)
- [ ] Endpoint de statut pour polling de secours
```

### 🟡 IMPORTANT : Validation et gestion d'erreurs

**Manquant dans la checklist** : Validation Factur-X et gestion d'erreurs détaillée.

**Recommandation** : Ajouter une étape :

```markdown
## 🔍 2bis. Validation et extraction Factur-X

- [ ] Extraction XML depuis PDF Factur-X
- [ ] Validation basique (structure, champs obligatoires)
- [ ] Extraction automatique des métadonnées
- [ ] Gestion des erreurs de validation (retour détaillé)
- [ ] Mode strict/lenient selon environnement
```

### 🟢 MINEUR : Monitoring et observabilité

**Présent mais insuffisant** : L'étape 7 mentionne Prometheus mais c'est minimal.

**Recommandation** : Enrichir :

```markdown
## 🧪 7. Tests, monitoring & observabilité

- [ ] Tests unitaires pour `/api/v1/invoices`
- [ ] Tests idempotence (doublons)
- [ ] Tests JWS et ledger
- [ ] Tests de transactions (rollback, cohérence)
- [ ] Tests de charge (ledger 10K+ entrées)
- [ ] Métriques Prometheus :
  - `documents_vaulted_total`
  - `vault_errors_total`
  - `vault_duration_seconds`
  - `ledger_size_total`
  - `jws_generation_duration_seconds`
- [ ] Tracing OpenTelemetry (flux complet)
- [ ] Alerting (échecs critiques, performance)
```

---

## 🔄 Ajustements recommandés à la checklist

### 1. Réorganiser les priorités

**Actuel** :
- Étape 3 (Lier Odoo) : 🔴 Haute

**Problème** : Lier Odoo nécessite d'abord que l'endpoint soit prêt et testé.

**Recommandation** :
- Étape 1 : 🔴 Haute (modèle + migration)
- Étape 2 : 🔴 Haute (endpoint `/api/v1/invoices`)
- **Étape 2bis** : 🟡 Moyenne (validation Factur-X - optionnel pour MVP)
- Étape 3 : 🟡 Moyenne (Lier Odoo - après tests endpoint)
- Étape 4 : 🟠 Moyenne-Haute (JWS)
- Étape 5 : 🟠 Moyenne-Haute (Ledger)
- Étape 6 : 🟡 Moyenne (Webhooks)
- Étape 7 : 🟢 Basse (Monitoring)

### 2. Ajouter une étape de tests intermédiaires

**Recommandation** : Tests après chaque étape majeure.

```markdown
## 🧪 Tests intermédiaires

- [ ] Après étape 1 : Tests migration DB
- [ ] Après étape 2 : Tests endpoint `/api/v1/invoices`
- [ ] Après étape 4 : Tests JWS (génération, vérification)
- [ ] Après étape 5 : Tests ledger (hash-chaîné, cohérence)
```

### 3. Préciser l'ordre d'intégration JWS/Ledger

**Problème** : L'ordre actuel (JWS puis Ledger) est correct, mais il faut préciser l'intégration.

**Recommandation** : Clarifier que JWS et Ledger doivent être dans la **même transaction** :

```markdown
## 🔐 4-5. Scellement (JWS + Ledger)

Ordre d'exécution dans une transaction :
1. Stocker fichier
2. INSERT documents
3. Générer JWS
4. Calculer hash ledger
5. INSERT ledger
6. COMMIT (tout ou rien)
```

---

## 📋 Checklist améliorée (recommandée)

### Étape 1 : Modèle & Base de données 🔴

- [x] Extension modèle Document (champs Odoo, PDP, métadonnées)
- [x] Migration SQL avec tous les nouveaux champs
- [x] **Index sur `odoo_id`, `sha256_hex`, `dispatch_status`**
- [x] **Contraintes d'intégrité** (foreign keys si nécessaire)

### Étape 2 : Endpoint `/api/v1/invoices` 🔴

- [x] Handler `internal/handlers/invoices.go`
- [x] Accepte JSON (base64) **ET** multipart
- [x] Validation payload complète
- [x] Calcul SHA256
- [x] **Idempotence** (retour document existant si même hash)
- [x] **Gestion d'erreurs détaillée**
- [x] Tests unitaires

### Étape 2bis : Validation Factur-X 🟡 (Optionnel MVP)

- [ ] Extraction XML depuis PDF
- [ ] Parsing basique métadonnées
- [ ] Validation structure (sans XSD pour MVP)
- [ ] Tests

### Étape 3 : Intégration Odoo 🟡

- [ ] Configuration webhook Odoo → Vault
- [ ] Tests avec Odoo (ou mock)
- [ ] Documentation intégration

### Étape 4 : JWS 🔴

- [ ] Package `internal/crypto/jws.go`
- [ ] **Génération clés RSA (hors code source)**
- [ ] Génération JWS avec payload `{doc_id, sha256, timestamp}`
- [ ] Stockage dans `evidence_jws`
- [ ] **Endpoint `/jwks.json`** pour vérification publique
- [ ] **Rotation des clés** (kid avec timestamp)
- [ ] Tests (génération, vérification)

### Étape 5 : Ledger 🔴

- [ ] Table `ledger` avec structure complète
- [ ] **Partitionnement mensuel** (prévoir croissance)
- [ ] **Index optimisés** (`document_id`, `timestamp`, `hash`)
- [ ] Fonction `AppendLedger()` avec calcul hash-chaîné
- [ ] **Transaction atomique** (documents + ledger)
- [ ] Tests (cohérence, performance)

### Étape 6 : Webhooks Odoo 🟡

- [ ] **Queue de messages** (Redis ou PostgreSQL LISTEN/NOTIFY)
- [ ] Worker asynchrone
- [ ] Payload avec signature HMAC
- [ ] Retry avec backoff exponentiel
- [ ] Dead Letter Queue
- [ ] Tests (retry, échecs)

### Étape 7 : Monitoring & Observabilité 🟢

- [ ] Métriques Prometheus complètes
- [ ] Tracing OpenTelemetry
- [ ] Alerting sur échecs critiques
- [ ] Dashboard Grafana (optionnel)
- [ ] Tests de charge

---

## 🎯 Recommandations finales

### Priorités absolues à ajouter

1. **🔴 Transactions atomiques** : Section dédiée avec exemples
2. **🔴 Sécurité des clés** : Ne pas stocker en clair sur disque
3. **🟡 Performance ledger** : Partitionnement et index dès le départ
4. **🟡 Queue de webhooks** : Ne pas appeler Odoo directement
5. **🟡 Tests intermédiaires** : Après chaque étape majeure

### Ordre d'exécution révisé

**Sprint 1 (2 semaines) - MVP** :
1. ✅ Modèle + Migration
2. ✅ Endpoint `/api/v1/invoices`
3. ✅ Tests
4. ⚠️ **Intégration Odoo** (après validation endpoint)

**Sprint 2 (2 semaines) - Scellement** :
1. ✅ JWS (avec sécurité des clés)
2. ✅ Ledger (avec transactions atomiques)
3. ✅ Tests

**Sprint 3 (2 semaines) - Production** :
1. ✅ Queue de webhooks
2. ✅ Monitoring
3. ✅ Optimisations

---

## 📝 Conclusion

La checklist est une **bonne base** mais nécessite des **ajustements critiques** :

- ✅ Structure et ordre : **Bon**
- ⚠️ Transactions atomiques : **À ajouter explicitement**
- ⚠️ Sécurité des clés : **À améliorer**
- ⚠️ Performance ledger : **À prévoir dès le départ**
- ⚠️ Queue de webhooks : **À modifier**

**Recommandation** : Utiliser cette checklist **en complément** de l'avis d'expert pour avoir une vision complète des risques et des précautions à prendre.

---

**Document créé le** : Janvier 2025  
**Version** : 1.0

