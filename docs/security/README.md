# Security Overview

> **Security documentation for Pandora Exchange**  
> **Last Updated:** November 8, 2025

---

## Security Architecture

Pandora Exchange implements defense-in-depth security with multiple layers:

```
┌─────────────────────────────────────────────┐
│     External Layer (Internet)                │
│  • TLS 1.3 Encryption                       │
│  • CORS Protection                          │
│  • Rate Limiting (planned)                  │
└─────────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────────┐
│     Authentication Layer                     │
│  • JWT Access Tokens (15min expiry)        │
│  • Refresh Tokens (7 days)                 │
│  • Argon2id Password Hashing               │
└─────────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────────┐
│     Authorization Layer                      │
│  • Role-Based Access Control (RBAC)        │
│  • Admin Middleware                         │
│  • Resource-level permissions               │
└─────────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────────┐
│     Application Layer                        │
│  • Input Validation                         │
│  • SQL Injection Protection (sqlc)         │
│  • Error Handling (no info leakage)        │
└─────────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────────┐
│     Data Layer                               │
│  • Database Encryption at Rest              │
│  • Audit Logging (immutable)                │
│  • PII Redaction                            │
│  • Soft Deletes (compliance)                │
└─────────────────────────────────────────────┘
                    │
┌─────────────────────────────────────────────┐
│     Secrets Management                       │
│  • HashiCorp Vault                          │
│  • No secrets in code/env files            │
│  • Secret rotation (planned)                │
│  • HSM integration (planned)                │
└─────────────────────────────────────────────┘
```

---

## Authentication

### Password Security

**Algorithm:** Argon2id (Winner of Password Hashing Competition 2015)

**Parameters:**
```go
time:    1 iteration
memory:  64 MB (65536 KB)
threads: 4
salt:    16 bytes (random, unique per password)
output:  32 bytes
```

**Why Argon2id?**
- ✅ Resistant to GPU/ASIC attacks (memory-hard)
- ✅ Resistant to side-channel attacks
- ✅ Tunable parameters for future-proofing
- ✅ Industry standard for password hashing

**Implementation:**
```go
// Never stored in plaintext
hashedPassword := argon2.IDKey(
    []byte(password),
    salt,
    1,      // time
    65536,  // memory (64 MB)
    4,      // threads
    32,     // key length
)
```

### JWT Tokens

**Access Token:**
- Expiry: 15 minutes (configurable)
- Algorithm: HS256 (HMAC with SHA-256)
- Claims: user_id, email, role, exp, iat
- Storage: Client-side (memory, not localStorage for XSS protection)

**Refresh Token:**
- Expiry: 7 days (configurable)
- Stored in database (can be revoked)
- One-time use (rotated on refresh)
- Hashed before storage

**Token Rotation:**
```
1. Client requests new access token with refresh token
2. Server validates refresh token
3. Server generates new access + refresh tokens
4. Server invalidates old refresh token
5. Client receives new token pair
```

---

## Authorization

### Role-Based Access Control (RBAC)

**Roles:**
- `user` - Standard user (default)
  - Can access own profile
  - Can update own profile
  - Cannot access admin endpoints
  
- `admin` - Administrative user
  - All user permissions
  - Can list all users
  - Can update any user's KYC status
  - Can delete any user

**Middleware Stack:**
```go
// Public endpoints
router.POST("/auth/register", handlers.Register)
router.POST("/auth/login", handlers.Login)

// Protected endpoints (JWT required)
auth := router.Group("/")
auth.Use(middleware.AuthMiddleware())
{
    auth.GET("/users/me", handlers.GetProfile)
    auth.PUT("/users/me", handlers.UpdateProfile)
}

// Admin endpoints (JWT + admin role required)
admin := router.Group("/admin")
admin.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware())
{
    admin.GET("/users", handlers.ListUsers)
    admin.PUT("/users/:id/kyc", handlers.UpdateUserKYC)
}
```

---

## Secrets Management

### HashiCorp Vault Integration

