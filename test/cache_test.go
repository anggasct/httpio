package test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/anggasct/httpio/middleware/cache"
)

type mockCache struct {
	data     map[string]*cache.CachedResponse
	setCalls int
	getCalls int
}

func newMockCache() *mockCache {
	return &mockCache{
		data:     make(map[string]*cache.CachedResponse),
		setCalls: 0,
		getCalls: 0,
	}
}

func (m *mockCache) Get(ctx context.Context, key string) (*cache.CachedResponse, bool) {
	m.getCalls++
	resp, exists := m.data[key]
	return resp, exists
}

func (m *mockCache) Set(ctx context.Context, key string, response *cache.CachedResponse) error {
	m.setCalls++
	m.data[key] = response
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache) Clear(ctx context.Context) error {
	m.data = make(map[string]*cache.CachedResponse)
	return nil
}

func (m *mockCache) Close() error {
	return nil
}

func TestCacheMiddleware(t *testing.T) {
	mockCache := newMockCache()
	config := cache.DefaultConfig()
	config.DefaultTTL = 5 * time.Minute

	cacheMiddleware := cache.NewMiddleware(mockCache, config)

	callCount := 0
	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		callCount++
		body := `{"message": "test response"}`
		resp := &http.Response{
			Status:        "200 OK",
			StatusCode:    200,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        make(http.Header),
			Body:          io.NopCloser(strings.NewReader(body)),
			ContentLength: int64(len(body)),
			Request:       req,
		}
		resp.Header.Set("Content-Type", "application/json")
		return resp, nil
	}

	handler := cacheMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	req.URL.Host = "example.com"
	req.URL.Scheme = "http"

	// Print debug info about the request
	t.Logf("Request URL: %s", req.URL.String())
	t.Logf("Request Method: %s", req.Method)

	// First call - should hit the handler
	resp1, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp1.StatusCode)
	}

	if callCount != 1 {
		t.Errorf("Expected handler to be called once, got %d", callCount)
	}

	// Wait a bit for the cache Set goroutine to complete
	time.Sleep(10 * time.Millisecond)

	// Check if anything was cached
	if len(mockCache.data) == 0 {
		t.Errorf("Expected something to be cached, but cache is empty. Set calls: %d, Get calls: %d", mockCache.setCalls, mockCache.getCalls)
	} else {
		t.Logf("Cache contains %d entries. Set calls: %d, Get calls: %d", len(mockCache.data), mockCache.setCalls, mockCache.getCalls)
		for key, _ := range mockCache.data {
			t.Logf("Cache key: %s", key)
		}
	}

	// Second call - should use cache
	resp2, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp2.StatusCode)
	}

	if callCount != 1 {
		t.Errorf("Expected handler to be called only once (cached), got %d", callCount)
	}
}

func TestCacheMiddlewareWithNonCacheableStatus(t *testing.T) {
	mockCache := newMockCache()
	config := cache.DefaultConfig()

	cacheMiddleware := cache.NewMiddleware(mockCache, config)

	callCount := 0
	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		callCount++
		body := `{"error": "server error"}`
		return &http.Response{
			StatusCode: 500,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil // Server error - not cacheable
	}

	handler := cacheMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	// First call
	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Second call - should not use cache for error responses
	_, err = handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected handler to be called twice (not cached), got %d", callCount)
	}
}

func TestCacheMiddlewareWithPOSTRequest(t *testing.T) {
	mockCache := newMockCache()
	config := cache.DefaultConfig()

	cacheMiddleware := cache.NewMiddleware(mockCache, config)

	callCount := 0
	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		callCount++
		body := `{"message": "post response"}`
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}

	handler := cacheMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("POST", "http://example.com/test", nil)

	// First call
	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Second call - POST requests typically shouldn't be cached
	_, err = handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected handler to be called twice (POST not cached), got %d", callCount)
	}
}

