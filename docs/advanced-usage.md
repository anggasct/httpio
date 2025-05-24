# Advanced Usage

This guide covers advanced features of goclient including interceptors, retry mechanisms, connection pooling, and more.

## Table of Contents

- [Request Interceptors](#request-interceptors)
- [Response Interceptors](#response-interceptors)
- [Retry Mechanism](#retry-mechanism)
- [Connection Pooling](#connection-pooling)
- [Custom HTTP Client](#custom-http-client)
- [Metrics and Monitoring](#metrics-and-monitoring)
- [Best Practices](#best-practices)

## Request Interceptors

Request interceptors allow you to modify requests before they are sent to the server.

### Basic Interceptor

```go
// Create an authentication interceptor
authInterceptor := goclient.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
    // Add authentication header
    if token := getTokenFromContext(ctx); token != "" {
        req.Header.Set("Authorization", "Bearer "+token)
    }
    
    // Add request ID for tracing
    if requestID := getRequestIDFromContext(ctx); requestID != "" {
        req.Header.Set("X-Request-ID", requestID)
    }
    
    return req, nil
})

client := goclient.New().
    WithInterceptor(authInterceptor)
```

### Multiple Interceptors

```go
// Chain multiple interceptors
loggingInterceptor := goclient.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
    log.Printf("Making request to %s", req.URL.String())
    return req, nil
})

rateLimitInterceptor := goclient.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
    // Wait for rate limit
    if err := rateLimiter.Wait(ctx); err != nil {
        return nil, err
    }
    return req, nil
})

client := goclient.New().
    WithInterceptor(goclient.ChainInterceptors(
        authInterceptor,
        loggingInterceptor,
        rateLimitInterceptor,
    ))
```

### Context-Aware Interceptors

```go
// Use context values in interceptors
ctx := context.Background()
ctx = context.WithValue(ctx, "user_id", "12345")
ctx = context.WithValue(ctx, "tenant_id", "tenant-abc")

tenantInterceptor := goclient.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
    if tenantID, ok := ctx.Value("tenant_id").(string); ok {
        req.Header.Set("X-Tenant-ID", tenantID)
    }
    
    if userID, ok := ctx.Value("user_id").(string); ok {
        req.Header.Set("X-User-ID", userID)
    }
    
    return req, nil
})

client := goclient.New().WithInterceptor(tenantInterceptor)
resp, err := client.GET(ctx, "/api/data")
```

## Response Interceptors

Response interceptors allow you to process responses before they are returned to the caller.

### Basic Response Interceptor

```go
// Create a logging response interceptor
responseLogger := goclient.ResponseInterceptorFunc(func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
    if err != nil {
        log.Printf("Request to %s failed: %v", req.URL.String(), err)
        return resp, err
    }
    
    log.Printf("Response from %s: %d %s", req.URL.String(), resp.StatusCode, resp.Status)
    return resp, nil
})

client := goclient.New().
    WithResponseInterceptor(responseLogger)
```

### Error Handling Interceptor

```go
// Custom error handling
errorHandler := goclient.ResponseInterceptorFunc(func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
    if err != nil {
        // Log error with context
        log.Printf("Request failed [%s]: %v", getRequestIDFromContext(ctx), err)
        return resp, err
    }
    
    // Handle specific HTTP errors
    switch resp.StatusCode {
    case 401:
        // Clear cached tokens, trigger re-authentication
        clearAuthToken(ctx)
        return resp, errors.New("authentication required")
    case 429:
        // Log rate limiting
        log.Printf("Rate limited by %s", req.URL.Host)
    case 503:
        // Service unavailable
        log.Printf("Service %s is unavailable", req.URL.Host)
    }
    
    return resp, nil
})

client := goclient.New().
    WithResponseInterceptor(errorHandler)
```

### Metrics Collection

```go
// Collect request metrics
metricsInterceptor := goclient.ResponseInterceptorFunc(func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
    start := time.Now()
    
    // Record request duration
    duration := time.Since(start)
    
    labels := map[string]string{
        "method":   req.Method,
        "endpoint": req.URL.Path,
    }
    
    if err != nil {
        labels["status"] = "error"
        requestDuration.WithLabelValues(labels).Observe(duration.Seconds())
        errorCounter.WithLabelValues(labels).Inc()
    } else {
        labels["status"] = fmt.Sprintf("%d", resp.StatusCode)
        requestDuration.WithLabelValues(labels).Observe(duration.Seconds())
        
        if resp.StatusCode >= 400 {
            errorCounter.WithLabelValues(labels).Inc()
        }
    }
    
    return resp, err
})
```

## Retry Mechanism

Configure automatic retries for failed requests.

### Basic Retry

```go
// Retry up to 3 times with 500ms wait between attempts
client := goclient.New().
    WithRetry(3, 500*time.Millisecond, nil) // Uses default retry policy
```

### Custom Retry Policy

```go
// Custom retry policy
customRetryPolicy := func(resp *http.Response, err error) bool {
    // Retry on network errors
    if err != nil {
        return true
    }
    
    // Retry on specific status codes
    return resp.StatusCode == 429 || // Rate limited
           resp.StatusCode == 502 || // Bad gateway
           resp.StatusCode == 503 || // Service unavailable
           resp.StatusCode == 504    // Gateway timeout
}

client := goclient.New().
    WithRetry(5, 1*time.Second, customRetryPolicy)
```

### Exponential Backoff

```go
// Implement exponential backoff
type ExponentialBackoff struct {
    baseDelay time.Duration
    maxDelay  time.Duration
    factor    float64
}

func (eb *ExponentialBackoff) Wait(attempt int) time.Duration {
    delay := time.Duration(float64(eb.baseDelay) * math.Pow(eb.factor, float64(attempt)))
    if delay > eb.maxDelay {
        delay = eb.maxDelay
    }
    return delay
}

// Custom retry with exponential backoff
func retryWithBackoff(ctx context.Context, client *goclient.Client, req *goclient.Request, maxRetries int) (*goclient.Response, error) {
    backoff := &ExponentialBackoff{
        baseDelay: 100 * time.Millisecond,
        maxDelay:  10 * time.Second,
        factor:    2.0,
    }
    
    var lastErr error
    for attempt := 0; attempt <= maxRetries; attempt++ {
        resp, err := req.Do(ctx, client)
        if err == nil && resp.IsSuccess() {
            return resp, nil
        }
        
        lastErr = err
        if resp != nil {
            resp.Close()
        }
        
        if attempt < maxRetries {
            wait := backoff.Wait(attempt)
            time.Sleep(wait)
        }
    }
    
    return nil, lastErr
}
```

## Connection Pooling

Configure HTTP connection pooling for better performance.

```go
poolConfig := &goclient.ConnectionPoolConfig{
    MaxIdleConns:        100,               // Maximum idle connections
    MaxIdleConnsPerHost: 10,                // Maximum idle connections per host
    IdleConnTimeout:     90 * time.Second,  // How long to keep idle connections
    KeepAlive:          30 * time.Second,   // TCP keep-alive period
}

client := goclient.New().
    WithConnectionPool(poolConfig)
```

## Custom HTTP Client

Use a custom underlying HTTP client.

```go
// Custom transport with proxy
transport := &http.Transport{
    Proxy: http.ProxyURL(proxyURL),
    TLSClientConfig: &tls.Config{
        InsecureSkipVerify: false,
    },
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     90 * time.Second,
}

httpClient := &http.Client{
    Transport: transport,
    Timeout:   30 * time.Second,
}

client := goclient.New().
    WithClient(httpClient)
```

## Metrics and Monitoring

### Prometheus Metrics

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    requestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "Duration of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
    
    requestCount = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )
)

// Metrics collection interceptor
metricsInterceptor := goclient.ResponseInterceptorFunc(func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
    start := time.Now()
    duration := time.Since(start)
    
    status := "error"
    if err == nil {
        status = fmt.Sprintf("%d", resp.StatusCode)
    }
    
    labels := []string{req.Method, req.URL.Path, status}
    requestDuration.WithLabelValues(labels...).Observe(duration.Seconds())
    requestCount.WithLabelValues(labels...).Inc()
    
    return resp, err
})
```

### OpenTelemetry Tracing

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

tracingInterceptor := goclient.InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
    tracer := otel.Tracer("goclient")
    
    ctx, span := tracer.Start(ctx, "http.request",
        trace.WithAttributes(
            attribute.String("http.method", req.Method),
            attribute.String("http.url", req.URL.String()),
        ),
    )
    
    // Add trace context to request
    req = req.WithContext(ctx)
    
    // Ensure span is finished when request completes
    // This would typically be done in a response interceptor
    
    return req, nil
})
```

## Best Practices

### 1. Client Reuse

```go
// Good: Reuse client instances
var apiClient = goclient.New().
    WithBaseURL("https://api.example.com").
    WithTimeout(30 * time.Second)

func makeRequest(ctx context.Context) {
    resp, err := apiClient.GET(ctx, "/data")
    // ...
}

// Bad: Creating new client for each request
func makeRequestBad(ctx context.Context) {
    client := goclient.New() // Don't do this!
    resp, err := client.GET(ctx, "https://api.example.com/data")
    // ...
}
```

### 2. Context Usage

```go
// Always use context with timeouts
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

resp, err := client.GET(ctx, "/api/data")
```

### 3. Resource Cleanup

```go
resp, err := client.GET(ctx, "/api/data")
if err != nil {
    return err
}
defer resp.Close() // Always close responses

// Process response
data, err := resp.Bytes()
```

### 4. Error Handling

```go
resp, err := client.GET(ctx, "/api/data")
if err != nil {
    // Handle request errors (network, timeout, etc.)
    return fmt.Errorf("request failed: %w", err)
}
defer resp.Close()

if !resp.IsSuccess() {
    // Handle HTTP errors
    body, _ := resp.String()
    return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, body)
}
```

### 5. Configuration Management

```go
type Config struct {
    BaseURL    string        `json:"base_url"`
    Timeout    time.Duration `json:"timeout"`
    MaxRetries int           `json:"max_retries"`
    APIKey     string        `json:"api_key"`
}

func NewClientFromConfig(cfg *Config) *goclient.Client {
    client := goclient.New().
        WithBaseURL(cfg.BaseURL).
        WithTimeout(cfg.Timeout).
        WithHeader("Authorization", "Bearer "+cfg.APIKey)
    
    if cfg.MaxRetries > 0 {
        client = client.WithRetry(cfg.MaxRetries, 500*time.Millisecond, nil)
    }
    
    return client
}
```
