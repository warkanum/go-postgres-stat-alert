# PostgreSQL Monitor Makefile

# Variables
BINARY_NAME=postgres-stat-alert
DOCKER_IMAGE=postgres-stat-alert
DOCKER_TAG=latest
CONFIG_FILE=config.yaml

# Go build variables
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
.PHONY: all
all: clean build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: $(BINARY_NAME)"

# Build for Linux (useful when building on other platforms)
.PHONY: build-linux
build-linux:
	@echo "Building $(BINARY_NAME) for linux/amd64..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	@echo "Build complete: $(BINARY_NAME)-linux-amd64"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	go clean
	@echo "Clean complete"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run with sample config
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME) with $(CONFIG_FILE)..."
	./$(BINARY_NAME) $(CONFIG_FILE)

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod verify

# Update dependencies
.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker build complete"

# Run with Docker Compose
.PHONY: docker-up
docker-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d
	@echo "Services started"

# Stop Docker Compose services
.PHONY: docker-down
docker-down:
	@echo "Stopping Docker Compose services..."
	docker-compose down
	@echo "Services stopped"

# View Docker logs
.PHONY: docker-logs
docker-logs:
	docker-compose logs -f postgres-stat-alert

# Install on CentOS (requires root)
.PHONY: install-centos
install-centos: build-linux
	@echo "Installing on CentOS..."
	sudo ./install-centos.sh
	@echo "Installation complete"

# Uninstall from CentOS (requires root)
.PHONY: uninstall-centos
uninstall-centos:
	@echo "Uninstalling from CentOS..."
	sudo ./install-centos.sh --uninstall
	@echo "Uninstallation complete"

# Create release package
.PHONY: package
package: clean build-linux
	@echo "Creating release package..."
	mkdir -p release
	cp $(BINARY_NAME)-linux-amd64 release/$(BINARY_NAME)
	cp $(CONFIG_FILE) release/config.yaml.sample
	cp postgres-stat-alert.service release/
	cp install-centos.sh release/
	cp Dockerfile release/
	cp docker-compose.yml release/
	cp README.md release/ 2>/dev/null || echo "README.md not found, skipping"
	tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C release .
	@echo "Release package created: release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz"

# Generate systemd service status
.PHONY: status
status:
	@echo "Checking service status..."
	systemctl status postgres-stat-alert || true

# View systemd logs
.PHONY: logs
logs:
	@echo "Viewing service logs..."
	journalctl -u postgres-stat-alert -f

# Restart systemd service
.PHONY: restart
restart:
	@echo "Restarting service..."
	sudo systemctl restart postgres-stat-alert
	@echo "Service restarted"

# Validate configuration
.PHONY: validate-config
validate-config:
	@echo "Validating configuration..."
	@if [ -f "$(CONFIG_FILE)" ]; then \
		python3 -c "import yaml; yaml.safe_load(open('$(CONFIG_FILE)'))" && echo "✓ Configuration is valid"; \
	else \
		echo "✗ Configuration file $(CONFIG_FILE) not found"; \
		exit 1; \
	fi

# Development: watch for changes and rebuild
.PHONY: dev
dev:
	@echo "Starting development mode (requires 'entr' tool)..."
	find . -name "*.go" | entr -r make run

# Show help
.PHONY: help
help:
	@echo "PostgreSQL Monitor Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build           Build the binary for current platform"
	@echo "  build-linux     Build the binary for Linux amd64"
	@echo "  clean           Remove build artifacts"
	@echo "  test            Run tests"
	@echo "  run             Build and run with sample config"
	@echo "  fmt             Format Go code"
	@echo "  lint            Lint Go code (requires golangci-lint)"
	@echo "  deps            Install dependencies"
	@echo "  update-deps     Update dependencies"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build    Build Docker image"
	@echo "  docker-up       Start with Docker Compose"
	@echo "  docker-down     Stop Docker Compose"
	@echo "  docker-logs     View Docker logs"
	@echo ""
	@echo "CentOS deployment:"
	@echo "  install-centos  Install on CentOS (requires sudo)"
	@echo "  uninstall-centos Uninstall from CentOS (requires sudo)"
	@echo "  status          Check systemd service status"
	@echo "  logs            View systemd service logs"
	@echo "  restart         Restart systemd service"
	@echo ""
	@echo "Utility targets:"
	@echo "  package         Create release package"
	@echo "  validate-config Validate YAML configuration"
	@echo "  dev             Development mode with auto-rebuild"
	@echo "  help            Show this help message"