# Viola Technologies — SOC 2 Type I Control Narrative

## Company Overview

Viola Technologies provides an AI-native Extended Detection and Response (XDR) platform that ingests endpoint, cloud, identity, and network telemetry to detect, correlate, and respond to security threats in real-time.

## Trust Service Criteria

### CC1 — Control Environment

**CC1.1 — The entity demonstrates a commitment to integrity and ethical values.**

Viola Technologies maintains a code of conduct that all employees acknowledge upon hire. Security is a core company value — the product is a security platform, and we hold ourselves to the same standards we help our customers achieve.

**CC1.2 — The board of directors demonstrates independence from management.**

[To be completed upon board formation]

**CC1.3 — Management establishes structure, authority, and responsibility.**

Engineering teams are organized by service domain (detection, platform, frontend). Each team has a designated lead responsible for security within their domain. The CTO maintains overall security authority.

---

### CC2 — Communication and Information

**CC2.1 — The entity obtains or generates relevant information.**

- **Audit logging:** All API requests, authentication events, and response actions are logged to a dedicated Kafka audit topic with tenant context, request IDs, and timestamps.
- **Infrastructure monitoring:** Prometheus collects metrics from all services. Grafana dashboards provide real-time visibility into system health, SLO compliance, and security events.
- **Alert rules:** 15+ Prometheus alert rules monitor SLO compliance with defined runbooks.

**CC2.2 — The entity internally communicates information.**

- **Incident response:** Runbooks are maintained in version control (`ops/runbooks/`) and linked from all alert rules.
- **Change management:** All code changes require pull request review. CI/CD pipelines enforce linting, testing, and security scanning before merge.

---

### CC3 — Risk Assessment

**CC3.1 — The entity specifies objectives with sufficient clarity.**

Viola's security objectives:
1. Protect customer telemetry data from unauthorized access
2. Maintain platform availability (99.9% SLO target)
3. Detect and respond to security events within defined SLAs
4. Ensure tenant data isolation

**CC3.2 — The entity identifies and analyzes risks.**

Key risks identified and mitigated:
- **Tenant data leakage:** Mitigated by tenant_id enforcement at application layer, composite primary keys, and API-level authorization checks.
- **Kafka message tampering:** Mitigated by TLS encryption in transit, required header validation, and dead-letter queue routing for invalid messages.
- **Credential exposure:** Mitigated by AWS Secrets Manager, KMS encryption, and no secrets in environment variables in production.

---

### CC5 — Control Activities

**CC5.1 — The entity selects and develops control activities.**

#### Access Control
- OIDC/JWT authentication required for all API endpoints
- RBAC policies stored in database with per-tenant scope
- JWT tokens signed with RSA-256 and rotated via JWKS
- MFA supported via OIDC provider configuration

#### Data Protection
- PostgreSQL encrypted at rest via AWS KMS
- Kafka (MSK) encrypted at rest and in transit via TLS
- VPC private subnets for all data-plane services
- No public internet access for database or Kafka clusters

#### Change Management
- All code changes via pull requests with required reviews
- CI pipeline: lint → test → build → security scan
- Container images scanned with Trivy before deployment
- CodeQL static analysis on every pull request

#### Network Security
- Kubernetes NetworkPolicies restrict inter-service communication
- Ingress limited to gateway-api via nginx ingress controller
- VPC flow logs enabled for network forensics
- Security groups enforce least-privilege network access

#### Monitoring
- Prometheus metrics from all services (15s scrape interval)
- SLO alerting rules with defined severity levels and runbooks
- Kafka consumer lag monitoring for pipeline health
- Dead-letter queue monitoring for data quality

---

### CC6 — Logical and Physical Access

**CC6.1 — The entity implements logical access controls.**

- AWS IAM with IRSA (IAM Roles for Service Accounts) for least-privilege Kubernetes workload access
- No shared credentials — each service has its own IAM role
- Database access requires TLS and credentials from Secrets Manager
- EKS API endpoint is private in production

**CC6.6 — The entity implements controls to prevent or detect unauthorized access.**

- JWT token validation on every API request
- Tenant ID extracted from JWT and enforced against requested resources
- Audit events emitted to Kafka for all state-changing operations
- 401 rate monitoring with alerting for credential stuffing detection

---

### CC7 — System Operations

**CC7.1 — The entity uses detection and monitoring procedures.**

- Real-time SLO monitoring via Prometheus + Grafana
- 15+ alert rules covering detection latency, error rates, Kafka lag, API health
- Each alert has a linked runbook with investigation and remediation steps
- Dead-letter queues capture failed messages for analysis

**CC7.2 — The entity monitors system components.**

- All services expose /health endpoints with liveness/readiness probes
- Kubernetes HPA auto-scales gateway-api based on CPU utilization
- Pod Disruption Budgets ensure availability during deployments
- Container image vulnerabilities scanned weekly via Trivy

---

### CC8 — Change Management

**CC8.1 — The entity authorizes, designs, develops, configures, implements, operates, approves, and maintains infrastructure and software.**

- Infrastructure defined as code (Terraform) with state stored in encrypted S3
- Kubernetes deployments managed via Helm charts with versioned values
- Three environments (dev, staging, prod) with progressive deployment
- Rollback capability via Kubernetes deployment revision history

---

### CC9 — Risk Mitigation

**CC9.1 — The entity identifies, selects, and develops risk mitigation activities.**

- Multi-AZ deployment in production for high availability
- Database multi-AZ with automated failover
- Kafka cluster with replication factor 3 and min.insync.replicas=2
- 30-day database backup retention in production
- Deletion protection enabled on production RDS instances

---

## Evidence Artifacts

| Control | Evidence Location |
|---------|-------------------|
| CI/CD Pipeline | `.github/workflows/ci.yaml` |
| Security Scanning | `.github/workflows/security.yaml` |
| SLO Alerting | `ops/alerts/slo_alert_rules.yml` |
| Runbooks | `ops/runbooks/` |
| Infrastructure as Code | `infra/terraform/` |
| Kubernetes Deployment | `infra/helm/viola/` |
| Network Policies | `infra/helm/viola/templates/networkpolicy.yaml` |
| RBAC | `services/gateway-api/migrations/0003_rbac.sql` |
| Audit Logging | `shared/go/proto/audit/audit.proto` |
| Tenant Isolation | All service stores enforce tenant_id |
| Encryption Config | `infra/terraform/modules/rds/main.tf` (KMS), `infra/terraform/modules/msk/main.tf` (TLS) |
