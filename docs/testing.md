# Testing Guidelines

> **TDD-First Testing Strategy for Pandora Exchange**  
> **Last Updated:** November 8, 2025

---

## Testing Philosophy

**Test-Driven Development (TDD) is MANDATORY for all code at Pandora Exchange.**

### The TDD Cycle

```
1. Write Test (RED) → 2. Write Code (GREEN) → 3. Refactor (REFACTOR)
      ↑                                                   ↓
      └──────────────────────────────────────────────────┘
```

**Rules:**
- ✅ **Write tests FIRST** - Before any production code
- ✅ **One test at a time** - Small incremental steps
- ✅ **Run tests frequently** - After each code change
- ✅ **Refactor with confidence** - Tests ensure correctness
- ❌ **Never commit untested code** - 100% of new code must have tests

---

## Test Coverage Goals

| Package Type | Minimum Coverage | Target Coverage |
|--------------|------------------|-----------------|
| Domain Logic | 90% | 95%+ |
| Services | 85% | 90%+ |
| Repositories | 85% | 90%+ |
| HTTP Handlers | 80% | 85%+ |
| Middleware | 90% | 95%+ |
| Overall | 85% | 90%+ |

**Current Status:** >85% coverage maintained across all packages

---

## Test Types

### 1. Unit Tests

**Purpose:** Test individual functions/methods in isolation

**Characteristics:**
- Fast (<10ms per test)
- No external dependencies (mock DB, Redis, etc.)
- Test one thing at a time
- Use table-driven tests for multiple scenarios

**Location:** Next to production code (`*_test.go`)

