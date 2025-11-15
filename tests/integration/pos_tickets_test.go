package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/config"
	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/doreviateam/dorevia-vault/internal/ledger"
	"github.com/doreviateam/dorevia-vault/internal/metrics"
	"github.com/doreviateam/dorevia-vault/internal/services"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/doreviateam/dorevia-vault/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB crée une connexion DB de test
func setupTestDB(t *testing.T) *storage.DB {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	log := logger.New("error") // Log minimal pour tests
	ctx := context.Background()
	db, err := storage.NewDB(ctx, dbURL, log)
	require.NoError(t, err)

	// Nettoyer la table documents avant le test
	_, err = db.Pool.Exec(ctx, "DELETE FROM documents WHERE source = 'pos'")
	require.NoError(t, err)

	return db
}

// setupTestJWS crée un service JWS de test avec des clés temporaires
func setupTestJWS(t *testing.T) *crypto.Service {
	// Créer des clés temporaires pour le test
	privateKeyPath := "/tmp/test_private_key.pem"
	publicKeyPath := "/tmp/test_public_key.pem"

	// Générer des clés de test (simplifié - en production, utiliser keygen)
	// Pour les tests, on peut utiliser des clés pré-générées ou générer à la volée
	// Ici, on suppose que les clés existent déjà ou on les génère
	// Pour simplifier, on utilise les clés du projet si disponibles
	if _, err := os.Stat("/opt/dorevia-vault/keys/private.pem"); err == nil {
		privateKeyPath = "/opt/dorevia-vault/keys/private.pem"
		publicKeyPath = "/opt/dorevia-vault/keys/public.pem"
	}

	service, err := crypto.NewService(privateKeyPath, publicKeyPath, "test-kid")
	if err != nil {
		t.Skipf("JWS service not available: %v", err)
	}
	return service
}

// getMetricValue récupère la valeur d'une métrique Prometheus
func getMetricValue(metric prometheus.CounterVec, labels ...string) float64 {
	metricProto := &dto.Metric{}
	err := metric.WithLabelValues(labels...).Write(metricProto)
	if err != nil {
		return 0
	}
	return metricProto.Counter.GetValue()
}

