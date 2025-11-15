# 🔍 Analyse Expert — Document `Dorevia_Vault_Certification_Roadmap.md`

**Date** : Janvier 2025  
**Analyseur** : Expert technique Dorevia Vault  
**Document analysé** : `docs/Dorevia_Vault_Certification_Roadmap.md`  
**Version projet** : v1.3.0 (Sprint 5 complété)

---

## 📋 Résumé Exécutif

Le document `Dorevia_Vault_Certification_Roadmap.md` présente une **roadmap stratégique de certification** pour Dorevia Vault, couvrant trois axes majeurs : **NF525** (conformité POS), **PDP Ready 2026** (facturation électronique), et **eIDAS/ANSSI** (signature qualifiée). Il définit un plan d'action structuré avec jalons temporels jusqu'en 2027.

**Verdict global** : ✅ **Document stratégique ambitieux et bien structuré**, avec des jalons réalistes et des opportunités d'enrichissement technique.

**Note** : **8.9/10** — Vision claire, roadmap cohérente, quelques précisions à apporter.

---

## ✅ Points Forts

### 1. Vision Stratégique Complète

✅ **Positionnement clair**
- Plateforme souveraine certifiable pour PME françaises
- Alignement sur 3 axes réglementaires (NF525, PDP, eIDAS)
- Vision à long terme (2025-2027)

✅ **Cohérence réglementaire**
- Références précises (NF525, Ordonnance 2021-1190, eIDAS, RGPD)
- Correspondance entre exigences et implémentation
- Plan d'audit structuré (interne, externe, conformité, sécurité)

✅ **Architecture technique**
- Mapping clair entre couches techniques et normes
- Standards reconnus (RFC, ISO, eIDAS)
- Technologies appropriées (JWS, PostgreSQL, TLS 1.3)

### 2. Structure et Planification

✅ **Roadmap temporelle**
- Jalons clairs par trimestre/année
- Progression logique (MVP → Audit → Certification)
- Objectifs mesurables et réalistes

✅ **Processus d'audit**
- Types d'audit bien définis (interne, externe, conformité, sécurité)
- Responsabilités claires
- Fréquences appropriées

✅ **Livrables documentaires**
- Liste exhaustive des documents à produire
- Formats standardisés
- Organisation claire

### 3. Alignement Technique

✅ **Technologies mentionnées**
- JWS (RFC 7515) → ✅ Implémenté (Sprint 2)
- Ledger PostgreSQL → ✅ Implémenté (Sprint 2)
- Prometheus / Audit log → ✅ Implémenté (Sprint 3-4)
- TLS 1.3 / HTTPS → ✅ Implémenté (Caddy)

---

## ⚠️ Points à Améliorer

### 1. Précisions Techniques

⚠️ **PDP Gateway**
- Mention `/api/v1/pdp/push` → À vérifier si c'est l'endpoint prévu
- **Recommandation** : Préciser l'API exacte et le format de données

⚠️ **Horodatage eIDAS**
- Mention `/internal/tsp` → À détailler l'intégration TSP
- **Recommandation** : Préciser le fournisseur TSP et le format d'horodatage

⚠️ **HSM local**
- Mention "YubiHSM / SoftHSM" → À préciser le choix et l'intégration
- **Recommandation** : Détailler l'architecture HSM et la gestion des clés

### 2. Conformité NF525

⚠️ **Preuve technique**
- Tableau "Preuve technique" → À enrichir avec détails concrets
- **Recommandation** : Ajouter des exemples de logs, hash, signatures

⚠️ **Audit interne**
- Mention "Simulation d'audit Veritas" → À détailler le processus
- **Recommandation** : Ajouter une section "Processus d'audit interne"

### 3. Plan PDP Ready 2026

⚠️ **Mapping Factur-X → UBL 2.1**
- Mention "XSLT + validation XSD" → À préciser l'implémentation
- **Recommandation** : Détailler la transformation et les schémas

⚠️ **Journalisation PPF**
- Mention "ledger_pdp" → À clarifier la différence avec le ledger principal
- **Recommandation** : Préciser la structure et l'utilisation

### 4. Étape eIDAS / ANSSI

⚠️ **Niveaux de certification**
- 3 niveaux définis mais progression non détaillée
- **Recommandation** : Ajouter une section "Progression par niveau"

