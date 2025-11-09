package logger

import (
	"os"

	"github.com/rs/zerolog"
)

// New crée une nouvelle instance de logger avec le niveau spécifié
func New(level string) *zerolog.Logger {
	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	logger := zerolog.New(os.Stdout).
		Level(logLevel).
		With().
		Timestamp().
		Logger()

	return &logger
}

// Default crée un logger avec le niveau par défaut (info)
func Default() *zerolog.Logger {
	return New("info")
}

