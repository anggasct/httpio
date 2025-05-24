#!/bin/bash

# Migration script for reorganizing goclient project structure
# This script helps migrate from current structure to recommended structure

set -e

echo "🚀 Starting goclient project restructuring..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    print_error "go.mod not found. Please run this script from the project root."
    exit 1
fi

# Backup current structure
print_info "Creating backup of current structure..."
BACKUP_DIR="backup_$(date +%Y%m%d_%H%M%S)"
if [ ! -d "$BACKUP_DIR" ]; then
    mkdir -p "$BACKUP_DIR"
    # Copy files excluding the backup directory itself
    find . -maxdepth 1 -type f -exec cp {} "$BACKUP_DIR/" \;
    # Copy directories excluding backup directories
    find . -maxdepth 1 -type d ! -name "." ! -name "backup_*" -exec cp -r {} "$BACKUP_DIR/" \;
    print_status "Backup created in $BACKUP_DIR"
fi

# Create new directory structure
print_info "Creating new directory structure..."

# Main directories
mkdir -p pkg/goclient
mkdir -p pkg/circuitbreaker
mkdir -p pkg/streaming
mkdir -p pkg/interceptors
mkdir -p internal/retry
mkdir -p internal/pool
mkdir -p internal/metrics
mkdir -p internal/utils
mkdir -p tests/integration
mkdir -p tests/benchmarks
mkdir -p tests/testdata/{responses,configs}
mkdir -p docs
mkdir -p scripts
mkdir -p tools
mkdir -p .github/workflows
mkdir -p .github/ISSUE_TEMPLATE

print_status "Directory structure created"

# Move and reorganize files
print_info "Moving and reorganizing files..."

# Move core client files to pkg/goclient
if [ -f "client.go" ]; then
    mv client.go pkg/goclient/
    print_status "Moved client.go"
fi

if [ -f "client_test.go" ]; then
    mv client_test.go pkg/goclient/
    print_status "Moved client_test.go"
fi

if [ -f "response.go" ]; then
    mv response.go pkg/goclient/
    print_status "Moved response.go"
fi

if [ -f "methods.go" ]; then
    mv methods.go pkg/goclient/
    print_status "Moved methods.go"
fi

# Move circuit breaker files
if [ -f "circuit_breaker.go" ]; then
    mv circuit_breaker.go pkg/circuitbreaker/
    print_status "Moved circuit_breaker.go"
fi

if [ -f "circuit_breaker_test.go" ]; then
    mv circuit_breaker_test.go pkg/circuitbreaker/
    print_status "Moved circuit_breaker_test.go"
fi

# Move streaming files
if [ -f "stream.go" ]; then
    mv stream.go pkg/streaming/
    print_status "Moved stream.go"
fi

if [ -f "stream_test.go" ]; then
    mv stream_test.go pkg/streaming/
    print_status "Moved stream_test.go"
fi

# Move interceptor files
if [ -f "interceptor.go" ]; then
    mv interceptor.go pkg/interceptors/
    print_status "Moved interceptor.go"
fi

# Reorganize examples
print_info "Reorganizing examples..."
if [ -d "example" ]; then
    if [ ! -d "examples" ]; then
        mv example examples
    else
        cp -r example/* examples/
        rm -rf example
    fi
    print_status "Examples reorganized"
fi

# Create essential files
print_info "Creating essential files..."

# Create main package file (pkg/goclient/goclient.go)
cat > pkg/goclient/goclient.go << 'EOF'
// Package goclient provides a minimal and elegant HTTP client wrapper for Go
package goclient

// Re-export main types for backward compatibility
// This ensures existing code continues to work
var (
    // New creates a new HTTP client
    New = newClient
)

// Re-export main types
type (
    Client   = client
    Response = response
    Request  = request
)
EOF

# Create Makefile
cat > Makefile << 'EOF'
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
EOF

# Create .gitignore
cat > .gitignore << 'EOF'
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with `go test -c`
*.test

# Output of the go coverage tool
*.out
coverage.html

# Dependency directories (remove the comment below to include it)
vendor/

# Go workspace file
go.work

# IDE files
.vscode/
.idea/
*.swp
*.swo
*~

# OS files
.DS_Store
Thumbs.db

# Backup files
backup_*/

# Temporary files
tmp/
temp/

# Log files
*.log
EOF

# Create tools.go for development dependencies
cat > tools/tools.go << 'EOF'
//go:build tools
// +build tools

package tools

import (
    _ "github.com/golangci/golangci-lint/cmd/golangci-lint"
    _ "mvdan.cc/gofumpt"
)
EOF

cat > tools/go.mod << 'EOF'
module github.com/anggasct/goclient/tools

go 1.18

require (
    github.com/golangci/golangci-lint v1.54.2
    mvdan.cc/gofumpt v0.5.0
)
EOF

# Create basic GitHub workflow
cat > .github/workflows/ci.yml << 'EOF'
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        go-version: [1.18, 1.19, 1.20, 1.21]

    steps:
    - uses: actions/checkout@v4

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: ${{ matrix.go-version }}

    - name: Cache Go modules
      uses: actions/cache@v3
      with:
        path: ~/go/pkg/mod
        key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
        restore-keys: |
          ${{ runner.os }}-go-

    - name: Download dependencies
      run: go mod download

    - name: Run tests
      run: make test

    - name: Run integration tests
      run: make test-integration

    - name: Run benchmarks
      run: make test-bench

  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: 1.21

    - name: golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest
EOF

print_status "Essential files created"

# Create documentation files
print_info "Creating documentation structure..."

cat > docs/README.md << 'EOF'
# Documentation

This directory contains comprehensive documentation for goclient.

## Files

- [getting-started.md](getting-started.md) - Quick start guide
- [advanced-usage.md](advanced-usage.md) - Advanced features and usage
- [circuit-breaker.md](circuit-breaker.md) - Circuit breaker pattern documentation
- [streaming.md](streaming.md) - Streaming and SSE documentation
- [interceptors.md](interceptors.md) - Request/response interceptors
- [api-reference.md](api-reference.md) - Complete API reference

## Examples

See the [examples/](../examples/) directory for working code examples.
EOF

# Create scripts
cat > scripts/build.sh << 'EOF'
#!/bin/bash
set -e

echo "Building goclient..."
go build ./pkg/...
echo "Build completed successfully!"
EOF

cat > scripts/test.sh << 'EOF'
#!/bin/bash
set -e

echo "Running tests..."
go test -v ./pkg/...
go test -v ./tests/integration/...
echo "All tests passed!"
EOF

chmod +x scripts/*.sh

print_status "Documentation and scripts created"

# Update go.mod if needed
print_info "Updating go.mod..."
go mod tidy

print_status "Migration completed successfully!"

echo ""
print_info "Next steps:"
echo "1. Review the new structure"
echo "2. Update import paths in your code"
echo "3. Run 'make test' to ensure everything works"
echo "4. Update documentation as needed"
echo "5. Commit changes"

echo ""
print_warning "Important notes:"
echo "- Backup was created in backup_$(date +%Y%m%d_%H%M%S)/"
echo "- You may need to update import paths in dependent projects"
echo "- Review and update the package documentation"
echo "- Consider creating release notes"

echo ""
print_status "🎉 Project restructuring completed!"
