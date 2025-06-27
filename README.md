# PostgreSQL Database Monitor

A comprehensive PostgreSQL monitoring solution with multi-database support, multi-channel alerting, automated actions, and flexible deployment options.

## 🚀 Features

- **Multi-Database Monitoring**: Connect to multiple PostgreSQL instances simultaneously
- **Real-time Monitoring**: Custom SQL queries with configurable intervals
- **Multi-Channel Alerts**: Email, Telegram, Discord, Teams, and Webhooks
- **Automated Actions**: Execute scripts/commands when alerts trigger
- **Smart Rate Limiting**: Prevent alert spam with per-channel intervals
- **Flexible Deployment**: systemd service, Docker, or Kubernetes
- **Instance Identification**: Track multiple database environments
- **Professional Email Templates**: HTML formatted alerts with color coding

## 📋 Quick Start

### 1. Build and Configure

```bash
# Clone and build
git clone https://github.com/warkanum/go-postgres-stat-alert.git
cd postgres-stat-alert
make build

# Configure
cp config.yaml.sample config.yaml
nano config.yaml  # Edit databases and alert settings
```

### 2. Run

```bash
# Local development
./postgres-stat-alert config.yaml

# Or with Docker
make docker-up

# Or install as systemd service (CentOS/RHEL)
sudo ./install-centos.sh
```

## 🔧 Configuration Overview

```yaml
databases:
  - instance: "production-db-01"
    host: "localhost"
    port: 5432
    username: "monitor_user"
    password: "secure_password"
    database: "production"
    sslmode: "verify-full"
  
  - instance: "staging-db-02"
    host: "staging.example.com"
    port: 5432
    username: "monitor_user"
    database: "staging"
    sslmode: "require"

alerts:
  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "alerts@company.com"
    password: "app_password"
    tls: true
    interval: "3m"
  
  telegram:
    enabled: true
    bot_token: "123456:your_bot_token"
    chat_id: "your_chat_id"
    interval: "1m"

queries:
  - name: "connection_count"
    sql: "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'"
    interval: "30s"
    alert_rules:
      - condition: "gt"
        value: 100
        message: "High connection count detected"
        category: "performance"
        to: "dba@company.com"
        channels: ["email", "telegram"]
        execute_action: "/scripts/restart_pgbouncer.sh"
```

## 📊 Monitoring Capabilities

### Multi-Database Architecture
- **Simultaneous Monitoring**: Monitor multiple PostgreSQL instances from a single service
- **Instance Isolation**: Each database connection is managed independently 
- **Per-Instance Tracking**: Alerts include instance identification for easy troubleshooting
- **Centralized Configuration**: Single config file manages all database connections

### Built-in Queries
- **Connection Monitoring**: Active connections, long-running queries
- **Performance Metrics**: Cache hit ratio, slow queries, replication lag
- **Storage Monitoring**: Database size, table bloat, disk usage
- **Security Alerts**: Failed connections, unusual activity
- **System Health**: Service availability, resource usage

### Alert Channels

| Channel | Features | Use Case |
|---------|----------|----------|
| **Email** | HTML templates, professional formatting | Management reports, audit trails |
| **Telegram** | Instant notifications, group chats | Development teams, immediate alerts |
| **Discord** | Rich embeds, color coding | Team collaboration, status updates |
| **Teams** | Professional cards, structured data | Business communications |
| **Webhook** | Custom integrations, JSON payload | External systems, custom workflows |

## 🛠️ Deployment Options

### CentOS/RHEL (Recommended for Production)

```bash
# One-command installation
make build-linux
sudo ./install-centos.sh

# Service management
sudo systemctl start postgres-stat-alert
sudo systemctl enable postgres-stat-alert
sudo journalctl -u postgres-stat-alert -f
```

### Docker

```bash
# Using Docker Compose
make docker-up
make docker-logs

# Or standalone Docker
docker run -d \
  --name postgres-stat-alert \
  -v $(pwd)/config.yaml:/etc/postgres-stat-alert/config.yaml:ro \
  -v $(pwd)/logs:/var/log/postgres-stat-alert \
  postgres-stat-alert:latest
```

