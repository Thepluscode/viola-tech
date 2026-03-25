# Runbook: Gateway API Latency High

**Alert:** `GatewayAPILatencyHigh`
**SLO:** P99 API latency ≤ 200ms
**Severity:** Warning

## Symptoms

- API responses are slow
- Dashboard loading times increase
- SOC analysts report sluggish UI

## Investigation

### 1. Check which routes are slow

Grafana "Gateway API SLOs" → "Latency by Route" panel.

### 2. Check gateway pod resources

```bash
kubectl top pods -l app=viola-gateway-api -n viola
```

### 3. Check database query performance

```bash
# Connect to RDS and check slow queries
psql $DATABASE_URL -c "SELECT query, mean_exec_time, calls FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"
```

### 4. Check connection pool

```bash
kubectl logs -n viola deploy/viola-gateway-api --tail=100 | grep -i "pool\|connection\|timeout"
```

### 5. Check if auth/JWKS is slow

JWKS cache miss can add latency:

```bash
kubectl logs -n viola deploy/viola-gateway-api --tail=100 | grep -i "jwks\|auth\|token"
```

## Remediation

| Cause | Action |
|-------|--------|
| Slow DB queries | Add indexes, optimize queries |
| High request volume | Scale gateway replicas, enable HPA |
| JWKS cache miss | Increase JWKS cache TTL |
| Large response payloads | Add pagination, reduce default page size |
| Connection pool exhaustion | Increase pool size in config |

## Escalation

If P99 > 1s for more than 5 minutes, page the gateway team.
