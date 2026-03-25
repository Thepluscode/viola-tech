# Gateway API Deployment Guide

## Overview

The gateway-api service provides the REST API for Viola's security platform. It consists of two components:

1. **HTTP API Server** (`gateway-api`) - REST endpoints for querying alerts and incidents
2. **Materializer Worker** (`materializer`) - Kafka consumer that persists events to Postgres

---

## Architecture

```
Kafka Topics                  Materializer Worker              Postgres
┌────────────────┐           ┌──────────────────┐           ┌─────────────┐
│ alert.created  │──────────>│ Alert Handler    │──────────>│ alerts      │
│ alert.updated  │──────────>│ Alert Handler    │──────────>│ alert_*     │
│ incident.upsert│──────────>│ Incident Handler │──────────>│ incidents   │
└────────────────┘           └──────────────────┘           │ incident_*  │
                                                            └─────────────┘
                                                                    ^
                                                                    │
                             ┌──────────────────┐                  │
                             │ HTTP API Server  │──────────────────┘
                             │ (OIDC + RBAC)    │
                             └──────────────────┘
                                      ^
                                      │
                                  REST Clients
                                  (UI, CLI)
```

---

## Configuration

### Environment Variables

#### Gateway API Server

```bash
# Postgres
PG_HOST=localhost
PG_PORT=5432
PG_USER=postgres
PG_PASSWORD=<secret>
PG_DATABASE=viola_gateway
PG_SSLMODE=disable  # Use 'require' in production

# OIDC (optional - if not set, auth is bypassed)
OIDC_ISSUER=https://your-tenant.auth0.com/
OIDC_AUDIENCE=https://api.viola.security
OIDC_JWKS_URL=https://your-tenant.auth0.com/.well-known/jwks.json

# Server
PORT=8080
```

#### Materializer Worker

```bash
# Postgres (same as API)
PG_HOST=localhost
PG_PORT=5432
PG_USER=postgres
PG_PASSWORD=<secret>
PG_DATABASE=viola_gateway
PG_SSLMODE=disable

# Kafka
VIOLA_ENV=dev  # prod|staging|dev
KAFKA_BROKER=localhost:9092
```

---

## Database Setup

Run migrations:

```bash
cd services/gateway-api
psql -h localhost -U postgres -d viola_gateway -f migrations/0000_severity.sql
psql -h localhost -U postgres -d viola_gateway -f migrations/0001_incidents.sql
psql -h localhost -U postgres -d viola_gateway -f migrations/0002_alerts.sql
psql -h localhost -U postgres -d viola_gateway -f migrations/0003_rbac.sql
```

Or use a migration tool like `golang-migrate`:

```bash
migrate -path migrations -database "postgres://postgres:password@localhost:5432/viola_gateway?sslmode=disable" up
```

---

## Running Locally

### 1. Start Postgres

```bash
docker run -d \
  --name viola-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=viola_gateway \
  -p 5432:5432 \
  postgres:15
```

### 2. Run Migrations

```bash
psql -h localhost -U postgres -d viola_gateway -f migrations/*.sql
```

### 3. Start Kafka (for materializer)

```bash
docker run -d \
  --name viola-kafka \
  -p 9092:9092 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  apache/kafka:latest
```

### 4. Start API Server

```bash
cd services/gateway-api
go run cmd/gateway-api/main.go
```

API available at: http://localhost:8080

### 5. Start Materializer Worker

```bash
cd services/gateway-api
go run cmd/materializer/main.go
```

---

## API Endpoints

### Health Checks

- `GET /health` - Basic health check
- `GET /ready` - Readiness check (includes DB ping)

### Incidents

- `GET /api/v1/incidents` - List incidents
  - Query params: `status`, `severity`, `limit`, `offset`
- `GET /api/v1/incidents/{id}` - Get incident details
- `PATCH /api/v1/incidents/{id}` - Update incident
  - Body: `{"status": "ack", "assigned_to": "user@example.com"}`

### Alerts

- `GET /api/v1/alerts` - List alerts
  - Query params: `status`, `severity`, `limit`, `offset`
- `GET /api/v1/alerts/{id}` - Get alert details
- `PATCH /api/v1/alerts/{id}` - Update alert
  - Body: `{"status": "closed", "closure_reason": "false positive"}`

