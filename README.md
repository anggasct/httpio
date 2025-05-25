# httpio - HTTP Client Library for Go

`httpio` is a flexible HTTP client library for Go that provides enhanced functionality through middleware support, streaming capabilities, and a fluent API. It's designed to simplify making HTTP requests in Go while providing robust features for advanced use cases.

![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)

## 📚 Features

- ✅ **Fluent, chainable API** for clean and readable HTTP requests
- ✅ **Middleware architecture** for customizing request/response handling
- ✅ **Streaming support** for processing large responses efficiently
- ✅ **Server-Sent Events (SSE)** support
- ✅ **Built-in middleware**:
  - Circuit breaker for resilience
  - Logging with configurable levels
  - OAuth authentication
  - Automatic retry with backoff
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
    client := httpio.NewClient().
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
    Level: logger.LevelDebug,
}
client := httpio.NewClient().
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
    FailureRate: 0.5,
    ResetTime:   10 * time.Second,
}
client = client.WithMiddleware(httpio.NewCircuitBreakerMiddleware(cbConfig))
```

### Handling Streaming Responses

The library provides several methods for processing streaming data:

```go
// Basic byte stream processing
err := httpio.GetStream(client, ctx, "/api/stream", func(chunk []byte) error {
    fmt.Printf("Received chunk of size %d bytes\n", len(chunk))
    return nil
})

// Process stream line by line
err := httpio.GetStreamLines(client, ctx, "/api/lines", func(line []byte) error {
    fmt.Printf("Line: %s\n", string(line))
    return nil
})

// Process stream as JSON objects
err := httpio.GetStreamJSON(client, ctx, "/api/json-stream", func(jsonMsg json.RawMessage) error {
    fmt.Printf("JSON object: %s\n", string(jsonMsg))
    return nil
})

// Process typed objects
type Event struct {
    ID    string `json:"id"`
    Type  string `json:"type"`
    Data  string `json:"data"`
}

err := httpio.GetStreamInto(client, ctx, "/api/events", func(event Event) error {
    fmt.Printf("Event ID: %s, Type: %s\n", event.ID, event.Type)
    return nil
})
```

### Server-Sent Events Support

The library has built-in support for Server-Sent Events (SSE):

```go
err := httpio.GetSSE(client, ctx, "/api/events", func(event httpio.SSEEvent) {
    fmt.Printf("Event: %s, Data: %s\n", event.Event, event.Data)
})
```

## 📋 Examples

See the [examples directory](https://github.com/anggasct/httpio/tree/main/examples) for complete examples including:

- Basic HTTP requests
- Middleware usage (logging, retries, circuit breaking)
- Streaming responses
- Server-Sent Events processing
- OAuth authentication

## 📝 Documentation

For full API documentation, see [docs](https://github.com/anggasct/httpio/tree/main/docs) or use Go's built-in documentation:

```bash
go doc github.com/anggasct/httpio
```

## 🛠️ Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the [MIT License](LICENSE).
