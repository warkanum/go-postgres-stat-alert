package monitor

import (
	"database/sql"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// loadConfig reads and parses the YAML configuration file
func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// connectToDatabase establishes a connection to PostgreSQL
func connectToDatabase(dbConfig DatabaseConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.Username, dbConfig.Password, dbConfig.Database, dbConfig.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("SET APPLICATION_NAME = 'postgres-stat-alert'") // Set application name for easier identification in logs
	if err != nil {
		return nil, err
	}

	return db, nil
}
