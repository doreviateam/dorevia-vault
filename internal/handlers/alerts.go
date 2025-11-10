package handlers

import (
	"fmt"

	"github.com/doreviateam/dorevia-vault/internal/audit"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

// AlertmanagerWebhookPayload représente le payload envoyé par Alertmanager
type AlertmanagerWebhookPayload struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"` // "firing" | "resolved"
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

// Alert représente une alerte individuelle
type Alert struct {
	Status       string            `json:"status"` // "firing" | "resolved"
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

// AlertsWebhookHandler gère l'endpoint POST /api/v1/alerts/webhook
// Reçoit les alertes depuis Alertmanager et peut les exporter vers Odoo
func AlertsWebhookHandler(odooExporter *audit.OdooExporter, log *zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var payload AlertmanagerWebhookPayload

		if err := c.BodyParser(&payload); err != nil {
			log.Error().Err(err).Msg("Failed to parse alertmanager webhook payload")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid JSON payload",
			})
		}

		// Logger les alertes reçues
		log.Info().
			Str("status", payload.Status).
			Str("receiver", payload.Receiver).
			Int("alert_count", len(payload.Alerts)).
			Msg("Received alerts from Alertmanager")

		// Traiter chaque alerte
		for _, alert := range payload.Alerts {
			alertName := alert.Labels["alertname"]
			severity := alert.Labels["severity"]
			summary := alert.Annotations["summary"]
			description := alert.Annotations["description"]

			// Logger l'alerte
			log.Warn().
				Str("alert", alertName).
				Str("severity", severity).
				Str("status", alert.Status).
				Str("summary", summary).
				Msg("Alert received")

			// Exporter vers Odoo si configuré et si alerte "firing"
			if odooExporter != nil && alert.Status == "firing" {
				if err := odooExporter.ExportAlert(alertName, severity, summary, description); err != nil {
					log.Error().
						Err(err).
						Str("alert", alertName).
						Msg("Failed to export alert to Odoo")
					// Ne pas bloquer, continuer avec les autres alertes
				} else {
					log.Info().
						Str("alert", alertName).
						Msg("Alert exported to Odoo")
				}
			}
		}

		// Retourner succès
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": fmt.Sprintf("Processed %d alerts", len(payload.Alerts)),
		})
	}
}

