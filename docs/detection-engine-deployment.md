# Detection Engine Deployment Guide

## Overview

The Viola detection engine is a rule-based system that processes normalized telemetry events and generates alerts when suspicious patterns are detected.

**Architecture:**

```
Kafka: telemetry.v1.normalized
           |
           v
   Detection Engine (Go)
    - Load rules from ./rules
    - Match events against rules
    - Track thresholds (stateful)
           |
           v
  Publish DetectionHit + Alert
           |
           v
Kafka: security.detection.v1.hit
Kafka: security.alert.v1.created
```

---

## Configuration

### Environment Variables

```bash
# Kafka
VIOLA_ENV=dev                       # prod|staging|dev
KAFKA_BROKER=localhost:9092

# Rules
RULES_DIR=./rules                   # Path to detection rules directory
```

---

## Running Locally

### 1. Start Kafka

```bash
docker run -d \
  --name viola-kafka \
  -p 9092:9092 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  apache/kafka:latest
```

### 2. Create Rules Directory

```bash
cd services/detection
mkdir -p rules
cp rules/*.yaml rules/  # Copy example rules
```

### 3. Start Detection Engine

```bash
cd services/detection
go run cmd/detection/main.go
```

**Expected output:**

```
Loaded 10 detection rules
detection consuming viola.dev.telemetry.v1.normalized -> producing viola.dev.security.detection.v1.hit + viola.dev.security.alert.v1.created
```

---

## Testing

### Send Test Telemetry Event

```bash
# Install kcat (formerly kafkacat)
brew install kcat

# Send test event
echo '{
  "tenant_id": "test-tenant",
  "entity_id": "host-123",
  "observed_at": "2026-02-14T12:00:00Z",
  "received_at": "2026-02-14T12:00:01Z",
  "event_type": "process_start",
  "source": "endpoint",
  "payload": "{\"process_name\":\"powershell.exe\",\"cmdline\":\"powershell.exe -EncodedCommand ABC123\",\"parent_process_name\":\"explorer.exe\",\"user\":\"alice\"}"
}' | kcat -P -b localhost:9092 -t viola.dev.telemetry.v1.normalized
```

### Verify Detection Hit

```bash
# Consume detection hits
kcat -C -b localhost:9092 -t viola.dev.security.detection.v1.hit
```

**Expected:** DetectionHit for rule `viola:exec_powershell_encoded`

### Verify Alert

```bash
# Consume alerts
kcat -C -b localhost:9092 -t viola.dev.security.alert.v1.created
```

**Expected:** Alert with severity "high"

---

## Production Deployment

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: detection-engine
  namespace: viola
spec:
  replicas: 3  # Scale for high throughput
  selector:
    matchLabels:
      app: detection-engine
  template:
    metadata:
      labels:
        app: detection-engine
    spec:
      containers:
      - name: detection
        image: viola/detection:latest
        env:
        - name: VIOLA_ENV
          value: prod
        - name: KAFKA_BROKER
          value: kafka.viola.svc.cluster.local:9092
        - name: RULES_DIR
          value: /rules
        volumeMounts:
        - name: rules
          mountPath: /rules
          readOnly: true
        resources:
          requests:
            cpu: 1000m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 1Gi
      volumes:
      - name: rules
        configMap:
          name: detection-rules
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: detection-rules
  namespace: viola
data:
  001_lsass_access.yaml: |
    # Rule content here
  002_psexec_lateral_movement.yaml: |
    # Rule content here
  # ... all rules
