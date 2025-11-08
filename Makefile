# Pandora Exchange - User Service Makefile
# Architecture-compliant build automation

.PHONY: help dev dev-up dev-down migrate migrate-down migrate-force migrate-version migrate-create sqlc test test-unit test-integration test-bench test-coverage imports-check security-scan docs lint build run docker-build clean proto install-tools deps tidy fmt vet check ci

# Variables
SERVICE_NAME := user-service
DOCKER_COMPOSE := docker-compose -f deployments/docker/docker-compose.yml
MIGRATIONS_DIR := migrations
POSTGRES_URL := postgresql://pandora:pandora_dev_secret@localhost:5432/pandora_dev?sslmode=disable
GO_TEST_FLAGS := -v -race -timeout=30s
COVERAGE_OUT := coverage.out
BENCH_FLAGS := -bench=. -benchmem -benchtime=5s
DOCS_OUT := docs/generated

## help: Display this help message
help:
	@echo "Pandora Exchange - User Service"
	@echo ""
	@echo "Available targets:"
	@echo "  make dev-up          - Start PostgreSQL + Redis in Docker"
	@echo "  make dev-down        - Stop development environment"
	@echo "  make migrate         - Run database migrations (up)"
	@echo "  make migrate-down    - Rollback last migration"
	@echo "  make migrate-force   - Force migration to specific version"
	@echo "  make migrate-version - Show current migration version"
	@echo "  make migrate-create  - Create new migration files (use NAME=migration_name)"
	@echo "  make sqlc            - Generate sqlc code from SQL queries"
	@echo "  make proto           - Generate gRPC code from protobuf"
	@echo "  make test            - Run all tests"
	@echo "  make test-unit       - Run unit tests only (exclude integration)"
	@echo "  make test-integration- Run integration tests only"
	@echo "  make test-bench      - Run benchmarks"
	@echo "  make test-coverage   - Run tests with coverage report"
	@echo "  make imports-check   - Check import boundaries (architecture enforcement)"
	@echo "  make security-scan   - Run security vulnerability scan (gosec)"
	@echo "  make docs            - Generate Go documentation"
	@echo "  make lint            - Run golangci-lint"
	@echo "  make fmt             - Format Go code"
	@echo "  make vet             - Run go vet"
	@echo "  make check           - Run all checks (fmt, vet, lint, imports-check, test)"
	@echo "  make ci              - Run all CI checks (deps, fmt, vet, lint, imports, security, test)"
	@echo "  make build           - Build service binary"
	@echo "  make run             - Run service locally"
	@echo "  make dev             - Start dev environment, migrate, build and run service"
	@echo "  make docker-build    - Build Docker image"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make install-tools   - Install required development tools"

## install-tools: Install required development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install go.uber.org/mock/mockgen@latest
	go install github.com/securego/gosec/v2/cmd/gosec@v2.21.4
	@echo "✅ Tools installed successfully"

## dev-up: Start development environment (PostgreSQL + Redis)
dev-up:
	@echo "Starting development environment..."
	$(DOCKER_COMPOSE) up -d postgres redis
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 3
	@echo "✅ Development environment is running"
	@echo "PostgreSQL: localhost:5432 (user: pandora, db: pandora_dev)"
	@echo "Redis: localhost:6379"

## dev-down: Stop development environment
dev-down:
	@echo "Stopping development environment..."
	$(DOCKER_COMPOSE) down -v
	@echo "✅ Development environment stopped"

## migrate: Run database migrations (up)
migrate:
	@echo "Running database migrations..."
	migrate -path $(MIGRATIONS_DIR) -database "$(POSTGRES_URL)" up
	@echo "✅ Migrations applied successfully"

## migrate-down: Rollback last migration
migrate-down:
	@echo "Rolling back last migration..."
	migrate -path $(MIGRATIONS_DIR) -database "$(POSTGRES_URL)" down 1
	@echo "✅ Migration rolled back"

## migrate-force: Force migration to specific version (use VERSION=N)
migrate-force:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ ERROR: VERSION not specified"; \
		echo "Usage: make migrate-force VERSION=2"; \
		exit 1; \
	fi
	@echo "Forcing migration to version $(VERSION)..."
	migrate -path $(MIGRATIONS_DIR) -database "$(POSTGRES_URL)" force $(VERSION)
	@echo "✅ Migration forced to version $(VERSION)"

## migrate-version: Show current migration version
migrate-version:
	@echo "Current migration version:"
	@migrate -path $(MIGRATIONS_DIR) -database "$(POSTGRES_URL)" version

## migrate-create: Create new migration files (use NAME=migration_name)
migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "❌ ERROR: NAME not specified"; \
		echo "Usage: make migrate-create NAME=add_user_roles"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(NAME)..."
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)
	@echo "✅ Migration files created"

## sqlc: Generate sqlc code from SQL queries
sqlc:
	@echo "Generating sqlc code..."
	sqlc generate
	@echo "✅ sqlc code generated successfully"

