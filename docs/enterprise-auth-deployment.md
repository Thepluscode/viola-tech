# Enterprise Authentication & Authorization Deployment Guide

## Overview

Viola's enterprise-ready auth system provides:
- **OIDC JWT validation** (Entra ID, Okta, Auth0, or any OIDC provider)
- **JWKS refresh + key rotation tolerance** (automatic key refresh)
- **RBAC** enforced per route via central policy table
- **Audit events** for every sensitive action (immutable Kafka log)
- **Multi-tenant isolation** at the authentication layer

---

## Architecture

```
┌────────────────────────────────────────┐
│   Client (Browser / API)               │
└──────────────┬─────────────────────────┘
               │ Bearer Token (JWT)
               ▼
┌────────────────────────────────────────┐
│   Gateway API                          │
│   ┌─────────────────────────────────┐  │
│   │  1. OIDC Middleware             │  │
│   │     - Extract Bearer token      │  │
│   │     - Validate JWT signature    │  │
│   │     - Check iss/aud/exp         │  │
│   │     - Load claims to context    │  │
│   └─────────────────────────────────┘  │
│   ┌─────────────────────────────────┐  │
│   │  2. RBAC Middleware             │  │
│   │     - Map claims → permissions  │  │
│   │     - Check route policy        │  │
│   │     - Deny if unauthorized      │  │
│   └─────────────────────────────────┘  │
│   ┌─────────────────────────────────┐  │
│   │  3. Handler + Audit             │  │
│   │     - Process request           │  │
│   │     - Emit audit event          │  │
│   └─────────────────────────────────┘  │
└────────────────────────────────────────┘
               │
               ▼
         ┌──────────────┐
         │  Kafka       │
         │  Audit Log   │
         └──────────────┘
```

---

## Quick Start

### 1. Environment Configuration

**`.env` or Kubernetes secrets:**

```bash
# OIDC Configuration
OIDC_ISSUER_URL="https://login.microsoftonline.com/<TENANT_ID>/v2.0"  # Entra ID
# OIDC_ISSUER_URL="https://dev-12345.okta.com/oauth2/default"         # Okta
# OIDC_ISSUER_URL="https://YOUR_DOMAIN.auth0.com/"                    # Auth0

OIDC_AUDIENCE="api://viola-gateway"  # Your API audience/client ID
OIDC_JWKS_URL=""                     # Optional; auto-discovered if empty

# Auth Behavior
AUTH_REQUIRE_BEARER="true"
AUTH_ALLOW_ALGOS="RS256,ES256"       # Restrict to asymmetric algorithms
OIDC_CLOCK_SKEW_SECONDS="120"        # Clock skew tolerance

# RBAC
RBAC_DEFAULT_DENY="true"             # Deny by default if no policy matches

# Audit
AUDIT_KAFKA_BROKER="localhost:9092"
AUDIT_TOPIC="viola.dev.audit.v1.event"
SERVICE_NAME="gateway-api"

# Service
PORT="8080"
PG_HOST="localhost"
PG_PORT="5432"
PG_USER="postgres"
PG_PASSWORD="postgres"
PG_DATABASE="viola_gateway"
```

### 2. Start Gateway API

```bash
cd services/gateway-api

# With OIDC enabled
OIDC_ISSUER_URL=https://login.microsoftonline.com/<TENANT_ID>/v2.0 \
OIDC_AUDIENCE=api://viola-gateway \
AUDIT_KAFKA_BROKER=localhost:9092 \
go run cmd/gateway-api/main.go
```

**Expected Output:**

```
Database connected
Discovering JWKS URL from https://login.microsoftonline.com/<TENANT_ID>/v2.0
Discovered JWKS URL: https://login.microsoftonline.com/<TENANT_ID>/discovery/v2.0/keys
JWKS cache initialized
OIDC authentication enabled (issuer=..., audience=api://viola-gateway)
Audit emitter initialized
Metrics server listening on :8080
gateway-api listening on :8080
```

