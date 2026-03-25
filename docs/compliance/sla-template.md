# Viola Technologies — Service Level Agreement (SLA)

## 1. Service Availability

| Tier | Monthly Uptime | Allowed Downtime |
|------|---------------|------------------|
| Standard | 99.5% | ~3.6 hours |
| Professional | 99.9% | ~43 minutes |
| Enterprise | 99.95% | ~22 minutes |

**Measurement:** Uptime is calculated as the percentage of 1-minute intervals in a calendar month where the Viola API (`/health`) returns HTTP 200.

**Exclusions:** Scheduled maintenance (notified 72 hours in advance), customer-caused incidents, force majeure.

## 2. Performance SLOs

| Metric | Target | Measurement |
|--------|--------|-------------|
| API P99 latency | ≤ 200ms | 5-minute rolling window |
| Detection P99 latency | ≤ 500ms | 5-minute rolling window |
| Alert publish error rate | < 1% | 5-minute rolling window |
| API 5xx error rate | < 0.5% | 5-minute rolling window |

## 3. Incident Response

| Severity | Response Time | Resolution Target |
|----------|--------------|-------------------|
| Critical (P1) | 15 minutes | 4 hours |
| High (P2) | 1 hour | 8 hours |
| Medium (P3) | 4 hours | 3 business days |
| Low (P4) | 1 business day | 10 business days |

### Severity Definitions

- **P1 Critical:** Platform unavailable, data loss, security breach
- **P2 High:** Major feature degraded, detection pipeline stalled
- **P3 Medium:** Minor feature issue, performance degradation
- **P4 Low:** Cosmetic issues, documentation errors, feature requests

## 4. Data Retention

| Data Type | Retention |
|-----------|-----------|
| Raw telemetry events | 30 days (configurable) |
| Alerts | 1 year |
| Incidents | 1 year |
| Audit logs | 2 years |
| Response action logs | 2 years |

## 5. Backup & Recovery

| Component | Backup Frequency | RPO | RTO |
|-----------|-----------------|-----|-----|
| PostgreSQL | Continuous (WAL) | 5 minutes | 1 hour |
| Kafka | Replicated (3x) | 0 (no data loss) | Minutes (failover) |
| Configuration | Git (version controlled) | 0 | 30 minutes |

- **RPO:** Recovery Point Objective — maximum data loss window
- **RTO:** Recovery Time Objective — maximum restoration time

## 6. Service Credits

If monthly uptime falls below the SLA target:

| Uptime | Credit |
|--------|--------|
| 99.0% – SLA target | 10% of monthly fee |
| 95.0% – 99.0% | 25% of monthly fee |
| < 95.0% | 50% of monthly fee |

Credits must be requested within 30 days. Maximum credit per month: 50% of monthly fee.

## 7. Support Channels

| Channel | Availability | Tier |
|---------|-------------|------|
| Email | Business hours | All |
| Slack | Business hours | Professional+ |
| Phone | 24/7 | Enterprise |
| Dedicated CSM | 24/7 | Enterprise |

## 8. Change Notification

| Change Type | Notice Period |
|-------------|--------------|
| Scheduled maintenance | 72 hours |
| Breaking API changes | 90 days |
| Feature deprecation | 180 days |
| Security patches | Immediate (best effort) |
