.PHONY: help start dev dev-backend dev-frontend dev-all build build-backend build-frontend test test-backend test-frontend lint lint-backend lint-frontend migrate-up migrate-down migrate-create docker-up docker-down clean install stop

help:
	@echo "RSS Summarizer - Unified Commands"
	@echo ""
	@echo "Quick Start:"
	@echo "  make start            - Start everything (Docker + migrations + dev servers)"
	@echo "  make stop             - Stop all services and dev servers"
	@echo ""
	@echo "Development:"
	@echo "  make dev              - Run both backend and frontend (parallel)"
	@echo "  make dev-backend      - Run backend development server"
	@echo "  make dev-frontend     - Run frontend development server"
	@echo ""
	@echo "Build:"
	@echo "  make build            - Build both backend and frontend"
	@echo "  make build-backend    - Build backend binary"
	@echo "  make build-frontend   - Build frontend for production"
	@echo ""
	@echo "Test:"
	@echo "  make test             - Run all tests"
	@echo "  make test-backend     - Run backend tests"
	@echo "  make test-frontend    - Run frontend type checking"
	@echo ""
	@echo "Lint:"
	@echo "  make lint             - Run all linters"
	@echo "  make lint-backend     - Run backend linters"
	@echo "  make lint-frontend    - Run frontend type checks"
	@echo ""
	@echo "Database:"
	@echo "  make migrate-up       - Run database migrations"
	@echo "  make migrate-down     - Rollback last migration"
	@echo "  make migrate-create   - Create new migration"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-up        - Start all services (db, temporal)"
	@echo "  make docker-down      - Stop all services"
	@echo ""
	@echo "Other:"
	@echo "  make install          - Install dependencies for both"
	@echo "  make clean            - Clean build artifacts"

# Quick Start
start:
	@echo "Starting all services..."
	@echo "1. Starting Docker services (PostgreSQL + Temporal)..."
	@docker compose up -d postgres temporal temporal-ui
	@echo "2. Waiting for services to be ready..."
	@sleep 5
	@echo "3. Running database migrations..."
	@cd backend && go run cmd/migrate/main.go up || true
	@echo "4. Starting backend and frontend..."
	@echo ""
	@echo "Services ready:"
	@echo "  - Frontend: http://localhost:5173"
	@echo "  - Backend:  http://localhost:8080"
	@echo "  - Temporal: http://localhost:8233"
	@echo ""
	@trap 'kill 0' INT; \
	(cd backend && go run ./cmd/api/main.go) & \
	(cd frontend && npm run dev) & \
	wait

stop:
	@echo "Stopping all services..."
	@docker compose down
	@pkill -f "go run ./cmd/api/main.go" || true
	@pkill -f "vite dev" || true
	@echo "All services stopped."

# Development
dev: dev-all

dev-all:
	@echo "Starting backend and frontend..."
	@echo "Note: Make sure PostgreSQL and Temporal are running (make docker-up)"
	@echo ""
	@trap 'kill 0' INT; \
	(cd backend && go run ./cmd/api/main.go) & \
	(cd frontend && npm run dev) & \
	wait

dev-backend:
	cd backend && go run ./cmd/api/main.go

dev-frontend:
	cd frontend && npm run dev

# Build
build: build-backend build-frontend

build-backend:
	cd backend && go build -o bin/api ./cmd/api

build-frontend:
	cd frontend && npm run build

# Test
test: test-backend test-frontend

test-backend:
	cd backend && go test -v -race -coverprofile=coverage.out ./...

test-frontend:
	cd frontend && npm run check

# Lint
lint: lint-backend lint-frontend

lint-backend:
	cd backend && go vet ./...

lint-frontend:
	cd frontend && npm run check

# Database
migrate-up:
	cd backend && go run cmd/migrate/main.go up

migrate-down:
	cd backend && go run cmd/migrate/main.go down

migrate-create:
	@read -p "Migration name: " name; \
	cd backend && go run cmd/migrate/main.go create $$name

# Docker
docker-up:
	docker compose up -d

docker-down:
	docker compose down

# Install
install: install-backend install-frontend

install-backend:
	cd backend && go mod download

install-frontend:
	cd frontend && npm install

# Clean
clean:
	cd backend && rm -rf bin/ coverage.out
	cd frontend && rm -rf build/ .svelte-kit/
