# 🧭 NOTE STRATÉGIQUE — ORIGINE & RAISON D'ÊTRE DE **DOREVIA VAULT**

## 📜 Contexte Réglementaire

La conformité fiscale et comptable en France impose des exigences strictes :

- **NF525** : Norme française pour systèmes de caisse certifiés
  - **Inaltérabilité** : Un ticket validé ne doit plus jamais être modifiable
  - **Sécurisation** : Chaque enregistrement doit être signé ou chaîné
  - **Conservation** : Les données doivent être gardées 6 ans, accessibles et horodatées
  - **Archivage** : Chaque clôture doit être scellée et exportable pour contrôle fiscal

- **PDP/PPF 2026** : Portail Public de Facturation / Portail Public de Facturation
  - Transmission électronique obligatoire des factures B2G (Business-to-Government)
  - Preuve d'intégrité requise avant transmission

- **EN 16931** : Standard européen pour factures électroniques (Factur-X)
  - Format XML structuré pour échanges B2B/B2G
  - Validation sémantique et syntaxique obligatoire

Ces exigences créent un besoin systémique de **preuve d'intégrité** indépendante des systèmes sources.

---

## 💡 Le point de départ

À l'origine, **Dorevia Vault** n'était pas un projet d'infrastructure,  
mais une **réponse humaine et pratique** à un problème bien réel :  
> comment prouver l'intégrité d'un ticket de caisse ou d'une facture  
> sans dépendre d'un éditeur fermé ou d'un tiers lointain ?

Tout a commencé avec le besoin d'un **POS certifié**, conforme aux règles NF525,  
mais **souverain** — c'est-à-dire installé, compris et maîtrisé localement.  

Les solutions existantes étaient toutes verrouillées, opaques, coûteuses.  
Elles ne laissaient aucune liberté d'évolution ni de contrôle technique.  
L'équipe a alors formulé ce qui allait devenir la question fondatrice du projet :

> "Comment rendre un système Odoo CE aussi fiable qu'un logiciel certifié,  
> sans renoncer à sa liberté ?"

---

## 🔐 La graine : le POS

Le problème du **POS** a tout déclenché.  
Il fallait garantir, comme le demande la loi française,  
que les données d'encaissement soient **inaltérables, sécurisées, conservées, et archivées**.

Les quatre piliers du NF525 étaient clairs :
1. **Inaltérabilité** : un ticket validé ne doit plus jamais être modifiable.  
2. **Sécurisation** : chaque enregistrement doit être signé ou chaîné.  
3. **Conservation** : les données doivent être gardées 6 ans, accessibles et horodatées.  
4. **Archivage** : chaque clôture doit être scellée et exportable pour contrôle fiscal.

Mais aucun module Odoo CE ne proposait cela.  
Et surtout, aucune solution ne garantissait la **preuve d'intégrité** indépendante de la base de données.  

L'équipe a donc posé l'idée simple, mais révolutionnaire :
> "On ne fait pas confiance à la base de données.  
> On fait confiance à la **preuve qu'elle produit**."

---

## ⚙️ Le basculement vers Dorevia Vault

De cette intuition est née l'architecture du **Vault** :  
un **coffre documentaire** qui scelle, signe et trace les documents générés par Odoo.

**Dorevia Vault** n'est pas un module de plus : c'est un **proxy d'intégrité**.

- Chaque facture ou ticket validé dans Odoo est envoyé au Vault.  
- Le Vault calcule son **hash (SHA-256)**, le signe via **JWS (RS256)**,  
  et stocke le tout dans un **ledger hash-chaîné**.  
- Il renvoie ensuite à Odoo un identifiant unique, la preuve signée,  
  et la possibilité de vérifier à tout moment que le document n'a pas été altéré.

Ce principe simple a donné naissance à la **règle des 3 V** :

> ✅ **Validé** → le document est approuvé dans Odoo  
> ✅ **Vaulté** → il est scellé et archivé dans le coffre  
> ✅ **Vérifiable** → sa preuve d'intégrité peut être contrôlée par quiconque, à tout moment  

---

## 🏗️ Architecture en Bref

**Dorevia Vault** est un microservice Go qui :

- Reçoit des documents depuis Odoo via API REST (`POST /api/v1/invoices`)
- Calcule un hash SHA-256 du document pour garantir l'intégrité
- Signe le hash via JWS (RS256) avec clé privée RSA
- Stocke la preuve dans un ledger PostgreSQL hash-chaîné (immutable)
- Retourne une preuve signée à Odoo (JWS token) pour stockage local

**Stack technique** : Go 1.23+, Fiber (framework HTTP), PostgreSQL, HashiCorp Vault (optionnel), Redis (webhooks)

**Connecteur Odoo** : Module OCA e-invoicing compatible, avec mécanisme de verrouillage après validation dans le Vault pour empêcher toute modification ultérieure.

---

## 🔄 Du POS à la conformité PDP/PPF

Au fur et à mesure que le Vault prenait forme,  
l'équipe a compris qu'il ne se limitait plus au POS.  

La même logique d'intégrité s'appliquait :
- aux **factures électroniques (Factur-X, EN 16931)**,  
- aux **flux PDP/PPF 2026**,  
- aux **rapports d'audit**,  
- voire demain aux **tickets POS certifiés**.

Autrement dit :
> le besoin de conformité n'est pas local, il est systémique.  
> Ce qu'on prouve pour un ticket, on peut le prouver pour toute donnée économique.

Ainsi, **Dorevia Vault** est devenu la **colonne vertébrale de la conformité Doreviateam** :
une plateforme souveraine qui garantit la **non-altération, la traçabilité et la vérifiabilité**  
de tous les flux sortants d'Odoo — factures, POS, rapports, inventaires, etc.

---

