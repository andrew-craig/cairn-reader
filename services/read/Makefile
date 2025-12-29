# Cairn Backend - Root Makefile

.PHONY: help build up down logs clean migrate-up migrate-down migrate-create

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build all services
	@echo "Building content-service..."
	cd services/content-service && go build -o bin/content-service ./cmd/server
	@echo "Building rss-fetcher-service..."
	cd services/rss-fetcher-service && go build -o bin/rss-fetcher-service ./cmd/server
	@echo "Build complete!"

up: ## Start all services with docker-compose
	docker-compose up -d

down: ## Stop all services
	docker-compose down

logs: ## Show logs from all services
	docker-compose logs -f

clean: ## Clean build artifacts
	rm -rf services/content-service/bin
	rm -rf services/rss-fetcher-service/bin
	docker-compose down -v

# Content Service migrations
migrate-up-content: ## Apply all pending migrations for content-service
	@echo "Running migrations for content-service..."
	migrate -path services/content-service/migrations -database "postgresql://cairn_content:cairn_content_pass@localhost:5433/cairn_content?sslmode=disable" up

migrate-down-content: ## Rollback last migration for content-service
	@echo "Rolling back last migration for content-service..."
	migrate -path services/content-service/migrations -database "postgresql://cairn_content:cairn_content_pass@localhost:5433/cairn_content?sslmode=disable" down 1

migrate-create-content: ## Create new migration for content-service (usage: make migrate-create-content name=migration_name)
	@if [ -z "$(name)" ]; then \
		echo "Error: name parameter is required. Usage: make migrate-create-content name=migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir services/content-service/migrations -seq $(name)

migrate-status-content: ## Show migration status for content-service
	migrate -path services/content-service/migrations -database "postgresql://cairn_content:cairn_content_pass@localhost:5433/cairn_content?sslmode=disable" version

# RSS Fetcher Service migrations
migrate-up-rss: ## Apply all pending migrations for rss-fetcher-service
	@echo "Running migrations for rss-fetcher-service..."
	migrate -path services/rss-fetcher-service/migrations -database "postgresql://cairn_rss:cairn_rss_pass@localhost:5434/cairn_rss?sslmode=disable" up

migrate-down-rss: ## Rollback last migration for rss-fetcher-service
	@echo "Rolling back last migration for rss-fetcher-service..."
	migrate -path services/rss-fetcher-service/migrations -database "postgresql://cairn_rss:cairn_rss_pass@localhost:5434/cairn_rss?sslmode=disable" down 1

migrate-create-rss: ## Create new migration for rss-fetcher-service (usage: make migrate-create-rss name=migration_name)
	@if [ -z "$(name)" ]; then \
		echo "Error: name parameter is required. Usage: make migrate-create-rss name=migration_name"; \
		exit 1; \
	fi
	migrate create -ext sql -dir services/rss-fetcher-service/migrations -seq $(name)

migrate-status-rss: ## Show migration status for rss-fetcher-service
	migrate -path services/rss-fetcher-service/migrations -database "postgresql://cairn_rss:cairn_rss_pass@localhost:5434/cairn_rss?sslmode=disable" version

# Run all migrations
migrate-up: migrate-up-content migrate-up-rss ## Apply all pending migrations for all services

# Rollback all migrations
migrate-down: migrate-down-content migrate-down-rss ## Rollback last migration for all services
