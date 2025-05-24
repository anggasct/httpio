.PHONY: help test test-integration test-bench lint fmt build clean examples docs

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Testing
test: ## Run unit tests
	go test -v ./pkg/...

test-integration: ## Run integration tests
	go test -v ./tests/integration/...

test-bench: ## Run benchmark tests
	go test -bench=. -benchmem ./tests/benchmarks/...

test-all: test test-integration test-bench ## Run all tests

# Code quality
lint: ## Run linters
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	gofumpt -w .

# Build
build: ## Build the library
	go build ./pkg/...

# Development
clean: ## Clean build artifacts
	go clean ./...
	rm -f coverage.out

examples: ## Run examples
	cd examples/basic && go run .
	cd examples/circuit-breaker && go run .
	cd examples/streaming && go run .

# Documentation
docs: ## Generate documentation
	go doc -all ./pkg/goclient > docs/api-reference.md

# Release
release: test-all lint ## Prepare for release
	@echo "Ready for release"