⚠️ **Certificat QSEAL**
- Mention "Autorité française" → À préciser l'autorité et le processus
- **Recommandation** : Détailler les prérequis et les étapes

### 5. Jalons Temporels

⚠️ **Réalisme des délais**
- Q1 2025 : "MVP certifiable NF525" → À vérifier si c'est encore réalisable
- **Recommandation** : Ajuster les jalons selon l'état actuel (v1.3.0)

⚠️ **Dépendances**
- Jalons non liés aux sprints du projet
- **Recommandation** : Aligner avec la roadmap des sprints

---

## 🔧 Recommandations d'Amélioration

### Priorité Haute

1. **Ajuster les jalons temporels**
   ```markdown
   | Période | Objectif clé | Résultat attendu | Sprint |
   |----------|---------------|------------------|--------|
   | **Q1 2025** | MVP certifiable NF525 | Prototype POS Vaulté | ✅ Sprint 1-5 complétés |
   | **Q2 2025** | Support POS explicite | Endpoint `/api/v1/pos/tickets` | Sprint 6 |
   | **Q3 2025** | Audit interne + documentation | Attestation auto NF525 | Sprint 7 |
   | **Q4 2025** | Gateway PDP opérationnelle | Facture Vault → PDP | Sprint 8 |
   | **Q1 2026** | Audit Bureau Veritas | Label conformité | Sprint 9 |
   | **Q3 2026** | Pré-audit eIDAS | Process certifiable | Sprint 10+ |
   | **2027** | Certification ANSSI / eIDAS QSEAL | Reconnaissance officielle | Sprint 11+ |
   ```

2. **Enrichir la section "Preuve technique NF525"**
   ```markdown
   | Pilier | Objectif | Implémentation Vault | Preuve technique | Exemple |
   |--------|-----------|----------------------|------------------|--------|
   | Inaltérabilité | Empêcher toute modification | SHA-256, hash chain | logs + hash JSON | `{"sha256": "abc123...", "ledger_hash": "def456..."}` |
   | Sécurisation | Signer chaque transaction | JWS signé localement | clé API + signature | `eyJhbGciOiJSUzI1NiIs...` |
   | Conservation | Garder les données 6 à 10 ans | Ledger PostgreSQL | backup / snapshot | Export mensuel signé |
   | Archivage | Clôture mensuelle scellée | Export signé (ZIP + manifest.json) | manifeste JWS | `manifest.json` avec JWS |
   | Traçabilité | Journal complet des actions | Audit log + RequestID | journaux Vault | JSONL signé quotidiennement |
   ```

3. **Détailler le processus d'audit interne**
   ```markdown
   ## 4.1 Processus d'Audit Interne NF525
   
   ### Étapes
   1. **Export des données**
      - Export 3 mois de tickets POS + factures
      - Format : JSON avec hash SHA-256, JWS, ledger hash
      - Compression : ZIP avec manifest.json signé
   
   2. **Vérification de cohérence**
      - Contrôle hash SHA-256 (intégrité fichiers)
      - Vérification JWS (signatures valides)
      - Validation ledger hash-chaîné (traçabilité)
   
   3. **Documentation**
      - Rapport d'audit avec résultats
      - Preuves d'intégrité (JWS, hash)
      - Attestation de conformité
   
   ### Fréquence
   - Mensuel : Export et vérification automatique
   - Trimestriel : Audit complet avec rapport
   - Annuel : Audit externe (Bureau Veritas / LNE)
   ```

### Priorité Moyenne

4. **Ajouter section "Architecture PDP Gateway"**
   ```markdown
   ## 5.1 Architecture PDP Gateway
   
   ### Endpoints
   - `POST /api/v1/pdp/push` : Envoi facture vers PDP
   - `GET /api/v1/pdp/status/:document_id` : Statut transmission
   - `POST /api/v1/pdp/callback` : Réception ACK/REJECT depuis PDP
   
   ### Format de données
   - Facture : Factur-X (UBL 2.1 ou UN/CEFACT CII)
   - Hash : SHA-256 du document
   - JWS : Preuve d'intégrité signée
   - Métadonnées : Invoice number, date, montant, etc.
   
   ### Sécurité
   - Authentification : API Key ou JWT
   - Signature : HMAC-SHA256 des webhooks
   - Retry : Backoff exponentiel (3 tentatives)
   ```

