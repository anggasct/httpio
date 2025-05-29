package test

import (
	"context"
	"net/http"
	"testing"

	"github.com/anggasct/httpio/middleware"
)

type mockHandler struct {
	called bool
}

func (m *mockHandler) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		m.called = true
		return next(ctx, req)
	}
}

type testMiddleware struct {
	name   string
	called bool
}

func (tm *testMiddleware) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		tm.called = true
		req.Header.Set("X-Middleware-"+tm.name, "called")
		return next(ctx, req)
	}
}

func TestChain(t *testing.T) {
	middleware1 := &testMiddleware{name: "First"}
	middleware2 := &testMiddleware{name: "Second"}

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		resp := &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
		}

		// Check if middlewares were called
		if req.Header.Get("X-Middleware-First") != "called" {
			t.Error("First middleware was not called")
		}
		if req.Header.Get("X-Middleware-Second") != "called" {
			t.Error("Second middleware was not called")
		}

		return resp, nil
	}

	handler := middleware.Chain(baseHandler, middleware1, middleware2)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header = make(http.Header)

	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !middleware1.called {
		t.Error("First middleware was not called")
	}

	if !middleware2.called {
		t.Error("Second middleware was not called")
	}
}

func TestWrapMiddleware(t *testing.T) {
	called := false

	middlewareFunc := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			called = true
			req.Header.Set("X-Wrapped", "true")
			return next(ctx, req)
		}
	}

	wrappedMiddleware := middleware.WrapMiddleware(middlewareFunc)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		if req.Header.Get("X-Wrapped") != "true" {
			t.Error("Wrapped middleware was not applied")
		}
		return &http.Response{StatusCode: 200}, nil
	}

	handler := wrappedMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header = make(http.Header)

	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !called {
		t.Error("Wrapped middleware was not called")
	}
}

func TestMiddlewareOrder(t *testing.T) {
	var order []string

	middleware1 := middleware.WrapMiddleware(func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			order = append(order, "before-1")
			resp, err := next(ctx, req)
			order = append(order, "after-1")
			return resp, err
		}
	})

	middleware2 := middleware.WrapMiddleware(func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *http.Request) (*http.Response, error) {
			order = append(order, "before-2")
			resp, err := next(ctx, req)
			order = append(order, "after-2")
			return resp, err
		}
	})

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		order = append(order, "handler")
		return &http.Response{StatusCode: 200}, nil
	}

	handler := middleware.Chain(baseHandler, middleware1, middleware2)

	req, _ := http.NewRequest("GET", "http://example.com", nil)

	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []string{"before-1", "before-2", "handler", "after-2", "after-1"}
	if len(order) != len(expected) {
		t.Fatalf("Expected %d calls, got %d", len(expected), len(order))
	}

	for i, expected := range expected {
		if order[i] != expected {
			t.Errorf("Expected order[%d] = %s, got %s", i, expected, order[i])
		}
	}
}