### 3. Test Authentication

**Get a JWT token from your IdP**, then:

```bash
# Get incidents (requires incidents:read permission)
curl -H "Authorization: Bearer <JWT_TOKEN>" \
  http://localhost:8080/api/v1/incidents

# Update incident (requires incidents:write permission)
curl -X PATCH \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"status": "ack"}' \
  http://localhost:8080/api/v1/incidents/INC-123
```

---

## OIDC Configuration by Provider

### Microsoft Entra ID (Azure AD)

**1. Register Application in Azure Portal**

1. Go to **Azure Active Directory** → **App registrations** → **New registration**
2. Name: `Viola Gateway API`
3. Supported account types: **Single tenant**
4. Redirect URI: Not needed for API-only apps
5. Click **Register**

**2. Configure API Permissions**

1. Go to **API permissions** → **Add a permission**
2. Select **APIs my organization uses**
3. Add custom scopes for your app (or use delegated permissions)

**3. Expose an API**

1. Go to **Expose an API**
2. Set **Application ID URI** = `api://viola-gateway`
3. Add scopes:
   - `incidents.read` (Display name: "Read incidents", Description: "...")
   - `incidents.write`
   - `alerts.read`
   - `alerts.write`
   - `rules.write`

**4. Get Configuration Values**

- **Tenant ID**: Found in **Overview** page
- **Issuer**: `https://login.microsoftonline.com/<TENANT_ID>/v2.0`
- **Audience**: `api://viola-gateway`

**5. Environment Variables**

```bash
OIDC_ISSUER_URL="https://login.microsoftonline.com/<TENANT_ID>/v2.0"
OIDC_AUDIENCE="api://viola-gateway"
```

### Okta

**1. Create API in Okta**

1. Go to **Applications** → **Applications** → **Create App Integration**
2. Select **API Services**
3. Name: `Viola Gateway API`
4. Click **Save**

**2. Add Scopes**

1. Go to **Security** → **API** → **Authorization Servers**
2. Select `default` or create a new one
3. Go to **Scopes** → **Add Scope**
4. Add scopes: `incidents.read`, `incidents.write`, etc.

**3. Environment Variables**

```bash
OIDC_ISSUER_URL="https://dev-12345.okta.com/oauth2/default"
OIDC_AUDIENCE="api://default"  # Or your custom audience
```

### Auth0

**1. Create API in Auth0**

1. Go to **Applications** → **APIs** → **Create API**
2. Name: `Viola Gateway API`
3. Identifier: `https://api.viola.com` (your API URL)
4. Signing Algorithm: **RS256**

**2. Add Permissions (Scopes)**

1. Go to **Permissions** tab
2. Add: `incidents:read`, `incidents:write`, etc.

**3. Environment Variables**

```bash
OIDC_ISSUER_URL="https://YOUR_DOMAIN.auth0.com/"
OIDC_AUDIENCE="https://api.viola.com"
```

---

## RBAC Policy Management

### Central Policy Table

**Location:** `services/gateway-api/internal/policy/policies.go`

This is the **single source of truth** for route authorization.

```go
var Policies = []RoutePolicy{
    // Incidents
    {Method: "GET", Path: "/api/v1/incidents", AnyOf: []Permission{PermIncidentsRead, PermAdmin}},
    {Method: "GET", Path: "/api/v1/incidents/{id}", AnyOf: []Permission{PermIncidentsRead, PermAdmin}},
    {Method: "PATCH", Path: "/api/v1/incidents/{id}", AnyOf: []Permission{PermIncidentsWrite, PermAdmin}},

    // Alerts
    {Method: "GET", Path: "/api/v1/alerts", AnyOf: []Permission{PermAlertsRead, PermAdmin}},
    {Method: "PATCH", Path: "/api/v1/alerts/{id}", AnyOf: []Permission{PermAlertsWrite, PermAdmin}},
}
```

