.PHONY: help build run test clean docker-build docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build both services
	@echo "Building explore_fetcher..."
	@go build -o bin/explore_fetcher ./fetcher/cmd/explore_fetcher
	@echo "Building explore_recommender..."
	@go build -o bin/explore_recommender ./recommender/cmd/explore_recommender
	@echo "Building cleanup utility..."
	@go build -o bin/explore_cleanup ./recommender/cmd/explore_cleanup
	@echo "Build complete!"

run-fetcher: ## Run the fetcher service locally
	@go run ./fetcher/cmd/explore_fetcher/main.go

run-recommender: ## Run the recommender service locally
	@go run ./recommender/cmd/explore_recommender/main.go

run-cleanup: ## Run the article cleanup utility
	@go run ./recommender/cmd/explore_cleanup/main.go

test: ## Run unit tests
	@go test -v ./...

test-integration: ## Run integration tests (requires running database)
	@echo "Setting up test database..."
	@cd recommender && ./scripts/setup_test_db.sh
	@echo "Running integration tests..."
	@go test -v ./recommender -run Test

test-all: test test-integration ## Run all tests (unit + integration)

clean: ## Clean build artifacts
	@rm -rf bin/
	@echo "Clean complete!"

docker-build: ## Build Docker images
	@docker compose build

docker-up: ## Start all services with Docker Compose
	@docker compose up -d

docker-down: ## Stop all services
	@docker compose down

docker-logs: ## Show logs from all services
	@docker compose logs -f

docker-restart: docker-down docker-up ## Restart all services

tidy: ## Tidy go modules
	@go mod tidy

fmt: ## Format Go code
	@go fmt ./...

vet: ## Run go vet
	@go vet ./...

lint: fmt vet ## Run formatters and linters
