package monitor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// TeamsConfig holds Microsoft Teams webhook configuration
type TeamsConfig struct {
	Enabled    bool          `yaml:"enabled"`
	WebhookURL string        `yaml:"webhook_url"`
	Interval   time.Duration `yaml:"interval"`
}

// TeamsMessage represents a Microsoft Teams message
type TeamsMessage struct {
	Type       string                `json:"@type"`
	Context    string                `json:"@context"`
	ThemeColor string                `json:"themeColor,omitempty"`
	Summary    string                `json:"summary"`
	Sections   []TeamsMessageSection `json:"sections"`
}

// TeamsMessageSection represents a section in Teams message
type TeamsMessageSection struct {
	ActivityTitle    string             `json:"activityTitle,omitempty"`
	ActivitySubtitle string             `json:"activitySubtitle,omitempty"`
	Facts            []TeamsMessageFact `json:"facts,omitempty"`
	Text             string             `json:"text,omitempty"`
}

// TeamsMessageFact represents a fact in Teams message
type TeamsMessageFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// sendTeamsAlert sends an alert to Microsoft Teams
func (m *MonitorInstance) sendTeamsAlert(queryName string, rule AlertRule) error {
	if m.monitor.config.Alerts.Teams.Interval > 0 && !m.alertTracker.CanSendAlert(queryName, "teams", m.monitor.config.Alerts.Teams.Interval) {
		m.monitor.logger.Printf("%s alert for query %s skipped due to interval limit", "teams", queryName)
		return fmt.Errorf("%s alert for query %s skipped due to interval limit", "teams", queryName)
	}
	// Choose theme color based on category
	themeColor := "FF0000" // Red default
	switch strings.ToLower(rule.Category) {
	case "performance":
		themeColor = "FFA500" // Orange
	case "storage":
		themeColor = "FFFF00" // Yellow
	case "security":
		themeColor = "FF0000" // Red
	case "maintenance":
		themeColor = "0080FF" // Blue
	}

	facts := []TeamsMessageFact{
		{Name: "Instance", Value: m.dbConfig.Instance},
		{Name: "Query", Value: queryName},
		{Name: "Category", Value: rule.Category},
		{Name: "Time", Value: time.Now().Format(time.RFC3339)},
	}

	section := TeamsMessageSection{
		ActivityTitle:    "🚨 Database Alert",
		ActivitySubtitle: rule.Message,
		Text:             fmt.Sprintf("**Instance:** %s\n**Query:** %s\n**Message:** %s\n**Value:** %v", m.dbConfig.Instance, queryName, rule.Message, rule.Value),
		Facts:            facts,
	}

	teamsMsg := TeamsMessage{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: themeColor,
		Summary:    fmt.Sprintf("Database Alert: %s", queryName),
		Sections:   []TeamsMessageSection{section},
	}

	jsonData, err := json.Marshal(teamsMsg)
	if err != nil {
		m.monitor.logger.Printf("Error marshaling Teams message: %v", err)
		return fmt.Errorf("failed to marshal Teams message: %w", err)
	}

	resp, err := http.Post(m.monitor.config.Alerts.Teams.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		m.monitor.logger.Printf("Error sending Teams alert: %v", err)
		return fmt.Errorf("failed to send Teams alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.monitor.logger.Printf("Teams alert sent successfully for query: %s", queryName)
	} else {
		m.monitor.logger.Printf("Teams alert failed with status code: %d for query: %s", resp.StatusCode, queryName)
	}

	return nil
}
