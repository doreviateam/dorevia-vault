package crypto

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Evidence représente le payload JWS pour un document
type Evidence struct {
	DocumentID string    `json:"document_id"`
	Sha256     string    `json:"sha256"`
	Timestamp  time.Time `json:"timestamp"`
}

// Service gère les opérations JWS (signature et vérification)
type Service struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
	keyManager KeyManager // Optionnel : pour support Vault
}

// NewService crée un nouveau service JWS
// Charge les clés depuis les fichiers PEM ou variables d'environnement
// (Méthode legacy - utilise NewServiceWithKeyManager pour support Vault)
func NewService(privateKeyPath, publicKeyPath, kid string) (*Service, error) {
	// Essayer de charger depuis les fichiers
	privateKey, publicKey, err := loadKeysFromFiles(privateKeyPath, publicKeyPath)
	if err != nil {
		// Essayer depuis les variables d'environnement (base64)
		privateKey, publicKey, err = loadKeysFromEnv()
		if err != nil {
			return nil, fmt.Errorf("failed to load keys: %w", err)
		}
	}

	return &Service{
		privateKey: privateKey,
		publicKey: publicKey,
		kid:        kid,
	}, nil
}

// NewServiceWithKeyManager crée un nouveau service JWS avec KeyManager
// Permet d'utiliser Vault ou fichiers locaux de manière transparente
func NewServiceWithKeyManager(keyManager KeyManager, kid string) (*Service, error) {
	ctx := context.Background()

	// Charger les clés depuis le KeyManager
	privateKey, err := keyManager.GetPrivateKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key from key manager: %w", err)
	}

	publicKey, err := keyManager.GetPublicKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key from key manager: %w", err)
	}

	return &Service{
		privateKey: privateKey,
		publicKey:  publicKey,
		kid:        kid,
		keyManager: keyManager,
	}, nil
}

// loadKeysFromFiles charge les clés depuis les fichiers PEM
func loadKeysFromFiles(privateKeyPath, publicKeyPath string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// Charger clé privée
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key: %w", err)
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Essayer PKCS8
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		privateKey = key.(*rsa.PrivateKey)
	}

	// Charger clé publique
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read public key: %w", err)
	}

	block, _ = pem.Decode(publicKeyData)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to decode public key PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, nil, fmt.Errorf("public key is not RSA")
	}

	return privateKey, publicKey, nil
}

// loadKeysFromEnv charge les clés depuis les variables d'environnement (base64)
func loadKeysFromEnv() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	// TODO: Implémenter si nécessaire
	return nil, nil, fmt.Errorf("loading from env not implemented yet")
}

// SignEvidence signe un triplet (document_id, sha256, timestamp) et retourne un JWS
func (s *Service) SignEvidence(docID, shaHex string, t time.Time) (string, error) {
	if s.privateKey == nil {
		return "", fmt.Errorf("private key not loaded")
	}

	// Créer les claims
	claims := jwt.MapClaims{
		"document_id": docID,
		"sha256":      shaHex,
		"timestamp":   t.UTC().Format(time.RFC3339),
		"iat":         t.UTC().Unix(),
	}

	// Créer le token avec header personnalisé (kid)
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = s.kid

	// Signer
	jws, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return jws, nil
}

// VerifyEvidence vérifie un JWS et retourne l'Evidence
func (s *Service) VerifyEvidence(jws string) (*Evidence, error) {
	if s.publicKey == nil {
		return nil, fmt.Errorf("public key not loaded")
	}

	// Parser et vérifier le token
	token, err := jwt.Parse(jws, func(token *jwt.Token) (interface{}, error) {
		// Vérifier l'algorithme
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	// Extraire les claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	// Construire l'Evidence
	docID, ok := claims["document_id"].(string)
	if !ok {
		return nil, fmt.Errorf("document_id not found in claims")
	}

	sha256, ok := claims["sha256"].(string)
	if !ok {
		return nil, fmt.Errorf("sha256 not found in claims")
	}

	timestampStr, ok := claims["timestamp"].(string)
	if !ok {
		return nil, fmt.Errorf("timestamp not found in claims")
	}

	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format: %w", err)
	}

	return &Evidence{
		DocumentID: docID,
		Sha256:     sha256,
		Timestamp:  timestamp,
	}, nil
}

// CurrentJWKS retourne le JWKS (JSON Web Key Set) pour les clés publiques
func (s *Service) CurrentJWKS() ([]byte, error) {
	if s.publicKey == nil {
		return nil, fmt.Errorf("public key not loaded")
	}

	// Encoder la clé publique en JWK
	nBytes := s.publicKey.N.Bytes()
	// Padding pour avoir la bonne taille
	keySize := (s.publicKey.N.BitLen() + 7) / 8
	paddedN := make([]byte, keySize)
	copy(paddedN[keySize-len(nBytes):], nBytes)

	nBase64 := base64.RawURLEncoding.EncodeToString(paddedN)
	eBytes := make([]byte, 4)
	e := s.publicKey.E
	for i := 3; i >= 0; i-- {
		eBytes[i] = byte(e)
		e >>= 8
	}
	eBase64 := base64.RawURLEncoding.EncodeToString(eBytes)

	// Construire le JWKS
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"kid": s.kid,
				"use": "sig",
				"alg": "RS256",
				"n":   nBase64,
				"e":   eBase64,
			},
		},
	}

	return json.MarshalIndent(jwks, "", "  ")
}

// GetKID retourne le kid (Key ID) actuel
func (s *Service) GetKID() string {
	return s.kid
}
