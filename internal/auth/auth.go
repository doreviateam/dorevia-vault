package auth

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

// AuthService gère l'authentification (JWT et API Keys)
type AuthService struct {
	jwtPublicKey *rsa.PublicKey // Pour vérification JWT
	apiKeys      map[string]*APIKey
	log          zerolog.Logger
	jwtEnabled   bool
	apiKeyEnabled bool
}

// APIKey représente une clé API
type APIKey struct {
	KeyID     string
	KeyHash   string // SHA256 de la clé réelle
	UserID    string
	Role      string
	CreatedAt time.Time
	ExpiresAt *time.Time // Optionnel
	IsActive  bool
}

// AuthConfig configuration pour AuthService
type AuthConfig struct {
	JWTPublicKey  *rsa.PublicKey // Clé publique pour vérification JWT
	APIKeys       map[string]*APIKey
	JWTEnabled    bool
	APIKeyEnabled bool
	Logger        zerolog.Logger
}

// NewAuthService crée un nouveau service d'authentification
func NewAuthService(cfg AuthConfig) *AuthService {
	return &AuthService{
		jwtPublicKey:  cfg.JWTPublicKey,
		apiKeys:       cfg.APIKeys,
		log:           cfg.Logger,
		jwtEnabled:    cfg.JWTEnabled,
		apiKeyEnabled: cfg.APIKeyEnabled,
	}
}

// Authenticate authentifie une requête et retourne les informations utilisateur
func (a *AuthService) Authenticate(ctx context.Context, authHeader string) (*UserInfo, error) {
	if authHeader == "" {
		return nil, fmt.Errorf("authorization header missing")
	}

	// Détecter le type d'authentification
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	scheme := strings.ToLower(parts[0])
	token := parts[1]

	switch scheme {
	case "bearer":
		if !a.jwtEnabled {
			return nil, fmt.Errorf("JWT authentication is not enabled")
		}
		return a.authenticateJWT(ctx, token)
	case "apikey":
		if !a.apiKeyEnabled {
			return nil, fmt.Errorf("API key authentication is not enabled")
		}
		return a.authenticateAPIKey(ctx, token)
	default:
		return nil, fmt.Errorf("unsupported authentication scheme: %s", scheme)
	}
}

// authenticateJWT authentifie avec un JWT
func (a *AuthService) authenticateJWT(ctx context.Context, tokenString string) (*UserInfo, error) {
	if a.jwtPublicKey == nil {
		return nil, fmt.Errorf("JWT public key not configured")
	}

	// Parser et vérifier le token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Vérifier l'algorithme
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtPublicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to verify JWT: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid JWT token")
	}

	// Extraire les claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid JWT claims format")
	}

	// Construire UserInfo depuis les claims
	userInfo := &UserInfo{
		UserID: getStringClaim(claims, "sub", "user_id"),
		Role:   getStringClaim(claims, "role"),
		Email:  getStringClaim(claims, "email"),
	}

	if userInfo.UserID == "" {
		return nil, fmt.Errorf("user ID not found in JWT claims")
	}

	return userInfo, nil
}

// authenticateAPIKey authentifie avec une clé API
func (a *AuthService) authenticateAPIKey(ctx context.Context, apiKey string) (*UserInfo, error) {
	// Hasher la clé API
	keyHash := hashAPIKey(apiKey)

	// Chercher la clé dans la map
	key, ok := a.apiKeys[keyHash]
	if !ok {
		return nil, fmt.Errorf("invalid API key")
	}

	// Vérifier si la clé est active
	if !key.IsActive {
		return nil, fmt.Errorf("API key is not active")
	}

	// Vérifier l'expiration
	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return nil, fmt.Errorf("API key has expired")
	}

	return &UserInfo{
		UserID: key.UserID,
		Role:   key.Role,
		KeyID:  key.KeyID,
	}, nil
}

// UserInfo contient les informations de l'utilisateur authentifié
type UserInfo struct {
	UserID string
	Role   string
	Email  string
	KeyID  string // Pour API keys
}

// getStringClaim extrait une claim string avec fallback
func getStringClaim(claims jwt.MapClaims, keys ...string) string {
	for _, key := range keys {
		if val, ok := claims[key].(string); ok && val != "" {
			return val
		}
	}
	return ""
}

// hashAPIKey hash une clé API avec SHA256
func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return base64.StdEncoding.EncodeToString(hash[:])
}

