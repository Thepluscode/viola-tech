# Observability Deployment Guide

## Overview

Viola's observability stack provides:
- **Prometheus** - Metrics collection and alerting
- **Grafana** - Metrics visualization and dashboards
- **OpenTelemetry Collector** - Trace collection and export
- **Jaeger** - Distributed tracing UI
- **Structured JSON Logs** - Request ID propagation across services

---

## Architecture

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Detection  │  │   Graph     │  │  Gateway    │
│   Engine    │  │   Service   │  │    API      │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       │ :9090/metrics  │ :9091/metrics  │ :8080/metrics
       │                │                │
       └────────────────┴────────────────┘
                        │
                        ▼
                ┌───────────────┐
                │  Prometheus   │
                │    :9090      │
                └───────┬───────┘
                        │
                        ▼
                ┌───────────────┐
                │    Grafana    │
                │    :3000      │
                └───────────────┘

       ┌────────────────────────────────┐
       │  OpenTelemetry Traces (gRPC)   │
       └────────────────┬───────────────┘
                        │ :4317
                        ▼
                ┌───────────────┐
                │  OTLP         │
                │  Collector    │
                └───────┬───────┘
                        │
                        ▼
                ┌───────────────┐
                │    Jaeger     │
                │    :16686     │
                └───────────────┘
```

---

## Quick Start (Docker Compose)

### 1. Create docker-compose.yml

```yaml
version: '3.8'

services:
  # Prometheus for metrics
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./observability/prometheus.yml:/etc/prometheus/prometheus.yml
      - ./observability/alerts.yml:/etc/prometheus/alerts.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.enable-lifecycle'

  # Grafana for visualization
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - ./observability/grafana-datasources.yml:/etc/grafana/provisioning/datasources/datasources.yml
      - ./observability/grafana-dashboards.yml:/etc/grafana/provisioning/dashboards/dashboards.yml
      - ./observability/dashboards:/var/lib/grafana/dashboards
      - grafana_data:/var/lib/grafana

  # OpenTelemetry Collector for traces
  otel-collector:
    image: otel/opentelemetry-collector:latest
    ports:
      - "4317:4317"  # OTLP gRPC receiver
      - "4318:4318"  # OTLP HTTP receiver
    volumes:
      - ./observability/otel-collector-config.yml:/etc/otel-collector-config.yml
    command: ["--config=/etc/otel-collector-config.yml"]

  # Jaeger for trace visualization
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "14250:14250"  # Jaeger collector gRPC
    environment:
      - COLLECTOR_OTLP_ENABLED=true

volumes:
  prometheus_data:
  grafana_data:
```

### 2. Create Prometheus Configuration

**`observability/prometheus.yml`:**

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

# Alert rules
rule_files:
  - 'alerts.yml'

# Scrape configs
scrape_configs:
  # Detection engine
  - job_name: 'detection'
    static_configs:
      - targets: ['host.docker.internal:9090']
        labels:
          service: 'detection'

  # Graph service
  - job_name: 'graph'
    static_configs:
      - targets: ['host.docker.internal:9091']
        labels:
          service: 'graph'

  # Gateway API
  - job_name: 'gateway-api'
    static_configs:
      - targets: ['host.docker.internal:8080']
        labels:
          service: 'gateway-api'

# Alertmanager configuration (optional)
# alerting:
#   alertmanagers:
#     - static_configs:
#         - targets: ['alertmanager:9093']
```

### 3. Create Alert Rules

**`observability/alerts.yml`:**

