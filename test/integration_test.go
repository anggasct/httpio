package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/anggasct/httpio"
	"github.com/anggasct/httpio/middleware/cache"
	"github.com/anggasct/httpio/middleware/retry"
)

func TestIntegrationHTTPClientWithMiddlewares(t *testing.T) {
	// Create a test server that fails first time, succeeds second time
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "success",
			"method":  r.Method,
		})
	}))
	defer server.Close()

	// Create client with retry and cache middleware
	retryConfig := retry.DefaultConfig()
	retryConfig.MaxRetries = 2
	retryConfig.BaseDelay = 10 * time.Millisecond

	memCache := cache.NewMemoryCache(10)
	cacheConfig := cache.DefaultConfig()

	client := httpio.New().
		WithBaseURL(server.URL).
		WithTimeout(5*time.Second).
		WithHeader("User-Agent", "httpio-test").
		WithMiddleware(retry.New(retryConfig)).
		WithMiddleware(cache.NewMiddleware(memCache, cacheConfig))

	// First request should retry and then succeed
	resp, err := client.GET(context.Background(), "/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var data map[string]string
	err = resp.JSON(&data)
	if err != nil {
		t.Fatalf("Expected no error parsing JSON, got %v", err)
	}

	if data["message"] != "success" {
		t.Errorf("Expected message 'success', got %s", data["message"])
	}

	if data["method"] != "GET" {
		t.Errorf("Expected method 'GET', got %s", data["method"])
	}

	// Reset attempts for cache test
	attempts = 0

	// Wait for cache goroutine to complete
	time.Sleep(50 * time.Millisecond)

	// Second request should be served from cache (attempts should remain 0)
	resp2, err := client.GET(context.Background(), "/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp2.Close()

	if attempts != 0 {
		t.Errorf("Expected request to be served from cache (0 attempts), got %d attempts", attempts)
	}

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("Expected cached response status 200, got %d", resp2.StatusCode)
	}
}

func TestIntegrationPOSTWithMiddlewares(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"received": body,
			"status":   "created",
		})
	}))
	defer server.Close()

	client := httpio.New().
		WithBaseURL(server.URL).
		WithHeader("Content-Type", "application/json")

	requestData := map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	resp, err := client.POST(context.Background(), "/users", requestData)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var responseData map[string]interface{}
	err = resp.JSON(&responseData)
	if err != nil {
		t.Fatalf("Expected no error parsing JSON, got %v", err)
	}

	if responseData["status"] != "created" {
		t.Errorf("Expected status 'created', got %s", responseData["status"])
	}

	received, ok := responseData["received"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'received' to be a map")
	}

	if received["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %s", received["name"])
	}
}

func TestIntegrationStreamingResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Send multiple JSON objects
		objects := []map[string]interface{}{
			{"id": 1, "name": "Alice"},
			{"id": 2, "name": "Bob"},
			{"id": 3, "name": "Charlie"},
		}

		for _, obj := range objects {
			json.NewEncoder(w).Encode(obj)
		}
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	resp, err := client.GET(context.Background(), "/stream")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	type Person struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	var people []Person
	err = resp.StreamInto(func(person *Person) error {
		people = append(people, *person)
		return nil
	})

	if err != nil {
		t.Fatalf("Expected no error streaming, got %v", err)
	}

	if len(people) != 3 {
		t.Fatalf("Expected 3 people, got %d", len(people))
	}

	expectedNames := []string{"Alice", "Bob", "Charlie"}
	for i, person := range people {
		if person.ID != i+1 {
			t.Errorf("Expected person %d to have ID %d, got %d", i, i+1, person.ID)
		}
		if person.Name != expectedNames[i] {
			t.Errorf("Expected person %d to have name %s, got %s", i, expectedNames[i], person.Name)
		}
	}
}

func TestIntegrationSSEStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		events := []string{
			"event: start\ndata: Stream started\n\n",
			"event: message\ndata: Hello World\n\n",
			"event: end\ndata: Stream ended\n\n",
		}

		for _, event := range events {
			w.Write([]byte(event))
		}
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	resp, err := client.GET(context.Background(), "/events")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var events []httpio.SSEEvent
	handler := httpio.SSEEventHandlerFunc(func(event httpio.SSEEvent) error {
		events = append(events, event)
		return nil
	})

	err = resp.StreamSSE(handler)
	if err != nil {
		t.Fatalf("Expected no error streaming SSE, got %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("Expected 3 SSE events, got %d", len(events))
	}

	expectedEvents := []struct {
		event string
		data  string
	}{
		{"start", "Stream started"},
		{"message", "Hello World"},
		{"end", "Stream ended"},
	}

	for i, event := range events {
		if event.Event != expectedEvents[i].event {
			t.Errorf("Expected event %d type to be %s, got %s", i, expectedEvents[i].event, event.Event)
		}
		if event.Data != expectedEvents[i].data {
			t.Errorf("Expected event %d data to be %s, got %s", i, expectedEvents[i].data, event.Data)
		}
	}
}

func TestIntegrationErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	resp, err := client.GET(context.Background(), "/nonexistent")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	body, err := resp.String()
	if err != nil {
		t.Fatalf("Expected no error reading body, got %v", err)
	}

	if body != "Not found" {
		t.Errorf("Expected body 'Not found', got %s", body)
	}
}

func TestIntegrationRequestModification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back request details
		response := map[string]interface{}{
			"method":  r.Method,
			"url":     r.URL.String(),
			"headers": r.Header,
			"query":   r.URL.Query(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := httpio.New().WithBaseURL(server.URL)

	req := client.NewRequest("GET", "/test").
		WithHeader("X-Custom", "test-value").
		WithQuery("param1", "value1").
		WithQuery("param2", "value2")

	resp, err := req.Do(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Close()

	var data map[string]interface{}
	err = resp.JSON(&data)
	if err != nil {
		t.Fatalf("Expected no error parsing JSON, got %v", err)
	}

	if data["method"] != "GET" {
		t.Errorf("Expected method GET, got %s", data["method"])
	}

	headers, ok := data["headers"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected headers to be a map")
	}

	customHeader, exists := headers["X-Custom"]
	if !exists {
		t.Error("Expected X-Custom header to exist")
	} else {
		headerValues, ok := customHeader.([]interface{})
		if !ok || len(headerValues) == 0 {
			t.Error("Expected X-Custom header to have values")
		} else if headerValues[0] != "test-value" {
			t.Errorf("Expected X-Custom header value to be 'test-value', got %s", headerValues[0])
		}
	}
}
