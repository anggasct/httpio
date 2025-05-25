package middleware

import (
	"context"
	"net/http"
)

// Handler defines the HTTP handler function signature
type Handler func(ctx context.Context, req *http.Request) (*http.Response, error)

// MiddlewareHandler defines the interface that all middleware must implement
type Middleware interface {
	// Handler wraps the next handler and returns a new handler
	Handle(next Handler) Handler
}

// Chain applies a series of middleware to a base handler function
// The middlewares are applied in reverse order, so the first middleware
// in the list is the outermost wrapper (executes first on the request, last on the response)
func Chain(base Handler, middlewares ...Middleware) Handler {
	handler := base

	for i := len(middlewares) - 1; i >= 0; i-- {
		middleware := middlewares[i]
		handler = middleware.Handle(handler)
	}

	return handler
}

// function-based middleware
type functionMiddleware struct {
	fn func(next Handler) Handler
}

func (m *functionMiddleware) Handle(next Handler) Handler {
	return m.fn(next)
}

// Middleware defines a function that wraps an HTTP handler and returns a new handler
// This is kept for backward compatibility
type MiddlewareFunc func(next Handler) Handler

// WrapMiddleware converts a function-based middleware to a struct-based one
func WrapMiddleware(mw MiddlewareFunc) Middleware {
	return &functionMiddleware{fn: mw}
}
