package monitor

import (
	"fmt"
	"log"
	"os"
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
	logFile, err := os.OpenFile(config.Logging.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(logFile, fmt.Sprintf("[Postgres Stat Alert] (%s) ", config.Instance), log.LstdFlags|log.Lshortfile)
	fmt.Printf("\nLogging to file: %s\n", config.Logging.FilePath)
	// Connect to database
	db, err := connectToDatabase(config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Monitor{
		config:       config,
		db:           db,
		logger:       logger,
		alertTracker: NewAlertTracker(),
	}, nil
}

// Start begins monitoring the database
func (m *Monitor) Start() {
	m.logger.Println("Starting database monitor...")
	fmt.Println("Connected to database:", m.config.Database.Host, m.config.Database.Database)

	// Start monitoring each query in separate goroutines
	for _, query := range m.config.Queries {
		go m.monitorQuery(query)
	}

	// Keep the main thread alive
	select {}
}

// monitorQuery monitors a specific query based on its configuration
func (m *Monitor) monitorQuery(queryConfig QueryConfig) {
	ticker := time.NewTicker(queryConfig.Interval)
	defer ticker.Stop()

	m.logger.Printf("Starting monitoring for query: %s (interval: %v)", queryConfig.Name, queryConfig.Interval)
	fmt.Printf("\nStarting monitoring for query: %s (interval: %v)", queryConfig.Name, queryConfig.Interval)

	for {
		select {
		case <-ticker.C:
			m.executeAndCheck(queryConfig)
		}
	}
}

// executeAndCheck executes a query and checks alert rules
func (m *Monitor) executeAndCheck(queryConfig QueryConfig) {
	m.logger.Printf("Executing query: %s", queryConfig.Name)

	rows, err := m.db.Query(queryConfig.SQL)
	if err != nil {
		m.logger.Printf("Error executing query %s: %v", queryConfig.Name, err)
		return
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		m.logger.Printf("Error getting columns for query %s: %v", queryConfig.Name, err)
		return
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
			m.logger.Printf("Error scanning row for query %s: %v", queryConfig.Name, err)
			continue
		}

		// Check alert rules
		m.checkAlertRules(queryConfig, columns, values)
	}

	if err = rows.Err(); err != nil {
		m.logger.Printf("Error iterating rows for query %s: %v", queryConfig.Name, err)
	}
}

// evaluateCondition checks if a condition is met
func (m *Monitor) evaluateCondition(actual interface{}, condition string, expected interface{}) bool {
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
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}
