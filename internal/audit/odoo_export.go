package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// OdooLogEntry représente une entrée de log Odoo (ir.logging)
type OdooLogEntry struct {
	Name    string `json:"name"`    // "dorevia.vault"
	Type    string `json:"type"`    // "server"
	Level   string `json:"level"`   // "error" | "warning" | "info"
	Message string `json:"message"`
	Func    string `json:"func"`    // Fonction/component
	Line    int    `json:"line"`    // Ligne (optionnel)
	Path    string `json:"path"`    // "dorevia-vault"
}

// OdooExporter gère l'export des alertes vers Odoo
type OdooExporter struct {
	odooURL      string
	odooDatabase string
	odooUser     string
	odooPassword string
	httpClient   *http.Client
	log          zerolog.Logger
}

// OdooConfig contient la configuration pour l'export Odoo
type OdooConfig struct {
	OdooURL      string        // URL Odoo (ex: https://odoo.doreviateam.com)
	OdooDatabase string        // Base de données Odoo
	OdooUser     string        // Utilisateur Odoo
	OdooPassword string        // Mot de passe Odoo
	Timeout      time.Duration // Timeout HTTP (défaut: 10s)
	Logger       zerolog.Logger
}

// NewOdooExporter crée un nouvel exporteur Odoo
func NewOdooExporter(cfg OdooConfig) *OdooExporter {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}

	return &OdooExporter{
		odooURL:      cfg.OdooURL,
		odooDatabase: cfg.OdooDatabase,
		odooUser:     cfg.OdooUser,
		odooPassword: cfg.OdooPassword,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		log: cfg.Logger,
	}
}

// ExportAlert exporte une alerte Prometheus vers Odoo (ir.logging)
func (e *OdooExporter) ExportAlert(alertName, severity, summary, description string) error {
	if e.odooURL == "" {
		return fmt.Errorf("Odoo URL not configured")
	}

	// Mapper severity Prometheus → level Odoo
	level := "info"
	switch severity {
	case "critical":
		level = "error"
	case "warning":
		level = "warning"
	case "info":
		level = "info"
	}

	// Créer l'entrée de log
	logEntry := OdooLogEntry{
		Name:    "dorevia.vault",
		Type:    "server",
		Level:   level,
		Message: fmt.Sprintf("[%s] %s: %s", alertName, summary, description),
		Func:    alertName,
		Path:    "dorevia-vault",
	}

	// Préparer la requête Odoo XML-RPC (format simplifié via JSON-RPC)
	// Note: Odoo utilise XML-RPC, mais on peut utiliser JSON-RPC si disponible
	// Pour simplifier, on utilise l'endpoint JSON-RPC si disponible
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "call",
		"params": map[string]interface{}{
			"service": "object",
			"method":  "execute_kw",
			"args": []interface{}{
				e.odooDatabase,
				1, // uid (sera remplacé par l'authentification)
				e.odooPassword,
				"ir.logging",
				"create",
				[]map[string]interface{}{
					{
						"name":    logEntry.Name,
						"type":    logEntry.Type,
						"level":   logEntry.Level,
						"message": logEntry.Message,
						"func":    logEntry.Func,
						"path":    logEntry.Path,
					},
				},
			},
		},
		"id": time.Now().Unix(),
	}

	// Sérialiser en JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Envoyer la requête
	url := fmt.Sprintf("%s/jsonrpc", e.odooURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Odoo returned status %d", resp.StatusCode)
	}

	e.log.Debug().
		Str("alert", alertName).
		Str("severity", severity).
		Msg("Alert exported to Odoo")

	return nil
}

// ExportAlertSimple exporte une alerte avec format simplifié (pour webhook)
func (e *OdooExporter) ExportAlertSimple(alertName, severity, message string) error {
	return e.ExportAlert(alertName, severity, message, "")
}

