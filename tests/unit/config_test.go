package unit

import (
	"os"
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestConfigLoad teste le chargement de la configuration avec valeurs par défaut
func TestConfigLoad(t *testing.T) {
	// Sauvegarder les variables d'environnement actuelles
	originalPort := os.Getenv("PORT")
	originalLogLevel := os.Getenv("LOG_LEVEL")

	// Nettoyer les variables d'environnement pour tester les valeurs par défaut
	os.Unsetenv("PORT")
	os.Unsetenv("LOG_LEVEL")

	// Restaurer après le test
	defer func() {
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		}
		if originalLogLevel != "" {
			os.Setenv("LOG_LEVEL", originalLogLevel)
		}
	}()

	cfg, err := config.Load()
	assert.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "info", cfg.LogLevel)
}

// TestConfigLoadWithEnv teste le chargement avec variables d'environnement
func TestConfigLoadWithEnv(t *testing.T) {
	// Sauvegarder les variables d'environnement actuelles
	originalPort := os.Getenv("PORT")
	originalLogLevel := os.Getenv("LOG_LEVEL")

	// Définir des valeurs de test
	os.Setenv("PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")

	// Restaurer après le test
	defer func() {
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		} else {
			os.Unsetenv("PORT")
		}
		if originalLogLevel != "" {
			os.Setenv("LOG_LEVEL", originalLogLevel)
		} else {
			os.Unsetenv("LOG_LEVEL")
		}
	}()

	cfg, err := config.Load()
	assert.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "debug", cfg.LogLevel)
}

// TestGetPort teste la fonction GetPort
func TestGetPort(t *testing.T) {
	originalPort := os.Getenv("PORT")
	defer func() {
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		} else {
			os.Unsetenv("PORT")
		}
	}()

	// Test avec valeur par défaut
	os.Unsetenv("PORT")
	assert.Equal(t, "8080", config.GetPort())

	// Test avec valeur personnalisée
	os.Setenv("PORT", "3000")
	assert.Equal(t, "3000", config.GetPort())
}

