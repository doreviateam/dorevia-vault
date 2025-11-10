package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// EventType représente le type d'événement audit
type EventType string

const (
	EventTypeDocumentVaulted    EventType = "document_vaulted"
	EventTypeJWSSigned          EventType = "jws_signed"
	EventTypeLedgerAppended     EventType = "ledger_appended"
	EventTypeReconciliationRun   EventType = "reconciliation_run"
	EventTypeVerificationRun    EventType = "verification_run"
	EventTypeDocumentDownloaded  EventType = "document_downloaded"
	EventTypeError              EventType = "error"
)

// EventStatus représente le statut de l'événement
type EventStatus string

const (
	EventStatusSuccess   EventStatus = "success"
	EventStatusError     EventStatus = "error"
	EventStatusIdempotent EventStatus = "idempotent"
)

// Event représente un événement d'audit
type Event struct {
	Timestamp  string                 `json:"timestamp"`  // RFC3339
	EventType  EventType              `json:"event_type"`
	DocumentID string                 `json:"document_id,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Source     string                 `json:"source,omitempty"` // sales, purchase, pos, stock, sale, unknown
	Status     EventStatus            `json:"status"`
	DurationMS int64                  `json:"duration_ms,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Logger gère l'écriture des logs d'audit en JSONL
type Logger struct {
	auditDir    string
	logsDir     string
	sigsDir     string
	buffer      []Event
	bufferMutex sync.Mutex
	flushTicker *time.Ticker
	stopChan    chan struct{}
	log         zerolog.Logger
	maxBuffer   int           // Nombre max de lignes avant flush
	flushInterval time.Duration // Intervalle max avant flush
	currentFile *os.File
	currentWriter *bufio.Writer
	currentDate   string // YYYY-MM-DD
	fileMutex    sync.Mutex
}

// Config contient la configuration du logger d'audit
type Config struct {
	AuditDir      string        // Répertoire racine audit (ex: /opt/dorevia-vault/audit)
	MaxBuffer     int           // Nombre max de lignes avant flush (défaut: 1000)
	FlushInterval time.Duration // Intervalle max avant flush (défaut: 10s)
	Logger        zerolog.Logger // Logger pour les logs internes
}

// NewLogger crée un nouveau logger d'audit
func NewLogger(cfg Config) (*Logger, error) {
	if cfg.AuditDir == "" {
		return nil, fmt.Errorf("audit directory not configured")
	}

	if cfg.MaxBuffer == 0 {
		cfg.MaxBuffer = 1000
	}
	if cfg.FlushInterval == 0 {
		cfg.FlushInterval = 10 * time.Second
	}

	logsDir := filepath.Join(cfg.AuditDir, "logs")
	sigsDir := filepath.Join(cfg.AuditDir, "signatures")

	// Créer les répertoires
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}
	if err := os.MkdirAll(sigsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create signatures directory: %w", err)
	}

	l := &Logger{
		auditDir:     cfg.AuditDir,
		logsDir:      logsDir,
		sigsDir:      sigsDir,
		buffer:       make([]Event, 0, cfg.MaxBuffer),
		flushTicker:  time.NewTicker(cfg.FlushInterval),
		stopChan:     make(chan struct{}),
		log:          cfg.Logger,
		maxBuffer:    cfg.MaxBuffer,
		flushInterval: cfg.FlushInterval,
	}

	// Démarrer le goroutine de flush périodique
	go l.flushLoop()

	return l, nil
}

// Log enregistre un événement d'audit (thread-safe)
func (l *Logger) Log(event Event) error {
	// Ajouter timestamp si absent
	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	l.bufferMutex.Lock()
	l.buffer = append(l.buffer, event)
	shouldFlush := len(l.buffer) >= l.maxBuffer
	l.bufferMutex.Unlock()

	// Flush si buffer plein
	if shouldFlush {
		return l.Flush()
	}

	return nil
}

// Flush écrit le buffer dans le fichier JSONL
func (l *Logger) Flush() error {
	l.bufferMutex.Lock()
	if len(l.buffer) == 0 {
		l.bufferMutex.Unlock()
		return nil
	}
	events := make([]Event, len(l.buffer))
	copy(events, l.buffer)
	l.buffer = l.buffer[:0]
	l.bufferMutex.Unlock()

	// Obtenir le fichier pour la date actuelle
	if err := l.ensureFile(); err != nil {
		return fmt.Errorf("failed to ensure file: %w", err)
	}

	// Écrire les événements en JSONL
	l.fileMutex.Lock()
	defer l.fileMutex.Unlock()

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			l.log.Error().Err(err).Interface("event", event).Msg("failed to marshal audit event")
			continue
		}

		if _, err := l.currentWriter.Write(data); err != nil {
			return fmt.Errorf("failed to write audit event: %w", err)
		}
		if _, err := l.currentWriter.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	// Flush du buffer
	if err := l.currentWriter.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer: %w", err)
	}

	// Sync du fichier pour garantir l'écriture disque
	if err := l.currentFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// ensureFile s'assure que le fichier pour la date actuelle est ouvert
func (l *Logger) ensureFile() error {
	now := time.Now().UTC()
	today := now.Format("2006-01-02")

	l.fileMutex.Lock()
	defer l.fileMutex.Unlock()

	// Si le fichier actuel est pour aujourd'hui, rien à faire
	if l.currentFile != nil && l.currentDate == today {
		return nil
	}

	// Fermer l'ancien fichier si nécessaire
	if l.currentFile != nil {
		if err := l.currentWriter.Flush(); err != nil {
			l.log.Error().Err(err).Msg("failed to flush old file")
		}
		if err := l.currentFile.Close(); err != nil {
			l.log.Error().Err(err).Msg("failed to close old file")
		}
	}

	// Ouvrir le nouveau fichier
	filename := filepath.Join(l.logsDir, fmt.Sprintf("audit-%s.log", today))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit file: %w", err)
	}

	l.currentFile = file
	l.currentWriter = bufio.NewWriter(file)
	l.currentDate = today

	l.log.Debug().Str("file", filename).Msg("opened audit log file")

	return nil
}

// flushLoop exécute le flush périodique
func (l *Logger) flushLoop() {
	for {
		select {
		case <-l.flushTicker.C:
			if err := l.Flush(); err != nil {
				l.log.Error().Err(err).Msg("failed to flush audit buffer")
			}
		case <-l.stopChan:
			return
		}
	}
}

// Close ferme le logger et flush le buffer restant
func (l *Logger) Close() error {
	// Arrêter le ticker
	l.flushTicker.Stop()
	close(l.stopChan)

	// Flush final
	if err := l.Flush(); err != nil {
		l.log.Error().Err(err).Msg("failed to flush on close")
	}

	// Fermer le fichier
	l.fileMutex.Lock()
	defer l.fileMutex.Unlock()

	if l.currentWriter != nil {
		if err := l.currentWriter.Flush(); err != nil {
			l.log.Error().Err(err).Msg("failed to flush writer on close")
		}
	}
	if l.currentFile != nil {
		if err := l.currentFile.Close(); err != nil {
			l.log.Error().Err(err).Msg("failed to close file on close")
		}
	}

	return nil
}

// GetLogPath retourne le chemin du fichier de log pour une date donnée
func (l *Logger) GetLogPath(date string) string {
	return filepath.Join(l.logsDir, fmt.Sprintf("audit-%s.log", date))
}

// GetSignaturePath retourne le chemin du fichier de signature pour une date donnée
func (l *Logger) GetSignaturePath(date string) string {
	return filepath.Join(l.sigsDir, fmt.Sprintf("audit-%s.log.jws", date))
}

