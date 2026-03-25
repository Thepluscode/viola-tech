# Runbook: Detection Event Throughput Zero

**Alert:** `DetectionEventThroughputZero`
**SLO:** Detection must process >0 events per 5-minute window
**Severity:** Critical

## Symptoms

- No events are being processed by the detection engine
- No alerts are being generated
- Kafka consumer lag is growing

## Investigation

### 1. Check if detection pods are running

```bash
kubectl get pods -l app=viola-detection -n viola
kubectl describe pod -l app=viola-detection -n viola | grep -A5 "Events:"
```

### 2. Check for crash loops

```bash
kubectl logs -n viola deploy/viola-detection --previous --tail=50
```

### 3. Check upstream: is ingestion producing?

```bash
# Check if normalized topic has recent messages
kubectl exec -n viola deploy/viola-detection -- \
  kafka-console-consumer.sh --bootstrap-server $KAFKA_BROKER \
  --topic viola.prod.telemetry.v1.normalized --timeout-ms 5000 --max-messages 1
```

### 4. Check Kafka consumer group status

```bash
kubectl exec -n viola deploy/viola-detection -- \
  kafka-consumer-groups.sh --bootstrap-server $KAFKA_BROKER \
  --describe --group detection-service
```

Look for: EMPTY state, or no active members.

### 5. Check if the issue is upstream (ingestion)

If the normalized topic has no messages, the issue is in ingestion, not detection.

## Remediation

| Cause | Action |
|-------|--------|
| Pod crash loop | Check logs, fix root cause, restart |
| Consumer group rebalancing | Wait 2-3 minutes, rebalances are transient |
| Kafka broker down | Check MSK health, failover if needed |
| No upstream data | Investigate ingestion service (separate runbook) |
| OOM kill | Increase memory limits, check for memory leaks |

## Escalation

This is a **critical** alert. If detection is down for >10 minutes:
1. Page detection team lead and platform on-call
2. Check if this correlates with a deployment
3. Consider rollback if a recent deployment caused it