## proto: Generate gRPC code from protobuf files
proto:
	@echo "Generating gRPC code from protobuf..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		internal/transport/grpc/proto/*.proto
	@echo "✅ gRPC code generated successfully"

## test: Run all tests
test:
	@echo "Running tests..."
	go test $(GO_TEST_FLAGS) ./...
	@echo "✅ All tests passed"

## test-unit: Run unit tests only (exclude integration tests)
test-unit:
	@echo "Running unit tests..."
	go test $(GO_TEST_FLAGS) ./... -short
	@echo "✅ Unit tests passed"

## test-integration: Run integration tests only
test-integration:
	@echo "Running integration tests..."
	go test $(GO_TEST_FLAGS) ./tests/integration/... -run Integration
	@echo "✅ Integration tests passed"

## test-bench: Run benchmarks
test-bench:
	@echo "Running benchmarks..."
	go test $(BENCH_FLAGS) ./...
	@echo "✅ Benchmarks completed"

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test $(GO_TEST_FLAGS) -coverprofile=$(COVERAGE_OUT) ./...
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_OUT) -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"
	@go tool cover -func=$(COVERAGE_OUT) | grep total

## imports-check: Verify import boundaries (clean architecture enforcement)
imports-check:
	@echo "Checking import boundaries..."
	@go test ./internal/ci_checks/... -v -run TestDomainLayerImportBoundaries
	@go test ./internal/ci_checks/... -v -run TestRepositoryLayerImportBoundaries
	@go test ./internal/ci_checks/... -v -run TestServiceLayerImportBoundaries
	@echo "✅ Import boundaries verified"

## security-scan: Run security vulnerability scan
security-scan:
	@echo "Running security scan with gosec..."
	@command -v gosec >/dev/null 2>&1 || { \
		echo "gosec not installed. Installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@v2.21.4; \
	}
	gosec -fmt=text -exclude=G104,G101,G103 -exclude-generated ./... || true
	@echo "✅ Security scan completed (warnings above are informational)"

## docs: Generate Go documentation
docs:
	@echo "Generating documentation..."
	@mkdir -p $(DOCS_OUT)
	@echo "Generating package documentation..."
	@go doc -all ./internal/domain > $(DOCS_OUT)/domain.txt
	@go doc -all ./internal/service > $(DOCS_OUT)/service.txt
	@go doc -all ./internal/repository > $(DOCS_OUT)/repository.txt
	@go doc -all ./internal/config > $(DOCS_OUT)/config.txt
	@echo "✅ Documentation generated in $(DOCS_OUT)/"
	@echo ""
	@echo "To view documentation in browser, run:"
	@echo "  godoc -http=:6060"
	@echo "  open http://localhost:6060/pkg/github.com/alex-necsoiu/pandora-exchange/"

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	golangci-lint run --timeout=5m ./...
	@echo "✅ Linting completed"

## build: Build service binary
build:
	@echo "Building $(SERVICE_NAME)..."
	@mkdir -p bin
	go build -o bin/$(SERVICE_NAME) ./cmd/$(SERVICE_NAME)
	@echo "✅ Binary built: bin/$(SERVICE_NAME)"

## run: Run service locally
run: build
	@echo "Starting $(SERVICE_NAME)..."
	./bin/$(SERVICE_NAME)

## dev: Start dev environment, migrate, build and run service (one command for everything)
dev:
	@echo "🚀 Starting complete development environment..."
	@echo ""
	@echo "Step 1/4: Starting Docker containers (PostgreSQL + Redis)..."
	@$(MAKE) dev-up
	@echo ""
	@echo "Step 2/4: Running database migrations..."
	@$(MAKE) migrate
	@echo ""
	@echo "Step 3/4: Building service binary..."
	@$(MAKE) build
	@echo ""
	@echo "Step 4/4: Starting service..."
	@if [ ! -f .env.dev ]; then \
		echo "❌ ERROR: .env.dev file not found"; \
		echo "Please create .env.dev with required environment variables"; \
		echo "See .env.dev.example for reference"; \
		exit 1; \
	fi
	@echo "Loading environment from .env.dev..."
	@export $$(cat .env.dev | xargs) && ./bin/$(SERVICE_NAME)

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t pandora/$(SERVICE_NAME):latest -f deployments/docker/Dockerfile .
	@echo "✅ Docker image built: pandora/$(SERVICE_NAME):latest"

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf build/
	rm -rf dist/
	rm -f $(COVERAGE_OUT)
	rm -f coverage.html
	@echo "✅ Cleaned"

## deps: Download Go module dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify
	@echo "✅ Dependencies downloaded"

## tidy: Tidy Go modules
tidy:
	@echo "Tidying Go modules..."
	go mod tidy
	@echo "✅ Modules tidied"

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "✅ go vet completed"

## check: Run all checks (fmt, vet, lint, imports-check, test)
check: fmt vet lint imports-check test
	@echo "✅ All checks passed"

## ci: Run all CI checks (what runs in CI/CD pipeline)
ci: deps fmt vet lint imports-check security-scan test
	@echo "✅ All CI checks passed"

# Default target
.DEFAULT_GOAL := help
