# 🚀 Dorevia Vault

**Dorevia Vault** est un microservice écrit en **Go + Fiber**.  
Il constitue la brique “coffre documentaire” du projet **Doreviateam**,  
destiné à héberger, indexer et archiver de manière souveraine  
les documents électroniques (Factur-X, pièces jointes, rapports, etc.)

---

## 🌍 Environnement

| Élément | Détail |
| :-- | :-- |
| **Langage** | Go 1.23+ |
| **Framework HTTP** | [Fiber](https://github.com/gofiber/fiber) |
| **Reverse Proxy** | Caddy (HTTPS automatique via Let’s Encrypt) |
| **Base de données (à venir)** | PostgreSQL |
| **Domaine** | [https://vault.doreviateam.com](https://vault.doreviateam.com) |
| **Auteur / Mainteneur** | [David Baron – Doreviateam](https://doreviateam.com) |

---

## 🔧 Endpoints disponibles (v0.0.1)

| Méthode | Route | Description |
| :-- | :-- | :-- |
| `GET` | `/` | Page d’accueil |
| `GET` | `/health` | Vérifie l’état du service |
| `GET` | `/version` | Retourne la version déployée |

Exemple :
```bash
curl -s https://vault.doreviateam.com/version
# → {"version":"0.0.1"}
```

---

## 🧱 Structure

```
/opt/dorevia-vault/
 ├── bin/vault                  # binaire compilé
 ├── cmd/vault/main.go          # code source principal
 ├── go.mod / go.sum            # dépendances
 ├── storage/                   # stockage local (à venir)
 └── deploy.sh                  # script de déploiement
```

---

## 🚀 Déploiement

Voir la documentation complète :  
👉 [`docs/DEPLOYMENT_DOREVIA_VAULT_v0.0.1.md`](docs/DEPLOYMENT_DOREVIA_VAULT_v0.0.1.md)

Pour un déploiement rapide :
```bash
./deploy.sh
```

---

## 🛣️ Roadmap (v0.1.x à venir)

- [ ] Connexion PostgreSQL (`dorevia_vault`)
- [ ] Endpoint `/upload` pour stockage et indexation des fichiers
- [ ] Endpoint `/documents` pour recherche et consultation
- [ ] Liaison Odoo CE 18 / OpenBee PDP
- [ ] Archivage long terme (NF525 / MinIO / S3)

---

© 2025 Doreviateam – Projet sous licence MIT
# dorevia-vault
