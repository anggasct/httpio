package goclient

import (
	"encoding/json"
	"io"
	"net/http"
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
