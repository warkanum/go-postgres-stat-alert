# Usefull Examples


## High Number of LWLocks

LWLocks happens when the server fails to keep up with a lot of transactions

```yaml
  - name: "High Number of LWLocks"
    sql: |
      select count(1)
      from pg_stat_activity a
      where a.wait_event_type = 'LWLock'
    interval: "5s"
    alert_rules:
      - condition: "gt"
        value: "10"
        message: "A High Number of LW Locks are detected. Check the server"
        category: "performance"

```

## Offline Replication slots

Check for offline replication slots

```yaml
  - name: "replication_offline"
    sql: |
      select nv(string_agg(s.slot_name,', ')) as offline_slots
      from pg_replication_slots s
      where not s.active
    interval: "30s"
    alert_rules:
      - condition: "ne"
        value: ""
        message: "Offline Replication slots"
        category: "replication"


```


## Replication lag

Check if the replication lag grows to much

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
        value: 60  # 60 seconds lag
        message: "Replication lag is high"
        category: "replication"


```

## Long Running Queries

Alert when queries are running for a very long time. This might keep auto vacuum from running properly

```yaml
  - name: "long_running_queries"
    sql: "SELECT count(*) FROM pg_stat_activity WHERE state = 'active' AND now() - query_start > interval '90 minutes' and query not ilike '%replication%'"
    interval: "1m"
    alert_rules:
      - condition: "gt"
        value: 0
        message: "Long running queries detected (> 90 minutes)"
        category: "performance"

```