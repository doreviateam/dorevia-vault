package crypto

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog"
)

// KeyManager interface pour abstraction de la gestion des clés
type KeyManager interface {
	// GetPrivateKey récupère la clé privée RSA
	GetPrivateKey(ctx context.Context, kid string) (*rsa.PrivateKey, error)
	// GetPublicKey récupère la clé publique RSA
	GetPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error)
	// ListKIDs liste tous les KIDs disponibles
	ListKIDs(ctx context.Context) ([]string, error)
	// IsAvailable vérifie si le KeyManager est disponible
	IsAvailable(ctx context.Context) bool
}

// VaultKeyManager implémente KeyManager avec HashiCorp Vault
type VaultKeyManager struct {
	client     *vault.Client
	mountPath  string
	keyPath    string
	log        zerolog.Logger
	enabled    bool
}

// VaultConfig configuration pour VaultKeyManager
type VaultConfig struct {
	Enabled   bool
	Addr      string
	Token     string
	MountPath string
	KeyPath   string
	Logger    zerolog.Logger
}

// NewVaultKeyManager crée un nouveau VaultKeyManager
func NewVaultKeyManager(cfg VaultConfig) (*VaultKeyManager, error) {
	if !cfg.Enabled {
		return &VaultKeyManager{
			enabled: false,
			log:     cfg.Logger,
		}, nil
	}

	// Créer le client Vault
	config := vault.DefaultConfig()
	config.Address = cfg.Addr

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// Définir le token
	client.SetToken(cfg.Token)

	// Vérifier la connexion
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Sys().HealthWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to vault: %w", err)
	}

	if !health.Initialized {
		return nil, fmt.Errorf("vault is not initialized")
	}

	cfg.Logger.Info().
		Str("vault_addr", cfg.Addr).
		Str("mount_path", cfg.MountPath).
		Msg("VaultKeyManager initialized successfully")

	return &VaultKeyManager{
		client:    client,
		mountPath: cfg.MountPath,
		keyPath:   cfg.KeyPath,
		log:       cfg.Logger,
		enabled:   true,
	}, nil
}

// GetPrivateKey récupère la clé privée depuis Vault
func (v *VaultKeyManager) GetPrivateKey(ctx context.Context, kid string) (*rsa.PrivateKey, error) {
	if !v.enabled {
		return nil, fmt.Errorf("vault is not enabled")
	}

	// Construire le chemin complet
	secretPath := fmt.Sprintf("%s/%s/%s/private", v.mountPath, v.keyPath, kid)

	// Lire le secret depuis Vault
	secret, err := v.client.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret not found: %s", secretPath)
	}

	// Extraire la clé privée (peut être en base64 ou PEM)
	keyData, ok := secret.Data["key"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid key format in vault secret")
	}

	// Décoder base64 si nécessaire
	keyBytes := []byte(keyData)
	if secret.Data["encoding"] == "base64" {
		keyBytes, err = base64.StdEncoding.DecodeString(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 key: %w", err)
		}
	}

	// Parser la clé PEM
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Parser la clé privée
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Essayer PKCS8
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		privateKey = key.(*rsa.PrivateKey)
	}

	return privateKey, nil
}

// GetPublicKey récupère la clé publique depuis Vault
func (v *VaultKeyManager) GetPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if !v.enabled {
		return nil, fmt.Errorf("vault is not enabled")
	}

	// Construire le chemin complet
	secretPath := fmt.Sprintf("%s/%s/%s/public", v.mountPath, v.keyPath, kid)

	// Lire le secret depuis Vault
	secret, err := v.client.Logical().ReadWithContext(ctx, secretPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret from vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret not found: %s", secretPath)
	}

	// Extraire la clé publique
	keyData, ok := secret.Data["key"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid key format in vault secret")
	}

	// Décoder base64 si nécessaire
	keyBytes := []byte(keyData)
	if secret.Data["encoding"] == "base64" {
		keyBytes, err = base64.StdEncoding.DecodeString(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 key: %w", err)
		}
	}

	// Parser la clé PEM
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Parser la clé publique
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}

	return publicKey, nil
}

// ListKIDs liste tous les KIDs disponibles dans Vault
func (v *VaultKeyManager) ListKIDs(ctx context.Context) ([]string, error) {
	if !v.enabled {
		return nil, fmt.Errorf("vault is not enabled")
	}

	// Lister les secrets dans le répertoire des clés
	listPath := fmt.Sprintf("%s/%s", v.mountPath, v.keyPath)

	secret, err := v.client.Logical().ListWithContext(ctx, listPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list keys from vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	// Extraire la liste des clés
	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	kids := make([]string, 0, len(keys))
	for _, key := range keys {
		if kid, ok := key.(string); ok {
			kids = append(kids, kid)
		}
	}

	return kids, nil
}

// IsAvailable vérifie si Vault est disponible
func (v *VaultKeyManager) IsAvailable(ctx context.Context) bool {
	if !v.enabled {
		return false
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	health, err := v.client.Sys().HealthWithContext(ctx)
	if err != nil {
		v.log.Warn().Err(err).Msg("Vault health check failed")
		return false
	}

	return health.Initialized && !health.Sealed
}

// FileKeyManager implémente KeyManager avec fichiers locaux (fallback)
type FileKeyManager struct {
	privateKeyPath string
	publicKeyPath  string
	log            zerolog.Logger
}

// NewFileKeyManager crée un nouveau FileKeyManager
func NewFileKeyManager(privateKeyPath, publicKeyPath string, log zerolog.Logger) *FileKeyManager {
	return &FileKeyManager{
		privateKeyPath: privateKeyPath,
		publicKeyPath:  publicKeyPath,
		log:            log,
	}
}

// GetPrivateKey charge la clé privée depuis le fichier
func (f *FileKeyManager) GetPrivateKey(ctx context.Context, kid string) (*rsa.PrivateKey, error) {
	// Ignorer kid pour FileKeyManager (une seule clé)
	privateKey, _, err := loadKeysFromFiles(f.privateKeyPath, f.publicKeyPath)
	return privateKey, err
}

// GetPublicKey charge la clé publique depuis le fichier
func (f *FileKeyManager) GetPublicKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Ignorer kid pour FileKeyManager (une seule clé)
	_, publicKey, err := loadKeysFromFiles(f.privateKeyPath, f.publicKeyPath)
	return publicKey, err
}

// ListKIDs retourne une liste vide (une seule clé pour FileKeyManager)
func (f *FileKeyManager) ListKIDs(ctx context.Context) ([]string, error) {
	// Vérifier si les fichiers existent
	if _, err := os.Stat(f.privateKeyPath); err != nil {
		return []string{}, nil
	}
	// Retourner un KID par défaut
	return []string{"default"}, nil
}

// IsAvailable vérifie si les fichiers existent
func (f *FileKeyManager) IsAvailable(ctx context.Context) bool {
	_, err1 := os.Stat(f.privateKeyPath)
	_, err2 := os.Stat(f.publicKeyPath)
	return err1 == nil && err2 == nil
}

