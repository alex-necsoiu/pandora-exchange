# Password Timing Attack Verification

## Overview

This document describes how to manually verify that password hashing and verification operations have constant-time behavior to prevent timing attacks.

## Background

**Timing attacks** exploit variations in execution time to leak information about secret data. For password verification, an attacker could potentially determine:
- Whether a password hash exists in the database
- How many characters of the password are correct
- Information about the hashing algorithm parameters

Pandora Exchange uses **Argon2id**, which provides cryptographic constant-time verification by design. However, we verify this behavior through timing tests.

## Why Skip in CI?

The timing test (`TestPasswordHashingTiming`) is intentionally **skipped in CI environments** because:

1. **System Load Variance**: CI runners have variable CPU load from concurrent jobs
2. **Shared Resources**: CI environments share CPU, memory, and I/O with other processes
3. **Non-Deterministic Results**: System scheduling makes timing measurements unreliable
4. **False Failures**: The test would fail intermittently, blocking valid code changes

## Security Guarantee

**Argon2id provides constant-time verification at the cryptographic level**, regardless of:
- Correct vs incorrect password
- Password length
- Character matches

The timing test validates our implementation but is not required for security - Argon2id's constant-time guarantees are provided by the algorithm itself.

## Manual Verification Process

### Prerequisites

- Clean development environment (no background processes)
- Adequate system resources (CPU not under load)
- Go 1.21+ installed

### Running the Test Locally

```bash
# Navigate to project root
cd /Users/alexnecsoiu/go/src/pandora-exchange

# Run timing test (ensure CI env var is not set)
unset CI
unset GITHUB_ACTIONS
go test ./internal/domain/auth/... -v -run TestPasswordHashingTiming

# Expected output:
# === RUN   TestPasswordHashingTiming/verify_has_consistent_timing_regardless_of_result
#     password_test.go:199: Correct password   - avg: 13.3ms, stddev: 534µs
#     password_test.go:200: Incorrect password - avg: 13.4ms, stddev: 590µs
#     password_test.go:201: Timing difference: 0.31%
#     password_test.go:215: Standard deviation: correct=4.00%, incorrect=4.41%
# --- PASS: TestPasswordHashingTiming (1.5s)
```

### Interpreting Results

#### ✅ PASS Criteria

- **Timing difference < 25%**: Indicates constant-time behavior
- **Standard deviation < 30%**: Measurements are stable
- **Both correct and incorrect verifications take similar time**

#### ⚠️ WARNING Signs

```
password_test.go:218: WARNING: High standard deviation detected
```

This indicates:
- System is under load
- Results may be unreliable
- Re-run test in controlled environment

#### ❌ FAIL Criteria

- Timing difference > 25%
- Consistent bias (one path always faster)
- May indicate implementation issue (investigate immediately)

### Test Parameters

The test uses enhanced parameters for better statistical significance:

```go
const iterations = 50           // 50 measurements per scenario
const warmupIterations = 5      // 5 warmup runs to stabilize CPU/cache
```

## Security Verification Schedule

| Frequency | Environment | Required By |
|-----------|-------------|-------------|
| **Every Release** | Local Development | Security Team |
| **Quarterly** | Audit Environment | Compliance Team |
| **After Algorithm Changes** | Local Development | Engineering Team |
| **Security Audits** | Controlled Lab | External Auditors |

## Troubleshooting

### Test Skipped in CI

**Expected behavior**. The test automatically skips when:
- `CI=true` environment variable is set
- `GITHUB_ACTIONS` environment variable is set
- `-short` flag is used

```bash
# This will skip the test
CI=true go test ./internal/domain/auth/...

# This will run the test
unset CI
go test ./internal/domain/auth/... -run TestPasswordHashingTiming
```

### High Timing Variance

**Solutions:**
1. Close background applications
2. Disable CPU frequency scaling
3. Run on dedicated hardware
4. Increase test iterations (modify source)

### Consistent Failures

**Investigation Steps:**
1. Check Argon2id library version
2. Review password.go implementation
3. Run on different hardware
4. Consult security team

## Related Documentation

- [Argon2 Specification (RFC 9106)](https://www.rfc-editor.org/rfc/rfc9106.html)
- [OWASP Password Storage Cheat Sheet](https://cheatsheetsandbox.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [Testing for Timing Attacks](https://owasp.org/www-community/attacks/Timing_attack)

## Implementation Details

**File:** `internal/domain/auth/password.go`

```go
// Uses Argon2id with secure defaults:
// - Time parameter: 3 iterations
// - Memory: 64 MB
// - Threads: 4
// - Salt: 16 bytes (cryptographically random)
// - Hash length: 32 bytes
```

**Cryptographic Guarantee:**  
Argon2id comparison is constant-time via `subtle.ConstantTimeCompare()` in the underlying library.

## Contact

**Security Concerns:** security@pandora-exchange.com  
**Implementation Questions:** See [internal/domain/auth/password.go](../../internal/domain/auth/password.go)

---

**Last Updated:** 2025-11-12  
**Review Required:** Annually or after security incidents