**See [VAULT_INTEGRATION.md](../VAULT_INTEGRATION.md) for complete guide.**

**Secrets Stored in Vault:**
- JWT signing secret
- Database password
- Redis password
- Future: API keys, HSM credentials

**Development vs Production:**
| Environment | Vault Enabled | Fallback |
|-------------|---------------|----------|
| Development | No | Environment variables |
| Sandbox | Yes | Environment variables (warning logged) |
| Production | Yes | **No fallback** (fail-fast) |

**Kubernetes Integration:**
- Vault Agent Injector sidecar
- Secrets injected as environment variables
- Automatic secret renewal
- No secrets in ConfigMaps/Secrets

---

## Data Protection

### PII Handling

**Personally Identifiable Information (PII):**
- Email addresses
- Names (first, last)
- IP addresses
- User agent strings

**Protection Measures:**
1. **Logging:** PII redacted in production logs
2. **Encryption:** Database encryption at rest
3. **Access Control:** Audit all PII access
4. **Retention:** Soft delete preserves audit trail
5. **GDPR:** Right to access, right to erasure

**Example Log Redaction:**
```go
// Development
log.Info().Str("email", "user@example.com").Msg("user login")

// Production
log.Info().Str("email", "u***@e***.com").Msg("user login")
```

### Database Security

**SQL Injection Prevention:**
- ✅ All queries via sqlc (parameterized)
- ❌ No raw SQL execution allowed
- ✅ Input validation before DB access

**Encryption:**
- At rest: PostgreSQL encryption
- In transit: TLS for database connections
- Backups: Encrypted with customer-managed keys

**Access Control:**
- Principle of least privilege
- Service accounts per environment
- No direct production DB access
- Audit all privileged operations

---

## Audit Logging

**See [AUDIT_RETENTION_POLICY.md](../AUDIT_RETENTION_POLICY.md) for complete policy.**

**All Sensitive Operations Logged:**
- User registration
- Login attempts (success and failure)
- Profile updates
- KYC status changes
- Account deletions
- Admin operations

**Audit Log Structure:**
```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID,
    action TEXT NOT NULL,          -- e.g., "user.created"
    resource TEXT NOT NULL,         -- e.g., "user"
    resource_id TEXT,               -- e.g., user UUID
    changes JSONB,                  -- old/new values
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMP NOT NULL
);
```

**Immutability:**
- No UPDATE or DELETE allowed (except automated cleanup)
- Retention period: 90 days (configurable)
- Compliance with SOC 2, GDPR requirements

---

## Network Security

### TLS Configuration

**Production Requirements:**
- TLS 1.3 only
- Strong cipher suites only
- HSTS headers
- Certificate pinning (planned)

**Development:**
- HTTP allowed for local development
- Self-signed certificates for testing

### CORS Protection

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://pandora.exchange"},
    AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    ExposeHeaders:    []string{"Content-Length"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}))