**Policy Fields:**
- `Method`: HTTP method (GET, POST, PATCH, DELETE)
- `Path`: Route pattern (supports `{id}` placeholders)
- `AnyOf`: Require **at least one** of these permissions
- `AllOf`: Require **all** of these permissions

### Permission Mapping

**Location:** `services/gateway-api/internal/authz/simple_authorizer.go`

Maps JWT claims (roles/scopes) → internal permissions:

```go
func (a SimpleAuthorizer) PermissionsFor(c any) map[policy.Permission]bool {
    claims := c.(*auth.Claims)
    out := map[policy.Permission]bool{}

    // Admin shortcut
    for _, r := range claims.Roles {
        if strings.EqualFold(r, "ViolaAdmin") || strings.EqualFold(r, "Admin") {
            out[policy.PermAdmin] = true
        }
    }

    // Scope mapping (recommended for APIs)
    for _, s := range claims.Scopes {
        switch s {
        case "incidents.read":
            out[policy.PermIncidentsRead] = true
        case "incidents.write":
            out[policy.PermIncidentsWrite] = true
        // ...
        }
    }

    // Role mapping (fallback for coarse-grained roles)
    for _, r := range claims.Roles {
        switch r {
        case "SOCReader":
            out[policy.PermIncidentsRead] = true
            out[policy.PermAlertsRead] = true
        case "SOCResponder":
            out[policy.PermIncidentsRead] = true
            out[policy.PermIncidentsWrite] = true
            out[policy.PermAlertsRead] = true
            out[policy.PermAlertsWrite] = true
        }
    }

    return out
}
```

### Recommended Role Hierarchy

| Role | Permissions | Use Case |
|------|-------------|----------|
| **SOCReader** | `incidents:read`, `alerts:read` | View-only access for analysts |
| **SOCResponder** | Reader + `incidents:write`, `alerts:write` | Triage and respond to incidents |
| **SOCEngineer** | Responder + `rules:write` | Create/update detection rules |
| **ViolaAdmin** | `admin:*` | Full access (bypasses all checks) |

---

## Audit Event Schema

All sensitive actions emit audit events to Kafka.

**Proto Schema:** `shared/proto/audit/audit.proto`

```protobuf
message AuditEvent {
  string audit_id = 1;
  string tenant_id = 2;
  string request_id = 3;
  string occurred_at = 4;  // RFC3339

  // Actor (who)
  string actor_type = 5;   // "user", "service", "system"
  string actor_id = 6;     // user email, service name
  string actor_ip = 7;

  // Action (what)
  string resource_type = 8;  // "incident", "alert", "rule"
  string resource_id = 9;
  string action = 10;        // "create", "update", "delete", "acknowledge", "close"
  string outcome = 11;       // "success", "failure", "denied"

  // Context (how/why)
  map<string, string> metadata = 12;
  string reason = 13;
}
```

### Audit Coverage Map

| Action | Resource Type | Event Triggered |
|--------|---------------|-----------------|
| Update incident status | `incident` | `acknowledge` / `close` / `update` |
| Assign incident | `incident` | `assign` |
| Update alert status | `alert` | `acknowledge` / `close` / `update` |
| Assign alert | `alert` | `assign` |
| Create detection rule | `rule` | `create` |
| Update detection rule | `rule` | `update` |
| Delete detection rule | `rule` | `delete` |
| Failed auth | `auth` | `login` (outcome=`denied`) |

**Example Audit Event:**

```json
{
  "audit_id": "AUD-abc123",
  "tenant_id": "tenant-xyz",
  "request_id": "req-456def",
  "occurred_at": "2026-02-14T10:30:45Z",
  "actor_type": "user",
  "actor_id": "alice@company.com",
  "actor_ip": "192.168.1.100",
  "resource_type": "incident",
  "resource_id": "INC-789",
  "action": "close",
  "outcome": "success",
  "metadata": {
    "status": "\"closed\"",
    "closure_reason": "\"False positive\""
  }
}
```

### Consuming Audit Events

**Kafka Topic:** `viola.dev.audit.v1.event`

