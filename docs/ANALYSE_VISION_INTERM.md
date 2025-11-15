# 🔍 Analyse Expert — Document `vision_interm.md`

**Date** : Janvier 2025  
**Analyseur** : Expert technique Dorevia Vault  
**Document analysé** : `docs/vision_interm.md`  
**Version projet** : v1.3.0 (Sprint 5 complété)

---

## 📋 Résumé Exécutif

Le document `vision_interm.md` présente le **positionnement stratégique** de Dorevia Vault dans la chaîne PDP/PPF pour la réforme française de facturation électronique 2026. Il s'agit d'un document de **vision intermédiaire** qui clarifie le rôle du Vault comme **proxy de confiance amont** entre Odoo et les PDP agréées.

**Verdict global** : ✅ **Document clair et bien structuré**, avec quelques points à préciser et enrichir.

---

## ✅ Points Forts

### 1. Vision Stratégique Claire

✅ **Positionnement précis**
- Le document définit clairement le rôle du Vault comme "proxy de confiance amont"
- La distinction entre les rôles (ERP, Vault, PDP, PPF) est bien expliquée
- Le cycle "Validé → Vaulté → Vérifiable" est cohérent avec la règle des 3V

✅ **Contexte réglementaire**
- Référence explicite à la réforme 2026
- Mention des PDP et PPF
- Conformité eIDAS mentionnée

### 2. Structure et Lisibilité

✅ **Organisation logique**
- 7 sections bien structurées
- Diagrammes ASCII clairs
- Tableaux comparatifs utiles

✅ **Communication efficace**
- Langage accessible
- Exemples concrets (Odoo ERP)
- Vision évolutive (v1.3 → v1.4 → v2.0)

### 3. Cohérence Technique

✅ **Alignement avec l'implémentation**
- SHA-256, JWS, Ledger PostgreSQL → ✅ Implémenté (Sprint 2)
- API `/verify/:sha` → ✅ Implémenté (Sprint 3)
- Webhooks → ✅ Implémenté (Sprint 5 Phase 5.3)
- Factur-X validation → ✅ Implémenté (Sprint 5 Phase 5.3)

---

## ⚠️ Points à Améliorer

### 1. Précisions Techniques

⚠️ **API `/verify/:sha`**
- Le document mentionne `/verify/:sha`, mais l'implémentation actuelle utilise `/api/v1/ledger/verify/:document_id` (UUID)
- **Recommandation** : Préciser que l'API accepte soit un UUID, soit un SHA-256 (à vérifier/implémenter)

⚠️ **Format Factur-X**
- Mention "Factur-X / UBL 2.1" → À préciser : Factur-X peut être UBL 2.1 ou UN/CEFACT CII
- **Recommandation** : Préciser les deux formats supportés

⚠️ **Horodatage eIDAS**
- Mention "Horodatage conforme eIDAS" → Actuellement, le Vault utilise des timestamps PostgreSQL
- **Recommandation** : Préciser si un service d'horodatage qualifié (TSA) est prévu pour v2.0

### 2. Évolution et Roadmap

⚠️ **Version v1.4 "PDP Gateway"**
- Mention "En développement" → À vérifier si c'est le Sprint 6 prévu
- **Recommandation** : Lier explicitement à la roadmap du projet (Sprint 6+)

⚠️ **Version v2.0 "Certification PDP / eIDAS"**
- Mention "Cible 2026" → À préciser les exigences de certification
- **Recommandation** : Détailler les prérequis de certification PDP

### 3. Conformité et Archivage

⚠️ **Conservation ≥ 10 ans**
- Mention "Conservation ≥ 10 ans (archivage légal)" → À vérifier si c'est configuré
- **Recommandation** : Préciser la stratégie d'archivage (rétention automatique, purge, etc.)

⚠️ **Points de conformité**
- Section 7 liste les conformités, mais manque de détails sur les tests/certifications
- **Recommandation** : Ajouter une section "Tests de conformité" ou "Certifications obtenues"

### 4. Intégration Technique

⚠️ **Webhooks PDP**
- Le document ne mentionne pas explicitement les webhooks pour intégration PDP
- **Recommandation** : Ajouter une section sur l'intégration webhook avec les PDP agréées

