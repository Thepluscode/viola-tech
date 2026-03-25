# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Viola Technologies software, **do not open a public issue.**

### How to Report

1. **Email:** security@viola.tech
2. **Subject line:** `[VULN] <brief description>`
3. **Include:**
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact assessment
   - Your name/handle for credit (optional)

### Response Timeline

| Stage | SLA |
|-------|-----|
| Acknowledgment | Within 24 hours |
| Initial triage | Within 72 hours |
| Status update | Every 5 business days |
| Fix deployed (critical) | Within 7 days |
| Fix deployed (high) | Within 30 days |
| Fix deployed (medium/low) | Within 90 days |

### Severity Classification

- **Critical:** Remote code execution, authentication bypass, data exfiltration
- **High:** Privilege escalation, SQL injection, XSS with session theft
- **Medium:** Information disclosure, CSRF, denial of service
- **Low:** Configuration issues, minor information leaks

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.x.x (current) | Yes |

## Security Architecture

### Data Protection
- All data encrypted at rest (AES-256 via AWS KMS)
- All data encrypted in transit (TLS 1.3)
- Tenant isolation enforced at application and database layer
- No PII stored in logs

### Authentication & Authorization
- OIDC/JWT-based authentication with JWKS key rotation
- Role-based access control (admin, analyst, viewer)
- Per-tenant authorization enforcement on all API routes
- MFA support via OIDC provider

### Infrastructure
- VPC with private subnets for all services
- Network policies restrict service-to-service communication
- Secrets managed via AWS Secrets Manager (never in environment variables in production)
- VPC flow logs enabled for network forensics

### Secure Development
- All dependencies scanned weekly (govulncheck, npm audit, Trivy)
- CodeQL static analysis on every pull request
- No vendor dependencies with known critical vulnerabilities
- Signed container images via GHCR

## Compliance

Viola Technologies maintains compliance with:
- **SOC 2 Type I** (in progress)
- **NIST CSF** alignment
- **OWASP Top 10** mitigation

## Bug Bounty

We do not currently operate a formal bug bounty program. We will credit security researchers who responsibly disclose vulnerabilities (with permission).
