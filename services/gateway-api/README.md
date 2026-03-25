# Viola Gateway API

Enterprise-grade REST API for Viola's AI-native cybersecurity XDR platform. Provides secure access to incidents, alerts, and detection rules with OAuth 2.0 authentication, RBAC authorization, and comprehensive audit logging.

---

## Features

### Core Capabilities
- ✅ **REST API** - Clean, RESTful endpoints for incidents & alerts
- ✅ **Multi-Tenant** - Complete tenant isolation
- ✅ **Real-time Updates** - Event-driven architecture via Kafka

### Security & Authentication
- ✅ **OAuth 2.0 / OIDC** - Industry-standard authentication
- ✅ **JWT Validation** - Automatic JWKS discovery and key rotation
- ✅ **Multi-IdP Support** - Works with Entra ID, Okta, Auth0
- ✅ **RBAC** - Fine-grained role-based access control
- ✅ **Audit Logging** - Every sensitive action logged to Kafka

### Enterprise Features
- ✅ **Per-User Rate Limiting** - Claim-based keys, custom limits
- ✅ **Failed Auth Logging** - Sampled audit events for security monitoring
- ✅ **Token Introspection** - Real-time revocation checks with IdP
- ✅ **MFA Enforcement** - Selective MFA requirements via JWT claims
- ✅ **CORS Support** - Browser-friendly with configurable origins

### Observability
- ✅ **Prometheus Metrics** - Request rates, latencies, errors
- ✅ **Structured Logging** - JSON logs with correlation IDs
- ✅ **OpenTelemetry Tracing** - Distributed tracing support
- ✅ **Health Checks** - Liveness (`/health`) and readiness (`/ready`) probes

---

## Quick Start

### Prerequisites

```bash
# Start dependencies (PostgreSQL, Kafka)
cd /path/to/viola-platform
make dev

# Or manually:
docker-compose -f ops/docker-compose.yml up -d
```

### Build & Run

```bash
cd services/gateway-api

# Apply database migrations
make migrate-up

# Build the binary
make build

# Run the server
make run
```

**Expected Output:**
```
Database connected
Discovering JWKS URL from https://login.microsoftonline.com/<TENANT_ID>/v2.0
Discovered JWKS URL: https://...
JWKS cache initialized
OIDC authentication enabled (issuer=..., audience=api://viola-gateway)
Audit emitter initialized
Rate limiting enabled (max=120 req/min, key=[sub tid])
gateway-api listening on :8080
```

### Test the API

```bash
# Health check (no auth required)
curl http://localhost:8080/health

# Get a JWT token from your IdP (see IdP setup guides)
export TOKEN="<your-jwt-token>"

# List incidents
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/incidents

# Get specific incident
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/incidents/INC-123

# Update incident status
curl -X PATCH \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status": "ack", "assigned_to": "analyst@company.com"}' \
  http://localhost:8080/api/v1/incidents/INC-123
```

---

## Configuration

### Environment Variables

```bash
# === OIDC Authentication ===
OIDC_ISSUER_URL="https://login.microsoftonline.com/<TENANT_ID>/v2.0"
OIDC_AUDIENCE="api://viola-gateway"
OIDC_JWKS_URL=""                      # Optional, auto-discovered
AUTH_REQUIRE_BEARER="true"
AUTH_ALLOW_ALGOS="RS256,ES256"
OIDC_CLOCK_SKEW_SECONDS="120"

# === Rate Limiting ===
RATE_LIMIT_ENABLED="true"
RATE_LIMIT_PER_MIN_DEFAULT="120"      # Default requests per minute
RATE_LIMIT_KEY_CLAIMS="sub,tid"       # Claims to use for rate limit key
RATE_LIMIT_PER_MIN_CLAIM=""           # Optional: Custom per-user limit claim

# === Token Introspection (Optional) ===
INTROSPECTION_ENABLED="false"
INTROSPECTION_ENDPOINT="https://login.microsoftonline.com/<TENANT_ID>/oauth2/v2.0/introspect"
INTROSPECTION_CLIENT_ID=""
INTROSPECTION_CLIENT_SECRET=""

# === MFA Enforcement (Optional) ===
MFA_REQUIRED="false"                  # Global default (use RequireMFA() for selective enforcement)
MFA_INDICATORS="mfa,otp,totp,sms,duo,fido,phone"

# === Audit ===
AUDIT_KAFKA_BROKER="localhost:9092"
AUDIT_TOPIC="viola.dev.audit.v1.event"
SERVICE_NAME="gateway-api"

# === Database ===
PG_HOST="localhost"
PG_PORT="5432"
PG_USER="postgres"
PG_PASSWORD="postgres"
PG_DATABASE="viola_gateway"
PG_SSLMODE="disable"                  # Use "require" in production

# === Server ===
PORT="8080"
VIOLA_ENV="dev"                       # dev, staging, prod

# === Observability (Optional) ===
TRACING_ENABLED="false"
OTLP_ENDPOINT="localhost:4317"
```

