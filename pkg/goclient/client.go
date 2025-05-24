package goclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/anggasct/goclient/pkg/circuitbreaker"
	"github.com/anggasct/goclient/pkg/interceptors"
)

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled   bool
	Namespace string
	Collector MetricsCollector
}

// MetricsCollector interface for collecting metrics
type MetricsCollector interface {
	RecordRequest(method, endpoint string, duration time.Duration, statusCode int)
	RecordError(method, endpoint string, err error)
}

// Client is a wrapper around http.Client with additional functionality
type Client struct {
	client              *http.Client
	baseURL             string
	headers             http.Header
	interceptor         interceptors.Interceptor
	responseInterceptor interceptors.ResponseInterceptor
	retryConfig         *RetryConfig
	circuitBreaker      *circuitbreaker.CircuitBreaker
	metricsConfig       *MetricsConfig
}

// RetryConfig holds configuration for request retries
type RetryConfig struct {
	MaxRetries  int
	WaitTime    time.Duration
	RetryPolicy func(resp *http.Response, err error) bool
}

// DefaultRetryPolicy returns true if the error is non-nil or the response has a status code >= 500
func DefaultRetryPolicy(resp *http.Response, err error) bool {
	return err != nil || (resp != nil && resp.StatusCode >= 500)
}

// New creates a new http Client
func New() *Client {
	return &Client{
		client:  &http.Client{},
		headers: make(http.Header),
	}
}

// WithBaseURL sets the base URL for all requests
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

// WithHeader sets a header for all requests
func (c *Client) WithHeader(key, value string) *Client {
	c.headers.Set(key, value)
	return c
}

// WithHeaders sets multiple headers for all requests
func (c *Client) WithHeaders(headers map[string]string) *Client {
	for k, v := range headers {
		c.headers.Set(k, v)
	}
	return c
}

// WithTimeout sets the timeout for all requests
func (c *Client) WithTimeout(timeout time.Duration) *Client {
	c.client.Timeout = timeout
	return c
}

// WithClient replaces the underlying http.Client
func (c *Client) WithClient(client *http.Client) *Client {
	c.client = client
	return c
}

// WithInterceptor sets the request interceptor
func (c *Client) WithInterceptor(interceptor interceptors.Interceptor) *Client {
	c.interceptor = interceptor
	return c
}

// WithResponseInterceptor sets the response interceptor
func (c *Client) WithResponseInterceptor(interceptor interceptors.ResponseInterceptor) *Client {
	c.responseInterceptor = interceptor
	return c
}

// WithRetry enables retry functionality
func (c *Client) WithRetry(maxRetries int, waitTime time.Duration, policy func(resp *http.Response, err error) bool) *Client {
	if policy == nil {
		policy = DefaultRetryPolicy
	}
	c.retryConfig = &RetryConfig{
		MaxRetries:  maxRetries,
		WaitTime:    waitTime,
		RetryPolicy: policy,
	}
	return c
}

// WithCircuitBreaker enables circuit breaker functionality
func (c *Client) WithCircuitBreaker(threshold int, timeout time.Duration, halfOpenMax int) *Client {
	config := &circuitbreaker.CircuitBreakerConfig{
		FailureThreshold: threshold,
		RecoveryTimeout:  timeout,
		HalfOpenMaxCalls: halfOpenMax,
	}
	c.circuitBreaker = circuitbreaker.NewCircuitBreaker(config)
	return c
}

// WithMetrics enables metrics collection
func (c *Client) WithMetrics(config *MetricsConfig) *Client {
	c.metricsConfig = config
	return c
}

// ConnectionPoolConfig holds connection pool configuration
type ConnectionPoolConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	KeepAlive           time.Duration
}

// WithConnectionPool configures connection pooling
func (c *Client) WithConnectionPool(config *ConnectionPoolConfig) *Client {
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
	}
	c.client.Transport = transport
	return c
}

// Request is a prepared HTTP request
type Request struct {
	method  string
	url     string
	headers http.Header
	query   url.Values
	body    interface{}
	client  *Client
}

// NewRequest creates a new request with the given method and URL
func (c *Client) NewRequest(method, path string) *Request {
	reqURL := path
	if c.baseURL != "" {
		reqURL = c.baseURL + path
	}

	req := &Request{
		method:  method,
		url:     reqURL,
		headers: make(http.Header),
		query:   make(url.Values),
		client:  c,
	}

	for k, vv := range c.headers {
		for _, v := range vv {
			req.headers.Add(k, v)
		}
	}

	return req
}

// WithHeader sets a header for this request
func (r *Request) WithHeader(key, value string) *Request {
	r.headers.Set(key, value)
	return r
}

