package config

import (
	"os"

	"github.com/caarlos0/env/v11"
)

// Config contient toute la configuration de l'application
type Config struct {
	Port       string `env:"PORT" envDefault:"8080"`
	LogLevel   string `env:"LOG_LEVEL" envDefault:"info"`
	DatabaseURL string `env:"DATABASE_URL" envDefault:""`
	StorageDir  string `env:"STORAGE_DIR" envDefault:"/opt/dorevia-vault/storage"`
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

