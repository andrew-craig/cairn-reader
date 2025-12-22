.PHONY: help build run test clean deps lint fmt vet migrate-up migrate-down docker-build docker-run

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the user service binary"
	@echo "  run          - Run the user service"
	@echo "  test         - Run all tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean        - Remove build artifacts"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  lint         - Run golangci-lint"
	@echo "  fmt          - Format code with gofmt"
	@echo "  vet          - Run go vet"
	@echo "  migrate-up   - Run database migrations (up)"
	@echo "  migrate-down - Rollback database migrations"
	@echo "  migrate-version - Check current migration version"
	@echo "  db-create    - Create database and user"
	@echo "  db-drop      - Drop database"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run service in Docker"

# Build the service
build:
	@echo "Building user service..."
	go build -o bin/user-service cmd/user-service/main.go
	go build -o bin/migrate cmd/migrate/main.go

# Run the service
run:
	@echo "Running user service..."
	go run cmd/user-service/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

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
