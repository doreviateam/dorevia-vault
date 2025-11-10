package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// RotationConfig contient la configuration de rotation
type RotationConfig struct {
	RetentionDays int           // Nombre de jours de rétention (défaut: 90)
	SignDaily     bool          // Signer les logs quotidiennement (défaut: true)
	Signer        *Signer       // Signer pour signature journalière
	Logger        zerolog.Logger // Logger pour les logs internes
}

// Rotator gère la rotation et la rétention des logs d'audit
type Rotator struct {
	logger *Logger
	config RotationConfig
}

// NewRotator crée un nouveau rotator
func NewRotator(logger *Logger, config RotationConfig) *Rotator {
	if config.RetentionDays == 0 {
		config.RetentionDays = 90
	}

	return &Rotator{
		logger: logger,
		config: config,
	}
}

// RotateDaily signe le log du jour précédent et prépare le nouveau fichier
// À appeler quotidiennement (ex: via cron à 00:00 UTC)
func (r *Rotator) RotateDaily() error {
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	dateStr := yesterday.Format("2006-01-02")

	// Vérifier si le fichier existe
	logPath := r.logger.GetLogPath(dateStr)
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		// Pas de log hier, rien à faire
		r.config.Logger.Debug().Str("date", dateStr).Msg("no log file to rotate")
		return nil
	}

	// Signer le log du jour précédent si configuré
	if r.config.SignDaily && r.config.Signer != nil {
		dailyHash, err := r.config.Signer.SignDailyLog(dateStr)
		if err != nil {
			r.config.Logger.Error().Err(err).Str("date", dateStr).Msg("failed to sign daily log")
			// Ne pas bloquer la rotation si la signature échoue
		} else {
			r.config.Logger.Info().
				Str("date", dateStr).
				Str("hash", dailyHash.Hash).
				Int64("line_count", dailyHash.LineCount).
				Msg("signed daily audit log")
		}
	}

	// S'assurer que le fichier d'aujourd'hui est prêt
	if err := r.logger.Flush(); err != nil {
		return fmt.Errorf("failed to flush current log: %w", err)
	}

	return nil
}

// CleanupOldLogs supprime les logs plus anciens que la période de rétention
func (r *Rotator) CleanupOldLogs() error {
	cutoffDate := time.Now().UTC().AddDate(0, 0, -r.config.RetentionDays)
	logsDir := r.logger.logsDir
	sigsDir := r.logger.sigsDir

	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return fmt.Errorf("failed to read logs directory: %w", err)
	}

	deletedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "audit-") || !strings.HasSuffix(name, ".log") {
			continue
		}

		// Extraire la date
		dateStr := strings.TrimPrefix(name, "audit-")
		dateStr = strings.TrimSuffix(dateStr, ".log")

		logDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Supprimer si plus ancien que la période de rétention
		if logDate.Before(cutoffDate) {
			logPath := filepath.Join(logsDir, name)
			if err := os.Remove(logPath); err != nil {
				r.config.Logger.Error().Err(err).Str("file", logPath).Msg("failed to delete old log file")
				continue
			}

			// Supprimer aussi le fichier de signature correspondant
			sigPath := filepath.Join(sigsDir, name+".jws")
			if _, err := os.Stat(sigPath); err == nil {
				if err := os.Remove(sigPath); err != nil {
					r.config.Logger.Error().Err(err).Str("file", sigPath).Msg("failed to delete old signature file")
				}
			}

			deletedCount++
			r.config.Logger.Debug().Str("date", dateStr).Msg("deleted old audit log")
		}
	}

	if deletedCount > 0 {
		r.config.Logger.Info().Int("count", deletedCount).Msg("cleaned up old audit logs")
	}

	return nil
}

// GetRetentionStats retourne les statistiques de rétention
type RetentionStats struct {
	TotalLogs      int       `json:"total_logs"`
	OldestLogDate  string    `json:"oldest_log_date"`  // YYYY-MM-DD
	NewestLogDate  string    `json:"newest_log_date"`  // YYYY-MM-DD
	RetentionDays  int       `json:"retention_days"`
	LogsToDelete   int       `json:"logs_to_delete"`   // Logs qui seront supprimés au prochain cleanup
}

func (r *Rotator) GetRetentionStats() (*RetentionStats, error) {
	cutoffDate := time.Now().UTC().AddDate(0, 0, -r.config.RetentionDays)
	logsDir := r.logger.logsDir

	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs directory: %w", err)
	}

	dates := make([]time.Time, 0)
	logsToDelete := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "audit-") || !strings.HasSuffix(name, ".log") {
			continue
		}

		dateStr := strings.TrimPrefix(name, "audit-")
		dateStr = strings.TrimSuffix(dateStr, ".log")

		logDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		dates = append(dates, logDate)

		if logDate.Before(cutoffDate) {
			logsToDelete++
		}
	}

	if len(dates) == 0 {
		return &RetentionStats{
			TotalLogs:     0,
			OldestLogDate: "",
			NewestLogDate: "",
			RetentionDays: r.config.RetentionDays,
			LogsToDelete:  0,
		}, nil
	}

	// Trier les dates
	sort.Slice(dates, func(i, j int) bool {
		return dates[i].Before(dates[j])
	})

	return &RetentionStats{
		TotalLogs:     len(dates),
		OldestLogDate: dates[0].Format("2006-01-02"),
		NewestLogDate: dates[len(dates)-1].Format("2006-01-02"),
		RetentionDays: r.config.RetentionDays,
		LogsToDelete:  logsToDelete,
	}, nil
}