## 🧱 La philosophie Dorevia Vault

**Ne pas verrouiller, mais prouver.**

C'est la différence entre une caisse certifiée fermée et une solution ouverte, auditable, souveraine.  
Le Vault n'interdit pas de modifier : il **enregistre la preuve que cela a été fait**.  
Il ne bloque pas le système : il le **rend vérifiable et honnête**.

Ce n'est pas un logiciel fiscal.  
C'est une **infrastructure de confiance**.

> "Le but n'est pas de se protéger de l'utilisateur,  
> mais de protéger la preuve de son intégrité."

---

## 🔎 Les apports concrets aujourd'hui (v1.3 – Sprint 5)

### Pour l'entreprise

- ✅ **Conformité légale** : Preuve d'intégrité cryptographique pour contrôles fiscaux et audits
- ✅ **Réduction des risques** : Auditabilité complète des documents avec traçabilité immuable
- ✅ **Interopérabilité** : Support Factur-X (EN 16931) pour échanges B2B/B2G conformes
- ✅ **Archivage légal** : Conservation 6 ans avec preuve cryptographique vérifiable à tout moment
- ✅ **Souveraineté numérique** : Hébergement local ou mutualisé, sans dépendance cloud américaine

### Pour les équipes techniques

- ✅ **Traçabilité complète** : Hash SHA-256, JWS (RS256), ledger hash-chaîné, audit logs signés
- ✅ **Interopérabilité Odoo 18** : Connecteur stable avec mécanisme de verrouillage après validation dans le Vault
- ✅ **Preuves signées (JWS)** : Stockage dans Odoo pour vérification locale sans dépendance au Vault
- ✅ **Support Factur-X** : Validation EN 16931 intégrée avec extraction de métadonnées
- ✅ **Export d'audit** : Rapports mensuels et trimestriels signés (JSON, CSV, PDF) pour conformité
- ✅ **Architecture auditable** : Prête pour intégration PDP/PPF 2026 avec webhooks asynchrones
- ✅ **Sécurité renforcée** : Authentification JWT/API Keys, RBAC, intégration HashiCorp Vault

---

## 🆚 Différenciation

### Contrairement aux solutions propriétaires

- ✅ **Code source ouvert** : Auditable, vérifiable, modifiable selon les besoins
- ✅ **Pas de dépendance à un éditeur unique** : Évolutivité sans contraintes commerciales
- ✅ **Hébergement souverain** : Local ou mutualisé, contrôle total de l'infrastructure
- ✅ **Pas de verrouillage** : Migration possible, pas de dépendance technique exclusive
- ✅ **Coûts maîtrisés** : Pas de licences récurrentes, pas de coûts cachés

### Contrairement aux solutions cloud américaines

- ✅ **Conformité RGPD native** : Données hébergées en UE, pas de transfert hors UE
- ✅ **Souveraineté numérique** : Aucune dépendance à des services cloud américains
- ✅ **Contrôle total** : Infrastructure maîtrisée, pas de black box
- ✅ **Auditabilité** : Logs et métriques accessibles, pas de dépendance à des dashboards externes
- ✅ **Résilience** : Fonctionnement hors ligne possible, pas de dépendance à la connectivité cloud

### Contrairement aux solutions open-source non spécialisées

- ✅ **Spécialisé pour Odoo** : Connecteur natif, intégration optimisée
- ✅ **Conformité réglementaire** : NF525, PDP/PPF, EN 16931 intégrés dès la conception
- ✅ **Infrastructure de confiance** : JWS, ledger, audit logs signés par défaut
- ✅ **Documentation complète** : Spécifications techniques, guides de déploiement, cas d'usage

---

## 💼 Cas d'Usage Actuels

1. **Factures électroniques** : Validation Factur-X avant transmission B2B/B2G
2. **Audit interne** : Rapports mensuels/trimestriels signés pour conformité comptable
3. **Vérification d'intégrité** : Endpoint `/api/v1/ledger/verify/:id` pour contrôles fiscaux
4. **Archivage légal** : Conservation 6 ans avec preuve cryptographique vérifiable
5. **Traçabilité complète** : Ledger hash-chaîné pour toutes les opérations documentaires
6. **Webhooks asynchrones** : Notifications automatiques pour intégrations externes

---

## 🚀 Ce que le Vault prépare

Le Vault ouvre la voie à une **infrastructure souveraine complète** :  

1. **POS certifié-like**  
   - Chaînage des tickets, clôtures Z signées, export probant.  
   - Compatible avec les exigences NF525 sans dépendre d'un éditeur agréé.

2. **PDP/PPF 2026**  
   - Intégration naturelle avec les modules OCA e-invoicing.  
   - Preuve cryptographique avant transmission.  

3. **Auditabilité totale**  
   - Rapports automatiques mensuels et trimestriels signés.  
   - Ledger immuable pour toutes les opérations.  

4. **Souveraineté numérique**  
   - Hébergement local ou mutualisé.  
   - Aucune dépendance cloud américaine.  
   - Code open-source, vérifiable, auditable.

---

## ❤️ En conclusion

**Dorevia Vault** est né d'un besoin concret — certifier un POS —  
et s'est transformé en un outil de **confiance numérique et de conformité légale**.

C'est une réponse rationnelle et éthique à une question essentielle :  
> "Comment prouver qu'une donnée économique est vraie,  
> sans dépendre d'un tiers pour le dire à ma place ?"

Aujourd'hui, l'équipe ne construit pas seulement un logiciel.  
Elle pose la **pierre fondatrice d'une infrastructure de confiance**  
pour toutes les entreprises qui veulent être libres **et** conformes.

> **Dorevia Vault**, c'est la promesse qu'une donnée juste le restera.  

---

© 2025 Doreviateam – Document stratégique interne  

