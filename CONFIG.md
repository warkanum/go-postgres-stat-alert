# PostgreSQL Monitor Configuration Guide

This guide explains all configuration options for the PostgreSQL Database Monitor.

## 📋 Table of Contents

- [Configuration Overview](#configuration-overview)
- [Instance Configuration](#instance-configuration)
- [Database Configuration](#database-configuration)
- [Logging Configuration](#logging-configuration)
- [Alert Configuration](#alert-configuration)
- [Query Configuration](#query-configuration)
- [Complete Examples](#complete-examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Configuration Overview

The configuration file uses YAML format with the following top-level sections:


```yaml
databases: { ... }
logging: { ... }
alerts: { ... }
queries: [ ... ]
```

---

## Instance Configuration

Instance has been moved to databases in the newer versions.

**Examples:**
- `"production-primary"`
- `"staging-replica"`
- `"dev-local"`
- `"us-west-prod-01"`

**Default:** `"default"` if not specified

**Usage:**
- Appears in all log entries: `[production-db-01-Monitor]`
- Included in alert messages for multi-database environments
- Helps distinguish between multiple monitoring instances

---

## Database Configuration

### `database` (object, required)

PostgreSQL connection parameters. Multiple databases support added in newer versions. Key changed to databases.

```yaml
databases:
  - instance: "production-db-01"
    host: "localhost"           # Database server hostname/IP
    port: 5432                 # PostgreSQL port (default: 5432)
    username: "monitor_user"    # Database username
    password: "secure_password" # Database password
    database: "app_database"    # Database name to connect to
    sslmode: "disable"         # SSL connection mode
```

#### SSL Mode Options

| Value | Description |
|-------|-------------|
| `disable` | No SSL encryption |
| `require` | SSL required, certificate not verified |
| `verify-ca` | SSL required, verify certificate authority |
| `verify-full` | SSL required, verify certificate and hostname |

#### Security Recommendations

**Production:**
```yaml
databases:
  - instance: "production-db-01"
    host: "db.company.com"
    port: 5432
    username: "monitor_readonly"
    password: "complex_password_123!"
    database: "production_db"
    sslmode: "verify-full"
```

**Development:**
```yaml
databases:
  - instance: "dev-db-01"
    host: "localhost"
    port: 5432
    username: "postgres"
    password: "devpass"
    database: "dev_db"
    sslmode: "disable"
```

---

## Logging Configuration

### `logging` (object, required)

Controls where log files are written.

```yaml
logging:
  file_path: "/var/log/postgres-stat-alert.log"
```

#### File Path Examples

**Linux/Unix:**
```yaml
logging:
  file_path: "/var/log/postgres-stat-alert.log"           # System logs
  # or
  file_path: "/home/user/logs/db-monitor.log"          # User directory
  # or  
  file_path: "./logs/monitor.log"                      # Relative path
```

**Windows:**
```yaml
logging:
  file_path: "C:\\logs\\postgres-stat-alert.log"         # Windows path
  # or
  file_path: ".\\logs\\monitor.log"                    # Relative Windows path
```

#### Log Format

Logs include timestamp, instance name, and message:
```
2024-01-15 10:30:01 [production-db-01-Monitor] main.go:123: Starting database monitor for instance: production-db-01
2024-01-15 10:30:01 [production-db-01-Monitor] main.go:156: Starting monitoring for query: connection_count (interval: 30s)
2024-01-15 10:30:15 [production-db-01-Monitor] main.go:234: Alert triggered for query connection_count: High connection count
```

---

## Alert Configuration

### `alerts` (object, required)

Configures alert channels and their intervals.

```yaml
alerts:
  webhook: { ... }
  telegram: { ... }
  discord: { ... }
  teams: { ... }
```

### Webhook Alerts

Generic HTTP webhook for custom alert systems.

```yaml
alerts:
  webhook:
    enabled: true                              # Enable/disable webhook alerts
    url: "https://api.company.com/alerts"     # Webhook endpoint URL
    interval: "5m"                            # Minimum time between alerts
```

**Payload Format:**
```json
{
  "type": "database_alert",
  "to": "admin@company.com",
  "message": "[connection_count] High connection count detected",
  "category": "performance",
  "instance": "production-db-01"
}
```

### Telegram Alerts

Send alerts via Telegram bot.

```yaml
alerts:
  telegram:
    enabled: true
    bot_token: "123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ"  # From @BotFather
    chat_id: "12345678"                                 # Your chat ID
    interval: "1m"                                      # Rate limit friendly
```

**Setup Steps:**
1. Message `@BotFather` on Telegram
2. Send `/newbot` and follow prompts
3. Save the bot token
4. Send a message to your bot
5. Visit `https://api.telegram.org/bot<TOKEN>/getUpdates` to get chat_id

**Chat ID Types:**
- Personal chat: Positive number (e.g., `"12345678"`)
- Group chat: Negative number (e.g., `"-987654321"`)

### Discord Alerts

Rich embed alerts in Discord channels.

```yaml
alerts:
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/123/abc..."
    interval: "30s"
```

**Setup Steps:**
1. Go to Discord channel
2. Channel Settings → Integrations → Webhooks
3. Create New Webhook
4. Copy webhook URL

**Features:**
- Color-coded embeds by alert category
- Timestamp formatting
- Rich text formatting

### Teams Alerts

Professional alerts for Microsoft Teams.

```yaml
alerts:
  teams:
    enabled: true
    webhook_url: "https://outlook.office.com/webhook/..."
    interval: "2m"
```

**Setup Steps:**
1. In Teams channel: ⋯ → Connectors
2. Find "Incoming Webhook" → Configure
3. Name webhook, copy URL

**Features:**
- Professional message cards
- Structured fact tables
- Color themes by category

### Email Alerts

SMTP email alerts for formal notifications and record-keeping.

```yaml
alerts:
  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "alerts@company.com"
    password: "app_password_here"
    from_email: "alerts@company.com"
    from_name: "PostgreSQL Monitor"
    tls: true
    interval: "3m"
```

**SMTP Configuration:**

| Field | Description | Example |
|-------|-------------|---------|
| `smtp_host` | SMTP server hostname | `"smtp.gmail.com"` |
| `smtp_port` | SMTP server port | `587` (TLS), `465` (SSL), `25` (plain) |
| `username` | SMTP authentication username | `"alerts@company.com"` |
| `password` | SMTP authentication password | `"app_password_123"` |
| `from_email` | Sender email address | `"alerts@company.com"` |
| `from_name` | Sender display name | `"PostgreSQL Monitor"` |
| `tls` | Use TLS encryption | `true` (recommended) |

**Common SMTP Providers:**

```yaml
# Gmail
email:
  smtp_host: "smtp.gmail.com"
  smtp_port: 587
  tls: true
  # Use app password, not regular password

# Outlook/Hotmail
email:
  smtp_host: "smtp-mail.outlook.com"
  smtp_port: 587
  tls: true

# Yahoo
email:
  smtp_host: "smtp.mail.yahoo.com"
  smtp_port: 587
  tls: true

# Custom SMTP server
email:
  smtp_host: "mail.company.com"
  smtp_port: 587
  tls: true
```

**Features:**
- HTML formatted emails with color coding
- Plain text fallback for compatibility
- Professional email templates
- Multipart MIME messages
- TLS/SSL encryption support

### Alert Intervals

Control how frequently alerts are sent for the same query.

| Service | Default Interval | Recommended Range |
|---------|------------------|-------------------|
| Webhook | 5 minutes | 1m - 10m |
| Telegram | 1 minute | 30s - 5m |
| Discord | 30 seconds | 15s - 2m |
| Teams | 2 minutes | 1m - 10m |

**Interval Format:**
- `"30s"` - 30 seconds
- `"5m"` - 5 minutes  
- `"1h"` - 1 hour
- `"24h"` - 24 hours

---

## Query Configuration

### `queries` (array, required)

List of database queries to monitor with their alert rules.

```yaml
queries:
  - name: "query_identifier"
    sql: "SELECT count(*) FROM table"
    interval: "30s"
    alert_rules: [ ... ]
    parameters: { ... }  # Optional
```

### Query Fields

#### `name` (string, required)
Unique identifier for the query.
```yaml
name: "connection_count"
```

#### `sql` (string, required)
PostgreSQL query to execute. Can be single line or multi-line.

**Single line:**
```yaml
sql: "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'"
```

**Multi-line:**
```yaml
sql: |
  SELECT 
    schemaname, 
    tablename,
    n_tup_ins + n_tup_upd + n_tup_del as total_changes
  FROM pg_stat_user_tables 
  WHERE n_tup_ins + n_tup_upd + n_tup_del > 1000
  ORDER BY total_changes DESC
  LIMIT 10
```

#### `interval` (duration, required)
How often to execute the query.

```yaml
interval: "30s"    # Every 30 seconds
interval: "5m"     # Every 5 minutes
interval: "1h"     # Every hour
interval: "24h"    # Once per day
```

#### `alert_rules` (array, required)
Conditions that trigger alerts.

```yaml
alert_rules:
  - condition: "gt"                              # Condition type
    value: 100                                  # Threshold value
    message: "High connection count"             # Alert message
    category: "performance"                     # Alert category
    to: "admin@company.com"                    # Recipient (for webhook)
    channels: ["telegram", "discord"]          # Specific channels (optional)
    execute_action: "/scripts/restart_pool.sh" # Command to execute (optional)
```

**Condition Types:**
- `gt` - Greater than
- `lt` - Less than
- `gte` - Greater than or equal
- `lte` - Less than or equal
- `eq` - Equal to
- `ne` - Not equal to

**Categories:**
- `performance` - Performance issues (orange in Discord)
- `security` - Security concerns (red in Discord)
- `storage` - Storage/capacity issues (yellow in Discord)
- `maintenance` - Maintenance needs (blue in Discord)
- `critical` - Critical issues (red in Discord)

**Execute Action:**
- Optional command/script to run when alert triggers
- Can include arguments: `"/scripts/cleanup.sh --force"`
- Executes with 30-second timeout
- Environment variables provided to script:
  - `MONITOR_INSTANCE` - Instance name
  - `MONITOR_QUERY` - Query name that triggered alert
  - `MONITOR_MESSAGE` - Alert message
  - `MONITOR_CATEGORY` - Alert category
  - `MONITOR_TO` - Alert recipient

#### `parameters` (object, optional)
Reserved for future use (parameterized queries).

### Query Examples

#### Connection Monitoring
```yaml
- name: "active_connections"
  sql: "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'"
  interval: "30s"
  alert_rules:
    - condition: "gt"
      value: 80
      message: "Connection count approaching limit"
      category: "performance"
      to: "dba@company.com"
      channels: ["telegram"]
    - condition: "gt"
      value: 100
      message: "Critical: Connection limit exceeded"
      category: "critical"
      to: "emergency@company.com"
```

#### Database Size Monitoring
```yaml
- name: "database_size"
  sql: |
    SELECT 
      pg_size_pretty(pg_database_size(current_database())) as size_pretty,
      pg_database_size(current_database()) as size_bytes
  interval: "1h"
  alert_rules:
    - condition: "gt"
      value: 50000000000  # 50GB
      message: "Database size exceeds 50GB"
      category: "storage"
      to: "infrastructure@company.com"
      channels: ["teams", "webhook"]
```

#### Long Running Queries
```yaml
- name: "long_queries"
  sql: |
    SELECT count(*) 
    FROM pg_stat_activity 
    WHERE state = 'active' 
    AND now() - query_start > interval '10 minutes'
  interval: "2m"
  alert_rules:
    - condition: "gt"
      value: 0
      message: "Long running queries detected (>10min)"
      category: "performance"
      to: "dev-team@company.com"
      channels: ["discord"]
```

#### Replication Lag
```yaml
- name: "replication_lag"
  sql: |
    SELECT CASE 
      WHEN pg_is_in_recovery() THEN 
        EXTRACT(EPOCH FROM (now() - pg_last_xact_replay_timestamp()))
      ELSE 0 
    END as lag_seconds
  interval: "30s"
  alert_rules:
    - condition: "gt"
      value: 60
      message: "Replication lag exceeds 1 minute"
      category: "critical"
      to: "dba@company.com"
      channels: ["telegram", "teams"]
```

---

## Execute Actions

### Overview

Execute actions allow automatic execution of scripts or commands when alerts are triggered. This enables automated remediation, logging, or notification workflows.

### Configuration

```yaml
alert_rules:
  - condition: "gt"
    value: 100
    message: "High load detected"
    category: "performance"
    to: "admin@company.com"
    execute_action: "/scripts/restart_service.sh --force"
```

### Security Considerations

#### Script Permissions
- Ensure scripts have appropriate execute permissions: `chmod +x /scripts/script.sh`
- Scripts run with the same user privileges as the monitor process
- Use dedicated service accounts with minimal required permissions

#### Path Safety
- Use absolute paths for security: `/usr/local/scripts/action.sh`
- Avoid relative paths or commands in PATH for security
- Validate script locations during deployment

#### Input Validation
- Scripts receive environment variables, not direct user input
- Environment variables are automatically escaped
- No SQL injection risk from execute actions

### Environment Variables

When your script executes, these variables are available:

| Variable | Description | Example |
|----------|-------------|---------|
| `MONITOR_INSTANCE` | Instance name | `"production-db-01"` |
| `MONITOR_QUERY` | Query that triggered | `"connection_count"` |
| `MONITOR_MESSAGE` | Alert message | `"High connection count"` |
| `MONITOR_CATEGORY` | Alert category | `"performance"` |
| `MONITOR_TO` | Alert recipient | `"admin@company.com"` |

### Script Examples

#### Bash Script Example
```bash
#!/bin/bash
# /scripts/restart_pooler.sh

echo "Alert triggered on $MONITOR_INSTANCE"
echo "Query: $MONITOR_QUERY"
echo "Message: $MONITOR_MESSAGE"

# Restart connection pooler
systemctl restart pgbouncer

# Log the action
echo "$(date): Restarted pgbouncer due to $MONITOR_MESSAGE" >> /var/log/auto-remediation.log

# Exit with success
exit 0
```

#### Python Script Example
```python
#!/usr/bin/env python3
# /scripts/cleanup_connections.py

import os
import subprocess
import sys
import logging

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Get environment variables
instance = os.environ.get('MONITOR_INSTANCE', 'unknown')
query = os.environ.get('MONITOR_QUERY', 'unknown')
message = os.environ.get('MONITOR_MESSAGE', 'unknown')

logger.info(f"Executing cleanup action for {instance}")
logger.info(f"Triggered by: {query} - {message}")

try:
    # Kill long-running queries
    result = subprocess.run([
        'psql', '-h', 'localhost', '-U', 'admin', '-d', 'postgres',
        '-c', "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '10 minutes';"
    ], capture_output=True, text=True, timeout=30)
    
    if result.returncode == 0:
        logger.info("Successfully terminated long-running queries")
        sys.exit(0)
    else:
        logger.error(f"Failed to terminate queries: {result.stderr}")
        sys.exit(1)
        
except subprocess.TimeoutExpired:
    logger.error("Cleanup action timed out")
    sys.exit(1)
except Exception as e:
    logger.error(f"Unexpected error: {e}")
    sys.exit(1)
```

#### PowerShell Script Example (Windows)
```powershell
# /scripts/restart_service.ps1

param(
    [string]$ServiceName = "PostgreSQL"
)

$instance = $env:MONITOR_INSTANCE
$query = $env:MONITOR_QUERY
$message = $env:MONITOR_MESSAGE

Write-Host "Alert triggered on $instance"
Write-Host "Query: $query"
Write-Host "Message: $message"

try {
    Restart-Service -Name $ServiceName -Force
    Write-Host "Successfully restarted $ServiceName"
    
    # Log to Windows Event Log
    Write-EventLog -LogName Application -Source "PostgreSQL Monitor" -EntryType Information -EventId 1001 -Message "Restarted $ServiceName due to: $message"
    
    exit 0
} catch {
    Write-Error "Failed to restart service: $_"
    exit 1
}
```

### Execution Behavior

#### Timeout
- All actions have a 30-second timeout
- Actions exceeding timeout are terminated
- Timeout prevents hanging processes

#### Output Handling
- Standard output (stdout) is logged for debugging
- Standard error (stderr) is logged on failure
- Both are captured but not sent to alerts

#### Error Handling
- Non-zero exit codes are logged as failures
- Action failures don't prevent alert sending
- Failed actions are retried on next alert trigger

#### Execution Order
1. Alert condition evaluated
2. Execute action runs (if specified)
3. Alert notifications sent to channels

### Use Cases

#### Automated Remediation
```yaml
- condition: "gt"
  value: 90
  message: "Connection pool exhausted"
  category: "critical"
  to: "oncall@company.com"
  execute_action: "/scripts/restart_pgbouncer.sh"
```

#### Data Collection
```yaml
- condition: "gt"
  value: 10
  message: "Slow queries detected"
  category: "performance"
  to: "dev-team@company.com"
  execute_action: "/scripts/capture_slow_queries.py --duration 300"
```

#### External Notifications
```yaml
- condition: "eq"
  value: 0
  message: "Service offline"
  category: "critical"
  to: "emergency@company.com"
  execute_action: "/scripts/page_oncall.sh --service database --priority high"
```

#### Log Rotation
```yaml
- condition: "gt"
  value: 1000000000  # 1GB
  message: "Log files growing large"
  category: "maintenance"
  to: "sysadmin@company.com"
  execute_action: "/scripts/rotate_postgres_logs.sh"
```

### Best Practices

#### Script Design
- Keep scripts simple and focused
- Use proper error handling and exit codes
- Log actions for audit trails
- Test scripts independently before deployment

#### Security
- Use dedicated service accounts
- Limit script permissions to minimum required
- Store scripts in protected directories
- Regularly audit script contents

#### Reliability
- Handle timeouts gracefully
- Implement idempotent actions (safe to run multiple times)
- Avoid actions that could cascade failures
- Test scripts under various conditions

#### Monitoring
- Log all script executions
- Monitor script success/failure rates
- Set up alerts for script failures
- Regular review of action effectiveness

### Testing Execute Actions

#### Manual Testing
```bash
# Test script manually with environment variables
MONITOR_INSTANCE="test" \
MONITOR_QUERY="connection_count" \
MONITOR_MESSAGE="Test alert" \
MONITOR_CATEGORY="performance" \
MONITOR_TO="admin@test.com" \
/scripts/your_script.sh
```

#### Configuration Testing
```yaml
# Create a test alert that always triggers
- name: "test_action"
  sql: "SELECT 1"  # Always returns 1
  interval: "1m"
  alert_rules:
    - condition: "eq"
      value: 1
      message: "Test action execution"
      category: "maintenance"
      to: "test@company.com"
      execute_action: "/scripts/test_action.sh"
      channels: []  # Don't send actual alerts during testing
```

---

## Complete Examples

### Production Environment

```yaml
instance: "production-primary-db"

database:
  host: "prod-db.company.com"
  port: 5432
  username: "monitor_user"
  password: "prod_secure_password_123!"
  database: "app_production"
  sslmode: "verify-full"

logging:
  file_path: "/var/log/postgres-stat-alert/production.log"

alerts:
  webhook:
    enabled: true
    url: "https://alerts.company.com/webhook"
    interval: "5m"
  
  telegram:
    enabled: true
    bot_token: "123456789:ABCdefGhIJKlmNoPQRsTUVwxyZ"
    chat_id: "-100123456789"  # DBA team group
    interval: "1m"
  
  discord:
    enabled: false
  
  teams:
    enabled: true
    webhook_url: "https://outlook.office.com/webhook/abc123..."
    interval: "3m"

queries:
  # Critical connection monitoring with auto-remediation
  - name: "connection_count"
    sql: "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'"
    interval: "30s"
    alert_rules:
      - condition: "gt"
        value: 150
        message: "High connection count - restarting pooler"
        category: "performance"
        to: "dba@company.com"
        channels: ["telegram"]
        execute_action: "/scripts/restart_pgbouncer.sh --graceful"
      - condition: "gt"
        value: 180
        message: "CRITICAL: Connection pool nearly exhausted"
        category: "critical"
        to: "oncall@company.com"
        channels: ["telegram", "teams"]
        execute_action: "/emergency/kill_idle_connections.py --force"
  
  # Database size monitoring with cleanup action
  - name: "database_size"
    sql: "SELECT pg_database_size(current_database())"
    interval: "1h"
    alert_rules:
      - condition: "gt"
        value: 80000000000  # 80GB
        message: "Database size warning - starting cleanup"
        category: "storage"
        to: "infrastructure@company.com"
        channels: ["teams"]
        execute_action: "/scripts/cleanup_old_partitions.py --age 30days"
      - condition: "gt"
        value: 100000000000  # 100GB
        message: "Database size critical - emergency cleanup"
        category: "critical"
        to: "infrastructure@company.com"
        channels: ["teams", "telegram"]
        execute_action: "/emergency/emergency_cleanup.sh --force --notify-management"
  
  # Performance monitoring
  - name: "slow_queries"
    sql: |
      SELECT count(*) 
      FROM pg_stat_activity 
      WHERE state = 'active' 
      AND now() - query_start > interval '5 minutes'
    interval: "1m"
    alert_rules:
      - condition: "gt"
        value: 0
        message: "Slow queries detected - investigate performance"
        category: "performance"
        to: "performance-team@company.com"
        channels: ["telegram"]
```

### Development Environment

```yaml
instance: "dev-local"

database:
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "devpass"
  database: "app_development"
  sslmode: "disable"

logging:
  file_path: "./logs/dev-monitor.log"

alerts:
  webhook:
    enabled: false
  
  telegram:
    enabled: true
    bot_token: "987654321:XYZabcDefGhiJklMnoPqRsTuVwXyZ"
    chat_id: "87654321"  # Personal dev chat
    interval: "30s"
  
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/dev-team/..."
    interval: "15s"
  
  teams:
    enabled: false

queries:
  # Basic connection monitoring
  - name: "connection_check"
    sql: "SELECT count(*) FROM pg_stat_activity"
    interval: "2m"
    alert_rules:
      - condition: "gt"
        value: 20
        message: "Dev environment has many connections"
        category: "info"
        to: "dev@company.com"
  
  # Simple size check
  - name: "db_size"
    sql: "SELECT pg_database_size(current_database())"
    interval: "6h"
    alert_rules:
      - condition: "gt"
        value: 5000000000  # 5GB
        message: "Dev database getting large"
        category: "maintenance"
        to: "dev@company.com"
```

### Staging Environment

```yaml
instance: "staging-replica"

database:
  host: "staging-db.company.com"
  port: 5432
  username: "monitor_user"
  password: "staging_password_456"
  database: "app_staging"
  sslmode: "require"

logging:
  file_path: "/var/log/postgres-stat-alert/staging.log"

alerts:
  webhook:
    enabled: true
    url: "https://staging-alerts.company.com/webhook"
    interval: "3m"
  
  telegram:
    enabled: true
    bot_token: "555666777:StagingBotTokenHere"
    chat_id: "-100987654321"  # QA team group
    interval: "2m"
  
  discord:
    enabled: true
    webhook_url: "https://discord.com/api/webhooks/staging/..."
    interval: "1m"
  
  teams:
    enabled: false

queries:
  # Test data monitoring
  - name: "test_data_volume"
    sql: |
      SELECT count(*) 
      FROM information_schema.tables 
      WHERE table_schema = 'public'
    interval: "4h"
    alert_rules:
      - condition: "gt"
        value: 100
        message: "Staging has many test tables"
        category: "maintenance"
        to: "qa-team@company.com"
  
  # Staging performance
  - name: "staging_load"
    sql: "SELECT count(*) FROM pg_stat_activity WHERE state = 'active'"
    interval: "1m"
    alert_rules:
      - condition: "gt"
        value: 50
        message: "High load on staging environment"
        category: "performance"
        to: "qa-team@company.com"
```

---

## Best Practices

### Security

#### Database User Permissions
Create a dedicated monitoring user with minimal permissions:

```sql
-- Create monitoring user
CREATE USER monitor_user WITH PASSWORD 'secure_random_password';

-- Grant necessary permissions
GRANT CONNECT ON DATABASE your_database TO monitor_user;
GRANT USAGE ON SCHEMA public TO monitor_user;
GRANT SELECT ON pg_stat_activity TO monitor_user;
GRANT SELECT ON pg_stat_database TO monitor_user;
GRANT SELECT ON pg_statio_user_tables TO monitor_user;

-- For size monitoring
GRANT EXECUTE ON FUNCTION pg_database_size(oid) TO monitor_user;
GRANT EXECUTE ON FUNCTION pg_size_pretty(bigint) TO monitor_user;
```

#### Credential Management
- Use environment variables for sensitive data:
```bash
export DB_PASSWORD="secure_password"
export TELEGRAM_TOKEN="bot_token"
```

- Or use external secret management systems
- Never commit passwords to version control

### Query Design

#### Efficient Queries
- Use specific conditions to limit result sets
- Avoid expensive operations in frequently-run queries
- Test query performance before adding to monitoring

#### Result Handling
- First column of query result is used for alert evaluation
- Ensure numeric columns for mathematical comparisons
- Use explicit column names for clarity

### Alert Strategy

#### Severity Levels
```yaml
# WARNING: Early notification
- condition: "gt"
  value: 75
  message: "Approaching threshold"
  category: "performance"
  channels: ["discord"]

# CRITICAL: Immediate action needed  
- condition: "gt"
  value: 90
  message: "Critical threshold exceeded"
  category: "critical"
  channels: ["telegram", "teams"]
```

#### Escalation
```yaml
# First level: Team notification
- condition: "gt"
  value: 80
  message: "High load detected"
  category: "performance"
  to: "dev-team@company.com"
  channels: ["discord"]

# Second level: Management notification
- condition: "gt"
  value: 95
  message: "CRITICAL: System overloaded"
  category: "critical"
  to: "management@company.com"
  channels: ["teams", "telegram"]
```

### Performance

#### Query Intervals
- Critical metrics: 30s - 1m
- Performance metrics: 1m - 5m
- Capacity metrics: 5m - 1h
- Maintenance metrics: 1h - 24h

#### Resource Usage
- Monitor the monitor: Check CPU/memory usage
- Tune intervals based on database load
- Consider query cache impact

---

## Troubleshooting

### Common Configuration Errors

#### YAML Syntax
```yaml
# ❌ WRONG: Missing quotes around time values
interval: 30s

# ✅ CORRECT: Quote time values
interval: "30s"
```

```yaml
# ❌ WRONG: Inconsistent indentation
alerts:
  telegram:
  enabled: true

# ✅ CORRECT: Consistent 2-space indentation
alerts:
  telegram:
    enabled: true
```

#### Database Connection
```yaml
# ❌ WRONG: Incorrect SSL mode
sslmode: "enabled"

# ✅ CORRECT: Valid SSL modes
sslmode: "disable"    # or "require", "verify-ca", "verify-full"
```

#### Alert Configuration
```yaml
# ❌ WRONG: Invalid condition
condition: "greater_than"

# ✅ CORRECT: Valid conditions
condition: "gt"       # or "lt", "eq", "ne", "gte", "lte"
```

### Testing Configuration

#### Validate YAML Syntax
```bash
# Using Python
python -c "import yaml; yaml.safe_load(open('config.yaml'))"

# Using yq tool
yq eval '.' config.yaml
```

#### Test Database Connection
```bash
# Using psql
psql "host=localhost port=5432 dbname=mydb user=monitor_user sslmode=disable"
```

#### Test Queries
```sql
-- Test each monitoring query manually
SELECT count(*) FROM pg_stat_activity WHERE state = 'active';
```

### Logging Issues

#### Log File Permissions
```bash
# Ensure monitor can write to log file
sudo chown monitor_user:monitor_group /var/log/postgres-stat-alert.log
sudo chmod 644 /var/log/postgres-stat-alert.log
```

#### Log Rotation
```bash
# Setup logrotate for monitor logs
sudo cat > /etc/logrotate.d/postgres-stat-alert << EOF
/var/log/postgres-stat-alert.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 monitor_user monitor_group
}
EOF
```

### Alert Issues

#### Telegram Problems
- Verify bot token with `@BotFather`
- Check chat_id using `/getUpdates` API
- Ensure bot is added to group chats

#### Discord Problems
- Verify webhook URL is active
- Check channel permissions
- Test webhook manually with curl

#### Teams Problems
- Verify connector is properly configured
- Check webhook URL format
- Test with simple message first

### Performance Issues

#### Query Timeouts
```yaml
# Reduce frequency for expensive queries
- name: "complex_analysis"
  sql: "SELECT complex_calculation()..."
  interval: "10m"  # Instead of "1m"
```

#### Memory Usage
- Monitor growth of AlertTracker maps
- Restart monitor periodically in high-volume environments
- Consider external alerting for very high-frequency monitoring

### Environment-Specific Issues

#### Development
- Use shorter intervals for faster feedback
- Enable all alert channels for testing
- Use less restrictive SSL settings

#### Production
- Use longer intervals to reduce load
- Disable development-only alert channels
- Implement proper SSL verification
- Set up log rotation and monitoring

This completes the comprehensive configuration guide for the PostgreSQL Monitor!