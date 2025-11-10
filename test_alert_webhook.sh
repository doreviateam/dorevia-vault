#!/bin/bash
# Script de test manuel pour le webhook d'alertes
# Sprint 4 Phase 4.3

set -e

VAULT_URL="${VAULT_URL:-http://localhost:8080}"
WEBHOOK_URL="${VAULT_URL}/api/v1/alerts/webhook"

echo "🧪 Test du webhook d'alertes Dorevia Vault"
echo "=========================================="
echo ""
echo "URL: ${WEBHOOK_URL}"
echo ""

# Test 1 : Payload valide avec une alerte
echo "📋 Test 1 : Payload valide avec une alerte"
cat > /tmp/test_alert.json << 'EOF'
{
  "version": "4",
  "groupKey": "test-group",
  "status": "firing",
  "receiver": "default",
  "groupLabels": {
    "alertname": "HighDocumentErrorRate"
  },
  "commonLabels": {
    "alertname": "HighDocumentErrorRate",
    "severity": "warning",
    "component": "vault"
  },
  "commonAnnotations": {
    "summary": "Taux d'erreur élevé lors du stockage de documents",
    "description": "15% des documents échouent sur les 5 dernières minutes."
  },
  "externalURL": "http://localhost:9093",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "HighDocumentErrorRate",
        "severity": "warning",
        "component": "vault",
        "service": "dorevia-vault"
      },
      "annotations": {
        "summary": "Taux d'erreur élevé lors du stockage de documents",
        "description": "15% des documents échouent sur les 5 dernières minutes. Vérifier les logs et la santé du système."
      },
      "startsAt": "2025-01-20T10:00:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "http://localhost:9090/graph?g0.expr=..."
    }
  ]
}
EOF

response=$(curl -s -w "\n%{http_code}" -X POST "${WEBHOOK_URL}" \
  -H "Content-Type: application/json" \
  -d @/tmp/test_alert.json)

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

echo "Status HTTP: ${http_code}"
echo "Réponse: ${body}"
echo ""

if [ "${http_code}" = "200" ]; then
  echo "✅ Test 1 réussi"
else
  echo "❌ Test 1 échoué (code ${http_code})"
  exit 1
fi

# Test 2 : Payload avec plusieurs alertes
echo ""
echo "📋 Test 2 : Payload avec plusieurs alertes"
cat > /tmp/test_alerts_multi.json << 'EOF'
{
  "version": "4",
  "groupKey": "test-group-multi",
  "status": "firing",
  "receiver": "default",
  "alerts": [
    {
      "status": "firing",
      "labels": {
        "alertname": "HighDocumentErrorRate",
        "severity": "warning"
      },
      "annotations": {
        "summary": "Taux d'erreur élevé",
        "description": "15% d'erreurs"
      }
    },
    {
      "status": "firing",
      "labels": {
        "alertname": "SlowLedgerAppend",
        "severity": "warning"
      },
      "annotations": {
        "summary": "Ledger append lent",
        "description": "P95 > 2s"
      }
    }
  ]
}
EOF

response=$(curl -s -w "\n%{http_code}" -X POST "${WEBHOOK_URL}" \
  -H "Content-Type: application/json" \
  -d @/tmp/test_alerts_multi.json)

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

echo "Status HTTP: ${http_code}"
echo "Réponse: ${body}"
echo ""

if [ "${http_code}" = "200" ]; then
  echo "✅ Test 2 réussi"
else
  echo "❌ Test 2 échoué (code ${http_code})"
  exit 1
fi

# Test 3 : Payload invalide (JSON mal formé)
echo ""
echo "📋 Test 3 : Payload invalide (JSON mal formé)"
response=$(curl -s -w "\n%{http_code}" -X POST "${WEBHOOK_URL}" \
  -H "Content-Type: application/json" \
  -d '{"invalid": json}')

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

echo "Status HTTP: ${http_code}"
echo "Réponse: ${body}"
echo ""

if [ "${http_code}" = "400" ]; then
  echo "✅ Test 3 réussi (erreur attendue pour JSON invalide)"
else
  echo "❌ Test 3 échoué (devrait retourner 400)"
  exit 1
fi

# Test 4 : Alerte résolue (ne doit pas être exportée vers Odoo)
echo ""
echo "📋 Test 4 : Alerte résolue"
cat > /tmp/test_alert_resolved.json << 'EOF'
{
  "version": "4",
  "status": "resolved",
  "receiver": "default",
  "alerts": [
    {
      "status": "resolved",
      "labels": {
        "alertname": "HighDocumentErrorRate",
        "severity": "warning"
      },
      "annotations": {
        "summary": "Taux d'erreur élevé (résolu)",
        "description": "Le problème est résolu"
      }
    }
  ]
}
EOF

response=$(curl -s -w "\n%{http_code}" -X POST "${WEBHOOK_URL}" \
  -H "Content-Type: application/json" \
  -d @/tmp/test_alert_resolved.json)

http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | head -n-1)

echo "Status HTTP: ${http_code}"
echo "Réponse: ${body}"
echo ""

if [ "${http_code}" = "200" ]; then
  echo "✅ Test 4 réussi"
else
  echo "❌ Test 4 échoué (code ${http_code})"
  exit 1
fi

# Nettoyage
rm -f /tmp/test_alert.json /tmp/test_alerts_multi.json /tmp/test_alert_resolved.json

echo ""
echo "🎉 Tous les tests sont passés !"
echo ""
echo "💡 Pour tester avec Odoo, configurez les variables d'environnement :"
echo "   export ODOO_URL='https://odoo.example.com'"
echo "   export ODOO_DATABASE='dorevia'"
echo "   export ODOO_USER='vault_user'"
echo "   export ODOO_PASSWORD='...'"
echo "   sudo systemctl restart dorevia-vault"

