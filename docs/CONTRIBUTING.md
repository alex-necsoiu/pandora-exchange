# 🤝 Contributing to Pandora Exchange

Thank you for your interest in contributing to Pandora Exchange! This guide will help you get started.

## 📋 Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Commit Guidelines](#commit-guidelines)
- [Pull Request Process](#pull-request-process)
- [Code Review Checklist](#code-review-checklist)
- [Project Conventions](#project-conventions)

---

## Getting Started

### Prerequisites

Before contributing, ensure you have:
- Go 1.21 or later
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- Make 4.0+

See [Quick Start Guide](./QUICK_START.md) for detailed setup instructions.

### First-Time Setup

```bash
# 1. Fork the repository on GitHub
# 2. Clone your fork
git clone https://github.com/YOUR_USERNAME/pandora-exchange.git
cd pandora-exchange

# 3. Add upstream remote
git remote add upstream https://github.com/pandora-exchange/pandora-exchange.git

# 4. Install dependencies
make deps

# 5. Start development environment
make dev-up

# 6. Run migrations
make migrate-up

# 7. Generate code
make generate

# 8. Run tests
make test
```

---

## Development Workflow

### 1. Create a Feature Branch

```bash
# Update your fork
git checkout main
git pull upstream main

# Create feature branch
git checkout -b feature/your-feature-name
```

**Branch Naming Conventions:**

| Type | Pattern | Example |
|------|---------|---------|
| **Feature** | `feature/<description>` | `feature/add-2fa-support` |
| **Bugfix** | `fix/<description>` | `fix/login-rate-limit` |
| **Hotfix** | `hotfix/<description>` | `hotfix/critical-auth-bug` |
| **Docs** | `docs/<description>` | `docs/update-api-docs` |
| **Refactor** | `refactor/<description>` | `refactor/extract-middleware` |
| **Test** | `test/<description>` | `test/add-integration-tests` |

### 2. Write Tests First (TDD)

We follow **Test-Driven Development (TDD)**:

```bash
# 1. Write failing test
vim internal/service/user_service_test.go

# 2. Run tests (should fail)
make test

# 3. Implement feature
vim internal/service/user_service.go

# 4. Run tests (should pass)
make test

# 5. Refactor if needed
# 6. Run tests again (should still pass)
make test
```

### 3. Run Quality Checks

```bash
# Run all checks before committing
make pre-commit

# Individual checks
make test          # Run all tests
make lint          # Run linters
make fmt           # Format code
make vet           # Run go vet
make security      # Security scan
make coverage      # Check coverage
```

### 4. Commit Your Changes

```bash
# Stage changes
git add .

# Commit with conventional commit message
git commit -m "feat(auth): add two-factor authentication support"

# Push to your fork
git push origin feature/your-feature-name
```

### 5. Create Pull Request

1. Go to your fork on GitHub
2. Click "New Pull Request"
3. Select `main` as base branch
4. Fill in PR template
5. Link related issues
6. Request review

---

## Code Standards

### Go Style Guide

**Follow:**
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### Code Formatting

```bash
# Format code (automatically)
make fmt

# This runs:
gofumpt -l -w .
gci write --skip-generated -s standard -s default -s "prefix(github.com/pandora-exchange)" .
```

**Rules:**
- Use `gofumpt` (stricter than `gofmt`)
- Import ordering: stdlib → external → internal
- Max line length: 120 characters
- No trailing whitespace
- Unix line endings (LF)

### Linting

```bash
# Run linter
make lint

# This runs golangci-lint with our config
golangci-lint run --config .golangci.yml
```

**Enabled Linters:**
- `errcheck` - Check error handling
- `gosimple` - Simplify code
- `govet` - Suspicious constructs
- `ineffassign` - Ineffectual assignments
- `staticcheck` - Static analysis
- `unused` - Unused code
- `gocyclo` - Cyclomatic complexity
- `gofmt` - Formatting
- `gosec` - Security issues

### Naming Conventions

**✅ DO:**
```go
// Variables: camelCase
var userID string
var isActive bool

// Constants: PascalCase or ALL_CAPS
const MaxRetries = 3
const DEFAULT_TIMEOUT = 30 * time.Second

// Functions/Methods: PascalCase (exported) or camelCase (private)
func GetUserByID(id string) (*User, error)
func validateEmail(email string) bool

// Interfaces: PascalCase with -er suffix
type UserRepository interface
type PasswordHasher interface

// Structs: PascalCase
type User struct
type AuthService struct
```

**❌ DON'T:**
```go
// Avoid stuttering
// BAD:
type UserUser struct
func UserGetUser() {}

// GOOD:
type User struct
func GetUser() {}

// Avoid generic names
// BAD:
var data []byte
var temp string

// GOOD:
var responseBody []byte
var tempFileName string
```

### Error Handling

**✅ DO:**
```go
// Always check errors
user, err := repo.GetUserByID(ctx, id)
if err != nil {
    return nil, fmt.Errorf("failed to get user: %w", err)
}

// Use custom error types
if errors.Is(err, domain.ErrUserNotFound) {
    return apperror.NotFound("user not found", err)
}

// Wrap errors with context
if err := service.DeleteUser(ctx, id); err != nil {
    return fmt.Errorf("delete user %s: %w", id, err)
}
```

**❌ DON'T:**
```go
// Don't ignore errors
user, _ := repo.GetUserByID(ctx, id)

// Don't use panic for business logic
if user == nil {
    panic("user not found")
}

// Don't lose error context
if err != nil {
    return errors.New("error occurred")
}
```

### Context Usage

**✅ DO:**
```go
// Always pass context as first parameter
func GetUser(ctx context.Context, id string) (*User, error)

// Use context for cancellation
select {
case <-ctx.Done():
    return ctx.Err()
case result := <-ch:
    return result
}

// Add values to context
ctx = context.WithValue(ctx, "trace_id", traceID)
```

**❌ DON'T:**
```go
// Don't store context in structs
type Service struct {
    ctx context.Context  // BAD
}

// Don't use nil context
GetUser(nil, id)  // BAD - use context.Background() or context.TODO()
```

---

## Testing Requirements

### Test Coverage

**Minimum Coverage Requirements:**
- Overall: **80%**
- Critical paths (auth, payments): **95%**
- New code: **90%**

```bash
# Generate coverage report
make coverage

# View coverage in browser
make coverage-html
```

### Test Types

**1. Unit Tests**
```go
// Test filename: *_test.go
// Location: Same package as code

func TestUserService_GetUser(t *testing.T) {
    // Arrange
    mockRepo := mocks.NewMockUserRepository(t)
    service := NewUserService(mockRepo)
    
    // Act
    user, err := service.GetUser(ctx, userID)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, user)
}
```

**2. Integration Tests**
```go
// Location: tests/integration/

func TestUserAPI_Register(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // Create test server
    server := setupTestServer(t, db)
    defer server.Close()
    
    // Test API
    resp := testRegister(t, server.URL, validUser)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
```

**3. Table-Driven Tests**
```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        want    bool
    }{
        {name: "valid email", email: "user@example.com", want: true},
        {name: "missing @", email: "userexample.com", want: false},
        {name: "missing domain", email: "user@", want: false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ValidateEmail(tt.email)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Test Best Practices

**✅ DO:**
- Write tests before implementation (TDD)
- Use table-driven tests for multiple scenarios
- Test error cases and edge cases
- Use mocks for external dependencies
- Clean up resources in tests (defer cleanup)
- Use descriptive test names
- Test one thing per test
- Use `t.Parallel()` for independent tests

**❌ DON'T:**
- Skip writing tests
- Test implementation details
- Have test dependencies on test order
- Use sleep for synchronization (use channels/WaitGroups)
- Commit failing tests
- Test private functions (test public API)

---

## Commit Guidelines

### Conventional Commits

We use [Conventional Commits](https://www.conventionalcommits.org/) specification:

**Format:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**

| Type | Description | Example |
|------|-------------|---------|
| `feat` | New feature | `feat(auth): add 2FA support` |
| `fix` | Bug fix | `fix(api): handle nil pointer in GetUser` |
| `docs` | Documentation | `docs(readme): update installation steps` |
| `style` | Formatting | `style: run gofumpt on all files` |
| `refactor` | Code restructure | `refactor(service): extract validation logic` |
| `test` | Add/update tests | `test(user): add integration tests` |
| `chore` | Maintenance | `chore(deps): update go modules` |
| `perf` | Performance | `perf(db): add index on email column` |
| `ci` | CI/CD changes | `ci: add security scanning to pipeline` |
| `build` | Build system | `build: update Dockerfile` |
| `revert` | Revert commit | `revert: revert "feat: add feature X"` |

**Scopes:**
- `auth` - Authentication
- `user` - User service
- `api` - API layer
- `db` - Database
- `cache` - Redis cache
- `grpc` - gRPC service
- `middleware` - Middleware
- `config` - Configuration
- `docs` - Documentation

**Examples:**

```bash
# Feature
git commit -m "feat(auth): add JWT refresh token rotation"

# Bug fix
git commit -m "fix(api): prevent SQL injection in GetUserByEmail"

# Documentation
git commit -m "docs(security): document password hashing algorithm"

# Multiple lines
git commit -m "feat(user): add soft delete functionality

- Add deleted_at column to users table
- Update repository to filter deleted users
- Add admin endpoint to restore users

Closes #123"
```

### Commit Best Practices

**✅ DO:**
- Write clear, descriptive commit messages
- Keep commits atomic (one logical change)
- Reference issue numbers (`Fixes #123`, `Closes #456`)
- Use imperative mood ("add" not "added")
- Explain *why* in commit body

**❌ DON'T:**
- Write vague messages (`fix bug`, `update code`)
- Mix multiple changes in one commit
- Commit commented-out code
- Commit sensitive data (.env files)
- Commit generated files (unless necessary)

---

## Pull Request Process

### PR Template

When creating a PR, fill in the template:

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-reviewed code
- [ ] Commented complex code
- [ ] Updated documentation
- [ ] No new warnings
- [ ] Tests pass locally
- [ ] Coverage maintained/improved

## Related Issues
Fixes #123
Closes #456

## Screenshots (if applicable)
```

### PR Review Process

**1. Automated Checks**
- ✅ All tests pass
- ✅ Linter passes
- ✅ Security scan passes
- ✅ Coverage maintained

**2. Code Review**
- 2 approvals required
- Address all comments
- Request re-review after changes

**3. Merge**
- Squash and merge (default)
- Delete branch after merge
- Link to deployment

---

## Code Review Checklist

### For Authors

**Before Requesting Review:**
- [ ] Self-review your code
- [ ] Run all tests locally
- [ ] Run linter and fix issues
- [ ] Check test coverage
- [ ] Update documentation
- [ ] Add/update tests
- [ ] Commit message follows guidelines
- [ ] PR description is clear

### For Reviewers

**Code Quality:**
- [ ] Code is readable and maintainable
- [ ] No unnecessary complexity
- [ ] Follows project conventions
- [ ] No code duplication
- [ ] Error handling is appropriate
- [ ] Logging is appropriate

**Security:**
- [ ] No hardcoded secrets
- [ ] Input validation present
- [ ] Authentication/authorization checked
- [ ] No SQL injection vulnerabilities
- [ ] Sensitive data not logged

**Testing:**
- [ ] Tests cover new code
- [ ] Tests cover edge cases
- [ ] Tests are readable
- [ ] No flaky tests
- [ ] Mocks used appropriately

**Documentation:**
- [ ] Public APIs documented
- [ ] Complex logic explained
- [ ] README updated if needed
- [ ] Migration guide (if breaking change)

---

## Project Conventions

### DO's ✅

**Architecture:**
- ✅ Follow Clean Architecture layers
- ✅ Keep domain logic independent
- ✅ Use dependency injection
- ✅ Return errors, don't panic

**Code Organization:**
- ✅ One struct/interface per file
- ✅ Group related files in packages
- ✅ Keep functions small (<50 lines)
- ✅ Use interfaces for dependencies

**Database:**
- ✅ Use sqlc for type-safe queries
- ✅ Use migrations for schema changes
- ✅ Use transactions for multi-step operations
- ✅ Add indexes for query performance

**API:**
- ✅ Version APIs (`/api/v1`)
- ✅ Use proper HTTP status codes
- ✅ Validate all inputs
- ✅ Return consistent error format

**Configuration:**
- ✅ Use environment variables
- ✅ Provide sensible defaults
- ✅ Validate configuration on startup
- ✅ Document all config options

**Logging:**
- ✅ Use structured logging
- ✅ Include trace IDs
- ✅ Log at appropriate levels
- ✅ Redact sensitive data

### DON'Ts ❌

**Code:**
- ❌ Don't use global variables
- ❌ Don't use `panic()` for errors
- ❌ Don't ignore errors
- ❌ Don't use `interface{}`/`any` unless necessary

**Database:**
- ❌ Don't use `SELECT *`
- ❌ Don't write raw SQL (use sqlc)
- ❌ Don't modify migrations after merging
- ❌ Don't commit with dirty migrations

**API:**
- ❌ Don't expose internal errors to users
- ❌ Don't return stack traces
- ❌ Don't log passwords/tokens
- ❌ Don't trust user input

**Security:**
- ❌ Don't hardcode secrets
- ❌ Don't commit `.env` files
- ❌ Don't disable TLS in production
- ❌ Don't use weak passwords in tests

**Testing:**
- ❌ Don't skip writing tests
- ❌ Don't test private methods
- ❌ Don't use sleep in tests
- ❌ Don't commit failing tests

---

## Getting Help

### Resources

- 📖 [Documentation](./README.md)
- 🏗️ [Architecture Guide](../ARCHITECTURE.md)
- 🔐 [Security Guide](./SECURITY.md)
- 🧪 [Testing Guide](../docs/testing.md)
- 📡 [API Documentation](./API_DOCUMENTATION.md)

### Communication

- **GitHub Issues**: Bug reports, feature requests
- **GitHub Discussions**: Questions, ideas
- **Pull Requests**: Code contributions
- **Email**: dev@pandora-exchange.com

### Issue Templates

**Bug Report:**
```markdown
**Describe the bug**
A clear description

**To Reproduce**
Steps to reproduce

**Expected behavior**
What should happen

**Actual behavior**
What actually happens

**Environment**
- OS: [e.g., Ubuntu 22.04]
- Go version: [e.g., 1.21.1]
- Version: [e.g., v1.0.0]
```

**Feature Request:**
```markdown
**Problem Statement**
What problem does this solve?

**Proposed Solution**
How would you solve it?

**Alternatives**
Other approaches considered

**Additional Context**
Any other information
```

---

## Code of Conduct

### Our Standards

**Positive Behavior:**
- ✅ Be respectful and inclusive
- ✅ Welcome newcomers
- ✅ Accept constructive criticism
- ✅ Focus on what's best for the community
- ✅ Show empathy

**Unacceptable Behavior:**
- ❌ Harassment or discrimination
- ❌ Trolling or insulting comments
- ❌ Personal or political attacks
- ❌ Publishing others' private information
- ❌ Other unprofessional conduct

### Enforcement

Violations may result in:
1. Warning
2. Temporary ban
3. Permanent ban

Report violations to: conduct@pandora-exchange.com

---

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

---

## Recognition

Contributors will be recognized in:
- 📜 CONTRIBUTORS.md file
- 🎉 Release notes
- 🏆 GitHub contributors page

Thank you for contributing to Pandora Exchange! 🚀

---

**Last Updated:** November 12, 2025  
**Contributing Guide Version:** 1.0
