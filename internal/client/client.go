package client

import (
	"net/http"
	"time"

	"github.com/anggasct/httpio/internal/middleware"
)

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
