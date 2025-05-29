// Package httpio provides a simple HTTP client with streaming capabilities
package httpio

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/anggasct/httpio/internal/client"
	"github.com/anggasct/httpio/middleware"
	"github.com/anggasct/httpio/middleware/circuitbreaker"
	"github.com/anggasct/httpio/middleware/logger"
	"github.com/anggasct/httpio/middleware/oauth"
	"github.com/anggasct/httpio/middleware/retry"
)

// Request is a prepared HTTP request
type Request = client.Request

// Response wraps the standard http.Response with additional utility methods
type Response = client.Response

// Event represents a Server-Sent Event
type SSEEvent = client.Event

// EventSourceHandler handles incoming Server-Sent Events
type SSEEventSourceHandler = client.EventSourceHandler

// EventSourceFullHandler extends EventSourceHandler with lifecycle methods
type SSEEventSourceFullHandler = client.EventSourceFullHandler

// EventHandlerFunc is a function type for handling SSE events
type SSEEventHandlerFunc = client.EventHandlerFunc

// EventFullHandlerFunc represents a function-based handler with lifecycle support
type SSEEventFullHandlerFunc = client.EventFullHandlerFunc

// StreamOption represents options for stream processing
type StreamOption = client.StreamOption

// WithBufferSize sets the buffer size for stream reading
var WithBufferSize = client.WithBufferSize

// WithDelimiter sets the delimiter for line-based stream reading
var WithDelimiter = client.WithDelimiter

// WithByteDelimiter sets a byte delimiter for stream reading
var WithByteDelimiter = client.WithByteDelimiter

// WithContentType sets the expected content type for the stream
var WithContentType = client.WithContentType

// Client is a wrapper around http.Client with additional functionality
type Client struct {
	client      *http.Client
	baseURL     string
	headers     http.Header
	middlewares []middleware.Middleware
}

// New creates a new http Client
func New() *Client {
	c := &Client{
		client:      &http.Client{},
		headers:     make(http.Header),
		middlewares: make([]middleware.Middleware, 0),
	}
	c.headers.Set("User-Agent", "httpio")
	return c
}

// Do implements the client.HTTPClient interface
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

// GetMiddlewares implements the client.HTTPClient interface
func (c *Client) GetMiddlewares() []middleware.Middleware {
	return c.middlewares
}

// GET performs a GET request
func (c *Client) GET(ctx context.Context, path string) (*client.Response, error) {
	return c.NewRequest("GET", path).Do(ctx)
}

// POST performs a POST request
func (c *Client) POST(ctx context.Context, path string, body interface{}) (*client.Response, error) {
	return c.NewRequest("POST", path).WithBody(body).Do(ctx)
}

// PUT performs a PUT request
func (c *Client) PUT(ctx context.Context, path string, body interface{}) (*client.Response, error) {
	return c.NewRequest("PUT", path).WithBody(body).Do(ctx)
}

// PATCH performs a PATCH request
func (c *Client) PATCH(ctx context.Context, path string, body interface{}) (*client.Response, error) {
	return c.NewRequest("PATCH", path).WithBody(body).Do(ctx)
}

// DELETE performs a DELETE request
func (c *Client) DELETE(ctx context.Context, path string) (*client.Response, error) {
	return c.NewRequest("DELETE", path).Do(ctx)
}

// HEAD performs a HEAD request
func (c *Client) HEAD(ctx context.Context, path string) (*client.Response, error) {
	return c.NewRequest("HEAD", path).Do(ctx)
}

// OPTIONS performs an OPTIONS request
func (c *Client) OPTIONS(ctx context.Context, path string) (*client.Response, error) {
	return c.NewRequest("OPTIONS", path).Do(ctx)
}

// WithBaseURL sets the base URL for all requests
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

// WithHeader sets a header for all requests
func (c *Client) WithHeader(key, value string) *Client {
	c.headers.Set(key, value)
	return c
}

// WithHeaders sets multiple headers for all requests
func (c *Client) WithHeaders(headers map[string]string) *Client {
	for k, v := range headers {
		c.headers.Set(k, v)
	}
	return c
}

// WithTimeout sets the timeout for all requests
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.client.Timeout = timeout
	return c
}

// WithMiddleware adds a middleware to the client's middleware chain
// Middlewares are applied in the order they are added
func (c *Client) WithMiddleware(m middleware.Middleware) *Client {
	c.middlewares = append(c.middlewares, m)
	return c
}

// WithMiddlewares allows adding multiple middlewares to the client
func (c *Client) WithMiddlewares(middlewares ...middleware.Middleware) func(*Client) {
	return func(c *Client) {
		c.middlewares = append(c.middlewares, middlewares...)
	}
}

// WithConnectionPool configures the connection pool settings for the HTTP client
func (c *Client) WithConnectionPool(maxIdleConns, maxConnsPerHost, maxIdleConnsPerHost int, idleConnTimeout time.Duration) *Client {
	if c.client.Transport == nil {
		c.client.Transport = &http.Transport{}
	}

	transport, ok := c.client.Transport.(*http.Transport)
	if !ok {
		transport = &http.Transport{}
		c.client.Transport = transport
	}

	transport.MaxIdleConns = maxIdleConns
	transport.MaxConnsPerHost = maxConnsPerHost
	transport.MaxIdleConnsPerHost = maxIdleConnsPerHost
	transport.IdleConnTimeout = idleConnTimeout

	return c
}

// NewRequest creates a new request with the given method and URL
func (c *Client) NewRequest(method, path string) *client.Request {
	reqURL := path
	if c.baseURL != "" {
		reqURL = c.baseURL + path
	}

	req := &client.Request{
		Method:  method,
		URL:     reqURL,
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  c,
	}

	for k, vv := range c.headers {
		for _, v := range vv {
			req.Headers.Add(k, v)
		}
	}

	return req
}

// Middleware defines the interface for HTTP middleware
type Middleware = middleware.Middleware

// MiddlewareFunc is a function type for middleware that wraps an HTTP handler
type MiddlewareFunc = middleware.MiddlewareFunc

// CircuitBreakerConfig represents configuration for circuit breaker middleware.
type CircuitBreakerConfig = circuitbreaker.Config

// LoggerConfig represents configuration for logging middleware.
type LoggerConfig = logger.Config

// OAuthConfig represents configuration for OAuth middleware.
type OAuthConfig = oauth.Config

// RetryConfig represents configuration for retry middleware.
type RetryConfig = retry.Config

// NewCircuitBreakerMiddleware creates a new circuit breaker middleware.
func NewCircuitBreakerMiddleware(config *CircuitBreakerConfig) middleware.Middleware {
	return circuitbreaker.New(config)
}

// NewLoggerMiddleware creates a new logger middleware.
func NewLoggerMiddleware(config *LoggerConfig) middleware.Middleware {
	return logger.New(config)
}

// NewOAuthMiddleware creates a new OAuth middleware.
func NewOAuthMiddleware(config *OAuthConfig) middleware.Middleware {
	return oauth.New(config)
}

// NewRetryMiddleware creates a new retry middleware.
func NewRetryMiddleware(config *RetryConfig) middleware.Middleware {
	return retry.New(config)
}

// LogLevel defines logging verbosity.
type LogLevel = logger.LogLevel

// Log levels
const (
	LevelNone  = logger.LevelNone
	LevelError = logger.LevelError
	LevelInfo  = logger.LevelInfo
	LevelDebug = logger.LevelDebug
	LevelTrace = logger.LevelTrace
)
