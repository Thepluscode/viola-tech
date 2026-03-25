# Runbook: Graph Update Lag High

**Alert:** `GraphUpdateLagHigh`
**Threshold:** Kafka consumer lag > 5,000 messages
**Severity:** Warning

## Symptoms

- Attack graph data is stale
- Risk scores not updating in real-time
- Graph-based detections may miss recent lateral movement

## Investigation

### 1. Check graph service pods

```bash
kubectl get pods -l app=viola-graph -n viola
kubectl top pods -l app=viola-graph -n viola
```

### 2. Check consumer group lag details

```bash
kafka-consumer-groups.sh --bootstrap-server $KAFKA_BROKER \
  --describe --group graph-service
```

### 3. Check graph service logs

```bash
kubectl logs -n viola deploy/viola-graph --tail=100
```

Look for: slow graph operations, memory pressure, serialization errors.

### 4. Check graph size

A very large graph (>100k nodes) can slow down path computations.

```bash
curl -s localhost:9091/metrics | grep graph_nodes_total
```

## Remediation

| Cause | Action |
|-------|--------|
| Graph too large | Prune old edges, increase TTL for edge expiry |
| CPU bound | Scale graph replicas, increase CPU limits |
| Memory pressure | Increase memory limits, optimize data structures |
| Slow path computation | Disable expensive algorithms temporarily |
| Burst of edge events | Temporary — lag should recover |

## Escalation

If graph is >30 minutes stale, attack path analysis is unreliable. Notify security operations.