```bash
# Read audit events
kcat -C -b localhost:9092 -t viola.dev.audit.v1.event | jq

# Filter by tenant
kcat -C -b localhost:9092 -t viola.dev.audit.v1.event | \
  jq 'select(.tenant_id == "tenant-xyz")'

# Find incident closures
kcat -C -b localhost:9092 -t viola.dev.audit.v1.event | \
  jq 'select(.resource_type == "incident" and .action == "close")'
```

---

## JWKS Key Rotation

The JWKS cache automatically handles key rotation:

1. **Initial Fetch**: Loads all public keys from JWKS endpoint at startup
2. **Periodic Refresh**: Refreshes every 10 minutes (configurable)
3. **On-Demand Refresh**: If a JWT uses an unknown `kid`, force refresh once
4. **Rotation Tolerance**: Old keys remain cached until JWKS endpoint stops listing them

**Configuration:**

```go
jwksCache, err := auth.NewJWKSCache(auth.JWKSCacheConfig{
    JWKSURL:      jwksURL,
    RefreshEvery: 10 * time.Minute,  // Adjust based on IdP key rotation policy
})
```

**Recommended Refresh Intervals:**
- **Entra ID**: 10 minutes (keys rotate ~every 90 days)
- **Okta**: 10 minutes (keys rotate every 90 days)
- **Auth0**: 5 minutes (more frequent rotation possible)

---

## Testing

### 1. Test with Mock JWT (Local Dev)

Disable OIDC for local testing:

```bash
AUTH_REQUIRE_BEARER=false go run cmd/gateway-api/main.go
```

**Warning:** This bypasses all authentication! Only use in local dev.

### 2. Test with Real JWT

**Get a token from your IdP:**

**Entra ID (using Azure CLI):**

```bash
az login
TOKEN=$(az account get-access-token --resource api://viola-gateway --query accessToken -o tsv)
echo $TOKEN
```

**Okta (using curl):**

```bash
curl -X POST https://dev-12345.okta.com/oauth2/default/v1/token \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -d "client_id=YOUR_CLIENT_ID" \
  -d "client_secret=YOUR_CLIENT_SECRET" \
  -d "scope=incidents.read alerts.read"
```

**Test API endpoints:**

```bash
# List incidents
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/incidents

# Update incident (should succeed with incidents:write scope)
curl -X PATCH \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "ack"}' \
  http://localhost:8080/api/v1/incidents/INC-123

# Expected on success: {"status": "updated"}
# Expected on forbidden: {"error": "forbidden"}
```

### 3. Test RBAC Enforcement

**Create tokens with different scopes:**

1. Token with `incidents.read` only → Can GET, cannot PATCH
2. Token with `incidents.write` → Can PATCH
3. Token with `Admin` role → Can do anything

### 4. Verify Audit Events

```bash
# Start Kafka consumer
kcat -C -b localhost:9092 -t viola.dev.audit.v1.event

# Trigger an action
curl -X PATCH -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "closed"}' \
  http://localhost:8080/api/v1/incidents/INC-123

# Expected audit event in Kafka:
{
  "actor_id": "alice@company.com",
  "resource_type": "incident",
  "resource_id": "INC-123",
  "action": "close",
  "outcome": "success"
}
```

---

## Production Deployment (Kubernetes)

### ConfigMap for Environment Variables

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: gateway-api-config
data:
  OIDC_ISSUER_URL: "https://login.microsoftonline.com/<TENANT_ID>/v2.0"
  OIDC_AUDIENCE: "api://viola-gateway"
  AUTH_REQUIRE_BEARER: "true"
  AUTH_ALLOW_ALGOS: "RS256,ES256"
  OIDC_CLOCK_SKEW_SECONDS: "120"
  RBAC_DEFAULT_DENY: "true"
  AUDIT_KAFKA_BROKER: "kafka.viola.svc.cluster.local:9092"
  AUDIT_TOPIC: "viola.prod.audit.v1.event"
  SERVICE_NAME: "gateway-api"
