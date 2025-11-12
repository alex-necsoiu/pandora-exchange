# 🔐 Security

Comprehensive security documentation for Pandora Exchange User Service.

## 📋 Table of Contents

- [Security Overview](#security-overview)
- [Password Security](#password-security)
- [Authentication & Authorization](#authentication--authorization)
- [Secrets Management](#secrets-management)
- [API Security](#api-security)
- [Audit & Compliance](#audit--compliance)
- [Network Security](#network-security)
- [Security Best Practices](#security-best-practices)
- [Incident Response](#incident-response)

---

## Security Overview

Pandora Exchange implements **defense-in-depth** security with multiple layers of protection:

```
┌─────────────────────────────────────────────┐
│  Network Security (TLS/HTTPS, mTLS)         │
├─────────────────────────────────────────────┤
│  API Security (Rate Limiting, CORS)         │
├─────────────────────────────────────────────┤
│  Authentication (JWT, Argon2id)             │
├─────────────────────────────────────────────┤
│  Authorization (RBAC, Middleware)           │
├─────────────────────────────────────────────┤
│  Data Security (Encryption at Rest/Transit) │
├─────────────────────────────────────────────┤
│  Audit & Compliance (Immutable Logs)        │
└─────────────────────────────────────────────┘
```

---

## Password Security

### Argon2id Hashing

We use **Argon2id**, the winner of the Password Hashing Competition (PHC), with secure parameters:

**Parameters:**
```go
const (
    Time        = 1      // 1 iteration
    Memory      = 64 * 1024  // 64 MB
    Threads     = 4      // 4 parallel threads
    KeyLength   = 32     // 32-byte hash
    SaltLength  = 16     // 16-byte salt
)
```

**Why Argon2id?**
- ✅ Memory-hard (resistant to GPU/ASIC attacks)
- ✅ Side-channel resistant
- ✅ Configurable work factor
- ✅ Industry standard (OWASP recommended)

**Implementation:**
```go
// Hash password
hash, err := argon2id.Hash(password)

// Verify password (constant-time comparison)
valid, err := argon2id.Verify(password, hash)
```

### Password Requirements

**Minimum Requirements:**
- Length: 8-128 characters
- Must contain:
  - At least 1 uppercase letter
  - At least 1 lowercase letter
  - At least 1 digit
  - At least 1 special character (`!@#$%^&*`)

**Forbidden:**
- Common passwords (checked against list)
- User's email or name
- Sequential characters (`123456`, `abcdef`)
- Repeated characters (`aaaaaa`, `111111`)

### Timing Attack Protection

All password comparisons use **constant-time** algorithms to prevent timing attacks:

```go
// Constant-time string comparison
func secureCompare(a, b string) bool {
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
```

---

## Authentication & Authorization

### JWT (JSON Web Tokens)

**Access Token Structure:**
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
.
{
  "sub": "user-uuid",
  "email": "user@example.com",
  "role": "user",
  "iat": 1699785600,
  "exp": 1699786500,
  "iss": "pandora-exchange",
  "jti": "token-uuid"
}
```

**Token Lifecycle:**

| Token Type | Expiry | Storage | Purpose |
|------------|--------|---------|---------|
| **Access Token** | 15 minutes | Client memory | API authentication |
| **Refresh Token** | 7 days | Database | Renew access tokens |

**Security Features:**
- ✅ Short-lived access tokens (15 min)
- ✅ Refresh token rotation (new token on each refresh)
- ✅ Token revocation (logout invalidates refresh tokens)
- ✅ Multi-device support (track sessions per device)
- ✅ Automatic expiry (database cleanup job)

### Role-Based Access Control (RBAC)

**Roles:**

| Role | Permissions | Endpoints |
|------|-------------|-----------|
| `user` | Standard operations | `/api/v1/users/me`, `/api/v1/auth/*` |
| `admin` | All operations | `/api/v1/admin/*`, all user endpoints |

**Middleware Protection:**

```go
// Require authentication
router.Use(middleware.AuthMiddleware())

// Require admin role
adminRouter.Use(middleware.RequireRole("admin"))
```

### Session Management

**Features:**
- Track sessions per device (user agent, IP address)
- Logout single session or all sessions
- Automatic session expiry
- Session activity logging

**Session Data:**
```json
{
  "refresh_token": "uuid",
  "user_id": "user-uuid",
  "device": "Mozilla/5.0...",
  "ip_address": "192.168.1.1",
  "created_at": "2024-11-12T10:30:00Z",
  "expires_at": "2024-11-19T10:30:00Z"
}
```

---

## Secrets Management

### HashiCorp Vault Integration

**Production Secret Management:**

```yaml
# Vault configuration
vault:
  enabled: true
  addr: https://vault.example.com
  token: ${VAULT_TOKEN}  # From K8s secret or environment
  secret_path: secret/data/user-service
```

**Secrets Stored in Vault:**
- JWT signing keys
- Database credentials
- Redis passwords
- API keys for external services
- Encryption keys

**Kubernetes Integration:**

```yaml
# Vault Agent Sidecar Injection
annotations:
  vault.hashicorp.com/agent-inject: "true"
  vault.hashicorp.com/role: "user-service"
  vault.hashicorp.com/agent-inject-secret-config: "secret/data/user-service"
```

### Environment-Based Secret Management

| Environment | Secret Storage | Rotation |
|-------------|---------------|----------|
| **dev** | `.env.dev` file (NOT committed) | Manual |
| **sandbox** | Vault | Weekly |
| **audit** | Vault | Weekly |
| **prod** | Vault | Daily |

### Secret Rotation

**Automated Rotation (Production):**
- Database passwords: Daily
- JWT signing keys: Weekly
- API keys: Monthly
- Vault tokens: Daily

**Manual Rotation Procedure:**
1. Generate new secret in Vault
2. Update application config (no restart needed)
3. Verify new secret works
4. Revoke old secret after grace period (24h)
5. Update audit logs

---

## API Security

### Rate Limiting

**Redis-Backed Sliding Window Algorithm:**

| Scope | Limit | Window | Purpose |
|-------|-------|--------|---------|
| **Global (IP)** | 100 req | 1 minute | Prevent DDoS |
| **Per-User** | 60 req | 1 minute | Fair usage |
| **Login Attempts** | 5 attempts | 15 minutes | Brute force protection |
| **Registration** | 10 attempts | 1 hour | Spam prevention |

**Response Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1699785660
```

**Rate Limit Exceeded:**
```http
HTTP/1.1 429 Too Many Requests
Retry-After: 45

{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Try again in 45 seconds."
  }
}
```

### CORS (Cross-Origin Resource Sharing)

**Configuration:**
```yaml
cors:
  allowed_origins:
    - https://app.pandora-exchange.com
    - https://admin.pandora-exchange.com
  allowed_methods:
    - GET
    - POST
    - PATCH
    - DELETE
  allowed_headers:
    - Authorization
    - Content-Type
  expose_headers:
    - X-RateLimit-Limit
    - X-RateLimit-Remaining
  max_age: 3600
  allow_credentials: true
```

### Security Headers

**Automatically Applied Headers:**

```http
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
Referrer-Policy: strict-origin-when-cross-origin
```

### Input Validation

**All inputs are validated:**
- Email format (RFC 5322)
- UUID format (RFC 4122)
- String length limits
- SQL injection prevention (parameterized queries via sqlc)
- XSS prevention (escaped output)
- JSON schema validation

**Example:**
```go
// Email validation
if !isValidEmail(email) {
    return ErrInvalidEmail
}

// UUID validation
if _, err := uuid.Parse(userID); err != nil {
    return ErrInvalidUserID
}
```

---

## Audit & Compliance

### Immutable Audit Logs

**All user actions are logged:**

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    details JSONB,
    ip_address TEXT,
    user_agent TEXT,
    trace_id UUID,
    created_at TIMESTAMP NOT NULL
);
```

**Logged Actions:**
- User registration
- Login attempts (success & failure)
- Password changes
- Profile updates
- KYC status changes
- Admin operations
- Logout events

**Log Entry Example:**
```json
{
  "id": "log-uuid",
  "user_id": "user-uuid",
  "action": "login",
  "resource": "auth",
  "details": {
    "success": true,
    "method": "password"
  },
  "ip_address": "192.168.1.1",
  "user_agent": "Mozilla/5.0...",
  "trace_id": "trace-uuid",
  "created_at": "2024-11-12T10:30:00Z"
}
```

### Retention Policies

| Environment | Retention Period | Cleanup Frequency |
|-------------|-----------------|-------------------|
| **dev** | 30 days | Daily |
| **sandbox** | 90 days | Daily |
| **audit** | 7 years (2,555 days) | Monthly |
| **prod** | 7 years (2,555 days) | Monthly |

**Automated Cleanup:**
```go
// Daily cleanup job
func CleanupExpiredAuditLogs(ctx context.Context, env string) {
    retentionDays := getRetentionDays(env)
    cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
    
    // Delete logs older than retention period
    db.Exec("DELETE FROM audit_logs WHERE created_at < $1", cutoffDate)
}
```

### Sensitive Data Redaction

**Never logged:**
- Passwords (plaintext or hashed)
- JWT tokens
- Refresh tokens
- Credit card numbers
- Social security numbers

**Redacted Example:**
```json
{
  "action": "update_profile",
  "details": {
    "email": "u***@example.com",  // Partially redacted
    "password": "[REDACTED]",
    "phone": "***-***-1234"
  }
}
```

---

## Network Security

### TLS/HTTPS

**Requirements:**

| Environment | TLS Version | Certificate |
|-------------|-------------|-------------|
| **dev** | Optional | Self-signed |
| **sandbox** | TLS 1.2+ | Let's Encrypt |
| **audit** | TLS 1.3 | Commercial CA |
| **prod** | TLS 1.3 | Commercial CA |

**Configuration:**
```yaml
server:
  tls:
    enabled: true
    cert_file: /etc/tls/cert.pem
    key_file: /etc/tls/key.pem
    min_version: "1.3"
    cipher_suites:
      - TLS_AES_128_GCM_SHA256
      - TLS_AES_256_GCM_SHA384
      - TLS_CHACHA20_POLY1305_SHA256
```

### Mutual TLS (mTLS)

**Internal Service Communication:**

```yaml
# gRPC server with mTLS
grpc:
  tls:
    enabled: true
    cert_file: /etc/tls/server-cert.pem
    key_file: /etc/tls/server-key.pem
    ca_file: /etc/tls/ca-cert.pem
    client_auth: require_and_verify_client_cert
```

**Planned for Phase 2** (Wallet Service, Trading Engine)

### Database Security

**PostgreSQL Security:**
- SSL/TLS encryption in transit
- Encrypted connections required
- Row-level security (RLS) for multi-tenancy
- Database user permissions (least privilege)
- Connection pooling with max limits

**Configuration:**
```yaml
database:
  sslmode: require
  sslcert: /etc/pgsql/client-cert.pem
  sslkey: /etc/pgsql/client-key.pem
  sslrootcert: /etc/pgsql/ca-cert.pem
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m
```

---

## Security Best Practices

### Development

**✅ DO:**
- Use `.env.dev` for local secrets (NOT committed to Git)
- Rotate secrets regularly
- Run linters and security scanners
- Write security tests
- Use prepared statements (sqlc does this)
- Validate all inputs
- Log security events
- Use HTTPS in all environments

**❌ DON'T:**
- Hardcode secrets in code
- Commit `.env` files to Git
- Use weak passwords
- Disable TLS in production
- Log sensitive data
- Use `SELECT *` queries
- Trust user input
- Skip security updates

### Code Review Checklist

- [ ] No hardcoded secrets
- [ ] All inputs validated
- [ ] Sensitive data redacted from logs
- [ ] Authentication required for protected endpoints
- [ ] Authorization checked (RBAC)
- [ ] SQL injection prevention (parameterized queries)
- [ ] XSS prevention (escaped output)
- [ ] CSRF protection (where applicable)
- [ ] Rate limiting configured
- [ ] Audit logging added
- [ ] Error messages don't leak information
- [ ] TLS/HTTPS enforced

---

## Incident Response

### Security Incident Procedure

**1. Detection**
- Monitor audit logs
- Alert on suspicious activity
- Track failed login attempts
- Monitor rate limit violations

**2. Assessment**
- Determine severity (Low/Medium/High/Critical)
- Identify affected users
- Assess data exposure
- Document timeline

**3. Containment**
- Revoke compromised tokens
- Block suspicious IPs
- Disable affected accounts
- Isolate affected services

**4. Eradication**
- Patch vulnerabilities
- Rotate compromised secrets
- Update firewall rules
- Deploy security fixes

**5. Recovery**
- Restore services
- Verify security measures
- Monitor for recurrence
- Notify affected users

**6. Post-Incident**
- Document lessons learned
- Update security policies
- Improve monitoring
- Conduct training

### Contact Information

**Security Team:**
- Email: security@pandora-exchange.com
- PGP Key: [Link to public key]
- Bug Bounty: [Link to program]

**Disclosure Policy:**
- Responsible disclosure: 90 days
- Critical vulnerabilities: Immediate patch
- Public disclosure: After fix deployed

---

## Compliance

### Standards & Regulations

**Compliance Frameworks:**
- GDPR (General Data Protection Regulation)
- PCI DSS (Payment Card Industry Data Security Standard)
- SOC 2 Type II
- ISO 27001

**Audit Requirements:**
- 7-year audit log retention
- Immutable log storage
- Regular security audits
- Penetration testing (annual)
- Vulnerability scanning (monthly)

---

## Related Documentation

- 🏗️ [Architecture Overview](../ARCHITECTURE.md)
- 📡 [API Documentation](./API_DOCUMENTATION.md)
- 🧪 [Testing Guide](./TESTING.md)
- 🚀 [Quick Start](./QUICK_START.md)
- 📜 [Audit Retention Policy](../AUDIT_RETENTION_POLICY.md)

---

**Last Updated:** November 12, 2025  
**Security Version:** 1.0  
**Next Security Audit:** December 2025