**Example:**
```go
// internal/domain/models_test.go
func TestUser_Validate(t *testing.T) {
    tests := []struct {
        name    string
        user    domain.User
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid user",
            user: domain.User{
                Email:     "user@example.com",
                FirstName: "John",
                LastName:  "Doe",
            },
            wantErr: false,
        },
        {
            name: "invalid email",
            user: domain.User{
                Email:     "invalid-email",
                FirstName: "John",
            },
            wantErr: true,
            errMsg:  "invalid email format",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.user.Validate()
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

---

### 2. Integration Tests

**Purpose:** Test complete request lifecycle with real dependencies

**Characteristics:**
- Slower (<1s per test)
- Use real database (testcontainers or dedicated test DB)
- Test end-to-end workflows
- Clean up after each test

**Location:** `/tests/integration/`

**Example:**
```go
// tests/integration/admin_test.go
func TestAdminWorkflow_CompleteLifecycle(t *testing.T) {
    // Setup: Start real DB, Redis, service
    db := setupTestDB(t)
    defer db.Close()
    
    // 1. Admin login
    adminToken := loginAsAdmin(t, db)
    
    // 2. Create user
    user := createTestUser(t, db)
    
    // 3. Admin updates KYC
    updateKYC(t, adminToken, user.ID, "verified")
    
    // 4. Verify KYC updated
    updatedUser := getUser(t, adminToken, user.ID)
    assert.Equal(t, "verified", updatedUser.KYCStatus)
}
```

---

### 3. Table-Driven Tests

**Purpose:** Test multiple scenarios with same logic

**Pattern:**
```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantErr  bool
    }{
        {name: "scenario 1", input: ..., expected: ..., wantErr: false},
        {name: "scenario 2", input: ..., expected: ..., wantErr: true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

**Benefits:**
- Easy to add new test cases
- Clear test documentation
- Reduces code duplication

---

## Mocking Strategy

### Use Interfaces for Dependencies

**✅ Good:**
```go
// internal/domain/repository.go
type UserRepository interface {
    Create(ctx context.Context, params CreateUserParams) (User, error)
    GetByID(ctx context.Context, id string) (User, error)
    GetByEmail(ctx context.Context, email string) (User, error)
}

// internal/service/user_service.go
type UserService struct {
    repo UserRepository // Interface, not concrete type
}
```

**❌ Bad:**
```go
// Concrete dependency - hard to test
type UserService struct {
    repo *postgres.UserRepository
}
```

---

### Generate Mocks with mockgen

**Install:**
```bash
go install go.uber.org/mock/mockgen@latest
```

**Generate:**
```bash
mockgen -source=internal/domain/repository.go \
        -destination=internal/mocks/mock_repository.go \
        -package=mocks
```

**Usage:**
```go
func TestUserService_Register(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    // Create mock
    mockRepo := mocks.NewMockUserRepository(ctrl)
    
    // Set expectations
    mockRepo.EXPECT().
        GetByEmail(gomock.Any(), "user@example.com").
        Return(domain.User{}, domain.ErrUserNotFound)
    
    mockRepo.EXPECT().
        Create(gomock.Any(), gomock.Any()).
        Return(domain.User{ID: "uuid"}, nil)
    
    // Test service with mock
    svc := service.NewUserService(mockRepo, nil, nil)
    user, err := svc.Register(context.Background(), "user@example.com", "password")
    
    require.NoError(t, err)
    assert.NotEmpty(t, user.ID)
}
```

---

## Test Assertions

### Use testify/assert and testify/require

**assert:** Continue test on failure  
**require:** Stop test on failure

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
    result, err := someFunction()
    
    // STOP if error (no point continuing)
    require.NoError(t, err)
    
    // CONTINUE testing (multiple assertions)
    assert.Equal(t, "expected", result.Field1)
    assert.True(t, result.Field2)
    assert.Len(t, result.Items, 3)
}
```

---

## Testing Database Code

### Use Transactions for Isolation

```go
func TestUserRepository_Create(t *testing.T) {
    db := setupTestDB(t)
    
    // Start transaction
    tx, err := db.Begin(context.Background())
    require.NoError(t, err)
    defer tx.Rollback(context.Background()) // Always rollback
    
    // Create repository with transaction
    repo := repository.NewUserRepository(tx)
    
    // Test
    user, err := repo.Create(ctx, params)
    require.NoError(t, err)
    assert.NotEmpty(t, user.ID)
    
    // Rollback happens automatically
}
```

---

### Clean Test Data

```go
func setupTestDB(t *testing.T) *pgxpool.Pool {
    db, err := pgxpool.Connect(context.Background(), testDSN)
    require.NoError(t, err)
    
    // Clean tables before tests
    _, err = db.Exec(context.Background(), `
        TRUNCATE users, refresh_tokens, audit_logs RESTART IDENTITY CASCADE
    `)
    require.NoError(t, err)
    
    return db
}
```

---

## Testing HTTP Handlers

### Use httptest Package

```go
func TestRegisterHandler(t *testing.T) {
    // Setup
    mockService := mocks.NewMockUserService(gomock.NewController(t))
    handler := NewUserHandler(mockService)
    
    router := gin.New()
    router.POST("/register", handler.Register)
    
    // Create request
    body := `{"email":"user@example.com","password":"password123"}`
    req := httptest.NewRequest("POST", "/register", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    
    // Record response
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // Assert
    assert.Equal(t, http.StatusCreated, w.Code)
    
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(t, "user@example.com", response["email"])
}
```

---

## Testing Middleware

### Test Middleware in Isolation

```go
func TestAuthMiddleware(t *testing.T) {
    tests := []struct {
        name           string
        authHeader     string
        expectedStatus int
    }{
        {
            name:           "valid token",
            authHeader:     "Bearer valid-token",
            expectedStatus: http.StatusOK,
        },
        {
            name:           "missing token",
            authHeader:     "",
            expectedStatus: http.StatusUnauthorized,
        },
        {
            name:           "invalid token",
            authHeader:     "Bearer invalid-token",
            expectedStatus: http.StatusUnauthorized,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            w := httptest.NewRecorder()
            c, _ := gin.CreateTestContext(w)
            
            // Set auth header
            c.Request = httptest.NewRequest("GET", "/", nil)
            if tt.authHeader != "" {
                c.Request.Header.Set("Authorization", tt.authHeader)
            }
            
            // Apply middleware
            AuthMiddleware()(c)
            
            // Assert
            assert.Equal(t, tt.expectedStatus, w.Code)
        })
    }
}
```

---

## Test Coverage

### Measure Coverage

```bash
# Run tests with coverage
go test ./... -coverprofile=coverage.out

# View coverage report in terminal
go tool cover -func=coverage.out

# View coverage report in browser
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Coverage Report Example

```
internal/domain/models.go:15:     Validate         100.0%
internal/domain/errors.go:10:     Error            100.0%
internal/service/user_service.go:50: Register      95.2%
internal/service/user_service.go:80: Login         90.5%
internal/repository/user_repository.go:30: Create   88.9%
------------------------------------------------
total:                            (statements)     92.3%
```

---

## Running Tests

### Quick Commands

```bash
# Run all tests
make test

# Run specific package
go test ./internal/domain/... -v

# Run specific test
go test ./internal/domain/... -run TestUser_Validate -v

# Run with coverage
go test ./... -cover

# Run integration tests only
go test ./tests/integration/... -v

# Run with race detector
go test ./... -race

# Run in parallel
go test ./... -parallel=4

# Verbose output
go test ./... -v

# Fail fast (stop on first failure)
go test ./... -failfast
```

---

### Makefile Targets

```makefile
# In Makefile
.PHONY: test
test:
	go test ./... -v -cover

.PHONY: test-coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-integration
test-integration:
	go test ./tests/integration/... -v

.PHONY: test-race
test-race:
	go test ./... -race

.PHONY: test-vault
test-vault:
	VAULT_INTEGRATION_TESTS=true go test ./internal/vault/... -v
```

---

## Test Organization

### Directory Structure

```
internal/
├── domain/
│   ├── models.go
│   ├── models_test.go           # Unit tests
│   ├── errors.go
│   └── errors_test.go
├── service/
│   ├── user_service.go
│   ├── user_service_test.go     # Unit tests with mocks
│   ├── audit_cleanup_job.go
│   └── audit_cleanup_job_test.go
├── repository/
│   ├── user_repository.go
│   ├── user_repository_test.go  # DB integration tests
│   └── *_test.go
├── transport/http/
│   ├── user_handlers.go
│   ├── user_handlers_test.go    # HTTP integration tests
│   └── *_test.go
└── mocks/
    ├── mock_repository.go       # Generated mocks
    ├── mock_event_publisher.go
    └── mock_*.go

tests/
└── integration/
    ├── admin_test.go            # Full E2E tests
    ├── user_workflow_test.go
    └── *_test.go
```

---

## Test Documentation

### Write Clear Test Names

**✅ Good:**
```go
func TestUserService_Register_WithValidInput_CreatesUser(t *testing.T)
func TestUserService_Register_WithExistingEmail_ReturnsError(t *testing.T)
func TestUserService_Login_WithInvalidPassword_ReturnsUnauthorized(t *testing.T)
```

**❌ Bad:**
```go
func TestRegister(t *testing.T)
func TestRegister2(t *testing.T)
func TestLogin1(t *testing.T)
```

---

### Document Complex Test Logic

```go
func TestAuditCleanupJob_DeletesOldLogs(t *testing.T) {
    // GIVEN: Audit logs older than retention period exist
    db := setupTestDB(t)
    createOldAuditLogs(t, db, 100) // 100 logs > 90 days old
    createRecentAuditLogs(t, db, 50) // 50 logs < 90 days old
    
    // WHEN: Cleanup job runs
    job := service.NewAuditCleanupJob(db, 90*24*time.Hour)
    deleted, err := job.Run(context.Background())
    
    // THEN: Old logs are deleted, recent logs remain
    require.NoError(t, err)
    assert.Equal(t, 100, deleted)
    
    remaining := countAuditLogs(t, db)
    assert.Equal(t, 50, remaining)
}
```

---

## Common Testing Patterns

### 1. Setup and Teardown

```go
func TestMain(m *testing.M) {
    // Setup before all tests
    setupGlobalResources()
    
    // Run tests
    code := m.Run()
    
    // Teardown after all tests
    teardownGlobalResources()
    
    os.Exit(code)
}

func TestExample(t *testing.T) {
    // Setup before this test
    db := setupTestDB(t)
    
    // Cleanup after this test
    defer func() {
        db.Close()
    }()
    
    // Test logic
}
```

---

### 2. Test Helpers

```go
// helpers_test.go
func createTestUser(t *testing.T, db *pgxpool.Pool) domain.User {
    t.Helper() // Mark as helper
    
    user, err := createUser(db, "test@example.com", "password")
    require.NoError(t, err)
    return user
}

func assertUserExists(t *testing.T, db *pgxpool.Pool, email string) {
    t.Helper()
    
    var exists bool
    err := db.QueryRow(context.Background(),
        "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
    require.NoError(t, err)
    assert.True(t, exists, "user should exist")
}
```

---

### 3. Golden File Testing

For testing complex outputs:

```go
func TestGenerateReport(t *testing.T) {
    result := GenerateReport(inputData)
    
    goldenFile := "testdata/report.golden"
    
    if os.Getenv("UPDATE_GOLDEN") == "true" {
        // Update golden file
        os.WriteFile(goldenFile, []byte(result), 0644)
    }
    
    expected, err := os.ReadFile(goldenFile)
    require.NoError(t, err)
    assert.Equal(t, string(expected), result)
}
```

---

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run tests
        run: make test
      
      - name: Upload coverage
        run: bash <(curl -s https://codecov.io/bash)
```

---

## Testing Best Practices

### ✅ DO

- Write tests FIRST (TDD)
- Use table-driven tests for multiple scenarios
- Test both success and failure cases
- Use meaningful test names
- Keep tests independent (no shared state)
- Clean up after tests (defer, transactions)
- Use mocks for external dependencies
- Test edge cases and boundary conditions
- Maintain >85% code coverage

### ❌ DON'T

- Skip writing tests
- Test implementation details
- Use sleep() for timing (use channels/contexts)
- Share mutable state between tests
- Ignore test failures
- Write flaky tests
- Test framework code (Gin, sqlc, etc.)
- Commit code with failing tests

---

## References

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [gomock Documentation](https://github.com/uber-go/mock)
- [Table-Driven Tests](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Test Reports](../ADMIN_TEST_COVERAGE_REPORT.md)
- [Vault Testing Guide](../internal/vault/TESTING.md)

---

**Last Updated:** November 8, 2025  
**Maintained By:** Pandora Engineering Team  
**Questions?** Ask in #pandora-backend on Slack
