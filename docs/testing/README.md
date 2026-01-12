# Testing Documentation

Test coverage reports and testing guides.

## 📊 Coverage Overview

| Layer | Target | Current Status |
|-------|--------|----------------|
| **Domain** | 100% | ✅ 100% |
| **Repository** | 80% | ✅ 78.1% |
| **Service** | 100% | ✅ 100% |
| **HTTP Handlers** | 90% | ✅ 91.7% |
| **gRPC Handlers** | 100% | ✅ 100% |
| **Middleware** | 100% | ✅ 100% |

## 🧪 Running Tests

```bash
# Run all tests
make test

# Run with coverage report
make test-coverage

# Run integration tests
make test-integration

# Run benchmarks
make benchmark
```

## 📁 Test Structure

```
tests/
├── integration/          # End-to-end integration tests
└── ...

internal/
├── domain/*_test.go      # Domain logic tests
├── service/*_test.go     # Service layer tests
├── repository/*_test.go  # Repository tests
└── transport/
    ├── http/*_test.go    # HTTP handler tests
    └── grpc/*_test.go    # gRPC handler tests
```

## Related Documentation

- [Contributing Guide](../CONTRIBUTING.md) - How to write tests
- [CI/CD Pipeline](../CI_CD.md) - Automated testing in CI
