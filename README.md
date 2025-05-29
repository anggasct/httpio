# httpio - HTTP Client Library for Go

`httpio` is a flexible HTTP client library for Go that provides enhanced functionality through middleware support, streaming capabilities, and a fluent API. It's designed to simplify making HTTP requests in Go while providing robust features for advanced use cases.

![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 📚 Features

- ✅ **Fluent, chainable API** for clean and readable HTTP requests
- ✅ **Middleware architecture** for customizing request/response handling
- ✅ **Streaming support** for processing large responses efficiently
- ✅ **Server-Sent Events (SSE)** support with multiple handler patterns
- ✅ **Built-in middleware**:
  - Circuit breaker for resilience
  - Logging with configurable levels and formats
  - OAuth authentication
  - Automatic retry with exponential backoff
  - Response caching with TTL and pattern matching
- ✅ **Connection pooling** with configurable settings
- ✅ **Timeouts** and cancellation support via `context.Context`

## 📦 Installation

```bash
go get -u github.com/anggasct/httpio
```

## 🚀 Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/anggasct/httpio"
)

func main() {
    // Create a new client
    client := httpio.New().
        WithBaseURL("https://api.example.com").
        WithTimeout(10 * time.Second).
        WithHeader("Content-Type", "application/json")
    
    ctx := context.Background()
    
    // Simple GET request
    resp, err := client.GET(ctx, "/users/123")
    if err != nil {
        panic(err)
    }
    defer resp.Close()
    
    // Check if the request was successful
    if !resp.IsSuccess() {
        fmt.Printf("Request failed: %s\n", resp.Status)
        return
    }
    
    // Parse the response body as JSON
    var user struct {
        ID   int    `json:"id"`
        Name string `json:"name"`
    }
    
    if err := resp.JSON(&user); err != nil {
        fmt.Printf("Failed to parse JSON: %v\n", err)
        return
    }
    
    fmt.Printf("User: %s (ID: %d)\n", user.Name, user.ID)
}
```

## 🔍 Key Concepts

### Making HTTP Requests

The library supports all standard HTTP methods with both simple and advanced usage patterns:

```go
// Simple GET
resp, err := client.GET(ctx, "/api/resource")

// Simple POST with JSON body
resp, err := client.POST(ctx, "/api/resource", map[string]interface{}{
    "name": "New Resource",
    "active": true,
})

// Advanced request with chaining
resp, err := client.NewRequest("GET", "/api/resources").
    WithQuery("limit", "10").
    WithQuery("sort", "name").
    WithHeader("X-Custom-Header", "value").
    Do(ctx)
```

### Using Middleware

Middleware can be added to clients to customize request processing:

```go
// Add logger middleware
loggerConfig := &httpio.LoggerConfig{
    Level: httpio.LevelDebug,
}
client := httpio.New().
    WithMiddleware(httpio.NewLoggerMiddleware(loggerConfig))

// Add retry middleware
retryConfig := &httpio.RetryConfig{
    MaxRetries: 3,
    RetryDelay: 100 * time.Millisecond,
}
client = client.WithMiddleware(httpio.NewRetryMiddleware(retryConfig))

// Circuit breaker middleware
cbConfig := &httpio.CircuitBreakerConfig{
    Threshold:   5,
    ResetTime:   10 * time.Second,
}
client = client.WithMiddleware(httpio.NewCircuitBreakerMiddleware(cbConfig))

// OAuth middleware
oauthConfig := &httpio.OAuthConfig{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    TokenURL:     "https://oauth.example.com/token",
}
client = client.WithMiddleware(httpio.NewOAuthMiddleware(oauthConfig))
```

### Connection Pooling

Configure connection pool settings for optimal performance:

```go
client := httpio.New().
    WithConnectionPool(
        100,                    // maxIdleConns
        30,                     // maxConnsPerHost
        10,                     // maxIdleConnsPerHost
        90 * time.Second,       // idleConnTimeout
    )
```

### Handling Streaming Responses

The library provides several methods for processing streaming data:

```go
// Basic byte stream processing
err := client.NewRequest("GET", "/api/stream").
    Stream(ctx, func(chunk []byte) error {
        fmt.Printf("Received chunk of size %d bytes\n", len(chunk))
        return nil
    })

// Process stream line by line
err := client.NewRequest("GET", "/api/lines").
    StreamLines(ctx, func(line []byte) error {
        fmt.Printf("Line: %s\n", string(line))
        return nil
    })

// Process stream as JSON objects
err := client.NewRequest("GET", "/api/json-stream").
    StreamJSON(ctx, func(jsonMsg json.RawMessage) error {
        fmt.Printf("JSON object: %s\n", string(jsonMsg))
        return nil
    })

// Process typed objects
type Event struct {
    ID   string `json:"id"`
    Type string `json:"type"`
    Data string `json:"data"`
}

err := client.NewRequest("GET", "/api/events").
    StreamInto(ctx, func(event Event) error {
        fmt.Printf("Event ID: %s, Type: %s\n", event.ID, event.Type)
        return nil
    })

// Stream with options
err := client.NewRequest("GET", "/api/large-stream").
    Stream(ctx, func(chunk []byte) error {
        // Process chunk
        return nil
    }, httpio.WithBufferSize(8192), httpio.WithContentType("application/json"))
