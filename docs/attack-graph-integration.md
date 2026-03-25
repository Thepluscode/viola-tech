# Attack Graph Integration Guide

## Overview

This guide shows how to integrate the attack graph into the detection engine to enrich alerts with graph-based risk scoring.

---

## Integration Architecture

### Option 1: In-Process (MVP - Recommended)

Detection engine imports graph package directly:

```
┌────────────────────────────────┐
│   Detection Engine Process     │
│                                 │
│  ┌──────────┐   ┌───────────┐ │
│  │ Rule     │   │ Graph     │ │
│  │ Engine   │──>│ Manager   │ │
│  └──────────┘   └───────────┘ │
│                                 │
│  Alert + Graph Context          │
└────────────────────────────────┘
```

**Pros:**
- Simple (no network calls)
- Low latency (<1ms)
- No additional service

**Cons:**
- Higher memory usage (graph in detection process)
- Less separation of concerns

---

### Option 2: Separate Service (Future)

Detection engine calls graph service via gRPC/HTTP:

```
┌──────────────┐         ┌──────────────┐
│  Detection   │         │  Graph       │
│  Engine      │──HTTP──>│  Service     │
│              │<────────│              │
└──────────────┘         └──────────────┘
   Alert Request            Graph Context
```

**Pros:**
- Separation of concerns
- Can scale independently
- Graph service shared across multiple consumers

**Cons:**
- Network latency
- More complex deployment

**For MVP:** Use Option 1 (in-process)

---

## In-Process Integration (MVP)

### 1. Update Detection Engine

Modify `services/detection/internal/engine/engine_v2.go`:

```go
package engine

import (
    // ... existing imports
    "github.com/viola/graph/internal/store"
)

type EngineV2 struct {
    rules          []*rule.Rule
    tracker        *rule.ThresholdTracker
    topics         sharedkafka.Topics
    hitProd        *sharedkafka.Producer
    alertProd      *sharedkafka.Producer
    partitionStrat *sharedkafka.PartitionKeyStrategy

    // NEW: Graph manager for risk scoring
    graphManager   *store.GraphManager
}

func NewV2(cfg ConfigV2) (*EngineV2, error) {
    // ... existing setup

    // NEW: Initialize graph manager
    graphManager := store.NewGraphManager()

    // Load crown jewels (optional)
    if cfg.CrownJewelsPath != "" {
        // Load and apply crown jewels config
    }

    return &EngineV2{
        // ... existing fields
        graphManager: graphManager,
    }, nil
}
```

### 2. Enrich Alerts with Graph Context

```go
func (e *EngineV2) publishAlert(ctx context.Context, r *rule.Rule, ev *telemetryv1.EventEnvelope, requestID string) error {
    now := time.Now().UTC()
    groupID := correlation.GroupID(ev.TenantId, r.ID, ev.EntityId, now, correlation.Bucket15m)

    alert := &securityv1.Alert{
        TenantId:          ev.TenantId,
        AlertId:           id.New(),
        CreatedAt:         now.Format(time.RFC3339),
        UpdatedAt:         now.Format(time.RFC3339),
        Status:            "open",
        Severity:          r.Severity,
        Confidence:        r.Confidence,
        RiskScore:         calculateRiskScore(r), // OLD: Basic risk score
        Title:             r.Name,
        Description:       r.Description,
        EntityIds:         []string{ev.EntityId},
        DetectionHitIds:   []string{},
        MitreTactic:       mitreTactic(r),
        MitreTechnique:    mitreTechnique(r),
        Labels:            convertTags(r.Tags),
        RequestId:         requestID,
        CorrelatedGroupId: groupID,
    }

    // NEW: Enrich with graph context
    e.enrichAlertWithGraph(alert)

    // ... rest of publish logic
}

func (e *EngineV2) enrichAlertWithGraph(alert *securityv1.Alert) {
    if e.graphManager == nil {
        return // Graph not available
    }

    // Get graph for tenant
    graph := e.graphManager.GetGraph(alert.TenantId)
    if graph == nil {
        return // No graph for this tenant
    }

    // For each affected entity, get graph context
    for _, entityID := range alert.EntityIds {
        nodeID := fmt.Sprintf("endpoint:%s", entityID)
        node := graph.GetNode(nodeID)
        if node == nil {
            continue // Node not in graph
        }

        // Enrich alert with graph-based risk score
        graphRiskScore := node.RiskScore
        alert.RiskScore = combineRiskScores(alert.RiskScore, graphRiskScore)

        // Add graph metadata to labels
        alert.Labels["graph_risk_score"] = fmt.Sprintf("%.2f", graphRiskScore)
        alert.Labels["crown_distance"] = fmt.Sprintf("%d", node.CrownDistance)
        alert.Labels["blast_radius"] = fmt.Sprintf("%d", node.BlastRadius)

        // If close to crown jewel, escalate severity
        if node.CrownDistance >= 0 && node.CrownDistance <= 2 {
            alert.Labels["crown_jewel_proximity"] = "high"
            // Optionally escalate severity
            if alert.Severity == "med" {
                alert.Severity = "high"
            }
        }
    }
}

func combineRiskScores(detectionScore, graphScore float64) float64 {
    // Weighted average: 40% detection, 60% graph
    return (detectionScore * 0.4) + (graphScore * 0.6)
}
```

### 3. Alert Suppression Based on Graph

