package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/anggasct/httpio/internal/middleware"
)

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
func (r *Request) Do(ctx context.Context) (*Response, error) {
	client := r.client
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
	var rawBody []byte

	if r.body != nil {
		switch b := r.body.(type) {
		case []byte:
			rawBody = b
			bodyReader = bytes.NewReader(b)
		case string:
			rawBody = []byte(b)
			bodyReader = bytes.NewReader(rawBody)
		default:
			jsonBody, err := json.Marshal(r.body)
			if err != nil {
				return nil, err
			}
			rawBody = jsonBody
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

	// Create a base handler that uses the client's HTTP client
	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return client.client.Do(req)
	}

	handler := baseHandler

	if len(client.middlewares) > 0 {
		handler = middleware.Chain(baseHandler, client.middlewares...)
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
