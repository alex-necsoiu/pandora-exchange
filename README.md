<div align="center">

<img src="docs/assets/logo.svg" alt="Pandora Exchange Logo" width="200"/>

# 🚀 Pandora Exchange

**Enterprise-Grade Cryptocurrency Exchange Platform**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Test Coverage](https://img.shields.io/badge/Coverage-85%25-brightgreen)](./docs/testing.md)
[![Build Status](https://img.shields.io/badge/Build-Passing-success)](https://github.com/pandora-exchange/pandora-exchange/actions)

[Features](#-features) • [Architecture](#-architecture) • [Quick Start](#-quick-start) • [Documentation](#-documentation)

</div>

---

## 📖 Overview

**Pandora Exchange** is a secure, scalable cryptocurrency exchange platform built with **Clean Architecture** principles.

**Status:** 🟡 **Phase 1 - 96% Complete** (27/28 tasks) | 🎯 **Production Ready**

---

## ✨ Features

### 🔐 Security & Authentication
- Argon2id password hashing
- JWT with refresh token rotation  
- HashiCorp Vault secrets management
- Redis-backed rate limiting
- Immutable audit logs (7-year retention)

### 🏗️ Clean Architecture
- Domain-Driven Design (DDD)
- Dependency Injection
- Repository Pattern
- Service Layer business logic
- REST + gRPC APIs

### 📊 Observability  
- Prometheus metrics
- OpenTelemetry distributed tracing
- Structured logging (zerolog)
- Health checks

### 🚀 Production Ready
- Docker containerization
- Kubernetes deployment manifests
- CI/CD pipeline (GitHub Actions)
- Swagger/OpenAPI documentation
- Multi-environment support

---

## 🏛️ Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for complete system design.

---

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### Get Running

```bash
# Clone repository
git clone https://github.com/pandora-exchange/pandora-exchange.git
cd pandora-exchange

# Start infrastructure
make dev-up

# Run migrations
make migrate-up

# Start service
make run

# Test
curl http://localhost:8080/health
```

**Service:** `http://localhost:8080`  
**Swagger:** `http://localhost:8080/swagger`  
**Metrics:** `http://localhost:8080/metrics`

📖 See [Quick Start Guide](./docs/QUICK_START.md) for detailed setup.

---

## 🗺️ Roadmap

**Progress:** 30% (27/58 tasks)

### Phase 1: User Service (🟡 96%)
- ✅ Authentication & Authorization
- ✅ User Management
- ✅ Vault Integration
- ✅ Observability
- 🔄 Documentation

### Phase 2: Wallet Service (⚪ Q1 2025)
### Phase 3: Trading Engine (⚪ Q2-Q3 2025)
### Phase 4: Advanced Features (⚪ Q4 2025)

📋 See [ROADMAP.md](./docs/ROADMAP.md) for complete details.

---

## 📚 Documentation

### Getting Started
- 🚀 [Quick Start Guide](./docs/QUICK_START.md)
- 📡 [API Documentation](./docs/API_DOCUMENTATION.md)  
- 🏗️ [Architecture](./ARCHITECTURE.md)

### Developer Guides
- 🤝 [Contributing](./docs/CONTRIBUTING.md)
- 🧪 [Testing](./docs/testing.md)
- 🔐 [Security](./docs/SECURITY.md)

### Operations
- 🐳 [Docker](./docs/DOCKER.md)
- ☸️ [Kubernetes](./deployments/k8s/README.md)
- 🔄 [CI/CD](./docs/CI_CD.md)

---

## 🤝 Contributing

We welcome contributions! See [Contributing Guide](./docs/CONTRIBUTING.md).

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file.

---

<div align="center">

**Built with ❤️ by the Pandora Exchange Team**

</div>
