# 🧩 Dorevia Vault — Certification Roadmap (NF525 → PDP → eIDAS)  
**Version : Novembre 2025 — Auteur : Doreviateam**

---

## 1️⃣ Introduction & Vision  
Objectif : positionner **Dorevia Vault** comme une **plateforme souveraine certifiable** pour la gestion documentaire et transactionnelle des PME françaises (ERP Odoo CE + POS).  
Alignement sur les 3 axes :  
- **NF525** : conformité logicielle POS (inaltérabilité / traçabilité / archivage)  
- **PDP Ready 2026** : intégrité documentaire et traçabilité pré-PPF  
- **eIDAS / ANSSI** : souveraineté & signature qualifiée (HSM / JWS / certificat)

---

## 2️⃣ Cadre réglementaire de référence  

| Domaine | Référence | Obligation clé |
|----------|------------|----------------|
| **Encaissement** | NF525 / BOI-TVA-DECLA-30-10-30 | Inaltérabilité, sécurisation, archivage |
| **Facturation électronique** | Ordonnance 2021-1190 / décret 2023-377 | Transmission via PDP, preuve d’origine |
| **Archivage électronique** | Code du Commerce L123-22 / eIDAS 910/2014 | Conservation probatoire 10 ans |
| **Protection des données** | RGPD 2016/679 | Sécurité, intégrité, consentement |

---

## 3️⃣ Architecture et conformité technique

| Couche | Fonction | Norme associée |
|--------|-----------|----------------|
| API Go (Fiber) | Ingestion, vérification | RFC 9110 (HTTP), RESTful |
| JWS Signer | Signature et scellement | RFC 7515, eIDAS |
| Ledger PostgreSQL | Traçabilité immuable | ISO 15489-1 (records mgmt) |
| Caddy Proxy HTTPS | Sécurité des échanges | TLS 1.3, HSTS |
| Prometheus / Audit log | Supervision et preuve d’intégrité | ISO 27001 (auditability) |

---

## 4️⃣ Plan de conformité NF525

| Pilier | Objectif | Implémentation Vault | Preuve technique |
|--------|-----------|----------------------|------------------|
| Inaltérabilité | Empêcher toute modification | SHA-256, hash chain | logs + hash JSON |
| Sécurisation | Signer chaque transaction | JWS signé localement | clé API + signature |
| Conservation | Garder les données 6 à 10 ans | Ledger PostgreSQL | backup / snapshot |
| Archivage | Clôture mensuelle scellée | Export signé (ZIP + manifest.json) | manifeste JWS |
| Traçabilité | Journal complet des actions | Audit log + RequestID | journaux Vault |

**Audit interne prévu :**
- Simulation d’audit Veritas : export 3 mois POS + facture → contrôle de cohérence SHA.  
- Documentation : `/docs/NF525_Attestation_Dorevia.md`

---

## 5️⃣ Plan PDP Ready 2026

| Étape | Description | Livrable |
|--------|-------------|-----------|
| PDP Gateway | API d’interconnexion Vault ↔ PDP (facture + hash) | `/api/v1/pdp/push` |
| Journalisation PPF | Ledger spécifique PDP | `ledger_pdp` |
| Mapping Factur-X → UBL 2.1 | Transformation native | XSLT + validation XSD |
| Horodatage eIDAS | Interfaçage TSP (trusted timestamp provider) | `/internal/tsp` |
| Preuve d’origine | Archivage JWS + certificat QSEAL | `/proofs/sha` |

---

## 6️⃣ Étape eIDAS / ANSSI (2027)

**Objectif :** devenir prestataire de service de confiance pour scellement documentaire (HSM).  

| Niveau | Action | Description |
|--------|---------|-------------|
| **Niveau 1** | HSM local (Vault certifié) | Intégration YubiHSM / SoftHSM |
| **Niveau 2** | Signature qualifiée eIDAS | Certificat QSEAL (Autorité française) |
| **Niveau 3** | Audit ANSSI / conformité RGS 2.0 | Processus ISO 27001 + HDS |

---

## 7️⃣ Processus d’audit et documentation

| Type | Contenu | Responsable | Fréquence |
|------|----------|--------------|------------|
| **Audit interne** | Revue des logs, ledger, hash, clés | Doreviateam | Mensuel |
| **Audit externe** | Audit technique NF525 | Bureau Veritas / LNE | Annuel |
| **Audit conformité** | RGPD + PDP | Consultant externe | Trimestriel |
| **Audit sécurité** | Vulnérabilités Go / Docker | Interne | Semestriel |

---

## 8️⃣ Livrables documentaires

| Document | Format | Dossier |
|-----------|---------|----------|
| `NF525_Attestation_Dorevia.md` | Markdown | `/docs/` |
| `PDP_Interface_Spec.md` | Markdown | `/docs/` |
| `eIDAS_Integration_Guide.md` | Markdown | `/docs/` |
| `Vault_Audit_Checklist.xlsx` | Tableur | `/audit/` |
| `Vault_Release_Signature.md` | Git signed tag | `/release/` |

---

## 9️⃣ Jalons temporels

| Période | Objectif clé | Résultat attendu |
|----------|---------------|------------------|
| **Q1 2025** | MVP certifiable NF525 | Prototype POS Vaulté |
| **Q2 2025** | Audit interne + documentation | Attestation auto NF525 |
| **Q3 2025** | Gateway PDP opérationnelle | Facture Vault → PDP |
| **Q1 2026** | Audit Bureau Veritas | Label conformité |
| **Q3 2026** | Pré-audit eIDAS | Process certifiable |
| **2027** | Certification ANSSI / eIDAS QSEAL | Reconnaissance officielle |

---