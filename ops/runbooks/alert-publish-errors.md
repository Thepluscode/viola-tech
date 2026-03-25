# Runbook: Alert Publish Error Rate High

**Alert:** `AlertPublishErrorRateHigh`
**SLO:** Alert publish error rate < 1%
**Severity:** Warning

## Symptoms

- Detection engine is processing events but failing to publish alerts to Kafka
- Alerts may be lost or delayed
- DLQ messages may increase

## Investigation

### 1. Check detection logs for publish errors

```bash
kubectl logs -n viola deploy/viola-detection --tail=200 | grep -i "publish\|error\|alert"
```

### 2. Check Kafka health

```bash
# Check if alert topic exists and is writable
kubectl exec -n viola deploy/viola-detection -- \
  kafka-topics.sh --bootstrap-server $KAFKA_BROKER --describe --topic viola.prod.security.alert.v1.created
```

### 3. Check DLQ for failed messages

```bash
kubectl exec -n viola deploy/viola-detection -- \
  kafka-console-consumer.sh --bootstrap-server $KAFKA_BROKER \
  --topic viola.prod.dlq.v1.detection --from-beginning --max-messages 5
```

### 4. Check broker disk space (MSK)

AWS Console → MSK → Cluster → Monitoring → check disk usage per broker.

## Remediation

| Cause | Action |
|-------|--------|
| Kafka broker unavailable | Check MSK cluster health, restart unhealthy brokers |
| Topic misconfigured | Recreate topic with correct replication factor |
| Network partition | Check VPC flow logs, security group rules |
| Schema mismatch | Check protobuf schema version compatibility |
| Disk full on broker | Increase MSK storage, reduce retention |

## Escalation

If error rate exceeds 5%: page platform team. Alerts are being lost.
