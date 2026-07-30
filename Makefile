.PHONY: help install setup doctor env-check \
        sync-core build run dev \
        ui-install ui-build ui-dev ui-lint ui-typecheck \
        test test-php test-python test-all \
        lint fmt fmt-check check ci \
        docker-build docker-push \
        pre-commit-install pre-commit \
        docs-site-install docs-site-build \
        clean

SERVICE  = microscope
VERSION  ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
IMAGE    = $(SERVICE):$(VERSION)
GO_DIR   = adaptor/go
UI_DIR   = core/ui
PHP_DIR  = adaptor/php
PY_DIR   = adaptor/python

.DEFAULT_GOAL := help

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------

help: ## Show available targets
	@echo "$(SERVICE) — Multi-SDK development targets"
	@echo ""
	@echo "Setup:"
	@echo "  make install              Install all SDK dependencies"
	@echo "  make setup                Full bootstrap (alias for install)"
	@echo "  make doctor               Verify prerequisites"
	@echo "  make env-check            Verify runtime environment"
	@echo ""
	@echo "Development:"
	@echo "  make build                Build Go binary with UI assets"
	@echo "  make run                  Build and run microscope"
	@echo "  make dev                  Alias for ui-dev"
	@echo "  make ui-dev               Start UI dev server"
	@echo "  make ui-build             Build UI and sync core assets"
	@echo "  make ui-install           Install UI dependencies"
	@echo ""
	@echo "Quality:"
	@echo "  make lint                 Lint Go code"
	@echo "  make fmt                  Format Go code"
	@echo "  make fmt-check            CI-safe format check"
	@echo "  make test                 Run Go tests"
	@echo "  make test-all             Run tests across all SDKs"
	@echo "  make check                Run all quality checks"
	@echo "  make ci                   Full CI pipeline"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build         Build microscope Docker image"
	@echo "  make docker-push          Push Docker image"
	@echo ""
	@echo "Docs:"
	@echo "  make docs-site-install    Install docs-site deps"
	@echo "  make docs-site-build      Build documentation site"
	@echo ""
	@echo "Hooks:"
	@echo "  make pre-commit-install   Install pre-commit hooks"
	@echo "  make pre-commit           Run pre-commit on all files"
	@echo ""
	@echo "Maintenance:"
	@echo "  make sync-core            Sync core assets into adaptors"
	@echo "  make clean                Clean build artifacts"
	@echo ""

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------

install: ## Install all SDK dependencies
	@echo "Installing Go deps..."
	cd $(GO_DIR) && go mod download
	@echo "Installing UI deps..."
	cd $(UI_DIR) && pnpm install --frozen-lockfile
	@echo "Installing Python deps..."
	cd $(PY_DIR) && pip install -e ".[dev]" 2>/dev/null || true
	@echo "Installing PHP deps..."
	cd $(PHP_DIR) && composer install 2>/dev/null || true
	@echo "✓ Install complete"

setup: install ## Full bootstrap

doctor: ## Verify prerequisites
	@echo "Checking Go..."
	@go version 2>/dev/null | grep -q "go1.25" && echo "✓ Go 1.25" || echo "⚠ Expected Go 1.25"
	@echo "Checking Node.js..."
	@command -v node >/dev/null 2>&1 && echo "✓ Node.js $$(node --version)" || echo "✗ Node.js not found (needed for UI)"
	@echo "Checking pnpm..."
	@command -v pnpm >/dev/null 2>&1 && echo "✓ pnpm $$(pnpm --version)" || echo "✗ pnpm not found (needed for UI)"
	@echo "Checking Python..."
	@command -v python3 >/dev/null 2>&1 && echo "✓ Python $$(python3 --version)" || echo "⚠ Python not found"
	@echo "Checking PHP..."
	@command -v php >/dev/null 2>&1 && echo "✓ PHP $$(php --version | head -1)" || echo "⚠ PHP not found"
	@echo "Checking Docker..."
	@command -v docker >/dev/null 2>&1 && echo "✓ Docker" || echo "⚠ Docker not found (optional)"

env-check: ## Verify runtime environment
	@test -n "$$MICROSCOPE_DATABASE_URL" || test -f .env || \
		(echo "Error: set MICROSCOPE_DATABASE_URL or copy .env.example to .env"; exit 1)
	@echo "✓ Environment OK"

