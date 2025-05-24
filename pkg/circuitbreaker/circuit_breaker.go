package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int
	RecoveryTimeout  time.Duration
	HalfOpenMaxCalls int
}

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

// String returns the string representation of the circuit breaker state
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

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config        *CircuitBreakerConfig
	state         CircuitBreakerState
	failureCount  int
	lastFailureAt time.Time
	halfOpenCount int
	mutex         sync.RWMutex
	onStateChange func(from, to CircuitBreakerState)
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config *CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.RecoveryTimeout <= 0 {
		config.RecoveryTimeout = 60 * time.Second
	}
	if config.HalfOpenMaxCalls <= 0 {
		config.HalfOpenMaxCalls = 3
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// OnStateChange sets a callback that will be called whenever the circuit breaker state changes
func (cb *CircuitBreaker) OnStateChange(callback func(from, to CircuitBreakerState)) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.onStateChange = callback
}

// Execute executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		return errors.New("circuit breaker is open - request rejected")
	}

	err := fn()
	cb.recordResult(err)
	return err
}

// allowRequest checks if the request should be allowed based on current state
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastFailureAt) >= cb.config.RecoveryTimeout {
			cb.setState(StateHalfOpen)
			cb.halfOpenCount = 0
			return true
		}
		return false
	case StateHalfOpen:
		if cb.halfOpenCount < cb.config.HalfOpenMaxCalls {
			cb.halfOpenCount++
			return true
		}
		return false
	default:
		return false
	}
}

// recordResult records the result of a request and updates the circuit breaker state accordingly
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
}

// RecordResult manually records a result for the circuit breaker
func (cb *CircuitBreaker) RecordResult(err error) {
	cb.recordResult(err)
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	cb.lastFailureAt = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.setState(StateOpen)
		}
	case StateHalfOpen:
		cb.setState(StateOpen)
		cb.failureCount = cb.config.FailureThreshold
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	switch cb.state {
	case StateClosed:
		cb.failureCount = 0
	case StateHalfOpen:
		cb.setState(StateClosed)
		cb.failureCount = 0
		cb.halfOpenCount = 0
	}
}

// setState changes the circuit breaker state and triggers the callback if set
func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	if cb.onStateChange != nil {
		// Call the callback synchronously to ensure proper ordering
		cb.onStateChange(oldState, newState)
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns the current statistics of the circuit breaker
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"state":           cb.state.String(),
		"failure_count":   cb.failureCount,
		"last_failure":    cb.lastFailureAt,
		"half_open_calls": cb.halfOpenCount,
	}
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.failureCount = 0
	cb.halfOpenCount = 0
	cb.lastFailureAt = time.Time{}

	if cb.onStateChange != nil && oldState != StateClosed {
		cb.onStateChange(oldState, StateClosed)
	}
}
