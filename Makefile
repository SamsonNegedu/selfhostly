.PHONY: dev dev-backend dev-frontend prod down clean install-air run-local help

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

run-local: ## Run backend locally with Air (no Docker)
	air

run-local-no-air: ## Run backend locally without Air
	go run cmd/server/main.go

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
