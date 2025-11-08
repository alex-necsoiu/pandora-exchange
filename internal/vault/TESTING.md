# Vault Integration Testing Guide

This guide describes how to run integration tests for the Vault client implementation.

## Overview

The Vault integration has two types of tests:

1. **Unit Tests** - Test individual functions with mocks/stubs (run by default)
2. **Integration Tests** - Test against a real Vault dev server (opt-in)

## Running Unit Tests

Unit tests run automatically and skip integration tests:

```bash
go test ./internal/vault/... -v
```

**Output:**
```
=== RUN   TestNewClient
=== PASS  TestNewClient (0.00s)
=== RUN   TestIsVaultAvailable
=== PASS  TestIsVaultAvailable (4.36s)
=== RUN   TestDisabledClient
=== PASS  TestDisabledClient (0.00s)
=== SKIP  TestVaultIntegration (0.00s)
PASS
ok      github.com/alex-necsoiu/pandora-exchange/internal/vault 4.597s
```

## Running Integration Tests

Integration tests require:
1. **Vault binary** installed and in PATH
2. **Environment variable** `VAULT_INTEGRATION_TESTS=true`

### Step 1: Install Vault

**macOS (Homebrew):**
```bash
brew tap hashicorp/tap
brew install hashicorp/tap/vault
```

**macOS (Manual Download):**
```bash
# Download from https://www.vaultproject.io/downloads
curl -O https://releases.hashicorp.com/vault/1.15.0/vault_1.15.0_darwin_arm64.zip
unzip vault_1.15.0_darwin_arm64.zip
sudo mv vault /usr/local/bin/
chmod +x /usr/local/bin/vault
```

**Linux:**
```bash
wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install vault
```

**Verify Installation:**
```bash
vault version
# Expected: Vault v1.15.0 or higher
```

### Step 2: Run Integration Tests

```bash
# Set environment variable and run tests
VAULT_INTEGRATION_TESTS=true go test ./internal/vault/... -v
```

**What happens:**
1. Test starts Vault dev server on `http://127.0.0.1:8200`
2. Creates test secrets using Vault CLI
3. Tests `NewClient`, `GetSecret`, `GetSecrets`, `IsAvailable`
4. Tests ENV fallback behavior
5. Tests error handling scenarios
6. Shuts down Vault dev server

**Expected Output:**
```
=== RUN   TestVaultIntegration
=== RUN   TestVaultIntegration/NewClient
=== RUN   TestVaultIntegration/IsAvailable
=== RUN   TestVaultIntegration/GetSecret
=== RUN   TestVaultIntegration/GetSecrets
=== RUN   TestVaultIntegration/GetSecretWithEnvFallback
=== RUN   TestVaultIntegration/ErrorHandling
--- PASS: TestVaultIntegration (5.12s)
    --- PASS: TestVaultIntegration/NewClient (0.00s)
    --- PASS: TestVaultIntegration/IsAvailable (0.01s)
    --- PASS: TestVaultIntegration/GetSecret (0.15s)
    --- PASS: TestVaultIntegration/GetSecrets (0.23s)
    --- PASS: TestVaultIntegration/GetSecretWithEnvFallback (0.18s)
    --- PASS: TestVaultIntegration/ErrorHandling (3.52s)
PASS
ok      github.com/alex-necsoiu/pandora-exchange/internal/vault 9.483s
```

### Step 3: Run With Coverage

