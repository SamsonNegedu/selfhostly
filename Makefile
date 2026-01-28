# =============================================================================
# Selfhostly Makefile
# =============================================================================
#
# Run with different .env files:
#   make run-local                          # Uses .env (default)
#   make run-local ENV_FILE=.env.primary    # Uses .env.primary
#   make run-local ENV_FILE=.env.secondary  # Uses .env.secondary
#
# Multi-Node Local Development:
#   Terminal 1: make run-local ENV_FILE=.env.primary
#   Terminal 2: make run-local ENV_FILE=.env.secondary
#
# =============================================================================

.PHONY: dev dev-backend dev-frontend prod down clean install-air run-local test test-verbose test-coverage help

# Development commands
dev: ## Start all services with live reload
	docker-compose -f docker-compose.dev.yml up

dev-backend: ## Start only backend with live reload
	docker-compose -f docker-compose.dev.yml up backend

dev-frontend: ## Start only frontend
	docker-compose -f docker-compose.dev.yml up frontend

dev-build: ## Rebuild dev containers
	docker-compose -f docker-compose.dev.yml build

# Production commands
prod: ## Start production services
	docker-compose -f docker-compose.prod.yml up -d

prod-build: ## Build and start production services
	docker-compose -f docker-compose.prod.yml up -d --build

# Control commands
down: ## Stop all running containers
	docker-compose -f docker-compose.dev.yml down
	docker-compose -f docker-compose.prod.yml down

clean: ## Clean build artifacts and containers
	docker-compose -f docker-compose.dev.yml down -v
	docker-compose -f docker-compose.prod.yml down -v
	rm -rf tmp/
	rm -f build-errors.log

# Local development (no Docker)
install-air: ## Install Air for local development
	go install github.com/air-verse/air@latest

run-local: ## Run backend with Air (usage: make run-local [ENV_FILE=.env.custom])
	@if [ -n "$(ENV_FILE)" ]; then \
		if [ ! -f "$(ENV_FILE)" ]; then \
			echo "ERROR: $(ENV_FILE) not found."; \
			exit 1; \
		fi; \
		echo "Starting with $(ENV_FILE)"; \
		ENV_FILE=$(ENV_FILE) air; \
	else \
		echo "Starting with .env"; \
		air; \
	fi

run-local-no-air: ## Run backend without Air (usage: make run-local-no-air [ENV_FILE=.env.custom])
	@if [ -n "$(ENV_FILE)" ]; then \
		if [ ! -f "$(ENV_FILE)" ]; then \
			echo "ERROR: $(ENV_FILE) not found."; \
			exit 1; \
		fi; \
		echo "Starting with $(ENV_FILE)"; \
		ENV_FILE=$(ENV_FILE) go run cmd/server/main.go; \
	else \
		echo "Starting with .env"; \
		go run cmd/server/main.go; \
	fi

# Testing commands
test: ## Run all tests
	go test ./...

test-verbose: ## Run all tests with verbose output
	go test -v ./...

test-coverage: ## Run all tests with coverage report
	go test -cover ./...

# Utility commands
logs: ## Show logs from all services
	docker-compose -f docker-compose.dev.yml logs -f

logs-backend: ## Show backend logs only
	docker-compose -f docker-compose.dev.yml logs -f backend

restart-backend: ## Restart backend service
	docker-compose -f docker-compose.dev.yml restart backend

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
