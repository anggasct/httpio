package goclient

import (
	"context"
)

// GET performs a GET request
func (c *Client) GET(ctx context.Context, path string) (*Response, error) {
	return c.NewRequest("GET", path).Do(ctx, c)
}

// POST performs a POST request
func (c *Client) POST(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.NewRequest("POST", path).WithBody(body).Do(ctx, c)
}

// PUT performs a PUT request
func (c *Client) PUT(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.NewRequest("PUT", path).WithBody(body).Do(ctx, c)
}

// PATCH performs a PATCH request
func (c *Client) PATCH(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.NewRequest("PATCH", path).WithBody(body).Do(ctx, c)
}

// DELETE performs a DELETE request
func (c *Client) DELETE(ctx context.Context, path string) (*Response, error) {
	return c.NewRequest("DELETE", path).Do(ctx, c)
}

// HEAD performs a HEAD request
func (c *Client) HEAD(ctx context.Context, path string) (*Response, error) {
	return c.NewRequest("HEAD", path).Do(ctx, c)
}

// OPTIONS performs an OPTIONS request
func (c *Client) OPTIONS(ctx context.Context, path string) (*Response, error) {
	return c.NewRequest("OPTIONS", path).Do(ctx, c)
}