```yaml
groups:
  - name: viola_alerts
    interval: 30s
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: rate(detection_events_errors_total[5m]) > 10
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High error rate in {{ $labels.service }}"
          description: "Error rate is {{ $value }} errors/sec for tenant {{ $labels.tenant_id }}"

      # Consumer lag
      - alert: HighConsumerLag
        expr: detection_consumer_lag > 1000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High consumer lag in {{ $labels.service }}"
          description: "Consumer lag is {{ $value }} messages on {{ $labels.topic }}"

      # Alert suppression rate too low
      - alert: LowSuppressionRate
        expr: |
          (
            sum(rate(detection_alerts_suppressed_total[5m])) /
            sum(rate(detection_alerts_generated_total[5m]))
          ) < 0.4
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Alert suppression rate below target"
          description: "Current suppression rate is {{ $value | humanizePercentage }}, target is 60-80%"

      # Graph size growing too large
      - alert: GraphSizeExceeded
        expr: graph_graph_nodes{tenant_id!=""} > 100000
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Graph size exceeded for tenant {{ $labels.tenant_id }}"
          description: "Node count is {{ $value }}, consider increasing cleanup frequency"

      # API latency high
      - alert: HighAPILatency
        expr: histogram_quantile(0.95, rate(gateway_api_http_request_duration_seconds_bucket[5m])) > 1.0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High API latency on {{ $labels.endpoint }}"
          description: "P95 latency is {{ $value }}s"
```

### 4. Create OpenTelemetry Collector Config

**`observability/otel-collector-config.yml`:**

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024

exporters:
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true

  logging:
    loglevel: debug

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [jaeger, logging]
```

### 5. Create Grafana Datasource

**`observability/grafana-datasources.yml`:**

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true

  - name: Jaeger
    type: jaeger
    access: proxy
    url: http://jaeger:16686
```

### 6. Create Grafana Dashboard Provisioning

**`observability/grafana-dashboards.yml`:**

```yaml
apiVersion: 1

providers:
  - name: 'Viola Dashboards'
    orgId: 1
    folder: 'Viola'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    options:
      path: /var/lib/grafana/dashboards
```

### 7. Start Observability Stack

```bash
docker-compose up -d
```

**Access UIs:**
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)
- Jaeger: http://localhost:16686

---

## Service Configuration

### Detection Engine

**Environment Variables:**

```bash
VIOLA_ENV=dev
KAFKA_BROKER=localhost:9092
RULES_DIR=./rules
METRICS_PORT=9090           # Prometheus metrics endpoint
TRACING_ENABLED=true        # Enable OpenTelemetry tracing
OTLP_ENDPOINT=localhost:4317
```

**Start with observability:**

```bash
cd services/detection
TRACING_ENABLED=true OTLP_ENDPOINT=localhost:4317 \
go run cmd/detection/main.go
```

**Metrics exposed:**
- `http://localhost:9090/metrics` - Prometheus metrics
- `http://localhost:9090/health` - Health check

### Graph Service

**Environment Variables:**

```bash
VIOLA_ENV=dev
KAFKA_BROKER=localhost:9092
CROWN_JEWELS_CONFIG=./config/crown_jewels.yaml
METRICS_PORT=9091           # Prometheus metrics endpoint
TRACING_ENABLED=true
OTLP_ENDPOINT=localhost:4317
```

**Start with observability:**

```bash
cd services/graph
TRACING_ENABLED=true OTLP_ENDPOINT=localhost:4317 \
go run cmd/graph/main_v2.go
```

**Metrics exposed:**
- `http://localhost:9091/metrics` - Prometheus metrics
- `http://localhost:9091/health` - Health check

### Gateway API

**Environment Variables:**

```bash
VIOLA_ENV=dev
PG_HOST=localhost
PG_PORT=5432
PG_USER=postgres
PG_PASSWORD=postgres
PG_DATABASE=viola_gateway
PORT=8080
TRACING_ENABLED=true
OTLP_ENDPOINT=localhost:4317
```

**Start with observability:**

```bash
cd services/gateway-api
TRACING_ENABLED=true OTLP_ENDPOINT=localhost:4317 \
go run cmd/gateway-api/main.go
```

**Metrics exposed:**
- `http://localhost:8080/metrics` - Prometheus metrics
- `http://localhost:8080/health` - Health check
- `http://localhost:8080/ready` - Readiness check

---

## Key Metrics

