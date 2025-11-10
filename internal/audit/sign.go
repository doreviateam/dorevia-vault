package audit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/rs/zerolog"
)

// DailyHash représente le hash cumulé d'un jour avec sa signature JWS
type DailyHash struct {
	Date      string `json:"date"`       // YYYY-MM-DD
	Hash      string `json:"hash"`        // SHA256 cumulé (hex)
	JWS       string `json:"jws"`         // Signature JWS du hash
	LineCount int64  `json:"line_count"`  // Nombre de lignes signées
	Timestamp string `json:"timestamp"`   // RFC3339 de la signature
}

// Signer gère la signature journalière des logs d'audit
type Signer struct {
	logger     *Logger
	jwsService *crypto.Service
	log        zerolog.Logger
}

// NewSigner crée un nouveau signer pour les logs d'audit
func NewSigner(logger *Logger, jwsService *crypto.Service, log zerolog.Logger) *Signer {
	return &Signer{
		logger:     logger,
		jwsService: jwsService,
		log:        log,
	}
}

// SignDailyLog signe le fichier de log d'une date donnée avec hash cumulé incrémental
// Algorithme : hash = SHA256(previous_hash + current_line)
// Plus performant que de re-hasher tout le fichier à chaque ligne
func (s *Signer) SignDailyLog(date string) (*DailyHash, error) {
	if s.jwsService == nil {
		return nil, fmt.Errorf("JWS service not available")
	}

	logPath := s.logger.GetLogPath(date)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("log file not found: %s", logPath)
	}

	// Lire le fichier ligne par ligne et calculer le hash cumulé
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	lineCount := int64(0)
	previousHash := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		lineCount++

		// Hash cumulé : SHA256(previous_hash + current_line)
		hash.Reset()
		if previousHash != "" {
			hash.Write([]byte(previousHash))
		}
		hash.Write(line)
		previousHash = hex.EncodeToString(hash.Sum(nil))
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	if lineCount == 0 {
		return nil, fmt.Errorf("log file is empty")
	}

	finalHash := previousHash

	// Signer le hash final avec JWS
	// Utiliser un document_id factice pour la signature (format requis par SignEvidence)
	evidence, err := s.jwsService.SignEvidence(
		fmt.Sprintf("audit-log-%s", date), // document_id factice
		finalHash,                         // sha256
		time.Now().UTC(),                  // timestamp
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign hash: %w", err)
	}

	dailyHash := &DailyHash{
		Date:      date,
		Hash:      finalHash,
		JWS:       evidence,
		LineCount: lineCount,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Sauvegarder la signature dans un fichier .jws
	sigPath := s.logger.GetSignaturePath(date)
	if err := s.saveSignature(sigPath, dailyHash); err != nil {
		return nil, fmt.Errorf("failed to save signature: %w", err)
	}

	s.log.Info().
		Str("date", date).
		Str("hash", finalHash).
		Int64("line_count", lineCount).
		Msg("signed daily audit log")

	return dailyHash, nil
}

// saveSignature sauvegarde la signature dans un fichier JSON
func (s *Signer) saveSignature(path string, dailyHash *DailyHash) error {
	// Créer le répertoire si nécessaire
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create signatures directory: %w", err)
	}

	// Écrire le JSON
	data, err := json.MarshalIndent(dailyHash, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal signature: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write signature file: %w", err)
	}

	return nil
}

// VerifyDailyLog vérifie la signature d'un fichier de log
func (s *Signer) VerifyDailyLog(date string) (bool, error) {
	sigPath := s.logger.GetSignaturePath(date)
	if _, err := os.Stat(sigPath); os.IsNotExist(err) {
		return false, fmt.Errorf("signature file not found: %s", sigPath)
	}

	// Charger la signature
	data, err := os.ReadFile(sigPath)
	if err != nil {
		return false, fmt.Errorf("failed to read signature file: %w", err)
	}

	var dailyHash DailyHash
	if err := json.Unmarshal(data, &dailyHash); err != nil {
		return false, fmt.Errorf("failed to unmarshal signature: %w", err)
	}

	// Vérifier le hash du fichier actuel
	logPath := s.logger.GetLogPath(date)
	file, err := os.Open(logPath)
	if err != nil {
		return false, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	previousHash := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		hash.Reset()
		if previousHash != "" {
			hash.Write([]byte(previousHash))
		}
		hash.Write(line)
		previousHash = hex.EncodeToString(hash.Sum(nil))
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("failed to read log file: %w", err)
	}

	// Comparer les hashs
	if previousHash != dailyHash.Hash {
		return false, fmt.Errorf("hash mismatch: expected %s, got %s", dailyHash.Hash, previousHash)
	}

	// Vérifier la signature JWS
	if s.jwsService == nil {
		return false, fmt.Errorf("JWS service not available")
	}

	evidence, err := s.jwsService.VerifyEvidence(dailyHash.JWS)
	if err != nil {
		return false, fmt.Errorf("failed to verify JWS: %w", err)
	}

	// Vérifier que le hash dans l'evidence correspond
	if evidence == nil {
		return false, fmt.Errorf("invalid JWS evidence")
	}

	if evidence.Sha256 != dailyHash.Hash {
		return false, fmt.Errorf("hash mismatch in JWS evidence")
	}

	return true, nil
}

