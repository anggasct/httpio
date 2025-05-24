package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/anggasct/goclient/pkg/circuitbreaker"
	"github.com/anggasct/goclient/pkg/goclient"
	"github.com/anggasct/goclient/pkg/interceptors"
	"github.com/anggasct/goclient/pkg/streaming"
)

// RequestError represents an HTTP error response
type RequestError struct {
	Status     string
	StatusCode int
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Status)
}

// TestIntegrationBasicClient tests basic client functionality with interceptors
func TestIntegrationBasicClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "test-value" {
			t.Errorf("Expected custom header not found")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	client := goclient.New().WithBaseURL(server.URL)

	// Add interceptor
	client.WithInterceptor(interceptors.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
		req.Header.Set("X-Custom-Header", "test-value")
		return req, nil
	}))

	ctx := context.Background()
	resp, err := client.GET(ctx, "/test")

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Close()

	if !resp.IsSuccess() {
		t.Errorf("Expected successful response, got %s", resp.Status)
	}

	var result map[string]string
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result["message"] != "success" {
		t.Errorf("Expected message 'success', got '%s'", result["message"])
	}
}

// TestIntegrationCircuitBreaker tests circuit breaker functionality
func TestIntegrationCircuitBreaker(t *testing.T) {
	failCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failCount++
		if failCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := goclient.New().WithBaseURL(server.URL)

	// Set up circuit breaker
	cb := circuitbreaker.NewCircuitBreaker(&circuitbreaker.CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	})

	stateChanges := []string{}
	cb.OnStateChange(func(from, to circuitbreaker.CircuitBreakerState) {
		stateChanges = append(stateChanges, from.String()+"->"+to.String())
	})

	ctx := context.Background()

	// Test circuit opening
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			resp, err := client.GET(ctx, "/test")
			if err != nil {
				return err
			}
			defer resp.Close()
			if !resp.IsSuccess() {
				return &RequestError{Status: resp.Status, StatusCode: resp.StatusCode}
			}
			return nil
		})
		if err == nil {
			t.Errorf("Expected error on attempt %d", i+1)
		}
	}

	// Verify circuit is open
	if cb.GetState() != circuitbreaker.StateOpen {
		t.Errorf("Expected circuit to be OPEN, got %s", cb.GetState())
	}

	// Wait for recovery timeout
	time.Sleep(60 * time.Millisecond)

	// Test recovery
	err := cb.Execute(func() error {
		resp, err := client.GET(ctx, "/test")
		if err != nil {
			return err
		}
		defer resp.Close()
		if !resp.IsSuccess() {
			return &RequestError{Status: resp.Status, StatusCode: resp.StatusCode}
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected successful recovery, got: %v", err)
	}

	// Verify circuit is closed
	if cb.GetState() != circuitbreaker.StateClosed {
		t.Errorf("Expected circuit to be CLOSED after recovery, got %s", cb.GetState())
	}

	// Verify state changes
	expectedChanges := []string{"CLOSED->OPEN", "OPEN->HALF_OPEN", "HALF_OPEN->CLOSED"}
	if len(stateChanges) != len(expectedChanges) {
		t.Errorf("Expected %d state changes, got %d: %v", len(expectedChanges), len(stateChanges), stateChanges)
	}
}

// TestIntegrationStreaming tests streaming functionality
func TestIntegrationStreaming(t *testing.T) {
	testData := "line1\nline2\nline3\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testData))
	}))
	defer server.Close()

	client := goclient.New().WithBaseURL(server.URL)
	ctx := context.Background()

	resp, err := client.GET(ctx, "/stream")
	if err != nil {
		t.Fatalf("Stream request failed: %v", err)
	}
	defer resp.Close()

	lines := []string{}
	err = streaming.StreamLines(resp, func(line []byte) error {
		lines = append(lines, string(line))
		return nil
	})

	if err != nil {
		t.Fatalf("Streaming failed: %v", err)
	}

	expectedLines := []string{"line1", "line2", "line3"}
	if len(lines) != len(expectedLines) {
		t.Errorf("Expected %d lines, got %d", len(expectedLines), len(lines))
	}

	for i, expected := range expectedLines {
		if i >= len(lines) || !strings.Contains(lines[i], expected) {
			t.Errorf("Expected line %d to contain '%s', got '%s'", i, expected, lines[i])
		}
	}
}

