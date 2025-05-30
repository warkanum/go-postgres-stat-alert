package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type WhatsAppConfig struct {
	Enabled       bool          `yaml:"enabled"`
	AccessToken   string        `yaml:"access_token"`
	PhoneNumberID string        `yaml:"phone_number_id"`
	ToNumber      string        `yaml:"to_number"`
	Interval      time.Duration `yaml:"interval"`
}

type WhatsAppMessage struct {
	MessagingProduct string               `json:"messaging_product"`
	To               string               `json:"to"`
	Type             string               `json:"type"`
	Text             *WhatsAppTextMessage `json:"text,omitempty"`
}

// WhatsAppTextMessage represents WhatsApp text message content
type WhatsAppTextMessage struct {
	Body string `json:"body"`
}

// sendWhatsAppAlert sends an alert via WhatsApp Business API
func (m *Monitor) sendWhatsAppAlert(queryName string, rule AlertRule) error {
	// Check if enough time has passed since last alert
	if !m.alertTracker.CanSendAlert(queryName, "whatsapp", m.config.Alerts.WhatsApp.Interval) {
		m.logger.Printf("WhatsApp alert for query %s skipped due to interval limit", queryName)
		return fmt.Errorf("WhatsApp alert for query %s skipped due to interval limit", queryName)
	}

	config := m.config.Alerts.WhatsApp

	// Create message content
	messageText := fmt.Sprintf("🚨 *Database Alert* 🚨\n\n"+
		"*Instance:* %s\n"+
		"*Query:* %s\n"+
		"*Category:* %s\n"+
		"*Message:* %s\n"+
		"*Time:* %s\n"+
		"*Value:* %s",
		m.config.Instance,
		queryName,
		rule.Category,
		rule.Message,
		time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf("%v", rule.Value))

	// Create WhatsApp message
	whatsappMsg := WhatsAppMessage{
		MessagingProduct: "whatsapp",
		To:               config.ToNumber,
		Type:             "text",
		Text: &WhatsAppTextMessage{
			Body: messageText,
		},
	}

	jsonData, err := json.Marshal(whatsappMsg)
	if err != nil {
		m.logger.Printf("Error marshaling WhatsApp message: %v", err)
		return fmt.Errorf("failed to marshal WhatsApp message: %w", err)
	}

	// Send via WhatsApp Business API
	url := fmt.Sprintf("https://graph.facebook.com/v22.0/%s/messages", config.PhoneNumberID)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		m.logger.Printf("Error creating WhatsApp request: %v", err)
		return fmt.Errorf("failed to create WhatsApp request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.AccessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		m.logger.Printf("Error sending WhatsApp alert: %v", err)
		return fmt.Errorf("failed to send WhatsApp alert: %w", err)
	}
	defer resp.Body.Close()

	bodyText := ""
	if resp.Body != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyText = string(bodyBytes)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.logger.Printf("WhatsApp alert sent successfully for query: %s -> %s", queryName, bodyText)
		m.alertTracker.RecordAlert(queryName, "whatsapp")
	} else {
		m.logger.Printf("WhatsApp alert failed with status code: %d for query: %s -> %s", resp.StatusCode, queryName, bodyText)
		return fmt.Errorf("WhatsApp alert failed with status code: %d for query: %s -> %s", resp.StatusCode, queryName, bodyText)
	}

	return nil
}
