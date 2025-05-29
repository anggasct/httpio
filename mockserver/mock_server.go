// Package main provides a mock HTTP server implementation for streaming data and JSON responses
package mockserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/anggasct/httpio"
)

// MockServer represents an HTTP mock server for streaming data and JSON responses
type MockServer struct {
	server       *http.Server
	routes       map[string]RouteConfig
	middleware   []MiddlewareFunc
	mutex        sync.RWMutex
	address      string
	stopChan     chan struct{}
	defaultDelay time.Duration
}

// RouteConfig contains configuration for a route
type RouteConfig struct {
	handler    http.HandlerFunc
	methods    []string
	middleware []MiddlewareFunc
}

// MiddlewareFunc represents a middleware function
type MiddlewareFunc func(http.Handler) http.Handler

// ResponseConfig contains configuration for JSON responses
type ResponseConfig struct {
	StatusCode int
	Headers    map[string]string
	Data       interface{}
	Delay      time.Duration
}

// NewMockServer creates a new mock server instance
func NewMockServer(address string) *MockServer {
	if address == "" {
		address = "localhost:8080"
	}

	return &MockServer{
		routes:       make(map[string]RouteConfig),
		middleware:   make([]MiddlewareFunc, 0),
		address:      address,
		stopChan:     make(chan struct{}),
		defaultDelay: 0,
	}
}

// AddMiddleware adds global middleware to the server
func (ms *MockServer) AddMiddleware(middleware MiddlewareFunc) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.middleware = append(ms.middleware, middleware)
}

// SetDefaultDelay sets the default delay for all responses
func (ms *MockServer) SetDefaultDelay(delay time.Duration) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.defaultDelay = delay
}

// AddRoute adds a route handler to the mock server
func (ms *MockServer) AddRoute(path string, handler http.HandlerFunc) {
	ms.AddRouteWithMethods(path, handler, []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"})
}

// AddRouteWithMethods adds a route handler with specific HTTP methods
func (ms *MockServer) AddRouteWithMethods(path string, handler http.HandlerFunc, methods []string) {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	ms.routes[path] = RouteConfig{
		handler: handler,
		methods: methods,
	}
}

// AddJSONRoute adds a route that returns JSON response from a struct
func (ms *MockServer) AddJSONRoute(path string, config ResponseConfig) {
	ms.AddRoute(path, func(w http.ResponseWriter, r *http.Request) {
		ms.handleJSONResponse(w, r, config)
	})
}

// AddJSONRouteWithMethods adds a JSON route with specific HTTP methods
func (ms *MockServer) AddJSONRouteWithMethods(path string, config ResponseConfig, methods []string) {
	ms.AddRouteWithMethods(path, func(w http.ResponseWriter, r *http.Request) {
		ms.handleJSONResponse(w, r, config)
	}, methods)
}

// handleJSONResponse handles JSON response with configuration
func (ms *MockServer) handleJSONResponse(w http.ResponseWriter, r *http.Request, config ResponseConfig) {
	delay := config.Delay
	if delay == 0 {
		delay = ms.defaultDelay
	}

	if delay > 0 {
		time.Sleep(delay)
	}

	w.Header().Set("Content-Type", "application/json")

	for key, value := range config.Headers {
		w.Header().Set(key, value)
	}

	statusCode := config.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	w.WriteHeader(statusCode)

	if config.Data != nil {
		if err := json.NewEncoder(w).Encode(config.Data); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// handler processes all incoming HTTP requests
func (ms *MockServer) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ms.mutex.RLock()
		routeConfig, exists := ms.routes[r.URL.Path]
		ms.mutex.RUnlock()

		if !exists {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Not Found"}`))
			return
		}

		methodAllowed := false
		for _, method := range routeConfig.methods {
			if method == r.Method {
				methodAllowed = true
				break
			}
		}

		if !methodAllowed {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte(`{"error": "Method Not Allowed"}`))
			return
		}

		handler := routeConfig.handler

		for _, middleware := range ms.middleware {
			handler = middleware(handler).ServeHTTP
		}

		for _, middleware := range routeConfig.middleware {
			handler = middleware(http.HandlerFunc(handler)).ServeHTTP
		}

		handler(w, r)
	})
}

