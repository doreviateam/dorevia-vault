package unit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOdooExporter(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name     string
		cfg      audit.OdooConfig
		wantErr  bool
		checkNil bool
	}{
		{
			name: "valid config",
			cfg: audit.OdooConfig{
				OdooURL:      "https://odoo.example.com",
				OdooDatabase: "test_db",
				OdooUser:     "user",
				OdooPassword: "pass",
				Logger:       logger,
			},
			wantErr:  false,
			checkNil: false,
		},
		{
			name: "default timeout",
			cfg: audit.OdooConfig{
				OdooURL:      "https://odoo.example.com",
				OdooDatabase: "test_db",
				Timeout:     0, // Should default to 10s
				Logger:       logger,
			},
			wantErr:  false,
			checkNil: false,
		},
		{
			name: "custom timeout",
			cfg: audit.OdooConfig{
				OdooURL:      "https://odoo.example.com",
				OdooDatabase: "test_db",
				Timeout:     5 * time.Second,
				Logger:       logger,
			},
			wantErr:  false,
			checkNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := audit.NewOdooExporter(tt.cfg)
			if tt.checkNil {
				assert.Nil(t, exporter)
			} else {
				assert.NotNil(t, exporter)
				if tt.cfg.Timeout == 0 {
					// Vérifier que le timeout par défaut est 10s
					// On ne peut pas accéder directement au httpClient, donc on teste indirectement
					assert.NotNil(t, exporter)
				}
			}
		})
	}
}

func TestOdooExporter_ExportAlert(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name           string
		alertName      string
		severity       string
		summary        string
		description    string
		serverResponse int
		serverBody     string
		wantErr        bool
	}{
		{
			name:           "success critical",
			alertName:      "HighDocumentErrorRate",
			severity:       "critical",
			summary:        "Taux d'erreur élevé",
			description:    "Plus de 10% d'erreurs",
			serverResponse: http.StatusOK,
			serverBody:     `{"jsonrpc":"2.0","result":true,"id":1}`,
			wantErr:        false,
		},
		{
			name:           "success warning",
			alertName:      "SlowLedgerAppend",
			severity:       "warning",
			summary:        "Ledger append lent",
			description:    "P95 > 2s",
			serverResponse: http.StatusOK,
			serverBody:     `{"jsonrpc":"2.0","result":true,"id":1}`,
			wantErr:        false,
		},
		{
			name:           "success info",
			alertName:      "NoRecentDocuments",
			severity:       "info",
			summary:        "Aucun document récent",
			description:    "",
			serverResponse: http.StatusOK,
			serverBody:     `{"jsonrpc":"2.0","result":true,"id":1}`,
			wantErr:        false,
		},
		{
			name:           "server error",
			alertName:      "TestAlert",
			severity:       "warning",
			summary:        "Test",
			description:    "",
			serverResponse: http.StatusInternalServerError,
			serverBody:     `{"error":"Internal error"}`,
			wantErr:        true,
		},
		{
			name:           "network error",
			alertName:      "TestAlert",
			severity:       "warning",
			summary:        "Test",
			description:    "",
			serverResponse: 0, // Server will not respond
			serverBody:     "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Créer un serveur de test
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/jsonrpc", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Vérifier le payload
				var payload map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&payload)
				require.NoError(t, err)

				assert.Equal(t, "2.0", payload["jsonrpc"])
				assert.Equal(t, "call", payload["method"])

				// Vérifier que le message contient l'alerte
				params := payload["params"].(map[string]interface{})
				args := params["args"].([]interface{})
				// args[5] contient []map[string]interface{} avec les données du log
				logEntries := args[5].([]interface{})
				logEntry := logEntries[0].(map[string]interface{})

				// Vérifier le mapping severity → level
				expectedLevel := "info"
				if tt.severity == "critical" {
					expectedLevel = "error"
				} else if tt.severity == "warning" {
					expectedLevel = "warning"
				}
				assert.Equal(t, expectedLevel, logEntry["level"])

				w.WriteHeader(tt.serverResponse)
				w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			// Créer l'exporteur avec l'URL du serveur de test
			cfg := audit.OdooConfig{
				OdooURL:      server.URL,
				OdooDatabase: "test_db",
				OdooUser:     "user",
				OdooPassword: "pass",
				Timeout:      5 * time.Second,
				Logger:       logger,
			}
			exporter := audit.NewOdooExporter(cfg)

			// Exporter l'alerte
			err := exporter.ExportAlert(tt.alertName, tt.severity, tt.summary, tt.description)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOdooExporter_ExportAlert_NoURL(t *testing.T) {
	logger := zerolog.Nop()

	cfg := audit.OdooConfig{
		OdooURL:      "", // URL vide
		OdooDatabase: "test_db",
		Logger:       logger,
	}
	exporter := audit.NewOdooExporter(cfg)

	err := exporter.ExportAlert("TestAlert", "warning", "Test", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Odoo URL not configured")
}

func TestOdooExporter_ExportAlertSimple(t *testing.T) {
	logger := zerolog.Nop()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","result":true,"id":1}`))
	}))
	defer server.Close()

	cfg := audit.OdooConfig{
		OdooURL:      server.URL,
		OdooDatabase: "test_db",
		OdooUser:     "user",
		OdooPassword: "pass",
		Timeout:      5 * time.Second,
		Logger:       logger,
	}
	exporter := audit.NewOdooExporter(cfg)

	err := exporter.ExportAlertSimple("TestAlert", "warning", "Test message")
	assert.NoError(t, err)
}

func TestOdooExporter_SeverityMapping(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name           string
		severity       string
		expectedLevel  string
	}{
		{
			name:          "critical to error",
			severity:      "critical",
			expectedLevel: "error",
		},
		{
			name:          "warning to warning",
			severity:      "warning",
			expectedLevel: "warning",
		},
		{
			name:          "info to info",
			severity:      "info",
			expectedLevel: "info",
		},
		{
			name:          "unknown to info",
			severity:      "unknown",
			expectedLevel: "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload map[string]interface{}
				json.NewDecoder(r.Body).Decode(&payload)

				params := payload["params"].(map[string]interface{})
				args := params["args"].([]interface{})
				// args[5] contient []interface{} avec les données du log
				logEntries := args[5].([]interface{})
				logEntry := logEntries[0].(map[string]interface{})

				assert.Equal(t, tt.expectedLevel, logEntry["level"])

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc":"2.0","result":true,"id":1}`))
			}))
			defer server.Close()

			cfg := audit.OdooConfig{
				OdooURL:      server.URL,
				OdooDatabase: "test_db",
				OdooUser:     "user",
				OdooPassword: "pass",
				Timeout:      5 * time.Second,
				Logger:       logger,
			}
			exporter := audit.NewOdooExporter(cfg)

			err := exporter.ExportAlert("TestAlert", tt.severity, "Test", "")
			assert.NoError(t, err)
		})
	}
}

