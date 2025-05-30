# Multi-stage build for PostgreSQL Monitor
# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY *.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o postgres-stat-alert .

# Final stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata postgresql-client

# Create non-root user
RUN addgroup -g 1001 monitor && \
    adduser -D -u 1001 -G monitor monitor

# Create directories
RUN mkdir -p /app /var/log/postgres-stat-alert /etc/postgres-stat-alert /scripts && \
    chown -R monitor:monitor /app /var/log/postgres-stat-alert /etc/postgres-stat-alert /scripts

# Copy binary from builder
COPY --from=builder /app/postgres-stat-alert /app/postgres-stat-alert
RUN chmod +x /app/postgres-stat-alert

# Copy sample configuration
COPY config.yaml /etc/postgres-stat-alert/config.yaml.sample
RUN chown monitor:monitor /etc/postgres-stat-alert/config.yaml.sample

# Switch to non-root user
USER monitor

# Set working directory
WORKDIR /app

# Expose health check endpoint (if you add one later)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep -f postgres-stat-alert || exit 1

# Default command
ENTRYPOINT ["/app/postgres-stat-alert"]
CMD ["/etc/postgres-stat-alert/config.yaml"]

# Labels for metadata
LABEL maintainer="hein@warky.dev" \
      description="PostgreSQL Database Monitor with Multi-Channel Alerts" \
      version="1.0.0" \
      org.opencontainers.image.source="https://github.com/warkanum/go-postgres-stat-alert"