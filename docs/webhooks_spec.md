# Webhooks API - Dorevia Vault

## Vue d'ensemble

Dorevia Vault peut envoyer des webhooks asynchrones pour notifier les systèmes externes des événements importants. Les webhooks sont traités de manière asynchrone via une queue Redis, avec retry automatique et signature HMAC pour la sécurité.

## Configuration

### Variables d'environnement

```bash
# Activer les webhooks
WEBHOOKS_ENABLED=true

# URL Redis pour la queue
WEBHOOKS_REDIS_URL=redis://localhost:6379/0

# Clé secrète pour signature HMAC (optionnel mais recommandé)
WEBHOOKS_SECRET_KEY=your-secret-key-here

# Nombre de workers parallèles
WEBHOOKS_WORKERS=3

# URLs webhooks par événement (format: event1:url1,url2|event2:url3)
WEBHOOKS_URLS=document.vaulted:https://example.com/webhook/vaulted|document.verified:https://example.com/webhook/verified
```

### Format de configuration des URLs

```
event_type1:url1,url2|event_type2:url3
```

Exemple :
```
document.vaulted:https://api.example.com/webhooks/vaulted,https://backup.example.com/webhooks/vaulted|document.verified:https://api.example.com/webhooks/verified
```

## Types d'événements

### `document.vaulted`

Émis lorsqu'un document est stocké avec succès dans le vault.

**Payload :**
```json
{
  "document_id": "550e8400-e29b-41d4-a716-446655440000",
  "sha256_hex": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "filename": "invoice-2025-001.pdf",
  "size_bytes": 12345,
  "created_at": "2025-01-15T10:30:00Z",
  "evidence_jws": true,
  "ledger_hash": true,
  "odoo_id": 12345,
  "model": "account.move",
  "source": "sales"
}
```

### `document.verified`

Émis lorsqu'une vérification d'intégrité est effectuée sur un document.

**Payload :**
```json
{
  "document_id": "550e8400-e29b-41d4-a716-446655440000",
  "valid": true,
  "checks": [
    {
      "type": "file_exists",
      "status": "ok"
    },
    {
      "type": "sha256_match",
      "status": "ok"
    },
    {
      "type": "ledger_chain",
      "status": "ok"
    }
  ],
  "signed_proof": false,
  "duration_ms": 45
}
```

### `ledger.appended`

Émis lorsqu'une entrée est ajoutée au ledger.

**Payload :**
```json
{
  "document_id": "550e8400-e29b-41d4-a716-446655440000",
  "ledger_hash": "abc123...",
  "previous_hash": "def456...",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

### `error.critical`

Émis lors d'erreurs critiques du système.

**Payload :**
```json
{
  "error_type": "database_connection",
  "message": "Failed to connect to database",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

## Format des requêtes webhook

### Headers HTTP

Chaque requête webhook inclut les headers suivants :

- `Content-Type: application/json`
- `User-Agent: Dorevia-Vault/1.0`
- `X-Event-Type: <event_type>`
- `X-Timestamp: <RFC3339 timestamp>`
- `X-Signature: <HMAC-SHA256 signature>` (si `WEBHOOKS_SECRET_KEY` est configuré)

### Corps de la requête

Le corps de la requête est un JSON contenant le payload de l'événement.

### Signature HMAC

Si `WEBHOOKS_SECRET_KEY` est configuré, chaque webhook inclut un header `X-Signature` avec une signature HMAC-SHA256 du payload JSON.

**Exemple de vérification (Python) :**
```python
import hmac
import hashlib
import json

def verify_webhook_signature(payload, signature, secret_key):
    expected_signature = hmac.new(
        secret_key.encode(),
        json.dumps(payload, sort_keys=True).encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected_signature, signature)
```

## Retry et backoff exponentiel

Les webhooks qui échouent sont automatiquement réessayés avec un backoff exponentiel :

- Tentative 1 : 1 seconde
- Tentative 2 : 2 secondes
- Tentative 3 : 4 secondes
- Tentative 4 : 8 secondes
- Tentative 5 : 16 secondes
- Maximum : 300 secondes (5 minutes)

Par défaut, 5 tentatives sont effectuées. Un webhook est considéré comme échoué après le nombre maximum de tentatives.

## Réponse attendue

Le serveur webhook doit répondre avec un code HTTP 2xx (200-299) pour indiquer que le webhook a été traité avec succès. Tout autre code sera considéré comme un échec et déclenchera un retry.

## Exemple d'implémentation (Flask)

```python
from flask import Flask, request, jsonify
import hmac
import hashlib
import json

app = Flask(__name__)
WEBHOOK_SECRET = "your-secret-key-here"

def verify_signature(payload, signature):
    expected = hmac.new(
        WEBHOOK_SECRET.encode(),
        json.dumps(payload, sort_keys=True).encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)

@app.route('/webhook/vaulted', methods=['POST'])
def handle_vaulted():
    payload = request.json
    signature = request.headers.get('X-Signature')
    event_type = request.headers.get('X-Event-Type')
    
    # Vérifier la signature
    if not verify_signature(payload, signature):
        return jsonify({"error": "Invalid signature"}), 401
    
    # Traiter l'événement
    document_id = payload['document_id']
    print(f"Document {document_id} vaulted")
    
    # Retourner 200 OK
    return jsonify({"status": "ok"}), 200
```

## Monitoring

### Queue Redis

Vous pouvez surveiller la queue Redis pour voir le nombre de webhooks en attente :

```bash
redis-cli LLEN dorevia:webhooks
```

### Logs

Les logs de l'application incluent des informations sur les webhooks :

- Événements enqueued
- Succès d'envoi
- Échecs et retries
- Échecs définitifs après max retries

## Bonnes pratiques

1. **Toujours vérifier la signature HMAC** pour garantir l'authenticité des webhooks
2. **Répondre rapidement** (idéalement < 1 seconde) pour éviter les timeouts
3. **Implémenter l'idempotence** : un même événement peut être envoyé plusieurs fois en cas de retry
4. **Logger les webhooks reçus** pour audit et debugging
5. **Gérer les erreurs gracieusement** : retourner 200 OK même en cas d'erreur de traitement (et logger l'erreur)

## Désactivation

Pour désactiver les webhooks, définir :

```bash
WEBHOOKS_ENABLED=false
```

Les événements ne seront plus émis, mais l'application continuera de fonctionner normalement.

