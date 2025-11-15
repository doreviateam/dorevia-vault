package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/config"
	"github.com/doreviateam/dorevia-vault/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPosTicketsService est un mock pour services.PosTicketsService
type MockPosTicketsService struct {
	mock.Mock
}

func (m *MockPosTicketsService) Ingest(ctx context.Context, input services.PosTicketInput) (*services.PosTicketResult, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.PosTicketResult), args.Error(1)
}

func TestPosTicketsHandler_Success(t *testing.T) {
	app := fiber.New()
	service := new(MockPosTicketsService)
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := zerolog.Nop()

	app.Post("/api/v1/pos-tickets", PosTicketsHandler(service, cfg, &log))

	payload := PosTicketPayload{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/001",
		Ticket: map[string]interface{}{
			"lines": []interface{}{},
		},
	}

	payloadBytes, _ := json.Marshal(payload)

	// Mock : service retourne succès
	result := &services.PosTicketResult{
		ID:        uuid.New(),
		Tenant:    "test-tenant",
		SHA256Hex: "test-hash",
		CreatedAt: time.Now(),
	}
	ledgerHash := "ledger-hash"
	evidenceJWS := "evidence-jws"
	result.LedgerHash = &ledgerHash
	result.EvidenceJWS = &evidenceJWS

	service.On("Ingest", mock.Anything, mock.MatchedBy(func(input services.PosTicketInput) bool {
		return input.Tenant == "test-tenant" &&
			input.SourceModel == "pos.order" &&
			input.SourceID == "POS/001"
	})).Return(result, nil)

	req := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var response PosTicketResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, result.ID.String(), response.ID)
	assert.Equal(t, result.SHA256Hex, response.SHA256Hex)
	assert.Equal(t, result.Tenant, response.Tenant)

	service.AssertExpectations(t)
}

func TestPosTicketsHandler_InvalidJSON(t *testing.T) {
	app := fiber.New()
	service := new(MockPosTicketsService)
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := zerolog.Nop()

	app.Post("/api/v1/pos-tickets", PosTicketsHandler(service, cfg, &log))

	req := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewReader([]byte("{invalid json}")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	service.AssertNotCalled(t, "Ingest")
}

func TestPosTicketsHandler_MissingFields(t *testing.T) {
	app := fiber.New()
	service := new(MockPosTicketsService)
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := zerolog.Nop()

	app.Post("/api/v1/pos-tickets", PosTicketsHandler(service, cfg, &log))

	tests := []struct {
		name    string
		payload PosTicketPayload
	}{
		{"missing tenant", PosTicketPayload{SourceModel: "pos.order", SourceID: "POS/001", Ticket: map[string]interface{}{}}},
		{"missing source_model", PosTicketPayload{Tenant: "test", SourceID: "POS/001", Ticket: map[string]interface{}{}}},
		{"missing source_id", PosTicketPayload{Tenant: "test", SourceModel: "pos.order", Ticket: map[string]interface{}{}}},
		{"missing ticket", PosTicketPayload{Tenant: "test", SourceModel: "pos.order", SourceID: "POS/001"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewReader(payloadBytes))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)

			require.NoError(t, err)
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

			service.AssertNotCalled(t, "Ingest")
		})
	}
}

func TestPosTicketsHandler_PayloadTooLarge(t *testing.T) {
	app := fiber.New()
	service := new(MockPosTicketsService)
	cfg := &config.Config{PosTicketMaxSizeBytes: 100} // Petite limite
	log := zerolog.Nop()

	app.Post("/api/v1/pos-tickets", PosTicketsHandler(service, cfg, &log))

	// Créer un payload trop volumineux
	largePayload := make([]byte, 200)
	req := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewReader(largePayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusRequestEntityTooLarge, resp.StatusCode)

	service.AssertNotCalled(t, "Ingest")
}

func TestPosTicketsHandler_ServiceError(t *testing.T) {
	app := fiber.New()
	service := new(MockPosTicketsService)
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := zerolog.Nop()

	app.Post("/api/v1/pos-tickets", PosTicketsHandler(service, cfg, &log))

	payload := PosTicketPayload{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/001",
		Ticket:      map[string]interface{}{},
	}
	payloadBytes, _ := json.Marshal(payload)

	// Mock : service retourne erreur
	service.On("Ingest", mock.Anything, mock.Anything).Return(nil, errors.New("service error"))

	req := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	service.AssertExpectations(t)
}

func TestPosTicketsHandler_Mapping(t *testing.T) {
	app := fiber.New()
	service := new(MockPosTicketsService)
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := zerolog.Nop()

	app.Post("/api/v1/pos-tickets", PosTicketsHandler(service, cfg, &log))

	payload := PosTicketPayload{
		Tenant:       "test-tenant",
		SourceSystem: "custom_system",
		SourceModel:  "pos.order",
		SourceID:     "POS/001",
		Currency:     stringPtr("EUR"),
		TotalInclTax: floatPtr(12.50),
		PosSession:   stringPtr("SESSION/001"),
		Cashier:      stringPtr("Test Cashier"),
		Location:     stringPtr("Test Location"),
		Ticket: map[string]interface{}{
			"lines": []interface{}{},
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	result := &services.PosTicketResult{
		ID:        uuid.New(),
		Tenant:    "test-tenant",
		SHA256Hex: "test-hash",
		CreatedAt: time.Now(),
	}

	// Vérifier que le mapping est correct
	service.On("Ingest", mock.Anything, mock.MatchedBy(func(input services.PosTicketInput) bool {
		return input.Tenant == "test-tenant" &&
			input.SourceSystem == "custom_system" &&
			input.SourceModel == "pos.order" &&
			input.SourceID == "POS/001" &&
			*input.Currency == "EUR" &&
			*input.TotalInclTax == 12.50 &&
			*input.PosSession == "SESSION/001" &&
			*input.Cashier == "Test Cashier" &&
			*input.Location == "Test Location"
	})).Return(result, nil)

	req := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	service.AssertExpectations(t)
}

func TestPosTicketsHandler_DefaultSourceSystem(t *testing.T) {
	app := fiber.New()
	service := new(MockPosTicketsService)
	cfg := &config.Config{PosTicketMaxSizeBytes: 65536}
	log := zerolog.Nop()

	app.Post("/api/v1/pos-tickets", PosTicketsHandler(service, cfg, &log))

	payload := PosTicketPayload{
		Tenant:      "test-tenant",
		SourceModel: "pos.order",
		SourceID:    "POS/001",
		// SourceSystem non fourni
		Ticket: map[string]interface{}{},
	}
	payloadBytes, _ := json.Marshal(payload)

	result := &services.PosTicketResult{
		ID:        uuid.New(),
		Tenant:    "test-tenant",
		SHA256Hex: "test-hash",
		CreatedAt: time.Now(),
	}

	// Vérifier que source_system par défaut est "odoo_pos"
	service.On("Ingest", mock.Anything, mock.MatchedBy(func(input services.PosTicketInput) bool {
		return input.SourceSystem == "odoo_pos"
	})).Return(result, nil)

	req := httptest.NewRequest("POST", "/api/v1/pos-tickets", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	service.AssertExpectations(t)
}

func TestGetPosTicket(t *testing.T) {
	app := fiber.New()
	app.Get("/api/v1/pos-tickets", GetPosTicket)

	req := httptest.NewRequest("GET", "/api/v1/pos-tickets", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusMethodNotAllowed, resp.StatusCode)
	assert.Equal(t, "POST", resp.Header.Get("Allow"))
}

// Helpers
func stringPtr(s string) *string {
	return &s
}

func floatPtr(f float64) *float64 {
	return &f
}

