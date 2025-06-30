package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/warkanum/go-postgres-stat-alert/pkg/monitor"
)

var version = "dev"

func main() {
	fmt.Println("Postgres Stat Alert🚨 - Monitoring Service")
	fmt.Println("Version: ", version)
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <config-file-path>")
		os.Exit(1)
	}

	configPath := os.Args[1]

	monitor, err := monitor.NewMonitor(configPath)
	if err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}
	defer monitor.Close()

	monitor.Start()
}
