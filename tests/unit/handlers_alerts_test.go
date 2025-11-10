package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlertsWebhookHandler(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name           string
		payload        handlers.AlertmanagerWebhookPayload
		odooExporter   *audit.OdooExporter
		expectedStatus int
		expectedBody   map[string]interface{}
	}{
		{
			name: "valid payload with odoo exporter",
			payload: handlers.AlertmanagerWebhookPayload{
				Version:  "4",
				Status:   "firing",
				Receiver: "default",
				Alerts: []handlers.Alert{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "HighDocumentErrorRate",
							"severity":   "warning",
						},
						Annotations: map[string]string{
							"summary":     "Taux d'erreur élevé",
							"description": "Plus de 10% d'erreurs",
						},
					},
				},
			},
			odooExporter:   nil, // Pas d'exporteur pour ce test
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status":  "ok",
				"message": "Processed 1 alerts",
			},
		},
		{
			name: "multiple alerts",
			payload: handlers.AlertmanagerWebhookPayload{
				Version:  "4",
				Status:   "firing",
				Receiver: "default",
				Alerts: []handlers.Alert{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "Alert1",
							"severity":   "warning",
						},
						Annotations: map[string]string{
							"summary": "Alert 1",
						},
					},
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "Alert2",
							"severity":   "critical",
						},
						Annotations: map[string]string{
							"summary": "Alert 2",
						},
					},
				},
			},
			odooExporter:   nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status":  "ok",
				"message": "Processed 2 alerts",
			},
		},
		{
			name: "resolved alert (not exported)",
			payload: handlers.AlertmanagerWebhookPayload{
				Version:  "4",
				Status:   "resolved",
				Receiver: "default",
				Alerts: []handlers.Alert{
					{
						Status: "resolved",
						Labels: map[string]string{
							"alertname": "TestAlert",
							"severity":   "warning",
						},
						Annotations: map[string]string{
							"summary": "Resolved alert",
						},
					},
				},
			},
			odooExporter:   nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status":  "ok",
				"message": "Processed 1 alerts",
			},
		},
		{
			name: "empty alerts",
			payload: handlers.AlertmanagerWebhookPayload{
				Version:  "4",
				Status:   "firing",
				Receiver: "default",
				Alerts:   []handlers.Alert{},
			},
			odooExporter:   nil,
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status":  "ok",
				"message": "Processed 0 alerts",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/api/v1/alerts/webhook", handlers.AlertsWebhookHandler(tt.odooExporter, &logger))

			// Sérialiser le payload
			payloadJSON, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			// Créer la requête
			req := httptest.NewRequest("POST", "/api/v1/alerts/webhook", bytes.NewBuffer(payloadJSON))
			req.Header.Set("Content-Type", "application/json")

			// Exécuter la requête
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Vérifier le body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedBody["status"], body["status"])
			assert.Equal(t, tt.expectedBody["message"], body["message"])
		})
	}
}

func TestAlertsWebhookHandler_InvalidJSON(t *testing.T) {
	logger := zerolog.Nop()

	app := fiber.New()
	app.Post("/api/v1/alerts/webhook", handlers.AlertsWebhookHandler(nil, &logger))

	// Requête avec JSON invalide
	req := httptest.NewRequest("POST", "/api/v1/alerts/webhook", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "Invalid JSON payload", body["error"])
}

func TestAlertsWebhookHandler_WithOdooExporter(t *testing.T) {
	logger := zerolog.Nop()

	// Créer un serveur mock pour Odoo
	odooServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":true,"id":1}`))
	}))
	defer odooServer.Close()

	// Créer l'exporteur Odoo
	odooCfg := audit.OdooConfig{
		OdooURL:      odooServer.URL,
		OdooDatabase: "test_db",
		OdooUser:     "user",
		OdooPassword: "pass",
		Logger:       logger,
	}
	odooExporter := audit.NewOdooExporter(odooCfg)

	app := fiber.New()
	app.Post("/api/v1/alerts/webhook", handlers.AlertsWebhookHandler(odooExporter, &logger))

	// Payload avec alerte "firing" (doit être exportée)
	payload := handlers.AlertmanagerWebhookPayload{
		Version:  "4",
		Status:   "firing",
		Receiver: "default",
		Alerts: []handlers.Alert{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "HighDocumentErrorRate",
					"severity":   "warning",
				},
				Annotations: map[string]string{
					"summary":     "Taux d'erreur élevé",
					"description": "Plus de 10% d'erreurs",
				},
			},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/alerts/webhook", bytes.NewBuffer(payloadJSON))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

func TestAlertsWebhookHandler_OdooExportFailure(t *testing.T) {
	logger := zerolog.Nop()

	// Créer un serveur mock qui retourne une erreur
	odooServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Internal error"}`))
	}))
	defer odooServer.Close()

	odooCfg := audit.OdooConfig{
		OdooURL:      odooServer.URL,
		OdooDatabase: "test_db",
		OdooUser:     "user",
		OdooPassword: "pass",
		Logger:       logger,
	}
	odooExporter := audit.NewOdooExporter(odooCfg)

	app := fiber.New()
	app.Post("/api/v1/alerts/webhook", handlers.AlertsWebhookHandler(odooExporter, &logger))

	payload := handlers.AlertmanagerWebhookPayload{
		Version:  "4",
		Status:   "firing",
		Receiver: "default",
		Alerts: []handlers.Alert{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "TestAlert",
					"severity":   "warning",
				},
				Annotations: map[string]string{
					"summary": "Test",
				},
			},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/api/v1/alerts/webhook", bytes.NewBuffer(payloadJSON))
	req.Header.Set("Content-Type", "application/json")

	// L'export Odoo échoue, mais le handler doit quand même retourner 200 OK
	// (l'erreur est loggée mais ne bloque pas)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

