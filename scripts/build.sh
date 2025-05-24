#!/bin/bash
set -e

echo "Building goclient..."
go build ./pkg/...
echo "Build completed successfully!"