### Kubernetes

```bash
# Apply manifests
kubectl apply -f k8s/
kubectl logs -f deployment/postgres-stat-alert -n monitoring
```

## 🔒 Security Features

- **Minimal Database Permissions**: Read-only monitoring user per database
- **SSL/TLS Support**: Encrypted database and SMTP connections
- **Secrets Management**: Environment variables and external secret stores
- **Service Isolation**: Non-privileged user execution
- **Configuration Security**: Protected file permissions

## 📚 Documentation

- **[Configuration Guide](config.md)**: Complete configuration reference
- **[Deployment Guide](deployment.md)**: Detailed deployment instructions
- **[Setup Guide](setup.md)**: Alert service setup (Telegram, Discord, etc.)

## 🎯 Example Use Cases

### Multi-Environment Monitoring
```yaml
databases:
  - instance: "prod-primary"
    host: "prod-db-01.internal"
    database: "app_production"
  
  - instance: "prod-replica"
    host: "prod-db-02.internal"
    database: "app_production"
  
  - instance: "staging"
    host: "staging-db.internal"
    database: "app_staging"

# Production: Email + Teams (management)
channels: ["email", "teams"]

# Staging: Discord + Telegram (development)  
channels: ["discord", "telegram"]
```

### Automated Remediation
```yaml
execute_action: "/scripts/restart_connection_pooler.sh"  # Auto-restart on high load
execute_action: "/scripts/kill_slow_queries.py --timeout 300"  # Terminate slow queries
execute_action: "/scripts/cleanup_temp_files.sh"  # Clean up storage issues
```

### Escalation Workflows
```yaml
# Warning level
- condition: "gt"
  value: 80
  channels: ["discord"]
  
# Critical level  
- condition: "gt"
  value: 95
  channels: ["email", "telegram", "teams"]
  execute_action: "/emergency/scale_up.sh"
```

## 🔧 Development

### Requirements
- Go 1.21+
- PostgreSQL client libraries
- Make (optional, for convenience)

### Building
```bash
# Install dependencies
make deps

# Build for current platform
make build

# Build for Linux (deployment)
make build-linux

# Run tests
make test

# Development mode (auto-rebuild)
make dev
```


## 📝 Todo

- User interface to setup monitors and alerts
- More monitor types for example Shell commands or programs to check for errors in logs.
- Remote Registrar, used to check all running instances are online and will be reporting.


### Contributing
1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Format code: `make fmt`
5. Lint code: `make lint`
6. Submit pull request

## 📈 Performance

- **Memory Usage**: ~50-100MB baseline + ~10MB per database
- **CPU Usage**: <1% during normal operation
- **Database Impact**: Minimal (read-only queries)
- **Network**: Low bandwidth usage
- **Scalability**: Handles 100+ queries across multiple databases efficiently

## 🐛 Troubleshooting

### Service Issues
```bash
# Check service status
make status

# View logs
make logs
tail -f /var/log/postgres-stat-alert/*.log

# Validate configuration
make validate-config
```

### Database Connectivity
```bash
# Test database connections
psql "host=localhost port=5432 dbname=mydb user=monitor_user"

# Check monitoring queries per instance
SELECT count(*) FROM pg_stat_activity WHERE state = 'active';
```

### Alert Issues
```bash
# Test Telegram bot
curl -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
  -H "Content-Type: application/json" \
  -d '{"chat_id":"$CHAT_ID","text":"Test"}'

# Check email configuration
telnet smtp.gmail.com 587
```

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🤝 Support

- **Issues**: [GitHub Issues](https://github.com/warkanum/go-postgres-stat-alert/issues)
- **Discussions**: [GitHub Discussions](https://github.com/warkanum/go-postgres-stat-alert/discussions)
- **Documentation**: [Wiki](https://github.com/warkanum/go-postgres-stat-alert/wiki)

## 🎉 Acknowledgments

- PostgreSQL community for excellent monitoring views
- Go community for robust libraries
- Alert service providers for reliable APIs