```

### Server-Sent Events Support

The library has flexible support for Server-Sent Events (SSE) with multiple handler options:

#### Simple Function Handler (Recommended)

```go
// Most straightforward approach
err := client.NewRequest("GET", "/api/events").
    StreamSSE(ctx, httpio.SSEEventHandlerFunc(func(event httpio.SSEEvent) error {
        fmt.Printf("Event: %s, Data: %s\n", event.Event, event.Data)
        return nil
    }))
```

#### Struct-based Handler

```go
type EventProcessor struct{}

func (p *EventProcessor) OnEvent(event httpio.SSEEvent) error {
    fmt.Printf("Event: %s, Data: %s\n", event.Event, event.Data)
    return nil
}

processor := &EventProcessor{}
err := client.NewRequest("GET", "/api/events").StreamSSE(ctx, processor)
```

#### Handler with Lifecycle Management

```go
handler := &httpio.SSEEventFullHandlerFunc{
    OnEventFunc: func(event httpio.SSEEvent) error {
        fmt.Printf("Event: %s\n", event.Data)
        return nil
    },
    OnOpenFunc: func() error {
        fmt.Println("Connection opened")
        return nil
    },
    OnCloseFunc: func() error {
        fmt.Println("Connection closed")
        return nil
    },
}

err := client.NewRequest("GET", "/api/events").StreamSSE(ctx, handler)
```

> **Note**: Only `OnEvent()` is required. `OnOpen()` and `OnClose()` are optional lifecycle methods that provide additional control over connection management.

## 📖 API Reference

### Client Methods

```go
// Create a new client
client := httpio.New()

// Configure the client
client.WithBaseURL("https://api.example.com").
    WithTimeout(30 * time.Second).
    WithHeader("User-Agent", "MyApp/1.0").
    WithHeaders(map[string]string{
        "Accept": "application/json",
        "Authorization": "Bearer token",
    })

// HTTP methods
resp, err := client.GET(ctx, "/users")
resp, err := client.POST(ctx, "/users", userData)
resp, err := client.PUT(ctx, "/users/123", updatedData)
resp, err := client.PATCH(ctx, "/users/123", partialData)
resp, err := client.DELETE(ctx, "/users/123")
resp, err := client.HEAD(ctx, "/users/123")
resp, err := client.OPTIONS(ctx, "/users")

// Advanced request building
req := client.NewRequest("GET", "/api/resource").
    WithQuery("limit", "10").
    WithQuery("sort", "name").
    WithHeader("X-Custom-Header", "value").
    WithBody(requestData)

resp, err := req.Do(ctx)
```

### Response Methods

```go
// Status checking
if resp.IsSuccess() { /* 2xx */ }
if resp.IsRedirect() { /* 3xx */ }
if resp.IsClientError() { /* 4xx */ }
if resp.IsServerError() { /* 5xx */ }
if resp.IsError() { /* Any error response */ }

// Body reading
bytes, err := resp.Bytes()
text, err := resp.String()
err := resp.JSON(&target)

// Resource management
resp.Close()
resp.Consume() // Read and discard body
```

### Middleware Configuration

```go
// Logger levels: LevelNone, LevelError, LevelInfo, LevelDebug, LevelTrace
logger := httpio.NewLoggerMiddleware(&httpio.LoggerConfig{
    Level: httpio.LevelDebug,
})

// Retry with exponential backoff
retry := httpio.NewRetryMiddleware(&httpio.RetryConfig{
    MaxRetries: 3,
    RetryDelay: 100 * time.Millisecond,
})

// Circuit breaker
cb := httpio.NewCircuitBreakerMiddleware(&httpio.CircuitBreakerConfig{
    Threshold: 5,
    ResetTime: 10 * time.Second,
})

// OAuth 2.0
oauth := httpio.NewOAuthMiddleware(&httpio.OAuthConfig{
    ClientID:     "client-id",
    ClientSecret: "client-secret",
    TokenURL:     "https://oauth.example.com/token",
})

client.WithMiddleware(logger).
    WithMiddleware(retry).
    WithMiddleware(cb).
    WithMiddleware(oauth)
```

## 🚀 Advanced Usage

### Custom Middleware

```go
func customMiddleware(next httpio.MiddlewareFunc) httpio.MiddlewareFunc {
    return func(ctx context.Context, req *http.Request) (*http.Response, error) {
        // Pre-request logic
        start := time.Now()
        
        // Call next middleware/handler
        resp, err := next(ctx, req)
        
        // Post-request logic
        duration := time.Since(start)
        fmt.Printf("Request took %v\n", duration)
        
        return resp, err
    }
}

client := httpio.New().WithMiddleware(customMiddleware)
```

### Error Handling

```go
resp, err := client.GET(ctx, "/api/resource")
if err != nil {
    // Handle network errors, timeouts, etc.
    return err
}
defer resp.Close()

if !resp.IsSuccess() {
    // Handle HTTP error responses
    body, _ := resp.String()
    return fmt.Errorf("API error %d: %s", resp.StatusCode, body)
}

// Process successful response
var result APIResponse
if err := resp.JSON(&result); err != nil {
    return fmt.Errorf("failed to parse response: %w", err)
}
```

## 🛠️ Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.


### Reporting Issues

When reporting issues, please include:

- Go version
- Operating system
- Minimal code example that reproduces the issue
- Expected vs actual behavior

## 📄 License

This project is licensed under the [MIT License](LICENSE).