func TestMemoryCache(t *testing.T) {
	memCache := cache.NewMemoryCache(100)

	ctx := context.Background()

	// Test Set and Get
	cachedResp := &cache.CachedResponse{
		Response: &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
		},
		Body:         []byte("test response"),
		RequestURL:   "http://example.com/test",
		LastAccessed: time.Now(),
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}

	err := memCache.Set(ctx, "test-key", cachedResp)
	if err != nil {
		t.Fatalf("Expected no error setting cache, got %v", err)
	}

	retrieved, exists := memCache.Get(ctx, "test-key")
	if !exists {
		t.Fatal("Expected cached response to exist")
	}

	if retrieved.Response.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", retrieved.Response.StatusCode)
	}

	if string(retrieved.Body) != "test response" {
		t.Errorf("Expected body 'test response', got %s", string(retrieved.Body))
	}

	// Test Delete
	err = memCache.Delete(ctx, "test-key")
	if err != nil {
		t.Fatalf("Expected no error deleting from cache, got %v", err)
	}

	_, exists = memCache.Get(ctx, "test-key")
	if exists {
		t.Error("Expected cached response to be deleted")
	}

	// Test Clear
	memCache.Set(ctx, "key1", cachedResp)
	memCache.Set(ctx, "key2", cachedResp)

	err = memCache.Clear(ctx)
	if err != nil {
		t.Fatalf("Expected no error clearing cache, got %v", err)
	}

	_, exists1 := memCache.Get(ctx, "key1")
	_, exists2 := memCache.Get(ctx, "key2")

	if exists1 || exists2 {
		t.Error("Expected all cached responses to be cleared")
	}
}

func TestDiskCache(t *testing.T) {
	tempDir := t.TempDir()

	diskCache, err := cache.NewDiskCache(tempDir, 10) // 10MB max size
	if err != nil {
		t.Fatalf("Failed to create disk cache: %v", err)
	}
	defer diskCache.Close()

	ctx := context.Background()

	cachedResp := &cache.CachedResponse{
		Response: &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
		},
		Body:         []byte("disk test response"),
		RequestURL:   "http://example.com/disk-test",
		LastAccessed: time.Now(),
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}

	err = diskCache.Set(ctx, "disk-test-key", cachedResp)
	if err != nil {
		t.Fatalf("Expected no error setting disk cache, got %v", err)
	}

	retrieved, exists := diskCache.Get(ctx, "disk-test-key")
	if !exists {
		t.Fatal("Expected cached response to exist on disk")
	}

	if retrieved.Response.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", retrieved.Response.StatusCode)
	}

	if string(retrieved.Body) != "disk test response" {
		t.Errorf("Expected body 'disk test response', got %s", string(retrieved.Body))
	}
}

func TestCacheConfig(t *testing.T) {
	config := cache.DefaultConfig()

	if config.DefaultTTL != 10*time.Minute {
		t.Errorf("Expected default TTL to be 10 minutes, got %v", config.DefaultTTL)
	}

	if !config.Enabled {
		t.Error("Expected default cache to be enabled")
	}

	if config.KeyStrategy != cache.KeyByURLAndMethod {
		t.Errorf("Expected default key strategy to be KeyByURLAndMethod, got %v", config.KeyStrategy)
	}
}

func TestCacheKeyStrategies(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com/test?param=value", nil)
	req.Header.Set("Accept", "application/json")

	// Test URL and Method strategy
	urlMethodStrategy := cache.NewMethodURLKeyStrategy()
	key1 := urlMethodStrategy.GenerateKey(req)
	if key1 == "" {
		t.Error("Expected non-empty key for URL and method strategy")
	}

	// Test URL only strategy
	urlOnlyStrategy := cache.NewURLOnlyKeyStrategy()
	key2 := urlOnlyStrategy.GenerateKey(req)
	if key2 == "" {
		t.Error("Expected non-empty key for URL only strategy")
	}

	// Test full request strategy
	fullReqStrategy := cache.NewFullRequestKeyStrategy()
	key3 := fullReqStrategy.GenerateKey(req)
	if key3 == "" {
		t.Error("Expected non-empty key for full request strategy")
	}

	// Keys should be different for different strategies
	if key1 == key2 || key1 == key3 || key2 == key3 {
		t.Error("Expected different keys for different strategies")
	}
}