```

### Secret for Sensitive Data

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gateway-api-secrets
type: Opaque
stringData:
  PG_PASSWORD: "your-secure-password"
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway-api
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: gateway-api
          image: viola/gateway-api:latest
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef:
                name: gateway-api-config
            - secretRef:
                name: gateway-api-secrets
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 5
```

---

## Troubleshooting

### JWT Validation Fails

**Symptom:** `401 Unauthorized` with "invalid token"

**Debug:**

1. **Check issuer/audience:**
   ```bash
   # Decode JWT (use jwt.io or jwt-cli)
   echo $TOKEN | jwt decode -
   # Verify "iss" matches OIDC_ISSUER_URL
   # Verify "aud" matches OIDC_AUDIENCE
   ```

2. **Check token expiry:**
   ```bash
   # Verify "exp" is in the future
   date -r <exp_timestamp>
   ```

3. **Check JWKS URL:**
   ```bash
   curl https://login.microsoftonline.com/<TENANT_ID>/discovery/v2.0/keys
   # Verify "kid" from JWT header exists in JWKS
   ```

4. **Check algorithm:**
   ```bash
   # Verify "alg" from JWT header is in AUTH_ALLOW_ALGOS
   ```

### RBAC Denies Access Unexpectedly

**Symptom:** `403 Forbidden`

**Debug:**

1. **Check claims:**
   - Decode JWT and verify `roles` or `scp` claims contain expected values
   - Example: Entra uses `roles`, Okta uses `scope`

2. **Check permission mapping:**
   - Review `authz/simple_authorizer.go`
   - Ensure claim values map to correct permissions

3. **Check route policy:**
   - Review `policy/policies.go`
   - Verify route pattern matches and required permissions are correct

4. **Enable debug logging:**
   - Add logging to `rbac.go` middleware to see computed permissions

### Audit Events Not Appearing

**Symptom:** No events in Kafka

**Debug:**

1. **Check Kafka broker:**
   ```bash
   kcat -L -b localhost:9092
   ```

2. **Check topic exists:**
   ```bash
   kcat -L -b localhost:9092 -t viola.dev.audit.v1.event
   ```

3. **Check auditor initialization:**
   - Look for "Audit emitter initialized" in logs
   - If "Audit disabled", set `AUDIT_KAFKA_BROKER`

4. **Check handler is calling emit:**
   - Verify `h.emitAudit()` is called in Update handlers

---

## Security Best Practices

1. **Use asymmetric algorithms only** (`RS256`, `ES256`) - never `HS256` for public APIs
2. **Set audience claim** - prevents token reuse across different APIs
3. **Limit clock skew** - 2 minutes max (`OIDC_CLOCK_SKEW_SECONDS=120`)
4. **Rotate JWKS regularly** - IdP handles this, cache refreshes automatically
5. **Audit all sensitive actions** - incident/alert updates, rule changes, admin actions
6. **Never log JWTs** - they contain sensitive claims
7. **Use HTTPS in production** - JWT tokens are bearer tokens (anyone with token has access)
8. **Enable rate limiting** - prevent brute force token guessing and API abuse (see Enterprise Security Enhancements)
9. **Monitor failed auth attempts** - use audit logs to detect credential stuffing and brute force attacks
10. **Require MFA for admin actions** - use selective MFA enforcement for high-privilege operations

---

## Enterprise Security Enhancements

The gateway API includes additional security features for production deployments:

### 1. Per-User Rate Limiting

Prevents abuse by limiting requests per user/tenant combination using JWT claims.

**Configuration:**

```bash
# Enable rate limiting
RATE_LIMIT_ENABLED="true"
RATE_LIMIT_PER_MIN_DEFAULT="120"              # Default: 120 requests per minute
RATE_LIMIT_KEY_CLAIMS="sub,tid"               # Claims to use for key (default: sub + tenant_id)
RATE_LIMIT_PER_MIN_CLAIM="viola_rl_per_min"   # Optional: Custom per-user limit claim
```