5. **Détailler l'intégration TSP (Horodatage eIDAS)**
   ```markdown
   ## 5.2 Intégration TSP (Trusted Timestamp Provider)
   
   ### Fournisseurs TSP qualifiés
   - Universign (France)
   - Docapost (France)
   - GlobalSign (UE)
   
   ### Architecture
   - Module `/internal/tsp` : Interface TSP
   - Format : RFC 3161 (Time-Stamp Protocol)
   - Intégration : Signature JWS + TSP timestamp
   
   ### Processus
   1. Génération JWS du document
   2. Envoi hash SHA-256 au TSP
   3. Réception timestamp signé
   4. Intégration timestamp dans JWS
   5. Stockage dans ledger
   ```

6. **Enrichir la section "Progression eIDAS par niveau"**
   ```markdown
   ## 6.1 Progression eIDAS par Niveau
   
   ### Niveau 1 : HSM local (Vault certifié)
   - **Objectif** : Intégration HSM pour stockage sécurisé des clés
   - **Technologies** : YubiHSM, SoftHSM, ou HashiCorp Vault
   - **Durée estimée** : 2-3 mois
   - **Livrables** : Module HSM, tests, documentation
   
   ### Niveau 2 : Signature qualifiée eIDAS
   - **Objectif** : Certificat QSEAL pour signatures qualifiées
   - **Autorité** : ANSSI ou autorité qualifiée française
   - **Durée estimée** : 4-6 mois
   - **Livrables** : Certificat QSEAL, intégration, tests
   
   ### Niveau 3 : Audit ANSSI / conformité RGS 2.0
   - **Objectif** : Certification ANSSI pour prestataire de service de confiance
   - **Processus** : ISO 27001 + HDS (Hébergement Données de Santé)
   - **Durée estimée** : 12-18 mois
   - **Livrables** : Certification ANSSI, documentation complète
   ```

### Priorité Basse

7. **Ajouter section "Mapping Factur-X → UBL 2.1"**
   ```markdown
   ## 5.3 Mapping Factur-X → UBL 2.1
   
   ### Transformation
   - **Format source** : Factur-X (PDF/A-3 avec XML embarqué)
   - **Format cible** : UBL 2.1 (XML pur)
   - **Méthode** : XSLT 2.0 ou transformation Go native
   
   ### Schémas de validation
   - **UBL 2.1** : XSD officiel OASIS
   - **EN 16931** : Validation sémantique
   - **Validation** : Avant envoi PDP
   
   ### Exemple de transformation
   ```xml
   <!-- Factur-X -->
   <rsm:CrossIndustryInvoice>
     <rsm:ExchangedDocument>
       <ram:ID>INV-2025-001</ram:ID>
     </rsm:ExchangedDocument>
   </rsm:CrossIndustryInvoice>
   
   <!-- UBL 2.1 -->
   <Invoice>
     <ID>INV-2025-001</ID>
   </Invoice>
   ```
   ```

8. **Ajouter section "Structure ledger_pdp"**
   ```markdown
   ## 5.4 Structure ledger_pdp
   
   ### Différences avec ledger principal
   - **Ledger principal** : Tous les documents (POS + Factures)
   - **ledger_pdp** : Uniquement factures transmises à PDP
   
   ### Champs supplémentaires
   - `pdp_provider` : Nom du PDP agréé
   - `pdp_transmission_id` : ID de transmission PDP
   - `pdp_status` : PENDING|SENT|ACK|REJECTED
   - `pdp_timestamp` : Horodatage transmission
   - `pdp_response` : Réponse PDP (ACK/REJECT avec raison)
   
   ### Utilisation
   - Traçabilité complète des transmissions PDP
   - Réconciliation avec réponses PDP
   - Audit de conformité PPF
   ```

---

## 📊 Analyse de Cohérence avec le Projet

### Alignements Confirmés

✅ **Technologies mentionnées**
- JWS (RFC 7515) → ✅ Implémenté (Sprint 2)
- Ledger PostgreSQL → ✅ Implémenté (Sprint 2)
- Prometheus / Audit log → ✅ Implémenté (Sprint 3-4)
- TLS 1.3 / HTTPS → ✅ Implémenté (Caddy)
- HashiCorp Vault → ✅ Implémenté (Sprint 5 Phase 5.1)

