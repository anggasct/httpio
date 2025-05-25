// Package circuitbreaker provides a circuit breaker middleware implementation for httpio.
//
// The circuit breaker pattern is designed to detect failures and prevent cascading failures
// in distributed systems. It works by monitoring consecutive failures and automatically
// rejecting requests when a failure threshold is reached. After a recovery timeout,
// it allows a limited number of test requests to determine if the system has recovered.
//
// Important: This implementation tracks consecutive failures, not just a total number
// of failures. Any successful request will reset the failure counter. This ensures
// that intermittent failures don't trigger the circuit breaker unnecessarily.
package circuitbreaker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/anggasct/httpio/internal/middleware"
)

// CircuitBreakerState represents the current state of the circuit breaker
type CircuitBreakerState int

const (
	// StateClosed means the circuit breaker is closed and requests are allowed
	StateClosed CircuitBreakerState = iota
	// StateOpen means the circuit breaker is open and requests are rejected
	StateOpen
	// StateHalfOpen means the circuit breaker is in half-open state, allowing limited requests
	StateHalfOpen
)

// String returns a string representation of the circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Config holds the configuration for a circuit breaker
type Config struct {
	// FailureThreshold is the number of consecutive failures required to trip the circuit
	FailureThreshold int
	// RecoveryTimeout is the time to wait before attempting to close the circuit again
	RecoveryTimeout time.Duration
	// HalfOpenMaxCalls is the maximum number of requests allowed in half-open state
	HalfOpenMaxCalls int
	// OnStateChange is called whenever the circuit breaker changes state
	OnStateChange func(from, to CircuitBreakerState)
	// ErrorPredicate is used to determine if a response should count as a failure
	// Default: returns true for any non-nil error or any status code >= 500
	ErrorPredicate func(resp *http.Response, err error) bool
}

// DefaultConfig returns a Config with sensible default values
func DefaultConfig() *Config {
	return &Config{
		FailureThreshold: 5,
		RecoveryTimeout:  60 * time.Second,
		HalfOpenMaxCalls: 3,
		ErrorPredicate:   defaultErrorPredicate,
	}
}

// defaultErrorPredicate determines if a response should count as a failure
func defaultErrorPredicate(resp *http.Response, err error) bool {
	if err != nil {
		return true
	}
	if resp != nil && resp.StatusCode >= 500 {
		return true
	}
	return false
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu                sync.RWMutex
	state             CircuitBreakerState
	config            *Config
	consecutiveErrors int
	lastAttempt       time.Time
	halfOpenCalls     int
	onStateChange     func(from, to CircuitBreakerState)
}

// transitionState changes the circuit breaker state and triggers the state change notification
func (c *CircuitBreaker) transitionState(newState CircuitBreakerState) {
	if c.state == newState {
		return
	}

	oldState := c.state
	c.state = newState

	if c.onStateChange != nil {
		go c.onStateChange(oldState, newState)
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetConsecutiveErrors returns the current consecutive error count
func (cb *CircuitBreaker) GetConsecutiveErrors() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.consecutiveErrors
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.consecutiveErrors = 0
	cb.halfOpenCalls = 0

	if cb.onStateChange != nil && oldState != StateClosed {
		go cb.onStateChange(oldState, StateClosed)
	}
}

// IsOpen returns true if the circuit is open or half-open
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == StateOpen || cb.state == StateHalfOpen
}

// String returns a string representation of the circuit breaker
func (cb *CircuitBreaker) String() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return fmt.Sprintf("CircuitBreaker [State: %s, Consecutive Errors: %d/%d]",
		cb.state, cb.consecutiveErrors, cb.config.FailureThreshold)
}

// Middleware wraps the circuit breaker as an httpio middleware
type Middleware struct {
	cb *CircuitBreaker
}

// NewMiddleware creates a new circuit breaker middleware with the given configuration
func New(config *Config) *Middleware {
	return &Middleware{
		cb: NewCircuitBreaker(config),
	}
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config *Config) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig()
	}

	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.RecoveryTimeout <= 0 {
		config.RecoveryTimeout = 60 * time.Second
	}
	if config.HalfOpenMaxCalls <= 0 {
		config.HalfOpenMaxCalls = 3
	}

	cb := &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}

	if config.OnStateChange != nil {
		cb.onStateChange = config.OnStateChange
	}

	return cb
}

// Handle implements the MiddlewareHandler interface
func (m *Middleware) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		modifiedReq, err := m.processRequest(ctx, req)
		if err != nil {
			return nil, err
		}

		return m.processResponse(next(ctx, modifiedReq))
	}
}

// ProcessRequest checks if the request can proceed based on circuit breaker state
func (m *Middleware) processRequest(ctx context.Context, req *http.Request) (*http.Request, error) {
	m.cb.mu.RLock()
	state := m.cb.state
	m.cb.mu.RUnlock()

	switch state {
	case StateOpen:
		if time.Since(m.cb.lastAttempt) > m.cb.config.RecoveryTimeout {
			m.cb.mu.Lock()
			if m.cb.state == StateOpen {
				m.cb.transitionState(StateHalfOpen)
				m.cb.halfOpenCalls = 0
			}
			m.cb.mu.Unlock()
		} else {
			return req, errors.New("circuit breaker is open - request rejected")
		}

	case StateHalfOpen:
		m.cb.mu.Lock()
		defer m.cb.mu.Unlock()

		if m.cb.halfOpenCalls >= m.cb.config.HalfOpenMaxCalls {
			return req, errors.New("circuit breaker is half-open and maximum test requests reached")
		}
		m.cb.halfOpenCalls++
	}

	return req, nil
}

// ProcessResponse records the success or failure of a request
func (m *Middleware) processResponse(resp *http.Response, err error) (*http.Response, error) {
	m.cb.mu.Lock()
	defer m.cb.mu.Unlock()

	predicate := m.cb.config.ErrorPredicate
	if predicate == nil {
		predicate = defaultErrorPredicate
	}

	isFailure := predicate(resp, err)
	m.cb.lastAttempt = time.Now()

	switch m.cb.state {
	case StateClosed:
		if isFailure {
			m.cb.consecutiveErrors++
			if m.cb.consecutiveErrors >= m.cb.config.FailureThreshold {
				m.cb.transitionState(StateOpen)
			}
		} else {
			m.cb.consecutiveErrors = 0
		}

	case StateHalfOpen:
		if isFailure {
			m.cb.transitionState(StateOpen)
		} else {
			m.cb.consecutiveErrors = 0

			if m.cb.halfOpenCalls >= m.cb.config.HalfOpenMaxCalls {
				m.cb.transitionState(StateClosed)
			}
		}
	}

	return resp, err
}

// GetCircuitBreaker returns the underlying CircuitBreaker for state inspection
func (m *Middleware) GetCircuitBreaker() *CircuitBreaker {
	return m.cb
}
