# Getting Started with goclient

goclient is a minimal and elegant HTTP client wrapper for Go that provides a clean API on top of the standard `net/http` library.

## Installation

```bash
go get github.com/anggasct/goclient
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/anggasct/goclient/pkg/goclient"
)

func main() {
    // Create a new client
    client := goclient.New().
        WithBaseURL("https://api.github.com").
        WithHeader("User-Agent", "my-app/1.0").
        WithTimeout(10 * time.Second)

    ctx := context.Background()

    // Make a GET request
    resp, err := client.GET(ctx, "/users/octocat")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Close()

    // Check if request was successful
    if !resp.IsSuccess() {
        log.Fatalf("Request failed: %s", resp.Status)
    }

    // Parse JSON response
    var user struct {
        Login string `json:"login"`
        Name  string `json:"name"`
    }
    if err := resp.JSON(&user); err != nil {
        log.Fatal(err)
    }

    fmt.Printf("User: %s (%s)\n", user.Name, user.Login)
}
```

### Building Requests

```go
// Simple GET with query parameters
resp, err := client.NewRequest("GET", "/search/repositories").
    WithQuery("q", "golang").
    WithQuery("sort", "stars").
    WithQuery("order", "desc").
    Do(ctx, client)

// POST with JSON body
data := map[string]interface{}{
    "name":        "my-repo",
    "description": "My awesome repository",
    "private":     false,
}

resp, err := client.NewRequest("POST", "/user/repos").
    WithHeader("Accept", "application/vnd.github.v3+json").
    WithBody(data).
    Do(ctx, client)
```

### Handling Responses

```go
// Get response as string
content, err := resp.String()
if err != nil {
    log.Fatal(err)
}
fmt.Println(content)

// Get response as bytes
data, err := resp.Bytes()
if err != nil {
    log.Fatal(err)
}

// Parse JSON into struct
var result MyStruct
if err := resp.JSON(&result); err != nil {
    log.Fatal(err)
}

// Check response status
if resp.IsSuccess() {
    fmt.Println("Request successful!")
} else if resp.IsClientError() {
    fmt.Println("Client error (4xx)")
} else if resp.IsServerError() {
    fmt.Println("Server error (5xx)")
}
```

## Configuration Options

### Timeouts

```go
client := goclient.New().
    WithTimeout(30 * time.Second)  // Overall request timeout
```

### Headers

```go
client := goclient.New().
    WithHeader("Authorization", "Bearer your-token").
    WithHeader("Content-Type", "application/json").
    WithHeaders(map[string]string{
        "X-API-Version": "v1",
        "X-Client-ID":   "my-client",
    })
```

### Base URL

```go
client := goclient.New().
    WithBaseURL("https://api.example.com/v1")

// Now you can use relative paths
resp, err := client.GET(ctx, "/users")  // Requests to https://api.example.com/v1/users
```

## Error Handling

```go
resp, err := client.GET(ctx, "/api/data")
if err != nil {
    // Handle request errors (network, timeout, etc.)
    log.Printf("Request failed: %v", err)
    return
}
defer resp.Close()

// Handle HTTP errors
if !resp.IsSuccess() {
    switch {
    case resp.StatusCode == 404:
        fmt.Println("Resource not found")
    case resp.IsClientError():
        fmt.Printf("Client error: %s", resp.Status)
    case resp.IsServerError():
        fmt.Printf("Server error: %s", resp.Status)
    }
    return
}
```

## Context Usage

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.GET(ctx, "/slow-endpoint")

// With cancellation
ctx, cancel := context.WithCancel(context.Background())

// Cancel after 2 seconds
go func() {
    time.Sleep(2 * time.Second)
    cancel()
}()

resp, err := client.GET(ctx, "/endpoint")
if err != nil {
    if err == context.Canceled {
        fmt.Println("Request was cancelled")
    }
}
```

## Next Steps

- [Advanced Usage](advanced-usage.md) - Learn about interceptors, retry, and more
- [Circuit Breaker](circuit-breaker.md) - Implement resilience patterns
- [Streaming](streaming.md) - Handle streaming responses and SSE
- [Examples](../examples/) - Check out working examples

## Common Patterns

### API Client Wrapper

```go
type GitHubClient struct {
    client *goclient.Client
}

func NewGitHubClient(token string) *GitHubClient {
    client := goclient.New().
        WithBaseURL("https://api.github.com").
        WithHeader("Authorization", "token "+token).
        WithHeader("Accept", "application/vnd.github.v3+json").
        WithTimeout(30 * time.Second)

    return &GitHubClient{client: client}
}

func (g *GitHubClient) GetUser(ctx context.Context, username string) (*User, error) {
    resp, err := g.client.GET(ctx, "/users/"+username)
    if err != nil {
        return nil, err
    }
    defer resp.Close()

    if !resp.IsSuccess() {
        return nil, fmt.Errorf("failed to get user: %s", resp.Status)
    }

    var user User
    if err := resp.JSON(&user); err != nil {
        return nil, err
    }

    return &user, nil
}
```

### Configuration from Environment

```go
func NewClientFromEnv() *goclient.Client {
    baseURL := os.Getenv("API_BASE_URL")
    if baseURL == "" {
        baseURL = "https://api.example.com"
    }

    timeout := 30 * time.Second
    if t := os.Getenv("API_TIMEOUT"); t != "" {
        if parsed, err := time.ParseDuration(t); err == nil {
            timeout = parsed
        }
    }

    return goclient.New().
        WithBaseURL(baseURL).
        WithHeader("Authorization", "Bearer "+os.Getenv("API_TOKEN")).
        WithTimeout(timeout)
}
```
