package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/webhooks"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestRedisClient crée un client Redis de test (mock ou réel)
// Pour les tests, on peut utiliser un Redis en mémoire ou un mock
func createTestRedisClient(t *testing.T) *redis.Client {
	// Utiliser Redis local si disponible, sinon skip le test
	opts := &redis.Options{
		Addr: "localhost:6379",
		DB:   15, // DB de test
	}
	client := redis.NewClient(opts)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	
	// Nettoyer la DB de test
	client.FlushDB(ctx)
	
	return client
}

// TestNewQueue teste la création d'une queue
func TestNewQueue(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()
	
	// Créer une queue avec un client Redis existant
	// Note: NewQueue attend une URL, mais pour les tests on peut créer directement
	cfg := webhooks.QueueConfig{
		RedisURL:  "redis://localhost:6379/15",
		QueueName: "test:webhooks",
		Logger:    zerolog.Nop(),
	}
	
	queue, err := webhooks.NewQueue(cfg)
	require.NoError(t, err)
	assert.NotNil(t, queue)
	defer queue.Close()
}

// TestQueue_EnqueueDequeue teste l'enqueue et dequeue
func TestQueue_EnqueueDequeue(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()
	
	cfg := webhooks.QueueConfig{
		RedisURL:  "redis://localhost:6379/15",
		QueueName: "test:webhooks",
		Logger:    zerolog.Nop(),
	}
	
	queue, err := webhooks.NewQueue(cfg)
	require.NoError(t, err)
	defer queue.Close()
	
	ctx := context.Background()
	
	// Créer un événement
	event := &webhooks.WebhookEvent{
		EventType:  "document.vaulted",
		DocumentID: "doc-123",
		Timestamp:  time.Now(),
		Payload: map[string]interface{}{
			"document_id": "doc-123",
			"sha256":      "abc123",
		},
		WebhookURL: "https://example.com/webhook",
		RetryCount: 0,
		MaxRetries: 5,
	}
	
	// Enqueue
	err = queue.Enqueue(ctx, event)
	require.NoError(t, err)
	
	// Dequeue
	dequeued, err := queue.Dequeue(ctx, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, dequeued)
	
	assert.Equal(t, event.EventType, dequeued.EventType)
	assert.Equal(t, event.DocumentID, dequeued.DocumentID)
	assert.Equal(t, event.WebhookURL, dequeued.WebhookURL)
}

// TestQueue_GetQueueLength teste GetQueueLength
func TestQueue_GetQueueLength(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()
	
	cfg := webhooks.QueueConfig{
		RedisURL:  "redis://localhost:6379/15",
		QueueName: "test:webhooks",
		Logger:    zerolog.Nop(),
	}
	
	queue, err := webhooks.NewQueue(cfg)
	require.NoError(t, err)
	defer queue.Close()
	
	ctx := context.Background()
	
	// Queue vide
	length, err := queue.GetQueueLength(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), length)
	
	// Ajouter des événements
	event1 := &webhooks.WebhookEvent{
		EventType:  "document.vaulted",
		DocumentID: "doc-1",
		Timestamp:  time.Now(),
		Payload:    map[string]interface{}{},
		WebhookURL: "https://example.com/webhook",
	}
	event2 := &webhooks.WebhookEvent{
		EventType:  "document.vaulted",
		DocumentID: "doc-2",
		Timestamp:  time.Now(),
		Payload:    map[string]interface{}{},
		WebhookURL: "https://example.com/webhook",
	}
	
	queue.Enqueue(ctx, event1)
	queue.Enqueue(ctx, event2)
	
	length, err = queue.GetQueueLength(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), length)
}

// TestNewWorker teste la création d'un worker
func TestNewWorker(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()
	
	cfg := webhooks.QueueConfig{
		RedisURL:  "redis://localhost:6379/15",
		QueueName: "test:webhooks",
		Logger:    zerolog.Nop(),
	}
	
	queue, err := webhooks.NewQueue(cfg)
	require.NoError(t, err)
	defer queue.Close()
	
	workerCfg := webhooks.WorkerConfig{
		Queue:     queue,
		SecretKey: "test-secret",
		Workers:   2,
		Logger:    zerolog.Nop(),
	}
	
	worker := webhooks.NewWorker(workerCfg)
	assert.NotNil(t, worker)
}

