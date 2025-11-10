# 🚀 Dorevia Vault

**Dorevia Vault** est un **proxy d'intégrité documentaire** open‑source, garantissant la traçabilité et la vérifiabilité des documents électroniques selon la **règle des 3 V** :

> **Validé** → Document validé dans Odoo  
> **Vaulté** → Archivé de manière sécurisée et immuable  
> **Vérifiable** → Preuve d'intégrité via JWS + Ledger

---

## ✨ Fonctionnalités clés

| Sprint | Intitulé | Livrables principaux |
|:--|:--|:--|
| **1** | MVP « Validé → Vaulté » | Ingestion Odoo, transaction atomique, idempotence par SHA256 |
| **2** | Documents « Vérifiables » | Scellement JWS RS256, Ledger hash‑chaîné, export JWKS |
| **3** | Expert Edition | Health checks, métriques Prometheus, vérification & réconciliation |
| **4** | Observabilité & Auditabilité continue | Logs signés JSONL, rapports d'audit JSON/CSV/PDF, CLI audit |

---

## 🔧 Endpoints essentiels

| Méthode | Route | Description |
|:--|:--|:--|
| `POST` | `/api/v1/invoices` | Ingestion d'une facture Odoo (JSON + base64) |
| `GET` | `/api/v1/ledger/verify/:id` | Vérifie l'intégrité d'un document |
| `GET` | `/metrics` | Expose les métriques Prometheus |
| `GET` | `/audit/export` | Exporte les journaux d'audit signés |
| `GET` | `/jwks.json` | Fournit la clé publique JWS (RS256) |

**Exemple :**  
```bash
curl -X POST https://vault.doreviateam.com/api/v1/invoices \
  -H "Content-Type: application/json" \
  -d '{
    "source": "sales",
    "model": "account.move",
    "odoo_id": 123,
    "state": "posted",
    "file": "base64_encoded_content",
    "filename": "invoice_001.pdf"
  }'
```

---

## 🧮 CLI Audit

L'outil `audit` permet de générer des **rapports d'audit consolidés** (mensuels, trimestriels ou personnalisés).

```bash
./bin/audit --period monthly --format pdf --sign --output report-2025-01.pdf
```

Résultat : un PDF 8 pages signé (JWS RS256) avec statistiques, QR code et hash SHA256.

---

## ⚙️ Stack technique

| Élément | Détail |
|:--|:--|
| **Langage** | Go 1.23 + |
| **Framework HTTP** | [Fiber v2.52.9](https://github.com/gofiber/fiber) |
| **Base de données** | PostgreSQL 14 + (pgxpool) |
| **Reverse proxy** | Caddy (HTTPS automatique) |
| **Logging** | Zerolog (JSON structuré) |
| **Métriques** | Prometheus + Alertmanager |
| **Signature** | JWS RS256 / RFC 7515 |
| **Licence** | MIT |

---

## 🛡️ Sécurité & conformité

- Clés RSA stockées avec permissions 600/644  
- Ledger hash‑chaîné immuable  
- Signature JWS RS256 avec export JWKS public  
- Logs JSONL signés et exportables  
- Mode dégradé optionnel (`JWS_REQUIRED=false`)  

---

## 🛣️ Roadmap synthétique

- [ ] Intégration HSM / Vault (HashiCorp Vault ou AWS KMS)  
- [ ] Rotation multi‑KID pour JWKS  
- [ ] Webhooks asynchrones (Redis Queue)  
- [ ] Validation Factur‑X (EN 16931)  
- [ ] Partitionnement Ledger > 100 k/an  

---

## 📚 Références

- [`CHANGELOG.md`](CHANGELOG.md) — historique complet des versions  
- [`RELEASE_NOTES_v1.2.0-rc1.md`](RELEASE_NOTES_v1.2.0-rc1.md) — notes de version détaillées  
- [`docs/audit_export_spec.md`](docs/audit_export_spec.md) — spécification export d'audit

---

## 👤 Contact & Contribution

**David Baron – Doreviateam**  
🌐 [https://doreviateam.com](https://doreviateam.com)  
📦 [GitHub : doreviateam/dorevia‑vault](https://github.com/doreviateam/dorevia-vault)  

Contributions bienvenues via issues & pull requests.  

> *Dorevia Vault — une brique souveraine, auditable et élégante pour la confiance numérique.*

