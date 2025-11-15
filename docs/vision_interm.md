# 🧩 Dorevia Vault — Positionnement dans la chaîne PDP / PPF  
**Version : Novembre 2025 — Auteur : Doreviateam**

---

## 1️⃣ Contexte
La réforme française de la facturation électronique (entrée en vigueur 2026) impose le passage par une **Plateforme de Dématérialisation Partenaire (PDP)** avant transmission au **Portail Public de Facturation (PPF)**.  
Les ERP (ex. Odoo) devront s’y connecter pour chaque facture émise.

**Dorevia Vault** agit comme **proxy de confiance amont**, garantissant l’intégrité cryptographique et la preuve d’origine des documents.

---

## 2️⃣ Chaîne documentaire simplifiée

### Sans Vault :
```
[Odoo ERP]
   │
   ▼
[PDP agréée]
   │
   ▼
[PPF / DGFIP]
```

### Avec Dorevia Vault :
```
[Odoo ERP 18 CE]
   │
   ▼
[Dorevia Vault 🔐]
   ├─ SHA-256 + JWS
   ├─ Ledger PostgreSQL
   ├─ API /verify/:sha
   ▼
[PDP agréée]
   │
   ▼
[PPF / DGFIP]
```

---

## 3️⃣ Rôles techniques comparés

| Acteur | Rôle principal | Responsabilité |
|--------|----------------|----------------|
| **ERP (Odoo)** | Crée et structure les factures | Métier et gestion |
| **Vault** | Scelle, journalise et certifie | Intégrité documentaire |
| **PDP** | Valide et transmet au PPF | Conformité fiscale |
| **PPF** | Centralise et trace | Collecte légale |

---

## 4️⃣ Cycle “Validé → Vaulté → Vérifiable”

1. Facture validée dans Odoo  
2. Vault reçoit, scelle et enregistre la preuve  
3. PDP transmet au PPF  
4. En cas d’audit, le hash Vault prouve l’origine

---

## 5️⃣ Avantages stratégiques

- 🧩 **Interopérabilité totale** : Odoo ↔ Vault ↔ PDP  
- 🇫🇷 **Souveraineté** : stockage auditable et local  
- 🔐 **Conformité PDP Ready 2026**  
- 📜 **Auditabilité** : API de vérification publique  
- 💼 **Intégration naturelle OCA / Odoo CE**

---

## 6️⃣ Évolution

| Version | Objectif | Statut |
|----------|-----------|---------|
| v1.3 | Sécurité & Interopérabilité | ✅ Stable |
| v1.4 | PDP Gateway (webhook + mapping) | 🧩 En développement |
| v2.0 | Certification PDP / eIDAS | 🚀 Cible 2026 |

---

## 7️⃣ Annexe — Points de conformité

- Format **Factur-X / UBL 2.1** validé  
- Hash SHA-256 conforme RFC 6234  
- Signature JWS conforme RFC 7515  
- Horodatage conforme eIDAS  
- Conservation ≥ 10 ans (archivage légal)
