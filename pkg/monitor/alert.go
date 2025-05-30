package monitor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// AlertTracker tracks last alert times to prevent spam
type AlertTracker struct {
	LastAlert map[string]map[string]time.Time // [queryName][channel] -> lastAlertTime
	mu        sync.RWMutex
}

// NewAlertTracker creates a new alert tracker
func NewAlertTracker() *AlertTracker {
	return &AlertTracker{
		LastAlert: make(map[string]map[string]time.Time),
	}
}

// CanSendAlert checks if enough time has passed since last alert for this query/channel
func (at *AlertTracker) CanSendAlert(queryName, channel string, interval time.Duration) bool {
	at.mu.RLock()
	defer at.mu.RUnlock()

	if queryAlerts, exists := at.LastAlert[queryName]; exists {
		if lastTime, exists := queryAlerts[channel]; exists {
			return time.Since(lastTime) >= interval
		}
	}
	return true // No previous alert found, allow sending
}

// RecordAlert records when an alert was sent
func (at *AlertTracker) RecordAlert(queryName, channel string) {
	at.mu.Lock()
	defer at.mu.Unlock()

	if at.LastAlert[queryName] == nil {
		at.LastAlert[queryName] = make(map[string]time.Time)
	}
	at.LastAlert[queryName][channel] = time.Now()
}

// checkAlertRules evaluates alert rules against query results
func (m *Monitor) checkAlertRules(queryConfig QueryConfig, columns []string, values []interface{}) {
	for _, rule := range queryConfig.AlertRules {
		// For simplicity, assume the first column contains the value to check
		// In a real implementation, you might want to specify which column to check
		if len(values) == 0 {
			continue
		}

		value := values[0]
		if m.evaluateCondition(value, rule.Condition, rule.Value) {
			m.logger.Printf("Alert triggered for query %s: %s", queryConfig.Name, rule.Message)
			m.sendAlerts(queryConfig.Name, rule)

			// Execute action if specified
			if rule.ExecuteAction != "" {
				m.executeAction(queryConfig.Name, rule)
			}

		}
		m.logger.Printf("Query Result %s: %v", queryConfig.Name, value)
	}
}

// sendAlerts sends alerts to all configured channels
func (m *Monitor) sendAlerts(queryName string, rule AlertRule) {
	// Determine which channels to use
	channels := rule.Channels
	if len(rule.Channels) == 0 {
		// If no specific channels specified, use all enabled channels
		channels = []string{}
		if m.config.Alerts.Webhook.Enabled {
			channels = append(channels, "webhook")
		}
		if m.config.Alerts.Telegram.Enabled {
			channels = append(channels, "telegram")
		}
		if m.config.Alerts.Discord.Enabled {
			channels = append(channels, "discord")
		}
		if m.config.Alerts.Teams.Enabled {
			channels = append(channels, "teams")
		}
		if m.config.Alerts.Email.Enabled {
			channels = append(channels, "email")
		}
		if m.config.Alerts.WhatsApp.Enabled {
			channels = append(channels, "whatsapp")
		}
	}

	// Send to each specified channel
	for _, channel := range channels {
		var err error

		switch strings.ToLower(channel) {
		case "webhook":
			if m.config.Alerts.Webhook.Enabled {
				err = m.sendWebhookAlert(queryName, rule)
			}
		case "telegram":
			if m.config.Alerts.Telegram.Enabled {
				err = m.sendTelegramAlert(queryName, rule)
			}
		case "discord":
			if m.config.Alerts.Discord.Enabled {
				err = m.sendDiscordAlert(queryName, rule)
			}
		case "teams":
			if m.config.Alerts.Teams.Enabled {
				err = m.sendTeamsAlert(queryName, rule)
			}
		case "email":
			if m.config.Alerts.Email.Enabled {
				err = m.sendEmailAlert(queryName, rule)
			}
		case "whatsapp":
			if m.config.Alerts.WhatsApp.Enabled {
				err = m.sendWhatsAppAlert(queryName, rule)
			}
		}

		if err == nil {
			m.alertTracker.RecordAlert(queryName, channel)
		}
	}
}

// executeAction runs the specified command/script when an alert is triggered
func (m *Monitor) executeAction(queryName string, rule AlertRule) {
	m.logger.Printf("Executing action for query %s: %s", queryName, rule.ExecuteAction)

	// Parse command and arguments
	parts := strings.Fields(rule.ExecuteAction)
	if len(parts) == 0 {
		m.logger.Printf("Error: Empty execute_action for query %s", queryName)
		return
	}

	command := parts[0]
	args := parts[1:]

	// Create command with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)

	// Set environment variables for the script
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MONITOR_INSTANCE=%s", m.config.Instance),
		fmt.Sprintf("MONITOR_QUERY=%s", queryName),
		fmt.Sprintf("MONITOR_MESSAGE=%s", rule.Message),
		fmt.Sprintf("MONITOR_CATEGORY=%s", rule.Category),
		fmt.Sprintf("MONITOR_TO=%s", rule.To),
		fmt.Sprintf("MONITOR_VALUE=%s", fmt.Sprintf("%v", rule.Value)),
	)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		m.logger.Printf("Action execution failed for query %s: %v", queryName, err)
		if stderr.Len() > 0 {
			m.logger.Printf("Action stderr: %s", stderr.String())
		}
		return
	}

	m.logger.Printf("Action executed successfully for query %s (duration: %v)", queryName, duration)

	// Log stdout if present (useful for debugging)
	if stdout.Len() > 0 {
		m.logger.Printf("Action output: %s", strings.TrimSpace(stdout.String()))
	}
}
