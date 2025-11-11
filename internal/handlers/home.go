package handlers

import (
	"github.com/gofiber/fiber/v2"
)

// Home retourne la page d'accueil HTML
func Home(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(`<!DOCTYPE html>
<html lang="fr">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width,initial-scale=1" />
  <title>Dorevia Vault — API d'intégrité souveraine</title>
  <style>
    :root { color-scheme: light dark; --bg:#fafafa; --fg:#1a1a1a; --accent:#007aff; --muted:#666; }
    @media (prefers-color-scheme: dark){ :root{ --bg:#0f1116; --fg:#eaeaea; --accent:#4ea1ff; --muted:#999; } }
    body{font-family:system-ui,-apple-system,"Segoe UI",Roboto,sans-serif;background:var(--bg);color:var(--fg);margin:0;height:100vh;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center}
    h1{font-size:2rem;margin:.1em 0}.muted{color:var(--muted)} a{color:var(--accent);text-decoration:none} a:hover{text-decoration:underline}
    ul{list-style:none;padding:0;margin:1em 0} li{margin:.3em 0} code{background:rgba(0,0,0,.06);padding:.2em .4em;border-radius:.3em}
    footer{margin-top:2rem;font-size:.85rem;color:var(--muted)}
    .section{margin-top:1.5rem;text-align:left;max-width:600px}
    .section-title{font-size:1.1rem;font-weight:600;margin-bottom:.5rem;color:var(--accent)}
  </style>
</head>
<body>
  <h1>🚀 Dorevia Vault API is running</h1>
  <div class="muted">Version <strong>v1.3.0</strong> — Sécurité & Interopérabilité</div>

  <div class="section">
    <div class="section-title">🔧 Routes de base</div>
    <ul>
      <li><a href="/health" target="_blank">🩺 /health</a> — état du service</li>
      <li><a href="/health/detailed" target="_blank">🔍 /health/detailed</a> — diagnostic complet</li>
      <li><a href="/version" target="_blank">📦 /version</a> — version déployée</li>
      <li><a href="/metrics" target="_blank">📊 /metrics</a> — métriques Prometheus</li>
    </ul>
  </div>

  <div class="section">
    <div class="section-title">🔐 Sécurité & Authentification</div>
    <ul>
      <li><a href="/jwks.json" target="_blank">🔑 /jwks.json</a> — clés publiques JWS</li>
    </ul>
  </div>

  <div class="section">
    <div class="section-title">📄 Documents & Ledger</div>
    <ul>
      <li><a href="/api/v1/invoices" target="_blank">📤 /api/v1/invoices</a> — endpoint d'ingestion (POST)</li>
      <li><a href="/api/v1/ledger/export" target="_blank">📋 /api/v1/ledger/export</a> — export ledger</li>
      <li><a href="/api/v1/ledger/verify/:document_id" target="_blank">✅ /api/v1/ledger/verify/:id</a> — vérification intégrité</li>
      <li><a href="/documents" target="_blank">📚 /documents</a> — liste documents</li>
      <li><a href="/download/:id" target="_blank">⬇️ /download/:id</a> — téléchargement</li>
    </ul>
  </div>

  <div class="section">
    <div class="section-title">📊 Audit & Conformité</div>
    <ul>
      <li><a href="/audit/export" target="_blank">📝 /audit/export</a> — export logs d'audit</li>
      <li><a href="/audit/dates" target="_blank">📅 /audit/dates</a> — dates disponibles</li>
    </ul>
  </div>

  <footer>
    © 2025 <a href="https://doreviateam.com" target="_blank" rel="noopener">Doreviateam</a> — Instance : <code>vault.doreviateam.com</code>
  </footer>
</body>
</html>`)
}

