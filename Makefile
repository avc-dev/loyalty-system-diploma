# Makefile for loyalty-system-diploma

.PHONY: help build test clean mocks generate-mocks install-mockery lint fmt vet

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the application
	go build -o bin/gophermart ./cmd/gophermart

build-accrual: ## Build the accrual service
	go build -o bin/accrual ./cmd/accrual

# Test targets
test: ## Run all tests
	go test ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-unit: ## Run unit tests only (excluding integration tests)
	go test -short ./...

# Mockery targets
install-mockery: ## Install mockery tool
	go install github.com/vektra/mockery/v2@latest

generate-mocks: ## Generate all mocks using mockery
	mockery

clean-mocks: ## Remove all generated mocks
	find internal/mocks -name "*_mock.go" -type f -delete

# Code quality targets
fmt: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run golangci-lint (if installed)
	golangci-lint run

# Database targets
migrate-up: ## Run database migrations up
	migrate -path migrations -database "postgres://user:password@localhost/loyalty_system?sslmode=disable" up

migrate-down: ## Run database migrations down
	migrate -path migrations -database "postgres://user:password@localhost/loyalty_system?sslmode=disable" down

# Docker targets
docker-build: ## Build Docker images
	docker-compose build

docker-up: ## Start Docker services
	docker-compose up -d

docker-down: ## Stop Docker services
	docker-compose down

# Development targets
dev: ## Run development setup (build and start services)
	make build
	make docker-up

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

# CI/CD targets
ci: ## Run CI pipeline (format, vet, test, build)
	make fmt
	make vet
	make test
	make build
