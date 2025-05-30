package monitor

import (
	"database/sql"
	"log"
	"time"
)

// Config represents the YAML configuration structure
type Config struct {
	Instance string         `yaml:"instance"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
	Queries  []QueryConfig  `yaml:"queries"`
	Alerts   AlertsConfig   `yaml:"alerts"`
}

// DatabaseConfig holds database connection details
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"sslmode"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	FilePath string `yaml:"file_path"`
}

// QueryConfig represents a query to monitor
type QueryConfig struct {
	Name       string            `yaml:"name"`
	SQL        string            `yaml:"sql"`
	Interval   time.Duration     `yaml:"interval"`
	AlertRules []AlertRule       `yaml:"alert_rules"`
	Parameters map[string]string `yaml:"parameters,omitempty"`
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	Condition     string      `yaml:"condition"` // "gt", "lt", "eq", "ne", "gte", "lte"
	Value         interface{} `yaml:"value"`
	Message       string      `yaml:"message"`
	Category      string      `yaml:"category"`
	To            string      `yaml:"to"`
	Channels      []string    `yaml:"channels,omitempty"`
	ExecuteAction string      `yaml:"execute_action,omitempty"` // Optional action to execute on alert
}

// AlertsConfig holds all alert configurations
type AlertsConfig struct {
	Webhook  WebhookConfig  `yaml:"webhook"`
	Telegram TelegramConfig `yaml:"telegram"`
	Discord  DiscordConfig  `yaml:"discord"`
	Teams    TeamsConfig    `yaml:"teams"`
	Email    EmailConfig    `yaml:"email"`
}

// AlertPayload represents the alert message structure
type AlertPayload struct {
	Type     string `json:"type"`
	To       string `json:"to"`
	Message  string `json:"message"`
	Category string `json:"category"`
	Instance string `json:"instance"`
}

// Monitor represents the database monitor
type Monitor struct {
	config       *Config
	db           *sql.DB
	logger       *log.Logger
	alertTracker *AlertTracker
}

// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	Enabled  bool          `yaml:"enabled"`
	BotToken string        `yaml:"bot_token"`
	ChatID   string        `yaml:"chat_id"`
	Interval time.Duration `yaml:"interval"`
}

// DiscordConfig holds Discord webhook configuration
type DiscordConfig struct {
	Enabled    bool          `yaml:"enabled"`
	WebhookURL string        `yaml:"webhook_url"`
	Interval   time.Duration `yaml:"interval"`
}

// TeamsConfig holds Microsoft Teams webhook configuration
type TeamsConfig struct {
	Enabled    bool          `yaml:"enabled"`
	WebhookURL string        `yaml:"webhook_url"`
	Interval   time.Duration `yaml:"interval"`
}

type WebhookConfig struct {
	Enabled  bool          `yaml:"enabled"`
	URL      string        `yaml:"url"`
	Interval time.Duration `yaml:"interval"`
}

// TelegramMessage represents a Telegram message
type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// EmailConfig holds SMTP email configuration
type EmailConfig struct {
	Enabled   bool          `yaml:"enabled"`
	SMTPHost  string        `yaml:"smtp_host"`
	SMTPPort  int           `yaml:"smtp_port"`
	Username  string        `yaml:"username"`
	Password  string        `yaml:"password"`
	FromEmail string        `yaml:"from_email"`
	FromName  string        `yaml:"from_name"`
	TLS       bool          `yaml:"tls"`
	Interval  time.Duration `yaml:"interval"`
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