```

---

## Error Handling Security

**See [ERROR_HANDLING.md](../ERROR_HANDLING.md) for complete guide.**

**Never Leak Internal Details:**

**❌ Insecure:**
```json
{
  "error": "pq: duplicate key value violates unique constraint \"users_email_key\""
}
```

**✅ Secure:**
```json
{
  "error": {
    "code": "USER_ALREADY_EXISTS",
    "message": "User already exists with this email",
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
  }
}
```

**Generic Error Messages:**
- "Invalid credentials" (not "Email not found" or "Wrong password")
- "User not found" (not "No user with ID X")
- "Internal server error" (not stack traces)

---

## Threat Model

### Threats & Mitigations

| Threat | Impact | Mitigation | Status |
|--------|--------|------------|--------|
| **Password Brute Force** | High | Argon2id + rate limiting | ✅ Partial |
| **SQL Injection** | Critical | sqlc parameterized queries | ✅ Complete |
| **XSS** | High | Input sanitization, CSP headers | ✅ Complete |
| **CSRF** | Medium | CORS, SameSite cookies | ✅ Complete |
| **JWT Theft** | High | Short expiry, HTTPOnly cookies | ✅ Complete |
| **Secrets Exposure** | Critical | Vault, no secrets in code | ✅ Complete |
| **Audit Tampering** | High | Immutable logs | ✅ Complete |
| **DDoS** | Medium | Rate limiting, CDN | ⚪ Planned |
| **Account Takeover** | High | MFA, anomaly detection | ⚪ Planned |

---

## Compliance

### GDPR (General Data Protection Regulation)

**Right to Access:**
- ✅ `GET /users/me` - User can access their data

**Right to Erasure:**
- ✅ `DELETE /users/me` - Soft delete with audit trail
- ⚪ Hard delete after retention period (planned)

**Data Minimization:**
- ✅ Only collect necessary data
- ✅ No tracking pixels or analytics (yet)

**Consent:**
- ✅ Explicit consent during registration
- ⚪ Granular consent management (planned)

### SOC 2 (Service Organization Control 2)

**Security:**
- ✅ Access control (RBAC)
- ✅ Encryption in transit (TLS)
- ✅ Encryption at rest (PostgreSQL)

**Availability:**
- ✅ Health checks
- ✅ Graceful shutdown
- ⚪ Auto-scaling (Kubernetes HPA)

**Confidentiality:**
- ✅ Secrets in Vault
- ✅ PII redaction in logs
- ✅ Audit logging

**Processing Integrity:**
- ✅ Input validation
- ✅ Error handling
- ✅ Idempotency (planned)

**Privacy:**
- ✅ Data minimization
- ✅ Soft delete
- ⚪ Data anonymization (planned)

---

## Security Testing

### Automated Security Scanning

**Planned:**
- Dependency scanning (Dependabot)
- SAST (Static Application Security Testing)
- Container scanning (Trivy)
- Secret scanning (git-secrets)

### Penetration Testing

**Schedule:** Annually (external firm)

**Scope:**
- API endpoints
- Authentication/authorization
- Database security
- Infrastructure security

---

## Incident Response

### Security Incident Process

1. **Detection**
   - Monitor logs for anomalies
   - Alert on suspicious patterns

2. **Containment**
   - Disable compromised accounts
   - Rotate secrets if exposed
   - Block malicious IPs

3. **Investigation**
   - Review audit logs
   - Trace using OpenTelemetry trace IDs
   - Identify root cause

4. **Remediation**
   - Patch vulnerabilities
   - Update security controls
   - Notify affected users

5. **Post-Mortem**
   - Document incident
   - Update runbooks
   - Improve detection

---

## Future Security Enhancements

### Planned Features

- [ ] **Multi-Factor Authentication (MFA)**
  - TOTP (Time-based One-Time Password)
  - SMS backup codes
  - Hardware keys (WebAuthn)

- [ ] **Rate Limiting**
  - Per IP address
  - Per user account
  - Adaptive rate limiting

- [ ] **HSM Integration**
  - Hardware Security Module for wallet keys
  - FIPS 140-2 Level 3 compliance

- [ ] **Anomaly Detection**
  - Login from unusual locations
  - Unusual trading patterns
  - Account takeover detection

- [ ] **Advanced Audit**
  - Real-time audit streaming
  - Audit analytics dashboard
  - Compliance reports

---

## Security Contacts

**Report Security Issues:**
- Email: security@pandora.exchange
- Slack: #pandora-security (internal)
- PGP Key: [Link to public key]

**Bug Bounty Program:** Coming soon

---

## References

- [VAULT_INTEGRATION.md](../VAULT_INTEGRATION.md) - Vault setup and usage
- [ERROR_HANDLING.md](../ERROR_HANDLING.md) - Secure error responses
- [AUDIT_RETENTION_POLICY.md](../AUDIT_RETENTION_POLICY.md) - Audit policy
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)

---

**Last Updated:** November 8, 2025  
**Security Officer:** [Name]  
**Next Review:** February 2026
