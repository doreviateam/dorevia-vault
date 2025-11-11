package crypto

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// KeyRotation gère la rotation des clés avec support multi-KID
type KeyRotation struct {
	currentKID      string
	previousKID     string
	nextRotationDate time.Time
	keys            map[string]*KeyPair
	keyManager      KeyManager
	log             zerolog.Logger
	mu              sync.RWMutex
	rotationPeriod  time.Duration // Durée avant rotation (ex: 90 jours)
}

// KeyPair représente une paire de clés avec métadonnées
type KeyPair struct {
	KID         string
	PrivateKey  *rsa.PrivateKey
	PublicKey   *rsa.PublicKey
	CreatedAt   time.Time
	ExpiresAt   time.Time
	IsActive    bool
}

// RotationConfig configuration pour KeyRotation
type RotationConfig struct {
	KeyManager     KeyManager
	CurrentKID     string
	RotationPeriod time.Duration // Durée avant rotation (défaut: 90 jours)
	Logger         zerolog.Logger
}

// NewKeyRotation crée un nouveau gestionnaire de rotation
func NewKeyRotation(cfg RotationConfig) (*KeyRotation, error) {
	if cfg.RotationPeriod == 0 {
		cfg.RotationPeriod = 90 * 24 * time.Hour // 90 jours par défaut
	}

	ctx := context.Background()

	// Charger la clé actuelle
	currentKey, err := loadKeyPair(ctx, cfg.KeyManager, cfg.CurrentKID)
	if err != nil {
		return nil, fmt.Errorf("failed to load current key: %w", err)
	}

	// Lister toutes les clés disponibles
	allKIDs, err := cfg.KeyManager.ListKIDs(ctx)
	if err != nil {
		cfg.Logger.Warn().Err(err).Msg("Failed to list KIDs, using current KID only")
		allKIDs = []string{cfg.CurrentKID}
	}

	// Charger toutes les clés
	keys := make(map[string]*KeyPair)
	keys[cfg.CurrentKID] = currentKey

	// Trouver la clé précédente (si existe)
	var previousKID string
	for _, kid := range allKIDs {
		if kid != cfg.CurrentKID {
			keyPair, err := loadKeyPair(ctx, cfg.KeyManager, kid)
			if err != nil {
				cfg.Logger.Warn().Err(err).Str("kid", kid).Msg("Failed to load key, skipping")
				continue
			}
			keys[kid] = keyPair
			// La clé précédente est celle qui n'est plus active mais pas expirée
			if !keyPair.IsActive && time.Now().Before(keyPair.ExpiresAt) {
				previousKID = kid
			}
		}
	}

	rotation := &KeyRotation{
		currentKID:       cfg.CurrentKID,
		previousKID:      previousKID,
		nextRotationDate: currentKey.ExpiresAt,
		keys:             keys,
		keyManager:       cfg.KeyManager,
		log:              cfg.Logger,
		rotationPeriod:   cfg.RotationPeriod,
	}

	cfg.Logger.Info().
		Str("current_kid", cfg.CurrentKID).
		Str("previous_kid", previousKID).
		Time("next_rotation", rotation.nextRotationDate).
		Msg("KeyRotation initialized")

	return rotation, nil
}

// loadKeyPair charge une paire de clés depuis le KeyManager
func loadKeyPair(ctx context.Context, keyManager KeyManager, kid string) (*KeyPair, error) {
	privateKey, err := keyManager.GetPrivateKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	publicKey, err := keyManager.GetPublicKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	// Pour simplifier, on considère que la clé est active si elle existe
	// et expire dans rotationPeriod jours
	createdAt := time.Now() // TODO: récupérer depuis métadonnées Vault si disponible
	expiresAt := createdAt.Add(90 * 24 * time.Hour) // 90 jours par défaut

	return &KeyPair{
		KID:        kid,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		CreatedAt:  createdAt,
		ExpiresAt:  expiresAt,
		IsActive:   true,
	}, nil
}

// GetCurrentKID retourne le KID actuel
func (kr *KeyRotation) GetCurrentKID() string {
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	return kr.currentKID
}

// GetPreviousKID retourne le KID précédent (si existe)
func (kr *KeyRotation) GetPreviousKID() string {
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	return kr.previousKID
}

// GetKeyPair récupère une paire de clés par KID
func (kr *KeyRotation) GetKeyPair(kid string) (*KeyPair, error) {
	kr.mu.RLock()
	defer kr.mu.RUnlock()

	keyPair, ok := kr.keys[kid]
	if !ok {
		// Essayer de charger depuis le KeyManager
		ctx := context.Background()
		keyPair, err := loadKeyPair(ctx, kr.keyManager, kid)
		if err != nil {
			return nil, fmt.Errorf("key not found: %s", kid)
		}
		// Mettre en cache
		kr.keys[kid] = keyPair
		return keyPair, nil
	}

	return keyPair, nil
}

