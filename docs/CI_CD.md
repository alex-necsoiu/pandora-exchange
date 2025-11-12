# CI/CD Pipeline Documentation

## Overview

Comprehensive CI/CD automation for the Pandora Exchange user service, featuring continuous integration, security scanning, and automated deployments with canary strategy.

## Workflows

### 1. Continuous Integration (`ci.yml`)

**Triggers**: Push to any branch, all pull requests

**Jobs** (10 total):
- **lint**: golangci-lint with 5-minute timeout
- **vet**: Go static analysis
- **security**: gosec security scanning with SARIF upload
- **imports-check**: Clean architecture boundary validation
- **test**: Full test suite with race detector, PostgreSQL + Redis services
- **build**: Binary compilation with version injection
- **migrations**: Database migration verification
- **codegen**: Verify sqlc and protobuf generation is current
- **dependencies**: Verify go.mod/go.sum are tidy
- **ci-success**: Gate job requiring all others to pass

**Features**:
- Race condition detection (`-race` flag)
- Code coverage with Codecov integration
- Security scanning with GitHub Code Scanning
- Clean architecture enforcement
- Parallel job execution for speed

**Runtime**: ~8-10 minutes

### 2. Deployment Pipeline (`deploy.yml`)

**Triggers**: 
- Push to tags matching `v*.*.*` (e.g., v1.2.3)
- Manual workflow dispatch

**Jobs** (5 total):

#### `build-image`
- Multi-platform Docker builds (linux/amd64, linux/arm64)
- Push to GitHub Container Registry (ghcr.io)
- Semantic versioning tags: `latest`, `v1.2.3`, `v1.2`, `v1`
- Build args injection: `VERSION`, `COMMIT`, `BUILD_TIME`
- SBOM (Software Bill of Materials) generation with Anchore
- GitHub Actions cache for faster builds

#### `deploy-staging`
- Kubectl deployment to staging namespace
- 5-minute rollout timeout
- Health check validation (`/health`)
- Pod status verification
- Smoke tests

#### `deploy-production`
- **Canary Deployment Strategy**:
  1. Backup current deployment
  2. Update deployment with new image
  3. Monitor with 20% traffic for 5 minutes
  4. Run smoke tests (`/health`, `/health/services`)
  5. Complete rollout to 100% if healthy
- GitHub Release creation with SBOM attachment

#### `rollback-production`
- Automatic on deployment failure
- Kubectl rollout undo
- Health verification
- Logging and notification

#### `notify`
- Aggregates deployment status
- Placeholder for Slack/Discord/Email notifications
- Runs on completion (success or failure)

**Features**:
- Canary deployment reduces blast radius
- Automatic rollback on failures
- Health check validation at each stage
- Deployment backups for safety
- SBOM for supply chain security
- Comprehensive logging

**Runtime**: ~15-20 minutes (including canary monitoring)

### 3. Security Scanning (`security.yml`)

**Triggers**:
- Push to main/develop
- All pull requests
- Daily schedule (2 AM UTC)
- Manual workflow dispatch

**Jobs** (9 total):

#### `dependency-review`
- Reviews dependency changes in PRs
- Fails on moderate+ severity vulnerabilities
- Denies GPL-2.0, GPL-3.0 licenses

#### `trivy-scan`
- Filesystem vulnerability scanning
- Configuration file scanning
- SARIF results to GitHub Security tab

#### `govulncheck`
- Go-specific vulnerability database check
- Uses official Go vulnerability database
- SARIF output for integration

#### `secret-scan`
- Gitleaks secret detection
- Scans entire commit history
- Prevents credential leaks

#### `codeql`
- Static code analysis
- Security and quality queries
- Automatic build detection

#### `semgrep`
- SAST (Static Application Security Testing)
- Auto-config for Go best practices
- Pattern-based security detection

#### `license-check`
- Uses google/go-licenses
- Generates license report
- Fails on forbidden licenses (GPL, AGPL)

#### `docker-scan`
- Builds Docker image for scanning
- Trivy image vulnerability scanning
- Grype (Anchore) vulnerability scanning

#### `scorecard`
- OpenSSF Scorecard analysis
- Security best practices scoring
- Supply chain security assessment

**Features**:
- Multiple scanning tools for comprehensive coverage
- SARIF integration with GitHub Security
- Daily automated scans
- License compliance enforcement
- Container security validation

**Runtime**: ~10-12 minutes

### 4. Dependency Updates (`dependabot.yml`)