✅ **Fonctionnalités mentionnées**
- Signature JWS → ✅ Implémenté
- Ledger hash-chaîné → ✅ Implémenté
- Audit logs signés → ✅ Implémenté
- Webhooks → ✅ Implémenté (Sprint 5)

### Écarts Identifiés

⚠️ **PDP Gateway**
- Mention `/api/v1/pdp/push` → Non implémenté actuellement
- **Action** : Planifier dans Sprint 6+ (PDP Gateway)

⚠️ **Horodatage eIDAS**
- Mention `/internal/tsp` → Non implémenté actuellement
- **Action** : Planifier dans Sprint 7+ (eIDAS TSP)

⚠️ **HSM local**
- Mention "YubiHSM / SoftHSM" → HashiCorp Vault implémenté, mais pas HSM physique
- **Action** : Considérer HSM physique pour niveau 1 eIDAS

⚠️ **Support POS explicite**
- MVP certifiable NF525 mentionné → Support POS partiel (champ `source` supporte `pos`)
- **Action** : Implémenter endpoint `/api/v1/pos/tickets` pour support complet

---

## 🎯 Recommandations Stratégiques

### Court Terme (Q1-Q2 2025)

1. **Support POS explicite (Sprint 6)**
   - Endpoint `/api/v1/pos/tickets` pour ingestion tickets POS
   - Validation spécifique POS (pas de Factur-X requis)
   - Documentation conformité NF525

2. **Audit interne NF525**
   - Processus d'audit automatisé
   - Export et vérification de cohérence
   - Documentation `/docs/NF525_Attestation_Dorevia.md`

3. **PDP Gateway (Sprint 6-7)**
   - Endpoint `/api/v1/pdp/push` pour transmission PDP
   - Mapping Factur-X → UBL 2.1
   - Journalisation `ledger_pdp`

### Moyen Terme (Q3-Q4 2025)

4. **Horodatage eIDAS (Sprint 7-8)**
   - Intégration TSP qualifié
   - Module `/internal/tsp`
   - Timestamp dans JWS

5. **Audit externe NF525**
   - Préparation audit Bureau Veritas / LNE
   - Tests de conformité
   - Documentation complète

6. **Certification QSEAL (préparation)**
   - Étude des prérequis
   - Sélection autorité qualifiée
   - Plan d'action certification

### Long Terme (2026-2027)

7. **Certification NF525 (Q1 2026)**
   - Audit Bureau Veritas / LNE
   - Label conformité
   - Documentation de certification

8. **Pré-audit eIDAS (Q3 2026)**
   - Processus certifiable
   - Tests de conformité eIDAS
   - Documentation complète

9. **Certification ANSSI / eIDAS QSEAL (2027)**
   - Certification ANSSI
   - Reconnaissance officielle
   - Prestation de service de confiance

---

## 📝 Modifications Suggérées au Document

### Section 3 — Architecture et conformité technique

**Enrichir** :
```markdown
| Couche | Fonction | Norme associée | Statut |
|--------|-----------|----------------|--------|
| API Go (Fiber) | Ingestion, vérification | RFC 9110 (HTTP), RESTful | ✅ Implémenté |
| JWS Signer | Signature et scellement | RFC 7515, eIDAS | ✅ Implémenté (Sprint 2) |
| Ledger PostgreSQL | Traçabilité immuable | ISO 15489-1 (records mgmt) | ✅ Implémenté (Sprint 2) |
| Caddy Proxy HTTPS | Sécurité des échanges | TLS 1.3, HSTS | ✅ Implémenté |
| Prometheus / Audit log | Supervision et preuve d'intégrité | ISO 27001 (auditability) | ✅ Implémenté (Sprint 3-4) |
| HashiCorp Vault | Gestion sécurisée des clés | HSM/Vault | ✅ Implémenté (Sprint 5) |
| TSP (Horodatage eIDAS) | Horodatage qualifié | RFC 3161, eIDAS | 🧩 Planifié (Sprint 7+) |
| HSM Physique | Stockage sécurisé clés | eIDAS, ANSSI | 🧩 Planifié (2026-2027) |
```

