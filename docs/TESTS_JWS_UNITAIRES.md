# 🧪 Tests Unitaires JWS — Module crypto

**Date** : Janvier 2025  
**Statut** : ✅ **15 tests créés et fonctionnels**

---

## 📊 Résumé des Tests

### Tests de Service (NewService)

| Test | Description | Statut |
|:-----|:------------|:-------|
| `TestNewService_Success` | Création service avec clés valides | ✅ |
| `TestNewService_InvalidPrivateKey` | Échec avec clé privée invalide | ✅ |
| `TestNewService_InvalidPublicKey` | Échec avec clé publique invalide | ✅ |
| `TestNewService_MissingFiles` | Échec avec fichiers manquants | ✅ |

### Tests de Signature (SignEvidence)

| Test | Description | Statut |
|:-----|:------------|:-------|
| `TestSignEvidence_Success` | Signature JWS réussie | ✅ |
| `TestSignEvidence_WithoutService` | Échec sans service initialisé | ✅ |

### Tests de Vérification (VerifyEvidence)

| Test | Description | Statut |
|:-----|:------------|:-------|
| `TestVerifyEvidence_Success` | Vérification JWS réussie | ✅ |
| `TestVerifyEvidence_InvalidJWS` | Échec avec JWS invalide | ✅ |
| `TestVerifyEvidence_TamperedJWS` | Échec avec JWS modifié | ✅ |
| `TestVerifyEvidence_WithoutService` | Échec sans service initialisé | ✅ |

### Tests JWKS

| Test | Description | Statut |
|:-----|:------------|:-------|
| `TestCurrentJWKS_Success` | Génération JWKS réussie | ✅ |
| `TestCurrentJWKS_WithoutService` | Échec sans service initialisé | ✅ |
| `TestJWKS_Structure` | Structure JWKS valide | ✅ |

### Tests Round-Trip

| Test | Description | Statut |
|:-----|:------------|:-------|
| `TestSignVerify_RoundTrip` | Cycle complet signer → vérifier | ✅ |
| `TestJWS_Format` | Format JWS valide (3 parties) | ✅ |

---

## 🚀 Exécution des Tests

### Tous les Tests JWS

```bash
go test ./tests/unit/... -run TestJWS -v
go test ./tests/unit/... -run TestNewService -v
go test ./tests/unit/... -run TestSignEvidence -v
go test ./tests/unit/... -run TestVerifyEvidence -v
go test ./tests/unit/... -run TestCurrentJWKS -v
go test ./tests/unit/... -run TestSignVerify -v
```

### Tests Spécifiques

```bash
# Test de signature
go test ./tests/unit/... -run TestSignEvidence_Success -v

# Test de vérification
go test ./tests/unit/... -run TestVerifyEvidence_Success -v

# Test round-trip
go test ./tests/unit/... -run TestSignVerify_RoundTrip -v
```

### Avec Coverage

```bash
go test -cover ./tests/unit/... -run TestJWS
```

---

## ✅ Validations Effectuées

### 1. Format JWS

- ✅ Structure : 3 parties séparées par des points (`header.payload.signature`)
- ✅ Encodage : Base64URL pour chaque partie
- ✅ Header : Contient `alg`, `kid`, `typ`

### 2. Signature

- ✅ Algorithme : RS256 (RSA-SHA256)
- ✅ Payload : `{document_id, sha256, timestamp, iat}`
- ✅ Kid : Inclus dans le header

### 3. Vérification

- ✅ Validation signature : Vérifie l'intégrité
- ✅ Extraction claims : DocumentID, SHA256, Timestamp
- ✅ Détection tampering : Rejette JWS modifiés

### 4. JWKS

- ✅ Structure : Format JSON standard
- ✅ Champs requis : `kty`, `kid`, `use`, `alg`, `n`, `e`
- ✅ Encodage : Base64URL pour `n` et `e`

### 5. Round-Trip

- ✅ Signer → Vérifier : Cycle complet fonctionnel
- ✅ Multiples documents : Test avec différents payloads
- ✅ Timestamps : Validation précision temporelle

---

## 🔒 Sécurité Testée

### Protection contre Tampering

- ✅ JWS modifié : Rejeté avec erreur
- ✅ Payload altéré : Signature invalide détectée
- ✅ Header modifié : Vérification échoue

### Gestion des Erreurs

- ✅ Clés invalides : Erreurs explicites
- ✅ Fichiers manquants : Gestion propre
- ✅ Service non initialisé : Messages d'erreur clairs

---

## 📝 Exemples de Tests

### Test de Signature

```go
service, _ := crypto.NewService(privateKeyPath, publicKeyPath, "test-kid")
jws, err := service.SignEvidence(docID, shaHex, timestamp)
assert.NoError(t, err)
assert.NotEmpty(t, jws)
```

### Test de Vérification

```go
evidence, err := service.VerifyEvidence(jws)
require.NoError(t, err)
assert.Equal(t, docID, evidence.DocumentID)
assert.Equal(t, shaHex, evidence.Sha256)
```

### Test Round-Trip

```go
// Signer
jws, _ := service.SignEvidence(docID, shaHex, timestamp)

// Vérifier
evidence, _ := service.VerifyEvidence(jws)
assert.Equal(t, docID, evidence.DocumentID)
```

---

## 🎯 Coverage Cible

- **NewService** : > 90%
- **SignEvidence** : > 85%
- **VerifyEvidence** : > 85%
- **CurrentJWKS** : > 80%

---

## 📚 Structure des Tests

### Setup/Teardown

- ✅ `setupTestKeys()` : Génère clés RSA temporaires
- ✅ `t.TempDir()` : Répertoire temporaire automatique
- ✅ Cleanup automatique : Suppression fichiers après tests

### Helpers

- ✅ `splitJWS()` : Divise JWS en 3 parties
- ✅ Validation Base64URL : Vérifie encodage correct

---

## ✅ Résultats

### Compilation

- ✅ `go test ./tests/unit/...` — **OK**
- ✅ Aucune erreur de linter
- ✅ Tous les tests passent

### Couverture

- ⏳ Coverage à mesurer avec `go test -cover`

---

**Document créé le** : Janvier 2025  
**Version** : 1.0  
**Statut** : ✅ **Tests complets et fonctionnels**

