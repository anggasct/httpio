package goclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anggasct/goclient/pkg/interceptors"
)

func TestClientBasics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); ua != "test-agent" {
			t.Errorf("Expected User-Agent header 'test-agent', got '%s'", ua)
		}
		if q := r.URL.Query().Get("key"); q != "value" {
			t.Errorf("Expected query param 'key=value', got 'key=%s'", q)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "success"})
	}))
	defer server.Close()

	client := New().
		WithBaseURL(server.URL).
		WithHeader("User-Agent", "test-agent").
		WithTimeout(5 * time.Second)

	ctx := context.Background()
	resp, err := client.NewRequest("GET", "/test").
		WithQuery("key", "value").
		Do(ctx, client)

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Close()

	if !resp.IsSuccess() {
		t.Errorf("Expected successful response, got status: %s", resp.Status)
	}

	var result struct {
		Message string `json:"message"`
	}
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.Message != "success" {
		t.Errorf("Expected message 'success', got '%s'", result.Message)
	}
}

func TestContextAwareInterceptor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer token123" {
			t.Errorf("Expected Authorization header 'Bearer token123', got '%s'", auth)
		}
		if reqID := r.Header.Get("X-Request-ID"); reqID == "" {
			t.Errorf("Expected X-Request-ID header to be set")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	interceptor := interceptors.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
		req.Header.Set("Authorization", "Bearer token123")
		if reqID, ok := ctx.Value("request_id").(string); ok {
			req.Header.Set("X-Request-ID", reqID)
		} else {
			req.Header.Set("X-Request-ID", "default-id")
		}
		return req, nil
	})

	client := New().
		WithBaseURL(server.URL).
		WithInterceptor(interceptor)

	ctx := context.WithValue(context.Background(), "request_id", "test-123")

	resp, err := client.GET(ctx, "/secured")

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Close()

	if !resp.IsSuccess() {
		t.Errorf("Expected successful response, got status: %s", resp.Status)
	}
}

func TestResponseInterceptor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "original"})
	}))
	defer server.Close()

	respInterceptor := interceptors.ResponseInterceptorFunc(func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
		if err != nil {
			return nil, err
		}
		newResp := httptest.NewRecorder()
		newResp.Header().Set("Content-Type", "application/json")
		newResp.WriteHeader(http.StatusOK)
		json.NewEncoder(newResp).Encode(map[string]string{"message": "modified"})
		return newResp.Result(), nil
	})

	client := New().
		WithBaseURL(server.URL).
		WithResponseInterceptor(respInterceptor)

	ctx := context.Background()
	resp, err := client.GET(ctx, "/test")

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Close()

	var result struct {
		Message string `json:"message"`
	}
	if err := resp.JSON(&result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.Message != "modified" {
		t.Errorf("Expected modified message 'modified', got '%s'", result.Message)
	}
}

func TestContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New().
		WithBaseURL(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.GET(ctx, "/slow")

	if err == nil {
		t.Fatal("Expected error due to context cancellation but got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded error, got: %v", err)
	}
}

func TestRetry(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := New().
		WithBaseURL(server.URL).
		WithRetry(3, 10*time.Millisecond, nil)

	ctx := context.Background()
	resp, err := client.GET(ctx, "/retry-test")

	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Close()

	if !resp.IsSuccess() {
		t.Errorf("Expected successful response after retries, got status: %s", resp.Status)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryWithContextCancellation(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := New().
		WithBaseURL(server.URL).
		WithRetry(5, 50*time.Millisecond, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 125*time.Millisecond)
	defer cancel()

	_, err := client.GET(ctx, "/retry-cancel-test")

	if err == nil {
		t.Fatal("Expected context cancellation error, but got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}

	if attempts > 3 {
		t.Errorf("Expected at most 3 attempts due to context timeout, got %d", attempts)
	}
}
