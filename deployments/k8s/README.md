# 🚀 Pandora Exchange - Kubernetes Deployment Guide

> **Complete manual for deploying, maintaining, and updating Pandora Exchange User Service on Kubernetes**

---

## 📋 Table of Contents

1. [Prerequisites](#prerequisites)
2. [Architecture Overview](#architecture-overview)
3. [Initial Setup](#initial-setup)
4. [Deployment Instructions](#deployment-instructions)
5. [Environment Configuration](#environment-configuration)
6. [Maintenance Operations](#maintenance-operations)
7. [Monitoring & Observability](#monitoring--observability)
8. [Troubleshooting](#troubleshooting)
9. [Security Best Practices](#security-best-practices)
10. [Disaster Recovery](#disaster-recovery)
11. [CI/CD Integration](#cicd-integration)

---

## Prerequisites

### Required Tools

| Tool | Version | Installation |
|------|---------|--------------|
| kubectl | ≥ 1.28 | `brew install kubectl` or [Official Docs](https://kubernetes.io/docs/tasks/tools/) |
| kustomize | ≥ 5.0 | Built into `kubectl` (use `kubectl apply -k`) |
| helm | ≥ 3.12 | `brew install helm` |
| docker | ≥ 24.0 | [Docker Desktop](https://www.docker.com/products/docker-desktop/) |

### Kubernetes Cluster

You need access to a Kubernetes cluster. Options:

**Local Development:**
- **Minikube**: `brew install minikube && minikube start`
- **Kind** (Kubernetes in Docker): `brew install kind && kind create cluster`
- **Docker Desktop**: Enable Kubernetes in settings

**Cloud Providers:**
- **AWS EKS**: [Setup Guide](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html)
- **GCP GKE**: [Setup Guide](https://cloud.google.com/kubernetes-engine/docs/quickstart)
- **Azure AKS**: [Setup Guide](https://learn.microsoft.com/en-us/azure/aks/learn/quick-kubernetes-deploy-cli)

**Verify cluster access:**
```bash
kubectl cluster-info
kubectl get nodes
```

### Required Add-ons

#### 1. NGINX Ingress Controller

```bash
# Install NGINX Ingress Controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.4/deploy/static/provider/cloud/deploy.yaml

# Verify installation
kubectl get pods -n ingress-nginx
kubectl get svc -n ingress-nginx
```

#### 2. cert-manager (for HTTPS/TLS)

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml

# Verify installation
kubectl get pods -n cert-manager
```

#### 3. Metrics Server (for HPA)

```bash
# Install Metrics Server
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Verify installation
kubectl get deployment metrics-server -n kube-system
```

---

## Architecture Overview

### Kubernetes Resources

```
pandora namespace
├── Deployment: user-service (3-10 pods via HPA)
├── StatefulSet: postgres (1 pod with persistent storage)
├── Deployment: redis (1 pod with persistent storage)
├── Services:
│   ├── user-service (ClusterIP - gRPC)
│   ├── user-service-external (LoadBalancer - REST API)
│   ├── postgres (Headless)
│   └── redis (ClusterIP)
├── Ingress: user-service-ingress (HTTPS routing)
├── HPA: user-service-hpa (auto-scaling)
├── ConfigMap: user-service-config
├── Secrets: user-service-secrets, postgres-secrets
└── NetworkPolicies: (security)
```

### Network Flow

```
Internet → Ingress (HTTPS) → user-service-external (LoadBalancer) → user-service pods (HTTP)
                                                                    ↓
Internal Services (gRPC) → user-service (ClusterIP) → user-service pods
                                                                    ↓
                                                        PostgreSQL (StatefulSet)
                                                                    ↓
                                                        Redis (Deployment)
```

---

## Initial Setup

### 1. Clone Repository

```bash
git clone https://github.com/alex-necsoiu/pandora-exchange.git
cd pandora-exchange/deployments/k8s
```

### 2. Create Namespace

```bash
kubectl apply -f base/namespace.yaml
```

**Verify:**
```bash
kubectl get namespace pandora
```

### 3. Label Namespaces (for NetworkPolicy)

```bash
# Label kube-system for DNS
kubectl label namespace kube-system name=kube-system

# Label ingress-nginx namespace
kubectl label namespace ingress-nginx name=ingress-nginx
```

---

## Deployment Instructions

### Development Environment

**Step 1: Build Docker Image**

```bash
# From repository root
cd /path/to/pandora-exchange

# Build Docker image
docker build -t pandora/user-service:dev-latest -f deployments/docker/Dockerfile .

# For Minikube, load image into cluster
minikube image load pandora/user-service:dev-latest

# For Kind
kind load docker-image pandora/user-service:dev-latest
```

**Step 2: Deploy to Kubernetes**

```bash
cd deployments/k8s

# Deploy all resources for dev environment
kubectl apply -k overlays/dev

# Or manually apply each file
kubectl apply -f base/namespace.yaml
kubectl apply -f base/user-service-configmap.yaml
kubectl apply -f base/user-service-secret.yaml
kubectl apply -f base/postgres-statefulset.yaml
kubectl apply -f base/redis-deployment.yaml
kubectl apply -f base/user-service-deployment.yaml
kubectl apply -f base/user-service-service.yaml
kubectl apply -f base/user-service-hpa.yaml
kubectl apply -f base/network-policy.yaml
```

**Step 3: Run Database Migrations**

```bash
# Get a shell into user-service pod
kubectl exec -it -n pandora deployment/user-service-dev -- sh

# Inside the pod, run migrations
/app/user-service migrate up

# Or use a Job (recommended)
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate-db
  namespace: pandora
spec:
  template:
    spec:
      containers:
      - name: migrate
        image: pandora/user-service:dev-latest
        command: ["/app/user-service", "migrate", "up"]
        envFrom:
        - configMapRef:
            name: user-service-config-dev
        - secretRef:
            name: user-service-secrets-dev
      restartPolicy: Never
  backoffLimit: 3
EOF

# Check migration job status
kubectl logs -n pandora job/migrate-db
```

**Step 4: Verify Deployment**

```bash
# Check all pods are running
kubectl get pods -n pandora

# Check services
kubectl get svc -n pandora

# Check HPA status
kubectl get hpa -n pandora

# Check ingress
kubectl get ingress -n pandora
```

**Step 5: Access the Service**

```bash
# Port-forward for local access
kubectl port-forward -n pandora svc/user-service-dev 8080:8080

# Test health endpoint
curl http://localhost:8080/health

# Test API
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "SecurePass123!",
    "first_name": "Test",
    "last_name": "User"
  }'
```

### Staging Environment

```bash
cd deployments/k8s

# Deploy staging environment
kubectl apply -k overlays/staging

# Check deployment status
kubectl get pods -n pandora -l environment=staging

# Run migrations
kubectl exec -it -n pandora deployment/user-service-staging -- /app/user-service migrate up

# Port-forward to test
kubectl port-forward -n pandora svc/user-service-staging 8080:8080
```

### Production Environment

**Prerequisites:**
- DNS configured (api.pandora.exchange → LoadBalancer IP)
- SSL certificate ready (or use cert-manager)
- Secrets stored in Vault (Task 22)
- Database backups configured
- Monitoring setup (Prometheus/Grafana)

**Deployment Steps:**

```bash
cd deployments/k8s

# IMPORTANT: Review all production patches
cat overlays/prod/deployment-patch.yaml
cat overlays/prod/kustomization.yaml

# Dry-run to preview changes
kubectl apply -k overlays/prod --dry-run=client

# Deploy to production
kubectl apply -k overlays/prod

# Monitor rollout
kubectl rollout status deployment/user-service-prod -n pandora

# Check pod health
kubectl get pods -n pandora -l environment=production -w

# Run migrations (use a Job)
kubectl apply -f migrations-job-prod.yaml

# Verify services are reachable
kubectl get svc -n pandora | grep prod
kubectl get ingress -n pandora
```

**DNS Configuration:**

```bash
# Get LoadBalancer external IP
kubectl get svc user-service-external-prod -n pandora -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# OR for AWS ELB
kubectl get svc user-service-external-prod -n pandora -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'

# Create DNS A record:
# api.pandora.exchange → <EXTERNAL_IP>
```

**SSL Certificate (via cert-manager):**

```bash
# Check certificate status
kubectl get certificate -n pandora

# Check certificate details
kubectl describe certificate pandora-tls -n pandora

# Force certificate renewal
kubectl delete certificate pandora-tls -n pandora
kubectl apply -k overlays/prod
```

---

## Environment Configuration

### ConfigMap Updates

**Update non-sensitive configuration:**

```bash
# Edit ConfigMap
kubectl edit configmap user-service-config-prod -n pandora

# Or patch specific values
kubectl patch configmap user-service-config-prod -n pandora \
  --type merge \
  -p '{"data":{"audit_retention_days":"365"}}'

# Restart pods to pick up changes
kubectl rollout restart deployment/user-service-prod -n pandora
```

### Secret Management

**⚠️ IMPORTANT: Never commit secrets to Git!**

**Development Secrets (Base64 encoded):**

```bash
# Encode a secret
echo -n 'my-secret-value' | base64

# Create/update secret
kubectl create secret generic user-service-secrets-dev \
  --from-literal=db_password='pandora_dev_secret' \
  --from-literal=jwt_secret='dev-secret-key-min-32-chars' \
  --from-literal=redis_password='' \
  -n pandora \
  --dry-run=client -o yaml | kubectl apply -f -
```

**Production Secrets (Use Vault - Task 22):**

```bash
# For now, use sealed secrets or manually create
kubectl create secret generic user-service-secrets-prod \
  --from-literal=db_password='<strong-password>' \
  --from-literal=jwt_secret='<64-char-random-string>' \
  --from-literal=redis_password='<redis-password>' \
  -n pandora
```

### Environment Variables

**Override environment variables:**

```bash
# Edit deployment
kubectl edit deployment user-service-prod -n pandora

# Or patch
kubectl set env deployment/user-service-prod -n pandora \
  AUDIT_RETENTION_DAYS=2555
```

---

## Maintenance Operations

### Scaling

**Manual Scaling:**

```bash
# Scale to 5 replicas
kubectl scale deployment user-service-prod -n pandora --replicas=5

# Verify scaling
kubectl get pods -n pandora -l app=user-service
```

**Auto-scaling (HPA):**

```bash
# Check HPA status
kubectl get hpa user-service-hpa-prod -n pandora

# Describe HPA for metrics
kubectl describe hpa user-service-hpa-prod -n pandora

# Edit HPA thresholds
kubectl edit hpa user-service-hpa-prod -n pandora
```

### Rolling Updates

**Update Docker Image:**

```bash
# Build new version
docker build -t pandora/user-service:v1.1.0 .
docker push pandora/user-service:v1.1.0

# Update deployment
kubectl set image deployment/user-service-prod user-service=pandora/user-service:v1.1.0 -n pandora

# Monitor rollout
kubectl rollout status deployment/user-service-prod -n pandora

# Check rollout history
kubectl rollout history deployment/user-service-prod -n pandora
```

**Rollback Deployment:**

```bash
# Rollback to previous version
kubectl rollout undo deployment/user-service-prod -n pandora

# Rollback to specific revision
kubectl rollout undo deployment/user-service-prod -n pandora --to-revision=2

# Verify rollback
kubectl rollout status deployment/user-service-prod -n pandora
```

### Database Maintenance

**Backup PostgreSQL:**

```bash
# Create a backup
kubectl exec -n pandora statefulset/postgres -- pg_dump -U pandora pandora_prod > backup-$(date +%Y%m%d).sql

# Or use a CronJob for automated backups
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: pandora
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:15-alpine
            command:
            - /bin/sh
            - -c
            - pg_dump -h postgres -U pandora pandora_prod | gzip > /backup/backup-\$(date +%Y%m%d-%H%M%S).sql.gz
            env:
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secrets-prod
                  key: postgres_password
            volumeMounts:
            - name: backup
              mountPath: /backup
          volumes:
          - name: backup
            persistentVolumeClaim:
              claimName: postgres-backup-pvc
          restartPolicy: OnFailure
EOF
```

**Restore PostgreSQL:**

```bash
# Copy backup to pod
kubectl cp backup-20250108.sql pandora/postgres-0:/tmp/

# Restore
kubectl exec -it -n pandora statefulset/postgres -- psql -U pandora -d pandora_prod < /tmp/backup-20250108.sql
```

### Log Management

**View Logs:**

```bash
# Logs from all user-service pods
kubectl logs -n pandora -l app=user-service --tail=100 -f

# Logs from specific pod
kubectl logs -n pandora user-service-prod-xyz123 -f

# Logs from previous pod (after crash)
kubectl logs -n pandora user-service-prod-xyz123 --previous

# Export logs
kubectl logs -n pandora -l app=user-service --since=24h > user-service-logs.txt
```

**Centralized Logging (ELK Stack):**

```bash
# Install Elasticsearch, Fluentd, Kibana (ELK)
# See: https://www.elastic.co/guide/en/cloud-on-k8s/current/k8s-deploy-eck.html
```

---

## Monitoring & Observability

### Health Checks

**Check Service Health:**

```bash
# Health endpoint
kubectl run curl --image=curlimages/curl -i --rm --restart=Never -- \
  curl http://user-service.pandora.svc.cluster.local:8080/health

# Readiness probe status
kubectl get pods -n pandora -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}'
```

### Resource Usage

**Check Resource Consumption:**

```bash
# Pod CPU/memory usage
kubectl top pods -n pandora

# Node usage
kubectl top nodes

# Detailed pod metrics
kubectl describe pod user-service-prod-xyz123 -n pandora | grep -A 5 "Requests\|Limits"
```

### Prometheus Metrics

**Setup Prometheus:**

```bash
# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack -n observability --create-namespace

# Verify installation
kubectl get pods -n observability
```

**ServiceMonitor for User Service:**

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: user-service-monitor
  namespace: pandora
spec:
  selector:
    matchLabels:
      app: user-service
  endpoints:
  - port: http
    path: /metrics
```

### Grafana Dashboards

```bash
# Get Grafana password
kubectl get secret -n observability prometheus-grafana -o jsonpath="{.data.admin-password}" | base64 --decode

# Port-forward Grafana
kubectl port-forward -n observability svc/prometheus-grafana 3000:80

# Access: http://localhost:3000
# Import dashboards for Go applications
```

---

## Troubleshooting

### Common Issues

#### 1. Pods Not Starting

**Diagnosis:**

```bash
# Check pod status
kubectl get pods -n pandora

# Describe pod for events
kubectl describe pod user-service-prod-xyz123 -n pandora

# Check logs
kubectl logs user-service-prod-xyz123 -n pandora
```

**Common Causes:**
- **ImagePullBackOff**: Image doesn't exist or registry auth failed
  ```bash
  # Check image name
  kubectl get pod user-service-prod-xyz123 -n pandora -o jsonpath='{.spec.containers[0].image}'
  ```
- **CrashLoopBackOff**: Application crashes on startup
  ```bash
  # Check previous logs
  kubectl logs user-service-prod-xyz123 -n pandora --previous
  ```
- **Pending**: Insufficient resources
  ```bash
  # Check node resources
  kubectl describe nodes
  ```

#### 2. Database Connection Failures

**Diagnosis:**

```bash
# Check PostgreSQL pod
kubectl get pods -n pandora -l app=postgres

# Test connection from user-service pod
kubectl exec -it -n pandora deployment/user-service-prod -- sh
# Inside pod:
psql -h postgres.pandora.svc.cluster.local -U pandora -d pandora_prod
```

**Solutions:**
- Verify ConfigMap has correct DB host
- Check Secret has correct password
- Ensure NetworkPolicy allows traffic

#### 3. Ingress Not Working

**Diagnosis:**

```bash
# Check Ingress
kubectl get ingress -n pandora
kubectl describe ingress user-service-ingress -n pandora

# Check NGINX Ingress Controller
kubectl get pods -n ingress-nginx
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller
```

**Solutions:**
- Verify DNS points to LoadBalancer IP
- Check cert-manager created certificate
- Ensure Service selector matches Deployment labels

#### 4. HPA Not Scaling

**Diagnosis:**

```bash
# Check HPA status
kubectl get hpa -n pandora
kubectl describe hpa user-service-hpa-prod -n pandora

# Check metrics server
kubectl get deployment metrics-server -n kube-system
kubectl logs -n kube-system deployment/metrics-server
```

**Solutions:**
- Ensure metrics-server is running
- Verify resource requests are set in Deployment
- Check HPA has correct target reference

### Debug Commands

```bash
# Get all resources in namespace
kubectl get all -n pandora

# Events in namespace
kubectl get events -n pandora --sort-by='.lastTimestamp'

# Shell into pod
kubectl exec -it -n pandora deployment/user-service-prod -- sh

# Test DNS resolution
kubectl run dnsutils --image=tutum/dnsutils -it --rm --restart=Never -- nslookup postgres.pandora.svc.cluster.local

# Network debugging
kubectl run netshoot --image=nicolaka/netshoot -it --rm --restart=Never -- sh
```

---

## Security Best Practices

### 1. RBAC (Role-Based Access Control)

**Create ServiceAccount:**

```bash
kubectl create serviceaccount user-service-sa -n pandora
```

**Bind Role:**

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: user-service-role
  namespace: pandora
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: user-service-rolebinding
  namespace: pandora
subjects:
- kind: ServiceAccount
  name: user-service-sa
roleRef:
  kind: Role
  name: user-service-role
  apiGroup: rbac.authorization.k8s.io
```

### 2. Network Policies

Already implemented in `base/network-policy.yaml`. Verify:

```bash
# Check NetworkPolicy
kubectl get networkpolicy -n pandora

# Test connectivity
kubectl run test-pod --image=busybox -it --rm --restart=Never -n pandora -- wget -O- http://user-service:8080/health
```

### 3. Pod Security Standards

```yaml
# Add to namespace
apiVersion: v1
kind: Namespace
metadata:
  name: pandora
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### 4. Secrets Encryption

Enable encryption at rest in etcd:

```bash
# For managed clusters, this is usually enabled by default
# For self-managed clusters, configure encryption provider
```

---

## Disaster Recovery

### Backup Strategy

**1. Database Backups:**
- Daily automated backups via CronJob
- Store backups in S3/GCS with 30-day retention
- Test restore monthly

**2. Configuration Backups:**

```bash
# Export all resources
kubectl get all,configmap,secret,ingress,pvc,networkpolicy -n pandora -o yaml > pandora-backup.yaml

# Export to Git (recommended)
git add deployments/k8s/
git commit -m "chore: backup k8s manifests"
git push
```

### Restore Procedure

**1. Restore Namespace:**

```bash
kubectl apply -f base/namespace.yaml
```

**2. Restore Secrets & ConfigMaps:**

```bash
kubectl apply -f pandora-backup.yaml
```

**3. Restore Database:**

```bash
# Copy backup to pod
kubectl cp backup.sql pandora/postgres-0:/tmp/

# Restore
kubectl exec -it -n pandora postgres-0 -- psql -U pandora -d pandora_prod -f /tmp/backup.sql
```

**4. Restore Application:**

```bash
kubectl apply -k overlays/prod
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Deploy to Kubernetes

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Build Docker Image
      run: |
        docker build -t pandora/user-service:${{ github.sha }} .
        docker tag pandora/user-service:${{ github.sha }} pandora/user-service:latest
    
    - name: Push to Registry
      run: |
        echo ${{ secrets.DOCKER_PASSWORD }} | docker login -u ${{ secrets.DOCKER_USERNAME }} --password-stdin
        docker push pandora/user-service:${{ github.sha }}
        docker push pandora/user-service:latest
    
    - name: Deploy to Kubernetes
      run: |
        echo "${{ secrets.KUBECONFIG }}" > kubeconfig
        export KUBECONFIG=kubeconfig
        
        cd deployments/k8s
        kubectl apply -k overlays/prod
        kubectl set image deployment/user-service-prod user-service=pandora/user-service:${{ github.sha }} -n pandora
        kubectl rollout status deployment/user-service-prod -n pandora
```

---

## Quick Reference

### Essential Commands

```bash
# Deploy
kubectl apply -k overlays/dev|staging|prod

# Scale
kubectl scale deployment/user-service-prod --replicas=5 -n pandora

# Update image
kubectl set image deployment/user-service-prod user-service=pandora/user-service:v1.1.0 -n pandora

# Rollback
kubectl rollout undo deployment/user-service-prod -n pandora

# Logs
kubectl logs -n pandora -l app=user-service -f

# Shell access
kubectl exec -it -n pandora deployment/user-service-prod -- sh

# Port-forward
kubectl port-forward -n pandora svc/user-service 8080:8080

# Delete everything
kubectl delete namespace pandora
```

### Useful Aliases

Add to `~/.bashrc` or `~/.zshrc`:

```bash
alias k='kubectl'
alias kgp='kubectl get pods -n pandora'
alias kgs='kubectl get svc -n pandora'
alias kl='kubectl logs -n pandora -f'
alias kd='kubectl describe -n pandora'
alias kx='kubectl exec -it -n pandora'
```

---

## Support & Documentation

- **Kubernetes Docs**: https://kubernetes.io/docs/
- **kubectl Cheat Sheet**: https://kubernetes.io/docs/reference/kubectl/cheatsheet/
- **Kustomize Guide**: https://kustomize.io/
- **NGINX Ingress**: https://kubernetes.github.io/ingress-nginx/
- **cert-manager**: https://cert-manager.io/docs/

---

**Last Updated**: November 8, 2025  
**Maintained By**: Pandora Exchange DevOps Team  
**Version**: 1.0.0
