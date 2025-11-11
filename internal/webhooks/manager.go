package webhooks

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

// Manager gère l'envoi de webhooks pour différents événements
type Manager struct {
	queue      *Queue
	worker     *Worker
	webhookURLs map[string][]string // event_type -> URLs
	log        zerolog.Logger
}

// ManagerConfig configuration pour le manager
type ManagerConfig struct {
	Queue      *Queue
	Worker     *Worker
	WebhookURLs map[string][]string
	Logger     zerolog.Logger
}

// NewManager crée un nouveau manager de webhooks
func NewManager(cfg ManagerConfig) *Manager {
	return &Manager{
		queue:       cfg.Queue,
		worker:     cfg.Worker,
		webhookURLs: cfg.WebhookURLs,
		log:         cfg.Logger,
	}
}

// EmitEvent émet un événement webhook
func (m *Manager) EmitEvent(ctx context.Context, eventType string, documentID string, payload map[string]interface{}) error {
	// Récupérer les URLs pour cet événement
	urls, ok := m.webhookURLs[eventType]
	if !ok || len(urls) == 0 {
		// Pas de webhooks configurés pour cet événement
		return nil
	}

	// Créer un événement pour chaque URL
	for _, url := range urls {
		event := &WebhookEvent{
			EventType:  eventType,
			DocumentID:  documentID,
			Timestamp:   time.Now(),
			Payload:     payload,
			WebhookURL:  url,
			RetryCount:  0,
			MaxRetries:  5,
		}

		if err := m.queue.Enqueue(ctx, event); err != nil {
			m.log.Error().
				Err(err).
				Str("event_type", eventType).
				Str("webhook_url", url).
				Msg("Failed to enqueue webhook event")
			// Continuer avec les autres URLs même en cas d'erreur
		}
	}

	return nil
}

// Start démarre le worker
func (m *Manager) Start(ctx context.Context) {
	if m.worker != nil {
		m.worker.Start(ctx)
	}
}

// Stop arrête le worker
func (m *Manager) Stop() {
	if m.worker != nil {
		m.worker.Stop()
	}
	if m.queue != nil {
		m.queue.Close()
	}
}

// Event types constants
const (
	EventTypeDocumentVaulted = "document.vaulted"
	EventTypeDocumentVerified = "document.verified"
	EventTypeLedgerAppended = "ledger.appended"
	EventTypeErrorCritical = "error.critical"
)