// TestPosTickets_EndToEnd teste le flux complet HTTP → DB → Ledger → JWS
func TestPosTickets_EndToEnd(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwsService := setupTestJWS(t)

	// Créer les services
	repo := storage.NewPostgresRepository(db.Pool, logger.New("error"))
	ledgerService := ledger.NewService()
	signer := crypto.NewLocalSigner(jwsService)
	posTicketsService := services.NewPosTicketsService(repo, ledgerService, signer)

	// Créer l'application Fiber
	app := fiber.New()
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := logger.New("error")
	app.Post("/api/v1/pos-tickets", handlers.PosTicketsHandler(posTicketsService, cfg, log))

	// Payload de test
	payload := handlers.PosTicketPayload{
		Tenant:       "test-tenant",
		SourceSystem: "odoo_pos",
		SourceModel:  "pos.order",
		SourceID:     "POS/001",
		Currency:     stringPtr("EUR"),
		TotalInclTax: floatPtr(12.50),
		TotalExclTax: floatPtr(10.42),
		PosSession:   stringPtr("SESSION/001"),
		Cashier:      stringPtr("Test Cashier"),
		Location:     stringPtr("Test Location"),
		Ticket: map[string]interface{}{
			"lines": []interface{}{
				map[string]interface{}{
					"product":  "Item 1",
					"quantity": 1,
					"price":    10.42,
				},
			},
			"payments": []interface{}{
				map[string]interface{}{
					"method": "CB",
					"amount": 12.50,
				},
			},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	// Appel HTTP
	req := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewBuffer(payloadJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Vérifier la réponse HTTP
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var response handlers.PosTicketResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	// Vérifier les champs de réponse
	assert.NotEmpty(t, response.ID)
	assert.Equal(t, "test-tenant", response.Tenant)
	assert.NotEmpty(t, response.SHA256Hex)
	assert.NotNil(t, response.EvidenceJWS)
	assert.NotNil(t, response.LedgerHash)

	// Vérifier dans la DB
	ctx := context.Background()
	doc, err := repo.GetDocumentBySHA256(ctx, response.SHA256Hex)
	require.NoError(t, err)
	require.NotNil(t, doc)

	assert.Equal(t, response.ID, doc.ID.String())
	assert.Equal(t, "pos", *doc.Source)
	assert.Equal(t, "pos.order", *doc.OdooModel)
	assert.Equal(t, "POS/001", *doc.SourceIDText)
	assert.Equal(t, "EUR", *doc.Currency)
	assert.Equal(t, 10.42, *doc.TotalHT)
	assert.Equal(t, 12.50, *doc.TotalTTC)
	assert.Equal(t, "SESSION/001", *doc.PosSession)
	assert.Equal(t, "Test Cashier", *doc.Cashier)
	assert.Equal(t, "Test Location", *doc.Location)
	assert.NotNil(t, doc.EvidenceJWS)
	assert.NotNil(t, doc.LedgerHash)

	// Vérifier le JWS
	if doc.EvidenceJWS != nil {
		evidence, err := jwsService.VerifyEvidence(*doc.EvidenceJWS)
		require.NoError(t, err)
		assert.Equal(t, doc.ID.String(), evidence.DocumentID)
		assert.Equal(t, doc.SHA256Hex, evidence.Sha256)
	}
}

// TestPosTickets_Idempotence teste que deux appels identiques retournent le même résultat
func TestPosTickets_Idempotence(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwsService := setupTestJWS(t)

	// Créer les services
	repo := storage.NewPostgresRepository(db.Pool, logger.New("error"))
	ledgerService := ledger.NewService()
	signer := crypto.NewLocalSigner(jwsService)
	posTicketsService := services.NewPosTicketsService(repo, ledgerService, signer)

	// Créer l'application Fiber
	app := fiber.New()
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := logger.New("error")
	app.Post("/api/v1/pos-tickets", handlers.PosTicketsHandler(posTicketsService, cfg, log))

	// Payload de test
	payload := handlers.PosTicketPayload{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/002",
		PosSession:  stringPtr("SESSION/002"),
		Ticket: map[string]interface{}{
			"lines": []interface{}{
				map[string]interface{}{"product": "Item 1", "quantity": 1},
			},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	// Premier appel
	req1 := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewBuffer(payloadJSON))
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp1.StatusCode)

	var response1 handlers.PosTicketResponse
	err = json.NewDecoder(resp1.Body).Decode(&response1)
	require.NoError(t, err)

	// Deuxième appel (identique)
	req2 := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewBuffer(payloadJSON))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp2.StatusCode) // 200 OK pour idempotence

	var response2 handlers.PosTicketResponse
	err = json.NewDecoder(resp2.Body).Decode(&response2)
	require.NoError(t, err)

	// Vérifier que c'est le même document
	assert.Equal(t, response1.ID, response2.ID)
	assert.Equal(t, response1.SHA256Hex, response2.SHA256Hex)
	assert.Equal(t, response1.LedgerHash, response2.LedgerHash)
	assert.Equal(t, response1.EvidenceJWS, response2.EvidenceJWS)

	// Vérifier qu'un seul document existe dans la DB
	ctx := context.Background()
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM documents WHERE sha256_hex = $1", response1.SHA256Hex).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestPosTickets_Idempotence_MetadataChange teste que changer les métadonnées ne crée pas un nouveau document
func TestPosTickets_Idempotence_MetadataChange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwsService := setupTestJWS(t)

	// Créer les services
	repo := storage.NewPostgresRepository(db.Pool, logger.New("error"))
	ledgerService := ledger.NewService()
	signer := crypto.NewLocalSigner(jwsService)
	posTicketsService := services.NewPosTicketsService(repo, ledgerService, signer)

	// Créer l'application Fiber
	app := fiber.New()
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := logger.New("error")
	app.Post("/api/v1/pos-tickets", handlers.PosTicketsHandler(posTicketsService, cfg, log))

	// Premier appel avec cashier "Cashier 1"
	payload1 := handlers.PosTicketPayload{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/003",
		PosSession:  stringPtr("SESSION/003"),
		Cashier:     stringPtr("Cashier 1"), // Métadonnée
		Ticket: map[string]interface{}{
			"lines": []interface{}{
				map[string]interface{}{"product": "Item 1", "quantity": 1},
			},
		},
	}

	payload1JSON, _ := json.Marshal(payload1)
	req1 := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewBuffer(payload1JSON))
	req1.Header.Set("Content-Type", "application/json")
	resp1, _ := app.Test(req1)
	assert.Equal(t, http.StatusCreated, resp1.StatusCode)

	var response1 handlers.PosTicketResponse
	json.NewDecoder(resp1.Body).Decode(&response1)

	// Deuxième appel avec cashier "Cashier 2" (métadonnée changée, mais ticket identique)
	payload2 := payload1
	payload2.Cashier = stringPtr("Cashier 2") // Métadonnée changée

	payload2JSON, _ := json.Marshal(payload2)
	req2 := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewBuffer(payload2JSON))
	req2.Header.Set("Content-Type", "application/json")
	resp2, _ := app.Test(req2)
	assert.Equal(t, http.StatusOK, resp2.StatusCode) // Idempotent

	var response2 handlers.PosTicketResponse
	json.NewDecoder(resp2.Body).Decode(&response2)

	// Vérifier que c'est le même document (idempotence métier stricte)
	assert.Equal(t, response1.ID, response2.ID)
	assert.Equal(t, response1.SHA256Hex, response2.SHA256Hex)
}

