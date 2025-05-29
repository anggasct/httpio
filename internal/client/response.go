// Package client implements the internal HTTP request/response handling
package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

// Response wraps the standard http.Response with additional utility methods
type Response struct {
	*http.Response
}

// Bytes reads the entire response body and returns it as a byte slice
func (r *Response) Bytes() ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(r.Body)
}

// String reads the entire response body and returns it as a string
func (r *Response) String() (string, error) {
	bytes, err := r.Bytes()
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// JSON unmarshals the response body into the provided interface
func (r *Response) JSON(v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

// Close closes the response body
func (r *Response) Close() error {
	return r.Body.Close()
}

// WriteTo implements io.WriterTo, streaming the response body to the provided writer
func (r *Response) WriteTo(w io.Writer) (int64, error) {
	defer r.Body.Close()
	return io.Copy(w, r.Body)
}

// Pipe allows for piping the response body to the provided channel
func (r *Response) Pipe(ch chan<- []byte) error {
	defer r.Body.Close()
	buf := make([]byte, 1024)
	for {
		n, err := r.Body.Read(buf)
		if n > 0 {
			ch <- buf[:n]
		}
		if err != nil {
			close(ch)
			return err
		}
	}
}

// Consume reads and discards the response body
func (r *Response) Consume() error {
	defer r.Body.Close()
	_, err := io.Copy(io.Discard, r.Body)
	return err
}

// IsSuccess returns true if the status code is between 200 and 299
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode <= 299
}

// IsRedirect returns true if the status code is 3xx
func (r *Response) IsRedirect() bool {
	return r.StatusCode >= 300 && r.StatusCode <= 399
}

// IsClientError returns true if the status code is 4xx
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode <= 499
}

// IsServerError returns true if the status code is 5xx
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500 && r.StatusCode <= 599
}

// IsError returns true if the status code indicates an error (4xx or 5xx)
func (r *Response) IsError() bool {
	return r.IsClientError() || r.IsServerError()
}

// Stream processes a response stream with the provided handler function.
// The handler is called for each chunk of data.
func (r *Response) Stream(handler func([]byte) error, opts ...StreamOption) error {
	return Stream(r, handler, opts...)
}

// StreamLines processes a response stream line by line with the provided handler function.
func (r *Response) StreamLines(handler func([]byte) error, opts ...StreamOption) error {
	return StreamLines(r, handler, opts...)
}

// StreamJSON processes a response stream as JSON objects with the provided handler function.
func (r *Response) StreamJSON(handler func(json.RawMessage) error, opts ...StreamOption) error {
	return StreamJSON(r, handler, opts...)
}

// StreamInto processes a response stream as JSON objects and unmarshals each object into a new
// instance of the provided type, then passes it to the handler function.
func (r *Response) StreamInto(handler interface{}, opts ...StreamOption) error {
	return StreamInto(r, handler, opts...)
}

// StreamSSE processes a Server-Sent Events stream with the provided handler function.
func (r *Response) StreamSSE(handler EventSourceHandler) error {
	if !strings.Contains(r.Header.Get("Content-Type"), "text/event-stream") {
		r.Close()
		return errors.New("unexpected content type for SSE: " + r.Header.Get("Content-Type"))
	}

	return StreamSSE(r.Body, handler)
}