**How It Works:**

1. Extracts claims from JWT (e.g., `sub`, `tid`, `email`)
2. Builds rate limit key by combining claims: `user-123:tenant-xyz`
3. Tracks requests in sliding window (default 1 minute)
4. Returns `429 Too Many Requests` when limit exceeded
5. Adds headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`

**Custom Per-User Limits:**

Add a custom claim to your JWT to override the default limit:

```json
{
  "sub": "user-123",
  "tid": "tenant-xyz",
  "viola_rl_per_min": 500  // This user gets 500 req/min instead of default 120
}
```

**Testing:**

```bash
# Get token
TOKEN="your-jwt-token"

# Send multiple requests
for i in {1..125}; do
  curl -H "Authorization: Bearer $TOKEN" \
    http://localhost:8080/api/v1/incidents
done

# After 120 requests, you should see:
# HTTP/1.1 429 Too Many Requests
# {"error":"rate_limit_exceeded","message":"Too many requests. Try again in 42 seconds."}
```

**Fallback for Unauthenticated Requests:**

For routes without authentication, rate limiting falls back to IP-based limiting.

---

### 2. Failed Authentication Audit Logging

Tracks authentication failures (401 errors) for security monitoring and threat detection.

**Configuration:**

Failed auth logging is automatically enabled when both `authMiddleware` and `auditor` are configured. Sampling is set to 10% by default to prevent log spam from brute force attacks.

**What Gets Logged:**

| Failure Reason | Description | Example |
|----------------|-------------|---------|
| `missing_token` | No Authorization header provided | Unauthorized API access attempt |
| `token_expired` | JWT past expiration time | Token refresh needed |
| `issuer_mismatch` | JWT issuer doesn't match config | Wrong IdP or malicious token |
| `audience_mismatch` | JWT audience doesn't match config | Token meant for different API |
| `unknown_key_id` | JWT signed with unknown key | Key rotation issue or forged token |
| `invalid_token` | Generic validation failure | Malformed or tampered token |

**Audit Event Example:**

```json
{
  "audit_id": "AUD-failed-auth-123",
  "tenant_id": "system",
  "actor_type": "user",
  "actor_id": "unknown",
  "actor_ip": "203.0.113.42",
  "resource_type": "auth",
  "resource_id": "/api/v1/incidents",
  "action": "authenticate",
  "outcome": "denied",
  "reason": "token_expired",
  "metadata": {
    "method": "GET",
    "path": "/api/v1/incidents",
    "details": "token expired at 2026-02-14T10:00:00Z"
  }
}
```

**Querying Failed Auth Events:**

```bash
# Find all failed auth attempts
kcat -C -b localhost:9092 -t viola.dev.audit.v1.event | \
  jq 'select(.resource_type == "auth" and .outcome == "denied")'

# Find brute force attempts from specific IP
kcat -C -b localhost:9092 -t viola.dev.audit.v1.event | \
  jq 'select(.resource_type == "auth" and .actor_ip == "203.0.113.42")' | \
  jq -s 'length'  # Count attempts

# Group by failure reason
kcat -C -b localhost:9092 -t viola.dev.audit.v1.event | \
  jq -r 'select(.resource_type == "auth") | .reason' | \
  sort | uniq -c