### Section 4 — Plan de conformité NF525

**Enrichir** :
```markdown
**Audit interne prévu :**
- Simulation d'audit Veritas : export 3 mois POS + facture → contrôle de cohérence SHA.
- Documentation : `/docs/NF525_Attestation_Dorevia.md`
- **Processus automatisé** : Export mensuel, vérification hash, génération rapport
- **Tests d'inaltérabilité** : Tentatives de modification, vérification détection
- **Validation JWS** : Vérification signatures, rotation clés, intégrité ledger
```

### Section 5 — Plan PDP Ready 2026

**Enrichir** :
```markdown
| Étape | Description | Livrable | Statut |
|--------|-------------|----------|--------|
| PDP Gateway | API d'interconnexion Vault ↔ PDP (facture + hash) | `/api/v1/pdp/push` | 🧩 Planifié (Sprint 6) |
| Journalisation PPF | Ledger spécifique PDP | `ledger_pdp` | 🧩 Planifié (Sprint 6) |
| Mapping Factur-X → UBL 2.1 | Transformation native | XSLT + validation XSD | 🧩 Planifié (Sprint 6) |
| Horodatage eIDAS | Interfaçage TSP (trusted timestamp provider) | `/internal/tsp` | 🧩 Planifié (Sprint 7) |
| Preuve d'origine | Archivage JWS + certificat QSEAL | `/proofs/sha` | 🧩 Planifié (2026) |
```

### Section 9 — Jalons temporels

**Ajuster** :
```markdown
| Période | Objectif clé | Résultat attendu | Sprint | Statut |
|----------|---------------|------------------|--------|--------|
| **Q1 2025** | MVP certifiable NF525 | Prototype POS Vaulté | Sprint 1-5 | ✅ Complété (v1.3.0) |
| **Q2 2025** | Support POS explicite | Endpoint `/api/v1/pos/tickets` | Sprint 6 | 🧩 En cours |
| **Q3 2025** | Audit interne + documentation | Attestation auto NF525 | Sprint 7 | 📅 Planifié |
| **Q4 2025** | Gateway PDP opérationnelle | Facture Vault → PDP | Sprint 8 | 📅 Planifié |
| **Q1 2026** | Audit Bureau Veritas | Label conformité | Sprint 9 | 📅 Planifié |
| **Q3 2026** | Pré-audit eIDAS | Process certifiable | Sprint 10+ | 📅 Planifié |
| **2027** | Certification ANSSI / eIDAS QSEAL | Reconnaissance officielle | Sprint 11+ | 📅 Planifié |
```

---

## 🎓 Conclusion

### Verdict Global

**Note** : **8.9/10** — Document stratégique ambitieux et bien structuré, avec des jalons réalistes.

### Points Remarquables

✅ **Vision stratégique complète**
- Roadmap claire sur 3 ans (2025-2027)
- Alignement sur 3 axes réglementaires (NF525, PDP, eIDAS)
- Jalons temporels réalistes

✅ **Cohérence réglementaire**
- Références précises et complètes
- Correspondance entre exigences et implémentation
- Processus d'audit structuré

✅ **Alignement technique**
- Technologies mentionnées sont implémentées ou planifiées
- Architecture cohérente avec le projet
- Standards reconnus (RFC, ISO, eIDAS)

### Améliorations Recommandées

⚠️ **Ajustements temporels**
- Aligner jalons avec sprints complétés (v1.3.0)
- Préciser dépendances entre jalons
- Ajuster Q1 2025 (déjà complété)

⚠️ **Enrichissements techniques**
- Détails processus d'audit interne
- Architecture PDP Gateway
- Intégration TSP (horodatage eIDAS)
- Progression eIDAS par niveau

### Prochaines Étapes

1. ✅ **Valider** les modifications suggérées avec l'équipe
2. 📝 **Mettre à jour** le document avec les ajustements temporels
3. 🔧 **Implémenter** les fonctionnalités manquantes (PDP Gateway, TSP, HSM)
4. 📚 **Documenter** les processus d'audit et de certification

---

**Document créé le** : Janvier 2025  
**Prochaine révision suggérée** : Après ajustement des jalons et validation de la roadmap

