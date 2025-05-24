# goclient

[![Go Report Card](https://goreportcard.com/badge/github.com/anggasct/goclient)](https://goreportcard.com/report/github.com/anggasct/goclient)
[![GoDoc](https://godoc.org/github.com/anggasct/goclient?status.svg)](https://godoc.org/github.com/anggasct/goclient)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![CI](https://github.com/anggasct/goclient/workflows/CI/badge.svg)](https://github.com/anggasct/goclient/actions)

A minimal and elegant HTTP client wrapper for Go, built on top of the standard `net/http` library.

## ✨ Features

- ✅ **Simple API** for GET, POST, PUT, DELETE, etc.
- 🧱 **Clean abstraction** over net/http (not a replacement)
- 📦 **JSON encoding/decoding** helpers
- 🧾 **Easy header, query params, and body handling**
- 🔄 **Built-in retry support** with configurable policies
- ⚡ **Circuit breaker pattern** for resilience and fault tolerance
- ⏱ **Context-aware** requests and interceptors
- 🔌 **Extensible** with request and response interceptors
- 📥 **Streaming support** for responses and SSE (Server-Sent Events)
- 🎯 **Type-safe** streaming with automatic JSON unmarshaling
- 📊 **Built-in metrics** and monitoring capabilities
- 🚀 **Production-ready** with comprehensive testing

## 📦 Installation

```bash
go get github.com/anggasct/goclient
```

## 🚀 Quick Start

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
    client := goclient.New().
        WithBaseURL("https://api.github.com").
        WithHeader("User-Agent", "my-app/1.0").
        WithTimeout(10 * time.Second)

    ctx := context.Background()
    resp, err := client.GET(ctx, "/users/octocat")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Close()

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

## 📖 Documentation

- **[Getting Started](docs/getting-started.md)** - Quick start guide and basic usage
- **[Advanced Usage](docs/advanced-usage.md)** - Interceptors, retry, connection pooling, and more
- **[Circuit Breaker](docs/circuit-breaker.md)** - Resilience patterns and fault tolerance
- **[Streaming](docs/streaming.md)** - Handling streaming responses and Server-Sent Events
- **[Examples](examples/)** - Working code examples for all features

## 🏗 Core Concepts

### Building Requests

```go
// Simple GET with query parameters
resp, err := client.NewRequest("GET", "/search").
    WithQuery("q", "golang").
    WithQuery("sort", "stars").
    Do(ctx, client)

// POST with JSON body
data := map[string]interface{}{
    "name":  "John",
    "email": "john@example.com",
}
resp, err := client.POST(ctx, "/users", data)
```

### Handling Responses

```go
// Get response as string
str, err := resp.String()

// Get response as bytes
bytes, err := resp.Bytes()

// Parse JSON into struct
var result MyStruct
err := resp.JSON(&result)

// Check status
if resp.IsSuccess() {
    // Handle success
}
```

### Streaming Responses

```go
// Stream raw chunks
err := resp.Stream(func(chunk []byte) error {
    fmt.Printf("Received %d bytes\n", len(chunk))
    return nil
})

// Stream lines
err := resp.StreamLines(func(line []byte) error {
    fmt.Println(string(line))
    return nil
})

// Stream JSON objects
err := resp.StreamJSON(func(raw json.RawMessage) error {
    var data MyStruct
    json.Unmarshal(raw, &data)
    return nil
})

// Type-safe streaming
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

err := resp.StreamInto(func(user User) error {
    fmt.Printf("User: %s\n", user.Name)
    return nil
})
```

## ⚡ Advanced Features

### Circuit Breaker

```go
// Enable circuit breaker
client := goclient.New().
    WithCircuitBreaker(3, 2*time.Second, 2) // 3 failures, 2s timeout, 2 half-open calls

// Monitor state changes
client.OnCircuitBreakerStateChange(func(from, to goclient.CircuitBreakerState) {
    fmt.Printf("Circuit breaker: %s -> %s\n", from, to)
})
```

### Retry Logic

```go
// Basic retry
client.WithRetry(3, 500*time.Millisecond, nil)

// Custom retry policy
customPolicy := func(resp *http.Response, err error) bool {
    return err != nil || (resp != nil && resp.StatusCode == 429)
}
client.WithRetry(5, 1*time.Second, customPolicy)
```

### Request Interceptors

```go
// Add authentication
authInterceptor := goclient.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
    if token := getTokenFromContext(ctx); token != "" {
        req.Header.Set("Authorization", "Bearer "+token)
    }
    return req, nil
})

client.WithInterceptor(authInterceptor)
```

### Response Interceptors

```go
// Log responses
logger := goclient.ResponseInterceptorFunc(func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
    if err != nil {
        log.Printf("Request failed: %v", err)
    } else {
        log.Printf("Response: %s", resp.Status)
    }
    return resp, err
})

client.WithResponseInterceptor(logger)
```

## 🎯 Examples

The [examples/](examples/) directory contains working examples for all features:

- **[Basic Usage](examples/basic/)** - Simple HTTP requests and responses
- **[Circuit Breaker](examples/circuit-breaker/)** - Resilience patterns in action
- **[Streaming](examples/streaming/)** - Real-time data processing
- **[Advanced](examples/advanced/)** - Interceptors and complex scenarios

## 🧪 Testing

```bash
# Run all tests
make test

# Run integration tests
make test-integration

# Run benchmarks
make test-bench

# Run with coverage
go test -cover ./pkg/...
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/anggasct/goclient.git
cd goclient

# Install development tools
make install-tools

# Run tests
make test

# Run linting
make lint

# Format code
make fmt
```

## 📊 Benchmarks

```
BenchmarkClient_GET-8              5000    250000 ns/op    1024 B/op     5 allocs/op
BenchmarkClient_POST-8             3000    350000 ns/op    1536 B/op     8 allocs/op
BenchmarkStreaming_JSON-8         10000    100000 ns/op     512 B/op     3 allocs/op
BenchmarkCircuitBreaker-8        100000     12000 ns/op      64 B/op     1 allocs/op
```

## 🔗 Related Projects

- [net/http](https://golang.org/pkg/net/http/) - Go's standard HTTP library
- [fasthttp](https://github.com/valyala/fasthttp) - Fast HTTP package for Go
- [resty](https://github.com/go-resty/resty) - Simple HTTP and REST client library

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- The Go team for the excellent standard library
- The open-source community for inspiration and feedback
- All contributors who help make this project better

---

Made with ❤️ by [anggasct](https://github.com/anggasct)
