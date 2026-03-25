# Runbook: Kafka Consumer Lag High

**Alert:** `KafkaConsumerLagHigh` / `KafkaConsumerLagCritical`
**SLO:** Consumer lag < 10,000 messages per partition
**Severity:** Warning (>10k) / Critical (>100k)

## Symptoms

- Consumer groups are falling behind on processing
- End-to-end pipeline latency increases
- Dashboard data becomes stale

## Investigation

### 1. Identify affected consumer group and topic

Check alert labels: `consumergroup`, `topic`, `partition`.

### 2. Check consumer group details

```bash
kafka-consumer-groups.sh --bootstrap-server $KAFKA_BROKER \
  --describe --group <consumergroup>
```

Look for:
- **LAG** column: how far behind
- **CLIENT-ID**: which pods are consuming
- **CURRENT-OFFSET** vs **LOG-END-OFFSET**: the gap

### 3. Check if consumers are healthy

```bash
kubectl get pods -l app=viola-<service> -n viola
kubectl top pods -l app=viola-<service> -n viola
```

### 4. Check producer rate (is there a burst?)

Look at Grafana "Kafka SLOs" dashboard → "Messages Produced/sec". A spike in upstream production can cause temporary lag.

### 5. Check Kafka broker health

```bash
# MSK Console or:
kafka-broker-api-versions.sh --bootstrap-server $KAFKA_BROKER
```

## Remediation

| Cause | Action |
|-------|--------|
| Consumer too slow | Scale up replicas for the affected service |
| Burst in production | Temporary — lag should recover. Monitor for 10 min |
| Consumer crash/restart | Check logs, ensure pods are healthy |
| Kafka partition imbalance | Rebalance partitions across brokers |
| Under-provisioned broker | Upgrade MSK instance type |

### Scaling commands

```bash
# Scale detection
kubectl scale deploy viola-detection -n viola --replicas=5

# Scale workers
kubectl scale deploy viola-workers -n viola --replicas=4

# Scale ingestion
kubectl scale deploy viola-ingestion -n viola --replicas=5
```

## Escalation

- Warning: monitor for 15 minutes, scale if not recovering
- Critical (>100k): immediate page, scale aggressively, check for data loss
