.PHONY: help build run test test-unit test-integration test-integration-up test-integration-down test-security clean deps lint fmt vet migrate-up migrate-down docker-build docker-run

# Default target
help:
	@echo "Available targets:"
	@echo "  build                 - Build the user service binary"
	@echo "  run                   - Run the user service"
	@echo "  test                  - Run unit tests only"
	@echo "  test-unit             - Run unit tests only (alias for test)"
	@echo "  test-integration      - Run integration tests (requires Docker)"
	@echo "  test-integration-up   - Start integration test environment"
	@echo "  test-integration-down - Stop integration test environment"
	@echo "  test-all              - Run all tests (unit + integration)"
	@echo "  test-coverage         - Run unit tests with coverage report"
	@echo "  test-security         - Run security penetration tests (requires running service)"
	@echo "  clean                 - Remove build artifacts"
	@echo "  deps                  - Download and tidy dependencies"
	@echo "  lint                  - Run golangci-lint"
	@echo "  fmt                   - Format code with gofmt"
	@echo "  vet                   - Run go vet"
	@echo "  migrate-up            - Run database migrations (up)"
	@echo "  migrate-down          - Rollback database migrations"
	@echo "  migrate-version       - Check current migration version"
	@echo "  db-create             - Create database and user"
	@echo "  db-drop               - Drop database"
	@echo "  docker-build          - Build Docker image"
	@echo "  docker-run            - Run service in Docker"

# Build the service
build:
	@echo "Building user service..."
	go build -o bin/user-service cmd/user-service/main.go
	go build -o bin/migrate cmd/migrate/main.go

# Run the service
run:
	@echo "Running user service..."
	go run cmd/user-service/main.go

# Run unit tests only (excludes integration tests)
test:
	@echo "Running unit tests..."
	go test -v ./internal/... ./pkg/...

# Alias for test
test-unit: test

# Run integration tests (requires Docker)
test-integration:
	@echo "Running integration tests..."
	@echo "Starting test environment..."
	cd test/integration && docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Running integration tests..."
	cd test/integration && go test -v -timeout 5m ./...
	@echo "Stopping test environment..."
	cd test/integration && docker-compose -f docker-compose.test.yml down

# Start integration test environment
test-integration-up:
	@echo "Starting integration test environment..."
	cd test/integration && docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Test environment ready!"
	@echo "Database: localhost:5433 (cairn_users_test)"
	@echo "Vault: http://localhost:8201 (token: test-root-token)"

# Stop integration test environment
test-integration-down:
	@echo "Stopping integration test environment..."
	cd test/integration && docker-compose -f docker-compose.test.yml down -v
	@echo "Test environment stopped and cleaned up"

# Run all tests (unit + integration)
test-all:
	@echo "Running all tests (unit + integration)..."
	@$(MAKE) test-unit
	@$(MAKE) test-integration

# Run tests with coverage (unit tests only)
test-coverage:
	@echo "Running unit tests with coverage..."
	go test -v -coverprofile=coverage.out ./internal/... ./pkg/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run security penetration tests
test-security:
	@echo "Running security penetration tests..."
	@echo "Note: This requires the User Service to be running at USER_SERVICE_URL"
	@echo "Default: http://localhost:8080"
	@echo ""
	cd test/security && ./run_all_tests.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Database migrations
migrate-up:
	@echo "Running database migrations..."
	go run cmd/migrate/main.go -command=up

migrate-down:
	@echo "Rolling back database migrations..."
	go run cmd/migrate/main.go -command=down

migrate-version:
	@echo "Checking migration version..."
	go run cmd/migrate/main.go -command=version

# Create database (requires psql)
db-create:
	@echo "Creating database..."
	psql -U postgres -c "CREATE DATABASE cairn_users;"
	psql -U postgres -c "CREATE USER cairn_user WITH PASSWORD 'your_secure_password_here';"
	psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE cairn_users TO cairn_user;"

# Drop database (requires psql)
db-drop:
	@echo "Dropping database..."
	psql -U postgres -c "DROP DATABASE IF EXISTS cairn_users;"

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t cairn-user-service:latest .

docker-run:
	@echo "Running service in Docker..."
	docker run -p 8080:8080 --env-file .env cairn-user-service:latest

# Development with live reload (requires air)
dev:
	@echo "Running with live reload..."
	air