// Start starts the mock server
func (ms *MockServer) Start() error {
	ms.server = &http.Server{
		Addr:    ms.address,
		Handler: ms.handler(),
	}

	go func() {
		_ = ms.server.ListenAndServe()
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Stop gracefully stops the mock server
func (ms *MockServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return ms.server.Shutdown(ctx)
}

// AddStreamingRoute adds a streaming route that sends data periodically
func (ms *MockServer) AddStreamingRoute(path string, streamFunc func(w http.ResponseWriter)) {
	ms.AddRoute(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if f, ok := w.(http.Flusher); ok {
			streamFunc(w)
			f.Flush()
		} else {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		}
	})
}

// AddStreamingJSONRoute adds a streaming route for JSON objects
func (ms *MockServer) AddStreamingJSONRoute(path string, items []interface{}, intervalMs int) {
	ms.AddRoute(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		encoder := json.NewEncoder(w)
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		for i, item := range items {
			if i > 0 {
				time.Sleep(time.Duration(intervalMs) * time.Millisecond)
			}

			if err := encoder.Encode(item); err != nil {
				return
			}

			flusher.Flush()
		}
	})
}

// AddSSERoute adds a Server-Sent Events (SSE) route
func (ms *MockServer) AddSSERoute(path string, events []httpio.SSEEvent, intervalMs int) {
	ms.AddRoute(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		for i, event := range events {
			if i > 0 {
				time.Sleep(time.Duration(intervalMs) * time.Millisecond)
			}

			data := fmt.Sprintf("id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Event, event.Data)

			if _, err := w.Write([]byte(data)); err != nil {
				return
			}

			flusher.Flush()
		}
	})
}

// AddNDJSONRoute adds a route that streams newline-delimited JSON
func (ms *MockServer) AddNDJSONRoute(path string, items []interface{}, intervalMs int) {
	ms.AddRoute(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		for i, item := range items {
			if i > 0 {
				time.Sleep(time.Duration(intervalMs) * time.Millisecond)
			}

			jsonData, err := json.Marshal(item)
			if err != nil {
				continue
			}

			if _, err := w.Write(append(jsonData, '\n')); err != nil {
				return
			}

			flusher.Flush()
		}
	})
}

// AddRESTRoute adds a REST route with CRUD operations for a resource
func (ms *MockServer) AddRESTRoute(basePath string, resource interface{}, storage map[string]interface{}) {
	ms.AddRouteWithMethods(basePath, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			ms.handleJSONResponse(w, r, ResponseConfig{Data: storage})
		case "POST":
			var data interface{}
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			id := fmt.Sprintf("%d", len(storage)+1)
			storage[id] = data
			ms.handleJSONResponse(w, r, ResponseConfig{
				StatusCode: http.StatusCreated,
				Data:       map[string]interface{}{"id": id, "data": data},
			})
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, []string{"GET", "POST"})

	ms.AddRouteWithMethods(basePath+"/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len(basePath)+1:]

		switch r.Method {
		case "GET":
			if data, exists := storage[id]; exists {
				ms.handleJSONResponse(w, r, ResponseConfig{Data: data})
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		case "PUT":
			var data interface{}
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			storage[id] = data
			ms.handleJSONResponse(w, r, ResponseConfig{Data: data})
		case "DELETE":
			if _, exists := storage[id]; exists {
				delete(storage, id)
				w.WriteHeader(http.StatusNoContent)
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}, []string{"GET", "PUT", "DELETE"})
}

// URL returns the full URL for a given path
func (ms *MockServer) URL(path string) string {
	return fmt.Sprintf("http://%s%s", ms.address, path)
}
