# Runbook: Detection Latency High

**Alert:** `DetectionLatencyHigh` / `DetectionLatencyCritical`
**SLO:** P99 detection latency ≤ 500ms
**Severity:** Warning (>500ms) / Critical (>2s)

## Symptoms

- Alert fires when detection processing P99 exceeds threshold
- SOC analysts may see delayed alerts

## Investigation

### 1. Check detection service health

```bash
kubectl get pods -l app=viola-detection -n viola
kubectl top pods -l app=viola-detection -n viola
```

### 2. Check Kafka consumer lag

```bash
# If lag is high, detection is falling behind
kubectl exec -n viola deploy/viola-detection -- curl -s localhost:9090/metrics | grep kafka_consumer
```

### 3. Check for hot tenants

Look at the Grafana "Detection SLOs" dashboard → "P99 by Tenant" table. A single tenant with high event volume can spike latency.

### 4. Check rule evaluation time

```bash
kubectl logs -n viola deploy/viola-detection --tail=100 | grep "slow_rule"
```

Complex rules (especially those with Bloom filter misses) can cause latency spikes.

### 5. Check resource utilization

```bash
kubectl describe node  # Check for memory pressure
kubectl top pods -n viola --sort-by=cpu
```

## Remediation

| Cause | Action |
|-------|--------|
| High consumer lag | Scale detection replicas: `kubectl scale deploy viola-detection -n viola --replicas=5` |
| Hot tenant | Enable per-tenant rate limiting in detection config |
| Slow rules | Identify and optimize rule, or move to async evaluation |
| CPU saturation | Increase CPU limits in Helm values, redeploy |
| Memory pressure | Check for memory leaks, increase limits |

## Escalation

If latency remains >2s after scaling:
1. Page the detection team lead
2. Consider temporarily disabling expensive rules
3. Check if Kafka brokers are healthy (MSK console)
