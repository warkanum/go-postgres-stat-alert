package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DiscordConfig holds Discord webhook configuration
type DiscordConfig struct {
	Enabled    bool          `yaml:"enabled"`
	WebhookURL string        `yaml:"webhook_url"`
	Interval   time.Duration `yaml:"interval"`
}

// DiscordMessage represents a Discord webhook message
type DiscordMessage struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

// DiscordEmbed represents a Discord embed
type DiscordEmbed struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Color       int    `json:"color,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

// sendDiscordAlert sends an alert to Discord
func (m *MonitorInstance) sendDiscordAlert(queryName string, rule AlertRule) error {
	if m.monitor.config.Alerts.Discord.Interval > 0 && !m.alertTracker.CanSendAlert(queryName, "discord", m.monitor.config.Alerts.Discord.Interval) {
		m.monitor.logger.Printf("%s alert for query %s skipped due to interval limit", "discord", queryName)
		return fmt.Errorf("%s alert for query %s skipped due to interval limit", "discord", queryName)
	}
	// Choose color based on category
	color := 0xff0000 // Red default
	switch strings.ToLower(rule.Category) {
	case "performance":
		color = 0xffa500 // Orange
	case "storage":
		color = 0xffff00 // Yellow
	case "security":
		color = 0xff0000 // Red
	case "maintenance":
		color = 0x0080ff // Blue
	}

	embed := DiscordEmbed{
		Title:       "🚨 Database Alert 🚨",
		Description: fmt.Sprintf("**Instance:** %s\n**Query:** %s\n**Message:** %s \n**Value** %v\n\n %v", m.dbConfig.Instance, queryName, rule.Message, rule.Value, rule.ResolutionNote),
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	discordMsg := DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	}

	jsonData, err := json.Marshal(discordMsg)
	if err != nil {
		m.monitor.logger.Printf("Error marshaling Discord message: %v", err)
		return fmt.Errorf("failed to marshal Discord message: %w", err)
	}

	resp, err := http.Post(m.monitor.config.Alerts.Discord.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		m.monitor.logger.Printf("Error sending Discord alert: %v", err)
		return fmt.Errorf("failed to send Discord alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.monitor.logger.Printf("Discord alert sent successfully for query: %s", queryName)
	} else {
		m.monitor.logger.Printf("Discord alert failed with status code: %d for query: %s", resp.StatusCode, queryName)
	}
	return nil
}
