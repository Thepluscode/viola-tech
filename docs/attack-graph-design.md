# Attack Graph Design

## Overview

The Viola attack graph is an **in-memory, per-tenant graph** that models relationships between entities in a security environment. It enables contextual risk scoring by analyzing:

- **Reachability** to critical assets (crown jewels)
- **Blast radius** from compromised entities
- **Attack paths** available to adversaries

**Goal:** Reduce alert noise by 60-80% through graph-based risk scoring.

---

## Architecture

```
Telemetry Events                Graph Builder                  In-Memory Graph
┌──────────────┐               ┌──────────────┐              ┌─────────────────┐
│ process_start│──────────────>│ Extract      │─────────────>│ Node: Endpoint  │
│ network_conn │──────────────>│ relationships│─────────────>│ Edge: Auth      │
│ auth_success │──────────────>│              │              │ Edge: NetConn   │
└──────────────┘               └──────────────┘              └─────────────────┘
                                                                      │
Identity Events                                                       │
┌──────────────┐                                                     │
│ user_created │──────────────────────────────────────────────────>  │
│ role_assigned│──────────────────────────────────────────────────>  │
└──────────────┘                                                     │
                                                                      │
Crown Jewel Config                                                   │
┌──────────────┐                                                     │
│ crown_jewels │──────────────────────────────────────────────────>  │
│ - domain-ctrl│                                                     │
│ - db-prod    │                                                     │
└──────────────┘                                                     │
                                                                      v
                                                            Risk Scoring Engine
                                                            ┌────────────────────┐
                                                            │ BFS to crown jewels│
                                                            │ Blast radius calc  │
                                                            │ Path scoring       │
                                                            └────────────────────┘
                                                                      │
                                                                      v
                                                           Alert Enrichment
                                                           ┌────────────────┐
                                                           │ risk_score     │
                                                           │ crown_distance │
                                                           │ blast_radius   │
                                                           └────────────────┘
```

---

## Graph Data Model

### Node Types

| Type           | ID Format              | Description                  | Properties                        |
|----------------|------------------------|------------------------------|-----------------------------------|
| `user`         | `user:<user_id>`       | Human user account           | email, name, roles, criticality   |
| `service`      | `service:<service_id>` | Service/machine account      | name, type, criticality           |
| `endpoint`     | `endpoint:<entity_id>` | Workstation, server          | hostname, os, ip, criticality     |
| `cloud`        | `cloud:<resource_id>`  | Cloud resource (VM, DB, etc.)| provider, type, region, critical  |
| `network_seg`  | `network:<segment_id>` | Network segment/VLAN         | cidr, zone, criticality           |

### Edge Types

| Type           | Direction | Description                           | Properties                  |
|----------------|-----------|---------------------------------------|-----------------------------|
| `auth`         | user → endpoint | User authenticated to endpoint   | timestamp, method           |
| `network`      | endpoint → endpoint | Network connection           | port, protocol, timestamp   |
| `role`         | user → role | User assigned to role                 | timestamp, scope            |
| `access`       | user/service → cloud | Resource access              | operation, timestamp        |
| `spawn`        | user → endpoint | Process spawned on endpoint          | process_name, timestamp     |

### Node Properties

```go
type Node struct {
    ID           string            // e.g., "user:alice", "endpoint:host-123"
    Type         NodeType          // user|service|endpoint|cloud|network_seg
    Labels       map[string]string // org_unit, zone, tags
    Criticality  int               // 0-100 (crown jewel = 100)
    FirstSeen    time.Time
    LastSeen     time.Time

    // Computed properties (cached)
    RiskScore       float64  // 0-100
    CrownDistance   int      // hops to nearest crown jewel (-1 if unreachable)
    BlastRadius     int      // number of reachable nodes
}
```

### Edge Properties

```go
type Edge struct {
    ID        string    // e.g., "auth:alice:host-123"
    Type      EdgeType  // auth|network|role|access|spawn
    Source    string    // source node ID
    Target    string    // target node ID
    Weight    float64   // trust/confidence score (0-1)
    Timestamp time.Time // when relationship observed
    TTL       time.Duration // how long before edge expires

    // Type-specific metadata
    Metadata map[string]string
}
```