---

## Authentication

### Local Development (OIDC Disabled)

Set `OIDC_ISSUER=""` to bypass authentication.

**INSECURE**: Only use for local development.

### Production (OIDC Enabled)

Set OIDC environment variables:

```bash
OIDC_ISSUER=https://your-tenant.auth0.com/
OIDC_AUDIENCE=https://api.viola.security
OIDC_JWKS_URL=https://your-tenant.auth0.com/.well-known/jwks.json
```

#### JWT Claims Required

```json
{
  "email": "user@example.com",
  "tenant_id": "tenant-123",
  "roles": ["analyst"],
  "iss": "https://your-tenant.auth0.com/",
  "aud": "https://api.viola.security",
  "exp": 1234567890
}
```

#### Example Request

```bash
curl -H "Authorization: Bearer <JWT_TOKEN>" \
  https://api.viola.security/api/v1/incidents
```

---

## RBAC

### Default Roles

| Role    | incidents:read | incidents:update | alerts:read | alerts:update |
|---------|----------------|------------------|-------------|---------------|
| admin   | ✅              | ✅                | ✅           | ✅             |
| analyst | ✅              | ✅                | ✅           | ✅             |
| viewer  | ✅              | ❌                | ✅           | ❌             |

### Custom Policies

Add tenant-specific policies:

```sql
INSERT INTO rbac_policies (tenant_id, role, resource, action, allowed)
VALUES ('tenant-abc', 'readonly', 'incidents', 'read', true);
```

---

## Production Deployment

### Kubernetes Deployment

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
        - name: PG_HOST
          value: postgres.viola.svc.cluster.local
        - name: PG_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres-secret
              key: password
        - name: OIDC_ISSUER
          value: https://viola.auth0.com/
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: materializer
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: materializer
        image: viola/materializer:latest
        env:
        - name: KAFKA_BROKER
          value: kafka.viola.svc.cluster.local:9092
        - name: PG_HOST
          value: postgres.viola.svc.cluster.local
```

---

## Security Considerations

### 1. OIDC Must Be Enabled in Production

**Never** run production with `OIDC_ISSUER=""`. This bypasses all authentication.

### 2. Use SSL for Postgres

Set `PG_SSLMODE=require` in production.

### 3. JWKS Key Rotation

The OIDC middleware automatically refreshes JWKS keys every 1 hour. Ensure your IdP supports JWKS endpoints.

### 4. Tenant Isolation

All queries are scoped by `tenant_id` extracted from JWT claims. **Never** allow users to specify `tenant_id` in API requests.

### 5. Audit Logging

API actions are logged via chi middleware. For compliance, integrate with external audit systems.

---

## Troubleshooting

### 401 Unauthorized

- Check JWT is valid (not expired)
- Verify `OIDC_ISSUER` and `OIDC_AUDIENCE` match JWT claims
- Ensure JWKS URL is accessible

### 403 Forbidden

- Check user's `roles` claim in JWT
- Verify RBAC policy exists: `SELECT * FROM rbac_policies WHERE tenant_id = 'xxx' AND role = 'analyst';`

### Materializer Not Persisting Events

- Check Kafka consumer lag: `kafka-consumer-groups --describe --group viola.dev.gateway-api.materializer.alerts-created`
- Check DLQ topic for failures: `viola.dev.dlq.v1.workers`
- Check Postgres logs for constraint violations

### Database Connection Refused

- Verify `PG_HOST` and `PG_PORT`
- Check network connectivity: `psql -h $PG_HOST -U $PG_USER -d $PG_DATABASE`
- Ensure Postgres is accepting connections

---

## Next Steps

1. **Add Observability**:
   - Prometheus metrics for API latency, error rates
   - Structured logging with request IDs
   - OpenTelemetry tracing

2. **Add Rate Limiting**:
   - Per-tenant rate limits
   - Redis-backed rate limiter

3. **Add Caching**:
   - Redis cache for frequently accessed incidents
   - Cache invalidation on Kafka events

4. **Add Compliance Features**:
   - SOC 2 audit logging
   - Data retention enforcement
   - GDPR data export endpoints
