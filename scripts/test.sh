#!/bin/bash
set -e

echo "Running tests..."
go test -v ./pkg/...
go test -v ./tests/integration/...
echo "All tests passed!"
