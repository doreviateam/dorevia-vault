package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/doreviateam/dorevia-vault/internal/handlers"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAlertsWebhookIntegration teste l'intégration complète webhook → Odoo
func TestAlertsWebhookIntegration(t *testing.T) {
	logger := zerolog.Nop()

	// Créer un serveur mock Odoo
	odooAlertsReceived := make([]map[string]interface{}, 0)
	odooServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		// Extraire les données de l'alerte
		params := payload["params"].(map[string]interface{})
		args := params["args"].([]interface{})
		logEntries := args[5].([]interface{})
		logEntry := logEntries[0].(map[string]interface{})

		odooAlertsReceived = append(odooAlertsReceived, logEntry)

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
		Timeout:      5 * time.Second,
		Logger:       logger,
	}
	odooExporter := audit.NewOdooExporter(odooCfg)

	// Créer l'application Fiber avec le webhook handler
	app := fiber.New()
	app.Post("/api/v1/alerts/webhook", handlers.AlertsWebhookHandler(odooExporter, &logger))

	// Test 1 : Envoyer une alerte "firing" (doit être exportée vers Odoo)
	t.Run("firing alert exported to Odoo", func(t *testing.T) {
		odooAlertsReceived = make([]map[string]interface{}, 0) // Reset

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

		// Vérifier que l'alerte a été exportée vers Odoo
		time.Sleep(100 * time.Millisecond) // Attendre traitement asynchrone
		assert.Len(t, odooAlertsReceived, 1)
		assert.Equal(t, "warning", odooAlertsReceived[0]["level"])
		assert.Contains(t, odooAlertsReceived[0]["message"].(string), "HighDocumentErrorRate")
	})

	// Test 2 : Envoyer une alerte "resolved" (ne doit PAS être exportée)
	t.Run("resolved alert not exported", func(t *testing.T) {
		odooAlertsReceived = make([]map[string]interface{}, 0) // Reset

		payload := handlers.AlertmanagerWebhookPayload{
			Version:  "4",
			Status:   "resolved",
			Receiver: "default",
			Alerts: []handlers.Alert{
				{
					Status: "resolved",
					Labels: map[string]string{
						"alertname": "HighDocumentErrorRate",
						"severity":   "warning",
					},
					Annotations: map[string]string{
						"summary": "Taux d'erreur élevé (résolu)",
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

		// Vérifier que l'alerte n'a PAS été exportée vers Odoo
		time.Sleep(100 * time.Millisecond)
		assert.Len(t, odooAlertsReceived, 0)
	})

	// Test 3 : Envoyer plusieurs alertes (toutes "firing")
	t.Run("multiple firing alerts exported", func(t *testing.T) {
		odooAlertsReceived = make([]map[string]interface{}, 0) // Reset

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
						"summary": "Taux d'erreur élevé",
					},
				},
				{
					Status: "firing",
					Labels: map[string]string{
						"alertname": "SlowLedgerAppend",
						"severity":   "warning",
					},
					Annotations: map[string]string{
						"summary": "Ledger append lent",
					},
				},
				{
					Status: "firing",
					Labels: map[string]string{
						"alertname": "StorageNearlyFull",
						"severity":   "critical",
					},
					Annotations: map[string]string{
						"summary": "Stockage presque plein",
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

		// Vérifier que toutes les alertes ont été exportées
		time.Sleep(200 * time.Millisecond)
		assert.Len(t, odooAlertsReceived, 3)

		// Vérifier les niveaux (mapping sévérité)
		assert.Equal(t, "warning", odooAlertsReceived[0]["level"])
		assert.Equal(t, "warning", odooAlertsReceived[1]["level"])
		assert.Equal(t, "error", odooAlertsReceived[2]["level"]) // critical → error
	})
}

// TestAlertsWebhookOdooFailure teste la résilience en cas d'échec Odoo
func TestAlertsWebhookOdooFailure(t *testing.T) {
	logger := zerolog.Nop()

	// Créer un serveur mock Odoo qui retourne une erreur
	odooServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"Internal server error"}`))
	}))
	defer odooServer.Close()

	odooCfg := audit.OdooConfig{
		OdooURL:      odooServer.URL,
		OdooDatabase: "test_db",
		OdooUser:     "user",
		OdooPassword: "pass",
		Timeout:      5 * time.Second,
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

	// Le handler doit retourner 200 OK même si l'export Odoo échoue
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

// TestAlertsWebhookWithoutOdoo teste le comportement sans exporteur Odoo
func TestAlertsWebhookWithoutOdoo(t *testing.T) {
	logger := zerolog.Nop()

	app := fiber.New()
	app.Post("/api/v1/alerts/webhook", handlers.AlertsWebhookHandler(nil, &logger))

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

	// Le handler doit fonctionner même sans exporteur Odoo
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

// TestAlertsWebhookRealPayload teste avec un payload réel d'Alertmanager
func TestAlertsWebhookRealPayload(t *testing.T) {
	logger := zerolog.Nop()

	odooAlertsReceived := make([]map[string]interface{}, 0)
	odooServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		json.NewDecoder(r.Body).Decode(&payload)
		params := payload["params"].(map[string]interface{})
		args := params["args"].([]interface{})
		logEntries := args[5].([]interface{})
		logEntry := logEntries[0].(map[string]interface{})
		odooAlertsReceived = append(odooAlertsReceived, logEntry)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":true,"id":1}`))
	}))
	defer odooServer.Close()

	odooCfg := audit.OdooConfig{
		OdooURL:      odooServer.URL,
		OdooDatabase: "test_db",
		OdooUser:     "user",
		OdooPassword: "pass",
		Timeout:      5 * time.Second,
		Logger:       logger,
	}
	odooExporter := audit.NewOdooExporter(odooCfg)

	app := fiber.New()
	app.Post("/api/v1/alerts/webhook", handlers.AlertsWebhookHandler(odooExporter, &logger))

	// Payload réel d'Alertmanager (format complet)
	realPayload := `{
		"version": "4",
		"groupKey": "{}:{alertname=\"HighDocumentErrorRate\"}",
		"status": "firing",
		"receiver": "default",
		"groupLabels": {
			"alertname": "HighDocumentErrorRate"
		},
		"commonLabels": {
			"alertname": "HighDocumentErrorRate",
			"severity": "warning",
			"component": "vault",
			"service": "dorevia-vault"
		},
		"commonAnnotations": {
			"summary": "Taux d'erreur élevé lors du stockage de documents",
			"description": "15% des documents échouent sur les 5 dernières minutes. Vérifier les logs et la santé du système."
		},
		"externalURL": "http://localhost:9093",
		"alerts": [
			{
				"status": "firing",
				"labels": {
					"alertname": "HighDocumentErrorRate",
					"severity": "warning",
					"component": "vault",
					"service": "dorevia-vault"
				},
				"annotations": {
					"summary": "Taux d'erreur élevé lors du stockage de documents",
					"description": "15% des documents échouent sur les 5 dernières minutes. Vérifier les logs et la santé du système."
				},
				"startsAt": "2025-01-20T10:00:00Z",
				"endsAt": "0001-01-01T00:00:00Z",
				"generatorURL": "http://localhost:9090/graph?g0.expr=..."
			}
		]
	}`

	req := httptest.NewRequest("POST", "/api/v1/alerts/webhook", bytes.NewBufferString(realPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	time.Sleep(100 * time.Millisecond)
	assert.Len(t, odooAlertsReceived, 1)
	assert.Equal(t, "warning", odooAlertsReceived[0]["level"])
	assert.Contains(t, odooAlertsReceived[0]["message"].(string), "HighDocumentErrorRate")
}