```

---

## Scaling

### Horizontal Scaling

The detection engine is **stateless** (except threshold tracking, which is per-instance).

**Recommended:**
- Run 2-5 instances for high availability
- Each instance processes a partition of the Kafka topic
- Kafka consumer group ensures no duplicate processing

**Threshold tracking caveat:**
- Each instance tracks thresholds independently
- For distributed threshold tracking, use Redis (future enhancement)

### Performance Tuning

**Kafka consumer config:**

```go
ConsumerConfig{
    Brokers:     brokers,
    Topic:       topics.TelemetryNormalized,
    GroupID:     "viola.prod.detection.scorer",
    MinBytes:    1024,        // Batch reads for efficiency
    MaxBytes:    10485760,    // 10MB max batch
    MaxWait:     100ms,       // Wait for batch to fill
}
```

**Rule optimization:**
- Use string matching over regex when possible
- Limit rules to <50 per event type
- Profile rule evaluation time (`go test -bench`)

---

## Monitoring

### Metrics to Track

1. **Events processed/sec** - Throughput
2. **Detection hits/sec** - Detection rate
3. **Alerts created/sec** - Alert rate
4. **Consumer lag** - Backlog size
5. **Rule evaluation time** - Performance
6. **DLQ message count** - Error rate

### Prometheus Metrics (Future)

```
detection_events_total{event_type}
detection_hits_total{rule_id,severity}
detection_alerts_total{severity}
detection_rule_evaluation_seconds{rule_id}
detection_consumer_lag
```

---

## Troubleshooting

### No Alerts Generating

**Symptom:** Telemetry flowing, but no detections

**Debug steps:**

1. Check rules loaded:
   ```
   # Should see "Loaded N detection rules" in logs
   ```

2. Verify event types match:
   ```
   # Rule expects "process_start"
   # Telemetry has "process.start" (mismatch)
   ```

3. Check field names:
   ```
   # Rule: "process_name"
   # Event: "name" (mismatch)
   ```

4. Add debug logging to `engine_v2.go`:
   ```go
   log.Printf("Evaluating rule %s against event %s", r.ID, ruleEvent.EventType)
   ```

### High False Positive Rate

**Solutions:**

1. **Tune confidence scores:**
   ```yaml
   confidence: 0.75  # Lower = more FPs, higher = more FNs
   ```

2. **Add suppression conditions:**
   ```yaml
   suppress_if:
     - field: process_path
       operator: startswith
       value: C:\Windows\System32\
   ```

3. **Increase threshold:**
   ```yaml
   threshold:
     count: 10  # Increase from 5
     window: 5m
   ```

### Consumer Lag Growing

**Symptom:** Kafka consumer lag increasing

**Causes:**
- Insufficient processing capacity
- Slow rule evaluation
- Kafka producer backpressure

**Solutions:**

1. **Scale horizontally:**
   - Add more detection engine instances
   - Increase Kafka topic partitions

2. **Optimize rules:**
   - Remove slow regex patterns
   - Reduce number of rules
   - Profile with benchmarks

3. **Increase batch size:**
   ```go
   MaxBytes: 10485760,  // Larger batches
   ```

### Memory Usage Growing

**Symptom:** Detection engine OOM

**Cause:** Threshold tracker accumulating state

**Fix:** The threshold tracker auto-cleans every 5 minutes, but for high-cardinality groups (many users/entities), consider:

1. **Lower threshold window:**
   ```yaml
   window: 1m  # Instead of 5m
   ```

2. **Add memory limits:**
   ```yaml
   resources:
     limits:
       memory: 1Gi
   ```

3. **Monitor memory:**
   ```bash
   kubectl top pod detection-engine-xxx
   ```

---

## Rule Management

### Adding New Rules

1. Create rule file in `rules/` directory
2. Commit to git
3. Deploy via ConfigMap or volume mount
4. Restart detection engine (or implement hot-reload)

### Hot-Reload (Future)

Watch rules directory for changes and reload without downtime:

```go
func (e *EngineV2) WatchRules(ctx context.Context) {
    watcher, _ := fsnotify.NewWatcher()
    watcher.Add(e.rulesDir)

    for {
        select {
        case event := <-watcher.Events:
            if event.Op&fsnotify.Write == fsnotify.Write {
                e.reloadRules()
            }
        }
    }
}
```

### Rule Versioning

Use git tags for rule versioning:

```bash
git tag -a detection-rules-v1.0.0 -m "Initial 10 rules"
git push origin detection-rules-v1.0.0
```

---

## Security Considerations

### 1. Rule Injection

**Risk:** Malicious actor modifies rules to disable detections

**Mitigation:**
- Store rules in read-only ConfigMap
- Use RBAC to restrict ConfigMap edits
- Audit all rule changes

### 2. DLQ Poisoning

**Risk:** Malformed events cause infinite retry loops

**Mitigation:**
- DLQ auto-publishes after retries
- Monitor DLQ size with alerts

### 3. Resource Exhaustion

**Risk:** Attacker floods telemetry to overwhelm detection engine

**Mitigation:**
- Rate limit telemetry ingestion
- Set memory limits on detection pods
- Monitor consumer lag and scale proactively

---

## Next Steps

1. **Add observability** - Prometheus metrics, OpenTelemetry tracing
2. **Add Linux rules** - Currently Windows-focused
3. **Add cloud rules** - AWS/Azure suspicious API calls
4. **Implement hot-reload** - Update rules without restart
5. **Add rule testing framework** - CI/CD validation
6. **Distributed threshold tracking** - Use Redis for cross-instance thresholds
7. **ML-based anomaly detection** - Complement rule-based detections

---

## References

- [Rule Schema Documentation](../services/detection/rules/schema.md)
- [Available Rules](../services/detection/rules/README.md)
- [MITRE ATT&CK Framework](https://attack.mitre.org/)
