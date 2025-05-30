package monitor

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
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
	if len(channels) == 0 {
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
		}

		if err == nil {
			m.alertTracker.RecordAlert(queryName, channel)
		}
	}
}

// sendWebhookAlert sends an alert to the configured webhook
func (m *Monitor) sendWebhookAlert(queryName string, rule AlertRule) error {
	if m.config.Alerts.Webhook.Interval > 0 && !m.alertTracker.CanSendAlert(queryName, "webhook", m.config.Alerts.Webhook.Interval) {
		m.logger.Printf("%s alert for query %s skipped due to interval limit", "webhook", queryName)
		return fmt.Errorf("%s alert for query %s skipped due to interval limit", "webhook", queryName)
	}
	payload := AlertPayload{
		Type:     "database_alert",
		To:       rule.To,
		Message:  fmt.Sprintf("[%s] %s", queryName, rule.Message),
		Category: rule.Category,
		Instance: m.config.Instance,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		m.logger.Printf("Error marshaling webhook payload: %v", err)
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	resp, err := http.Post(m.config.Alerts.Webhook.URL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		m.logger.Printf("Error sending webhook alert: %v", err)
		return fmt.Errorf("failed to send webhook alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.logger.Printf("Webhook alert sent successfully for query: %s", queryName)

	} else {
		m.logger.Printf("Webhook alert failed with status code: %d for query: %s", resp.StatusCode, queryName)
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

// sendTelegramAlert sends an alert to Telegram
func (m *Monitor) sendTelegramAlert(queryName string, rule AlertRule) error {
	if m.config.Alerts.Telegram.Interval > 0 && !m.alertTracker.CanSendAlert(queryName, "telegram", m.config.Alerts.Telegram.Interval) {
		m.logger.Printf("%s alert for query %s skipped due to interval limit", "telegram", queryName)
		return fmt.Errorf("%s alert for query %s skipped due to interval limit", "telegram", queryName)
	}
	// Use HTML parse mode which is more reliable than Markdown
	message := fmt.Sprintf("🚨 <b>Database Alert</b>\n\n"+
		"<b>Instance:</b> %s\n"+
		"<b>Query:</b> %s\n"+
		"<b>Category:</b> %s\n"+
		"<b>Message:</b> %s\n"+
		"<b>Time:</b> %s",
		escapeHTML(m.config.Instance),
		escapeHTML(queryName),
		escapeHTML(rule.Category),
		escapeHTML(rule.Message),
		time.Now().Format("2006-01-02 15:04:05"))

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

// sendDiscordAlert sends an alert to Discord
func (m *Monitor) sendDiscordAlert(queryName string, rule AlertRule) error {
	if m.config.Alerts.Discord.Interval > 0 && !m.alertTracker.CanSendAlert(queryName, "discord", m.config.Alerts.Discord.Interval) {
		m.logger.Printf("%s alert for query %s skipped due to interval limit", "discord", queryName)
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
		Title:       "🚨 Database Alert",
		Description: fmt.Sprintf("**Instance:** %s\n**Query:** %s\n**Message:** %s", m.config.Instance, queryName, rule.Message),
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	discordMsg := DiscordMessage{
		Embeds: []DiscordEmbed{embed},
	}

	jsonData, err := json.Marshal(discordMsg)
	if err != nil {
		m.logger.Printf("Error marshaling Discord message: %v", err)
		return fmt.Errorf("failed to marshal Discord message: %w", err)
	}

	resp, err := http.Post(m.config.Alerts.Discord.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		m.logger.Printf("Error sending Discord alert: %v", err)
		return fmt.Errorf("failed to send Discord alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.logger.Printf("Discord alert sent successfully for query: %s", queryName)
	} else {
		m.logger.Printf("Discord alert failed with status code: %d for query: %s", resp.StatusCode, queryName)
	}
	return nil
}

// sendTeamsAlert sends an alert to Microsoft Teams
func (m *Monitor) sendTeamsAlert(queryName string, rule AlertRule) error {
	if m.config.Alerts.Teams.Interval > 0 && !m.alertTracker.CanSendAlert(queryName, "teams", m.config.Alerts.Teams.Interval) {
		m.logger.Printf("%s alert for query %s skipped due to interval limit", "teams", queryName)
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
		{Name: "Instance", Value: m.config.Instance},
		{Name: "Query", Value: queryName},
		{Name: "Category", Value: rule.Category},
		{Name: "Time", Value: time.Now().Format(time.RFC3339)},
	}

	section := TeamsMessageSection{
		ActivityTitle:    "🚨 Database Alert",
		ActivitySubtitle: rule.Message,
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
		m.logger.Printf("Error marshaling Teams message: %v", err)
		return fmt.Errorf("failed to marshal Teams message: %w", err)
	}

	resp, err := http.Post(m.config.Alerts.Teams.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		m.logger.Printf("Error sending Teams alert: %v", err)
		return fmt.Errorf("failed to send Teams alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		m.logger.Printf("Teams alert sent successfully for query: %s", queryName)
	} else {
		m.logger.Printf("Teams alert failed with status code: %d for query: %s", resp.StatusCode, queryName)
	}

	return nil
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

// sendEmailAlert sends an alert via SMTP email
func (m *Monitor) sendEmailAlert(queryName string, rule AlertRule) error {

	// Check if enough time has passed since last alert
	if !m.alertTracker.CanSendAlert(queryName, "email", m.config.Alerts.Email.Interval) {
		m.logger.Printf("Email alert for query %s skipped due to interval limit", queryName)
		return fmt.Errorf("email alert for query %s skipped due to interval limit", queryName)
	}

	// Prepare email content
	subject := fmt.Sprintf("[%s] Database Alert: %s", m.config.Instance, queryName)

	// Create HTML email body
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background-color: #f4f4f4; padding: 10px; border-left: 4px solid #d32f2f; }
        .content { padding: 20px; }
        .alert-info { background-color: #fff3cd; padding: 15px; border-radius: 5px; margin: 10px 0; }
        .critical { border-left: 4px solid #d32f2f; }
        .performance { border-left: 4px solid #ff9800; }
        .storage { border-left: 4px solid #ffeb3b; }
        .maintenance { border-left: 4px solid #2196f3; }
        .security { border-left: 4px solid #f44336; }
        table { border-collapse: collapse; width: 100%%; margin: 10px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h2>🚨 Database Alert</h2>
    </div>
    <div class="content">
        <div class="alert-info %s">
            <h3>%s</h3>
            <p><strong>Message:</strong> %s</p>
        </div>
        
        <table>
            <tr><th>Instance</th><td>%s</td></tr>
            <tr><th>Query</th><td>%s</td></tr>
            <tr><th>Category</th><td>%s</td></tr>
            <tr><th>Timestamp</th><td>%s</td></tr>
            <tr><th>Recipient</th><td>%s</td></tr>
        </table>
        
        <p><em>This alert was automatically generated by PostgreSQL Database Monitor.</em></p>
    </div>
</body>
</html>`,
		rule.Category,
		rule.Message,
		rule.Message,
		m.config.Instance,
		queryName,
		rule.Category,
		time.Now().Format("2006-01-02 15:04:05 MST"),
		rule.To,
	)

	// Create plain text version
	textBody := fmt.Sprintf(`Database Alert: %s

Instance: %s
Query: %s  
Category: %s
Message: %s
Timestamp: %s
Recipient: %s

This alert was automatically generated by PostgreSQL Database Monitor.`,
		queryName,
		m.config.Instance,
		queryName,
		rule.Category,
		rule.Message,
		time.Now().Format("2006-01-02 15:04:05 MST"),
		rule.To,
	)

	// Send email
	err := m.sendEmail(rule.To, subject, textBody, htmlBody)
	if err != nil {
		m.logger.Printf("Email alert failed for query %s: %v", queryName, err)
		return fmt.Errorf("failed to send email alert for query %s: %w", queryName, err)
	}

	m.logger.Printf("Email alert sent successfully for query: %s", queryName)
	m.alertTracker.RecordAlert(queryName, "email")

	return nil
}

// sendEmail sends an email using SMTP
func (m *Monitor) sendEmail(to, subject, textBody, htmlBody string) error {
	config := m.config.Alerts.Email

	// Create authentication
	auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)

	// Prepare headers
	from := config.FromEmail
	if config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", config.FromName, config.FromEmail)
	}

	// Create multipart message
	boundary := "boundary-postgres-stat-alert-" + fmt.Sprintf("%d", time.Now().Unix())

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%s\r\n\r\n",
		from, to, subject, boundary)

	textPart := fmt.Sprintf("--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n\r\n", boundary, textBody)
	htmlPart := fmt.Sprintf("--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n\r\n", boundary, htmlBody)
	ending := fmt.Sprintf("--%s--\r\n", boundary)

	message := headers + textPart + htmlPart + ending

	// Connect and send
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)

	if config.TLS {
		// Use TLS connection
		tlsConfig := &tls.Config{
			ServerName: config.SMTPHost,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect with TLS: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, config.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Quit()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}

		if err = client.Mail(config.FromEmail); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		if err = client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient: %w", err)
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %w", err)
		}

		_, err = w.Write([]byte(message))
		if err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}

		err = w.Close()
		if err != nil {
			return fmt.Errorf("failed to close data writer: %w", err)
		}

	} else {
		// Use plain SMTP
		err := smtp.SendMail(addr, auth, config.FromEmail, []string{to}, []byte(message))
		if err != nil {
			return fmt.Errorf("failed to send email: %w", err)
		}
	}

	return nil
}
