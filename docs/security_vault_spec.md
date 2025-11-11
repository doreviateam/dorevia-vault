# 🔐 Spécification — Sécurité & Gestion des Clés

**Version** : v1.3.0  
**Date** : Janvier 2025  
**Sprint** : Sprint 5 Phase 5.1  
**Statut** : ✅ Implémenté

---

## 🎯 Vue d'Ensemble

Dorevia Vault intègre la gestion sécurisée des clés cryptographiques via **HashiCorp Vault** ou **AWS KMS**, avec support de la rotation multi-KID et du chiffrement au repos pour les logs d'audit.

### Fonctionnalités

- ✅ **Intégration HSM/Vault** : Stockage sécurisé des clés privées
- ✅ **Rotation multi-KID** : Support de plusieurs clés actives simultanément
- ✅ **Chiffrement au repos** : AES-256-GCM pour les logs d'audit
- ✅ **Fallback local** : Support fichiers locaux si Vault non disponible

---

## 🔑 Gestion des Clés (Key Management)

### Architecture

```
┌─────────────────┐
│  Dorevia Vault  │
└────────┬────────┘
         │
         ├──► HashiCorp Vault (production)
         │    └──► Clés privées RSA
         │
         ├──► AWS KMS (alternative)
         │    └──► Clés privées RSA
         │
         └──► Fichiers locaux (développement)
              └──► /opt/dorevia-vault/keys/
```

### Interface KeyManager

```go
type KeyManager interface {
    GetPrivateKey(ctx context.Context, kid string) (*rsa.PrivateKey, error)
    GetPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error)
    ListKIDs(ctx context.Context) ([]string, error)
    IsAvailable() bool
}
```

### Implémentations

#### 1. VaultKeyManager (HashiCorp Vault)

**Configuration :**
```bash
VAULT_ENABLED=true
VAULT_ADDR=https://vault.example.com:8200
VAULT_TOKEN=hvs.xxxxx
VAULT_KEY_PATH=secret/data/dorevia/keys
```

**Fonctionnalités :**
- Récupération des clés depuis Vault
- Support des secrets KV v2
- Gestion automatique des tokens
- Désactivation gracieuse si Vault indisponible

**Format des secrets dans Vault :**
```json
{
  "data": {
    "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...",
    "public_key": "-----BEGIN RSA PUBLIC KEY-----\n...",
    "kid": "key-2025-Q1"
  }
}
```

#### 2. FileKeyManager (Fichiers locaux)

**Configuration :**
```bash
JWS_PRIVATE_KEY_PATH=/opt/dorevia-vault/keys/private.pem
JWS_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/public.pem
```

**Fonctionnalités :**
- Lecture depuis fichiers PEM
- Support multi-KID (dossiers par KID)
- Fallback par défaut si Vault non configuré

**Structure des fichiers :**
```
/opt/dorevia-vault/keys/
├── key-2025-Q1/
│   ├── private.pem
│   └── public.pem
├── key-2025-Q2/
│   ├── private.pem
│   └── public.pem
└── default/
    ├── private.pem
    └── public.pem
```

---

## 🔄 Rotation Multi-KID

### Principe

La rotation multi-KID permet d'avoir **plusieurs clés actives simultanément** pour une transition en douceur lors des rotations.

### Structure KeyRotation

```go
type KeyRotation struct {
    CurrentKID      string
    PreviousKID     string
    NextKID         string
    RotationDate    time.Time
    PreviousRotationDate *time.Time
    NextRotationDate    *time.Time
}
```

### Cycle de Rotation

```
┌─────────────┐
│  key-2024-Q4│ (previous)
└─────────────┘
       │
       ▼
┌─────────────┐
│  key-2025-Q1│ (current) ← Active
└─────────────┘
       │
       ▼
┌─────────────┐
│  key-2025-Q2│ (next) ← Préparée
└─────────────┘
```

### Configuration

```bash
# Rotation automatique (optionnel)
KEY_ROTATION_ENABLED=true
KEY_ROTATION_INTERVAL=90d  # Rotation tous les 90 jours
```

