// Package example demonstrates custom middleware implementations for httpio
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/anggasct/httpio"
	"github.com/anggasct/httpio/middleware"
)

// RequestTimer is a struct-based middleware that measures request duration
type RequestTimer struct {
	name string
}

// Config holds configuration for the RequestTimer middleware
type Config struct {
	Name           string
	EnableDetailed bool
	Threshold      time.Duration
}

// NewRequestTimer creates a new RequestTimer middleware
func NewRequestTimer(config *Config) *RequestTimer {
	if config == nil {
		config = &Config{Name: "RequestTimer"}
	}
	return &RequestTimer{
		name: config.Name,
	}
}

// Handle implements the middleware.Middleware interface
func (rt *RequestTimer) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		start := time.Now()

		// Add request start time to context
		ctx = context.WithValue(ctx, "request_start", start)

		// Call next middleware/handler
		resp, err := next(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Log timing information
		fmt.Printf("[%s] %s %s took %v\n",
			rt.name,
			req.Method,
			req.URL.Path,
			duration)

		// Add timing header to response
		if resp != nil {
			resp.Header.Set("X-Request-Duration", duration.String())
		}

		return resp, err
	}
}

// Function-based middleware examples

// correlationIDMiddleware is a function-based middleware that adds correlation IDs
func correlationIDMiddleware(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		// Check if correlation ID already exists
		correlationID := req.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			// Generate new correlation ID
			correlationID = fmt.Sprintf("req-%d", time.Now().UnixNano())
			req.Header.Set("X-Correlation-ID", correlationID)
		}

		// Add to context for downstream use
		ctx = context.WithValue(ctx, "correlation_id", correlationID)

		fmt.Printf("Processing request with correlation ID: %s\n", correlationID)

		// Call next handler
		resp, err := next(ctx, req)

		// Add correlation ID to response
		if resp != nil {
			resp.Header.Set("X-Correlation-ID", correlationID)
		}

		return resp, err
	}
}

// userAgentMiddleware is a function-based middleware that adds/modifies User-Agent header
func userAgentMiddleware(userAgent string) middleware.MiddlewareFunc {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			// Set custom User-Agent
			originalUA := req.Header.Get("User-Agent")
			if originalUA != "" {
				req.Header.Set("User-Agent", fmt.Sprintf("%s %s", userAgent, originalUA))
			} else {
				req.Header.Set("User-Agent", userAgent)
			}

			fmt.Printf("Set User-Agent: %s\n", req.Header.Get("User-Agent"))

			res, err := next(ctx, req)

			return res, err
		}
	}
}

func main() {
	client := httpio.New().
		WithBaseURL("https://httpbin.org").
		WithTimeout(30 * time.Second)

	timer := NewRequestTimer(&Config{
		Name:           "APITimer",
		EnableDetailed: true,
		Threshold:      time.Second,
	})

	ctx := context.Background()

	test := correlationIDMiddleware

	combinedClient := client.
		WithMiddleware(timer).                                                            // struct-based
		WithMiddleware(middleware.WrapMiddleware(test)).                                  // function-based
		WithMiddleware(middleware.WrapMiddleware(userAgentMiddleware("CombinedApp/1.0"))) // function-based

	resp, err := combinedClient.POST(ctx, "/post", map[string]interface{}{
		"message": "Hello from combined middleware!",
		"data":    []int{1, 2, 3, 4, 5},
	})
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		defer resp.Close()
		fmt.Printf("Response status: %s\n", resp.Status)
		fmt.Printf("All headers: %v\n", resp.Header)
	}
}
