# GitHub Secrets Configuration

This document describes the GitHub secrets required for CI/CD workflows.

## Required Secrets

### Deployment Secrets

#### `KUBE_CONFIG_STAGING`
- **Purpose**: Kubernetes configuration for staging environment
- **Type**: base64-encoded kubeconfig file
- **Required for**: `.github/workflows/deploy.yml` (staging deployment)
- **Setup**:
  ```bash
  # Generate kubeconfig for staging cluster
  kubectl config view --minify --flatten > staging-kubeconfig.yaml
  
  # Base64 encode
  cat staging-kubeconfig.yaml | base64 | tr -d '\n'
  
  # Add to GitHub: Settings → Secrets → Actions → New repository secret
  # Name: KUBE_CONFIG_STAGING
  # Value: <base64-encoded-content>
  ```

#### `KUBE_CONFIG_PRODUCTION`
- **Purpose**: Kubernetes configuration for production environment
- **Type**: base64-encoded kubeconfig file
- **Required for**: `.github/workflows/deploy.yml` (production deployment)
- **Setup**: Same as `KUBE_CONFIG_STAGING` but for production cluster
- **Security**: 
  - Use service account with minimal permissions
  - Rotate credentials quarterly
  - Enable audit logging for all actions

### Optional Secrets

#### `CODECOV_TOKEN`
- **Purpose**: Upload test coverage to Codecov
- **Type**: API token
- **Required for**: `.github/workflows/ci.yml` (test coverage)
- **Setup**:
  1. Sign up at https://codecov.io
  2. Add repository
  3. Copy token
  4. Add to GitHub secrets as `CODECOV_TOKEN`

#### `SLACK_WEBHOOK_URL`
- **Purpose**: Send deployment notifications to Slack
- **Type**: Webhook URL
- **Required for**: `.github/workflows/deploy.yml` (notifications)
- **Setup**:
  1. Create Slack app: https://api.slack.com/apps
  2. Enable Incoming Webhooks
  3. Add webhook to channel
  4. Copy webhook URL
  5. Add to GitHub secrets as `SLACK_WEBHOOK_URL`

#### `GITLEAKS_LICENSE`
- **Purpose**: Gitleaks Pro license (optional)
- **Type**: License key
- **Required for**: `.github/workflows/security.yml` (secret scanning)
- **Setup**: Only needed for Gitleaks Pro features

## GitHub Container Registry (GHCR)

### `GITHUB_TOKEN`
- **Purpose**: Push Docker images to ghcr.io
- **Type**: Automatically provided by GitHub Actions
- **Required for**: `.github/workflows/deploy.yml` (Docker build)
- **Permissions**: Automatically granted, no manual setup needed
- **Configuration**:
  ```yaml
  # Already configured in deploy.yml
  - name: Login to GitHub Container Registry
    uses: docker/login-action@v3
    with:
      registry: ghcr.io
      username: ${{ github.actor }}
      password: ${{ secrets.GITHUB_TOKEN }}
  ```

## Kubernetes Service Account Setup

### Staging Environment

```yaml
# staging-service-account.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: github-actions
  namespace: pandora-staging
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: github-actions-deployer
  namespace: pandora-staging
rules:
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: [""]
    resources: ["pods", "services"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: github-actions-deployer
  namespace: pandora-staging
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: github-actions-deployer
subjects:
  - kind: ServiceAccount
    name: github-actions
    namespace: pandora-staging
```

### Production Environment

```yaml
# production-service-account.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: github-actions
  namespace: pandora-production
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: github-actions-deployer
  namespace: pandora-production
rules:
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "patch"]
  - apiGroups: [""]
    resources: ["pods", "services"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: github-actions-deployer
  namespace: pandora-production
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: github-actions-deployer
subjects:
  - kind: ServiceAccount
    name: github-actions
    namespace: pandora-production
```

### Generate Kubeconfig for Service Account

