package client

import (
	"log"
	"net/http"
	"time"
)

// LoggingConfig holds configuration for request and response logging
type LoggingConfig struct {
	Enabled        bool
	LogRequestBody bool
	Logger         *log.Logger
}

// RetryStrategy defines the strategy for retry delays
type RetryStrategy int

const (
	RetryStrategyFixed RetryStrategy = iota
	RetryStrategyExponential
)

// RetryConfig holds configuration for request retries with unified strategy
type RetryConfig struct {
	MaxRetries  int
	RetryPolicy func(resp *http.Response, err error) bool
	Strategy    RetryStrategy

	// For fixed strategy
	FixedDelay time.Duration

	// For exponential strategy
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

// ConnectionPoolConfig holds connection pool configuration
type ConnectionPoolConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	KeepAlive           time.Duration
}

// DefaultRetryPolicy returns true if the error is non-nil or the response has a status code >= 500
func DefaultRetryPolicy(resp *http.Response, err error) bool {
	return err != nil || (resp != nil && resp.StatusCode >= 500)
}

// WithMaxDelay sets the maximum delay for exponential backoff retries
func WithMaxDelay(maxDelay time.Duration) func(*RetryConfig) {
	return func(config *RetryConfig) {
		config.MaxDelay = maxDelay
	}
}

// WithMultiplier sets the multiplier for exponential backoff retries
func WithMultiplier(multiplier float64) func(*RetryConfig) {
	return func(config *RetryConfig) {
		config.Multiplier = multiplier
	}
}

// WithJitter enables or disables jitter for exponential backoff retries
func WithJitter(enabled bool) func(*RetryConfig) {
	return func(config *RetryConfig) {
		config.Jitter = enabled
	}
}

// WithRetryPolicy sets a custom retry policy
func WithRetryPolicy(policy func(resp *http.Response, err error) bool) func(*RetryConfig) {
	return func(config *RetryConfig) {
		config.RetryPolicy = policy
	}
}