⚠️ **Mapping PDP**
- Mention "PDP Gateway (webhook + mapping)" → À détailler
- **Recommandation** : Expliquer le mapping (format de données, transformation, etc.)

---

## 🔧 Recommandations d'Amélioration

### Priorité Haute

1. **Préciser l'API de vérification**
   ```markdown
   ### API de Vérification
   - Endpoint actuel : `GET /api/v1/ledger/verify/:document_id` (UUID)
   - Extension prévue : `GET /api/v1/ledger/verify/sha/:sha256` (SHA-256)
   - Format réponse : Preuve JWS signée avec métadonnées complètes
   ```

2. **Détailler la roadmap v1.4**
   ```markdown
   ### v1.4 — PDP Gateway (Sprint 6 prévu)
   - Webhooks sortants vers PDP agréées
   - Mapping Factur-X → Format PDP
   - Gestion des erreurs de transmission
   - Retry automatique avec backoff exponentiel
   ```

3. **Clarifier l'horodatage eIDAS**
   ```markdown
   ### Horodatage eIDAS
   - Actuel : Timestamps PostgreSQL (UTC)
   - v2.0 prévu : Intégration TSA qualifié (eIDAS)
   - Preuve d'horodatage : Signature TSA dans JWS
   ```

### Priorité Moyenne

4. **Ajouter section "Intégration Webhooks PDP"**
   ```markdown
   ## 8️⃣ Intégration Webhooks PDP
   
   Le Vault émet des webhooks asynchrones vers les PDP agréées :
   - Événement `document.vaulted` : Document scellé et prêt pour transmission
   - Événement `document.verified` : Vérification d'intégrité effectuée
   - Signature HMAC-SHA256 pour authentification
   - Retry automatique en cas d'échec
   ```

5. **Préciser la stratégie d'archivage**
   ```markdown
   ## 9️⃣ Stratégie d'Archivage
   
   - Conservation minimale : 10 ans (archivage légal)
   - Rétention configurable par type de document
   - Purge automatique après expiration (optionnel)
   - Export avant purge pour archivage externe
   ```

6. **Ajouter section "Tests de Conformité"**
   ```markdown
   ## 🔟 Tests de Conformité
   
   - ✅ Factur-X : Validation EN 16931 (Sprint 5)
   - ✅ JWS : Conforme RFC 7515 (Sprint 2)
   - ✅ SHA-256 : Conforme RFC 6234 (Sprint 1)
   - ⏳ eIDAS : En attente certification TSA (v2.0)
   - ⏳ PDP : Tests d'intégration prévus (v1.4)
   ```

---

## 📊 Analyse de Cohérence avec le Projet

### Alignements Confirmés

✅ **Sprint 1-5** → Toutes les fonctionnalités mentionnées sont implémentées
- SHA-256, JWS, Ledger → ✅ Sprint 2
- API `/verify` → ✅ Sprint 3
- Webhooks → ✅ Sprint 5
- Factur-X → ✅ Sprint 5

### Écarts Identifiés

⚠️ **API `/verify/:sha`**
- Document mentionne `/verify/:sha` (SHA-256)
- Implémentation actuelle : `/api/v1/ledger/verify/:document_id` (UUID)
- **Action** : Vérifier si support SHA-256 existe, sinon l'ajouter

⚠️ **Horodatage eIDAS**
- Document mentionne "Horodatage conforme eIDAS"
- Implémentation actuelle : Timestamps PostgreSQL standard
- **Action** : Préciser que TSA qualifié est prévu pour v2.0

⚠️ **Conservation 10 ans**
- Document mentionne "Conservation ≥ 10 ans"
- Implémentation actuelle : Pas de stratégie de rétention automatique
- **Action** : Documenter la stratégie d'archivage ou l'implémenter

---

## 🎯 Recommandations Stratégiques

### Court Terme (v1.4 — Sprint 6)

1. **PDP Gateway**
   - Implémenter webhooks sortants vers PDP agréées
   - Mapping Factur-X → Format PDP
   - Gestion des erreurs et retry

2. **API `/verify/sha/:sha256`**
   - Ajouter endpoint pour vérification par SHA-256
   - Maintenir compatibilité avec UUID

3. **Documentation PDP**
   - Guide d'intégration pour PDP agréées
   - Exemples de webhooks
   - Format de données attendu