```bash
VAULT_INTEGRATION_TESTS=true go test ./internal/vault/... -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

Open `coverage.html` in browser to see line-by-line coverage.

## Test Coverage

### Unit Tests

- ✅ `TestNewClient` - Client initialization with valid/invalid inputs
- ✅ `TestIsVaultAvailable` - Health check with unreachable Vault
- ✅ `TestDisabledClient` - Disabled client behavior and ENV fallback
- ⏭️ Skipped tests (require integration):
  - `TestGetSecret`
  - `TestGetSecrets`
  - `TestGetSecretWithFallback`
  - `TestErrorHandling`
  - `TestSecretsCache`

### Integration Tests (VAULT_INTEGRATION_TESTS=true)

- ✅ `TestVaultIntegration/NewClient` - Create client with Vault dev server
- ✅ `TestVaultIntegration/IsAvailable` - Check Vault availability
- ✅ `TestVaultIntegration/GetSecret` - Fetch single secret
  - Fetch existing secret
  - Handle non-existent secret
  - ENV fallback when secret missing
- ✅ `TestVaultIntegration/GetSecrets` - Batch fetch secrets
  - Fetch multiple secrets successfully
  - Handle partial failures
- ✅ `TestVaultIntegration/GetSecretWithEnvFallback` - Fallback logic
  - ENV fallback when Vault secret doesn't exist
  - Vault value takes precedence over ENV
  - Disabled client uses ENV
- ✅ `TestVaultIntegration/ErrorHandling` - Error scenarios
  - Invalid token authentication
  - Unreachable Vault server
  - Empty path/key validation
  - Context timeout

## Troubleshooting

### Issue: "vault: command not found"

**Solution:** Install Vault binary (see Step 1 above)

### Issue: "Failed to start Vault dev server"

**Cause:** Port 8200 already in use

**Solution:**
```bash
# Find process using port 8200
lsof -ti :8200

# Kill the process
lsof -ti :8200 | xargs kill -9

# Or use different port (requires test modification)
```

### Issue: "context deadline exceeded"

**Cause:** Vault dev server slow to start

**Solution:** Increase timeout in `integration_test.go`:
```go
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
```

### Issue: Tests hang indefinitely

**Cause:** Vault dev server didn't shut down

**Solution:**
```bash
# Kill all Vault processes
pkill -9 vault

# Re-run tests
VAULT_INTEGRATION_TESTS=true go test ./internal/vault/... -v
```

## CI/CD Integration

To run integration tests in CI pipelines:

**GitHub Actions:**
```yaml
- name: Install Vault
  run: |
    wget -O vault.zip https://releases.hashicorp.com/vault/1.15.0/vault_1.15.0_linux_amd64.zip
    unzip vault.zip
    sudo mv vault /usr/local/bin/
    vault version

- name: Run Integration Tests
  env:
    VAULT_INTEGRATION_TESTS: true
  run: go test ./internal/vault/... -v
```

**GitLab CI:**
```yaml
test:vault:integration:
  image: golang:1.23
  before_script:
    - apt-get update && apt-get install -y wget unzip
    - wget -O vault.zip https://releases.hashicorp.com/vault/1.15.0/vault_1.15.0_linux_amd64.zip
    - unzip vault.zip && mv vault /usr/local/bin/
  script:
    - VAULT_INTEGRATION_TESTS=true go test ./internal/vault/... -v
```

## Manual Testing with Vault Dev Server

For manual testing or debugging:

```bash
# Terminal 1: Start Vault dev server
vault server -dev -dev-root-token-id=dev-root-token

# Terminal 2: Set environment and test
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='dev-root-token'

# Write test secrets
vault kv put secret/pandora/user-service/jwt \
  secret="test-jwt-secret-key-min-32-characters-long"

vault kv put secret/pandora/user-service/database \
  password="test-db-password"

vault kv put secret/pandora/user-service/redis \
  password="test-redis-password"

# Read secrets to verify
vault kv get secret/pandora/user-service/jwt

# Run user service with Vault enabled
export VAULT_ENABLED=true
export VAULT_SECRET_PATH=secret/data/pandora/user-service
./bin/user-service
```

## Best Practices

1. **Run unit tests frequently** - Fast feedback during development
2. **Run integration tests before commit** - Verify Vault client works end-to-end
3. **Use Vault dev mode locally** - Never use production Vault for testing
4. **Clean up secrets** - Integration tests create test secrets, but they're ephemeral in dev mode
5. **Check test coverage** - Aim for >80% coverage on critical paths

## References

- [Vault Dev Server Documentation](https://www.vaultproject.io/docs/commands/server#dev)
- [Vault Testing Best Practices](https://www.vaultproject.io/docs/internals/testing)
- [Go Testing Package](https://pkg.go.dev/testing)

---

**Last Updated:** 2024-11-08  
**Maintainer:** Pandora Engineering Team