### Detection Engine

| Metric | Description | Labels |
|--------|-------------|--------|
| `detection_events_processed_total` | Events processed | `tenant_id`, `event_type` |
| `detection_events_errors_total` | Processing errors | `tenant_id`, `event_type` |
| `detection_event_processing_duration_seconds` | Processing time | `tenant_id`, `event_type` |
| `detection_alerts_generated_total` | Alerts generated | `tenant_id`, `rule`, `severity` |
| `detection_alerts_suppressed_total` | Alerts suppressed by graph | `tenant_id`, `rule` |
| `detection_alert_risk_score` | Risk score distribution | `tenant_id`, `severity` |

**Critical Query - Suppression Rate:**

```promql
sum(rate(detection_alerts_suppressed_total[5m])) /
sum(rate(detection_alerts_generated_total[5m]))
```

**Target:** 0.6-0.8 (60-80% suppression)

### Graph Service

| Metric | Description | Labels |
|--------|-------------|--------|
| `graph_graph_nodes` | Current node count | `tenant_id` |
| `graph_graph_edges` | Current edge count | `tenant_id` |
| `graph_cleanups_total` | Edges cleaned up | `tenant_id` |
| `graph_risk_score_computation_duration_seconds` | Risk scoring time | `tenant_id` |
| `graph_events_processed_total` | Telemetry events processed | `tenant_id`, `event_type` |

**Critical Query - Graph Size:**

```promql
sum(graph_graph_nodes) by (tenant_id)
```

**Target:** <100,000 nodes per tenant

### Gateway API

| Metric | Description | Labels |
|--------|-------------|--------|
| `gateway_api_http_requests_total` | HTTP requests | `endpoint`, `method`, `status` |
| `gateway_api_http_request_duration_seconds` | Request latency | `endpoint`, `method` |
| `gateway_api_http_errors_total` | HTTP errors | `endpoint`, `method`, `status` |

**Critical Query - P95 Latency:**

```promql
histogram_quantile(0.95,
  rate(gateway_api_http_request_duration_seconds_bucket[5m])
) by (endpoint)
```

**Target:** <500ms

---

## Grafana Dashboards

### Viola Overview Dashboard

Create **`observability/dashboards/overview.json`** (simplified):

**Key Panels:**

1. **Alert Generation Rate**
   ```promql
   sum(rate(detection_alerts_generated_total[5m])) by (severity)
   ```

2. **Alert Suppression Rate**
   ```promql
   (
     sum(rate(detection_alerts_suppressed_total[5m])) /
     sum(rate(detection_alerts_generated_total[5m]))
   ) * 100
   ```

3. **Graph Size by Tenant**
   ```promql
   sum(graph_graph_nodes) by (tenant_id)
   ```

4. **Event Processing Rate**
   ```promql
   sum(rate(detection_events_processed_total[5m])) by (event_type)
   ```

5. **API Latency (P50, P95, P99)**
   ```promql
   histogram_quantile(0.50, rate(gateway_api_http_request_duration_seconds_bucket[5m]))
   histogram_quantile(0.95, rate(gateway_api_http_request_duration_seconds_bucket[5m]))
   histogram_quantile(0.99, rate(gateway_api_http_request_duration_seconds_bucket[5m]))
   ```

6. **Consumer Lag**
   ```promql
   sum(detection_consumer_lag) by (topic, partition)
   ```

### Detection Engine Dashboard

**Key Panels:**

1. Rules Evaluated per Second
2. Alerts by Severity (Stacked)
3. Suppression Rate Gauge (target: 60-80%)
4. Top 10 Firing Rules
5. Error Rate by Event Type

### Graph Service Dashboard

**Key Panels:**

1. Total Nodes/Edges by Tenant
2. Risk Score Computation Time
3. Cleanup Rate (edges/sec)
4. Memory Usage (if available)
5. BFS Operation Latency

---

## Distributed Tracing

### Enable Tracing

All services support OpenTelemetry tracing via environment variable:

