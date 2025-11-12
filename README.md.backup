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
| 6 | Password Hashing with Argon2id | ✅ Completed | 1debc6c | Argon2id (64MB, t=1, p=4), timing attack resistant | 2024-01-XX |
| 7 | JWT Token Service | ✅ Completed | a02114b | HS256, access (15min) + refresh (7d), Vault-ready | 2024-01-XX |
| 8 | User Service Implementation | ✅ Completed | 57449a7 | 11 methods, 10 test suites, 22 tests passing | 2024-01-XX |
| 9 | Configuration Management | ✅ Completed | 37571bc | Viper config, 4 environments, 6 test suites passing | 2024-01-XX |
| 10 | Logging with Zerolog | ✅ Completed | - | Structured logging, 9 test suites, audit logs, sensitive data redaction | 2024-01-XX |
| 11 | OpenTelemetry Tracing Setup | ✅ Completed | - | OTLP exporter, Gin middleware, 9 test suites, Jaeger integration | 2024-11-08 |
| 12 | Gin HTTP Transport Layer | ✅ Completed | 76db8a0 | 11 handlers, 91.7% coverage, 483 tests passing | 2024-11-08 |
| 13 | gRPC Service Definition & Implementation | ✅ Completed | a653502 | 5 RPCs, interceptors, 100% coverage, 50 tests passing | 2024-11-08 |
| 14 | Redis Streams Event Publisher | ✅ Completed | - | 6 event types, 92.2% coverage, 17 tests, async publishing | 2024-11-08 |
| 15 | Middleware - Auth & Security | ✅ Completed | 76db8a0 | Auth, CORS, Recovery, Admin middleware, 100% coverage | 2024-11-08 |
| 16 | Health Check Endpoints | ✅ Completed | 76db8a0 | `/health` endpoint implemented and tested | 2024-11-08 |
| 17 | Main Application Wiring | ✅ Completed | - | Full application with user & admin routers | 2024-11-08 |
| 18 | Docker & Docker Compose | ✅ Completed | - | PostgreSQL + service containers configured | 2024-11-08 |
| 19 | Integration Tests | ✅ Completed | 9ac7c81 | 4 E2E test suites, real DB, full workflows | 2024-11-08 |
| 20 | CI/CD Pipeline - GitHub Actions | ⚪ Not Started | - | - | - |
| 21 | Kubernetes Manifests | ✅ Completed | - | 18 manifests, Kustomize overlays, complete deployment guide | 2024-11-08 |
| 22 | Vault Integration | ✅ Completed | 486fcbe, 4d1dafc | Vault client (251 lines), K8s integration, comprehensive integration tests (310 lines), testing guide | 2024-11-08 |
| 23 | Enhanced Audit Logging | ✅ Completed | bcc0612, ee13c3c | Audit logs table, repository (16 tests), cleanup job (9 tests), middleware (15 tests) | 2024-11-08 |
| 15 | Error Handling System | ✅ Completed | 6ce3c76, dee6c4c | AppError struct, HTTP/gRPC middleware, 35 tests, comprehensive docs | 2024-11-08 |
| 24 | Documentation & README | 🔵 In Progress | - | K8s deployment guide complete, main README updates pending | 2024-11-08 |

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

### gRPC (Internal Service-to-Service)

**Port:** 9090 (configurable via `GRPC_PORT`)

```protobuf
service UserService {
  // User retrieval
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc GetUserByEmail(GetUserByEmailRequest) returns (GetUserResponse);
  
  // KYC management
  rpc UpdateKYCStatus(UpdateKYCRequest) returns (UpdateKYCResponse);
  
  // User validation
  rpc ValidateUser(ValidateUserRequest) returns (ValidateUserResponse);
  
  // Admin operations
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}
```

**Interceptors:**
- Recovery: Panic recovery with error logging
- Logging: Request/response logging with duration
- Tracing: OpenTelemetry span creation
- Auth: JWT validation (planned)

**Testing:** 50 test suites with table-driven tests, 100% coverage

---

## 📡 Event-Driven Architecture (Redis Streams)

The service publishes domain events to Redis Streams for async processing by other microservices.

### Event Types

| Event Type | Trigger | Payload |
|------------|---------|---------|
| `user.registered` | New user registration | email, first_name, last_name, role |
| `user.kyc.updated` | KYC status change | email, kyc_status, old_status |
| `user.profile.updated` | Profile update | email, first_name, last_name |
| `user.deleted` | Account deletion | deleted_at |
| `user.logged_in` | Successful login | email, ip_address, user_agent |
| `user.password.changed` | Password change | email (planned) |

### Event Structure

```json
{
  "id": "uuid-v4",
  "type": "user.registered",
  "timestamp": "2024-11-08T10:30:00Z",
  "user_id": "user-uuid",
  "payload": {
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe"
  },
  "metadata": {
    "ip_address": "192.168.1.1",
    "user_agent": "Mozilla/5.0..."
  }
}
```

**Stream:** `user-service:events`  
**Max Length:** 10,000 events (auto-trimmed)  
**Testing:** 17 test suites, 92.2% coverage

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
- **Integration Tests:** Full service tests with real Vault dev server
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

# Run Vault integration tests (requires vault binary)
VAULT_INTEGRATION_TESTS=true go test ./internal/vault/... -v
```

**Vault Testing:**
- Unit tests: PASS (3 tests, 4.6s) - run by default
- Integration tests: 6 suites, 15+ scenarios - opt-in with `VAULT_INTEGRATION_TESTS=true`
- See [internal/vault/TESTING.md](internal/vault/TESTING.md) for detailed guide

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
- [VAULT_INTEGRATION.md](./VAULT_INTEGRATION.md) - HashiCorp Vault setup and usage guide
- [internal/vault/TESTING.md](./internal/vault/TESTING.md) - Vault integration testing guide
- [ERROR_HANDLING.md](./ERROR_HANDLING.md) - Error handling patterns and middleware
- [AUDIT_RETENTION_POLICY.md](./AUDIT_RETENTION_POLICY.md) - Audit log retention and cleanup
- [Go 1.21 Documentation](https://go.dev/doc/)
- [sqlc Documentation](https://docs.sqlc.dev/)
- [Gin Framework](https://gin-gonic.com/docs/)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
- [HashiCorp Vault](https://www.vaultproject.io/docs)

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

**Last Updated:** November 8, 2025  
**Current Phase:** Core Implementation (Tasks 1-19 mostly complete)  
**Next Priority:** Task 11 (OpenTelemetry), Task 13 (gRPC), Task 14 (Redis Events), Task 20 (CI/CD)
