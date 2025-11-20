# 📚 Pandora Exchange Documentation

> **Enterprise-Grade Documentation for Pandora Exchange Backend**  
> **Version:** 1.0  
> **Last Updated:** November 8, 2025  
> **Scope:** User Service (Phase 1)

---

## 🎯 Quick Navigation

| Section | Description | Status |
|---------|-------------|--------|
| [Architecture](#architecture) | System design & principles | ✅ Complete |
| [Services](#services) | Microservice documentation | ✅ User Service |
| [API Reference](#api-reference) | REST & gRPC endpoints | ✅ REST, 🔄 gRPC |
| [Security](#security) | Auth, Vault, compliance | ✅ Complete |
| [Testing](#testing) | TDD guidelines & patterns | ✅ Complete |
| [Database](#database) | Schema, migrations, conventions | ✅ Complete |
| [Runbooks](#runbooks) | SRE procedures | ✅ Complete |
| [Diagrams](#diagrams) | Architecture visualizations | ✅ Complete |

**Legend:** ✅ Complete | 🔄 In Progress | ⚪ Planned

---

## 📖 Documentation Structure

```
docs/
├── README.md                    # ← You are here
├── services/
│   └── user-service.md         # User Service complete documentation
├── api/
│   ├── openapi/                # Swagger/OpenAPI specs (auto-generated)
│   └── rest-examples.md        # Curl examples (planned)
├── security/
│   ├── README.md               # Security overview
│   └── vault.md                # Vault integration guide
├── runbooks/
│   ├── README.md               # Runbook index
│   └── deployment.md           # Deployment procedures
├── db/
│   └── migrations.md           # Migration guidelines
└── diagrams/
    ├── README.md               # Diagram index
    ├── user-registration.mmd   # User registration flow
    ├── authentication.mmd      # Authentication flow
    ├── clean-architecture.mmd  # Layer dependencies
    └── event-flow.mmd          # Event publishing
```

---

## 🏛️ Architecture

### Core Documents
- **[ARCHITECTURE.md](../ARCHITECTURE.md)** - Complete architecture specification (single source of truth)
- **[Clean Architecture Diagram](diagrams/clean-architecture.mmd)** - Layer dependencies visualization

### Key Principles
- **Clean Architecture** - Domain-centric design, dependency inversion
- **TDD-First** - All code written test-first
- **Event-Driven** - Redis Streams for async communication
- **Security-First** - Vault secrets, Argon2id hashing, JWT auth
- **Observability** - OpenTelemetry tracing throughout

### Technology Stack
| Layer | Technology |
|-------|------------|
| **Language** | Go 1.24+ |
| **Web Framework** | Gin (REST) |
| **Internal RPC** | gRPC (planned) |
| **Database** | PostgreSQL 15+ |
| **Data Access** | sqlc (SQL-first code generation) |
| **Cache/Events** | Redis 7+ (cache + streams) |
| **Auth** | JWT + Argon2id |
| **Secrets** | HashiCorp Vault |
| **Observability** | OpenTelemetry + Zerolog |
| **Orchestration** | Kubernetes |

---

## 🔧 Services

### User Service (Phase 1)
**[Complete Documentation →](services/user-service.md)**

**Responsibilities:**
- User registration & authentication
- JWT access/refresh token management
- KYC status tracking
- Profile management
- Admin operations
- Audit logging

**Status:** ✅ Production Ready

**Key Metrics:**
- Test Coverage: >85%
- API Endpoints: 15+ REST endpoints
- Database Tables: 3 (users, refresh_tokens, audit_logs)
- Event Types: 6 domain events

---

## 📡 API Reference

### REST API (Gin)
**Base URL:** `http://localhost:8080/api/v1`

**Swagger UI:** Generate with `make docs`, then open `docs/api/openapi/docs/index.html`

**Quick Reference:**

| Method | Endpoint | Description | Auth |
|--------|----------|-------------|------|
| POST | `/auth/register` | Register new user | None |
| POST | `/auth/login` | Login user | None |
| POST | `/auth/refresh` | Refresh access token | Refresh Token |
| GET | `/users/me` | Get current user profile | JWT |
| PUT | `/users/me` | Update profile | JWT |
| PATCH | `/users/me/kyc` | Update KYC status | JWT |
| DELETE | `/users/me` | Delete account | JWT |
| GET | `/admin/users` | List all users | Admin JWT |
| GET | `/admin/users/:id` | Get user by ID | Admin JWT |
| PUT | `/admin/users/:id/kyc` | Admin update KYC | Admin JWT |
| DELETE | `/admin/users/:id` | Admin delete user | Admin JWT |
| GET | `/health` | Health check | None |

**Authentication:**
- **Access Token:** JWT (15 min expiry) in `Authorization: Bearer <token>` header
- **Refresh Token:** Long-lived (7 days) for obtaining new access tokens
- **Admin Role:** Required for `/admin/*` endpoints

**Detailed Specs:**
- [OpenAPI/Swagger](api/openapi/) - Auto-generated from code annotations
- [Curl Examples](api/rest-examples.md) - Coming soon

### gRPC API (Internal)
**Port:** `9090` (configurable via `GRPC_PORT`)

**Status:** 🔄 Planned - Proto definitions coming soon

**Planned Services:**
```protobuf
service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc GetUserByEmail(GetUserByEmailRequest) returns (GetUserResponse);
  rpc UpdateKYCStatus(UpdateKYCRequest) returns (UpdateKYCResponse);
  rpc ValidateUser(ValidateUserRequest) returns (ValidateUserResponse);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}
```

---

## 🔐 Security

**[Security Overview →](security/README.md)**

### Key Security Features
- **Password Hashing:** Argon2id (time=1, memory=64MB, threads=4, salt=16 bytes)
- **Authentication:** JWT access tokens (15 min) + refresh tokens (7 days)
- **Secrets Management:** HashiCorp Vault (no credentials in code/env)
- **TLS:** Required for all external communication in production
- **Audit Logging:** Immutable logs for all sensitive operations
- **PII Protection:** Redaction in logs, secure deletion

### Documentation
- **[Vault Integration](../VAULT_INTEGRATION.md)** - Complete Vault setup guide
- **[Vault Testing](../internal/vault/TESTING.md)** - Vault integration testing
- **[Error Handling](../ERROR_HANDLING.md)** - Secure error responses
- **[Audit Policy](../AUDIT_RETENTION_POLICY.md)** - Audit log retention

### Security Compliance
- ✅ No secrets in code or environment files
- ✅ All passwords hashed with Argon2id
- ✅ JWT tokens with short expiry
- ✅ Audit trail for all user actions
- ✅ PII redacted from logs
- ✅ TLS enforced in production

---

## 🧪 Testing

**[Complete Testing Guide →](testing.md)**

### Testing Philosophy
**TDD-First:** Tests are written BEFORE production code in all cases.

### Test Coverage by Package
| Package | Unit Tests | Integration Tests | Coverage |
|---------|-----------|-------------------|----------|
| domain | ✅ 25+ tests | - | >90% |
| repository | ✅ 48+ tests | ✅ 4 workflows | >85% |
| service | ✅ 40+ tests | - | >90% |
| transport/http | ✅ 45+ tests | - | >85% |
| middleware | ✅ 40+ tests | - | >90% |
| events | ✅ 17+ tests | - | >92% |
| vault | ✅ 3 tests | ✅ 6 suites | >80% |
| **TOTAL** | **218+ tests** | **10 suites** | **>85%** |

### Quick Commands
```bash
# Run all tests
make test

# Run with coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package
go test ./internal/domain/... -v

# Run integration tests
go test ./tests/integration/... -v

# Run Vault integration tests (requires vault binary)
VAULT_INTEGRATION_TESTS=true go test ./internal/vault/... -v
```

### Test Documentation
- **[Testing Guidelines](testing.md)** - TDD rules, patterns, coverage targets
- **[Test Reports](../ADMIN_TEST_COVERAGE_REPORT.md)** - Admin workflow coverage
- **[gRPC Test Coverage](../GRPC_TEST_COVERAGE_REPORT.md)** - gRPC layer testing

---

## 🗄️ Database

**[Migration Guidelines →](db/migrations.md)**

### Schema
**Current Tables:**
- `users` - User accounts with auth credentials
- `refresh_tokens` - JWT refresh token storage
- `audit_logs` - Immutable audit trail

**Migrations:** 5 migrations applied (users, tokens, name split, roles, audit)

### Data Access
- **Tool:** sqlc (SQL-first code generation)
- **Pattern:** Repository pattern wrapping sqlc generated code
- **Rule:** No raw SQL in application code, no sqlc structs in API responses

### Quick Commands
```bash
# Run migrations
make migrate

# Rollback one migration
make migrate-down

# Generate sqlc code
make sqlc

# Create new migration
migrate create -ext sql -dir migrations -seq <migration_name>
```

### Documentation
- **[Migration Guide](db/migrations.md)** - How to write safe migrations
- **[Schema Conventions](db/migrations.md#naming-conventions)** - Naming rules

---

## 📋 Runbooks

**[Runbook Index →](runbooks/README.md)**

### Available Runbooks
- **[Deployment](runbooks/deployment.md)** - How to deploy to each environment
- **[Rollback](runbooks/deployment.md#rollback-procedures)** - Safe rollback procedures
- **[Database Migrations](runbooks/deployment.md#database-migrations)** - Migration safety
- **[Debugging](runbooks/deployment.md#debugging)** - Debug services in production
- **[Incident Response](runbooks/deployment.md#incident-response)** - Recovery procedures

### Quick Links
- **Emergency Contacts:** See Slack #pandora-ops
- **Monitoring:** Grafana dashboard (link TBD)
- **Logs:** Kubernetes logs via `kubectl logs`
- **Tracing:** OpenTelemetry collector (configured)

---

## 📊 Diagrams

**[Diagram Index →](diagrams/README.md)**

### Architecture Diagrams
All diagrams use Mermaid format for version control and rendering in GitHub/GitLab.

**Available Diagrams:**
- **[Clean Architecture](diagrams/clean-architecture.mmd)** - Layer dependencies
- **[User Registration Flow](diagrams/user-registration.mmd)** - Registration sequence
- **[Authentication Flow](diagrams/authentication.mmd)** - Login & token refresh
- **[Event Publishing Flow](diagrams/event-flow.mmd)** - Redis Streams events

**Viewing Diagrams:**
- **GitHub/GitLab:** Renders automatically in web UI
- **VS Code:** Install Mermaid Preview extension
- **CLI:** Use `mmdc` (mermaid-cli) to generate PNG/SVG

---

## 🚀 Getting Started

### For Developers
1. Read **[ARCHITECTURE.md](../ARCHITECTURE.md)** - Mandatory
2. Review **[User Service Docs](services/user-service.md)** - Understand current service
3. Check **[Testing Guidelines](testing.md)** - Learn TDD approach
4. Review **[Clean Architecture Diagram](diagrams/clean-architecture.mmd)** - Understand layers
5. Start coding with tests first!

### For SREs/DevOps
1. Read **[Deployment Runbook](runbooks/deployment.md)** - Deployment procedures
2. Review **[Kubernetes Manifests](../deployments/k8s/)** - Infrastructure setup
3. Check **[Vault Integration](../VAULT_INTEGRATION.md)** - Secret management
4. Review **[Database Migrations](db/migrations.md)** - Migration safety

### For QA/Testing
1. Read **[Testing Guidelines](testing.md)** - Test philosophy
2. Review **[Error Catalog](../ERROR_HANDLING.md)** - Expected error responses
3. Check **[API Reference](#api-reference)** - Endpoint documentation
4. Review **[Sequence Diagrams](diagrams/)** - Understand flows

---

## 🔄 Documentation Maintenance

### How to Update Docs

**For Code Changes:**
1. Update GoDoc comments in code
2. Update Swagger annotations if API changed
3. Run `make docs` to regenerate OpenAPI spec
4. Update sequence diagrams if flow changed
5. Update relevant service documentation

**For Architecture Changes:**
1. **MUST** get approval before changing ARCHITECTURE.md
2. Update affected service docs
3. Update diagrams
4. Update this README navigation

**For New Features:**
1. Write tests first (TDD)
2. Add GoDoc comments to new functions
3. Add Swagger annotations to new endpoints
4. Update service documentation
5. Add/update sequence diagrams if needed
6. Update error catalog if new errors added

### Documentation Standards
- ✅ All public functions have GoDoc comments
- ✅ All HTTP endpoints have Swagger annotations
- ✅ All gRPC methods have proto comments
- ✅ Complex flows have sequence diagrams
- ✅ Error codes documented in error catalog
- ✅ Examples provided for all endpoints

---

## 📞 Support & Contributing

### Getting Help
- **Architecture Questions:** Re-read [ARCHITECTURE.md](../ARCHITECTURE.md)
- **Code Issues:** Check existing tests for examples
- **Deployment Issues:** See [Runbooks](runbooks/)
- **Security Questions:** See [Security Docs](security/)

### Contributing
1. Read [ARCHITECTURE.md](../ARCHITECTURE.md) - **Mandatory**
2. Follow TDD - Write tests FIRST
3. Add GoDoc comments to all functions
4. Add Swagger annotations to HTTP handlers
5. Update relevant documentation
6. Use conventional commits: `feat:`, `fix:`, `docs:`, `test:`
7. Ensure all tests pass: `make test`

---

## 📄 License

Pandora Exchange Backend - Proprietary

---

**Last Updated:** November 8, 2025  
**Maintained By:** Pandora Engineering Team  
**Questions?** Open an issue or contact #pandora-backend on Slack