```

**Sampling Configuration:**

The default 10% sampling means only 1 in 10 failed auth attempts are logged. This is configurable in code:

```go
authMiddleware = &apimiddleware.AuthMiddleware{
    RequireBearer: true,
    Verifier:      verifier,
    Auditor:       auditor,
    AuditSample:   0.1,  // 10% sampling (0.0-1.0)
}
```

Adjust sampling based on your threat model:
- **High security environments**: 1.0 (100% logging)
- **Normal operations**: 0.1 (10% sampling)
- **Cost-sensitive**: 0.01 (1% sampling)

---

### 3. Token Introspection (Revocation Checks)

Checks with the IdP if a token has been revoked, providing real-time revocation enforcement.

**When to Use:**

Token introspection adds 50-200ms latency per request, so only use it for:
- High-security operations (DELETE, admin actions)
- Sensitive data access
- Compliance requirements (immediate revocation needed)

**Configuration:**

```bash
# Enable token introspection
INTROSPECTION_ENABLED="true"
INTROSPECTION_ENDPOINT="https://login.microsoftonline.com/<TENANT_ID>/oauth2/v2.0/introspect"
INTROSPECTION_CLIENT_ID="your-client-id"
INTROSPECTION_CLIENT_SECRET="your-client-secret"
```

**IdP-Specific Endpoints:**

**Microsoft Entra ID:**
```bash
INTROSPECTION_ENDPOINT="https://login.microsoftonline.com/<TENANT_ID>/oauth2/v2.0/introspect"
```

**Okta:**
```bash
INTROSPECTION_ENDPOINT="https://dev-12345.okta.com/oauth2/default/v1/introspect"
```

**Auth0:**
```bash
INTROSPECTION_ENDPOINT="https://YOUR_DOMAIN.auth0.com/oauth/token/introspection"
```

**Usage Patterns:**

**Global Introspection (not recommended):**
```go
// main.go - applies to ALL routes
if introspectionMiddleware != nil {
    r.Use(introspectionMiddleware.Handler)
}
```

**Selective Introspection (recommended):**
```go
// Apply only to sensitive routes
r.With(introspectionMiddleware.RequireIntrospection()).Delete("/api/v1/incidents/{id}", handler)
r.With(introspectionMiddleware.RequireIntrospection()).Post("/api/v1/rules", handler)
r.With(introspectionMiddleware.RequireIntrospection()).Delete("/api/v1/rules/{id}", handler)
```

**Error Handling:**

The middleware is **fail-open** by default - if introspection fails (IdP down, timeout), the request is allowed. This prevents IdP outages from breaking your entire API.

For critical routes requiring **fail-closed** behavior, modify the middleware:

```go
// introspection.go
if err != nil {
    // Fail-closed: deny request on introspection errors
    http.Error(w, "unable to verify token status", http.StatusServiceUnavailable)
    return
}
```

**Testing:**

```bash
# 1. Get a token
TOKEN=$(az account get-access-token --resource api://viola-gateway --query accessToken -o tsv)

# 2. Use token (should succeed)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/incidents

# 3. Revoke token in IdP (via Azure Portal, Okta Admin, etc.)

# 4. Try again (should fail with 401)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/incidents

# Expected: {"error": "token revoked"}
```

**Performance Considerations:**

- **Cache**: Consider caching introspection results for 30-60 seconds
- **Async**: For non-critical routes, check revocation asynchronously and log violations
- **Selective**: Only enable for routes where immediate revocation is critical

---

### 4. MFA Enforcement

Requires multi-factor authentication for sensitive operations by checking JWT claims.

**Configuration:**

```bash
# Enable MFA enforcement capability
MFA_REQUIRED="false"  # Global default - use RequireMFA() for specific routes
MFA_INDICATORS="mfa,otp,totp,sms,duo,fido,phone"
```

**How It Works:**

The middleware checks two JWT claims:

1. **`amr` (Authentication Methods Reference)** - Primary check
   - Array or string of authentication methods used
   - Common values: `["mfa"]`, `["pwd", "otp"]`, `["fido"]`

2. **`acr` (Authentication Context Class Reference)** - Fallback
   - String indicating authentication strength
   - Entra ID: `"http://schemas.microsoft.com/claims/multipleauthn"`
   - REFEDS MFA: `"https://refeds.org/profile/mfa"`

**Supported MFA Indicators:**

| Indicator | Description | Example IdP |
|-----------|-------------|-------------|
| `mfa` | Generic MFA | Most IdPs |
| `otp` | One-time password | Okta, Auth0 |
| `totp` | Time-based OTP | Google Authenticator |
| `sms` | SMS verification | Entra, Okta |
| `duo` | Duo Security | Duo |
| `fido` | FIDO/WebAuthn | Entra, Auth0 |
| `phone` | Phone call | Entra |
| `hwk` | Hardware key | Entra |

**JWT Examples:**

**Entra ID with MFA:**
```json
{
  "sub": "user-123",
  "amr": ["pwd", "mfa"],
  "acr": "http://schemas.microsoft.com/claims/multipleauthn"
}
```

**Okta with TOTP:**
```json
{
  "sub": "user-123",
  "amr": ["pwd", "totp"]
}
```

**Auth0 with FIDO:**
```json
{
  "sub": "user-123",
  "amr": ["fido"]
}
```

**Usage Patterns:**

**Selective MFA (recommended):**
```go
// Require MFA for sensitive routes only
r.With(mfaMiddleware.RequireMFA()).Patch("/api/v1/incidents/{id}", handler)
r.With(mfaMiddleware.RequireMFA()).Delete("/api/v1/rules/{id}", handler)
r.With(mfaMiddleware.RequireMFA()).Post("/api/v1/admin/users", handler)
```

**Combined Protections:**
```go
// Combine MFA + introspection for maximum security
r.With(
    mfaMiddleware.RequireMFA(),
    introspectionMiddleware.RequireIntrospection(),
).Delete("/api/v1/rules/{id}", handler)
```

**Error Response:**

When MFA is missing:

```json
HTTP/1.1 403 Forbidden
{
  "error": "mfa_required",
  "message": "Multi-factor authentication is required for this operation"
}
```

**Testing:**

```bash
# 1. Get token WITHOUT MFA (username + password only)
TOKEN_NO_MFA="..."

# 2. Try to update incident (should fail)
curl -X PATCH \
  -H "Authorization: Bearer $TOKEN_NO_MFA" \
  -H "Content-Type: application/json" \
  -d '{"status": "closed"}' \
  http://localhost:8080/api/v1/incidents/INC-123

# Expected: 403 Forbidden with mfa_required error

# 3. Get token WITH MFA (username + password + TOTP)
TOKEN_WITH_MFA="..."

# 4. Try again (should succeed)
curl -X PATCH \
  -H "Authorization: Bearer $TOKEN_WITH_MFA" \
  -H "Content-Type: application/json" \
  -d '{"status": "closed"}' \
  http://localhost:8080/api/v1/incidents/INC-123

# Expected: 200 OK
```

**Configuring MFA in IdPs:**

**Microsoft Entra ID:**
1. Go to **Azure Active Directory** → **Security** → **MFA**
2. Enable MFA for users/groups
3. Configure Conditional Access policy to require MFA for API access
4. Result: `amr` claim will include `"mfa"` or `acr` will be set

**Okta:**
1. Go to **Security** → **Multifactor**
2. Enable factors (TOTP, SMS, etc.)
3. Create Sign-On Policy requiring MFA
4. Result: `amr` claim will include `"otp"` or `"totp"`

**Auth0:**
1. Go to **Security** → **Multi-factor Auth**
2. Enable MFA factors
3. Create Rule to require MFA for API access
4. Result: `amr` claim will include authentication method

---

## Next Steps

1. **Add SAML support** - For enterprises requiring SAML 2.0
2. **Add device fingerprinting** - Track device trust scores
3. **Add geofencing** - Restrict access by geographic location
4. **Add session management** - Track active sessions per user
5. **Add anomaly detection** - ML-based unusual activity detection

---

## References

- [Microsoft Entra ID (Azure AD) Docs](https://learn.microsoft.com/en-us/azure/active-directory/)
- [Okta OIDC](https://developer.okta.com/docs/concepts/oauth-openid/)
- [Auth0 API Authorization](https://auth0.com/docs/get-started/apis)
- [OIDC Specification](https://openid.net/specs/openid-connect-core-1_0.html)
- [JWT Best Practices](https://datatracker.ietf.org/doc/html/rfc8725)
