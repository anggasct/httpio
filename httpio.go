// Package httpio provides HTTP client utilities with middlewares and stream capabilities.
package httpio

import (
	"context"
	"encoding/json"

	"github.com/anggasct/httpio/internal/client"
	"github.com/anggasct/httpio/internal/middleware"
	"github.com/anggasct/httpio/internal/middleware/circuitbreaker"
	"github.com/anggasct/httpio/internal/middleware/logger"
	"github.com/anggasct/httpio/internal/middleware/oauth"
	"github.com/anggasct/httpio/internal/middleware/retry"
	"github.com/anggasct/httpio/internal/stream"
)

// NewClient creates a new HTTP client with httpio's enhanced functionality.
func NewClient() *client.Client {
	return client.New()
}

// Middleware types and utilities

// Middleware defines the interface that all middleware must implement.
type Middleware = middleware.Middleware

// Handler defines the HTTP handler function signature.
type Handler = middleware.Handler

type CircuitBreakerConfig = circuitbreaker.Config
type LoggerConfig = logger.Config
type OAuthConfig = oauth.Config
type RetryConfig = retry.Config

// CircuitBreaker returns a new circuit breaker middleware.
func NewCircuitBreakerMiddleware(config *CircuitBreakerConfig) Middleware {
	return circuitbreaker.New(config)
}

// NewLoggerMiddleware returns a new logger middleware.
func NewLoggerMiddleware(config *LoggerConfig) Middleware {
	return logger.New(config)
}

// NewOAuthMiddleware returns a new OAuth middleware.
func NewOAuthMiddleware(config *OAuthConfig) Middleware {
	return oauth.New(config)
}

// NewRetryMiddleware returns a new retry middleware.
func NewRetryMiddleware(config *RetryConfig) Middleware {
	return retry.New(config)
}

// Streaming utilities

// SSEEvent represents a Server-Sent Event with all its fields
type SSEEvent = stream.SSEEvent

// EventSourceHandler is a handler function type for Server-Sent Events
type EventSourceHandler = stream.EventSourceHandler

// StreamOption represents an option for stream processing
type StreamOption = stream.StreamOption

// WithBufferSize sets the buffer size for stream reading
func WithBufferSize(size int) StreamOption {
	return stream.WithBufferSize(size)
}

// WithDelimiter sets the delimiter for line-based stream reading
func WithDelimiter(delimiter string) StreamOption {
	return stream.WithDelimiter(delimiter)
}

// WithByteDelimiter sets a byte delimiter for stream reading
func WithByteDelimiter(delimiter byte) StreamOption {
	return stream.WithByteDelimiter(delimiter)
}

// WithContentType sets the expected content type for the stream
func WithContentType(contentType string) StreamOption {
	return stream.WithContentType(contentType)
}

// GetStream processes a stream response from the specified path with a handler function
func GetStream(c *client.Client, ctx context.Context, path string, handler func([]byte) error, opts ...StreamOption) error {
	return stream.GetStream(c, ctx, path, handler, opts...)
}

// GetStreamLines processes a stream response from the specified path line by line
func GetStreamLines(c *client.Client, ctx context.Context, path string, handler func([]byte) error, opts ...StreamOption) error {
	return stream.GetStreamLines(c, ctx, path, handler, opts...)
}

// GetStreamJSON processes a stream response from the specified path as JSON objects
func GetStreamJSON(c *client.Client, ctx context.Context, path string, handler func(json.RawMessage) error, opts ...StreamOption) error {
	return stream.GetStreamJSON(c, ctx, path, handler, opts...)
}

// GetStreamInto processes a stream response into typed objects
func GetStreamInto[T any](c *client.Client, ctx context.Context, path string, handler func(T) error, opts ...StreamOption) error {
	return stream.GetStreamInto(c, ctx, path, handler, opts...)
}

// GetSSE processes a Server-Sent Events stream from the specified path
func GetSSE(c *client.Client, ctx context.Context, path string, handler EventSourceHandler) error {
	return stream.GetSSE(c, ctx, path, handler)
}
