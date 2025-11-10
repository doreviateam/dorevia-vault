package config

import (
	"os"

	"github.com/caarlos0/env/v11"
)

// Config contient toute la configuration de l'application
type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	DatabaseURL string `env:"DATABASE_URL" envDefault:""`
	StorageDir  string `env:"STORAGE_DIR" envDefault:"/opt/dorevia-vault/storage"`
	
	// Audit Configuration (Sprint 4 Phase 4.2)
	AuditDir string `env:"AUDIT_DIR" envDefault:"/opt/dorevia-vault/audit"`
	
	// Odoo Export Configuration (Sprint 4 Phase 4.3)
	OdooURL      string `env:"ODOO_URL" envDefault:""`
	OdooDatabase string `env:"ODOO_DATABASE" envDefault:""`
	OdooUser     string `env:"ODOO_USER" envDefault:""`
	OdooPassword string `env:"ODOO_PASSWORD" envDefault:""`
	
	// JWS Configuration (Sprint 2)
	JWSEnabled         bool   `env:"JWS_ENABLED" envDefault:"true"`
	JWSRequired        bool   `env:"JWS_REQUIRED" envDefault:"true"`
	JWSPrivateKeyPath  string `env:"JWS_PRIVATE_KEY_PATH" envDefault:""`
	JWSPublicKeyPath   string `env:"JWS_PUBLIC_KEY_PATH" envDefault:""`
	JWSPrivateKeyBase64 string `env:"JWS_PRIVATE_KEY_BASE64" envDefault:""`
	JWSPublicKeyBase64  string `env:"JWS_PUBLIC_KEY_BASE64" envDefault:""`
	JWSKID              string `env:"JWS_KID" envDefault:"key-2025-Q1"`
	
	// Ledger Configuration (Sprint 2)
	LedgerEnabled bool `env:"LEDGER_ENABLED" envDefault:"true"`
}

// Load charge la configuration depuis les variables d'environnement
func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// LoadOrDie charge la configuration ou termine le programme en cas d'erreur
func LoadOrDie() Config {
	cfg, err := Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}
	return cfg
}

// GetPort retourne le port depuis la config ou la variable d'environnement
func GetPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		return "8080"
	}
	return port
}

