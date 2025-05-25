.PHONY: help test test-integration clean examples

# Default target
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Testing
test: ## Run unit tests
	go test -v ./...


test-all: test test-integration 

# Development
clean: ## Clean build artifacts
	go clean ./...
	rm -f coverage.out

examples: ## Run examples
	cd examples/basic && go run .
	cd examples/circuit-breaker && go run .
	cd examples/streaming && go run .
