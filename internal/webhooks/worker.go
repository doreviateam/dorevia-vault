package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// Worker traite les webhooks de manière asynchrone
type Worker struct {
	queue      *Queue
	httpClient *http.Client
	log        zerolog.Logger
	secretKey  string // Pour signature HMAC
	workers    int    // Nombre de workers parallèles
	stopChan   chan struct{}
}

// WorkerConfig configuration pour le worker
type WorkerConfig struct {
	Queue     *Queue
	SecretKey string
	Workers   int
	Logger    zerolog.Logger
}

// NewWorker crée un nouveau worker webhook
func NewWorker(cfg WorkerConfig) *Worker {
	workers := cfg.Workers
	if workers <= 0 {
		workers = 3 // Par défaut 3 workers
	}

	return &Worker{
		queue: cfg.Queue,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log:       cfg.Logger,
		secretKey:  cfg.SecretKey,
		workers:   workers,
		stopChan:  make(chan struct{}),
	}
}

// Start démarre les workers
func (w *Worker) Start(ctx context.Context) {
	for i := 0; i < w.workers; i++ {
		go w.workerLoop(ctx, i)
	}
	w.log.Info().Int("workers", w.workers).Msg("Webhook workers started")
}

// Stop arrête les workers
func (w *Worker) Stop() {
	close(w.stopChan)
	w.log.Info().Msg("Webhook workers stopped")
}

// workerLoop boucle principale d'un worker
func (w *Worker) workerLoop(ctx context.Context, workerID int) {
	w.log.Debug().Int("worker_id", workerID).Msg("Worker started")

	for {
		select {
		case <-w.stopChan:
			w.log.Debug().Int("worker_id", workerID).Msg("Worker stopped")
			return
		case <-ctx.Done():
			w.log.Debug().Int("worker_id", workerID).Msg("Worker context cancelled")
			return
		default:
			// Déqueuer un événement (timeout 5 secondes)
			event, err := w.queue.Dequeue(ctx, 5*time.Second)
			if err != nil {
				w.log.Error().Err(err).Int("worker_id", workerID).Msg("Failed to dequeue event")
				time.Sleep(1 * time.Second)
				continue
			}

			if event == nil {
				// Timeout, continuer
				continue
			}

			// Traiter l'événement
			w.processEvent(ctx, event, workerID)
		}
	}
}

// processEvent traite un événement webhook
func (w *Worker) processEvent(ctx context.Context, event *WebhookEvent, workerID int) {
	w.log.Info().
		Int("worker_id", workerID).
		Str("event_type", event.EventType).
		Str("webhook_url", event.WebhookURL).
		Int("retry_count", event.RetryCount).
		Msg("Processing webhook event")

	// Préparer le payload
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		w.log.Error().Err(err).Msg("Failed to marshal payload")
		return
	}

	// Créer la requête HTTP
	req, err := http.NewRequestWithContext(ctx, "POST", event.WebhookURL, bytes.NewBuffer(payload))
	if err != nil {
		w.log.Error().Err(err).Msg("Failed to create HTTP request")
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Dorevia-Vault/1.0")
	req.Header.Set("X-Event-Type", event.EventType)
	req.Header.Set("X-Timestamp", event.Timestamp.Format(time.RFC3339))

	// Ajouter signature HMAC si secret key configuré
	if w.secretKey != "" {
		signature := w.generateHMAC(payload)
		req.Header.Set("X-Signature", signature)
	}

	// Envoyer la requête
	resp, err := w.httpClient.Do(req)
	if err != nil {
		w.log.Error().Err(err).Msg("Failed to send webhook")
		w.handleRetry(ctx, event, workerID)
		return
	}
	defer resp.Body.Close()

	// Lire la réponse (limite pour éviter consommation mémoire)
	_, _ = io.CopyN(io.Discard, resp.Body, 1024)

	// Vérifier le statut HTTP
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.log.Info().
			Int("worker_id", workerID).
			Str("event_type", event.EventType).
			Int("status_code", resp.StatusCode).
			Msg("Webhook sent successfully")
	} else {
		w.log.Warn().
			Int("worker_id", workerID).
			Str("event_type", event.EventType).
			Int("status_code", resp.StatusCode).
			Msg("Webhook returned non-2xx status")
		w.handleRetry(ctx, event, workerID)
	}
}

// handleRetry gère les retries avec backoff exponentiel
func (w *Worker) handleRetry(ctx context.Context, event *WebhookEvent, workerID int) {
	event.RetryCount++

	// Vérifier si on a atteint le max de retries
	maxRetries := event.MaxRetries
	if maxRetries == 0 {
		maxRetries = 5 // Par défaut 5 retries
	}

	if event.RetryCount >= maxRetries {
		w.log.Error().
			Int("worker_id", workerID).
			Str("event_type", event.EventType).
			Str("webhook_url", event.WebhookURL).
			Int("retry_count", event.RetryCount).
			Msg("Webhook failed after max retries")
		return
	}

	// Calculer le délai avec backoff exponentiel
	delay := time.Duration(1<<uint(event.RetryCount)) * time.Second
	if delay > 300*time.Second {
		delay = 300 * time.Second // Max 5 minutes
	}

	w.log.Info().
		Int("worker_id", workerID).
		Str("event_type", event.EventType).
		Int("retry_count", event.RetryCount).
		Dur("delay", delay).
		Msg("Scheduling webhook retry")

	// Attendre avant de réessayer
	time.Sleep(delay)

	// Réenqueuer l'événement
	if err := w.queue.Enqueue(ctx, event); err != nil {
		w.log.Error().Err(err).Msg("Failed to re-enqueue webhook event")
	}
}

// generateHMAC génère une signature HMAC-SHA256
func (w *Worker) generateHMAC(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(w.secretKey))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

