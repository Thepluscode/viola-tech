# Runbook: Alert Generation Rate Drop

**Alert:** `AlertGenerationRateDrop`
**Threshold:** >80% drop vs 1 hour ago
**Severity:** Warning

## Symptoms

- Sudden drop in alert generation
- Could indicate detection blind spot
- Not necessarily a problem — could be legitimate reduction in threats

## Investigation

### 1. Check if detection is processing events

```bash
kubectl logs -n viola deploy/viola-detection --tail=50 | grep "processed"
```

If events are being processed but no alerts generated, the issue is in rule matching.

### 2. Check rule loading

```bash
kubectl logs -n viola deploy/viola-detection --tail=100 | grep -i "rule\|load\|index"
```

A failed rule reload can cause detection to run with zero rules.

### 3. Check upstream telemetry volume

Is the raw event volume also down? If so, the issue is upstream (fewer events = fewer alerts).

### 4. Check for recent rule changes

```bash
# If rules are in ConfigMap:
kubectl get configmap viola-detection-rules -n viola -o yaml | head -20
```

### 5. Verify with manual test

Produce a test event that should trigger a known rule and verify an alert is generated.

## Remediation

| Cause | Action |
|-------|--------|
| Failed rule reload | Restart detection, check rule syntax |
| Upstream event drop | Investigate ingestion/agent pipeline |
| Rule accidentally deleted | Restore from git, redeploy |
| Bloom filter corruption | Restart detection (rebuilds filter) |
| Legitimate quiet period | Acknowledge alert, no action needed |

## Escalation

If alert rate is zero for >15 minutes and events are flowing, page detection team — this is a detection blind spot.