// TestNewManager teste la création d'un manager
func TestNewManager(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()
	
	cfg := webhooks.QueueConfig{
		RedisURL:  "redis://localhost:6379/15",
		QueueName: "test:webhooks",
		Logger:    zerolog.Nop(),
	}
	
	queue, err := webhooks.NewQueue(cfg)
	require.NoError(t, err)
	defer queue.Close()
	
	workerCfg := webhooks.WorkerConfig{
		Queue:     queue,
		SecretKey: "test-secret",
		Workers:   1,
		Logger:    zerolog.Nop(),
	}
	
	worker := webhooks.NewWorker(workerCfg)
	
	webhookURLs := map[string][]string{
		webhooks.EventTypeDocumentVaulted: {"https://example.com/webhook"},
	}
	
	managerCfg := webhooks.ManagerConfig{
		Queue:       queue,
		Worker:     worker,
		WebhookURLs: webhookURLs,
		Logger:      zerolog.Nop(),
	}
	
	manager := webhooks.NewManager(managerCfg)
	assert.NotNil(t, manager)
}

// TestManager_EmitEvent teste EmitEvent
func TestManager_EmitEvent(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()
	
	cfg := webhooks.QueueConfig{
		RedisURL:  "redis://localhost:6379/15",
		QueueName: "test:webhooks",
		Logger:    zerolog.Nop(),
	}
	
	queue, err := webhooks.NewQueue(cfg)
	require.NoError(t, err)
	defer queue.Close()
	
	webhookURLs := map[string][]string{
		webhooks.EventTypeDocumentVaulted: {"https://example.com/webhook"},
	}
	
	managerCfg := webhooks.ManagerConfig{
		Queue:       queue,
		WebhookURLs: webhookURLs,
		Logger:      zerolog.Nop(),
	}
	
	manager := webhooks.NewManager(managerCfg)
	
	ctx := context.Background()
	payload := map[string]interface{}{
		"document_id": "doc-123",
		"sha256":      "abc123",
	}
	
	err = manager.EmitEvent(ctx, webhooks.EventTypeDocumentVaulted, "doc-123", payload)
	require.NoError(t, err)
	
	// Vérifier que l'événement est dans la queue
	length, err := queue.GetQueueLength(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), length)
	
	// Déqueuer et vérifier
	event, err := queue.Dequeue(ctx, 1*time.Second)
	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, webhooks.EventTypeDocumentVaulted, event.EventType)
	assert.Equal(t, "doc-123", event.DocumentID)
}

// TestManager_EmitEvent_NoURLs teste EmitEvent sans URLs configurées
func TestManager_EmitEvent_NoURLs(t *testing.T) {
	client := createTestRedisClient(t)
	defer client.Close()
	
	cfg := webhooks.QueueConfig{
		RedisURL:  "redis://localhost:6379/15",
		QueueName: "test:webhooks",
		Logger:    zerolog.Nop(),
	}
	
	queue, err := webhooks.NewQueue(cfg)
	require.NoError(t, err)
	defer queue.Close()
	
	managerCfg := webhooks.ManagerConfig{
		Queue:       queue,
		WebhookURLs: map[string][]string{},
		Logger:      zerolog.Nop(),
	}
	
	manager := webhooks.NewManager(managerCfg)
	
	ctx := context.Background()
	payload := map[string]interface{}{
		"document_id": "doc-123",
	}
	
	// Ne devrait pas générer d'erreur, juste ne rien faire
	err = manager.EmitEvent(ctx, webhooks.EventTypeDocumentVaulted, "doc-123", payload)
	require.NoError(t, err)
	
	// Vérifier que la queue est vide
	length, err := queue.GetQueueLength(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), length)
}

// TestWebhookEvent_MarshalUnmarshal teste la sérialisation JSON
func TestWebhookEvent_MarshalUnmarshal(t *testing.T) {
	event := &webhooks.WebhookEvent{
		EventType:  "document.vaulted",
		DocumentID: "doc-123",
		Timestamp:  time.Now(),
		Payload: map[string]interface{}{
			"document_id": "doc-123",
			"sha256":      "abc123",
		},
		WebhookURL: "https://example.com/webhook",
		RetryCount: 0,
		MaxRetries: 5,
	}
	
	// Marshal
	data, err := json.Marshal(event)
	require.NoError(t, err)
	
	// Unmarshal
	var unmarshaled webhooks.WebhookEvent
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)
	
	assert.Equal(t, event.EventType, unmarshaled.EventType)
	assert.Equal(t, event.DocumentID, unmarshaled.DocumentID)
	assert.Equal(t, event.WebhookURL, unmarshaled.WebhookURL)
}

