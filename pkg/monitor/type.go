package monitor

import (
	"database/sql"
	"log"
	"time"
)

// Config represents the YAML configuration structure
type Config struct {
	Database []DatabaseConfig `yaml:"databases"`
	Logging  LoggingConfig    `yaml:"logging"`
	Queries  []QueryConfig    `yaml:"queries"`
	Alerts   AlertsConfig     `yaml:"alerts"`
}

// DatabaseConfig holds database connection details
type DatabaseConfig struct {
	Instance string `yaml:"instance"`
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
	Instances     []string    `yaml:"instances,omitempty"`      // Optional list of instances to apply this rule
	ExecuteAction string      `yaml:"execute_action,omitempty"` // Optional action to execute on alert
	AlertHours    *AlertHours `yaml:"alert_hours,omitempty"`    // Optional time range for alerts

}

// AlertHours defines time range when alerts should be sent
type AlertHours struct {
	Start    string   `yaml:"start"`              // Start time in HH:MM format (24-hour)
	End      string   `yaml:"end"`                // End time in HH:MM format (24-hour)
	Timezone string   `yaml:"timezone,omitempty"` // Timezone (e.g., "UTC", "America/New_York")
	Days     []string `yaml:"days,omitempty"`     // Days of week (optional: "mon", "tue", etc.)
}

// AlertsConfig holds all alert configurations
type AlertsConfig struct {
	Webhook  WebhookConfig  `yaml:"webhook"`
	Telegram TelegramConfig `yaml:"telegram"`
	Discord  DiscordConfig  `yaml:"discord"`
	Teams    TeamsConfig    `yaml:"teams"`
	Email    EmailConfig    `yaml:"email"`
	WhatsApp WhatsAppConfig `yaml:"whatsapp"`
}

// AlertPayload represents the alert message structure
type AlertPayload struct {
	Type     string      `json:"type"`
	To       string      `json:"to"`
	Message  string      `json:"message"`
	Category string      `json:"category"`
	Instance string      `json:"instance"`
	Value    interface{} `json:"value"`
}

// Monitor represents the database monitor
type Monitor struct {
	config *Config
	logger *log.Logger

	instances map[string]*MonitorInstance
}

type MonitorInstance struct {
	monitor      *Monitor
	dbConfig     *DatabaseConfig
	db           *sql.DB
	alertTracker *AlertTracker
	startedAt    time.Time
}
