package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"time"

	"github.com/anggasct/goclient/pkg/circuitbreaker"
	"github.com/anggasct/goclient/pkg/goclient"
)

func main() {
	fmt.Println("Circuit Breaker Example")
	fmt.Println("======================")

	// Create a test server that fails sometimes
	var requestCount int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&requestCount, 1)

		// Fail the first 6 requests, then succeed
		if count <= 6 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Server: Request %d - FAILED (500)\n", count)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
		fmt.Printf("Server: Request %d - SUCCESS (200)\n", count)
	}))
	defer server.Close()

	// Create client with circuit breaker
	client := goclient.New().
		WithBaseURL(server.URL).
		WithCircuitBreaker(3, 2*time.Second, 2). // 3 failures threshold, 2s recovery, 2 half-open calls
		WithTimeout(1 * time.Second)

	// Set up circuit breaker state change callback
	client.OnCircuitBreakerStateChange(func(from, to circuitbreaker.CircuitBreakerState) {
		fmt.Printf("🔄 Circuit Breaker State Changed: %s -> %s\n", from, to)
	})

	ctx := context.Background()

	fmt.Println("\n1. Testing Circuit Breaker Behavior")
	fmt.Println("-----------------------------------")

	// Make requests to demonstrate circuit breaker behavior
	for i := 1; i <= 15; i++ {
		fmt.Printf("\nAttempt %d:\n", i)

		// Print current circuit breaker stats
		stats := client.GetCircuitBreakerStats()
		fmt.Printf("  Circuit Breaker State: %s\n", stats["state"])

		resp, err := client.GET(ctx, "/test")

		if err != nil {
			fmt.Printf("  ❌ Request failed: %v\n", err)
		} else {
			fmt.Printf("  ✅ Request succeeded: %s\n", resp.Status)
			resp.Close()
		}

		// Small delay between requests
		time.Sleep(200 * time.Millisecond)

		// Wait for recovery after circuit opens
		if i == 6 {
			fmt.Printf("\n⏳ Waiting for recovery timeout (2 seconds)...\n")
			time.Sleep(2200 * time.Millisecond)
		}
	}

	fmt.Println("\n2. Circuit Breaker Statistics")
	fmt.Println("-----------------------------")
	finalStats := client.GetCircuitBreakerStats()
	for key, value := range finalStats {
		fmt.Printf("  %s: %v\n", key, value)
	}

	fmt.Println("\n3. Testing Manual Reset")
	fmt.Println("----------------------")

	// Reset the server to fail again
	atomic.StoreInt64(&requestCount, 0)

	// Make some failing requests to open the circuit
	for i := 1; i <= 4; i++ {
		resp, err := client.GET(ctx, "/test")
		if err != nil {
			fmt.Printf("  Request %d failed: %v\n", i, err)
		} else {
			fmt.Printf("  Request %d succeeded: %s\n", i, resp.Status)
			resp.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("Circuit breaker state before reset: %s\n", client.GetCircuitBreakerStats()["state"])

	// Reset the circuit breaker
	client.ResetCircuitBreaker()
	fmt.Printf("Circuit breaker state after reset: %s\n", client.GetCircuitBreakerStats()["state"])

	fmt.Println("\n4. Demonstrating Half-Open State")
	fmt.Println("--------------------------------")

	// Create a new server that fails first request, then succeeds
	var halfOpenCount int64
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&halfOpenCount, 1)

		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Server: Half-open test request %d - FAILED\n", count)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Printf("Server: Half-open test request %d - SUCCESS\n", count)
	}))
	defer server2.Close()

	client2 := goclient.New().
		WithBaseURL(server2.URL).
		WithCircuitBreaker(2, 1*time.Second, 3).
		WithTimeout(1 * time.Second)

	client2.OnCircuitBreakerStateChange(func(from, to circuitbreaker.CircuitBreakerState) {
		fmt.Printf("🔄 Client2 Circuit Breaker: %s -> %s\n", from, to)
	})

	// Trigger circuit to open
	for i := 1; i <= 3; i++ {
		resp, err := client2.GET(ctx, "/test")
		if err != nil {
			fmt.Printf("  Opening circuit - Request %d failed: %v\n", i, err)
		} else {
			fmt.Printf("  Opening circuit - Request %d succeeded: %s\n", i, resp.Status)
			resp.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Wait for recovery timeout
	fmt.Println("⏳ Waiting for recovery timeout...")
	time.Sleep(1200 * time.Millisecond)

	// Make requests in half-open state
	for i := 1; i <= 5; i++ {
		fmt.Printf("Half-open request %d:\n", i)
		stats := client2.GetCircuitBreakerStats()
		fmt.Printf("  State: %s\n", stats["state"])

		resp, err := client2.GET(ctx, "/test")
		if err != nil {
			fmt.Printf("  ❌ Failed: %v\n", err)
		} else {
			fmt.Printf("  ✅ Succeeded: %s\n", resp.Status)
			resp.Close()
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n✅ Circuit Breaker example completed!")
}
