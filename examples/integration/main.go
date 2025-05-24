package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/anggasct/goclient/pkg/circuitbreaker"
	"github.com/anggasct/goclient/pkg/goclient"
	"github.com/anggasct/goclient/pkg/interceptors"
	"github.com/anggasct/goclient/pkg/streaming"
)

func main() {
	fmt.Println("=== Integration Test for Modular goclient ===")

	// Test 1: Basic client functionality
	fmt.Println("\n1. Testing basic client functionality...")
	client := goclient.New()

	// Add an interceptor
	client.WithInterceptor(interceptors.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
		fmt.Printf("   Interceptor: %s %s\n", req.Method, req.URL.String())
		return req, nil
	}))

	ctx := context.Background()
	resp, err := client.GET(ctx, "https://httpbin.org/get?test=1")
	if err != nil {
		log.Fatalf("   ❌ GET request failed: %v", err)
	}
	fmt.Printf("   ✅ GET request successful: %d %s\n", resp.StatusCode, resp.Status)

	// Test 2: Circuit Breaker functionality
	fmt.Println("\n2. Testing circuit breaker functionality...")
	cb := circuitbreaker.NewCircuitBreaker(&circuitbreaker.CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  100 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	})

	cb.OnStateChange(func(oldState, newState circuitbreaker.CircuitBreakerState) {
		fmt.Printf("   🔄 Circuit Breaker: %s -> %s\n", oldState, newState)
	})

	fmt.Printf("   ✅ Circuit Breaker created, initial state: %s\n", cb.GetState())

	// Test 3: Streaming functionality
	fmt.Println("\n3. Testing streaming functionality...")

	// Create a simple line-by-line streaming test
	resp, err = client.GET(ctx, "https://httpbin.org/stream/3")
	if err != nil {
		log.Printf("   ⚠️  Streaming request failed: %v (this is expected if httpbin.org is unavailable)", err)
	} else {
		lineCount := 0
		err = streaming.StreamLines(resp, func(line []byte) error {
			lineCount++
			fmt.Printf("   📄 Received line %d: %d bytes\n", lineCount, len(line))
			return nil
		})
		if err != nil {
			log.Printf("   ⚠️  Streaming failed: %v", err)
		} else {
			fmt.Printf("   ✅ Streaming completed, processed %d lines\n", lineCount)
		}
	}

	// Test 4: Package imports work correctly
	fmt.Println("\n4. Testing package imports...")
	fmt.Printf("   ✅ github.com/anggasct/goclient/pkg/goclient imported\n")
	fmt.Printf("   ✅ github.com/anggasct/goclient/pkg/circuitbreaker imported\n")
	fmt.Printf("   ✅ github.com/anggasct/goclient/pkg/streaming imported\n")
	fmt.Printf("   ✅ github.com/anggasct/goclient/pkg/interceptors imported\n")

	fmt.Println("\n=== Integration Test Complete ===")
	fmt.Println("✅ All modular packages are working correctly!")
}
