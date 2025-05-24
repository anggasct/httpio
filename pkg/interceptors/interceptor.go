package interceptors

import (
	"context"
	"net/http"
)

// Interceptor defines the interface for intercepting and possibly modifying requests
// before they are sent to the server.
type Interceptor interface {
	// Intercept is called before a request is sent to the server.
	// It can modify the request or return an error to prevent the request from being sent.
	// The context allows interceptors to access request context values or cancel the request.
	Intercept(ctx context.Context, req *http.Request) (*http.Request, error)
}

// InterceptorFunc is a functional implementation of the Interceptor interface
type InterceptorFunc func(ctx context.Context, req *http.Request) (*http.Request, error)

// Intercept implements the Interceptor interface
func (f InterceptorFunc) Intercept(ctx context.Context, req *http.Request) (*http.Request, error) {
	return f(ctx, req)
}

// ChainInterceptors chains multiple interceptors together
func ChainInterceptors(interceptors ...Interceptor) Interceptor {
	return InterceptorFunc(func(ctx context.Context, req *http.Request) (*http.Request, error) {
		var err error
		for _, interceptor := range interceptors {
			req, err = interceptor.Intercept(ctx, req)
			if err != nil {
				return nil, err
			}
		}
		return req, nil
	})
}

// ResponseInterceptor defines an interface for intercepting and modifying responses
// after they are received from the server.
type ResponseInterceptor interface {
	// InterceptResponse is called after a response is received from the server.
	// It can modify the response or process it in some way.
	InterceptResponse(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error)
}

// ResponseInterceptorFunc is a functional implementation of the ResponseInterceptor interface
type ResponseInterceptorFunc func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error)

// InterceptResponse implements the ResponseInterceptor interface
func (f ResponseInterceptorFunc) InterceptResponse(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
	return f(ctx, req, resp, err)
}

// ChainResponseInterceptors chains multiple response interceptors together
func ChainResponseInterceptors(interceptors ...ResponseInterceptor) ResponseInterceptor {
	return ResponseInterceptorFunc(func(ctx context.Context, req *http.Request, resp *http.Response, err error) (*http.Response, error) {
		var e error = err
		r := resp
		for _, interceptor := range interceptors {
			r, e = interceptor.InterceptResponse(ctx, req, r, e)
		}
		return r, e
	})
}