**Ecosystems**:
- **Go modules**: Weekly Monday 3 AM UTC
- **GitHub Actions**: Weekly Monday 3 AM UTC
- **Docker**: Weekly Monday 3 AM UTC

**Features**:
- Grouped minor/patch updates
- Ignores major version updates (breaking changes)
- Automatic security vulnerability PRs
- Conventional commit messages
- Labeled for easy filtering

**Limits**:
- Go modules: 10 open PRs max
- GitHub Actions: 5 open PRs max
- Docker: 5 open PRs max

## Pre-Commit Hooks (`.pre-commit-config.yaml`)

**Purpose**: Catch issues before they reach CI

**Hooks**:
- **Code Quality**: go-fmt, go-vet, golangci-lint
- **Security**: gitleaks secret detection
- **Formatting**: trailing whitespace, end-of-file, line endings
- **Validation**: YAML, JSON syntax
- **SQL**: sqlfluff linting for migrations
- **Markdown**: markdownlint with auto-fix
- **Commits**: Conventional commit message validation
- **Project-Specific**:
  - sqlc generation verification
  - protobuf generation verification
  - tests on changed files
  - import boundary checks
  - migration pair verification

**Setup**:
```bash
# Automated setup
./scripts/setup-pre-commit.sh

# Manual setup
pip3 install pre-commit
pre-commit install
pre-commit install --hook-type commit-msg
pre-commit run --all-files
```

**Commit Message Format**:
```
<type>(<scope>): <description>

Types: feat, fix, docs, style, refactor, test, chore
Example: feat(auth): add JWT token rotation
```

## Secrets Configuration

See `.github/SECRETS.md` for detailed setup instructions.

### Required Secrets

| Secret | Purpose | Workflow |
|--------|---------|----------|
| `KUBE_CONFIG_STAGING` | Staging cluster access | deploy.yml |
| `KUBE_CONFIG_PRODUCTION` | Production cluster access | deploy.yml |

### Optional Secrets

| Secret | Purpose | Workflow |
|--------|---------|----------|
| `CODECOV_TOKEN` | Coverage reporting | ci.yml |
| `SLACK_WEBHOOK_URL` | Deployment notifications | deploy.yml |
| `GITLEAKS_LICENSE` | Gitleaks Pro features | security.yml |

### Auto-Provided

| Secret | Purpose | Workflow |
|--------|---------|----------|
| `GITHUB_TOKEN` | GHCR push, releases | deploy.yml |

## Deployment Process

### Staging Deployment

**Automatic on tag push:**
```bash
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

**Workflow**:
1. Build multi-platform Docker image
2. Push to ghcr.io with version tags
3. Generate SBOM
4. Deploy to staging namespace
5. Wait for rollout completion (5 min timeout)
6. Run smoke tests
7. Verify pod health

### Production Deployment

**Continues from staging if successful**

**Canary Strategy**:
1. Backup current production deployment
2. Update deployment with new image
3. **Canary Phase** (5 minutes):
   - 20% of pods run new version
   - 80% run old version
   - Monitor health checks
   - Run smoke tests
4. **Complete Rollout**:
   - If canary healthy → roll out to 100%
   - If canary fails → automatic rollback
5. Create GitHub Release with SBOM

**Monitoring During Canary**:
```bash
# Watch pod status
kubectl get pods -n pandora-production -w

# Check logs
kubectl logs -n pandora-production -l app=user-service --tail=100

# Verify health
curl https://production.example.com/health
curl https://production.example.com/health/services
```

### Manual Rollback

If automatic rollback fails:
```bash
# Get rollout history
kubectl rollout history deployment/user-service -n pandora-production

# Rollback to previous version
kubectl rollout undo deployment/user-service -n pandora-production

# Rollback to specific revision
kubectl rollout undo deployment/user-service -n pandora-production --to-revision=2

# Verify rollback
kubectl rollout status deployment/user-service -n pandora-production
```

## GitHub Container Registry

**Image Naming**:
```
ghcr.io/alex-necsoiu/pandora-exchange/user-service:latest
ghcr.io/alex-necsoiu/pandora-exchange/user-service:v1.2.3
ghcr.io/alex-necsoiu/pandora-exchange/user-service:v1.2
ghcr.io/alex-necsoiu/pandora-exchange/user-service:v1
```

**Pull Images**:
```bash
# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Pull specific version
docker pull ghcr.io/alex-necsoiu/pandora-exchange/user-service:v1.2.3