### Moyen Terme (v1.5-v1.6)

4. **Stratégie d'Archivage**
   - Configuration de rétention par type de document
   - Export automatique avant purge
   - Intégration avec systèmes d'archivage externes

5. **Tests de Conformité**
   - Tests automatisés Factur-X
   - Validation eIDAS (préparation)
   - Certification PDP (préparation)

### Long Terme (v2.0)

6. **Certification eIDAS**
   - Intégration TSA qualifié
   - Horodatage conforme eIDAS
   - Preuve d'horodatage dans JWS

7. **Certification PDP**
   - Tests d'intégration avec PDP agréées
   - Validation conformité PPF
   - Documentation de certification

---

## 📝 Modifications Suggérées au Document

### Section 3 — Rôles Techniques

**Ajouter** :
```markdown
| **Vault** | Scelle, journalise et certifie | Intégrité documentaire |
|           | Émet webhooks vers PDP         | Notification événements |
|           | API de vérification publique   | Preuve d'intégrité |
```

### Section 4 — Cycle "Validé → Vaulté → Vérifiable"

**Enrichir** :
```markdown
1. Facture validée dans Odoo
2. Vault reçoit, scelle (JWS) et enregistre dans le ledger
3. Vault émet webhook `document.vaulted` vers PDP
4. PDP transmet au PPF
5. En cas d'audit, le hash Vault + JWS prouve l'origine et l'intégrité
```

### Section 5 — Avantages Stratégiques

**Ajouter** :
```markdown
- 🔔 **Webhooks asynchrones** : Notification automatique PDP
- 🔄 **Retry automatique** : Résilience en cas d'échec transmission
- 📊 **Métriques** : Monitoring complet (Prometheus)
- 🔐 **Sécurité** : Authentification JWT/API Keys, RBAC
```

### Section 6 — Évolution

**Préciser** :
```markdown
| Version | Objectif | Statut | Sprint |
|---------|----------|--------|--------|
| v1.3 | Sécurité & Interopérabilité | ✅ Stable | Sprint 5 |
| v1.4 | PDP Gateway (webhook + mapping) | 🧩 Planifié | Sprint 6 |
| v2.0 | Certification PDP / eIDAS | 🚀 Cible 2026 | Sprint 7+ |
```

### Section 7 — Points de Conformité

**Enrichir** :
```markdown
- Format **Factur-X / UBL 2.1** validé ✅ (Sprint 5)
- Format **Factur-X / UN/CEFACT CII** → À valider (v1.4)
- Hash SHA-256 conforme RFC 6234 ✅ (Sprint 1)
- Signature JWS conforme RFC 7515 ✅ (Sprint 2)
- Horodatage conforme eIDAS → TSA qualifié prévu (v2.0)
- Conservation ≥ 10 ans (archivage légal) → Stratégie à documenter
- Webhooks HMAC-SHA256 ✅ (Sprint 5)
- Authentification JWT/API Keys ✅ (Sprint 5)
```

---

## 🎓 Conclusion

### Verdict Global

**Note** : **8.5/10** — Document clair et bien structuré, avec quelques précisions à apporter.

### Points Remarquables

✅ **Vision stratégique claire**
- Positionnement précis dans la chaîne PDP/PPF
- Rôles bien définis
- Évolution cohérente

✅ **Cohérence technique**
- Aligné avec l'implémentation actuelle
- Fonctionnalités mentionnées sont implémentées

### Améliorations Recommandées

⚠️ **Précisions techniques**
- API `/verify/:sha` → Vérifier/implémenter support SHA-256
- Horodatage eIDAS → Préciser TSA qualifié pour v2.0
- Conservation 10 ans → Documenter stratégie d'archivage

⚠️ **Enrichissements**
- Section webhooks PDP
- Section tests de conformité
- Section stratégie d'archivage

### Prochaines Étapes

1. ✅ **Valider** les modifications suggérées avec l'équipe
2. 📝 **Mettre à jour** le document avec les précisions
3. 🔧 **Implémenter** les fonctionnalités manquantes (API `/verify/sha/:sha256`, etc.)
4. 📚 **Documenter** la stratégie d'archivage et les tests de conformité

---

**Document créé le** : Janvier 2025  
**Prochaine révision suggérée** : Après validation des modifications

