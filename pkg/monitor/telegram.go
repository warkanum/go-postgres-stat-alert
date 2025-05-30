package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	Enabled  bool          `yaml:"enabled"`
	BotToken string        `yaml:"bot_token"`
	ChatID   string        `yaml:"chat_id"`
	Interval time.Duration `yaml:"interval"`
}

// TelegramMessage represents a Telegram message
type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// sendTelegramAlert sends an alert to Telegram
func (m *Monitor) sendTelegramAlert(queryName string, rule AlertRule) error {
	if m.config.Alerts.Telegram.Interval > 0 && !m.alertTracker.CanSendAlert(queryName, "telegram", m.config.Alerts.Telegram.Interval) {
		m.logger.Printf("%s alert for query %s skipped due to interval limit", "telegram", queryName)
		return fmt.Errorf("%s alert for query %s skipped due to interval limit", "telegram", queryName)
	}
	// Use HTML parse mode which is more reliable than Markdown
	message := fmt.Sprintf("🚨 <b>Database Alert</b> 🚨\n\n"+
		"<b>Instance:</b> %s\n"+
		"<b>Query:</b> %s\n"+
		"<b>Category:</b> %s\n"+
		"<b>Message:</b> %s\n"+
		"<b>Time:</b> %s\n"+
		"<b>Value:</b> %s",
		escapeHTML(m.config.Instance),
		escapeHTML(queryName),
		escapeHTML(rule.Category),
		escapeHTML(rule.Message),
		time.Now().Format("2006-01-02 15:04:05"),
		escapeHTML(fmt.Sprintf("%v", rule.Value)))

	telegramMsg := TelegramMessage{
		ChatID:    m.config.Alerts.Telegram.ChatID,
		Text:      message,
		ParseMode: "HTML",
	}

	jsonData, err := json.Marshal(telegramMsg)
	if err != nil {
		m.logger.Printf("Error marshaling Telegram message: %v", err)
		return fmt.Errorf("failed to marshal Telegram message: %w", err)
	}

	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", m.config.Alerts.Telegram.BotToken)

	resp, err := http.Post(telegramURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		m.logger.Printf("Error sending Telegram alert: %v", err)
		return fmt.Errorf("failed to send Telegram alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.logger.Printf("Telegram alert sent successfully for query: %s", queryName)
	} else {
		defer func() {
			if resp.Body != nil {
				resp.Body.Close()
			}
		}()
		respBody, _ := io.ReadAll(resp.Body)

		m.logger.Printf("Telegram alert failed with status code: %d %s (%s) for query: %s", resp.StatusCode, resp.Status, string(respBody), queryName)
		return fmt.Errorf("Telegram alert failed with status code: %d %s (%s) for query: %s", resp.StatusCode, resp.Status, string(respBody), queryName)
	}

	return nil
}