// GetCurrentKeyPair récupère la paire de clés actuelle
func (kr *KeyRotation) GetCurrentKeyPair() (*KeyPair, error) {
	return kr.GetKeyPair(kr.GetCurrentKID())
}

// GetAllActiveKeys retourne toutes les clés actives (actuelle + précédente si valide)
func (kr *KeyRotation) GetAllActiveKeys() []*KeyPair {
	kr.mu.RLock()
	defer kr.mu.RUnlock()

	var activeKeys []*KeyPair
	now := time.Now()

	// Ajouter la clé actuelle
	if current, ok := kr.keys[kr.currentKID]; ok && current.IsActive && now.Before(current.ExpiresAt) {
		activeKeys = append(activeKeys, current)
	}

	// Ajouter la clé précédente si elle n'est pas expirée
	if kr.previousKID != "" {
		if previous, ok := kr.keys[kr.previousKID]; ok && now.Before(previous.ExpiresAt) {
			activeKeys = append(activeKeys, previous)
		}
	}

	return activeKeys
}

// ShouldRotate vérifie si une rotation est nécessaire
func (kr *KeyRotation) ShouldRotate() bool {
	kr.mu.RLock()
	defer kr.mu.RUnlock()

	// Rotation nécessaire si la date de rotation est passée
	return time.Now().After(kr.nextRotationDate)
}

// GetNextRotationDate retourne la date de la prochaine rotation
func (kr *KeyRotation) GetNextRotationDate() time.Time {
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	return kr.nextRotationDate
}

// Rotate effectue une rotation vers un nouveau KID
func (kr *KeyRotation) Rotate(newKID string) error {
	kr.mu.Lock()
	defer kr.mu.Unlock()

	ctx := context.Background()

	// Charger la nouvelle clé
	newKeyPair, err := loadKeyPair(ctx, kr.keyManager, newKID)
	if err != nil {
		return fmt.Errorf("failed to load new key: %w", err)
	}

	// Mettre à jour les KIDs
	kr.previousKID = kr.currentKID
	kr.currentKID = newKID

	// Désactiver l'ancienne clé
	if oldKey, ok := kr.keys[kr.previousKID]; ok {
		oldKey.IsActive = false
	}

	// Ajouter la nouvelle clé
	kr.keys[newKID] = newKeyPair
	kr.nextRotationDate = newKeyPair.ExpiresAt

	kr.log.Info().
		Str("previous_kid", kr.previousKID).
		Str("new_kid", newKID).
		Time("next_rotation", kr.nextRotationDate).
		Msg("Key rotation completed")

	return nil
}

// GetJWKS retourne le JWKS avec toutes les clés actives
func (kr *KeyRotation) GetJWKS() ([]byte, error) {
	activeKeys := kr.GetAllActiveKeys()

	if len(activeKeys) == 0 {
		return nil, fmt.Errorf("no active keys available")
	}

	// Construire le JWKS avec toutes les clés actives
	keys := make([]map[string]interface{}, 0, len(activeKeys))

	for _, keyPair := range activeKeys {
		jwk, err := keyPairToJWK(keyPair)
		if err != nil {
			kr.log.Warn().Err(err).Str("kid", keyPair.KID).Msg("Failed to convert key to JWK, skipping")
			continue
		}
		keys = append(keys, jwk)
	}

	jwks := map[string]interface{}{
		"keys": keys,
	}

	return json.MarshalIndent(jwks, "", "  ")
}

// keyPairToJWK convertit une KeyPair en JWK (JSON Web Key)
func keyPairToJWK(keyPair *KeyPair) (map[string]interface{}, error) {
	publicKey := keyPair.PublicKey

	// Encoder la clé publique en JWK
	nBytes := publicKey.N.Bytes()
	keySize := (publicKey.N.BitLen() + 7) / 8
	paddedN := make([]byte, keySize)
	copy(paddedN[keySize-len(nBytes):], nBytes)

	nBase64 := base64.RawURLEncoding.EncodeToString(paddedN)

	eBytes := make([]byte, 4)
	e := publicKey.E
	for i := 3; i >= 0; i-- {
		eBytes[i] = byte(e)
		e >>= 8
	}
	eBase64 := base64.RawURLEncoding.EncodeToString(eBytes)

	return map[string]interface{}{
		"kty": "RSA",
		"kid": keyPair.KID,
		"use": "sig",
		"alg": "RS256",
		"n":   nBase64,
		"e":   eBase64,
	}, nil
}