# ---------------------------------------------------------------------------
# Core assets
# ---------------------------------------------------------------------------

sync-core: ## Sync core assets into adaptors
	sh scripts/sync-core-assets.sh

# ---------------------------------------------------------------------------
# UI
# ---------------------------------------------------------------------------

ui-install: ## Install UI dependencies
	cd $(UI_DIR) && pnpm install --frozen-lockfile

ui-build: ui-install ## Build UI and sync core assets
	cd $(UI_DIR) && pnpm run build
	$(MAKE) sync-core

ui-dev: ## Start UI dev server
	cd $(UI_DIR) && pnpm run dev

dev: ui-dev ## Alias for UI development

ui-lint: ## Lint UI TypeScript
	@cd $(UI_DIR) && pnpm run lint 2>/dev/null || \
		echo "No UI linter configured (add a lint script to $(UI_DIR)/package.json)"

ui-typecheck: ## Type-check UI
	@cd $(UI_DIR) && pnpm exec vue-tsc --noEmit 2>/dev/null || \
		echo "UI typecheck failed or vue-tsc not available"

# ---------------------------------------------------------------------------
# Build & Run
# ---------------------------------------------------------------------------

build: sync-core ## Build Go binary with UI assets
	cd $(GO_DIR) && go build -o ../../bin/microscope ./cmd/server
	@echo "✓ Binary: bin/microscope"

run: build ## Build and run microscope
	@set -a; \
		if [ -f .env ]; then \
			MSYS_NO_PATHCONV=1 . ./.env; \
		fi; \
		set +a; \
		if [ -z "$$MICROSCOPE_DATABASE_URL" ]; then \
			echo "Error: set MICROSCOPE_DATABASE_URL or copy .env.example to .env"; \
			exit 1; \
		fi; \
		./bin/microscope

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------

test: sync-core ## Run Go tests
	cd $(GO_DIR) && go test ./...

test-php: ## Run PHP tests
	cd $(PHP_DIR) && composer test

test-python: ## Run Python tests
	cd $(PY_DIR) && python -m pytest

test-all: test test-php test-python ## Run tests across all SDKs

# ---------------------------------------------------------------------------
# Quality
# ---------------------------------------------------------------------------

lint: ## Lint Go code
	@if command -v golangci-lint >/dev/null 2>&1; then \
		cd $(GO_DIR) && golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed (skipping)"; \
	fi

fmt: ## Format Go code
	cd $(GO_DIR) && gofmt -w .
	@echo "✓ Go code formatted"

fmt-check: ## CI-safe format check
	@cd $(GO_DIR) && test -z "$$(gofmt -l .)" || (gofmt -l .; exit 1)
	@echo "✓ Formatting OK"

check: lint fmt-check test ## Pre-submit quality gate

ci: sync-core check test-all build ## Full CI pipeline

# ---------------------------------------------------------------------------
# Docker
# ---------------------------------------------------------------------------

docker-build: ## Build Docker image
	@echo "Building Docker image: $(IMAGE)"
	docker build -t $(IMAGE) -t $(SERVICE):latest .
	@echo "✓ Image built: $(IMAGE)"

docker-push: ## Push Docker image
	@echo "Pushing Docker image: $(IMAGE)"
	docker push $(IMAGE)
	@echo "✓ Image pushed"

# ---------------------------------------------------------------------------
# Pre-commit
# ---------------------------------------------------------------------------

pre-commit-install: ## Install pre-commit hooks
	@echo "Installing pre-commit hooks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit install; \
		echo "✓ Pre-commit hooks installed"; \
	else \
		echo "Error: pre-commit not found. Install with: pip install pre-commit"; \
		exit 1; \
	fi

pre-commit: ## Run pre-commit on all files
	pre-commit run --all-files

# ---------------------------------------------------------------------------
# Docs-site
# ---------------------------------------------------------------------------

docs-site-install: ## Install docs-site dependencies
	@echo "No docs-site directory yet. Documentation lives in core/docs/."

docs-site-build: ## Build documentation site
	@echo "No docs-site directory yet. Documentation lives in core/docs/."

# ---------------------------------------------------------------------------
# Maintenance
# ---------------------------------------------------------------------------

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf $(UI_DIR)/dist/
	rm -rf $(GO_DIR)/bin/
	cd $(GO_DIR) && go clean -cache 2>/dev/null || true
	@echo "✓ Cleaned"
