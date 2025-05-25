package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/anggasct/httpio/internal/client"
	"github.com/anggasct/httpio/internal/middleware/circuitbreaker"
)

func createMockServer() *httptest.Server {
	var requestCount = 0
	var mutex sync.Mutex

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		requestCount++
		currentCount := requestCount
		mutex.Unlock()

		if r.URL.Path == "/reset" {
			mutex.Lock()
			requestCount = 0
			mutex.Unlock()
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Counter reset")
			return
		}

		if r.URL.Path == "/healthy" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Healthy endpoint - always works")
			return
		}

		if r.URL.Path == "/failing" {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "This endpoint always fails")
			return
		}

		// Main endpoint: Send 3 failures in a row followed by a success
		// This pattern is important for demonstrating the circuit breaker behavior.
		// The circuit breaker requires CONSECUTIVE failures to trip, not just a threshold
		// of failures over time.
		if currentCount%4 == 1 || currentCount%4 == 2 || currentCount%4 == 3 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Server error (request #%d)", currentCount)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Success response (request #%d)", currentCount)
		}
	}))
}

func main() {
	server := createMockServer()
	defer server.Close()

	cbConfig := &circuitbreaker.Config{
		FailureThreshold: 3,
		RecoveryTimeout:  5 * time.Second,
		HalfOpenMaxCalls: 2,
		OnStateChange: func(from, to circuitbreaker.CircuitBreakerState) {
			fmt.Printf("\n[EVENT] Circuit state changed: %s -> %s\n\n", from, to)
		},
		ErrorPredicate: func(resp *http.Response, err error) bool {
			if err != nil {
				return true
			}
			if resp != nil && resp.StatusCode >= 500 {
				return true
			}
			return false
		},
	}

	cbMiddleware := circuitbreaker.New(cbConfig)
	httpClient := client.New().
		WithBaseURL(server.URL).
		WithMiddleware(cbMiddleware)

	ctx := context.Background()
	cb := cbMiddleware.GetCircuitBreaker()

	for i := 1; i <= 10; i++ {
		resp, err := httpClient.GET(ctx, "/")
		if err != nil {
			fmt.Printf("Request %d: Error: %v (Circuit State: %s)\n", i, err, cb.GetState())
			continue
		}
		defer resp.Close()

		body, _ := resp.String()
		fmt.Printf("Request %d: Status %d - %s (Circuit State: %s)\n",
			i, resp.StatusCode, body, cb.GetState())

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\nWaiting for recovery timeout...")
	time.Sleep(5 * time.Second)

	for i := 1; i <= 3; i++ {
		resp, err := httpClient.GET(ctx, "/healthy")
		if err != nil {
			fmt.Printf("Recovery request %d: Error: %v (Circuit State: %s)\n",
				i, err, cb.GetState())
		} else {
			defer resp.Close()
			body, _ := resp.String()
			fmt.Printf("Recovery request %d: Status %d - %s (Circuit State: %s)\n",
				i, resp.StatusCode, body, cb.GetState())
		}
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("\nFinal circuit state: %s\n", cb.GetState())
}
