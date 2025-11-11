# 🔐 Spécification — Authentification & Autorisation

**Version** : v1.3.0  
**Date** : Janvier 2025  
**Sprint** : Sprint 5 Phase 5.2  
**Statut** : ✅ Implémenté

---

## 🎯 Vue d'Ensemble

Dorevia Vault implémente un système complet d'authentification (JWT/API Keys) et d'autorisation basée sur les rôles (RBAC) pour protéger les endpoints sensibles.

### Fonctionnalités

- ✅ **Authentification JWT** : Support RS256 avec clés publiques
- ✅ **Authentification API Keys** : Clés API avec expiration
- ✅ **RBAC** : 4 rôles avec permissions granulaires
- ✅ **Middleware Fiber** : Protection automatique des endpoints

---

## 🔑 Authentification

### Méthodes Supportées

#### 1. JWT (JSON Web Token)

**Format** : `Authorization: Bearer <token>`

**Algorithme** : RS256 (RSA avec SHA-256)

**Claims requis** :
```json
{
  "sub": "user-123",        // User ID (requis)
  "role": "operator",       // Rôle utilisateur (requis)
  "email": "user@example.com", // Optionnel
  "iat": 1234567890,       // Issued at
  "exp": 1234567890        // Expiration
}
```

**Configuration** :
```bash
AUTH_ENABLED=true
AUTH_JWT_ENABLED=true
AUTH_JWT_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/jwt-public.pem
```

#### 2. API Keys

**Format** : `Authorization: apikey <key>`

**Structure** :
```go
type APIKey struct {
    KeyID     string
    KeyHash   string  // SHA256 de la clé réelle
    UserID    string
    Role      string
    CreatedAt time.Time
    ExpiresAt *time.Time  // Optionnel
    IsActive  bool
}
```

**Configuration** :
```bash
AUTH_ENABLED=true
AUTH_APIKEY_ENABLED=true
```

**Note** : Les API Keys sont actuellement chargées depuis la configuration. Une intégration DB est prévue pour la gestion dynamique.

---

## 👥 Rôles & Permissions (RBAC)

### Rôles Disponibles

| Rôle | Description | Permissions |
|:-----|:-----------|:-----------|
| `admin` | Administrateur système | Toutes les permissions |
| `auditor` | Auditeur/conformité | Lecture documents, audit, ledger, vérification |
| `operator` | Opérateur métier | Lecture/écriture documents, audit |
| `viewer` | Consultation seule | Lecture documents uniquement |

### Permissions

| Permission | Description | Rôles |
|:-----------|:-----------|:------|
| `documents:read` | Lire les documents | admin, auditor, operator, viewer |
| `documents:write` | Créer/modifier documents | admin, operator |
| `audit:read` | Lire les logs d'audit | admin, auditor |
| `ledger:read` | Lire le ledger | admin |
| `documents:verify` | Vérifier l'intégrité | admin, auditor |
| `reconcile:execute` | Exécuter réconciliation | admin |
| `users:manage` | Gérer les utilisateurs | admin |

### Mapping Endpoints → Permissions

| Endpoint | Permission Requise | Rôles Autorisés |
|:---------|:-----------------|:----------------|
| `/api/v1/invoices` | `documents:write` | admin, operator |
| `/api/v1/ledger/export` | `ledger:read` | admin |
| `/audit/export` | `audit:read` | admin, auditor |
| `/api/v1/ledger/verify/:id` | `documents:verify` | admin, auditor |
| `/documents` | `documents:read` | admin, auditor, operator, viewer |
| `/download/:id` | `documents:read` | admin, auditor, operator, viewer |

---

## 🛡️ Middleware

### AuthMiddleware

Authentifie la requête et extrait les informations utilisateur.

```go
app.Use(auth.AuthMiddleware(authService, log))
```

**Comportement** :
- Vérifie le header `Authorization`
- Parse JWT ou API Key
- Stocke `UserInfo` dans `c.Locals("user")`
- Retourne 401 si authentification échoue

### RequirePermission

Vérifie qu'un utilisateur a une permission spécifique.

```go
app.Use(auth.RequirePermission(rbacService, auth.PermissionReadAudit, log))
```

**Comportement** :
- Récupère `UserInfo` depuis le contexte
- Vérifie la permission via RBAC
- Retourne 403 si permission refusée

### RequireRole

Vérifie qu'un utilisateur a un rôle spécifique.

```go
app.Use(auth.RequireRole(rbacService, auth.RoleAdmin, log))
```

### RequireEndpointPermission

Vérifie automatiquement la permission basée sur l'endpoint.

```go
app.Use(auth.RequireEndpointPermission(rbacService, log))
```

---

## 📋 Configuration

### Variables d'Environnement

```bash
# Activation authentification
AUTH_ENABLED=true

# JWT
AUTH_JWT_ENABLED=true
AUTH_JWT_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/jwt-public.pem

# API Keys
AUTH_APIKEY_ENABLED=true
```

### Exemple de Configuration Complète

```bash
# Authentification
AUTH_ENABLED=true
AUTH_JWT_ENABLED=true
AUTH_APIKEY_ENABLED=true
AUTH_JWT_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/jwt-public.pem

# JWS (peut être utilisé pour JWT aussi)
JWS_PUBLIC_KEY_PATH=/opt/dorevia-vault/keys/public.pem
```

---

## 🔧 Utilisation

### Protection d'un Endpoint

```go
// Route protégée avec permission
auditGroup := app.Group("/audit")
auditGroup.Use(auth.AuthMiddleware(authService, log))
auditGroup.Use(auth.RequirePermission(rbacService, auth.PermissionReadAudit, log))
auditGroup.Get("/export", handlers.AuditExportHandler(...))
```

### Récupérer l'Utilisateur

```go
func MyHandler(c *fiber.Ctx) error {
    userInfo, err := auth.GetUserInfo(c)
    if err != nil {
        return c.Status(401).JSON(fiber.Map{"error": "not authenticated"})
    }
    
    // Utiliser userInfo.UserID, userInfo.Role, etc.
    return c.JSON(fiber.Map{"user_id": userInfo.UserID})
}
```

---

## 🧪 Tests

### Tests Unitaires

- ✅ `TestNewAuthService` : Création service
- ✅ `TestAuthService_AuthenticateJWT` : Authentification JWT
- ✅ `TestAuthService_AuthenticateAPIKey` : Authentification API Key
- ✅ `TestRBACService_HasPermission` : Vérification permissions
- ✅ `TestRequirePermission` : Middleware permissions
- ✅ `TestRequireRole` : Middleware rôles

**Total** : 25 tests unitaires

---

## 🔐 Sécurité

### Bonnes Pratiques

1. **JWT** : Utiliser des clés RSA 2048+ bits
2. **API Keys** : Stocker uniquement le hash (SHA256)
3. **Expiration** : Configurer expiration pour API Keys
4. **HTTPS** : Toujours utiliser HTTPS en production
5. **Rotation** : Roter les clés JWT régulièrement

### Headers de Sécurité

Les middlewares existants ajoutent automatiquement :
- `Helmet` : Headers sécurité HTTP
- `CORS` : Configuration CORS
- `RateLimit` : Limitation de débit

---

## 📚 Références

- [RFC 7519 - JSON Web Token (JWT)](https://tools.ietf.org/html/rfc7519)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [RBAC Best Practices](https://www.okta.com/identity-101/role-based-access-control/)

