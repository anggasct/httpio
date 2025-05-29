package test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/anggasct/httpio/middleware/retry"
)

func TestRetryMiddleware(t *testing.T) {
	attempts := 0

	config := retry.DefaultConfig()
	config.MaxRetries = 2
	config.BaseDelay = 10 * time.Millisecond

	retryMiddleware := retry.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return &http.Response{StatusCode: 500}, nil
		}
		return &http.Response{StatusCode: 200}, nil
	}

	handler := retryMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryMiddlewareWithError(t *testing.T) {
	attempts := 0

	config := retry.DefaultConfig()
	config.MaxRetries = 2
	config.BaseDelay = 10 * time.Millisecond
	config.ErrorPredicate = func(err error) bool {
		return err.Error() == "temporary error"
	}

	retryMiddleware := retry.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("temporary error")
		}
		return &http.Response{StatusCode: 200}, nil
	}

	handler := retryMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryMiddlewareNoRetryNeeded(t *testing.T) {
	attempts := 0

	config := retry.DefaultConfig()
	config.MaxRetries = 2

	retryMiddleware := retry.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: 200}, nil
	}

	handler := retryMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryMiddlewareMaxRetriesExceeded(t *testing.T) {
	attempts := 0

	config := retry.DefaultConfig()
	config.MaxRetries = 2
	config.BaseDelay = 10 * time.Millisecond

	retryMiddleware := retry.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: 500}, nil
	}

	handler := retryMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	if attempts != 3 { // initial + 2 retries
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryMiddlewareNonRetryableStatus(t *testing.T) {
	attempts := 0

	config := retry.DefaultConfig()
	config.MaxRetries = 2
	config.RetryableStatusCodes = []int{502, 503} // 500 is not retryable

	retryMiddleware := retry.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: 500}, nil
	}

	handler := retryMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	resp, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt (no retries), got %d", attempts)
	}
}

func TestRetryDefaultConfig(t *testing.T) {
	config := retry.DefaultConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected default max retries to be 3, got %d", config.MaxRetries)
	}

	if config.BaseDelay != 100*time.Millisecond {
		t.Errorf("Expected default base delay to be 100ms, got %v", config.BaseDelay)
	}

	if config.MaxDelay != 10*time.Second {
		t.Errorf("Expected default max delay to be 10s, got %v", config.MaxDelay)
	}

	if config.JitterFactor != 0.0 {
		t.Errorf("Expected default jitter factor to be 0.0, got %f", config.JitterFactor)
	}

	expectedRetryableStatus := []int{429, 500, 502, 503, 504}
	if len(config.RetryableStatusCodes) != len(expectedRetryableStatus) {
		t.Errorf("Expected %d retryable status codes, got %d", len(expectedRetryableStatus), len(config.RetryableStatusCodes))
	}
}

func TestRetryWithContextCancellation(t *testing.T) {
	attempts := 0

	config := retry.DefaultConfig()
	config.MaxRetries = 5
	config.BaseDelay = 100 * time.Millisecond

	retryMiddleware := retry.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: 500}, nil
	}

	handler := retryMiddleware.Handle(baseHandler)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	_, err := handler(ctx, req)

	// Should get a context deadline exceeded error
	if err == nil {
		t.Error("Expected context deadline exceeded error")
	}

	// Should have made at least one attempt but not all retries due to context cancellation
	if attempts == 0 {
		t.Error("Expected at least one attempt")
	}

	if attempts > 3 {
		t.Errorf("Expected context cancellation to prevent all retries, got %d attempts", attempts)
	}
}
