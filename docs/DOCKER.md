# Docker Guide

Comprehensive guide for building, running, and deploying the Pandora Exchange User Service using Docker.

## Quick Start

```bash
# Build the image
make docker-build

# Start development environment
make compose-up

# View logs
make compose-logs

# Stop everything
make compose-down
```

## Docker Image Details

### Image Characteristics

- **Base Image**: `gcr.io/distroless/static-debian12:nonroot`
- **Size**: ~45MB (optimized with multi-stage build)
- **User**: `nonroot` (UID 65532, GID 65532)
- **Architecture**: Multi-platform (amd64, arm64)
- **Security**: Minimal attack surface, no shell, no package manager

### Exposed Ports

| Port | Service | Description |
|------|---------|-------------|
| 8080 | HTTP | REST API endpoints |
| 9090 | gRPC | gRPC service endpoints |
| 2112 | Metrics | Prometheus metrics endpoint |

### Build Arguments

| Argument | Description | Default | Example |
|----------|-------------|---------|---------|
| `VERSION` | Semantic version | `dev` | `v1.2.3` |
| `COMMIT` | Git commit hash | `unknown` | `abc123def` |
| `BUILD_TIME` | ISO 8601 timestamp | `unknown` | `2025-11-12T10:00:00Z` |
| `GO_VERSION` | Go version for build | `1.23` | `1.23` |

## Building Images

### Development Build

```bash
# Simple build
docker build -t pandora/user-service:dev .

# Using make
make docker-build
```

### Production Build with Version Info

```bash
# Manual build
docker build \
  --build-arg VERSION=v1.2.3 \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t pandora/user-service:v1.2.3 \
  .

# Using make (automatically extracts version from git)
make docker-build
```

### Multi-Platform Build

```bash
# Set up buildx (one time)
docker buildx create --name multiplatform --use
docker buildx inspect --bootstrap

# Build for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=v1.2.3 \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  -t ghcr.io/alex-necsoiu/pandora-exchange/user-service:v1.2.3 \
  --push \
  .
```

## Running Containers

### Single Container (with external dependencies)

```bash
# Run with environment variables
docker run -d \
  --name user-service \
  -p 8080:8080 \
  -p 9090:9090 \
  -p 2112:2112 \
  -e APP_ENV=production \
  -e DATABASE_URL=postgres://user:pass@host:5432/db \
  -e REDIS_URL=redis://host:6379 \
  -e JWT_ACCESS_SECRET=your-secret \
  -e JWT_REFRESH_SECRET=your-refresh-secret \
  pandora/user-service:latest

# View logs
docker logs -f user-service

# Stop and remove
docker stop user-service
docker rm user-service
```

### Using Make

```bash
# Run container (requires pandora-network to exist)
make docker-run

# Stop container
make docker-stop
```

### Full Stack with Docker Compose

```bash
# Start all services (user-service, postgres, redis)
docker-compose -f docker-compose.dev.yml up -d

# Or using make
make compose-up

# View logs from all services
docker-compose -f docker-compose.dev.yml logs -f

# View logs from specific service
docker-compose -f docker-compose.dev.yml logs -f user-service

# Stop all services
make compose-down
```

## Docker Compose Profiles

### Basic Services (default)

```bash
# Starts: user-service, postgres, redis
docker-compose -f docker-compose.dev.yml up -d
```

### With Monitoring

```bash
# Starts: basic + prometheus + grafana
docker-compose -f docker-compose.dev.yml --profile monitoring up -d

# Access Grafana: http://localhost:3000 (admin/admin)
# Access Prometheus: http://localhost:9091
```

### With Vault

```bash
# Starts: basic + vault
docker-compose -f docker-compose.dev.yml --profile vault up -d

# Access Vault: http://localhost:8200 (token: dev-token)
```

### All Services

```bash
# Start everything
docker-compose -f docker-compose.dev.yml --profile monitoring --profile vault up -d
```

## Environment Variables

### Required Variables

```bash
# Application
APP_ENV=production              # Environment: development, staging, production
LOG_LEVEL=info                  # Logging: debug, info, warn, error

# Database
DATABASE_URL=postgres://user:pass@host:5432/db
# OR individual vars:
DATABASE_HOST=postgres
DATABASE_PORT=5432
DATABASE_NAME=pandora
DATABASE_USER=pandora
DATABASE_PASSWORD=secret
DATABASE_MAX_CONNECTIONS=25

# Redis
REDIS_URL=redis://host:6379
# OR individual vars:
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0
REDIS_PASSWORD=

# JWT Secrets
JWT_ACCESS_SECRET=long-random-secret-here
JWT_REFRESH_SECRET=another-long-random-secret
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h
```

### Optional Variables

```bash
# Server Ports
HTTP_PORT=8080
GRPC_PORT=9090
METRICS_PORT=2112
SHUTDOWN_TIMEOUT=30s

# Vault (if using)
VAULT_ADDRESS=http://vault:8200
VAULT_TOKEN=vault-token
VAULT_PATH=secret/data/user-service

# Rate Limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Observability
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_SERVICE_NAME=user-service
OTEL_SERVICE_VERSION=v1.0.0
```

