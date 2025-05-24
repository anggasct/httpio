package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/anggasct/goclient/pkg/goclient"
	"github.com/anggasct/goclient/pkg/interceptors"
)

func main() {
	// Create a new client
	client := goclient.New().
		WithBaseURL("https://api.github.com").
		WithHeader("User-Agent", "goclient-example").
		WithTimeout(10*time.Second).
		WithRetry(3, 500*time.Millisecond, nil) // Use default retry policy

	// Add a context-aware logging interceptor
	client.WithInterceptor(interceptors.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
		// Extract request ID from context if available
		requestID := "unknown"
		if id, ok := ctx.Value("request_id").(string); ok {
			requestID = id
		}
		fmt.Printf("[%s] Making request to: %s %s\n", requestID, req.Method, req.URL.String())
		return req, nil
	}))

	// Add a response interceptor for logging
	client.WithResponseInterceptor(interceptors.ResponseInterceptorFunc(func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
		if err != nil {
			fmt.Printf("Request error: %v\n", err)
			return resp, err
		}

		fmt.Printf("Response status: %s\n", resp.Status)
		return resp, nil
	}))

	// Create a context with values and deadline
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "req-123")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Make a simple GET request
	resp, err := client.GET(ctx, "/users/octocat")
	if err != nil {
		log.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Close()

	// Check if the request was successful
	if !resp.IsSuccess() {
		log.Fatalf("Request failed with status: %s", resp.Status)
	}

	// Parse the response as JSON
	var user struct {
		Login       string `json:"login"`
		Name        string `json:"name"`
		Company     string `json:"company"`
		PublicRepos int    `json:"public_repos"`
	}
	if err := resp.JSON(&user); err != nil {
		log.Fatalf("Failed to parse response: %v", err)
	}

	fmt.Printf("User: %s (%s)\n", user.Login, user.Name)
	fmt.Printf("Company: %s\n", user.Company)
	fmt.Printf("Public Repositories: %d\n", user.PublicRepos)

	// Example of creating a more complex request with query parameters
	fmt.Println("\nFetching repositories...")

	// Create a new context with a different request ID
	repoCtx := context.WithValue(ctx, "request_id", "repo-fetch-456")

	repos, err := client.NewRequest("GET", "/users/octocat/repos").
		WithQuery("sort", "updated").
		WithQuery("per_page", "5").
		Do(repoCtx, client)

	if err != nil {
		log.Fatalf("Failed to fetch repos: %v", err)
	}
	defer repos.Close()

	// Parse the response as JSON
	var repositories []struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		StargazersCount int    `json:"stargazers_count"`
	}

	if err := repos.JSON(&repositories); err != nil {
		log.Fatalf("Failed to parse repos: %v", err)
	}

	// Display the repositories
	for _, repo := range repositories {
		fmt.Printf("Repo: %s - Stars: %d\n", repo.Name, repo.StargazersCount)
		fmt.Printf("Description: %s\n\n", repo.Description)
	}

	// Demonstrate context cancellation
	fmt.Println("Demonstrating context cancellation:")
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	// Cancel the context after 100ms
	go func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Cancelling request...")
		cancelFunc()
	}()

	// Try to make a request that will be cancelled
	_, err = client.WithRetry(5, 500*time.Millisecond, nil).GET(cancelCtx, "/rate_limit")

	if err != nil {
		fmt.Printf("Request was cancelled as expected: %v\n", err)
	}
}
