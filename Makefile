# Pandora Exchange - User Service Makefile
# Architecture-compliant build automation

.PHONY: help dev-up dev-down migrate migrate-down migrate-force migrate-version migrate-create sqlc test test-coverage lint build run docker-build clean proto install-tools

# Variables
SERVICE_NAME := user-service
DOCKER_COMPOSE := docker-compose -f deployments/docker/docker-compose.yml
MIGRATIONS_DIR := migrations
POSTGRES_URL := postgresql://pandora:pandora_dev_secret@localhost:5432/pandora_dev?sslmode=disable
GO_TEST_FLAGS := -v -race -timeout=30s
COVERAGE_OUT := coverage.out

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
	@echo "  make test-coverage   - Run tests with coverage report"
	@echo "  make lint            - Run golangci-lint"
	@echo "  make build           - Build service binary"
	@echo "  make run             - Run service locally"
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

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test $(GO_TEST_FLAGS) -coverprofile=$(COVERAGE_OUT) ./...
	@echo "Generating HTML coverage report..."
	go tool cover -html=$(COVERAGE_OUT) -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"
	@go tool cover -func=$(COVERAGE_OUT) | grep total

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

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "✅ All checks passed"

# Default target
.DEFAULT_GOAL := help