```go
func (e *EngineV2) shouldSuppressAlert(alert *securityv1.Alert) bool {
    if e.graphManager == nil {
        return false // No graph, don't suppress
    }

    graph := e.graphManager.GetGraph(alert.TenantId)
    if graph == nil {
        return false
    }

    for _, entityID := range alert.EntityIds {
        nodeID := fmt.Sprintf("endpoint:%s", entityID)
        node := graph.GetNode(nodeID)
        if node == nil {
            continue
        }

        // Suppression rules:

        // 1. Suppress low-risk alerts on isolated endpoints
        if node.RiskScore < 30 && node.CrownDistance == -1 {
            log.Printf("Suppressing alert %s: low risk (%.2f), isolated from crown jewels",
                alert.AlertId, node.RiskScore)
            return true
        }

        // 2. Suppress if blast radius is tiny (< 3 nodes)
        if node.BlastRadius < 3 {
            log.Printf("Suppressing alert %s: minimal blast radius (%d)",
                alert.AlertId, node.BlastRadius)
            return true
        }
    }

    return false
}
```

### 4. Update Alert Publishing Logic

```go
func (e *EngineV2) publishAlert(ctx context.Context, r *rule.Rule, ev *telemetryv1.EventEnvelope, requestID string) error {
    // ... create alert

    // Enrich with graph
    e.enrichAlertWithGraph(alert)

    // Check if should suppress
    if e.shouldSuppressAlert(alert) {
        // Log suppressed alert (for metrics)
        log.Printf("SUPPRESSED: alert=%s rule=%s entity=%s risk=%.2f",
            alert.AlertId, r.ID, ev.EntityId, alert.RiskScore)
        return nil // Don't publish
    }

    // Publish alert (only if not suppressed)
    // ... rest of publish logic
}
```

---

## Running Detection Engine with Graph

### 1. Start Graph Service (Separate Process)

```bash
cd services/graph
go run cmd/graph/main_v2.go
```

This builds the graph from telemetry.

### 2. Start Detection Engine (with Graph Manager)

```bash
cd services/detection
# Set crown jewels config
CROWN_JEWELS_CONFIG=../../services/graph/config/crown_jewels.yaml \
go run cmd/detection/main.go
```

**Note:** For in-process integration, detection engine also runs a graph manager. Both services build graphs independently (future: shared graph service via gRPC).

---

## Testing Alert Enrichment

### 1. Send Telemetry (Build Graph)

```bash
# Send authentication event (creates auth edge)
kcat -P -b localhost:9092 -t viola.dev.telemetry.v1.normalized <<EOF
{
  "tenant_id": "test-tenant",
  "entity_id": "dc-01",
  "observed_at": "2026-02-14T12:00:00Z",
  "received_at": "2026-02-14T12:00:01Z",
  "event_type": "authentication_success",
  "source": "endpoint",
  "payload": "{\"user\":\"alice\",\"method\":\"password\"}"
}
EOF
```

### 2. Trigger Detection

```bash
# Send suspicious process start
kcat -P -b localhost:9092 -t viola.dev.telemetry.v1.normalized <<EOF
{
  "tenant_id": "test-tenant",
  "entity_id": "dc-01",
  "observed_at": "2026-02-14T12:01:00Z",
  "received_at": "2026-02-14T12:01:01Z",
  "event_type": "process_start",
  "source": "endpoint",
  "payload": "{\"process_name\":\"powershell.exe\",\"cmdline\":\"powershell.exe -EncodedCommand ABC123\",\"user\":\"alice\"}"
}
EOF
```

### 3. Check Alert (Should Be Escalated)

```bash
kcat -C -b localhost:9092 -t viola.dev.security.alert.v1.created
```

**Expected:**
- Alert severity: `high` (escalated due to crown jewel proximity)
- Labels include:
  - `graph_risk_score: 95.00` (high because dc-01 is a crown jewel)
  - `crown_distance: 0` (IS a crown jewel)
  - `blast_radius: X`
  - `crown_jewel_proximity: high`

---

## Alert Suppression Metrics

Track suppression rates to measure value:

```go
type SuppressionMetrics struct {
    TotalAlerts      int
    SuppressedAlerts int
    SuppressionRate  float64
}

func (e *EngineV2) logSuppressionMetrics() {
    rate := float64(suppressedCount) / float64(totalCount) * 100
    log.Printf("Alert suppression rate: %.1f%% (%d/%d)", rate, suppressedCount, totalCount)
}
```

**Target:** 60-80% suppression rate

---

## Performance Considerations

### In-Process Graph Impact

**Memory:**
- Per-tenant graph: ~100 MB (10k employees)
- 100 tenants: 10 GB

**CPU:**
- Graph enrichment: <1 ms per alert
- Risk score computation: runs every 5 minutes (background)

**Recommendation:** For >100 tenants, move to separate graph service (Option 2).

---

## Future: Separate Graph Service (gRPC)

```proto
// graph_service.proto
service GraphService {
  rpc GetNodeRiskScore(GetNodeRequest) returns (NodeRiskScoreResponse);
  rpc EnrichAlert(EnrichAlertRequest) returns (EnrichAlertResponse);
}

message GetNodeRequest {
  string tenant_id = 1;
  string node_id = 2;
}

message NodeRiskScoreResponse {
  double risk_score = 1;
  int32 crown_distance = 2;
  int32 blast_radius = 3;
}
```

**Benefits:**
- Centralized graph (no duplication)
- Independent scaling
- Shared across detection, response, threat hunting

---

## Next Steps

1. **Measure alert reduction** - Deploy with suppression enabled, measure reduction rate
2. **Tune suppression thresholds** - Adjust risk_score < 30 based on false negatives
3. **Add graph API** - Expose graph data for UI visualization
4. **Add graph-based threat hunting** - Query language for exploring attack paths

---

## References

- [Attack Graph Design](./attack-graph-design.md)
- [Detection Engine Deployment](./detection-engine-deployment.md)
- [Graph Service Deployment](./graph-service-deployment.md) (to be created)