## Security

### Image Scanning

```bash
# Trivy scan
trivy image pandora/user-service:latest

# Grype (Anchore) scan
grype pandora/user-service:latest

# Using make
make docker-scan
```

### Security Best Practices

1. **Non-root User**: Image runs as `nonroot` (UID 65532)
2. **Distroless Base**: No shell, no package manager, minimal attack surface
3. **Read-only Filesystem**: Application doesn't need write access
4. **No Secrets in Image**: All secrets via environment variables
5. **Multi-stage Build**: Build dependencies not in final image
6. **Minimal Layers**: Optimized layer caching for faster builds

### Running with Additional Security

```bash
docker run -d \
  --name user-service \
  --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=10m \
  --security-opt=no-new-privileges:true \
  --cap-drop=ALL \
  --cap-add=NET_BIND_SERVICE \
  -p 8080:8080 \
  pandora/user-service:latest
```

## Health Checks

### Kubernetes Readiness/Liveness Probes

```yaml
# Liveness probe
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 3

# Readiness probe (checks dependencies)
readinessProbe:
  httpGet:
    path: /health/services
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

### Docker Health Check

Note: Distroless images don't have shell/curl, so health checks should be done via orchestrator (Kubernetes, Docker Swarm).

For testing with standard images:
```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1
```

## Registry Operations

### GitHub Container Registry (GHCR)

```bash
# Login
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Tag image
docker tag pandora/user-service:latest \
  ghcr.io/alex-necsoiu/pandora-exchange/user-service:latest

# Push image
docker push ghcr.io/alex-necsoiu/pandora-exchange/user-service:latest

# Using make
make docker-push REGISTRY=ghcr.io/alex-necsoiu/pandora-exchange
```

### DockerHub

```bash
# Login
docker login

# Tag and push
docker tag pandora/user-service:latest your-username/user-service:latest
docker push your-username/user-service:latest
```

## Troubleshooting

### Build Issues

**Problem**: Build fails with "go mod download" error

**Solution**: Check internet connectivity and proxy settings
```bash
docker build --build-arg HTTPS_PROXY=http://proxy:8080 .
```

**Problem**: Build is very slow

**Solution**: Use BuildKit and layer caching
```bash
DOCKER_BUILDKIT=1 docker build --cache-from pandora/user-service:latest .
```

### Runtime Issues

**Problem**: Container exits immediately

**Solution**: Check logs and environment variables
```bash
docker logs user-service
docker inspect user-service
```

**Problem**: Cannot connect to database/redis

**Solution**: Verify network connectivity
```bash
# Check container network
docker network inspect pandora-network

# Verify services are running
docker ps

# Test connectivity from container
docker exec user-service ping postgres  # Won't work with distroless
```

**Problem**: Health check fails

**Solution**: Verify service is listening on correct port
```bash
# Check port mapping
docker port user-service

# Test endpoint
curl http://localhost:8080/health
```

### Common Errors

**Error**: "bind: address already in use"

**Solution**: Stop service using the port or use different port
```bash
# Find process using port
lsof -ti:8080 | xargs kill -9

# Or use different port
docker run -p 8081:8080 pandora/user-service:latest
```

**Error**: "permission denied" when accessing volumes

**Solution**: Check volume permissions (distroless runs as UID 65532)
```bash
# Fix volume permissions
chown -R 65532:65532 /path/to/volume
```

## Performance Optimization

### Build Optimization

```bash
# Use BuildKit for parallel builds
export DOCKER_BUILDKIT=1

# Use build cache
docker build --cache-from pandora/user-service:latest .

# Multi-stage builds already optimize size
# Current image: ~45MB (vs ~400MB+ without multi-stage)
```

### Runtime Optimization

```bash
# Limit memory and CPU
docker run -d \
  --memory=512m \
  --memory-swap=512m \
  --cpus=2 \
  pandora/user-service:latest

# Use restart policy
docker run -d \
  --restart=unless-stopped \
  pandora/user-service:latest
```

## CI/CD Integration

### GitHub Actions (already configured)

See `.github/workflows/deploy.yml` for automated builds:
- Multi-platform builds on tag push
- SBOM generation
- Push to GHCR
- Deployment to Kubernetes

### Manual Release Process

```bash
# 1. Tag release
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3

# 2. Build image
make docker-build

# 3. Push to registry
make docker-push REGISTRY=ghcr.io/alex-necsoiu/pandora-exchange

# 4. Deploy to Kubernetes
kubectl set image deployment/user-service \
  user-service=ghcr.io/alex-necsoiu/pandora-exchange/user-service:v1.2.3 \
  -n production

# 5. Verify deployment
kubectl rollout status deployment/user-service -n production
```

## References

- [Dockerfile Best Practices](https://docs.docker.com/develop/develop-images/dockerfile_best-practices/)
- [Distroless Images](https://github.com/GoogleContainerTools/distroless)
- [Multi-stage Builds](https://docs.docker.com/build/building/multi-stage/)
- [Docker Security](https://docs.docker.com/engine/security/)
- [GHCR Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
