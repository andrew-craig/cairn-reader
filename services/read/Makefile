# Cairn Read Service - Makefile

.PHONY: help build build-all build-content build-content-worker build-fetcher build-fetcher-worker
.PHONY: test test-all test-content test-fetcher
.PHONY: up down logs clean
.PHONY: migrate-up migrate-down migrate-create migrate-status

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build targets
build-content: ## Build content service
	@echo "Building content service..."
	cd content && go build -o ../bin/content ./cmd/content

build-content-worker: ## Build content worker
	@echo "Building content worker..."
	cd content && go build -o ../bin/content-worker ./cmd/worker

build-fetcher: ## Build fetcher service
	@echo "Building fetcher service..."
	cd fetcher && go build -o ../bin/fetcher ./cmd/fetcher

build-fetcher-worker: ## Build fetcher worker
	@echo "Building fetcher worker..."
	cd fetcher && go build -o ../bin/fetcher-worker ./cmd/worker

build-all: build-content build-content-worker build-fetcher build-fetcher-worker ## Build all services
	@echo "All services built successfully!"

build: build-all ## Build all services (alias)

# Test targets
test-content: ## Run content service tests
	@echo "Running content service tests..."
	cd content && go test ./...

test-fetcher: ## Run fetcher service tests
	@echo "Running fetcher service tests..."
	cd fetcher && go test ./...

test-all: test-content test-fetcher ## Run all tests
	@echo "All tests complete!"

test: test-all ## Run all tests (alias)

# Docker targets
up: ## Start all services with docker-compose
	docker-compose up -d

down: ## Stop all services
	docker-compose down

logs: ## Show logs from all services
	docker-compose logs -f

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf content/bin/
	rm -rf fetcher/bin/
	docker-compose down -v

# Content Service migrations
migrate-up-content: ## Apply all pending migrations for content-service
	@echo "Running migrations for content-service..."
	migrate -path content/migrations -database "postgresql://cairn_content:cairn_content_pass@localhost:5433/cairn_content?sslmode=disable" up

migrate-down-content: ## Rollback last migration for content-service
	@echo "Rolling back last migration for content-service..."
	migrate -path content/migrations -database "postgresql://cairn_content:cairn_content_pass@localhost:5433/cairn_content?sslmode=disable" down 1

migrate-create-content: ## Create new migration for content-service (usage: make migrate-create-content name=migration_name)
	@if [ -z "$(name)" ]; then \
		echo "Error: name parameter is required. Usage: make migrate-create-content name=migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir content/migrations -seq $(name)

migrate-status-content: ## Show migration status for content-service
	migrate -path content/migrations -database "postgresql://cairn_content:cairn_content_pass@localhost:5433/cairn_content?sslmode=disable" version

# RSS Fetcher Service migrations
migrate-up-fetcher: ## Apply all pending migrations for fetcher-service
	@echo "Running migrations for fetcher-service..."
	migrate -path fetcher/migrations -database "postgresql://cairn_rss:cairn_rss_pass@localhost:5434/cairn_rss?sslmode=disable" up

migrate-down-fetcher: ## Rollback last migration for fetcher-service
	@echo "Rolling back last migration for fetcher-service..."
	migrate -path fetcher/migrations -database "postgresql://cairn_rss:cairn_rss_pass@localhost:5434/cairn_rss?sslmode=disable" down 1

migrate-create-fetcher: ## Create new migration for fetcher-service (usage: make migrate-create-fetcher name=migration_name)
	@if [ -z "$(name)" ]; then \
		echo "Error: name parameter is required. Usage: make migrate-create-fetcher name=migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir fetcher/migrations -seq $(name)

migrate-status-fetcher: ## Show migration status for fetcher-service
	migrate -path fetcher/migrations -database "postgresql://cairn_rss:cairn_rss_pass@localhost:5434/cairn_rss?sslmode=disable" version

# Run all migrations
migrate-up: migrate-up-content migrate-up-fetcher ## Apply all pending migrations for all services

# Rollback all migrations
migrate-down: migrate-down-content migrate-down-fetcher ## Rollback last migration for all services

# Code quality targets
fmt: ## Format code
	@echo "Formatting content service..."
	cd content && go fmt ./...
	@echo "Formatting fetcher service..."
	cd fetcher && go fmt ./...

vet: ## Run go vet
	@echo "Vetting content service..."
	cd content && go vet ./...
	@echo "Vetting fetcher service..."
	cd fetcher && go vet ./...

tidy: ## Tidy go modules
	go mod tidy

lint: fmt vet ## Run fmt and vet
