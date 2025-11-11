package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// WebhookEvent représente un événement webhook
type WebhookEvent struct {
	EventType   string                 `json:"event_type"`
	DocumentID  string                 `json:"document_id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Payload     map[string]interface{} `json:"payload"`
	WebhookURL  string                 `json:"webhook_url"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
}

// Queue gère la queue Redis pour les webhooks
type Queue struct {
	client *redis.Client
	log    zerolog.Logger
	queueName string
}

// QueueConfig configuration pour la queue
type QueueConfig struct {
	RedisURL   string
	QueueName  string
	Logger     zerolog.Logger
}

// NewQueue crée une nouvelle queue webhook
func NewQueue(cfg QueueConfig) (*Queue, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	
	// Tester la connexion
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	queueName := cfg.QueueName
	if queueName == "" {
		queueName = "dorevia:webhooks"
	}

	return &Queue{
		client:    client,
		log:       cfg.Logger,
		queueName: queueName,
	}, nil
}

// Enqueue ajoute un événement webhook à la queue
func (q *Queue) Enqueue(ctx context.Context, event *WebhookEvent) error {
	// Sérialiser l'événement
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Ajouter à la queue Redis (LPUSH)
	err = q.client.LPush(ctx, q.queueName, data).Err()
	if err != nil {
		return fmt.Errorf("failed to enqueue event: %w", err)
	}

	q.log.Info().
		Str("event_type", event.EventType).
		Str("webhook_url", event.WebhookURL).
		Msg("Webhook event enqueued")

	return nil
}

// Dequeue récupère un événement de la queue (bloquant)
func (q *Queue) Dequeue(ctx context.Context, timeout time.Duration) (*WebhookEvent, error) {
	// BRPOP avec timeout (bloquant)
	result, err := q.client.BRPop(ctx, timeout, q.queueName).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Timeout, pas d'événement
		}
		return nil, fmt.Errorf("failed to dequeue event: %w", err)
	}

	if len(result) < 2 {
		return nil, fmt.Errorf("invalid queue result")
	}

	// Désérialiser l'événement
	var event WebhookEvent
	if err := json.Unmarshal([]byte(result[1]), &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return &event, nil
}

// Close ferme la connexion Redis
func (q *Queue) Close() error {
	return q.client.Close()
}

// GetQueueLength retourne le nombre d'éléments dans la queue
func (q *Queue) GetQueueLength(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.queueName).Result()
}