```bash
TRACING_ENABLED=true
OTLP_ENDPOINT=localhost:4317
```

### Trace Flow

**Example: Alert Generation Trace**

```
Span: HandleNormalized (detection)
  ├─ Span: parsePayload
  ├─ Span: matchRules
  └─ Span: publishAlert
      ├─ Span: enrichWithGraph (if graph integrated)
      └─ Span: kafkaProduce
```

**View in Jaeger:**

1. Open http://localhost:16686
2. Select service: `detection`, `graph`, or `gateway-api`
3. Search for traces with tag `tenant.id`
4. View end-to-end request flow

### Key Traces to Monitor

1. **Telemetry → Detection → Alert** - End-to-end alert generation latency
2. **Graph Building** - Time to process telemetry and update graph
3. **API Request** - Gateway API → Database query latency

---

## Structured Logging

All services log in JSON format with request ID propagation:

**Example Log Entry:**

```json
{
  "timestamp": "2026-02-14T10:30:45Z",
  "level": "INFO",
  "service": "detection",
  "request_id": "req_abc123",
  "tenant_id": "tenant-xyz",
  "message": "Detection matched",
  "fields": {
    "rule_id": "001_lsass_access",
    "rule_name": "LSASS Process Access",
    "entity_id": "dc-01",
    "severity": "high"
  }
}
```

**Log Aggregation (Optional):**

Use Loki, Elasticsearch, or CloudWatch for log aggregation:

```bash
docker run -d --name=loki -p 3100:3100 grafana/loki:latest
```

Add Loki to Grafana datasources for log querying.

---

## Production Deployment (Kubernetes)

### Prometheus Operator

Use Prometheus Operator for automated service discovery:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: viola-services
spec:
  selector:
    matchLabels:
      app: viola
  endpoints:
    - port: metrics
      interval: 15s
```

### OpenTelemetry Collector (Kubernetes)

Deploy as DaemonSet or Deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: otel-collector
          image: otel/opentelemetry-collector:latest
          ports:
            - containerPort: 4317
              name: otlp-grpc
```

---

## Monitoring Best Practices

1. **Alert on Trends, Not Spikes**
   - Use `for: 5m` to avoid alert fatigue

2. **Track Suppression Rate Daily**
   - Goal: 60-80% suppression
   - Alert if < 40%

3. **Monitor Graph Growth**
   - Set alerts for graphs > 100k nodes
   - Review crown jewel proximity distribution

4. **Track P95 Latency**
   - API endpoints should be <500ms
   - Detection processing <100ms

5. **Log Request IDs**
   - Correlate logs across services
   - Debug end-to-end flows

---

## Troubleshooting

### Metrics Not Appearing

1. Check service is running:
   ```bash
   curl http://localhost:9090/health
   ```

2. Check metrics endpoint:
   ```bash
   curl http://localhost:9090/metrics
   ```

3. Check Prometheus scrape targets:
   - Visit http://localhost:9090/targets
   - Ensure all services are "UP"

### Traces Not Showing in Jaeger

1. Verify tracing enabled:
   ```bash
   echo $TRACING_ENABLED
   ```

2. Check OTLP Collector logs:
   ```bash
   docker logs otel-collector
   ```

3. Test OTLP endpoint:
   ```bash
   nc -zv localhost 4317
   ```

### High Cardinality Issues

**Problem:** Too many unique label combinations

**Solution:**
- Avoid high-cardinality labels (e.g., `alert_id`, `request_id`)
- Use `tenant_id`, `rule`, `severity` only
- Aggregate before recording metrics

---

## Next Steps

1. **Set up Alertmanager** - Route alerts to PagerDuty, Slack
2. **Create SLI/SLO dashboards** - Track 99.9% uptime
3. **Add custom business metrics** - Cost per alert, MTTD, MTTR
4. **Integrate with APM** - DataDog, New Relic for deeper insights

---

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Dashboards](https://grafana.com/docs/)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
