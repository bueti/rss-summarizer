.PHONY: help test test-setup test-coverage test-watch build run migrate-up migrate-down docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the API binary
	go build -o bin/api cmd/api/main.go

run: ## Run the API server
	go run cmd/api/main.go

migrate-up: ## Run database migrations
	go run cmd/migrate/main.go up

migrate-down: ## Rollback last migration
	go run cmd/migrate/main.go down

docker-up: ## Start all services with Docker Compose
	docker-compose up -d

docker-down: ## Stop all Docker services
	docker-compose down

test-setup: ## Setup test database
	./scripts/setup-test-db.sh

test: ## Run all integration tests
	go test ./internal/api/handlers/... -v

test-coverage: ## Run tests with coverage report
	go test ./internal/api/handlers/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-watch: ## Run tests in watch mode (requires entr)
	find . -name '*.go' | entr -c go test ./internal/api/handlers/... -v

clean: ## Clean build artifacts
	rm -f bin/api coverage.out coverage.html