```bash
#!/bin/bash
# generate-kubeconfig.sh

NAMESPACE="pandora-staging"  # or pandora-production
SERVICE_ACCOUNT="github-actions"
CLUSTER_NAME="your-cluster-name"
SERVER="https://your-k8s-api-server:6443"

# Get service account token
SECRET_NAME=$(kubectl get serviceaccount $SERVICE_ACCOUNT -n $NAMESPACE -o jsonpath='{.secrets[0].name}')
TOKEN=$(kubectl get secret $SECRET_NAME -n $NAMESPACE -o jsonpath='{.data.token}' | base64 -d)
CA_CERT=$(kubectl get secret $SECRET_NAME -n $NAMESPACE -o jsonpath='{.data.ca\.crt}')

# Create kubeconfig
cat <<EOF > ${NAMESPACE}-kubeconfig.yaml
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: ${CA_CERT}
    server: ${SERVER}
  name: ${CLUSTER_NAME}
contexts:
- context:
    cluster: ${CLUSTER_NAME}
    namespace: ${NAMESPACE}
    user: ${SERVICE_ACCOUNT}
  name: ${SERVICE_ACCOUNT}@${CLUSTER_NAME}
current-context: ${SERVICE_ACCOUNT}@${CLUSTER_NAME}
users:
- name: ${SERVICE_ACCOUNT}
  user:
    token: ${TOKEN}
EOF

echo "Kubeconfig created: ${NAMESPACE}-kubeconfig.yaml"
echo ""
echo "Base64 encoded for GitHub secret:"
cat ${NAMESPACE}-kubeconfig.yaml | base64 | tr -d '\n'
echo ""
```

## Security Best Practices

### Credential Rotation

- **Kubernetes credentials**: Rotate every 90 days
- **API tokens**: Rotate every 180 days
- **Webhook URLs**: Rotate if compromised

### Access Control

- Use least-privilege principle
- Limit service account permissions to required namespaces
- Enable audit logging for all deployments
- Require 2FA for GitHub account with secret access

### Monitoring

- Monitor secret usage in GitHub Actions logs
- Alert on failed authentication attempts
- Track deployment success/failure rates
- Review audit logs monthly

## Verification

### Test Staging Deployment

```bash
# Verify staging kubeconfig works
echo $KUBE_CONFIG_STAGING | base64 -d > /tmp/staging-kubeconfig
export KUBECONFIG=/tmp/staging-kubeconfig
kubectl get pods -n pandora-staging
```

### Test Production Deployment

```bash
# Verify production kubeconfig works
echo $KUBE_CONFIG_PRODUCTION | base64 -d > /tmp/production-kubeconfig
export KUBECONFIG=/tmp/production-kubeconfig
kubectl get pods -n pandora-production
```

### Test GHCR Push

The `GITHUB_TOKEN` is automatically available. Test by running:

```bash
# Trigger workflow manually
gh workflow run deploy.yml --ref main
```

## Troubleshooting

### "Error: Forbidden"

**Problem**: Kubernetes service account lacks permissions

**Solution**:
```bash
# Check service account permissions
kubectl auth can-i --as=system:serviceaccount:pandora-staging:github-actions \
  patch deployment -n pandora-staging

# Update role if needed
kubectl apply -f staging-service-account.yaml
```

### "Error: Unauthorized"

**Problem**: Token expired or invalid

**Solution**:
```bash
# Regenerate service account token
kubectl delete secret $(kubectl get sa github-actions -n pandora-staging -o jsonpath='{.secrets[0].name}') -n pandora-staging
kubectl get sa github-actions -n pandora-staging -o yaml  # New secret auto-created

# Update GitHub secret with new token
```

### "Error: Cannot connect to cluster"

**Problem**: Server URL incorrect or cluster unreachable

**Solution**:
```bash
# Verify cluster connectivity
kubectl cluster-info

# Check server URL in kubeconfig
kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}'
```

## References

- [GitHub Actions Secrets Documentation](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [Kubernetes RBAC Documentation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [GitHub Container Registry Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Codecov Documentation](https://docs.codecov.com/docs)
