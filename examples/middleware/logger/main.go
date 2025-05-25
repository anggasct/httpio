package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/anggasct/httpio/internal/client"
	"github.com/anggasct/httpio/internal/middleware/logger"
)

func main() {
	// Start a test HTTP server
	serverURL, server := startTestServer()
	defer server.Shutdown(context.Background())

	fmt.Println("=== Starting Logger Middleware Example ===")

	// Create a client with default logger (INFO level, text format)
	fmt.Println("\n--- Example 1: Basic Logger (INFO Level) ---")
	basicClient := client.New().
		WithBaseURL(serverURL).
		WithMiddleware(logger.New(nil))

	makeRequests(basicClient)

	// Create a client with DEBUG level logger
	fmt.Println("\n--- Example 2: DEBUG Level Logger (with headers) ---")
	debugConfig := &logger.Config{
		Level: logger.LevelDebug,
	}
	debugClient := client.New().
		WithBaseURL(serverURL).
		WithMiddleware(logger.New(debugConfig))

	makeRequests(debugClient)

	// Create a client with JSON output format
	fmt.Println("\n--- Example 3: JSON Format Logger ---")
	jsonConfig := &logger.Config{
		Format: logger.FormatJSON,
		Level:  logger.LevelTrace, // Set the middleware level to TRACE to see bodies
		// Also set the logger's format
		Logger: &logger.StandardLogger{
			Level:  logger.LevelTrace,
			Format: logger.FormatJSON,
		},
		SensitiveFields: []string{"name"},
	}
	jsonClient := client.New().
		WithBaseURL(serverURL).
		WithMiddleware(logger.New(jsonConfig))

	makeRequests(jsonClient)

	// Create a client with TRACE level and custom configuration
	fmt.Println("\n--- Example 4: TRACE Level Logger (with custom config) ---")
	config := &logger.Config{
		Level:            logger.LevelTrace,
		SensitiveHeaders: []string{"Authorization", "X-API-Key", "Cookie"},
		SensitiveFields:  []string{"password", "token", "secret"},
		SkipPaths:        []string{"/health"},
	}

	traceClient := client.New().
		WithBaseURL(serverURL).
		WithMiddleware(logger.New(config))

	makeRequests(traceClient)

	// Example with delayed API request
	fmt.Println("\n--- Example 5: Logging Delayed API Responses ---")
	delayConfig := &logger.Config{
		Level:  logger.LevelDebug,
		Format: logger.FormatJSON, // Use JSON format to clearly show the latency
		Logger: &logger.StandardLogger{
			Level:  logger.LevelDebug,
			Format: logger.FormatJSON,
		},
	}
	delayClient := client.New().
		WithBaseURL(serverURL).
		WithMiddleware(logger.New(delayConfig))

	// Make requests to the delayed API endpoint
	makeDelayedRequests(delayClient)

	fmt.Println("\n=== Logger Middleware Example Completed ===")
}

func makeRequests(client *client.Client) {
	ctx := context.Background()

	// Make a successful GET request
	resp, err := client.GET(ctx, "/api/users")
	if err != nil {
		log.Printf("Error making GET request: %v", err)
	} else {
		resp.Body.Close()
	}

	// Make a POST request with JSON body
	data := map[string]interface{}{
		"name":     "John Doe",
		"email":    "john@example.com",
		"password": "secret123", // This will be redacted in logs with proper config
	}
	resp, err = client.POST(ctx, "/api/users", data)
	if err != nil {
		log.Printf("Error making POST request: %v", err)
	} else {
		resp.Body.Close()
	}

	// Make a request to a path that might be skipped
	resp, err = client.GET(ctx, "/health")
	if err != nil {
		log.Printf("Error making health request: %v", err)
	} else {
		resp.Body.Close()
	}

	// Make a request that will result in an error (404)
	resp, err = client.GET(ctx, "/not-found")
	if err != nil {
		log.Printf("Error making not-found request: %v", err)
	} else {
		resp.Body.Close()
	}
}

// makeDelayedRequests makes requests to the delayed API endpoint with different delays
func makeDelayedRequests(client *client.Client) {
	ctx := context.Background()

	// Make requests with different delay values
	delays := []string{"1s", "2s", "500ms"}

	for _, delay := range delays {
		fmt.Printf("\nMaking request to delayed API endpoint with delay=%s...\n", delay)

		// Call the delayed endpoint and measure time
		startTime := time.Now()

		// Build request with query parameter
		resp, err := client.NewRequest("GET", "/api/slow-operation").
			WithQuery("delay", delay).
			Do(ctx)

		duration := time.Since(startTime)

		if err != nil {
			log.Printf("Error making request to delayed endpoint: %v", err)
			continue
		}

		fmt.Printf("Request completed in %.2f seconds (client-side measurement)\n", duration.Seconds())

		// Print the response headers
		fmt.Println("Response headers:")
		fmt.Printf("  X-Processing-Time: %s\n", resp.Header.Get("X-Processing-Time"))
		fmt.Printf("  X-Server-Time: %s\n", resp.Header.Get("X-Server-Time"))

		// Print the response body
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			fmt.Printf("Response body: %v\n", result)
		}

		resp.Body.Close()
	}

	// Let's also test a longer delay that might push timeout limits
	fmt.Println("\nTesting longer delay (5s) with context timeout...")
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	startTime := time.Now()
	_, err := client.NewRequest("GET", "/api/slow-operation").
		WithQuery("delay", "5s").
		Do(ctxWithTimeout)

	duration := time.Since(startTime)

	if err != nil {
		fmt.Printf("Request failed after %.2f seconds: %v\n", duration.Seconds(), err)
	} else {
		fmt.Printf("Request unexpectedly succeeded in %.2f seconds\n", duration.Seconds())
	}
}

func startTestServer() (string, *http.Server) {
	mux := http.NewServeMux()

	// GET endpoint for users
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"users": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`))
		} else if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": 3, "status": "created"}`))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "healthy"}`))
	})

	// Delayed API endpoint - simulates slow response
	mux.HandleFunc("/api/slow-operation", func(w http.ResponseWriter, r *http.Request) {
		// Extract the delay parameter from query or use default
		delayStr := r.URL.Query().Get("delay")
		var delayDuration time.Duration = 3 * time.Second // Default delay

		if delayStr != "" {
			// Try to parse a custom delay
			if customDelay, err := time.ParseDuration(delayStr); err == nil {
				delayDuration = customDelay
			}
		}

		// Log the delay on the server side
		log.Printf("API: Processing request with %s delay", delayDuration)

		// Simulate processing delay
		time.Sleep(delayDuration)

		// Set response headers and add timestamp
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Processing-Time", delayDuration.String())
		w.Header().Set("X-Server-Time", time.Now().Format(time.RFC3339))

		// Return response with timestamp and delay info
		response := map[string]interface{}{
			"status":              "completed",
			"message":             "Operation completed after delay",
			"delay_seconds":       delayDuration.Seconds(),
			"server_timestamp":    time.Now().Format(time.RFC3339),
			"processing_complete": true,
		}

		responseJSON, _ := json.Marshal(response)
		w.Write(responseJSON)
	})

	// Any other path returns 404
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Not Found"}`))
			return
		}
		w.Write([]byte("Test Server"))
	})

	// Start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("Starting test server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Return the server URL and the server instance
	return "http://localhost:8080", server
}
