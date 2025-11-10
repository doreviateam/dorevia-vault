package unit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/google/uuid"
)

// createTestAuditData crée un jeu de données de test réaliste pour les tests de rapports
func createTestAuditData(logger *audit.Logger, startDate, endDate string) error {
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)

	currentDate := start
	eventCount := 0

	for !currentDate.After(end) {
		dateStr := currentDate.Format("2006-01-02")
		logPath := logger.GetLogPath(dateStr)

		// Créer le répertoire si nécessaire
		dir := filepath.Dir(logPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// Ouvrir le fichier en mode append
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		// Générer des événements variés pour cette journée
		events := generateDayEvents(dateStr, eventCount)
		for _, event := range events {
			data, err := json.Marshal(event)
			if err != nil {
				file.Close()
				return err
			}
			file.Write(data)
			file.WriteString("\n")
			eventCount++
		}

		file.Close()
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return nil
}

// generateDayEvents génère des événements pour une journée
func generateDayEvents(dateStr string, baseCount int) []audit.Event {
	events := []audit.Event{}

	// Générer 10-20 événements par jour pour simuler une activité réaliste
	numEvents := 10 + (baseCount % 11) // Entre 10 et 20 événements

	for i := 0; i < numEvents; i++ {
		hour := 8 + (i % 10) // Entre 8h et 17h
		timestamp := dateStr + "T" + formatHour(hour) + ":00:00Z"

		// Varier les types d'événements
		eventType := getEventType(i)
		status := getEventStatus(i)
		source := getSource(i)

		event := audit.Event{
			Timestamp:  timestamp,
			EventType:  eventType,
			DocumentID: uuid.New().String(),
			RequestID:  uuid.New().String(),
			Source:     source,
			Status:     status,
			DurationMS: getDuration(i),
			Metadata:   getMetadata(eventType, i),
		}

		events = append(events, event)
	}

	return events
}

// getEventType retourne un type d'événement selon l'index
func getEventType(i int) audit.EventType {
	types := []audit.EventType{
		audit.EventTypeDocumentVaulted,
		audit.EventTypeDocumentVaulted,
		audit.EventTypeDocumentVaulted, // Plus fréquent
		audit.EventTypeJWSSigned,
		audit.EventTypeLedgerAppended,
		audit.EventTypeVerificationRun,
		audit.EventTypeDocumentDownloaded,
		audit.EventTypeReconciliationRun,
	}
	return types[i%len(types)]
}

// getEventStatus retourne un statut selon l'index
func getEventStatus(i int) audit.EventStatus {
	// 80% success, 15% idempotent, 5% error
	switch i % 20 {
	case 0, 1, 2:
		return audit.EventStatusError
	case 3, 4:
		return audit.EventStatusIdempotent
	default:
		return audit.EventStatusSuccess
	}
}

// getSource retourne une source selon l'index
func getSource(i int) string {
	sources := []string{"sales", "purchase", "pos", "stock", "sale", "unknown"}
	return sources[i%len(sources)]
}

// getDuration retourne une durée en millisecondes selon l'index
func getDuration(i int) int64 {
	// Durées variées : 50ms à 500ms
	durations := []int64{50, 100, 150, 200, 250, 300, 350, 400, 450, 500}
	return durations[i%len(durations)]
}

// getMetadata retourne des métadonnées selon le type d'événement
func getMetadata(eventType audit.EventType, i int) map[string]interface{} {
	metadata := map[string]interface{}{}

	switch eventType {
	case audit.EventTypeDocumentVaulted:
		metadata["filename"] = "invoice_" + formatInt(i) + ".pdf"
		metadata["size_bytes"] = 10000 + (i * 1000)
		metadata["content_type"] = "application/pdf"
		if i%20 < 3 {
			metadata["error"] = "storage error"
		}

	case audit.EventTypeJWSSigned:
		metadata["kid"] = "key-2025-Q1"
		if i%20 < 1 {
			metadata["error"] = "JWS signing failed"
		}

	case audit.EventTypeLedgerAppended:
		metadata["hash"] = "abc123" + formatInt(i)
		if i%20 < 1 {
			metadata["error"] = "ledger append failed"
		}

	case audit.EventTypeReconciliationRun:
		metadata["orphan_files_found"] = float64(i % 5)
		metadata["orphan_files_fixed"] = float64((i % 5) - 1)
		if metadata["orphan_files_fixed"].(float64) < 0 {
			metadata["orphan_files_fixed"] = float64(0)
		}
		metadata["documents_fixed"] = float64(i % 3)

	case audit.EventTypeVerificationRun:
		metadata["verified"] = true
		metadata["integrity_ok"] = i%10 != 0

	case audit.EventTypeDocumentDownloaded:
		metadata["ip_address"] = "192.168.1." + formatInt(i%255)

	default:
		metadata["test"] = true
	}

	return metadata
}

// formatHour formate une heure sur 2 chiffres
func formatHour(h int) string {
	return fmt.Sprintf("%02d", h)
}

// formatInt formate un entier en string
func formatInt(i int) string {
	return strconv.Itoa(i)
}

// createTestSignatures crée des signatures de test pour une période
func createTestSignatures(logger *audit.Logger, startDate, endDate string) error {
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)

	currentDate := start
	for !currentDate.After(end) {
		dateStr := currentDate.Format("2006-01-02")
		sigPath := logger.GetSignaturePath(dateStr)

		// Créer le répertoire si nécessaire
		dir := filepath.Dir(sigPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// Créer une signature de test
		dailyHash := audit.DailyHash{
			Date:      dateStr,
			Hash:      "test-hash-" + dateStr,
			JWS:       "test-jws-" + dateStr,
			LineCount: int64(10 + (currentDate.Day() % 20)),
			Timestamp: dateStr + "T23:59:59Z",
		}

		data, err := json.Marshal(dailyHash)
		if err != nil {
			return err
		}

		if err := os.WriteFile(sigPath, data, 0644); err != nil {
			return err
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return nil
}


