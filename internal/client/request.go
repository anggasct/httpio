// Package client implements the internal HTTP request/response handling
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/anggasct/httpio/middleware"
)

// Request represents a prepared HTTP request with middleware support
type Request struct {
	Method      string
	URL         string
	Headers     http.Header
	Query       url.Values
	Body        interface{}
	Client      HTTPClient
	middlewares []middleware.Middleware
	timeout     *time.Duration
}

// HTTPClient defines the interface for the HTTP client
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	GetMiddlewares() []middleware.Middleware
}

// WithHeader sets a header for this request
func (r *Request) WithHeader(key, value string) *Request {
	r.Headers.Set(key, value)
	return r
}

// WithHeaders sets multiple headers for this request
func (r *Request) WithHeaders(headers map[string]string) *Request {
	for k, v := range headers {
		r.Headers.Set(k, v)
	}
	return r
}

// WithQuery adds a query parameter to the request
func (r *Request) WithQuery(key, value string) *Request {
	r.Query.Add(key, value)
	return r
}

// WithQueryMap adds multiple query parameters to the request
func (r *Request) WithQueryMap(params map[string]string) *Request {
	for k, v := range params {
		r.Query.Add(k, v)
	}
	return r
}

// WithBody sets the request body
func (r *Request) WithBody(body interface{}) *Request {
	r.Body = body
	return r
}

// WithMiddleware adds middleware specific to this request
func (r *Request) WithMiddleware(m middleware.Middleware) *Request {
	if r.middlewares == nil {
		r.middlewares = make([]middleware.Middleware, 0)
	}
	r.middlewares = append(r.middlewares, m)
	return r
}

// WithMiddlewares adds multiple middlewares specific to this request
func (r *Request) WithMiddlewares(middlewares ...middleware.Middleware) *Request {
	if r.middlewares == nil {
		r.middlewares = make([]middleware.Middleware, 0, len(middlewares))
	}
	r.middlewares = append(r.middlewares, middlewares...)
	return r
}

// WithTimeout sets a timeout specific to this request
func (r *Request) WithTimeout(timeout time.Duration) *Request {
	r.timeout = &timeout
	return r
}

// Do executes the request and returns the response
func (r *Request) Do(ctx context.Context) (*Response, error) {
	if r.timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *r.timeout)
		defer cancel()
	}

	client := r.Client
	parsedURL, err := url.Parse(r.URL)
	if err != nil {
		return nil, err
	}

	query := parsedURL.Query()
	for k, values := range r.Query {
		for _, v := range values {
			query.Add(k, v)
		}
	}
	parsedURL.RawQuery = query.Encode()

	var bodyReader io.Reader
	var rawBody []byte

	if r.Body != nil {
		switch b := r.Body.(type) {
		case []byte:
			rawBody = b
			bodyReader = bytes.NewReader(b)
		case string:
			rawBody = []byte(b)
			bodyReader = bytes.NewReader(rawBody)
		default:
			jsonBody, err := json.Marshal(r.Body)
			if err != nil {
				return nil, err
			}
			rawBody = jsonBody
			bodyReader = bytes.NewReader(jsonBody)
			if r.Headers.Get("Content-Type") == "" {
				r.Headers.Set("Content-Type", "application/json")
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, r.Method, parsedURL.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header = r.Headers

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return client.Do(req)
	}

	handler := baseHandler

	allMiddlewares := r.buildMiddlewareChain()
	if len(allMiddlewares) > 0 {
		handler = middleware.Chain(baseHandler, allMiddlewares...)
	}

	resp, err := handler(ctx, req)
	if err != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, err
	}

	response := &Response{
		Response: resp,
	}

	return response, nil
}

// buildMiddlewareChain combines client middlewares with request-specific middlewares
func (r *Request) buildMiddlewareChain() []middleware.Middleware {
	clientMiddlewares := r.Client.GetMiddlewares()
	if len(clientMiddlewares) == 0 && len(r.middlewares) == 0 {
		return nil
	}

	totalLen := len(clientMiddlewares) + len(r.middlewares)
	allMiddlewares := make([]middleware.Middleware, 0, totalLen)

	allMiddlewares = append(allMiddlewares, clientMiddlewares...)
	allMiddlewares = append(allMiddlewares, r.middlewares...)

	return allMiddlewares
}

// Stream executes the request and streams the response as raw bytes
func (r *Request) Stream(ctx context.Context, handler func([]byte) error, opts ...StreamOption) error {
	resp, err := r.Do(ctx)
	if err != nil {
		return err
	}
	return resp.Stream(handler, opts...)
}

// StreamLines executes the request and streams the response line by line
func (r *Request) StreamLines(ctx context.Context, handler func([]byte) error, opts ...StreamOption) error {
	resp, err := r.Do(ctx)
	if err != nil {
		return err
	}
	return resp.StreamLines(handler, opts...)
}

// StreamJSON executes the request and streams the response as JSON objects
func (r *Request) StreamJSON(ctx context.Context, handler func(json.RawMessage) error) error {
	resp, err := r.Do(ctx)
	if err != nil {
		return err
	}
	return resp.StreamJSON(handler)
}

// StreamInto executes the request and unmarshals each JSON object into the specified type
func (r *Request) StreamInto(ctx context.Context, handler interface{}) error {
	resp, err := r.Do(ctx)
	if err != nil {
		return err
	}
	return resp.StreamInto(handler)
}

// StreamSSE executes the request and streams the response as Server-Sent Events
func (r *Request) StreamSSE(ctx context.Context, handler EventSourceHandler) error {
	resp, err := r.Do(ctx)
	if err != nil {
		return err
	}
	return resp.StreamSSE(handler)
}
