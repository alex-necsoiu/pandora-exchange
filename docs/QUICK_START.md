# 🚀 Quick Start Guide

Complete setup guide for local development and deployment of Pandora Exchange User Service.

## 📋 Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Running the Service](#running-the-service)
- [Development Workflow](#development-workflow)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

| Tool | Version | Purpose | Installation |
|------|---------|---------|--------------|
| **Go** | 1.21+ | Programming language | [go.dev/doc/install](https://go.dev/doc/install) |
| **Docker** | 20.10+ | Container runtime | [docs.docker.com/get-docker](https://docs.docker.com/get-docker/) |
| **Docker Compose** | 2.0+ | Multi-container orchestration | Included with Docker Desktop |
| **PostgreSQL** | 15+ | Primary database | Via Docker or [postgresql.org](https://www.postgresql.org/download/) |
| **Redis** | 7+ | Cache & event streaming | Via Docker or [redis.io](https://redis.io/download) |
| **Make** | 4.0+ | Build automation | Pre-installed on macOS/Linux, [GnuWin32](http://gnuwin32.sourceforge.net/packages/make.htm) for Windows |

### Optional Tools

| Tool | Purpose | Installation |
|------|---------|--------------|
| **sqlc** | SQL code generation | `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest` |
| **golangci-lint** | Code linting | [golangci-lint.run/usage/install](https://golangci-lint.run/usage/install/) |
| **mockgen** | Mock generation | `go install go.uber.org/mock/mockgen@latest` |
| **migrate** | Database migrations | `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest` |

---

## Installation

### 1. Clone Repository

```bash
git clone https://github.com/alex-necsoiu/pandora-exchange.git
cd pandora-exchange
```

### 2. Install Go Dependencies

```bash
go mod download
go mod verify
```

### 3. Start Development Environment

```bash
# Start PostgreSQL + Redis in Docker
make dev-up

# Verify services are running
docker ps
```

Expected output:
```
CONTAINER ID   IMAGE          PORTS                    NAMES
abc123...      postgres:15    0.0.0.0:5432->5432/tcp   pandora-postgres
def456...      redis:7        0.0.0.0:6379->6379/tcp   pandora-redis
```

### 4. Run Database Migrations

```bash
# Apply all migrations
make migrate

# Verify migrations
psql -h localhost -U postgres -d pandora_dev -c "\dt"
```

Expected tables:
- `users`
- `refresh_tokens`
- `audit_logs`
- `schema_migrations`

### 5. Generate sqlc Code

```bash
# Generate type-safe SQL code
make sqlc

# Verify generated files
ls internal/postgres/*.go
```

### 6. Run Tests

```bash
# Run all tests
make test

# Expected: All tests pass
# ✓ Domain tests
# ✓ Service tests
# ✓ Repository tests
# ✓ Handler tests
# ✓ Middleware tests
```

### 7. Build the Service

```bash
# Build binary
make build

# Verify binary exists
./bin/user-service --version
```

### 8. Start the Service

```bash
# Run locally
make run

# Service should start on:
# - HTTP: http://localhost:8080
# - gRPC: localhost:9090
# - Admin: http://localhost:8081
```

---

## Configuration

### Environment Variables

Create a `.env.dev` file in the project root:

```bash
# Application
APP_ENV=dev
APP_NAME=user-service
LOG_LEVEL=debug

# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
GRPC_PORT=9090
ADMIN_PORT=8081
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=pandora_dev
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_POOL_SIZE=10

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production-min-32-chars
JWT_ISSUER=pandora-exchange
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=168h

# Rate Limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_WINDOW=100
RATE_LIMIT_WINDOW_DURATION=1m
RATE_LIMIT_ENABLE_PER_USER=true
RATE_LIMIT_USER_REQUESTS_PER_WINDOW=60
RATE_LIMIT_LOGIN_REQUESTS=5
RATE_LIMIT_LOGIN_WINDOW=15m

# OpenTelemetry
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
OTEL_SERVICE_NAME=user-service
OTEL_SERVICE_VERSION=1.0.0
OTEL_SAMPLE_RATE=1.0
OTEL_INSECURE=true

# Vault (optional for dev)
VAULT_ENABLED=false
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=dev-only-token
VAULT_SECRET_PATH=secret/data/user-service

# Audit Logs
AUDIT_RETENTION_DAYS=30
AUDIT_CLEANUP_ENABLED=true
AUDIT_CLEANUP_INTERVAL=24h
```

### Configuration Loading Priority

1. **Environment Variables** (highest priority)
2. **Config File** (`configs/dev.yaml`)
3. **Default Values** (lowest priority)

### Example YAML Config

Create `configs/dev.yaml`:

```yaml
app:
  name: user-service
  env: dev

server:
  host: 0.0.0.0
  port: 8080
  grpc_port: 9090

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: pandora_dev
  sslmode: disable

redis:
  host: localhost
  port: 6379
  db: 0

jwt:
  secret: ${JWT_SECRET}
  access_token_expiry: 15m
  refresh_token_expiry: 168h

otel:
  enabled: true
  endpoint: localhost:4317
  service_name: user-service
```

---

## Running the Service

### Using Make

```bash
# Start service with hot reload (requires air)
make dev

# Start service normally
make run

# Run in background
nohup make run > service.log 2>&1 &
```

### Using Docker

```bash
# Build image
make docker-build

# Run container
docker run -p 8080:8080 -p 9090:9090 \
  --env-file .env.dev \
  --network pandora-network \
  pandora-exchange/user-service:latest
```

### Using Docker Compose

```bash
# Start all services (PostgreSQL, Redis, User Service)
docker-compose -f deployments/docker/docker-compose.yml up -d

# View logs
docker-compose -f deployments/docker/docker-compose.yml logs -f user-service

# Stop all services
docker-compose -f deployments/docker/docker-compose.yml down
```

### Verify Service is Running

```bash
# Check health endpoint
curl http://localhost:8080/health

# Expected response:
# {"status":"ok","timestamp":"2024-11-12T10:30:00Z"}

# Check readiness endpoint
curl http://localhost:8080/ready

# Expected response:
# {"status":"ready","checks":{"database":"ok","redis":"ok"}}

# View Swagger UI
open http://localhost:8080/swagger/index.html
```

---

## Development Workflow

### 1. Create Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Write Tests First (TDD)

```bash
# Create test file
touch internal/service/user_service_test.go

# Write failing tests
go test ./internal/service/... -v -run TestYourFeature

# Expected: FAIL (test not implemented yet)
```

### 3. Implement Feature

```bash
# Write implementation
vim internal/service/user_service.go

# Run tests again
go test ./internal/service/... -v -run TestYourFeature

# Expected: PASS
```

### 4. Run Full Test Suite

```bash
# All tests
make test

# With coverage
make test-coverage

# Open coverage report
open coverage.html
```

### 5. Run Linter

```bash
# Run golangci-lint
make lint

# Fix auto-fixable issues
golangci-lint run --fix
```

### 6. Update Documentation

```bash
# If you changed API endpoints
make swagger

# If you changed database schema
# Add new migration file in migrations/
```

### 7. Commit Changes

```bash
# Stage changes
git add .

# Commit with conventional commit message
git commit -m "feat: add user profile update endpoint"

# Push to remote
git push origin feature/your-feature-name
```

### 8. Create Pull Request

- Go to GitHub repository
- Click "New Pull Request"
- Select your branch
- Fill in PR template
- Request reviews

---

## Troubleshooting

### Common Issues

#### Issue: `sqlc: command not found`

**Solution:**
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Verify installation
sqlc version
```

#### Issue: Database connection refused

**Solution:**
```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# If not running, start it
make dev-up

# Verify connection
psql -h localhost -U postgres -d pandora_dev -c "SELECT 1"
```

#### Issue: Redis connection timeout

**Solution:**
```bash
# Check if Redis is running
docker ps | grep redis

# Test connection
redis-cli -h localhost -p 6379 ping

# Expected: PONG
```

#### Issue: Port already in use

**Solution:**
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or use different port
SERVER_PORT=8081 make run
```

#### Issue: Migration fails with "version X is dirty"

**Solution:**
```bash
# Force version (use with caution)
migrate -path migrations -database "postgresql://postgres:postgres@localhost:5432/pandora_dev?sslmode=disable" force <version>

# Then retry migration
make migrate
```

#### Issue: Tests fail with "context deadline exceeded"

**Solution:**
```bash
# Increase test timeout
go test ./... -v -timeout 5m

# Or specific package
go test ./internal/repository/... -v -timeout 2m
```

#### Issue: Docker build fails

**Solution:**
```bash
# Clear Docker cache
docker builder prune -a

# Rebuild
make docker-build

# Or build with no cache
docker build --no-cache -t pandora-exchange/user-service:latest .
```

#### Issue: Go module download fails

**Solution:**
```bash
# Clear module cache
go clean -modcache

# Re-download
go mod download

# Verify
go mod verify
```

---

## Make Targets Reference

| Command | Description |
|---------|-------------|
| `make dev-up` | Start PostgreSQL + Redis in Docker |
| `make dev-down` | Stop development environment |
| `make migrate` | Run database migrations (up) |
| `make migrate-down` | Rollback last migration |
| `make sqlc` | Generate sqlc code from SQL |
| `make test` | Run all tests |
| `make test-coverage` | Run tests with coverage report |
| `make test-integration` | Run integration tests only |
| `make test-unit` | Run unit tests only |
| `make lint` | Run golangci-lint |
| `make build` | Build service binary |
| `make run` | Run service locally |
| `make dev` | Run with hot reload (requires air) |
| `make docker-build` | Build Docker image |
| `make docker-push` | Push Docker image to registry |
| `make swagger` | Generate Swagger documentation |
| `make clean` | Clean build artifacts |
| `make proto` | Generate Protocol Buffer code |
| `make mocks` | Generate mock implementations |

---

## Next Steps

- ✅ Service is running
- 📚 Read [API Documentation](./API_DOCUMENTATION.md)
- 🏗️ Understand [Architecture](../ARCHITECTURE.md)
- 🔐 Review [Security Guidelines](./SECURITY.md)
- 🤝 Check [Contributing Guide](./CONTRIBUTING.md)
- 🧪 Explore [Testing Strategy](./TESTING.md)

---

**Need Help?**
- 💬 [GitHub Discussions](https://github.com/alex-necsoiu/pandora-exchange/discussions)
- 🐛 [Report Issues](https://github.com/alex-necsoiu/pandora-exchange/issues)
- 📖 [Full Documentation](../docs/)