---

## Graph Operations

### 1. Add Node

```go
func (g *Graph) AddNode(node *Node) error
```

**Behavior:**
- Upsert (create if new, update LastSeen if exists)
- Update criticality if changed
- Invalidate cached risk scores

---

### 2. Add Edge

```go
func (g *Graph) AddEdge(edge *Edge) error
```

**Behavior:**
- Upsert edge
- Update timestamp
- If edge already exists, refresh TTL
- Invalidate cached risk scores for affected nodes

---

### 3. Find Path (BFS)

```go
func (g *Graph) FindPath(from, to string) []string
```

**Behavior:**
- Breadth-first search from `from` to `to`
- Returns shortest path (list of node IDs)
- Returns nil if no path exists

---

### 4. Calculate Blast Radius

```go
func (g *Graph) BlastRadius(nodeID string, maxDepth int) int
```

**Behavior:**
- BFS from `nodeID` up to `maxDepth` hops
- Count all reachable nodes
- Used to assess impact of compromise

---

### 5. Compute Crown Distance

```go
func (g *Graph) CrownDistance(nodeID string) int
```

**Behavior:**
- BFS to nearest crown jewel node
- Returns hop count (-1 if unreachable)
- Cached per node, recomputed on graph changes

---

### 6. Compute Risk Score

```go
func (g *Graph) ComputeRiskScore(nodeID string) float64
```

**Formula:**

```
risk_score = base_criticality
           + proximity_penalty
           + blast_radius_multiplier
           + recent_activity_bonus

where:
  proximity_penalty = (1 - crown_distance / max_distance) * 40
  blast_radius_multiplier = (blast_radius / total_nodes) * 30
  recent_activity_bonus = if last_seen < 5 min: +10, else 0
```

**Range:** 0-100

**Interpretation:**
- 0-30: Low risk (isolated, non-critical)
- 31-60: Medium risk
- 61-85: High risk (path to crown jewels)
- 86-100: Critical risk (crown jewel or 1-hop away)

---

## Graph Refresh Strategy

### Problem: Stale Edges

**Challenge:** Edges represent observed relationships, but they don't necessarily persist.

**Example:**
- User `alice` authenticated to `host-123` at 10:00 AM
- At 10:05 AM, user logged out
- Edge `auth:alice:host-123` is now stale

**Solution: TTL-based Edge Expiration**

```go
type Edge struct {
    TTL time.Duration  // e.g., 1 hour
    CreatedAt time.Time
}

func (e *Edge) IsExpired() bool {
    return time.Since(e.CreatedAt) > e.TTL
}
```

**Cleanup Process:**
- Run every 5 minutes
- Remove edges where `IsExpired() == true`
- Recompute risk scores

---

## Graph Rebuild Strategy

### Option 1: Event-Driven (Real-Time)

**Pros:**
- Always up-to-date
- Low latency

**Cons:**
- Complex (must handle out-of-order events)
- State management overhead

---

### Option 2: Periodic Rebuild (Snapshot)

**Pros:**
- Simple (rebuild from scratch)
- Easier to reason about

**Cons:**
- Stale data between rebuilds

**Recommendation for MVP:** **Option 2 - Rebuild every 5 minutes**

**Process:**
1. Query last 24 hours of telemetry from ClickHouse
2. Build graph from scratch
3. Swap in-memory graphs atomically
4. Old graph garbage collected

---

## Crown Jewel Configuration

### Static Configuration (MVP)

```yaml
# crown_jewels.yaml
tenants:
  tenant-abc:
    crown_jewels:
      - id: endpoint:dc-01
        reason: Domain Controller
      - id: endpoint:db-prod-01
        reason: Production Database
      - id: cloud:s3-bucket-secrets
        reason: Secrets Storage
      - id: user:admin@company.com
        reason: Global Admin
```

### Dynamic Detection (Future)

Automatically identify crown jewels based on:
- High inbound connections (central services)
- Privileged role assignments
- Access to sensitive data

