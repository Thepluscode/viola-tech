# Runbook: Auth Failure Spike

**Alert:** `GatewayAuthFailureSpike`
**Threshold:** 401 rate > 10% of all requests
**Severity:** Warning (security team)

## Symptoms

- High volume of 401 Unauthorized responses
- Possible credential stuffing or brute force attack
- Legitimate users may be affected if auth service is degraded

## Investigation

### 1. Check if auth service is healthy

```bash
curl -s http://localhost:8081/health
kubectl logs -n viola deploy/viola-auth --tail=100
```

### 2. Determine if it's an attack or service issue

**Attack indicators:**
- High 401 rate from many different IPs
- Consistent request patterns (automated)
- Targeting /token endpoint

**Service issue indicators:**
- All users getting 401s
- JWKS endpoint returning errors
- Key rotation in progress

### 3. Check request patterns

```bash
kubectl logs -n viola deploy/viola-gateway-api --tail=500 | grep "401" | awk '{print $NF}' | sort | uniq -c | sort -rn | head
```

### 4. Check JWKS endpoint

```bash
curl -s http://auth:8081/.well-known/jwks.json | jq .
```

## Remediation

| Cause | Action |
|-------|--------|
| Credential stuffing | Enable rate limiting per IP, block offending IPs |
| Token expiry | Check TOKEN_TTL config, users may need to re-auth |
| JWKS key rotation | Ensure old keys are still served during transition |
| Auth service down | Restart auth service, check logs |
| Clock skew | Check NTP sync on all nodes |

## Security Response

If confirmed as an attack:
1. Enable aggressive rate limiting
2. Block source IPs at WAF/ALB level
3. Notify security team
4. Check if any credentials were compromised
5. Consider forcing password resets for affected accounts
