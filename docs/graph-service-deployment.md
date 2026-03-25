# Graph Service Deployment Guide

## Overview

The Viola graph service builds an in-memory attack graph from telemetry events and provides risk scoring for alerts.

---

## Quick Start

```bash
# 1. Start dependencies
docker run -d --name viola-kafka \
  -p 9092:9092 \
  apache/kafka:latest

# 2. Create crown jewels config
cp services/graph/config/crown_jewels.example.yaml \
   services/graph/config/crown_jewels.yaml

# Edit to define your crown jewels
vim services/graph/config/crown_jewels.yaml

# 3. Start graph service
cd services/graph
CROWN_JEWELS_CONFIG=./config/crown_jewels.yaml \
go run cmd/graph/main_v2.go
```

**Expected output:**

```
Graph manager initialized
Loaded crown jewels from ./config/crown_jewels.yaml
Graph cleanup worker started (interval: 5m0s)
Risk scoring worker started (interval: 5m0s)
Graph service consuming viola.dev.telemetry.v1.normalized
```

---

## Configuration

### Environment Variables

```bash
# Kafka
VIOLA_ENV=dev
KAFKA_BROKER=localhost:9092

# Crown Jewels
CROWN_JEWELS_CONFIG=./config/crown_jewels.yaml
```

---

## Crown Jewels Configuration

Define critical assets per tenant in YAML:

```yaml
tenants:
  tenant-abc:
    crown_jewels:
      - id: endpoint:dc-01
        reason: Domain Controller
        criticality: 100

      - id: user:admin@company.com
        reason: Global Administrator
        criticality: 95
```

**Criticality Scale:**
- 100: Mission-critical (domain controllers, prod databases)
- 90-99: High-value targets (backup servers, payment gateways)
- 80-89: Sensitive systems

---

## How It Works

```
Telemetry Events → Graph Builder → In-Memory Graph → Risk Scoring
```

### 1. Graph Building

**Consumes:** `telemetry.v1.normalized`

**Extracts Relationships:**
- `authentication_success` → auth edge (user → endpoint)
- `process_start` → spawn edge (user → endpoint)
- `network_connect` → network edge (endpoint → endpoint)

**Creates Nodes:**
- Users
- Endpoints
- Services
- Cloud resources

### 2. Risk Scoring

Runs every 5 minutes:

```
risk_score = base_criticality (0-30)
           + proximity_to_crown_jewels (0-40)
           + blast_radius_multiplier (0-20)
           + recent_activity_bonus (0-10)
```

**Range:** 0-100

---

## Performance

### Memory Usage

**Per-tenant graph (10k employees):**
- Nodes: 50k × 200 bytes = 10 MB
- Edges: 500k × 150 bytes = 75 MB
- **Total: ~100 MB**

**100 tenants:** ~10 GB

### CPU Usage

- Graph updates: O(1) per event
- Risk scoring: O(N) per tenant (every 5 min)
- BFS (path finding): O(V + E) per query

### Throughput

- Event ingestion: 10k+ EPS
- Risk score queries: <5 ms

---

## Monitoring

### Key Metrics

1. **Graph size** - Nodes/edges per tenant
2. **Risk score distribution** - How many nodes are high-risk?
3. **Edge expiration rate** - Cleanup efficiency
4. **Memory usage** - Per-tenant graph size

### Logs

```
Cleaned up expired edges: {tenant-abc: 120, tenant-xyz: 45}
Recomputed risk scores for tenant tenant-abc
```

---

## Troubleshooting

### Graph Not Building

**Symptom:** No nodes/edges created

**Debug:**
1. Check telemetry events arriving:
   ```bash
   kcat -C -b localhost:9092 -t viola.dev.telemetry.v1.normalized
   ```

2. Check event types match:
   ```
   # Supported: authentication_success, process_start, network_connect
   # Others: ignored
   ```

3. Check payload parsing:
   ```
   # Payload must be valid JSON
   ```

### Memory Usage Growing

**Symptom:** OOM or high memory usage

**Causes:**
- Too many tenants
- Large graphs (millions of edges)
- Edge TTL too long

**Solutions:**
1. Reduce edge TTL:
   ```go
   TTL: 30 * time.Minute  // Instead of 1 hour
   ```

2. Increase cleanup frequency:
   ```go
   go bldr.StartCleanupWorker(ctx, 1*time.Minute)
   ```

3. Scale horizontally (partition by tenant)

---

## Production Deployment

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: graph-service
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: graph
        image: viola/graph:latest
        env:
        - name: VIOLA_ENV
          value: prod
        - name: KAFKA_BROKER
          value: kafka.viola.svc.cluster.local:9092
        - name: CROWN_JEWELS_CONFIG
          value: /config/crown_jewels.yaml
        volumeMounts:
        - name: crown-jewels
          mountPath: /config
        resources:
          requests:
            memory: 2Gi
            cpu: 1000m
          limits:
            memory: 8Gi
            cpu: 2000m
      volumes:
      - name: crown-jewels
        configMap:
          name: crown-jewels
```

---

## Next Steps

1. **Add graph API** - Expose graph data via HTTP/gRPC
2. **Graph visualization** - UI for exploring attack paths
3. **Graph persistence** - Save/restore graphs for forensics
4. **Multi-hop attack detection** - Detect lateral movement patterns
5. **Anomaly detection** - ML-based graph change detection

---

## References

- [Attack Graph Design](./attack-graph-design.md)
- [Attack Graph Integration](./attack-graph-integration.md)
- [Detection Engine Integration](./detection-engine-deployment.md)
