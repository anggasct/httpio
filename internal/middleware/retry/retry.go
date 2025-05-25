// Package retry provides a retry middleware implementation for httpio.
//
// The retry pattern automatically re-attempts failed HTTP requests using
// exponential backoff with jitter. This helps improve reliability in the presence
// of transient errors, such as network timeouts or temporary server issues.
//
// The middleware allows configuration of maximum retries, which HTTP status codes
// and errors are considered retryable, and the backoff strategy. Each retry waits
// for an exponentially increasing delay, randomized with jitter to avoid thundering herd.
//
// Important: Only requests that match the configured retryable status codes or errors
// will be retried. The middleware also ensures that request bodies are properly cloned
// for each retry, and respects the context deadline for cancellation.
package retry

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/anggasct/httpio/internal/middleware"
	"slices"
)

// Use a global random source for jitter calculation.
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// Config defines the configuration for the retry middleware.
type Config struct {
	// MaxRetries is the maximum number of retries before giving up.
	MaxRetries int
	// RetryableStatusCodes defines which HTTP status codes should trigger a retry.
	RetryableStatusCodes []int
	// BaseDelay is the base delay for exponential backoff.
	BaseDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration
	// ErrorPredicate allows custom error checking to determine if an error should be retried.
	ErrorPredicate func(err error) bool
	// JitterFactor is the randomization factor for backoff delay (0 = no jitter, 0.2 = 20% jitter, etc).
	JitterFactor float64
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxRetries: 3,
		RetryableStatusCodes: []int{
			http.StatusRequestTimeout,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  10 * time.Second,
		ErrorPredicate: func(err error) bool {
			return err != nil
		},
		JitterFactor: 0,
	}
}

// RetryMiddleware implements the struct-based middleware for retrying failed requests
type Middleware struct {
	config *Config
}

// New creates a new retry middleware with the provided configuration.
func New(config *Config) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}
	return &Middleware{
		config: config,
	}
}

// Handle implements the MiddlewareHandler interface
func (m *Middleware) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		resp, err := next(ctx, req)

		if err == nil && resp != nil && !shouldRetry(m.config, resp, err) {
			return resp, nil
		}

		var lastResp *http.Response = resp
		var lastErr error = err

		for attempt := 0; attempt < m.config.MaxRetries; attempt++ {
			if lastResp != nil && lastResp.Body != nil {
				lastResp.Body.Close()
			}

			backoffDuration := calcBackoff(m.config, attempt)
			select {
			case <-ctx.Done():
				return lastResp, ctx.Err()
			case <-time.After(backoffDuration):
			}

			retryReq := req.Clone(ctx)
			if retryReq.Body != nil && req.GetBody != nil {
				var bodyErr error
				retryReq.Body, bodyErr = req.GetBody()
				if bodyErr != nil {
					return lastResp, bodyErr
				}
			} else if retryReq.Body != nil {
				return lastResp, lastErr
			}

			timeout := 30 * time.Second
			if deadline, ok := ctx.Deadline(); ok {
				timeout = time.Until(deadline)
				if timeout <= 0 {
					return lastResp, ctx.Err()
				}
			}

			client := &http.Client{
				Timeout:   timeout,
				Transport: http.DefaultTransport,
			}

			retryResp, retryErr := client.Do(retryReq)
			lastResp = retryResp
			lastErr = retryErr

			if retryResp != nil && retryResp.StatusCode < 500 && retryErr == nil {
				return retryResp, retryErr
			}

			if !shouldRetry(m.config, retryResp, retryErr) {
				return retryResp, retryErr
			}
		}

		return lastResp, lastErr
	}
}

// shouldRetry checks if a response or error should trigger a retry.
func shouldRetry(config *Config, resp *http.Response, err error) bool {
	if err != nil && config.ErrorPredicate != nil {
		return config.ErrorPredicate(err)
	}
	if resp == nil {
		return false
	}
	return slices.Contains(config.RetryableStatusCodes, resp.StatusCode)
}

// calcBackoff calculates the exponential backoff delay with jitter.
func calcBackoff(config *Config, attempt int) time.Duration {
	delay := float64(config.BaseDelay) * math.Pow(2, float64(attempt))
	if config.MaxDelay > 0 && delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	jitterFactor := config.JitterFactor
	jitter := delay * jitterFactor * (2*rng.Float64() - 1)
	finalDelay := delay + jitter
	if config.MaxDelay > 0 && finalDelay > float64(config.MaxDelay) {
		finalDelay = float64(config.MaxDelay)
	}
	return time.Duration(finalDelay)
}
