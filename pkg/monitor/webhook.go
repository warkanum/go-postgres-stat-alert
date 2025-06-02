package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type WebhookConfig struct {
	Enabled  bool          `yaml:"enabled"`
	URL      string        `yaml:"url"`
	Interval time.Duration `yaml:"interval"`
}

// sendWebhookAlert sends an alert to the configured webhook
func (m *MonitorInstance) sendWebhookAlert(queryName string, rule AlertRule) error {
	if m.monitor.config.Alerts.Webhook.Interval > 0 && !m.alertTracker.CanSendAlert(queryName, "webhook", m.monitor.config.Alerts.Webhook.Interval) {
		m.monitor.logger.Printf("%s alert for query %s skipped due to interval limit", "webhook", queryName)
		return fmt.Errorf("%s alert for query %s skipped due to interval limit", "webhook", queryName)
	}
	payload := AlertPayload{
		Type:     "database_alert",
		To:       rule.To,
		Message:  fmt.Sprintf("[%s] %s", queryName, rule.Message),
		Category: rule.Category,
		Value:    rule.Value,
		Instance: m.dbConfig.Instance,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		m.monitor.logger.Printf("Error marshaling webhook payload: %v", err)
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	resp, err := http.Post(m.monitor.config.Alerts.Webhook.URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		m.monitor.logger.Printf("Error sending webhook alert: %v", err)
		return fmt.Errorf("failed to send webhook alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.monitor.logger.Printf("Webhook alert sent successfully for query: %s", queryName)

	} else {
		m.monitor.logger.Printf("Webhook alert failed with status code: %d for query: %s", resp.StatusCode, queryName)
		return fmt.Errorf("webhook alert failed with status code: %d for query: %s", resp.StatusCode, queryName)
	}
	return nil
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