// TestIntegrationResponseInterceptor tests response interceptors
func TestIntegrationResponseInterceptor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"original": "data"}`))
	}))
	defer server.Close()

	client := goclient.New().WithBaseURL(server.URL)

	// Add response interceptor that modifies response
	client.WithResponseInterceptor(interceptors.ResponseInterceptorFunc(func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
		if err != nil {
			return resp, err
		}
		// Create a new response with modified content
		newResp := httptest.NewRecorder()
		newResp.Header().Set("Content-Type", "application/json")
		newResp.Header().Set("X-Modified", "true")
		newResp.WriteHeader(http.StatusOK)
		newResp.WriteString(`{"modified": "data"}`)
		return newResp.Result(), nil
	}))

	ctx := context.Background()
	resp, err := client.GET(ctx, "/test")

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Close()

	if resp.Header.Get("X-Modified") != "true" {
		t.Errorf("Expected X-Modified header to be 'true', got '%s'", resp.Header.Get("X-Modified"))
	}

	var result map[string]string
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if result["modified"] != "data" {
		t.Errorf("Expected modified data, got %v", result)
	}
}

// TestIntegrationChainedInterceptors tests multiple interceptors working together
func TestIntegrationChainedInterceptors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify both interceptors ran
		if r.Header.Get("X-Auth") != "Bearer token" {
			t.Errorf("Expected auth header not found")
		}
		if r.Header.Get("X-Request-ID") == "" {
			t.Errorf("Expected request ID header not found")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer server.Close()

	client := goclient.New().WithBaseURL(server.URL)

	// Chain multiple interceptors
	authInterceptor := interceptors.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
		req.Header.Set("X-Auth", "Bearer token")
		return req, nil
	})

	requestIDInterceptor := interceptors.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
		req.Header.Set("X-Request-ID", "test-123")
		return req, nil
	})

	chainedInterceptor := interceptors.ChainInterceptors(authInterceptor, requestIDInterceptor)
	client.WithInterceptor(chainedInterceptor)

	ctx := context.Background()
	resp, err := client.GET(ctx, "/test")

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Close()

	if !resp.IsSuccess() {
		t.Errorf("Expected successful response, got %s", resp.Status)
	}
}

// TestIntegrationAllPackagesWorkTogether tests that all packages can be used together
func TestIntegrationAllPackagesWorkTogether(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Verify interceptor worked
		if r.Header.Get("X-Integration-Test") != "true" {
			t.Errorf("Expected integration test header not found")
		}

		// Simulate failure for first few requests to test circuit breaker
		if requestCount <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
			return
		}

		// Return streaming data
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data1\ndata2\ndata3\n"))
	}))
	defer server.Close()

	// Create client with all features
	client := goclient.New().
		WithBaseURL(server.URL).
		WithTimeout(5 * time.Second)

	// Add interceptor
	client.WithInterceptor(interceptors.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
		req.Header.Set("X-Integration-Test", "true")
		return req, nil
	}))

	// Add response interceptor for logging
	client.WithResponseInterceptor(interceptors.ResponseInterceptorFunc(func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
		// Just log and pass through
		if err != nil {
			t.Logf("Request failed: %v", err)
		} else {
			t.Logf("Request succeeded: %s", resp.Status)
		}
		return resp, err
	}))

	// Create circuit breaker
	cb := circuitbreaker.NewCircuitBreaker(&circuitbreaker.CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  50 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	})

	stateChanges := 0
	cb.OnStateChange(func(from, to circuitbreaker.CircuitBreakerState) {
		stateChanges++
		t.Logf("Circuit breaker: %s -> %s", from, to)
	})

	ctx := context.Background()

	// Test circuit breaker behavior
	for i := 0; i < 3; i++ {
		err := cb.Execute(func() error {
			resp, err := client.GET(ctx, "/test")
			if err != nil {
				return err
			}
			defer resp.Close()
			if !resp.IsSuccess() {
				return &RequestError{Status: resp.Status, StatusCode: resp.StatusCode}
			}
			return nil
		})

		if i < 2 && err == nil {
			t.Errorf("Expected error on attempt %d", i+1)
		}
	}

	// Wait for recovery
	time.Sleep(60 * time.Millisecond)

	// Test successful streaming after recovery
	err := cb.Execute(func() error {
		resp, err := client.GET(ctx, "/test")
		if err != nil {
			return err
		}
		defer resp.Close()

		if !resp.IsSuccess() {
			return &RequestError{Status: resp.Status, StatusCode: resp.StatusCode}
		}

		// Test streaming
		lines := []string{}
		return streaming.StreamLines(resp, func(line []byte) error {
			lines = append(lines, string(line))
			return nil
		})
	})

	if err != nil {
		t.Errorf("Expected successful integration test, got: %v", err)
	}

	// Verify circuit breaker state changed
	if stateChanges == 0 {
		t.Errorf("Expected circuit breaker state changes, got none")
	}

	t.Log("✅ Integration test passed - all packages work together correctly!")
}
