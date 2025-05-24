// Package goclient provides a minimal and elegant HTTP client wrapper for Go
package goclient

// This file serves as the main entry point for the goclient package.
// All main types and functions are exported from their respective files:
// - Client, New() from client.go
// - Response from response.go
// - HTTP methods (GET, POST, etc.) from methods.go
// - Interceptors from interceptor.go (moved to pkg/interceptors)
//
// This maintains backward compatibility while organizing code better.
