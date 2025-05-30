# PostgreSQL Monitor Deployment Guide

This guide covers various deployment methods for the PostgreSQL Monitor with complete setup instructions.

## 📋 Table of Contents

- [Prerequisites](#prerequisites)
- [Local Development](#local-development)
- [CentOS/RHEL Deployment](#centosrhel-deployment)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Email Alert Setup](#email-alert-setup)
- [Security Considerations](#security-considerations)
- [Monitoring & Maintenance](#monitoring--maintenance)

---

## Prerequisites

### System Requirements
- **OS**: Linux (CentOS 7+, RHEL 7+, Ubuntu 18.04+, or Docker)
- **Memory**: 128MB minimum, 256MB recommended
- **CPU**: 0.1 CPU cores minimum, 0.5 cores recommended
- **Storage**: 100MB for application, additional space for logs

### Software Dependencies
- **PostgreSQL Client**: Required for database connectivity
- **Go 1.21+**: For building from source
- **Docker**: For containerized deployment
- **systemd**: For service management on Linux

### Network Requirements
- **Database Access**: TCP connection to PostgreSQL server
- **SMTP Access**: Port 587/465/25 for email alerts (if enabled)
- **Webhook Access**: HTTPS/HTTP access for alert webhooks
- **API Access**: HTTPS access to Telegram, Discord, Teams APIs

---

## Local Development

### Quick Start

1. **Clone and Build**
   ```bash
   git clone https://github.com/warkanum/go-postgres-stat-alert.git
   cd postgres-stat-alert
   make deps
   make build
   ```

2. **Configure**
   ```bash
   cp config.yaml.sample config.yaml
   nano config.yaml  # Edit database and alert settings
   ```

3. **Run**
   ```bash
   make run
   # OR
   ./postgres-stat-alert config.yaml
   ```

### Development Workflow

```bash
# Format and lint code
make fmt
make lint

# Run tests
make test

# Development mode (auto-rebuild on changes)
make dev

# Validate configuration
make validate-config
```

---

## CentOS/RHEL Deployment

### Automated Installation

The installation script handles everything automatically:

```bash
# 1. Build the binary for Linux
make build-linux

# 2. Run the installation script
sudo ./install-centos.sh
```

### What the Installer Does

1. **System Setup**
   - Installs PostgreSQL client and dependencies
   - Creates `postgres-stat-alert` system user/group
   - Sets up directory structure with proper permissions

2. **Binary Installation**
   - Copies binary to `/usr/local/bin/postgres-stat-alert`
   - Sets appropriate permissions and ownership

3. **Service Configuration**
   - Installs systemd service file
   - Configures log rotation
   - Sets up SELinux contexts (if enabled)

4. **Directory Structure**
   ```
   /usr/local/bin/postgres-stat-alert           # Binary
   /etc/postgres-stat-alert/config.yaml         # Configuration
   /var/log/postgres-stat-alert/                # Log files
   /var/lib/postgres-stat-alert/                # Working directory
   /usr/local/share/postgres-stat-alert/scripts/# Sample scripts
   ```

### Manual Post-Installation

1. **Configure Database Connection**
   ```bash
   sudo nano /etc/postgres-stat-alert/config.yaml
   ```

2. **Start Service**
   ```bash
   sudo systemctl start postgres-stat-alert
   sudo systemctl enable postgres-stat-alert
   ```

3. **Check Status**
   ```bash
   sudo systemctl status postgres-stat-alert
   sudo journalctl -u postgres-stat-alert -f
   ```

### Service Management

```bash
# Start/stop/restart
sudo systemctl start postgres-stat-alert
sudo systemctl stop postgres-stat-alert
sudo systemctl restart postgres-stat-alert

# Enable/disable auto-start
sudo systemctl enable postgres-stat-alert
sudo systemctl disable postgres-stat-alert

# View logs
sudo journalctl -u postgres-stat-alert -f
tail -f /var/log/postgres-stat-alert/*.log

# Check service status
make status
```

### Uninstallation

```bash
sudo ./install-centos.sh --uninstall
```

---

## Docker Deployment

### Simple Docker Run

1. **Build Image**
   ```bash
   make docker-build
   ```

2. **Run Container**
   ```bash
   docker run -d \
     --name postgres-stat-alert \
     --restart unless-stopped \
     -v $(pwd)/config.yaml:/etc/postgres-stat-alert/config.yaml:ro \
     -v $(pwd)/logs:/var/log/postgres-stat-alert \
     postgres-stat-alert:latest
   ```

### Docker Compose Deployment

1. **Configure Environment**
   ```bash
   # Copy and edit configuration
   cp config.yaml.sample config.yaml
   nano config.yaml
   
   # Create directories
   mkdir -p logs scripts
   ```

2. **Start Services**
   ```bash
   make docker-up
   # OR
   docker-compose up -d
   ```

3. **View Logs**
   ```bash
   make docker-logs
   # OR
   docker-compose logs -f postgres-stat-alert
   ```

4. **Stop Services**
   ```bash
   make docker-down
   # OR
   docker-compose down
   ```

### Docker Configuration Options

#### Environment Variables Override
```yaml
# In docker-compose.yml
environment:
  - DB_HOST=postgres
  - DB_PORT=5432
  - DB_USER=monitor_user
  - DB_PASSWORD=monitor_password
  - TELEGRAM_BOT_TOKEN=your_token
  - SMTP_PASSWORD=your_smtp_password
```

#### Volume Mounts
```yaml
volumes:
  - ./config.yaml:/etc/postgres-stat-alert/config.yaml:ro  # Configuration
  - ./logs:/var/log/postgres-stat-alert                    # Logs
  - ./scripts:/scripts:ro                               # Custom scripts
  - /etc/ssl/certs:/etc/ssl/certs:ro                   # SSL certificates
```

#### Health Checks
```yaml
healthcheck:
  test: ["CMD", "pgrep", "-f", "postgres-stat-alert"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 10s
```

---

## Kubernetes Deployment

### Basic Kubernetes Manifests

1. **ConfigMap for Configuration**
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: postgres-stat-alert-config
     namespace: monitoring
   data:
     config.yaml: |
       instance: "k8s-cluster-01"
       database:
         host: "postgres.database.svc.cluster.local"
         port: 5432
         username: "monitor_user"
         password: "monitor_password"
         database: "production"
         sslmode: "require"
       # ... rest of config
   ```

2. **Secret for Sensitive Data**
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: postgres-stat-alert-secrets
     namespace: monitoring
   type: Opaque
   data:
     db-password: bW9uaXRvcl9wYXNzd29yZA==      # base64 encoded
     telegram-token: eW91cl90ZWxlZ3JhbV90b2tlbg== # base64 encoded
     smtp-password: eW91cl9zbXRwX3Bhc3N3b3Jk      # base64 encoded
   ```

3. **Deployment**
   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: postgres-stat-alert
     namespace: monitoring
   spec:
     replicas: 1
     selector:
       matchLabels:
         app: postgres-stat-alert
     template:
       metadata:
         labels:
           app: postgres-stat-alert
       spec:
         serviceAccountName: postgres-stat-alert
         securityContext:
           runAsUser: 1001
           runAsGroup: 1001
           fsGroup: 1001
         containers:
         - name: postgres-stat-alert
           image: postgres-stat-alert:latest
           imagePullPolicy: IfNotPresent
           resources:
             requests:
               memory: "128Mi"
               cpu: "100m"
             limits:
               memory: "256Mi"
               cpu: "500m"
           volumeMounts:
           - name: config
             mountPath: /etc/postgres-stat-alert
             readOnly: true
           - name: logs
             mountPath: /var/log/postgres-stat-alert
           env:
           - name: DB_PASSWORD
             valueFrom:
               secretKeyRef:
                 name: postgres-stat-alert-secrets
                 key: db-password
           livenessProbe:
             exec:
               command:
               - pgrep
               - postgres-stat-alert
             initialDelaySeconds: 30
             periodSeconds: 30
           readinessProbe:
             exec:
               command:
               - pgrep
               - postgres-stat-alert
             initialDelaySeconds: 10
             periodSeconds: 10
         volumes:
         - name: config
           configMap:
             name: postgres-stat-alert-config
         - name: logs
           emptyDir: {}
   ```

### Helm Chart Structure

```
helm-chart/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── deployment.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── serviceaccount.yaml
│   └── rbac.yaml
```

---

## Email Alert Setup

### Gmail Configuration

1. **Enable 2FA and App Passwords**
   - Go to Google Account settings
   - Enable 2-Factor Authentication
   - Generate App Password for "Mail"

2. **Configuration**
   ```yaml
   alerts:
     email:
       enabled: true
       smtp_host: "smtp.gmail.com"
       smtp_port: 587
       username: "alerts@yourdomain.com"
       password: "your_app_password_here"  # Not your regular password!
       from_email: "alerts@yourdomain.com"
       from_name: "Database Monitor"
       tls: true
       interval: "3m"
   ```

### Office 365/Outlook Configuration

```yaml
alerts:
  email:
    enabled: true
    smtp_host: "smtp-mail.outlook.com"
    smtp_port: 587
    username: "alerts@company.com"
    password: "your_password"
    from_email: "alerts@company.com"
    from_name: "PostgreSQL Monitor"
    tls: true
    interval: "3m"
```

### Custom SMTP Server

```yaml
alerts:
  email:
    enabled: true
    smtp_host: "mail.company.com"
    smtp_port: 587                    # or 465 for SSL, 25 for plain
    username: "monitor@company.com"
    password: "smtp_password"
    from_email: "noreply@company.com"
    from_name: "Database Monitoring"
    tls: true                        # false for unencrypted
    interval: "5m"
```

### Email Template Customization

The system sends HTML emails with:
- **Color-coded headers** based on alert category
- **Structured tables** with alert details
- **Plain text fallback** for compatibility
- **Professional formatting** suitable for business use

Example email content:
```
Subject: [production-db-01] Database Alert: connection_count

🚨 Database Alert

Message: High number of active connections detected

Instance: production-db-01
Query: connection_count
Category: performance
Timestamp: 2024-01-15 10:30:01 MST
Recipient: dba@company.com
```

---

## Security Considerations

### Database User Permissions

Create a dedicated monitoring user with minimal permissions:

```sql
-- Create monitoring user
CREATE USER monitor_user WITH PASSWORD 'secure_random_password';

-- Grant connection permission
GRANT CONNECT ON DATABASE production_db TO monitor_user;

-- Grant schema access
GRANT USAGE ON SCHEMA public TO monitor_user;

-- Grant read access to monitoring views
GRANT SELECT ON pg_stat_activity TO monitor_user;
GRANT SELECT ON pg_stat_database TO monitor_user;
GRANT SELECT ON pg_statio_user_tables TO monitor_user;
GRANT SELECT ON pg_stat_user_tables TO monitor_user;

-- Grant function execution for size monitoring
GRANT EXECUTE ON FUNCTION pg_database_size(oid) TO monitor_user;
GRANT EXECUTE ON FUNCTION pg_size_pretty(bigint) TO monitor_user;
```

### SSL/TLS Configuration

**Database Connection:**
```yaml
database:
  host: "db.company.com"
  port: 5432
  username: "monitor_user"
  password: "secure_password"
  database: "production"
  sslmode: "verify-full"  # Strongest SSL verification
```

**Email Security:**
```yaml
alerts:
  email:
    smtp_host: "smtp.company.com"
    smtp_port: 587
    tls: true           # Always use encryption
    username: "monitor@company.com"
    password: "app_password"
```

### Secrets Management

#### Environment Variables
```bash
# Set sensitive data via environment
export DB_PASSWORD="secure_db_password"
export TELEGRAM_BOT_TOKEN="123456:telegram_token"
export SMTP_PASSWORD="email_password"
```

#### Docker Secrets
```bash
# Create Docker secrets
echo "secure_password" | docker secret create db_password -
echo "telegram_token" | docker secret create telegram_token -

# Use in docker-compose.yml
secrets:
  - db_password
  - telegram_token
```

#### Kubernetes Secrets
```bash
# Create Kubernetes secrets
kubectl create secret generic postgres-stat-alert-secrets \
  --from-literal=db-password=secure_password \
  --from-literal=telegram-token=your_token \
  --namespace=monitoring
```

### File Permissions

```bash
# Configuration files (readable by service user only)
sudo chown root:postgres-stat-alert /etc/postgres-stat-alert/config.yaml
sudo chmod 640 /etc/postgres-stat-alert/config.yaml

# Log files (writable by service user)
sudo chown postgres-stat-alert:postgres-stat-alert /var/log/postgres-stat-alert/
sudo chmod 755 /var/log/postgres-stat-alert/

# Binary (executable by all, writable by root only)
sudo chown root:root /usr/local/bin/postgres-stat-alert
sudo chmod 755 /usr/local/bin/postgres-stat-alert
```

---

## Monitoring & Maintenance

### Log Management

#### Log Locations
- **systemd**: `journalctl -u postgres-stat-alert`
- **Application logs**: `/var/log/postgres-stat-alert/*.log`
- **Docker logs**: `docker logs postgres-stat-alert`

#### Log Rotation
The installer sets up automatic log rotation:
```bash
# View logrotate configuration
cat /etc/logrotate.d/postgres-stat-alert

# Test log rotation
sudo logrotate -d /etc/logrotate.d/postgres-stat-alert
```

### Performance Monitoring

#### Resource Usage
```bash
# Check memory usage
ps aux | grep postgres-stat-alert

# Check file descriptors
lsof -p $(pgrep postgres-stat-alert)

# Monitor with top/htop
top -p $(pgrep postgres-stat-alert)
```

#### Database Impact
```sql
-- Monitor monitor's database usage
SELECT 
    application_name,
    state,
    query_start,
    query
FROM pg_stat_activity 
WHERE usename = 'monitor_user';

-- Check connection count
SELECT count(*) 
FROM pg_stat_activity 
WHERE usename = 'monitor_user';
```

### Health Checks

#### Service Health
```bash
# Check if service is running
systemctl is-active postgres-stat-alert

# Check if service is enabled
systemctl is-enabled postgres-stat-alert

# Get service status
systemctl status postgres-stat-alert --no-pager
```

#### Alert Health
```bash
# Test alert channels manually
curl -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
  -H "Content-Type: application/json" \
  -d '{"chat_id":"$CHAT_ID","text":"Test message"}'

# Check SMTP connectivity
telnet smtp.gmail.com 587
```

### Backup & Recovery

#### Configuration Backup
```bash
# Backup configuration
sudo cp /etc/postgres-stat-alert/config.yaml \
       /etc/postgres-stat-alert/config.yaml.backup.$(date +%Y%m%d)

# Backup entire configuration directory
sudo tar -czf postgres-stat-alert-config-$(date +%Y%m%d).tar.gz \
          /etc/postgres-stat-alert/
```

#### Log Archival
```bash
# Archive old logs
sudo tar -czf postgres-stat-alert-logs-$(date +%Y%m%d).tar.gz \
          /var/log/postgres-stat-alert/*.log.1*

# Clean up archived logs
sudo find /var/log/postgres-stat-alert/ -name "*.log.*" -mtime +30 -delete
```

### Troubleshooting

#### Common Issues

**Service Won't Start:**
```bash
# Check configuration syntax
make validate-config

# Check service logs
sudo journalctl -u postgres-stat-alert --no-pager

# Check database connectivity
psql "host=localhost port=5432 dbname=mydb user=monitor_user"
```

**Alerts Not Sending:**
```bash
# Check alert intervals
grep -i "interval limit" /var/log/postgres-stat-alert/*.log

# Test individual alert channels
# See email setup section for testing commands
```

**High Resource Usage:**
```bash
# Check query intervals
grep "interval:" /etc/postgres-stat-alert/config.yaml

# Monitor database connections
SELECT * FROM pg_stat_activity WHERE usename = 'monitor_user';
```

#### Log Analysis
```bash
# View recent errors
sudo journalctl -u postgres-stat-alert --since "1 hour ago" | grep -i error

# Monitor alert frequency
tail -f /var/log/postgres-stat-alert/*.log | grep "Alert triggered"

# Check action execution
grep "execute.*action" /var/log/postgres-stat-alert/*.log
```

This deployment guide provides comprehensive instructions for setting up PostgreSQL Monitor in any environment, from development to production Kubernetes clusters.