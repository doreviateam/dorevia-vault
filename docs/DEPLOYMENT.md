# 🗂️ DOREVIA VAULT — DEPLOYMENT DOC (v0.0.1)

## 1. Objectif

Mise en service du microservice **Dorevia Vault**, développé en Go + Fiber, déployé sur le serveur `doreviateam.com`, accessible publiquement via **Caddy (Docker)** en HTTPS :  
➡️ `https://vault.doreviateam.com`

Ce service constitue la base du futur **coffre documentaire** (Factur-X, pièces jointes, etc.) interfacé avec Odoo CE 18 et OpenBee / PDP.

---

## 2. Infrastructure actuelle

| Élément | Description |
| :-- | :-- |
| **Serveur** | VPS Ubuntu 22.04 (user : `dorevia`) |
| **Stack existante** | `/opt/stacks/caddy` (reverse proxy Docker) |
| **Services internes** | `ai`, `caddy`, `uptime-kuma` |
| **Nouvel ajout** | `vault.doreviateam.com` (Go + Fiber) |
| **Port interne** | `8080` |
| **Réseau Docker** | bridge `172.20.0.0/16` |
| **Gateway Docker (vue de caddy)** | `172.20.0.1` |
| **HTTPS / Certificat** | Automatique via Caddy + Let’s Encrypt |
| **Firewall** | `ufw` actif avec ouverture 80/443 + 8080 interne |

---

## 3. Arborescence du service

```
/opt/dorevia-vault/
 ├── bin/vault                  # binaire compilé
 ├── cmd/vault/main.go          # code source principal
 ├── go.mod / go.sum            # dépendances
 ├── storage/                   # stockage local (à venir)
 └── deploy.sh                  # script de mise à jour
```

---

## 4. Service systemd

Fichier : `/etc/systemd/system/dorevia-vault.service`
```ini
[Unit]
Description=Dorevia Vault API
After=network.target

[Service]
User=dorevia
WorkingDirectory=/opt/dorevia-vault
ExecStart=/opt/dorevia-vault/bin/vault
Restart=always
Environment=PORT=8080
ExecStartPre=/bin/sleep 3

[Install]
WantedBy=multi-user.target
```

Commandes utiles :
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now dorevia-vault
sudo systemctl restart dorevia-vault
journalctl -u dorevia-vault -f
```

---

## 5. Configuration Caddy (reverse proxy HTTPS)

Fichier : `/opt/stacks/caddy/Caddyfile`
```caddy
vault.doreviateam.com {
    reverse_proxy 172.20.0.1:8080
}
```

Validation :
```bash
docker exec -it caddy curl -I http://172.20.0.1:8080/health
# → HTTP/1.1 200 OK
curl -I https://vault.doreviateam.com/
# → HTTP/2 200
```

---

## 6. Code Go minimal (v0.0.1)

```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Get("/version", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"version": "0.0.1"})
	})
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("🚀 Dorevia Vault API is running!")
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	log.Printf("Starting server on port %s", port)
	log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
}
```

Compilation :
```bash
cd /opt/dorevia-vault
go mod init github.com/doreviateam/dorevia-vault
go get github.com/gofiber/fiber/v2
go build -o bin/vault ./cmd/vault
sudo systemctl restart dorevia-vault
```

---

## 7. Tests de validation

```bash
curl -s https://vault.doreviateam.com/version
# → {"version":"0.0.1"}

curl -s https://vault.doreviateam.com/health
# → ok
```

---

## 8. Script de déploiement rapide

`/opt/dorevia-vault/deploy.sh`
```bash
#!/usr/bin/env bash
set -e
cd /opt/dorevia-vault
git pull
go build -o bin/vault ./cmd/vault
sudo systemctl restart dorevia-vault
echo "✅ Deploy OK"
```

---

## 9. Étapes suivantes (v0.1.x)

- 🔗 Ajouter **PostgreSQL** (`dorevia_vault` / user `vault`)
- 📂 Endpoint `/upload` pour stockage et indexation
- 🧾 Préparer liaison Odoo 18 CE (Factur-X)
- 🧠 Étudier intégration PDP (OpenBee) / e-Archivage (NF525)
- ☁️ Ajouter sauvegarde automatique vers S3/MinIO