---

## API Documentation

### OpenAPI Specification

The full API specification is available in `docs/openapi.yaml`.

View it with Swagger UI:
```bash
# Install swagger-ui
npx serve docs/

# Or use Swagger Editor
# https://editor.swagger.io/
# Upload docs/openapi.yaml
```

### Endpoints

| Method | Path | Description | Required Permission |
|--------|------|-------------|---------------------|
| `GET` | `/health` | Health check | None (public) |
| `GET` | `/ready` | Readiness check | None (public) |
| `GET` | `/metrics` | Prometheus metrics | None (public) |
| `GET` | `/api/v1/incidents` | List incidents | `incidents:read` or `SOCReader` |
| `GET` | `/api/v1/incidents/{id}` | Get incident | `incidents:read` or `SOCReader` |
| `PATCH` | `/api/v1/incidents/{id}` | Update incident | `incidents:write` or `SOCResponder` |
| `GET` | `/api/v1/alerts` | List alerts | `alerts:read` or `SOCReader` |
| `GET` | `/api/v1/alerts/{id}` | Get alert | `alerts:read` or `SOCReader` |
| `PATCH` | `/api/v1/alerts/{id}` | Update alert | `alerts:write` or `SOCResponder` |

### Query Parameters

**List Endpoints:**
- `status` - Filter by status (`open`, `ack`, `closed`)
- `severity` - Filter by severity (`low`, `medium`, `high`, `critical`)
- `limit` - Max results (1-200, default: 50)
- `offset` - Pagination offset

**Example:**
```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/incidents?status=open&severity=critical&limit=10"
```

---

## Identity Provider Setup

Complete step-by-step guides for configuring your IdP:

- **[Microsoft Entra ID Setup](docs/idp-setup-entra-id.md)** - App registration, roles, scopes, MFA
- **[Okta Setup](docs/idp-setup-okta.md)** - Authorization server, groups, claims
- **[Auth0 Setup](docs/idp-setup-auth0.md)** - API configuration, roles, actions

---

## Database Migrations

### Using the Migration Tool

```bash
# Check migration status
make migrate-status

# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create NAME=add_users_table
```

### Manual Migration

```bash
# Apply migrations directly
psql -h localhost -U postgres -d viola_gateway -f migrations/0000_severity.sql
psql -h localhost -U postgres -d viola_gateway -f migrations/0001_incidents.sql
psql -h localhost -U postgres -d viola_gateway -f migrations/0002_alerts.sql
psql -h localhost -U postgres -d viola_gateway -f migrations/0003_rbac.sql
```

---

## RBAC & Authorization

### Roles

| Role | Permissions | Description |
|------|-------------|-------------|
| **SOCReader** | `incidents:read`, `alerts:read` | View-only access |
| **SOCResponder** | Reader + `incidents:write`, `alerts:write` | Triage and respond |
| **SOCEngineer** | Responder + `rules:read`, `rules:write` | Manage detection rules |
| **ViolaAdmin** | `admin:*` | Full administrative access |

### Policy Configuration

Policies are defined in `internal/policy/policies.go`:

```go
var Policies = []RoutePolicy{
    {Method: "GET", Path: "/api/v1/incidents", AnyOf: []Permission{PermIncidentsRead, PermAdmin}},
    {Method: "PATCH", Path: "/api/v1/incidents/{id}", AnyOf: []Permission{PermIncidentsWrite, PermAdmin}},
    // ...
}
```

### Permission Mapping

Role/scope claims are mapped to internal permissions in `internal/authz/simple_authorizer.go`:

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

    // Scope mapping
    for _, s := range claims.Scopes {
        switch s {
        case "incidents.read":
            out[policy.PermIncidentsRead] = true
        // ...
        }
    }

    return out
}
```

---

## Testing

### Run Integration Tests

```bash
# Run all tests
make test

# Run specific test
go test -v ./test -run TestIncidentEndpoints

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Coverage

Current test suites:
- ✅ Authentication tests (valid/invalid tokens, expiry)
- ✅ Authorization tests (RBAC, tenant isolation)
- ✅ Incident CRUD operations
- ✅ Alert CRUD operations
- ✅ Multi-tenant isolation
- ✅ Workflow tests (open → ack → close)
- ✅ Error response formats

---

## Deployment

### Docker

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o gateway-api ./cmd/gateway-api

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/gateway-api /gateway-api
EXPOSE 8080
CMD ["/gateway-api"]
```

### Kubernetes

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
          env:
            - name: OIDC_ISSUER_URL
              valueFrom:
                configMapKeyRef:
                  name: gateway-config
                  key: oidc_issuer_url
            - name: PG_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: gateway-secrets
                  key: pg_password
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          resources:
            requests:
              memory: "256Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
```

---

## Troubleshooting

### Authentication Issues

**Problem:** `401 Unauthorized - invalid token`