### JWKS Dynamique

Le endpoint `/jwks.json` génère dynamiquement un JWKS incluant toutes les clés actives :

```json
{
  "keys": [
    {
      "kty": "RSA",
      "kid": "key-2025-Q1",
      "use": "sig",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    },
    {
      "kty": "RSA",
      "kid": "key-2024-Q4",
      "use": "sig",
      "alg": "RS256",
      "n": "...",
      "e": "AQAB"
    }
  ]
}
```

---

## 🔒 Chiffrement au Repos

### Principe

Les logs d'audit sensibles peuvent être chiffrés avec **AES-256-GCM** avant stockage sur disque.

### Configuration

```bash
AUDIT_ENCRYPTION_ENABLED=true
AUDIT_ENCRYPTION_KEY_ID=encryption-key-1
```

### Clé de Chiffrement

La clé de chiffrement est récupérée depuis le **KeyManager** (Vault ou fichiers) :

- **Vault** : `secret/data/dorevia/encryption-keys/encryption-key-1`
- **Fichiers** : `/opt/dorevia-vault/keys/encryption/encryption-key-1.key`

### Format Chiffré

```
[Nonce 12 bytes][Ciphertext][Tag 16 bytes]
```

**Total** : 12 + ciphertext_length + 16 bytes

### API

```go
// Chiffrer
encrypted, err := encryptionService.EncryptString("sensitive data")

// Déchiffrer
decrypted, err := encryptionService.DecryptString(encrypted)
```

---

## 🔧 Intégration JWS

### Service JWS avec KeyManager

```go
// Avec Vault
jwsService, err := crypto.NewServiceWithKeyManager(
    vaultKeyManager,
    "key-2025-Q1",
    log,
)

// Legacy (fichiers)
jwsService, err := crypto.NewService(
    "/path/to/private.pem",
    "/path/to/public.pem",
    "key-2025-Q1",
)
```

### Signature avec KID

Le JWS inclut automatiquement le KID dans le header :

```json
{
  "alg": "RS256",
  "kid": "key-2025-Q1",
  "typ": "JWT"
}
```

---

## 📋 Variables d'Environnement

### HashiCorp Vault

```bash
VAULT_ENABLED=true
VAULT_ADDR=https://vault.example.com:8200
VAULT_TOKEN=hvs.xxxxx
VAULT_KEY_PATH=secret/data/dorevia/keys
VAULT_NAMESPACE=  # Optionnel
```

### Rotation

```bash
KEY_ROTATION_ENABLED=false
KEY_ROTATION_INTERVAL=90d
CURRENT_KID=key-2025-Q1
PREVIOUS_KID=key-2024-Q4
NEXT_KID=key-2025-Q2
```

### Chiffrement Audit

```bash
AUDIT_ENCRYPTION_ENABLED=false
AUDIT_ENCRYPTION_KEY_ID=encryption-key-1
```

---

## 🧪 Tests

### Tests Unitaires

- ✅ `TestFileKeyManager` : Gestion fichiers locaux
- ✅ `TestVaultKeyManager` : Gestion Vault (mock)
- ✅ `TestKeyRotation` : Rotation multi-KID
- ✅ `TestEncryptionService` : Chiffrement/déchiffrement

**Total** : 24 tests unitaires

---

## 🔐 Bonnes Pratiques

1. **Production** : Toujours utiliser HashiCorp Vault ou AWS KMS
2. **Rotation** : Planifier rotation tous les 90 jours
3. **Backup** : Sauvegarder les clés publiques (pas les privées)
4. **Monitoring** : Surveiller l'état de Vault
5. **Fallback** : Tester le mode dégradé sans Vault

---

## 📚 Références

- [HashiCorp Vault API](https://developer.hashicorp.com/vault/api-docs)
- [RFC 7517 - JSON Web Key (JWK)](https://tools.ietf.org/html/rfc7517)
- [AES-GCM Specification](https://nvlpubs.nist.gov/nistpubs/Legacy/SP/nistspecialpublication800-38d.pdf)

