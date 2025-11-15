# 🧩 Dorevia Vault — Convergence POS & PDP  
**Version : Novembre 2025 — Auteur : Doreviateam**

---

## 🎯 Objectif

Garantir la conformité **du cycle complet de transaction** :  
de la **vente POS (NF525)** à la **facturation électronique (PDP / PPF)**,  
grâce à une seule couche de confiance : **Dorevia Vault**.

---

## ⚙️ 1️⃣ Rappel réglementaire

| Domaine | Référence | Exigence |
|----------|------------|-----------|
| **POS (Point de Vente)** | NF525 / BOI-TVA-DECLA-30-10-30 | Inaltérabilité, sécurisation, conservation, archivage |
| **Facturation électronique** | Ordonnance 2021-1190 / PPF-PDP | Preuve d’origine, intégrité, traçabilité des échanges |
| **Archivage documentaire** | Code du commerce / eIDAS | Conservation probatoire 10 ans |

---

## 🧱 2️⃣ Architecture convergente

```
[Odoo POS / ERP]
   │  (Ticket fiscal / Facture)
   ▼
[Dorevia Vault 🔐]
   ├─ SHA-256 + JWS (scellement)
   ├─ Ledger PostgreSQL (traçabilité)
   ├─ API /verify/:sha (preuve publique)
   ▼
[PDP agréée]
   │
   ▼
[PPF / DGFIP]
```

> **Vault = Proxy d’intégrité universel**
>
> Il agit avant le PDP et en amont de la caisse, garantissant que **chaque document ou transaction** est :
> - signé, scellé et horodaté,  
> - conservé dans un registre auditable,  
> - vérifiable indépendamment du logiciel source.

---

## 🧩 3️⃣ Correspondance réglementaire

| Exigence | Domaine | Couverture Vault |
|-----------|----------|-----------------|
| Inaltérabilité | POS / NF525 | Hash SHA-256 |
| Sécurisation | POS / PDP | Signature JWS + HTTPS |
| Conservation | POS / PDP | Archivage PostgreSQL / Vault |
| Traçabilité | POS / PDP | Ledger immuable |
| Auditabilité | Tous | API de vérification /verify/:sha |
| Intégrité d’origine | PDP | Preuve d’émission scellée |

---

## 🚀 4️⃣ Enjeux stratégiques

- ✅ **Auto-certification NF525 possible** pour Odoo CE + Vault  
- ✅ **Préparation PDP Ready 2026**  
- ✅ **Interopérabilité open source (OCA)**  
- ✅ **Mutualisation documentaire et transactionnelle**  
- ✅ **Souveraineté numérique française**

---

## 🧭 5️⃣ Vision

> “Ce que Dorevia Vault apporte, c’est la continuité de confiance :
> de la caisse au fisc, du ticket à la facture, du local au cloud.”

---

## 📚 Annexe technique

- **Format POS exporté :** JSON ou XML  
- **Hashing :** SHA-256 (RFC 6234)  
- **Signature :** JWS (RFC 7515)  
- **Ledger :** PostgreSQL audit trail  
- **Transmission PDP :** REST / webhook sécurisé  
- **Archivage :** ≥ 10 ans (Vault ou PDP)