**Solutions:**
1. Verify `OIDC_ISSUER_URL` matches JWT `iss` claim exactly
2. Verify `OIDC_AUDIENCE` matches JWT `aud` claim
3. Check token expiration (`exp` claim)
4. Verify signing algorithm is allowed (`AUTH_ALLOW_ALGOS`)
5. Ensure JWKS endpoint is accessible

**Debug:**
```bash
# Decode JWT to inspect claims
echo $TOKEN | cut -d. -f2 | base64 -d | jq

# Check JWKS URL
curl https://login.microsoftonline.com/<TENANT_ID>/discovery/v2.0/keys
```

### Authorization Issues

**Problem:** `403 Forbidden`

**Solutions:**
1. Check user has required role assigned in IdP
2. Verify role/scope claim mapping in `authz/simple_authorizer.go`
3. Check route policy in `policy/policies.go`

**Debug:**
```bash
# Decode token and check roles claim
echo $TOKEN | jwt decode -

# Expected claims:
# {
#   "roles": ["SOCReader", "SOCResponder"],
#   "scp": "incidents.read incidents.write"
# }
```

### Rate Limiting

**Problem:** `429 Too Many Requests`

**Solutions:**
1. Wait for the time indicated in `Retry-After` header
2. Request custom rate limit via `RATE_LIMIT_PER_MIN_CLAIM`
3. Check `RATE_LIMIT_PER_MIN_DEFAULT` configuration

**Headers:**
```
X-RateLimit-Limit: 120
X-RateLimit-Remaining: 0
Retry-After: 42
```

### Database Connection

**Problem:** `503 Service Unavailable - DB unavailable`

**Solutions:**
1. Verify PostgreSQL is running
2. Check connection parameters (`PG_HOST`, `PG_PORT`, `PG_USER`, `PG_PASSWORD`)
3. Verify database exists (`PG_DATABASE`)
4. Check network connectivity and firewall rules

**Test:**
```bash
psql -h localhost -U postgres -d viola_gateway -c "SELECT 1"
```

---

## Performance Tuning

### Connection Pooling

Database connection pool is configured in `internal/db/db.go`:
```go
pool.SetMaxOpenConns(20)  // Max connections
pool.SetMaxIdleConns(10)  // Idle connections
```

### Rate Limiting

Sliding window implementation provides accurate rate limiting:
- Default: 120 requests/minute per user+tenant
- Configurable via `RATE_LIMIT_PER_MIN_DEFAULT`
- Custom limits via JWT claim

### Caching

- JWKS cache: 10-minute refresh (configurable)
- Rate limit windows: In-memory with periodic cleanup

---

## Security Best Practices

1. **Use HTTPS in production** - JWT tokens are bearer tokens
2. **Enable MFA for sensitive operations** - Use `RequireMFA()` middleware
3. **Rotate secrets regularly** - Client secrets, database passwords
4. **Monitor audit logs** - Set up alerts for suspicious activity
5. **Use strong JWT algorithms** - Only `RS256`, `ES256` (never `HS256`)
6. **Implement token introspection** - For real-time revocation
7. **Set appropriate clock skew** - Max 2 minutes (`OIDC_CLOCK_SKEW_SECONDS=120`)
8. **Use short token lifetimes** - Access token: 1 hour, Refresh token: 7 days

---

## Architecture

```
┌─────────────────────────────────────────┐
│  Client (Browser / CLI / Service)      │
└──────────────┬──────────────────────────┘
               │ HTTPS + Bearer Token
               ▼
┌─────────────────────────────────────────┐
│  Gateway API (This Service)             │
│  ┌────────────────────────────────────┐ │
│  │ Middleware Stack:                  │ │
│  │  1. RequestID, RealIP, Logging     │ │
│  │  2. OIDC Auth (JWT validation)     │ │
│  │  3. Rate Limiting (per-user)       │ │
│  │  4. RBAC (role-based access)       │ │
│  │  5. MFA (optional, selective)      │ │
│  │  6. Introspection (optional)       │ │
│  └────────────────────────────────────┘ │
│  ┌────────────────────────────────────┐ │
│  │ Handlers:                          │ │
│  │  - Incidents (List, Get, Update)   │ │
│  │  - Alerts (List, Get, Update)      │ │
│  └────────────────────────────────────┘ │
│  ┌────────────────────────────────────┐ │
│  │ Store Layer (pgx/v5):              │ │
│  │  - IncidentStore                   │ │
│  │  - AlertStore                      │ │
│  └────────────────────────────────────┘ │
└──────┬─────────────────┬────────────────┘
       │                 │
       ▼                 ▼
┌─────────────┐   ┌──────────────┐
│  PostgreSQL │   │  Kafka       │
│  (State)    │   │  (Audit Log) │
└─────────────┘   └──────────────┘
```

---

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for development guidelines.

---

## License

Proprietary - Viola Technologies

---

## Support

- **Documentation**: [docs/enterprise-auth-deployment.md](docs/enterprise-auth-deployment.md)
- **Issues**: https://github.com/viola/platform/issues
- **Security**: security@viola.com
