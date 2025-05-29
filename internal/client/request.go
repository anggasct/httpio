// Package client implements the internal HTTP request/response handling
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/anggasct/httpio/middleware"
)

// Request is a prepared HTTP request
type Request struct {
	Method  string
	URL     string
	Headers http.Header
	Query   url.Values
	Body    interface{}
	Client  HTTPClient
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

// Do executes the request and returns the response
func (r *Request) Do(ctx context.Context) (*Response, error) {
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

	if len(client.GetMiddlewares()) > 0 {
		handler = middleware.Chain(baseHandler, client.GetMiddlewares()...)
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

// Stream executes the request and streams the response as raw bytes
func (r *Request) Stream(ctx context.Context, handler func([]byte) error) error {
	resp, err := r.Do(ctx)
	if err != nil {
		return err
	}
	return resp.Stream(handler)
}

// StreamLines executes the request and streams the response line by line
func (r *Request) StreamLines(ctx context.Context, handler func([]byte) error) error {
	resp, err := r.Do(ctx)
	if err != nil {
		return err
	}
	return resp.StreamLines(handler)
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
