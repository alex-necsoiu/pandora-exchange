# 🔥 Pandora Exchange - User Service (Phase 1)

> **High-performance digital asset trading platform backend**  
> Status: 🚧 In Development  
> Architecture: Microservices | Event-Driven | Cloud-Native

---

## 📋 Project Overview

Pandora Exchange is a secure, scalable, and compliant trading platform backend built with:

- **Go 1.21+** - High-performance, compiled language
- **Gin** - HTTP REST API framework
- **gRPC** - Internal service-to-service communication
- **PostgreSQL** - Primary data store
- **sqlc** - Type-safe SQL query generation
- **Redis Streams** - Event-driven async messaging
- **OpenTelemetry** - Distributed tracing
- **Argon2id** - Password hashing
- **JWT** - Authentication tokens
- **Vault** - Secrets management
- **Docker & Kubernetes** - Containerization & orchestration

**Phase 1 Scope:** User Service (Authentication, Registration, KYC Management)

---

## 🏗️ Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for complete specification.

### Service Structure
```
user-service/
├── cmd/user-service/          # Application entry point
├── internal/
│   ├── domain/                # Business logic & interfaces
│   ├── repository/            # Data access implementations
│   ├── postgres/              # sqlc generated code
│   ├── transport/
│   │   ├── http/              # Gin REST handlers
│   │   └── grpc/              # gRPC service
│   ├── events/                # Redis Streams publishers
│   ├── config/                # Configuration management
│   ├── middleware/            # HTTP/gRPC middleware
│   └── observability/         # Logging, tracing, metrics
├── migrations/                # PostgreSQL schema migrations
├── deployments/
│   ├── docker/                # Dockerfiles
│   └── k8s/                   # Kubernetes manifests
├── .github/workflows/         # CI/CD pipelines
├── go.mod
├── Makefile
└── README.md
```

---

## 📊 Development Roadmap

### Epic: User Service MVP (Phase 1)

| # | Task | Status | Branch | Commit | Files |
|---|------|--------|--------|--------|-------|
| 1 | Bootstrap User Service Repository | ✅ Completed | `feature/bootstrap-repo` | `feat: initialize go module and folder structure` | go.mod, .gitignore, Makefile, folder structure |
| 2 | Database Schema & Migrations | ✅ Completed | 2b3b527 | users + refresh_tokens tables with migrations | 2024-01-XX |
| 3 | sqlc Configuration & Queries | ✅ Completed | 4cb0710 | 15 type-safe SQL queries generated | 2024-01-XX |
| 4 | Domain Layer - Models & Interfaces | ✅ Completed | cb82e41 | Models, interfaces, errors, 24 passing tests | 2024-01-XX |
| 5 | Repository Implementation with sqlc | ✅ Completed | a5e278d | UserRepo + RefreshTokenRepo, 16 test suites passing | 2024-01-XX |
| 6 | Password Hashing with Argon2id | ✅ Completed | - | Argon2id (64MB, t=1, p=4), timing attack resistant | 2024-01-XX |
| 7 | JWT Token Service | ⚪ Not Started | - | - | - |
| 8 | User Service Implementation | ⚪ Not Started | - | - | - |
| 9 | Configuration Management | ⚪ Not Started | - | - | - |
| 10 | Logging with Zerolog | ⚪ Not Started | - | - | - |
| 11 | OpenTelemetry Tracing Setup | ⚪ Not Started | - | - | - |
| 12 | Gin HTTP Transport Layer | ⚪ Not Started | - | - | - |
| 13 | gRPC Service Definition & Implementation | ⚪ Not Started | - | - | - |
| 14 | Redis Streams Event Publisher | ⚪ Not Started | - | - | - |
| 15 | Middleware - Auth & Security | ⚪ Not Started | - | - | - |
| 16 | Health Check Endpoints | ⚪ Not Started | - | - | - |
| 17 | Main Application Wiring | ⚪ Not Started | - | - | - |
| 18 | Docker & Docker Compose | ⚪ Not Started | - | - | - |
| 19 | Integration Tests | ⚪ Not Started | - | - | - |
| 20 | CI/CD Pipeline - GitHub Actions | ⚪ Not Started | - | - | - |
| 21 | Kubernetes Manifests | ⚪ Not Started | - | - | - |
| 22 | Vault Integration | ⚪ Not Started | - | - | - |
| 23 | Audit Logging | ⚪ Not Started | - | - | - |
| 24 | Documentation & README | ⚪ Not Started | - | - | - |

