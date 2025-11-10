package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// ExportFormat représente le format d'export
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
)

// ExportOptions contient les options d'export
type ExportOptions struct {
	From   string        // YYYY-MM-DD (optionnel, défaut: aujourd'hui)
	To     string        // YYYY-MM-DD (optionnel, défaut: aujourd'hui)
	Page   int           // Numéro de page (défaut: 1)
	Limit  int           // Nombre de lignes par page (défaut: 1000, max: 10000)
	Format ExportFormat  // Format d'export (défaut: json)
}

// ExportResult contient le résultat de l'export
type ExportResult struct {
	Events      []Event `json:"events"`
	Total       int64   `json:"total"`        // Nombre total d'événements
	Page        int     `json:"page"`         // Page actuelle
	Limit       int     `json:"limit"`        // Limite par page
	TotalPages  int     `json:"total_pages"`   // Nombre total de pages
	HasNext     bool    `json:"has_next"`      // Y a-t-il une page suivante ?
	HasPrevious bool    `json:"has_previous"`  // Y a-t-il une page précédente ?
}

// Exporter gère l'export des logs d'audit
type Exporter struct {
	logger *Logger
}

// NewExporter crée un nouvel exporteur
func NewExporter(logger *Logger) *Exporter {
	return &Exporter{
		logger: logger,
	}
}

// Export exporte les logs selon les options
func (e *Exporter) Export(opts ExportOptions) (*ExportResult, error) {
	// Valeurs par défaut
	if opts.Limit == 0 {
		opts.Limit = 1000
	}
	if opts.Limit > 10000 {
		opts.Limit = 10000 // Limite max
	}
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Format == "" {
		opts.Format = ExportFormatJSON
	}

	// Dates par défaut (aujourd'hui)
	now := time.Now().UTC()
	if opts.From == "" {
		opts.From = now.Format("2006-01-02")
	}
	if opts.To == "" {
		opts.To = now.Format("2006-01-02")
	}

	// Valider les dates
	fromDate, err := time.Parse("2006-01-02", opts.From)
	if err != nil {
		return nil, fmt.Errorf("invalid from date: %w", err)
	}
	toDate, err := time.Parse("2006-01-02", opts.To)
	if err != nil {
		return nil, fmt.Errorf("invalid to date: %w", err)
	}
	if fromDate.After(toDate) {
		return nil, fmt.Errorf("from date must be before or equal to to date")
	}

	// Collecter tous les événements dans la plage de dates
	allEvents := make([]Event, 0)
	currentDate := fromDate

	for !currentDate.After(toDate) {
		dateStr := currentDate.Format("2006-01-02")
		logPath := e.logger.GetLogPath(dateStr)

		// Lire le fichier s'il existe
		if _, err := os.Stat(logPath); err == nil {
			events, err := e.readLogFile(logPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read log file %s: %w", dateStr, err)
			}
			allEvents = append(allEvents, events...)
		}

		// Passer au jour suivant
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// Trier par timestamp (plus récent en premier)
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].Timestamp > allEvents[j].Timestamp
	})

	total := int64(len(allEvents))

	// Pagination
	offset := (opts.Page - 1) * opts.Limit
	if offset >= len(allEvents) {
		// Page vide
		return &ExportResult{
			Events:      []Event{},
			Total:       total,
			Page:        opts.Page,
			Limit:       opts.Limit,
			TotalPages:  (int(total) + opts.Limit - 1) / opts.Limit,
			HasNext:     false,
			HasPrevious: opts.Page > 1,
		}, nil
	}

	end := offset + opts.Limit
	if end > len(allEvents) {
		end = len(allEvents)
	}

	pageEvents := allEvents[offset:end]

	totalPages := (int(total) + opts.Limit - 1) / opts.Limit

	return &ExportResult{
		Events:      pageEvents,
		Total:       total,
		Page:        opts.Page,
		Limit:       opts.Limit,
		TotalPages:  totalPages,
		HasNext:     opts.Page < totalPages,
		HasPrevious: opts.Page > 1,
	}, nil
}

// readLogFile lit un fichier JSONL et retourne les événements
func (e *Exporter) readLogFile(path string) ([]Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	events := make([]Event, 0)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Ignorer les lignes invalides (log warning si nécessaire)
			continue
		}

		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return events, nil
}

// ExportToCSV exporte les événements en CSV (format simplifié)
func (e *Exporter) ExportToCSV(result *ExportResult) (string, error) {
	if len(result.Events) == 0 {
		return "timestamp,event_type,document_id,request_id,source,status,duration_ms\n", nil
	}

	var sb strings.Builder
	sb.WriteString("timestamp,event_type,document_id,request_id,source,status,duration_ms\n")

	for _, event := range result.Events {
		sb.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%d\n",
			event.Timestamp,
			event.EventType,
			event.DocumentID,
			event.RequestID,
			event.Source,
			event.Status,
			event.DurationMS,
		))
	}

	return sb.String(), nil
}

// ListAvailableDates liste les dates disponibles dans les logs
func (e *Exporter) ListAvailableDates() ([]string, error) {
	logsDir := e.logger.logsDir
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs directory: %w", err)
	}

	dates := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, "audit-") || !strings.HasSuffix(name, ".log") {
			continue
		}

		// Extraire la date : audit-YYYY-MM-DD.log
		dateStr := strings.TrimPrefix(name, "audit-")
		dateStr = strings.TrimSuffix(dateStr, ".log")

		// Valider le format
		if _, err := time.Parse("2006-01-02", dateStr); err == nil {
			dates = append(dates, dateStr)
		}
	}

	// Trier par date (plus récent en premier)
	sort.Slice(dates, func(i, j int) bool {
		return dates[i] > dates[j]
	})

	return dates, nil
}

