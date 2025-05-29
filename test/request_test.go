package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/anggasct/httpio/internal/client"
	"github.com/anggasct/httpio/middleware"
)

type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func (m *mockHTTPClient) GetMiddlewares() []middleware.Middleware {
	return nil
}

func TestRequestWithHeader(t *testing.T) {
	mockClient := &mockHTTPClient{}
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com",
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  mockClient,
	}

	req = req.WithHeader("X-Test", "value")

	if req.Headers.Get("X-Test") != "value" {
		t.Errorf("Expected header X-Test: value, got %s", req.Headers.Get("X-Test"))
	}
}

func TestRequestWithHeaders(t *testing.T) {
	mockClient := &mockHTTPClient{}
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com",
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  mockClient,
	}

	headers := map[string]string{
		"X-Header-1": "value1",
		"X-Header-2": "value2",
	}

	req = req.WithHeaders(headers)

	for key, expectedValue := range headers {
		if req.Headers.Get(key) != expectedValue {
			t.Errorf("Expected header %s: %s, got %s", key, expectedValue, req.Headers.Get(key))
		}
	}
}

func TestRequestWithQuery(t *testing.T) {
	mockClient := &mockHTTPClient{}
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com",
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  mockClient,
	}

	req = req.WithQuery("param1", "value1")

	if req.Query.Get("param1") != "value1" {
		t.Errorf("Expected query param1: value1, got %s", req.Query.Get("param1"))
	}
}

func TestRequestWithQueryMap(t *testing.T) {
	mockClient := &mockHTTPClient{}
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com",
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  mockClient,
	}

	params := map[string]string{
		"param1": "value1",
		"param2": "value2",
	}

	req = req.WithQueryMap(params)

	for key, expectedValue := range params {
		if req.Query.Get(key) != expectedValue {
			t.Errorf("Expected query %s: %s, got %s", key, expectedValue, req.Query.Get(key))
		}
	}
}

func TestRequestWithBody(t *testing.T) {
	mockClient := &mockHTTPClient{}
	req := &client.Request{
		Method:  "POST",
		URL:     "https://example.com",
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  mockClient,
	}

	body := map[string]string{"key": "value"}
	req = req.WithBody(body)

	if req.Body == nil {
		t.Error("Expected body to be set")
	}
}

func TestRequestWithMiddleware(t *testing.T) {
	mockClient := &mockHTTPClient{}
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com",
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  mockClient,
	}

	// Create a simple test middleware
	testMiddleware := middleware.WrapMiddleware(func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Test-Middleware", "applied")
			return next(ctx, req)
		}
	})

	req = req.WithMiddleware(testMiddleware)

	// Test that middleware was added (we can't directly access the private field,
	// but we can test the behavior through integration)
	if req == nil {
		t.Error("Expected request to be returned")
	}
}

func TestRequestWithMultipleMiddlewares(t *testing.T) {
	mockClient := &mockHTTPClient{}
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com",
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  mockClient,
	}

	// Create test middlewares
	middleware1 := middleware.WrapMiddleware(func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Middleware-1", "applied")
			return next(ctx, req)
		}
	})

	middleware2 := middleware.WrapMiddleware(func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			req.Header.Set("X-Middleware-2", "applied")
			return next(ctx, req)
		}
	})

	req = req.WithMiddlewares(middleware1, middleware2)

	if req == nil {
		t.Error("Expected request to be returned")
	}
}

func TestRequestWithTimeout(t *testing.T) {
	mockClient := &mockHTTPClient{}
	req := &client.Request{
		Method:  "GET",
		URL:     "https://example.com",
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  mockClient,
	}

	req = req.WithTimeout(5)

	if req == nil {
		t.Error("Expected request to be returned with timeout")
	}
}

func TestRequestDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	// Create a real client for this test
	realClient := &http.Client{}
	httpClient := &httpClientWrapper{client: realClient}

	req := &client.Request{
		Method:  "GET",
		URL:     server.URL,
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  httpClient,
	}

	resp, err := req.Do(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// httpClientWrapper wraps http.Client to implement our HTTPClient interface
type httpClientWrapper struct {
	client *http.Client
}

func (w *httpClientWrapper) Do(req *http.Request) (*http.Response, error) {
	return w.client.Do(req)
}

func (w *httpClientWrapper) GetMiddlewares() []middleware.Middleware {
	return nil
}

func TestRequestStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("line1\nline2\nline3"))
	}))
	defer server.Close()

	realClient := &http.Client{}
	httpClient := &httpClientWrapper{client: realClient}

	req := &client.Request{
		Method:  "GET",
		URL:     server.URL,
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  httpClient,
	}

	var receivedData []byte
	err := req.Stream(context.Background(), func(chunk []byte) error {
		receivedData = append(receivedData, chunk...)
		return nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(string(receivedData), "line1") {
		t.Error("Expected received data to contain 'line1'")
	}
}

func TestRequestStreamLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("line1\nline2\nline3"))
	}))
	defer server.Close()

	realClient := &http.Client{}
	httpClient := &httpClientWrapper{client: realClient}

	req := &client.Request{
		Method:  "GET",
		URL:     server.URL,
		Headers: make(http.Header),
		Query:   make(url.Values),
		Client:  httpClient,
	}

	var lines []string
	err := req.StreamLines(context.Background(), func(line []byte) error {
		lines = append(lines, string(line))
		return nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(lines) == 0 {
		t.Error("Expected to receive some lines")
	}
}
