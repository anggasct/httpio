// Package headers provides middleware for adding custom headers to HTTP requests.
package headers

import (
	"context"
	"net/http"

	"github.com/anggasct/httpio/middleware"
)

// Config represents the configuration for the headers middleware
type Config struct {
	// Headers contains the custom headers to add to all requests
	Headers map[string]string
	// ConditionalHeaders contains headers that are added based on request conditions
	ConditionalHeaders []ConditionalHeader
	// OverwriteExisting determines whether to overwrite existing headers with the same name
	OverwriteExisting bool
}

// ConditionalHeader represents a header that is added based on request conditions
type ConditionalHeader struct {
	// Header name and value
	Name  string
	Value string
	// Condition function that determines if this header should be added
	Condition func(*http.Request) bool
}

// Middleware is the headers middleware implementation
type Middleware struct {
	config *Config
}

// New creates a new headers middleware with the provided configuration
func New(config *Config) *Middleware {
	if config == nil {
		config = &Config{
			Headers:            make(map[string]string),
			ConditionalHeaders: make([]ConditionalHeader, 0),
			OverwriteExisting:  false,
		}
	}

	if config.Headers == nil {
		config.Headers = make(map[string]string)
	}

	if config.ConditionalHeaders == nil {
		config.ConditionalHeaders = make([]ConditionalHeader, 0)
	}

	return &Middleware{
		config: config,
	}
}

// NewSimple creates a new headers middleware with simple static headers
// This is a convenience function for the common case of just adding static headers
func NewSimple(headers map[string]string) *Middleware {
	return New(&Config{
		Headers:            headers,
		ConditionalHeaders: make([]ConditionalHeader, 0),
		OverwriteExisting:  false,
	})
}

// Handle implements the middleware.Middleware interface
func (m *Middleware) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		for name, value := range m.config.Headers {
			if m.config.OverwriteExisting || req.Header.Get(name) == "" {
				req.Header.Set(name, value)
			}
		}

		for _, conditionalHeader := range m.config.ConditionalHeaders {
			if conditionalHeader.Condition != nil && conditionalHeader.Condition(req) {
				if m.config.OverwriteExisting || req.Header.Get(conditionalHeader.Name) == "" {
					req.Header.Set(conditionalHeader.Name, conditionalHeader.Value)
				}
			}
		}

		return next(ctx, req)
	}
}
