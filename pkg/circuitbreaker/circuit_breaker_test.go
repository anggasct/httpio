package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerBasics(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		RecoveryTimeout:  100 * time.Millisecond,
		HalfOpenMaxCalls: 2,
	}

	cb := NewCircuitBreaker(config)

	// Initially closed
	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be CLOSED, got %s", cb.GetState())
	}

	// Record failures to open the circuit
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			return errors.New("test failure")
		})
		if err == nil {
			t.Errorf("Expected error but got nil on failure %d", i+1)
		}
	}

	// Circuit should be open now
	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be OPEN after failures, got %s", cb.GetState())
	}

	// Requests should be rejected
	err := cb.Execute(func() error {
		return nil
	})
	if err == nil {
		t.Error("Expected circuit breaker rejection but got nil")
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	}

	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be OPEN, got %s", cb.GetState())
	}

	// Wait for recovery timeout
	time.Sleep(60 * time.Millisecond)

	// First request should move to half-open and succeed
	err := cb.Execute(func() error {
		return nil // Success
	})

	if err != nil {
		t.Errorf("Expected successful execution in half-open state, got: %v", err)
	}

	// Should be closed now after success
	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be CLOSED after successful half-open request, got %s", cb.GetState())
	}
}

func TestCircuitBreakerStateChanges(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	}

	cb := NewCircuitBreaker(config)

	var stateChanges []string
	cb.OnStateChange(func(from, to CircuitBreakerState) {
		stateChanges = append(stateChanges, from.String()+"->"+to.String())
	})

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	// Wait for recovery timeout
	time.Sleep(60 * time.Millisecond)

	// Execute in half-open
	cb.Execute(func() error {
		return nil // Success
	})

	// Give callback time to execute
	time.Sleep(10 * time.Millisecond)

	expectedChanges := []string{"CLOSED->OPEN", "OPEN->HALF_OPEN", "HALF_OPEN->CLOSED"}
	if len(stateChanges) != len(expectedChanges) {
		t.Errorf("Expected %d state changes, got %d: %v", len(expectedChanges), len(stateChanges), stateChanges)
	}

	for i, expected := range expectedChanges {
		if i >= len(stateChanges) || stateChanges[i] != expected {
			t.Errorf("Expected state change %d to be %s, got %s", i, expected, stateChanges[i])
		}
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  1 * time.Hour, // Long timeout to ensure reset works
		HalfOpenMaxCalls: 1,
	}

	cb := NewCircuitBreaker(config)

	// Open the circuit
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be OPEN, got %s", cb.GetState())
	}

	// Reset should close the circuit immediately
	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be CLOSED after reset, got %s", cb.GetState())
	}

	// Should be able to execute requests normally
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected successful execution after reset, got: %v", err)
	}
}