---

## Integration with Detection Engine

### Alert Enrichment

When a detection fires:

```go
func (e *EngineV2) enrichAlert(alert *Alert, graph *Graph) {
    // Get affected entities
    for _, entityID := range alert.EntityIDs {
        node := graph.GetNode(entityID)
        if node == nil {
            continue
        }

        // Enrich alert with graph context
        alert.GraphRiskScore = node.RiskScore
        alert.CrownDistance = node.CrownDistance
        alert.BlastRadius = node.BlastRadius

        // Find path to nearest crown jewel
        if node.CrownDistance > 0 {
            path := graph.PathToNearestCrown(entityID)
            alert.CrownJewelPath = path
        }
    }
}
```

### Alert Suppression Logic

```go
func shouldSuppress(alert *Alert) bool {
    // Suppress low-risk alerts on isolated endpoints
    if alert.GraphRiskScore < 30 && alert.CrownDistance == -1 {
        return true
    }

    // Suppress if blast radius is tiny (< 5 nodes)
    if alert.BlastRadius < 5 {
        return true
    }

    return false
}
```

**Expected impact:** 60-80% alert reduction

---

## Performance Targets

### Graph Size Estimates

**Medium Organization (1,000 employees):**
- Nodes: ~5,000 (users, endpoints, services, cloud resources)
- Edges: ~50,000 (auth, network, role relationships)

**Large Organization (10,000 employees):**
- Nodes: ~50,000
- Edges: ~500,000

### Performance Requirements

| Operation           | Target Latency | Notes                          |
|---------------------|----------------|--------------------------------|
| Add Node            | < 1 ms         | O(1) hash map insert           |
| Add Edge            | < 1 ms         | O(1) hash map insert           |
| Find Path (BFS)     | < 10 ms        | Max 10 hops                    |
| Blast Radius        | < 50 ms        | BFS up to 5 hops               |
| Crown Distance      | < 10 ms        | Cached, computed once          |
| Risk Score          | < 5 ms         | Cached, invalidated on changes |
| Full Graph Rebuild  | < 30 sec       | 50k nodes, 500k edges          |

### Memory Estimates

**Per-tenant graph (10k employees):**
- Nodes: 50k × 200 bytes = 10 MB
- Edges: 500k × 150 bytes = 75 MB
- **Total: ~100 MB per tenant**

**100 tenants:** 10 GB (fits in memory on modern servers)

---

## Implementation Plan

### Phase 1: Core Graph (Week 1)

- [x] Design data model
- [ ] Implement in-memory graph store
- [ ] Implement basic operations (AddNode, AddEdge, GetNode)
- [ ] Unit tests

### Phase 2: Graph Builder (Week 2)

- [ ] Consume telemetry events
- [ ] Extract relationships (auth, network, spawn)
- [ ] Build graph incrementally
- [ ] Periodic rebuild (every 5 minutes)

### Phase 3: Risk Scoring (Week 2-3)

- [ ] Crown jewel configuration
- [ ] BFS algorithms (path finding, blast radius)
- [ ] Risk score calculation
- [ ] Caching and invalidation

### Phase 4: Integration (Week 3)

- [ ] Enrich alerts with graph context
- [ ] Alert suppression logic
- [ ] Graph API endpoints (for UI visualization)
- [ ] Metrics and monitoring

---

## Future Enhancements

1. **Multi-hop attack path detection** - Detect when attacker is progressing toward crown jewels
2. **Graph anomaly detection** - ML-based detection of unusual graph changes
3. **Graph persistence** - Save/restore graphs (for forensic analysis)
4. **Graph visualization** - Interactive UI for exploring attack paths
5. **Cross-tenant graph analysis** - Detect similar attack patterns across tenants
6. **Graph-based threat hunting** - Query language for graph exploration

---

## References

- [BloodHound](https://github.com/BloodHoundAD/BloodHound) - AD attack path analysis
- [Graphistry](https://www.graphistry.com/) - Graph visualization
- [Neo4j Graph Algorithms](https://neo4j.com/docs/graph-algorithms/)