**Legend:**  
⚪ Not Started | 🔵 In Progress | ✅ Completed | 🔴 Blocked

---

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- Make

### Local Development

```bash
# 1. Clone repository
git clone <repository-url>
cd pandora-exchange

# 2. Start development environment
make dev-up

# 3. Run migrations
make migrate

# 4. Generate sqlc code
make sqlc

# 5. Run tests
make test

# 6. Start service
make run
```

### Available Make Targets

```bash
make dev-up        # Start PostgreSQL + Redis in Docker
make dev-down      # Stop development environment
make migrate       # Run database migrations
make sqlc          # Generate sqlc code from SQL queries
make test          # Run all tests
make test-coverage # Run tests with coverage report
make lint          # Run golangci-lint
make build         # Build service binary
make run           # Run service locally
make docker-build  # Build Docker image
make clean         # Clean build artifacts
```

---

## 🔐 Security

- **Password Hashing:** Argon2id (time=1, memory=64MB, threads=4)
- **Authentication:** JWT access tokens (15min) + refresh tokens (7 days)
- **Secrets Management:** HashiCorp Vault (no credentials in code/env)
- **TLS:** Required for all external communication
- **Audit Logging:** Immutable logs for compliance

---

## 📡 API Endpoints

### REST API (Gin)

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/api/v1/auth/register` | Register new user | No |
| POST | `/api/v1/auth/login` | Login user | No |
| POST | `/api/v1/auth/refresh` | Refresh access token | Refresh Token |
| GET | `/api/v1/users/me` | Get current user | JWT |
| PATCH | `/api/v1/users/me/kyc` | Update KYC status | JWT |
| GET | `/health` | Health check | No |
| GET | `/ready` | Readiness check | No |

### gRPC (Internal)

```protobuf
service UserService {
  rpc GetUser(GetUserRequest) returns (UserResponse);
  rpc UpdateKYC(UpdateKYCRequest) returns (UserResponse);
}
```

---

## 🌍 Environments

| Environment | Purpose | Database | Config |
|-------------|---------|----------|--------|
| `dev` | Local development | Local PostgreSQL | `.env.dev` |
| `sandbox` | QA/Testing | Cloud DB (synthetic data) | `.env.sandbox` |
| `audit` | Compliance testing | Anonymized prod clone | `.env.audit` |
| `prod` | Production | Secure cloud DB | Vault secrets |

Set environment: `export APP_ENV=dev`

---

## 🧪 Testing Strategy

- **Unit Tests:** All domain logic with table-driven tests
- **Integration Tests:** Full service tests with testcontainers
- **TDD Approach:** Tests written BEFORE implementation
- **Coverage Target:** >80% code coverage
- **Mock Generation:** Using mockgen for interfaces

```bash
# Run unit tests
go test ./internal/... -v

# Run with coverage
go test ./internal/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run integration tests
go test ./tests/integration/... -v
```

---

## 📦 Database Schema

### Users Table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    full_name TEXT,
    hashed_password TEXT NOT NULL,
    kyc_status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
```

### Refresh Tokens Table
```sql
CREATE TABLE refresh_tokens (
    token TEXT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

---

## 🐛 Troubleshooting

### Common Issues

**Issue:** `sqlc: command not found`  
**Solution:** Install sqlc: `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`

**Issue:** Database connection refused  
**Solution:** Ensure PostgreSQL is running: `make dev-up`

**Issue:** Migration fails  
**Solution:** Check migration files and ensure DB is accessible

---

## 📚 References

- [ARCHITECTURE.md](./ARCHITECTURE.md) - Complete architecture specification
- [Go 1.21 Documentation](https://go.dev/doc/)
- [sqlc Documentation](https://docs.sqlc.dev/)
- [Gin Framework](https://gin-gonic.com/docs/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)

---

## 👥 Contributing

1. Read [ARCHITECTURE.md](./ARCHITECTURE.md) - **mandatory**
2. Follow TDD: Write tests first
3. Use conventional commits: `feat:`, `fix:`, `test:`, `ci:`
4. Ensure all tests pass: `make test`
5. Run linter: `make lint`
6. Update task table in this README

---

## 📄 License

Proprietary - Pandora Exchange © 2025

---

**Last Updated:** November 6, 2025  
**Current Phase:** Bootstrap & Foundation (Tasks 1-4)
