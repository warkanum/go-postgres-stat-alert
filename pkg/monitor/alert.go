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
func (m *MonitorInstance) checkAlertRules(queryConfig QueryConfig, columns []string, values []interface{}) {
	for _, rule := range queryConfig.AlertRules {
		// For simplicity, assume the first column contains the value to check
		// In a real implementation, you might want to specify which column to check
		if len(values) == 0 {
			continue
		}

		if rule.Instances != nil && len(rule.Instances) > 0 {
			allow := false
			for _, instance := range rule.Instances {
				if instance == m.dbConfig.Instance {
					allow = true
					break
				}
			}

			if !allow {
				m.monitor.logger.Printf("Skipping alert for query %s on %s due to instance restriction", queryConfig.Name, m.dbConfig.Instance)
				continue
			}
		}

		value := values[0]
		if m.evaluateCondition(value, rule.Condition, rule.Value) {
			m.monitor.logger.Printf("Alert triggered for query %s: %s", queryConfig.Name, rule.Message)
			if !m.isWithinAlertHours(rule) {
				m.monitor.logger.Printf("Alert for query %s suppressed due to time restrictions", queryConfig.Name)
				continue
			}
			m.sendAlerts(queryConfig.Name, rule)

			// Execute action if specified
			if rule.ExecuteAction != "" {
				m.executeAction(queryConfig.Name, rule)
			}

		}
		m.monitor.logger.Printf("Query Result %s: %v", queryConfig.Name, value)
	}
}

// sendAlerts sends alerts to all configured channels
func (m *MonitorInstance) sendAlerts(queryName string, rule AlertRule) {
	// Determine which channels to use
	channels := rule.Channels
	if len(rule.Channels) == 0 {
		// If no specific channels specified, use all enabled channels
		channels = []string{}
		if m.monitor.config.Alerts.Webhook.Enabled {
			channels = append(channels, "webhook")
		}
		if m.monitor.config.Alerts.Telegram.Enabled {
			channels = append(channels, "telegram")
		}
		if m.monitor.config.Alerts.Discord.Enabled {
			channels = append(channels, "discord")
		}
		if m.monitor.config.Alerts.Teams.Enabled {
			channels = append(channels, "teams")
		}
		if m.monitor.config.Alerts.Email.Enabled {
			channels = append(channels, "email")
		}
		if m.monitor.config.Alerts.WhatsApp.Enabled {
			channels = append(channels, "whatsapp")
		}
	}

	// Send to each specified channel
	for _, channel := range channels {
		var err error

		switch strings.ToLower(channel) {
		case "webhook":
			if m.monitor.config.Alerts.Webhook.Enabled {
				err = m.sendWebhookAlert(queryName, rule)
			}
		case "telegram":
			if m.monitor.config.Alerts.Telegram.Enabled {
				err = m.sendTelegramAlert(queryName, rule)
			}
		case "discord":
			if m.monitor.config.Alerts.Discord.Enabled {
				err = m.sendDiscordAlert(queryName, rule)
			}
		case "teams":
			if m.monitor.config.Alerts.Teams.Enabled {
				err = m.sendTeamsAlert(queryName, rule)
			}
		case "email":
			if m.monitor.config.Alerts.Email.Enabled {
				err = m.sendEmailAlert(queryName, rule)
			}
		case "whatsapp":
			if m.monitor.config.Alerts.WhatsApp.Enabled {
				err = m.sendWhatsAppAlert(queryName, rule)
			}
		}

		if err == nil {
			m.alertTracker.RecordAlert(queryName, channel)
		}
	}
}

// executeAction runs the specified command/script when an alert is triggered
func (m *MonitorInstance) executeAction(queryName string, rule AlertRule) {
	m.monitor.logger.Printf("Executing action for query %s: %s", queryName, rule.ExecuteAction)

	// Parse command and arguments
	parts := strings.Fields(rule.ExecuteAction)
	if len(parts) == 0 {
		m.monitor.logger.Printf("Error: Empty execute_action for query %s", queryName)
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
		fmt.Sprintf("MONITOR_INSTANCE=%s", m.dbConfig.Instance),
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
		m.monitor.logger.Printf("Action execution failed for query %s: %v", queryName, err)
		if stderr.Len() > 0 {
			m.monitor.logger.Printf("Action stderr: %s", stderr.String())
		}
		return
	}

	m.monitor.logger.Printf("Action executed successfully for query %s (duration: %v)", queryName, duration)

	// Log stdout if present (useful for debugging)
	if stdout.Len() > 0 {
		m.monitor.logger.Printf("Action output: %s", strings.TrimSpace(stdout.String()))
	}
}

// isWithinAlertHours checks if current time is within allowed alert hours
func (m *MonitorInstance) isWithinAlertHours(rule AlertRule) bool {
	// If no alert hours specified, always allow alerts
	if rule.AlertHours == nil {
		return true
	}

	alertHours := rule.AlertHours

	// Parse timezone
	var loc *time.Location
	var err error
	if alertHours.Timezone != "" {
		loc, err = time.LoadLocation(alertHours.Timezone)
		if err != nil {
			m.monitor.logger.Printf("Invalid timezone '%s', using UTC: %v", alertHours.Timezone, err)
			loc = time.UTC
		}
	} else {
		loc = time.Local // Use system local time if not specified
	}

	now := time.Now().In(loc)

	// Check day of week if specified
	if len(alertHours.Days) > 0 {
		dayOfWeek := strings.ToLower(now.Weekday().String()[:3]) // "mon", "tue", etc.
		dayAllowed := false
		for _, allowedDay := range alertHours.Days {
			if strings.ToLower(allowedDay) == dayOfWeek {
				dayAllowed = true
				break
			}
		}
		if !dayAllowed {
			return false
		}
	}

	// Parse start time
	startTime, err := time.ParseInLocation("15:04", alertHours.Start, loc)
	if err != nil {
		m.monitor.logger.Printf("Invalid start time format '%s', expected HH:MM: %v", alertHours.Start, err)
		return true // Allow alert if time format is invalid
	}

	// Parse end time
	endTime, err := time.ParseInLocation("15:04", alertHours.End, loc)
	if err != nil {
		m.monitor.logger.Printf("Invalid end time format '%s', expected HH:MM: %v", alertHours.End, err)
		return true // Allow alert if time format is invalid
	}

	// Set dates to today for comparison
	today := now.Format("2006-01-02")
	startTime, _ = time.ParseInLocation("2006-01-02 15:04", today+" "+alertHours.Start, loc)
	endTime, _ = time.ParseInLocation("2006-01-02 15:04", today+" "+alertHours.End, loc)

	// Handle overnight ranges (e.g., 22:00 to 06:00)
	if endTime.Before(startTime) {
		// If end time is before start time, it spans midnight
		// Check if current time is after start OR before end
		return now.After(startTime) || now.Before(endTime)
	}

	// Normal range (e.g., 08:00 to 21:00)
	return now.After(startTime) && now.Before(endTime)
}
