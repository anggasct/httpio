package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anggasct/httpio"
)

func TestNew(t *testing.T) {
	client := httpio.New()

	if client == nil {
		t.Fatal("Expected client to be created, got nil")
	}
}

func TestWithBaseURL(t *testing.T) {
	client := httpio.New()
	baseURL := "https://api.example.com"

	client = client.WithBaseURL(baseURL)

	// Create a test server to verify the base URL is used
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Override base URL with test server
	client = client.WithBaseURL(server.URL)

	resp, err := client.GET(context.Background(), "/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestWithHeader(t *testing.T) {
	client := httpio.New()
	headerKey := "X-Custom-Header"
	headerValue := "test-value"

	client = client.WithHeader(headerKey, headerValue)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerKey) != headerValue {
			t.Errorf("Expected header %s: %s, got %s", headerKey, headerValue, r.Header.Get(headerKey))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client = client.WithBaseURL(server.URL)

	_, err := client.GET(context.Background(), "/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestWithHeaders(t *testing.T) {
	client := httpio.New()
	headers := map[string]string{
		"X-Header-1": "value1",
		"X-Header-2": "value2",
	}

	client = client.WithHeaders(headers)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for key, value := range headers {
			if r.Header.Get(key) != value {
				t.Errorf("Expected header %s: %s, got %s", key, value, r.Header.Get(key))
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client = client.WithBaseURL(server.URL)

	_, err := client.GET(context.Background(), "/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestWithTimeout(t *testing.T) {
	client := httpio.New()
	timeout := 100 * time.Millisecond

	client = client.WithTimeout(timeout)

	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client = client.WithBaseURL(server.URL)

	_, err := client.GET(context.Background(), "/test")
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}
}

func TestWithConnectionPool(t *testing.T) {
	client := httpio.New()

	client = client.WithConnectionPool(10, 5, 5, 30*time.Second)

	if client == nil {
		t.Fatal("Expected client to be configured, got nil")
	}
}

func TestGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GET response"))
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	resp, err := client.GET(context.Background(), "/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := resp.String()
	if err != nil {
		t.Fatalf("Expected no error reading body, got %v", err)
	}

	if body != "GET response" {
		t.Errorf("Expected 'GET response', got %s", body)
	}
}

func TestPOST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("POST response"))
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	resp, err := client.POST(context.Background(), "/test", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}

func TestPUT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	resp, err := client.PUT(context.Background(), "/test", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestPATCH(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	resp, err := client.PATCH(context.Background(), "/test", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestDELETE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	resp, err := client.DELETE(context.Background(), "/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}
}

func TestHEAD(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("Expected HEAD method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	resp, err := client.HEAD(context.Background(), "/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestOPTIONS(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "OPTIONS" {
			t.Errorf("Expected OPTIONS method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	resp, err := client.OPTIONS(context.Background(), "/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestNewRequest(t *testing.T) {
	client := httpio.New().WithBaseURL("https://api.example.com")

	req := client.NewRequest("GET", "/test")

	if req == nil {
		t.Fatal("Expected request to be created, got nil")
	}

	if req.Method != "GET" {
		t.Errorf("Expected method GET, got %s", req.Method)
	}

	if req.URL != "https://api.example.com/test" {
		t.Errorf("Expected URL https://api.example.com/test, got %s", req.URL)
	}
}