// WithHeaders sets multiple headers for this request
func (r *Request) WithHeaders(headers map[string]string) *Request {
	for k, v := range headers {
		r.headers.Set(k, v)
	}
	return r
}

// WithQuery adds a query parameter to the request
func (r *Request) WithQuery(key, value string) *Request {
	r.query.Add(key, value)
	return r
}

// WithQueryMap adds multiple query parameters to the request
func (r *Request) WithQueryMap(params map[string]string) *Request {
	for k, v := range params {
		r.query.Add(k, v)
	}
	return r
}

// WithBody sets the request body
func (r *Request) WithBody(body interface{}) *Request {
	r.body = body
	return r
}

// Do executes the request and returns the response
func (r *Request) Do(ctx context.Context, client *Client) (*Response, error) {
	parsedURL, err := url.Parse(r.url)
	if err != nil {
		return nil, err
	}

	query := parsedURL.Query()
	for k, values := range r.query {
		for _, v := range values {
			query.Add(k, v)
		}
	}
	parsedURL.RawQuery = query.Encode()

	var bodyReader io.Reader
	if r.body != nil {
		switch b := r.body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(b)
		case string:
			bodyReader = bytes.NewReader([]byte(b))
		case io.Reader:
			bodyReader = b
		default:
			jsonBody, err := json.Marshal(r.body)
			if err != nil {
				return nil, err
			}
			bodyReader = bytes.NewReader(jsonBody)
			if r.headers.Get("Content-Type") == "" {
				r.headers.Set("Content-Type", "application/json")
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, r.method, parsedURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header = r.headers

	if client.interceptor != nil {
		req, err = client.interceptor.Intercept(ctx, req)
		if err != nil {
			return nil, err
		}
	}

	var resp *http.Response

	executeRequest := func() error {
		var execErr error
		if client.retryConfig != nil {
			resp, execErr = executeWithRetry(ctx, client, req, client.retryConfig)
		} else {
			resp, execErr = client.client.Do(req)
		}

		// For circuit breaker, consider HTTP 5xx responses as failures
		// but still return the response to the client
		if execErr == nil && resp != nil && resp.StatusCode >= 500 {
			return errors.New("server error response")
		}

		return execErr
	}

	if client.circuitBreaker != nil {
		err = client.circuitBreaker.Execute(executeRequest)
		// If circuit breaker rejected the request, return the error
		if err != nil && err.Error() == "circuit breaker is open - request rejected" {
			return nil, err
		}
		// If we got a "server error response" from executeRequest,
		// don't return it as an error to the client - the response was successful from HTTP perspective
		if err != nil && err.Error() == "server error response" {
			err = nil
		}
	} else {
		err = executeRequest()
	}

	if client.responseInterceptor != nil {
		resp, err = client.responseInterceptor.InterceptResponse(ctx, req, resp, err)
	}

	if err != nil {
		return nil, err
	}

	return &Response{
		Response: resp,
	}, nil
}

// executeWithRetry executes a request with retry logic
func executeWithRetry(ctx context.Context, client *Client, req *http.Request, config *RetryConfig) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		reqClone := cloneRequest(req)

		resp, err = client.client.Do(reqClone)
		if !config.RetryPolicy(resp, err) {
			return resp, err
		}

		if attempt == config.MaxRetries {
			return resp, err
		}

		if resp != nil {
			resp.Body.Close()
		}

		select {
		case <-ctx.Done():
			if resp != nil {
				resp.Body.Close()
			}
			return nil, ctx.Err()
		case <-time.After(config.WaitTime):
		}
	}

	return resp, err
}

// cloneRequest creates a clone of the provided request
func cloneRequest(req *http.Request) *http.Request {
	clone := req.Clone(req.Context())

	if req.Body == nil {
		return clone
	}

	return clone
}

// GetCircuitBreakerStats returns the current circuit breaker statistics
func (c *Client) GetCircuitBreakerStats() map[string]interface{} {
	if c.circuitBreaker == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}
	stats := c.circuitBreaker.GetStats()
	stats["enabled"] = true
	return stats
}

// ResetCircuitBreaker resets the circuit breaker to its initial state
func (c *Client) ResetCircuitBreaker() {
	if c.circuitBreaker != nil {
		c.circuitBreaker.Reset()
	}
}

// OnCircuitBreakerStateChange sets a callback for circuit breaker state changes
func (c *Client) OnCircuitBreakerStateChange(callback func(from, to circuitbreaker.CircuitBreakerState)) *Client {
	if c.circuitBreaker != nil {
		c.circuitBreaker.OnStateChange(callback)
	}
	return c
}