# Pull latest
docker pull ghcr.io/alex-necsoiu/pandora-exchange/user-service:latest
```

## Monitoring & Alerting

### GitHub Actions

- **Status Badges**: Add to README.md
  ```markdown
  ![CI](https://github.com/alex-necsoiu/pandora-exchange/workflows/CI/badge.svg)
  ![Deploy](https://github.com/alex-necsoiu/pandora-exchange/workflows/Deploy/badge.svg)
  ![Security](https://github.com/alex-necsoiu/pandora-exchange/workflows/Security%20Scanning/badge.svg)
  ```

- **Email Notifications**: Configure in GitHub account settings
- **Slack/Discord**: Add webhook URL to secrets and update notify job

### Security Tab

- **Code Scanning**: SARIF results from security.yml
- **Dependabot Alerts**: Automatic vulnerability notifications
- **Secret Scanning**: GitHub built-in secret detection

### Deployment Verification

**Health Endpoints**:
```bash
# Basic health
curl https://staging.example.com/health

# Service dependencies (DB, Redis, Vault)
curl https://staging.example.com/health/services
```

**Smoke Tests**:
```bash
# User registration
curl -X POST https://staging.example.com/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Test123!@#"}'

# Login
curl -X POST https://staging.example.com/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"Test123!@#"}'
```

## Best Practices

### Branching Strategy

- **main**: Production-ready code, requires PR review
- **develop**: Integration branch for features
- **feature/***: Feature branches, merge to develop
- **hotfix/***: Emergency fixes, merge to main and develop

### Tagging Convention

- **Semantic Versioning**: MAJOR.MINOR.PATCH (v1.2.3)
- **Major**: Breaking changes
- **Minor**: New features, backward compatible
- **Patch**: Bug fixes, backward compatible

### Pull Request Workflow

1. Create feature branch from develop
2. Make changes, commit with conventional messages
3. Push and create PR to develop
4. CI runs automatically (lint, test, security)
5. Code review required
6. Merge to develop
7. Periodically merge develop to main
8. Tag main for production release

### Security

- Rotate Kubernetes credentials every 90 days
- Review Dependabot PRs weekly
- Monitor GitHub Security tab daily
- Use branch protection rules:
  - Require CI success
  - Require code review
  - Require signed commits
  - Restrict force push

## Troubleshooting

### CI Failures

**Import boundary violations**:
```bash
# Run locally
go test -v ./internal/ci_checks/...
```

**Linting failures**:
```bash
# Run locally
golangci-lint run --timeout=5m
```

**Race conditions**:
```bash
# Run locally with race detector
go test -race ./...
```

### Deployment Failures

**Image pull errors**:
- Check GHCR permissions
- Verify image tag exists
- Check Kubernetes secret for GHCR credentials

**Health check failures**:
- Check pod logs: `kubectl logs -n <namespace> -l app=user-service`
- Verify config: Database, Redis, Vault connectivity
- Check resource limits: CPU, memory

**Rollout timeout**:
- Check pod events: `kubectl describe pod <pod-name> -n <namespace>`
- Verify readiness probe configuration
- Check application startup logs

### Security Scan Failures

**High/Critical vulnerabilities**:
- Review Dependabot PRs for updates
- Check if vulnerability is exploitable in your context
- Document exceptions if accepting risk

**License violations**:
- Review `licenses.txt` artifact
- Replace dependencies with compatible licenses
- Document approved exceptions

## Performance

### CI Optimization

- **Caching**: Go modules, build cache
- **Parallelization**: Jobs run concurrently
- **Targeted tests**: Only run affected tests (future)

### Build Optimization

- **Multi-stage builds**: Minimal final image size
- **Layer caching**: Reuse unchanged layers
- **BuildKit**: Parallel builds, better caching

### Deployment Optimization

- **Canary**: Reduce blast radius, faster detection
- **Health checks**: Fast fail on unhealthy deployments
- **Resource limits**: Prevent resource exhaustion

## Future Enhancements

- [ ] Progressive delivery with Flagger
- [ ] Blue-green deployments
- [ ] Integration with ArgoCD for GitOps
- [ ] Automated performance testing in CI
- [ ] Chaos engineering with LitmusChaos
- [ ] Multi-region deployments
- [ ] Feature flags with LaunchDarkly
- [ ] A/B testing framework

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [Dependabot Configuration](https://docs.github.com/en/code-security/dependabot/dependabot-version-updates)
- [Pre-commit Framework](https://pre-commit.com/)
- [Kubernetes Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/)
- [SARIF Format](https://sarifweb.azurewebsites.net/)
