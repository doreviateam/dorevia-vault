package audit

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/rs/zerolog"
)

// EncryptionService gère le chiffrement/déchiffrement des logs d'audit
type EncryptionService struct {
	keyManager crypto.KeyManager
	keyID      string // ID de la clé de chiffrement dans Vault
	log        zerolog.Logger
	enabled    bool
}

// EncryptionConfig configuration pour EncryptionService
type EncryptionConfig struct {
	Enabled    bool
	KeyManager crypto.KeyManager
	KeyID      string // ID de la clé de chiffrement (ex: "audit-encryption-key")
	Logger     zerolog.Logger
}

// NewEncryptionService crée un nouveau service de chiffrement
func NewEncryptionService(cfg EncryptionConfig) (*EncryptionService, error) {
	if !cfg.Enabled {
		return &EncryptionService{
			enabled: false,
			log:     cfg.Logger,
		}, nil
	}

	if cfg.KeyManager == nil {
		return nil, fmt.Errorf("key manager is required when encryption is enabled")
	}

	if cfg.KeyID == "" {
		cfg.KeyID = "audit-encryption-key"
	}

	cfg.Logger.Info().
		Str("key_id", cfg.KeyID).
		Msg("EncryptionService initialized")

	return &EncryptionService{
		keyManager: cfg.KeyManager,
		keyID:      cfg.KeyID,
		log:        cfg.Logger,
		enabled:    true,
	}, nil
}

// getEncryptionKey récupère la clé de chiffrement depuis le KeyManager
// La clé est dérivée de la clé privée RSA (SHA256 de la clé privée)
func (e *EncryptionService) getEncryptionKey() ([]byte, error) {
	if !e.enabled {
		return nil, fmt.Errorf("encryption is not enabled")
	}

	ctx := context.Background()

	// Récupérer la clé privée depuis le KeyManager
	privateKey, err := e.keyManager.GetPrivateKey(ctx, e.keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key from key manager: %w", err)
	}

	// Dériver une clé AES-256 (32 bytes) depuis la clé privée RSA
	// On utilise SHA256 de la clé privée encodée
	privateKeyBytes := privateKey.N.Bytes()
	hash := sha256.Sum256(privateKeyBytes)

	// Retourner les 32 premiers bytes (AES-256)
	return hash[:], nil
}

// Encrypt chiffre des données avec AES-256-GCM
func (e *EncryptionService) Encrypt(plaintext []byte) ([]byte, error) {
	if !e.enabled {
		// Si le chiffrement est désactivé, retourner le texte en clair
		return plaintext, nil
	}

	// Récupérer la clé de chiffrement
	key, err := e.getEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Créer le cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Créer le GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Générer un nonce aléatoire (12 bytes pour GCM)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Chiffrer
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// Decrypt déchiffre des données avec AES-256-GCM
func (e *EncryptionService) Decrypt(ciphertext []byte) ([]byte, error) {
	if !e.enabled {
		// Si le chiffrement est désactivé, retourner tel quel
		return ciphertext, nil
	}

	// Récupérer la clé de chiffrement
	key, err := e.getEncryptionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Créer le cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Créer le GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extraire le nonce (12 premiers bytes)
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Déchiffrer
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString chiffre une chaîne et retourne base64
func (e *EncryptionService) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString déchiffre une chaîne base64
func (e *EncryptionService) DecryptString(ciphertextBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// IsEnabled retourne si le chiffrement est activé
func (e *EncryptionService) IsEnabled() bool {
	return e.enabled
}

