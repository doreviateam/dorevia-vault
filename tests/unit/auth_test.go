package unit

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAuthService teste la création d'un AuthService
func TestNewAuthService(t *testing.T) {
	cfg := auth.AuthConfig{
		JWTEnabled:    true,
		APIKeyEnabled: true,
		Logger:        zerolog.Nop(),
	}

	service := auth.NewAuthService(cfg)
	assert.NotNil(t, service)
}

// TestAuthService_Authenticate_MissingHeader teste avec header manquant
func TestAuthService_Authenticate_MissingHeader(t *testing.T) {
	cfg := auth.AuthConfig{
		Logger: zerolog.Nop(),
	}

	service := auth.NewAuthService(cfg)

	ctx := context.Background()
	_, err := service.Authenticate(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authorization header missing")
}

// TestAuthService_Authenticate_InvalidFormat teste avec format invalide
func TestAuthService_Authenticate_InvalidFormat(t *testing.T) {
	cfg := auth.AuthConfig{
		Logger: zerolog.Nop(),
	}

	service := auth.NewAuthService(cfg)

	ctx := context.Background()
	_, err := service.Authenticate(ctx, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid authorization header format")
}

// TestAuthService_Authenticate_UnsupportedScheme teste avec schéma non supporté
func TestAuthService_Authenticate_UnsupportedScheme(t *testing.T) {
	cfg := auth.AuthConfig{
		Logger: zerolog.Nop(),
	}

	service := auth.NewAuthService(cfg)

	ctx := context.Background()
	_, err := service.Authenticate(ctx, "Basic token123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported authentication scheme")
}

// TestAuthService_AuthenticateAPIKey teste l'authentification avec API Key
func TestAuthService_AuthenticateAPIKey(t *testing.T) {
	apiKeys := make(map[string]*auth.APIKey)
	apiKey := "test-api-key-12345"
	keyHash := hashAPIKey(apiKey)

	apiKeys[keyHash] = &auth.APIKey{
		KeyID:     "key-1",
		KeyHash:   keyHash,
		UserID:    "user-123",
		Role:      "operator",
		CreatedAt: time.Now(),
		IsActive:  true,
	}

	cfg := auth.AuthConfig{
		APIKeys:       apiKeys,
		APIKeyEnabled: true,
		Logger:        zerolog.Nop(),
	}

	service := auth.NewAuthService(cfg)

	ctx := context.Background()
	userInfo, err := service.Authenticate(ctx, "apikey "+apiKey)
	require.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, "user-123", userInfo.UserID)
	assert.Equal(t, "operator", userInfo.Role)
}

// TestAuthService_AuthenticateAPIKey_Invalid teste avec clé API invalide
func TestAuthService_AuthenticateAPIKey_Invalid(t *testing.T) {
	cfg := auth.AuthConfig{
		APIKeys:       make(map[string]*auth.APIKey),
		APIKeyEnabled: true,
		Logger:        zerolog.Nop(),
	}

	service := auth.NewAuthService(cfg)

	ctx := context.Background()
	_, err := service.Authenticate(ctx, "apikey invalid-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid API key")
}

// TestAuthService_AuthenticateAPIKey_Expired teste avec clé API expirée
func TestAuthService_AuthenticateAPIKey_Expired(t *testing.T) {
	apiKeys := make(map[string]*auth.APIKey)
	apiKey := "test-api-key-12345"
	keyHash := hashAPIKey(apiKey)

	expiredTime := time.Now().Add(-1 * time.Hour)
	apiKeys[keyHash] = &auth.APIKey{
		KeyID:     "key-1",
		KeyHash:   keyHash,
		UserID:    "user-123",
		Role:      "operator",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		ExpiresAt: &expiredTime,
		IsActive:  true,
	}

	cfg := auth.AuthConfig{
		APIKeys:       apiKeys,
		APIKeyEnabled: true,
		Logger:        zerolog.Nop(),
	}

	service := auth.NewAuthService(cfg)

	ctx := context.Background()
	_, err := service.Authenticate(ctx, "apikey "+apiKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

// TestAuthService_AuthenticateJWT teste l'authentification avec JWT
func TestAuthService_AuthenticateJWT(t *testing.T) {
	// Créer une paire de clés RSA
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	cfg := auth.AuthConfig{
		JWTPublicKey: &privateKey.PublicKey,
		JWTEnabled:   true,
		Logger:       zerolog.Nop(),
	}

	service := auth.NewAuthService(cfg)

	// Créer un JWT valide
	claims := jwt.MapClaims{
		"sub":   "user-123",
		"role":  "operator",
		"email": "user@example.com",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	require.NoError(t, err)

	ctx := context.Background()
	userInfo, err := service.Authenticate(ctx, "Bearer "+tokenString)
	require.NoError(t, err)
	assert.NotNil(t, userInfo)
	assert.Equal(t, "user-123", userInfo.UserID)
	assert.Equal(t, "operator", userInfo.Role)
	assert.Equal(t, "user@example.com", userInfo.Email)
}

// TestAuthService_AuthenticateJWT_Invalid teste avec JWT invalide
func TestAuthService_AuthenticateJWT_Invalid(t *testing.T) {
	cfg := auth.AuthConfig{
		JWTEnabled: true,
		Logger:     zerolog.Nop(),
	}

	service := auth.NewAuthService(cfg)

	ctx := context.Background()
	_, err := service.Authenticate(ctx, "Bearer invalid.jwt.token")
	assert.Error(t, err)
}

// Helper function
func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return base64.StdEncoding.EncodeToString(hash[:])
}

