# Runbook: Gateway API Error Rate High

**Alert:** `GatewayAPIErrorRateHigh`
**SLO:** 5xx error rate < 0.5%
**Severity:** Warning

## Symptoms

- API returning 500 errors
- Frontend shows error states
- SOC analysts can't access data

## Investigation

### 1. Check error logs

```bash
kubectl logs -n viola deploy/viola-gateway-api --tail=200 | grep -i "error\|panic\|500"
```

### 2. Check which routes are failing

Grafana "Gateway API SLOs" → filter by status code 5xx.

### 3. Check downstream dependencies

```bash
# Is Postgres healthy?
kubectl exec -n viola deploy/viola-gateway-api -- pg_isready -h $PG_HOST

# Is Kafka reachable?
kubectl logs -n viola deploy/viola-gateway-api --tail=50 | grep -i "kafka\|broker"
```

### 4. Check for recent deployments

```bash
kubectl rollout history deploy/viola-gateway-api -n viola
```

## Remediation

| Cause | Action |
|-------|--------|
| Database down | Check RDS health, failover if needed |
| Kafka unreachable | Check MSK cluster, security groups |
| Application bug | Check logs, rollback if recent deploy |
| OOM / resource limits | Increase limits, restart pods |
| Panic/crash | Fix bug, deploy hotfix |

## Escalation

If 5xx rate > 5%: page gateway team and platform on-call.
