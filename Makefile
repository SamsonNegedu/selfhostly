# =============================================================================
# Selfhostly Makefile. Run `make help` to list the commands.
# =============================================================================
#
# Options that work on the commands below:
#   ENV_FILE=.env.primary   which env file the local backend or gateway reads (default .env)
#   NOAIR=1                 run with `go run` instead of Air hot reload
#   SERVICE=backend         limit `dev` and `logs` to one Docker service
#   ARGS=-v                 extra flags for `make test`
#   OLD_REF=<commit>        for `make e2e-update`: the version the install runs today (SKIP_BUILD=1, KEEP=1 also work)
#
# Two nodes on one machine: `make backend ENV_FILE=.env.primary` in one terminal,
# `make backend ENV_FILE=.env.secondary` in another.

DC       ?= docker compose
ENV_FILE ?= .env
SERVICE  ?=
ARGS     ?=

BACKEND_RUN = $(if $(NOAIR),go run ./cmd/server,air)
GATEWAY_RUN = $(if $(NOAIR),go run ./cmd/gateway,air -c .air-gateway.toml)

.DEFAULT_GOAL := help
.PHONY: help dev backend gateway frontend prod down clean logs test ctl docs e2e-update hooks

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# --- Run in Docker -----------------------------------------------------------

dev: ## Backend and frontend in Docker with live reload (SERVICE=backend for one)
	$(DC) -f docker-compose.dev.yml up --build $(SERVICE)

prod: ## Production services in the background
	$(DC) -f docker-compose.prod.yml up -d --build

down: ## Stop the dev and production containers
	$(DC) -f docker-compose.dev.yml down
	$(DC) -f docker-compose.prod.yml down

logs: ## Follow the dev logs (SERVICE=backend for one)
	$(DC) -f docker-compose.dev.yml logs -f $(SERVICE)

# --- Run on this machine, no Docker -----------------------------------------

backend: check-env check-air ## Backend with hot reload on :8080 (ENV_FILE=..., NOAIR=1)
	ENV_FILE=$(ENV_FILE) $(BACKEND_RUN)

gateway: check-env check-air ## API gateway with hot reload (ENV_FILE=..., NOAIR=1)
	ENV_FILE=$(ENV_FILE) $(GATEWAY_RUN)

frontend: ## Frontend dev server on :5173
	cd web && npm run dev

# --- Checks and cleanup ------------------------------------------------------

test: ## Run the Go tests (ARGS=-v, ARGS=-cover)
	go test $(ARGS) ./...

hooks: ## Install the pre-push hook (runs CI's web checks locally before a push that touches web/)
	git config core.hooksPath .githooks

docs: ## Regenerate docs/reference/selfhostlyctl.md from the command definitions
	go run ./cmd/selfhostlyctl docs --out docs/reference/selfhostlyctl.md

ctl: ## Build the selfhostlyctl command line into bin/
	go build -ldflags "-X github.com/selfhostly/internal/ctl.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)" -o bin/selfhostlyctl ./cmd/selfhostlyctl

e2e-update: ## Build production images locally and test the UI-driven update in a browser (OLD_REF=, SKIP_BUILD=1, KEEP=1)
	scripts/e2e/run.sh all

clean: ## Remove containers, volumes and build output
	-scripts/e2e/run.sh purge
	$(DC) -f docker-compose.dev.yml down -v
	$(DC) -f docker-compose.prod.yml down -v
	rm -rf tmp tmp-gateway build-errors.log build-errors-gateway.log

# Not listed in help: guards used by the run commands above.
.PHONY: check-env check-air
check-env:
	@test -f $(ENV_FILE) || { echo "$(ENV_FILE) not found. Copy env.example to $(ENV_FILE) first."; exit 1; }

check-air:
	@$(if $(NOAIR),true,command -v air >/dev/null || { echo "Installing Air"; go install github.com/air-verse/air@latest; })
