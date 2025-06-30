package monitor

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// NewMonitor creates a new database monitor
func NewMonitor(configPath string) (*Monitor, error) {
	// Load configuration
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Setup logging
	logFile, err := os.OpenFile(config.Logging.FilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(logFile, fmt.Sprintf("[Postgres Stat Alert] "), log.LstdFlags|log.Lshortfile)
	fmt.Printf("\nLogging to file: %s\n", config.Logging.FilePath)

	monitor := &Monitor{
		config:    config,
		instances: nil,
		logger:    logger,
		osSignal:  make(chan os.Signal, 1),
	}

	signal.Notify(monitor.osSignal, syscall.SIGINT, syscall.SIGTERM)

	instances := make(map[string]*MonitorInstance)

	if len(config.Database) == 0 {
		return nil, fmt.Errorf("no database configurations found in the config file")
	}

	for _, dbConfig := range config.Database {
		fmt.Printf("Connecting to database: %s at %s\n", dbConfig.Database, dbConfig.Host)

		// Connect to database
		db, err := connectToDatabase(dbConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		instances[dbConfig.Database] = &MonitorInstance{
			monitor:      monitor, // Will be set later
			db:           db,
			dbConfig:     &dbConfig,
			alertTracker: NewAlertTracker(),
		}

	}
	monitor.instances = instances
	return monitor, nil
}

// Start begins monitoring the database
func (m *Monitor) Start() {
	m.logger.Println("Starting database monitor...")

	for _, instance := range m.instances {
		// Start monitoring each query in separate goroutines
		for _, query := range instance.monitor.config.Queries {
			go instance.monitorQuery(query)
		}
	}

	// Keep the main thread alive
	select {
	case sig := <-m.osSignal:
		m.logger.Printf("Received signal: %s. Shutting down...", sig)
		fmt.Printf("\nReceived signal: %s. Shutting down...\n", sig)
	}
}

// monitorQuery monitors a specific query based on its configuration
func (m *MonitorInstance) monitorQuery(queryConfig QueryConfig) {
	ticker := time.NewTicker(queryConfig.Interval)
	defer ticker.Stop()

	m.monitor.logger.Printf("Starting monitoring for query: %s (interval: %v) on host: %s database: %s", queryConfig.Name, queryConfig.Interval, m.dbConfig.Host, m.dbConfig.Database)
	fmt.Printf("\nStarting monitoring for query: %s (interval: %v) on host: %s database: %s", queryConfig.Name, queryConfig.Interval, m.dbConfig.Host, m.dbConfig.Database)

	for {
		select {
		case <-ticker.C:
			err := m.executeAndCheck(queryConfig)
			if err != nil {
				//If an error occurs, send alerts for all alert rules
				for r := range queryConfig.AlertRules {
					rule := queryConfig.AlertRules[r]
					rule.Message = fmt.Sprintf("Error executing query %s: %v", queryConfig.Name, err)
					rule.Category = "error"
					rule.Value = err.Error()

					m.sendAlerts(queryConfig.Name, rule)
				}
			}
		}
	}
}

// executeAndCheck executes a query and checks alert rules
func (m *MonitorInstance) executeAndCheck(queryConfig QueryConfig) error {
	m.monitor.logger.Printf("Executing query: %s", queryConfig.Name)

	if strings.HasPrefix(queryConfig.SQL, "[started]") {
		now := time.Now()
		if m.startedAt.IsZero() {
			m.startedAt = now
			m.checkAlertRules(queryConfig, nil, []interface{}{1})
		}

		return nil
	}

	rows, err := m.db.Query(queryConfig.SQL)
	if err != nil {
		m.monitor.logger.Printf("Error executing query %s: %v", queryConfig.Name, err)
		return fmt.Errorf("failed to execute query %s: %w", queryConfig.Name, err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		m.monitor.logger.Printf("Error getting columns for query %s: %v", queryConfig.Name, err)
		return fmt.Errorf("failed to get columns for query %s: %w", queryConfig.Name, err)
	}

	// Process results
	for rows.Next() {
		// Create a slice to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		err = rows.Scan(valuePtrs...)
		if err != nil {
			m.monitor.logger.Printf("Error scanning row for query %s: %v", queryConfig.Name, err)
			continue
		}

		// Check alert rules
		m.checkAlertRules(queryConfig, columns, values)
	}

	if err = rows.Err(); err != nil {
		m.monitor.logger.Printf("Error iterating rows for query %s: %v", queryConfig.Name, err)
		return fmt.Errorf("error iterating rows for query %s: %w", queryConfig.Name, err)
	}
	return nil
}

// evaluateCondition checks if a condition is met
func (m *MonitorInstance) evaluateCondition(actual interface{}, condition string, expected interface{}) bool {
	// Convert to float64 for numeric comparisons
	actualFloat, actualOk := convertToFloat64(actual)
	expectedFloat, expectedOk := convertToFloat64(expected)

	if actualOk && expectedOk {
		switch condition {
		case "gt":
			return actualFloat > expectedFloat
		case "lt":
			return actualFloat < expectedFloat
		case "gte":
			return actualFloat >= expectedFloat
		case "lte":
			return actualFloat <= expectedFloat
		case "eq":
			return actualFloat == expectedFloat
		case "ne":
			return actualFloat != expectedFloat
		}
	}

	// String comparison fallback
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)

	switch condition {
	case "eq":
		return actualStr == expectedStr
	case "ne":
		return actualStr != expectedStr
	}

	return false
}

// convertToFloat64 attempts to convert an interface{} to float64
func convertToFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case []uint8: // Handle PostgreSQL numeric types that come as byte slices
		str := string(v)
		if f, err := parseFloat(str); err == nil {
			return f, true
		}
	}
	return 0, false
}

// parseFloat is a helper function to parse string to float64
func parseFloat(str string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(str, "%f", &f)
	return f, err
}

// Close closes the database connection and cleans up resources
func (m *Monitor) Close() error {
	if m.instances != nil {
		for _, instance := range m.instances {
			if instance.db != nil {
				return instance.db.Close()
			}
		}
		m.instances = nil
	}

	return nil
}
