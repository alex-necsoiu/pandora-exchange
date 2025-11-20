# 🚀 Pandora Exchange

**Enterprise-Grade Cryptocurrency Exchange Platform**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/Coverage-85%25-brightgreen)](./docs/testing.md)
[![Build Status](https://img.shields.io/badge/Build-Passing-success)](https://github.com/alex-necsoiu/pandora-exchange/actions)
[![Documentation](https://img.shields.io/badge/Docs-Complete-informational)](./docs/README.md)
[![OpenSSF Scorecard](https://img.shields.io/badge/OpenSSF-Secured-success)](https://github.com/alex-necsoiu/pandora-exchange/security)
[![Code Quality](https://img.shields.io/badge/golangci--lint-passing-brightgreen)](https://golangci-lint.run/)


---

## 📑 Table of Contents

- [Overview](#-overview)
- [Features](#-features)
- [Architecture](#-architecture)
- [Tech Stack](#️-tech-stack)
- [Quick Start](#-quick-start)
- [API Overview](#-api-overview)
- [Project Structure](#-project-structure)
- [Testing](#-testing)
- [Roadmap](#️-roadmap)
- [Documentation](#-documentation)
- [Contributing](#-contributing)
- [License](#-license)

---

## 📖 Overview

**Pandora Exchange** is a production-grade cryptocurrency exchange backend built with **Clean Architecture** and **Domain-Driven Design** principles. Phase 1 delivers a robust **User Service** with enterprise-level authentication, comprehensive audit logging, and observability—providing the secure foundation for a complete digital asset trading platform.

**Current Status:** ✅ **Phase 1 Complete** (28/28 tasks) | **Production Ready**

**Key Differentiators:**
- 🏗️ True Clean Architecture with zero infrastructure dependencies in domain layer
- 🔒 Military-grade security (Argon2id, JWT rotation, Vault integration, immutable audit logs)
- 📊 Full observability stack (OpenTelemetry traces, Prometheus metrics, structured logging)
- 🧪 Test-Driven Development with 85%+ coverage and table-driven tests
- 🚀 Production-hardened with rate limiting, graceful shutdown, and health checks

---

## ✨ Features

### 🔐 Security & Authentication
- **Argon2id** password hashing (PHC winner, 64MB memory cost, resistant to GPU attacks)
- **JWT** with automatic refresh token rotation (15 min access / 7 day refresh)
- **HashiCorp Vault** integration for production secrets management
- **Multi-layer rate limiting**:
  - Global: 100 req/min per IP
  - User: 60 req/min per authenticated user  
  - Login: 5 attempts per 15 min (brute-force protection)
- **Immutable audit logs** with 7-year retention (SOC2/GDPR compliance)
- **RBAC** with middleware-enforced authorization (user/admin roles)
- **Timing-safe** password verification (prevents timing attacks)
- **Security event logging** for failed admin access attempts

### 🏗️ Clean Architecture & Design Patterns
- **Domain-Driven Design** - Pure business logic with zero external dependencies
- **Dependency Inversion** - Domain defines interfaces, infrastructure implements
- **Repository Pattern** - Type-safe data access via `sqlc` (no ORM magic)
- **Service Layer** - Business logic orchestration with proper error handling
- **Transport Layer** - Separate HTTP (Gin) and gRPC servers (port 8080 & 50051)
- **Event-Driven** - Domain events published to Redis Streams for async processing
- **Test-Driven Development** - 85%+ coverage with table-driven tests and mocks
- **Factory Pattern** - Centralized dependency injection in `main.go`

### 📊 Production-Grade Observability
- **Prometheus** metrics with RED methodology:
  - Request rate per endpoint
  - Error rates with status codes
  - Duration histograms (p50, p95, p99)
  - Custom business metrics (registrations, logins, KYC updates)
- **OpenTelemetry** distributed tracing with context propagation
- **Structured JSON logging** (zerolog) with:
  - Trace ID correlation
  - PII redaction for compliance
  - Log levels: DEBUG, INFO, WARN, ERROR
- **Health checks** - Kubernetes-ready liveness/readiness probes
- **Graceful shutdown** - Proper resource cleanup and connection draining

### 🚀 Production Ready
- **Docker** containerization with multi-stage builds
- **Kubernetes** deployment manifests (base + overlays)
- **CI/CD** pipeline (GitHub Actions)
- **Swagger/OpenAPI 3.0** documentation
- **Multi-environment** support (dev/sandbox/audit/prod)
- **Database migrations** (golang-migrate)

---

## 🏛️ Architecture

Pandora Exchange follows **Clean Architecture** principles with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                     Transport Layer                         │
│  • REST API (Gin) - HTTP endpoints for clients              │
│  • gRPC API - Inter-service communication                   │
│  • Middleware - Auth, logging, rate limiting, tracing       │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                          │
│  • User Service - Registration, profile management          │
│  • Auth Service - Login, JWT, session management            │
│  • Admin Service - User administration, KYC                 │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    Repository Layer                         │
│  • User Repository - User CRUD operations                   │
│  • Token Repository - Refresh token management              │
│  • Audit Repository - Immutable audit logging               │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│                    Infrastructure                           │
│  • PostgreSQL 15 - Primary data store                       │
│  • Redis 7 - Cache, rate limiting, pub/sub                  │
│  • Vault - Secrets management (production)                  │
└─────────────────────────────────────────────────────────────┘
```

### Architecture Principles

- ✅ **Dependency Rule**: Dependencies point inward (Infrastructure → Repository → Service → Domain)
- ✅ **Domain Independence**: Business logic has zero external dependencies
- ✅ **Interface Segregation**: Inner layers define interfaces, outer layers implement
- ✅ **Testability**: Each layer can be tested in isolation with mocks
- ✅ **Flexibility**: Easy to swap infrastructure (e.g., PostgreSQL → MySQL)

### Key Components

| Layer | Responsibility | Technologies |
|-------|---------------|--------------|
| **Domain** | Business models, errors, interfaces | Pure Go structs & interfaces |
| **Service** | Business logic, validation | JWT, Argon2id, business rules |
| **Repository** | Data access, persistence | sqlc, PostgreSQL, migrations |
| **Transport** | HTTP/gRPC handlers, middleware | Gin, gRPC, OpenAPI |
| **Infrastructure** | External services, config | Vault, Redis, Prometheus, OTel |

> 📚 **Detailed Architecture**: See [ARCHITECTURE.md](./docs/ARCHITECTURE.md) for data models, event flows, and sequence diagrams.

---

## 🛠️ Tech Stack

| Category | Technology | Purpose |
|----------|-----------|---------|
| **Language** | Go 1.24+ | High-performance backend |
| **HTTP Framework** | Gin v1.9.1 | Fast HTTP router with rich middleware ecosystem |
| **Database** | PostgreSQL 15 | Primary data store |
| **Cache** | Redis 7 | Caching, rate limiting, events |
| **RPC** | gRPC | Inter-service communication |
| **ORM** | sqlc | Type-safe SQL generation |
| **Migrations** | golang-migrate | Schema versioning |
| **Auth** | JWT + Argon2id | Tokens + password hashing |
| **Secrets** | HashiCorp Vault | Production secrets |
| **Metrics** | Prometheus | Time-series metrics |
| **Tracing** | OpenTelemetry | Distributed tracing |
| **Logging** | zerolog | Structured JSON logs |
| **Containers** | Docker + Compose | Development & deployment |
| **Orchestration** | Kubernetes | Production deployment |
| **CI/CD** | GitHub Actions | Automated testing & deployment |

---

## 🚀 Quick Start

### Prerequisites
- Go 1.24+
- Docker & Docker Compose
- Make

### Installation

```bash
# Clone repository
git clone https://github.com/pandora-exchange/pandora-exchange.git
cd pandora-exchange

# Start infrastructure (PostgreSQL + Redis)
make dev-up

# Run database migrations
make migrate-up

# Generate code (sqlc, mocks)
make generate

# Run tests
make test

# Start service
make run
```

**Service Endpoints:**
- REST API: `http://localhost:8080/api/v1`
- gRPC Server: `localhost:50051`
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Health Check: `http://localhost:8080/health`
- Readiness Probe: `http://localhost:8080/ready`
- Prometheus Metrics: `http://localhost:8080/metrics`
- Admin API: `http://localhost:8080/api/v1/admin` (requires admin JWT)

### Quick Test

```bash
# Register a new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!",
    "first_name": "John",
    "last_name": "Doe"
  }'

# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'
```

> 📖 See [Quick Start Guide](./docs/QUICK_START.md) for detailed setup, configuration, and troubleshooting.

---

## 📡 API Overview

### REST Endpoints

**Authentication (Public)**
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login and get JWT tokens
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/admin-login` - Admin-only login endpoint

**User Management (Authenticated)**
- `GET /api/v1/users/me` - Get current user profile
- `PATCH /api/v1/users/me` - Update user profile (name)
- `POST /api/v1/users/me/logout` - Logout current session (revoke refresh token)
- `POST /api/v1/users/me/logout-all` - Logout all sessions

**Admin Operations (Admin Role Required)**
- `GET /api/v1/admin/users` - List all users (paginated)
- `GET /api/v1/admin/users/:id` - Get user by ID
- `PATCH /api/v1/admin/users/:id/kyc` - Update KYC status (pending/verified/rejected)
- `DELETE /api/v1/admin/users/:id` - Soft delete user account

**System & Monitoring**
- `GET /health` - Liveness probe (always returns 200)
- `GET /ready` - Readiness probe (checks DB/Redis connectivity)
- `GET /metrics` - Prometheus metrics (RED + business metrics)
- `GET /swagger/index.html` - Interactive API documentation

### gRPC Services

**Internal RPC API** (Port 50051 - Service-to-Service Communication)

```protobuf
service UserService {
  // User Queries
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc GetUserByEmail(GetUserByEmailRequest) returns (GetUserResponse);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  
  // User Operations
  rpc UpdateKYCStatus(UpdateKYCRequest) returns (UpdateKYCResponse);
  rpc ValidateUser(ValidateUserRequest) returns (ValidateUserResponse);
  
  // Health Check
  rpc Health(HealthRequest) returns (HealthResponse);
}
```

**Features:**
- mTLS authentication (production)
- Request/response interceptors for logging & tracing
- Error mapping to gRPC status codes
- Context propagation for distributed tracing

> 📡 See [API Documentation](./docs/API_DOCUMENTATION.md) for complete API reference with examples.

---

## 📁 Project Structure

```
pandora-exchange/
├── cmd/
│   └── user-service/          # Service entry point
├── internal/
│   ├── domain/                # Domain models, interfaces, errors
│   ├── service/               # Business logic layer
│   ├── repository/            # Data access layer (PostgreSQL)
│   ├── transport/             # HTTP & gRPC handlers
│   │   ├── http/              # REST API (Gin)
│   │   └── grpc/              # gRPC server
│   ├── middleware/            # Auth, logging, rate limiting
│   ├── vault/                 # HashiCorp Vault integration
│   ├── observability/         # Metrics, tracing, logging
│   └── config/                # Configuration management
├── migrations/                # Database migrations (SQL)
├── deployments/
│   ├── docker/                # Docker Compose configs
│   └── k8s/                   # Kubernetes manifests
├── docs/                      # Documentation
├── tests/integration/         # Integration tests
└── pkg/                       # Shared utilities
```

---

## 🔐 Security Features

**Security-First Design** - Built with defense-in-depth principles

| Feature | Implementation | Configuration | Rationale |
|---------|---------------|---------------|------------|
| **Password Hashing** | Argon2id (PHC winner) | 64MB memory, 1 iteration, 4 threads | Memory-hard function resistant to GPU/ASIC attacks |
| **Authentication** | JWT (RS256 for production) | Access: 15 min, Refresh: 7 days | Short-lived tokens reduce exposure window |
| **Token Rotation** | Automatic refresh token rotation | On each refresh, old token revoked | Prevents token replay attacks |
| **Secrets Management** | HashiCorp Vault (production) | Dynamic secrets, lease renewal | Zero secrets in code/config, audit trail |
| **Rate Limiting** | Redis sliding window | Global: 100/min, User: 60/min, Login: 5/15min | Prevents brute-force and DDoS attacks |
| **Audit Logging** | PostgreSQL (immutable) | 7-year retention | SOC2/GDPR compliance, forensic analysis |
| **Authorization** | RBAC with JWT claims | Middleware validates on each request | Fail-secure with deny-by-default |
| **Input Validation** | Gin binding + custom validators | Email, UUID, SQL injection checks | Prevents injection attacks |
| **Timing Attacks** | Constant-time comparison | Password verification | Prevents timing-based enumeration |
| **Admin Separation** | Separate login endpoint | Admin-only endpoint validation | Prevents privilege escalation |
| **Security Events** | High-priority audit logs | Failed admin logins, suspicious activity | Real-time threat detection |

> 🔐 See [Security Guide](./docs/SECURITY.md) for comprehensive security documentation.

---

## 🧪 Testing & Quality

```bash
# Run all tests
make test

# Run with coverage
make coverage

# View coverage in browser
make coverage-html

# Run linters
make lint

# Security scan
make security
```

**Test Coverage by Layer:**
- **Domain Layer**: 90%+ (pure business logic)
- **Service Layer**: 90%+ (user service, auth)
- **Repository Layer**: 85%+ (database operations)
- **Transport Layer**: 80%+ (HTTP handlers, gRPC)
- **Middleware**: 85%+ (auth, rate limiting, metrics)
- **Overall**: **85%+** (exceeds 80% requirement)

**Test Types:**
- ✅ Unit tests with mocks (testify + gomock)
- ✅ Integration tests (PostgreSQL + Redis)
- ✅ Table-driven tests (Go best practice)
- ✅ Security tests (timing attacks, injection)
- ✅ Concurrent access tests (race detector)
- 🚧 E2E tests (planned for Phase 2)

**Quality Tools:**
- `golangci-lint` - Comprehensive linting
- `gosec` - Security scanning
- `gofumpt` - Strict formatting
- `mockgen` - Mock generation for testing

---

## 🗺️ Roadmap

**Overall Progress:** 30% (28/58 tasks across 4 phases)

### Phase 1: User Service ✅ **100% Complete**
- ✅ Authentication & Authorization (JWT, Argon2id)
- ✅ User Management (CRUD operations)
- ✅ Vault Integration (secrets management)
- ✅ Rate Limiting & Audit Logging
- ✅ Observability (metrics, tracing, logging)
- ✅ CI/CD Pipeline
- ✅ Comprehensive Documentation

### Phase 2: Wallet Service (Q1 2025)
- Multi-currency wallets (BTC, ETH, USDT)
- HSM key management
- Deposit/withdrawal processing
- Blockchain integration (Bitcoin, Ethereum nodes)

### Phase 3: Trading Engine (Q2-Q3 2025)
- High-performance matching engine
- Order book management
- Real-time market data (WebSocket)
- Order types: limit, market, stop-loss

### Phase 4: Advanced Features (Q4 2025)
- Margin trading (2x, 5x, 10x leverage)
- Advanced order types (OCO, trailing stop, iceberg)
- API keys for trading bots
- Admin dashboard & analytics

> 🗺️ See [ROADMAP.md](./docs/ROADMAP.md) for detailed task breakdown and timelines.

---

## 📚 Documentation

### Getting Started
- 🚀 [Quick Start Guide](./docs/QUICK_START.md) - Installation, configuration, development
- 📡 [API Documentation](./docs/API_DOCUMENTATION.md) - Complete REST & gRPC reference
- 🏗️ [Architecture](./docs/ARCHITECTURE.md) - System design, patterns, data models

### Developer Guides
- 🤝 [Contributing](./docs/CONTRIBUTING.md) - Workflow, code standards, PR process
- 🧪 [Testing Guide](./docs/testing.md) - Testing strategy, examples, coverage
- 🔐 [Security](./docs/SECURITY.md) - Security architecture, best practices

### Operations
- 🐳 [Docker Guide](./docs/DOCKER.md) - Container setup, Docker Compose
- ☸️ [Kubernetes](./deployments/k8s/README.md) - Deployment manifests
- 🔄 [CI/CD](./docs/CI_CD.md) - GitHub Actions pipeline

### Reference
- 🗺️ [Roadmap](./docs/ROADMAP.md) - Project phases, milestones, progress
- 📖 [Error Codes](./docs/errors.md) - Complete error catalog
- 📊 [Metrics](./docs/observability/prometheus-metrics.md) - Prometheus metrics catalog

---

## 🤝 Contributing

We welcome contributions! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests (TDD approach - we maintain 80%+ coverage)
4. Commit changes (`git commit -m 'feat(auth): add 2FA support'`)
5. Push to branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

**Code Standards:**
- ✅ Follow [Conventional Commits](https://www.conventionalcommits.org/)
- ✅ Maintain 80%+ test coverage
- ✅ Pass `golangci-lint` checks
- ✅ Use `gofumpt` for formatting
- ✅ Update documentation for new features

> 🤝 See [Contributing Guide](./docs/CONTRIBUTING.md) for complete guidelines.

---

## 📄 License

This project is licensed under the **MIT License** - see [LICENSE](LICENSE) file.

---

## 👥 Team

**Maintainer:** Alex Necsoiu - [@alexnecsoiu](https://github.com/alexnecsoiu)

**Contributors:** See [CONTRIBUTORS.md](CONTRIBUTORS.md)

---

## 📞 Contact

- **Email:** dev@pandora-exchange.com
- **Security:** security@pandora-exchange.com
- **Issues:** [GitHub Issues](https://github.com/pandora-exchange/pandora-exchange/issues)
- **Discussions:** [GitHub Discussions](https://github.com/pandora-exchange/pandora-exchange/discussions)

---

<div align="center">

**Built with ❤️ by the Pandora Exchange Team**

⭐ Star us on GitHub if you find this project useful!

[⬆ Back to Top](#-pandora-exchange)

</div>
