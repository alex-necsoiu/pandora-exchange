# HashiCorp Vault Integration - Pandora Exchange

> **Security:** This document describes how Pandora Exchange integrates with HashiCorp Vault for secret management in production.

---

## Overview

Pandora Exchange uses [HashiCorp Vault](https://www.vaultproject.io/) to securely manage sensitive configuration such as:

- **JWT Signing Keys** - Used for token generation/validation
- **Database Passwords** - PostgreSQL credentials
- **Redis Passwords** - Redis authentication
- **Future Secrets** - API keys, HSM tokens, etc.

**Environment Strategy:**
- **Development**: Vault disabled, uses environment variables from `.env.dev`
- **Production**: Vault enabled, secrets injected via Vault Agent Sidecar in Kubernetes

---

## Architecture

### Local Development (Vault Disabled)

```
┌─────────────────┐
│  User Service   │
│                 │
│  Config Load:   │
│  1. Read ENV    │──► JWT_SECRET=dev-secret...
│  2. Skip Vault  │──► DB_PASSWORD=pandora_dev_secret
│  3. Validate    │──► REDIS_PASSWORD=""
│                 │
└─────────────────┘
```

**Configuration:**
```bash
VAULT_ENABLED=false  # Default in dev
JWT_SECRET=dev-secret-key-change-this-in-production-min-32-chars
DB_PASSWORD=pandora_dev_secret
REDIS_PASSWORD=""
```

### Production (Vault Enabled with Kubernetes)

```
┌──────────────────────────────────────────────────────────┐
│  Pod: user-service-xxxxx                                 │
│                                                           │
│  ┌─────────────────┐      ┌────────────────────────┐   │
│  │  Vault Agent    │      │   User Service         │   │
│  │  (Sidecar)      │      │                        │   │
│  │                 │      │  1. Load Config        │   │
│  │  1. Authenticate├─────►│  2. Init Vault Client  │   │
│  │  2. Fetch Secrets     │  3. LoadSecretsFrom... │   │
│  │  3. Write to /vault/  │  4. Validate & Start   │   │
│  │                 │      │                        │   │
│  └─────────────────┘      └────────────────────────┘   │
│         │                                                │
│         │ Periodic Renewal                               │
│         ▼                                                │
│  ┌─────────────────┐                                    │
│  │ Vault Server    │                                    │
│  │ (External)      │                                    │
│  └─────────────────┘                                    │
└──────────────────────────────────────────────────────────┘
```

**Flow:**
1. Pod starts with Vault Agent sidecar (via Vault Injector mutating webhook)
2. Vault Agent authenticates using Kubernetes ServiceAccount token
3. Vault Agent fetches secrets from configured paths
4. Vault Agent writes secrets to `/vault/secrets/` as environment files
5. User Service reads config, initializes Vault client
6. User Service calls `LoadSecretsFromVault()` to override ENV with Vault values
7. Vault Agent periodically renews secrets in the background

---

## Secret Paths in Vault

All Pandora secrets are stored under the KV v2 engine:

```
secret/
  data/
    pandora/
      user-service/
        jwt:
          secret: "<32+ character JWT key>"
        database:
          password: "<PostgreSQL password>"
        redis:
          password: "<Redis password>"
```

**Path Format:**
- **Mount Point**: `secret/` (KV v2 engine)
- **Base Path**: `pandora/user-service`
- **Secret Keys**: `jwt`, `database`, `redis`

**Example Vault CLI commands:**
```bash
# Write JWT secret
vault kv put secret/pandora/user-service/jwt \
  secret="your-super-secret-jwt-key-min-32-chars"

# Write database password
vault kv put secret/pandora/user-service/database \
  password="your-postgres-password"

# Write Redis password
vault kv put secret/pandora/user-service/redis \
  password="your-redis-password"

# Read secrets
vault kv get secret/pandora/user-service/jwt
vault kv get secret/pandora/user-service/database
vault kv get secret/pandora/user-service/redis
```

---

## Kubernetes Setup

### Prerequisites

1. **Vault Server** installed in Kubernetes cluster
2. **Vault Agent Injector** installed
3. **Kubernetes Auth** enabled in Vault
4. **ServiceAccount** for user-service

### 1. Install Vault via Helm

```bash
helm repo add hashicorp https://helm.releases.hashicorp.com
helm repo update

# Install Vault in dev mode (NOT for production)
helm install vault hashicorp/vault \
  --namespace default \
  --set "server.dev.enabled=true"

# For production: Use external Vault or HA mode
helm install vault hashicorp/vault \
  --namespace default \
  --values vault-values.yaml
```

### 2. Configure Kubernetes Auth

```bash
# Enable Kubernetes auth
vault auth enable kubernetes

# Configure Kubernetes auth
vault write auth/kubernetes/config \
  kubernetes_host="https://$KUBERNETES_PORT_443_TCP_ADDR:443"

# Create policy for user-service
vault policy write pandora-user-service - <<EOF
path "secret/data/pandora/user-service/*" {
  capabilities = ["read", "list"]
}
EOF

# Create Kubernetes role
vault write auth/kubernetes/role/pandora-user-service \
  bound_service_account_names=user-service \
  bound_service_account_namespaces=pandora \
  policies=pandora-user-service \
  ttl=24h
```

### 3. Deploy with Vault Annotations

The `user-service-deployment.yaml` already includes Vault Agent Injector annotations:

```yaml
annotations:
  vault.hashicorp.com/agent-inject: "true"
  vault.hashicorp.com/role: "pandora-user-service"
  vault.hashicorp.com/agent-inject-secret-jwt: "secret/data/pandora/user-service/jwt"
  vault.hashicorp.com/agent-inject-template-jwt: |
    {{- with secret "secret/data/pandora/user-service/jwt" -}}
    export JWT_SECRET="{{ .Data.data.secret }}"
    {{- end -}}
  # ... (database and redis annotations)
```

**What happens:**
- Vault Agent Injector mutating webhook intercepts pod creation
- Injects Vault Agent sidecar container
- Vault Agent fetches secrets and writes to `/vault/secrets/`
- User Service container can source these files

### 4. Deploy User Service

```bash
# Deploy to production (Vault enabled)
kubectl apply -k deployments/k8s/overlays/prod/

# Verify Vault Agent injection
kubectl get pod -n pandora -l app=user-service
# Should show 2/2 containers (user-service + vault-agent)

# Check Vault Agent logs
kubectl logs -n pandora user-service-xxxxx -c vault-agent

# Check User Service logs for Vault initialization
kubectl logs -n pandora user-service-xxxxx -c user-service | grep -i vault
```

---

## Configuration Reference

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `VAULT_ENABLED` | Yes | `false` | Enable/disable Vault integration |
| `VAULT_ADDR` | If enabled | `http://localhost:8200` | Vault server address |
| `VAULT_TOKEN` | If enabled | - | Vault authentication token (injected by Agent) |
| `VAULT_SECRET_PATH` | If enabled | `secret/data/pandora/user-service` | Base path for secrets |

### Fallback Behavior

**Vault Enabled + Vault Available:**
- ✅ Fetch secrets from Vault
- ✅ Override ENV variables with Vault values
- ❌ Fail if Vault fetch fails

**Vault Enabled + Vault Unavailable:**
- ⚠️ Log warning
- ✅ Fall back to environment variables
- ✅ Continue startup (for resilience)

**Vault Disabled (Dev Mode):**
- ✅ Use environment variables only
- ℹ️ Log "Using environment variable secrets (dev mode)"

---

## Testing

### Test Vault Integration Locally

1. **Run Vault Dev Server:**
```bash
# Terminal 1: Start Vault dev server
vault server -dev

# Terminal 2: Set Vault address
export VAULT_ADDR='http://127.0.0.1:8200'
export VAULT_TOKEN='<root token from terminal 1>'

# Write test secrets
vault kv put secret/pandora/user-service/jwt \
  secret="test-jwt-secret-key-min-32-characters-long"

vault kv put secret/pandora/user-service/database \
  password="test-db-password"

vault kv put secret/pandora/user-service/redis \
  password="test-redis-password"
```

2. **Run User Service with Vault:**
```bash
# Set environment to enable Vault
export VAULT_ENABLED=true
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=<root token>
export VAULT_SECRET_PATH=secret/data/pandora/user-service

# Run service
./bin/user-service
```

3. **Verify Secrets Loaded:**
```bash
# Check logs for:
# "Vault client initialized successfully"
# "Secrets loaded from Vault successfully"
```

### Integration Tests

Integration tests skip Vault by default. To test with Vault:

```bash
# Start Vault dev server first
vault server -dev

# Run integration tests with Vault
VAULT_ENABLED=true \
VAULT_ADDR=http://127.0.0.1:8200 \
VAULT_TOKEN=<root token> \
go test ./tests/integration/... -v
```

---

## Security Considerations

### ✅ Best Practices

1. **Never commit secrets** - All secrets in Vault, not in code/ENV
2. **Use specific Vault policies** - Least privilege access
3. **Rotate secrets regularly** - Vault supports dynamic secrets
4. **Enable audit logging** - Track all Vault access
5. **Use TLS in production** - `https://vault...` not `http://`
6. **Limit token TTL** - Short-lived tokens (24h max)
7. **Monitor Vault metrics** - Track authentication failures

### ⚠️ Common Pitfalls

1. **Hardcoded tokens** - Use Kubernetes auth, not static tokens
2. **Root tokens in prod** - Create service-specific policies
3. **Missing fallback** - Always test with Vault unavailable
4. **Insecure transport** - Never use HTTP Vault in production
5. **Broad policies** - Limit to exact secret paths needed

---

## Troubleshooting

### Pod fails to start: "Failed to initialize Vault client"

**Diagnosis:**
```bash
kubectl logs -n pandora user-service-xxxxx -c user-service
```

**Common causes:**
- Vault server unreachable (`VAULT_ADDR` incorrect)
- Invalid token (`VAULT_TOKEN` not set or expired)
- Network policy blocking Vault access

**Solution:**
1. Check Vault server status: `kubectl get pod -n default -l app=vault`
2. Verify Vault address: `kubectl get svc -n default vault`
3. Check NetworkPolicy allows Vault access
4. Test connectivity: `kubectl exec -it user-service-xxxxx -c user-service -- curl $VAULT_ADDR/v1/sys/health`

### Secrets not loading: "Failed to load secrets from Vault"

**Diagnosis:**
```bash
kubectl logs -n pandora user-service-xxxxx -c vault-agent
```

**Common causes:**
- Vault Agent not injected (check annotations)
- Authentication failed (check Kubernetes auth config)
- Secret path not found (check paths in Vault)
- Policy doesn't allow read (check Vault policy)

**Solution:**
1. Verify Agent injection: `kubectl describe pod user-service-xxxxx`
2. Check Vault Agent logs for auth errors
3. Test secret read manually:
   ```bash
   vault read secret/data/pandora/user-service/jwt
   ```
4. Verify policy:
   ```bash
   vault policy read pandora-user-service
   ```

### Vault Agent sidecar not injected

**Diagnosis:**
```bash
kubectl get mutatingwebhookconfiguration vault-agent-injector-cfg
```

**Common causes:**
- Vault Agent Injector not installed
- Webhook not configured
- Namespace label missing

**Solution:**
1. Install Vault with injector: `helm install vault --set "injector.enabled=true"`
2. Check webhook: `kubectl get mutatingwebhookconfiguration`
3. Verify namespace labels match webhook selectors

---

## Migration Guide

### Migrating from ENV to Vault

**Step 1: Store secrets in Vault**
```bash
vault kv put secret/pandora/user-service/jwt \
  secret="$JWT_SECRET"

vault kv put secret/pandora/user-service/database \
  password="$DB_PASSWORD"
```

**Step 2: Update deployment**
```yaml
# Add Vault annotations to deployment
metadata:
  annotations:
    vault.hashicorp.com/agent-inject: "true"
    # ... (see deployment.yaml)
```

**Step 3: Enable Vault in config**
```yaml
# In ConfigMap or overlay
data:
  vault_enabled: "true"
  vault_addr: "https://vault.default.svc.cluster.local:8200"
```

**Step 4: Deploy and verify**
```bash
kubectl apply -k deployments/k8s/overlays/prod/
kubectl logs -n pandora user-service-xxxxx | grep -i vault
```

**Step 5: Remove ENV secrets**
```bash
# Remove from Secret (keep only as fallback)
kubectl delete secret user-service-secrets
```

---

## Future Enhancements

- [ ] **Dynamic Database Credentials** - Vault generates short-lived DB credentials
- [ ] **Secret Rotation** - Automatic key rotation with zero downtime
- [ ] **HSM Integration** - Hardware security module for wallet operations
- [ ] **Transit Engine** - Encryption as a service for sensitive data
- [ ] **PKI Integration** - Certificate management for mTLS

---

## References

- [HashiCorp Vault Documentation](https://www.vaultproject.io/docs)
- [Vault Agent Injector](https://www.vaultproject.io/docs/platform/k8s/injector)
- [Kubernetes Auth Method](https://www.vaultproject.io/docs/auth/kubernetes)
- [KV Secrets Engine v2](https://www.vaultproject.io/docs/secrets/kv/kv-v2)
- [Vault Policies](https://www.vaultproject.io/docs/concepts/policies)

---

**Last Updated:** 2024-11-08  
**Maintainer:** Pandora Engineering Team
