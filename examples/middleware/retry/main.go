package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/anggasct/httpio"
)

func main() {
	serverSuccessAttempt := 3
	currentAttempt := 0
	requestTimes := make([]time.Time, 0, 5)

	server := http.NewServeMux()
	server.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		currentAttempt++
		requestTimes = append(requestTimes, time.Now())

		if currentAttempt < serverSuccessAttempt {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "Service unavailable, try again later")
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Success after %d attempts!", currentAttempt)
	})

	go func() {
		log.Println("Starting test server on :8081")
		if err := http.ListenAndServe(":8081", server); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	retryConfig := &httpio.RetryConfig{
		MaxRetries: 5,
		RetryableStatusCodes: []int{
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  2 * time.Second,
		ErrorPredicate: func(err error) bool {
			return err != nil
		},
	}

	retryMiddleware := httpio.NewRetryMiddleware(retryConfig)

	httpClient := httpio.NewClient().
		WithMiddleware(retryMiddleware).
		WithTimeout(10 * time.Second)

	fmt.Println("Sending request with custom retry configuration...")
	start := time.Now()

	resp, err := httpClient.GET(context.Background(), "http://localhost:8081/test")
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	body, err := resp.String()
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	duration := time.Since(start)
	fmt.Printf("Request completed in %v\n", duration)
	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Body: %s\n", body)

	if len(requestTimes) > 1 {
		fmt.Println("\nRetry timing analysis:")
		for i := 1; i < len(requestTimes); i++ {
			delay := requestTimes[i].Sub(requestTimes[i-1])
			fmt.Printf("Retry %d delay: %v\n", i, delay)
		}
	}

	currentAttempt = 0
	requestTimes = requestTimes[:0]

	fmt.Println("\nUsing default retry configuration...")
	httpClientDefault := httpio.NewClient().
		WithMiddleware(httpio.NewRetryMiddleware(nil))

	startDefault := time.Now()
	respDefault, err := httpClientDefault.GET(context.Background(), "http://localhost:8081/test")
	if err != nil {
		log.Fatalf("Request with default config failed: %v", err)
	}

	bodyDefault, _ := respDefault.String()
	durationDefault := time.Since(startDefault)
	fmt.Printf("Default config response: %s (completed in %v)\n", bodyDefault, durationDefault)
}