// TestPosTickets_Canonicalization teste que la canonicalisation fonctionne correctement
func TestPosTickets_Canonicalization(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwsService := setupTestJWS(t)

	// Créer les services
	repo := storage.NewPostgresRepository(db.Pool, logger.New("error"))
	ledgerService := ledger.NewService()
	signer := crypto.NewLocalSigner(jwsService)
	posTicketsService := services.NewPosTicketsService(repo, ledgerService, signer)

	// Créer l'application Fiber
	app := fiber.New()
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := logger.New("error")
	app.Post("/api/v1/pos-tickets", handlers.PosTicketsHandler(posTicketsService, cfg, log))

	// Premier appel avec JSON non-canonique (clés non triées, null, 10.0)
	payload1 := handlers.PosTicketPayload{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/004",
		PosSession:  stringPtr("SESSION/004"),
		Ticket: map[string]interface{}{
			"b": 2.0,
			"a": 1,
			"c": nil,
		},
	}

	payload1JSON, _ := json.Marshal(payload1)
	req1 := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewBuffer(payload1JSON))
	req1.Header.Set("Content-Type", "application/json")
	resp1, _ := app.Test(req1)
	assert.Equal(t, http.StatusCreated, resp1.StatusCode)

	var response1 handlers.PosTicketResponse
	json.NewDecoder(resp1.Body).Decode(&response1)

	// Deuxième appel avec JSON canonique (clés triées, null supprimé, 10 normalisé)
	payload2 := handlers.PosTicketPayload{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/004",
		PosSession:  stringPtr("SESSION/004"),
		Ticket: map[string]interface{}{
			"a": 1.0,
			"b": 2,
		},
	}

	payload2JSON, _ := json.Marshal(payload2)
	req2 := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewBuffer(payload2JSON))
	req2.Header.Set("Content-Type", "application/json")
	resp2, _ := app.Test(req2)
	assert.Equal(t, http.StatusOK, resp2.StatusCode) // Idempotent

	var response2 handlers.PosTicketResponse
	json.NewDecoder(resp2.Body).Decode(&response2)

	// Vérifier que le hash est identique (canonicalisation)
	assert.Equal(t, response1.SHA256Hex, response2.SHA256Hex)
}

// TestPosTickets_Metrics teste que les métriques Prometheus sont correctement enregistrées
func TestPosTickets_Metrics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	jwsService := setupTestJWS(t)

	// Créer les services
	repo := storage.NewPostgresRepository(db.Pool, logger.New("error"))
	ledgerService := ledger.NewService()
	signer := crypto.NewLocalSigner(jwsService)
	posTicketsService := services.NewPosTicketsService(repo, ledgerService, signer)

	// Créer l'application Fiber
	app := fiber.New()
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := logger.New("error")
	app.Post("/api/v1/pos-tickets", handlers.PosTicketsHandler(posTicketsService, cfg, log))

	// Récupérer les valeurs initiales des métriques
	initialSuccess := getMetricValue(metrics.DocumentsVaulted, "success", "pos")
	initialIdempotent := getMetricValue(metrics.DocumentsVaulted, "idempotent", "pos")
	initialError := getMetricValue(metrics.DocumentsVaulted, "error", "pos")

	// Payload de test
	payload := handlers.PosTicketPayload{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/005",
		PosSession:  stringPtr("SESSION/005"),
		Ticket: map[string]interface{}{
			"lines": []interface{}{
				map[string]interface{}{"product": "Item 1", "quantity": 1},
			},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	// Premier appel (succès)
	req1 := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewBuffer(payloadJSON))
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := app.Test(req1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp1.StatusCode)

	// Vérifier que la métrique "success" a été incrémentée
	time.Sleep(100 * time.Millisecond) // Attendre traitement asynchrone
	successAfter := getMetricValue(metrics.DocumentsVaulted, "success", "pos")
	assert.Greater(t, successAfter, initialSuccess, "Success metric should be incremented")

	// Deuxième appel (idempotent)
	req2 := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewBuffer(payloadJSON))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := app.Test(req2)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Vérifier que la métrique "idempotent" a été incrémentée
	time.Sleep(100 * time.Millisecond)
	idempotentAfter := getMetricValue(metrics.DocumentsVaulted, "idempotent", "pos")
	assert.Greater(t, idempotentAfter, initialIdempotent, "Idempotent metric should be incremented")

	// Vérifier que la métrique "error" n'a pas changé
	errorAfter := getMetricValue(metrics.DocumentsVaulted, "error", "pos")
	assert.Equal(t, initialError, errorAfter, "Error metric should not change")
}

// Helpers
func stringPtr(s string) *string {
	return &s
}

func floatPtr(f float64) *float64 {
	return &f
}

