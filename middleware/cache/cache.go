// Package cache provides HTTP response caching middleware for httpio.
//
// The cache middleware improves performance by storing HTTP responses and
// serving them from cache for subsequent identical requests. It supports
// multiple storage backends including in-memory, disk-based, and distributed
// caching systems.
package cache

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/anggasct/httpio/middleware"
)

func init() {
	gob.Register(&http.Response{})
	gob.Register(&http.Request{})
}

// CacheableStatus defines HTTP status codes that can be cached
var CacheableStatus = map[int]bool{
	http.StatusOK:                   true,
	http.StatusNonAuthoritativeInfo: true,
	http.StatusNoContent:            true,
	http.StatusPartialContent:       true,
	http.StatusMultipleChoices:      true,
	http.StatusMovedPermanently:     true,
	http.StatusNotFound:             true,
	http.StatusGone:                 true,
}

// CachedResponse represents a cached HTTP response
type CachedResponse struct {
	// Response is the HTTP response
	Response *http.Response
	// Body contains the response body bytes
	Body []byte
	// RequestURL is the original request URL
	RequestURL string
	// LastAccessed tracks when this entry was last used
	LastAccessed time.Time
	// CreatedAt tracks when this entry was created
	CreatedAt time.Time
	// ExpiresAt defines when this entry expires
	ExpiresAt time.Time
}

// Cache defines the interface that all cache implementations must satisfy
type Cache interface {
	Get(ctx context.Context, key string) (*CachedResponse, bool)
	Set(ctx context.Context, key string, response *CachedResponse) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
	Close() error
}

// Middleware implements the middleware.Middleware interface for HTTP caching
type Middleware struct {
	// cache is the underlying cache implementation
	cache Cache
	// config holds configuration settings
	config *Config
	// keyStrategy defines how cache keys are generated
	keyStrategy KeyStrategy
}

// NewMiddleware creates a new cache middleware instance with the specified cache and config
func NewMiddleware(cache Cache, config *Config) *Middleware {
	if config == nil {
		config = DefaultConfig()
	}

	var keyStrategy KeyStrategy
	switch config.KeyStrategy {
	case KeyByURLAndMethod:
		keyStrategy = NewMethodURLKeyStrategy()
	case KeyByFullRequest:
		keyStrategy = NewFullRequestKeyStrategy()
	case KeyByURLOnly:
		keyStrategy = NewURLOnlyKeyStrategy()
	default:
		keyStrategy = NewMethodURLKeyStrategy()
	}

	return &Middleware{
		cache:       cache,
		config:      config,
		keyStrategy: keyStrategy,
	}
}

// Handle implements the middleware.Middleware interface
func (m *Middleware) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		if !m.config.Enabled || !isCacheableMethod(req.Method) {
			return next(ctx, req)
		}

		if !m.shouldCache(req) {
			return next(ctx, req)
		}

		key := m.keyStrategy.GenerateKey(req)

		if cachedResp, found := m.cache.Get(ctx, key); found {
			if m.isFresh(cachedResp, req) {
				resp := cachedResp.Response
				resp.Body = io.NopCloser(bytes.NewReader(cachedResp.Body))
				return resp, nil
			}

			m.cache.Delete(ctx, key)
		}

		resp, err := next(ctx, req)
		if err != nil || resp == nil {
			return resp, err
		}

		if m.isCacheable(resp) {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return resp, nil
			}

			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			expiresAt := calculateExpiration(resp, m.config.DefaultTTL)

			respCopy := &http.Response{
				Status:           resp.Status,
				StatusCode:       resp.StatusCode,
				Proto:            resp.Proto,
				ProtoMajor:       resp.ProtoMajor,
				ProtoMinor:       resp.ProtoMinor,
				Header:           resp.Header.Clone(),
				ContentLength:    resp.ContentLength,
				TransferEncoding: resp.TransferEncoding,
				Close:            resp.Close,
				Uncompressed:     resp.Uncompressed,
				Trailer:          resp.Trailer.Clone(),
			}

			cachedResp := &CachedResponse{
				Response:     respCopy,
				Body:         bodyBytes,
				RequestURL:   req.URL.String(),
				LastAccessed: time.Now(),
				CreatedAt:    time.Now(),
				ExpiresAt:    expiresAt,
			}

			go func() {
				m.cache.Set(context.Background(), key, cachedResp)
			}()
		}

		return resp, nil
	}
}

// KeyStrategy defines how cache keys are generated from HTTP requests
type KeyStrategy interface {
	GenerateKey(req *http.Request) string
}

// KeyStrategyType defines the available key generation strategies
type KeyStrategyType int

const (
	// KeyByURLAndMethod uses URL + HTTP method for cache keys
	KeyByURLAndMethod KeyStrategyType = iota
	// KeyByURLOnly uses only the URL for cache keys
	KeyByURLOnly
	// KeyByFullRequest uses URL + method + headers + body for cache keys
	KeyByFullRequest
)

// MethodURLKeyStrategy generates keys based on HTTP method + URL
type MethodURLKeyStrategy struct{}

// NewMethodURLKeyStrategy creates a new MethodURLKeyStrategy
func NewMethodURLKeyStrategy() *MethodURLKeyStrategy {
	return &MethodURLKeyStrategy{}
}

func (s *MethodURLKeyStrategy) GenerateKey(req *http.Request) string {
	return req.Method + ":" + req.URL.String()
}

// URLOnlyKeyStrategy generates keys based only on URL
type URLOnlyKeyStrategy struct{}

func NewURLOnlyKeyStrategy() *URLOnlyKeyStrategy {
	return &URLOnlyKeyStrategy{}
}

func (s *URLOnlyKeyStrategy) GenerateKey(req *http.Request) string {
	return req.URL.String()
}

// FullRequestKeyStrategy generates keys based on method, URL, headers, and body
type FullRequestKeyStrategy struct{}

func NewFullRequestKeyStrategy() *FullRequestKeyStrategy {
	return &FullRequestKeyStrategy{}
}

func (s *FullRequestKeyStrategy) GenerateKey(req *http.Request) string {
	hasher := md5.New()

	io.WriteString(hasher, req.Method)
	io.WriteString(hasher, req.URL.String())

	var headerKeys []string
	for key := range req.Header {
		headerKeys = append(headerKeys, key)
	}
	sort.Strings(headerKeys)

	for _, key := range headerKeys {
		io.WriteString(hasher, key)
		for _, val := range req.Header[key] {
			io.WriteString(hasher, val)
		}
	}

	if req.Body != nil && req.ContentLength > 0 && req.ContentLength < 1024*1024 {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			req.Body.Close()
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			hasher.Write(bodyBytes)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func isCacheableMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

func (m *Middleware) shouldCache(req *http.Request) bool {
	urlStr := req.URL.String()
	host := req.URL.Host

	for _, pattern := range m.config.ExcludePatterns {
		if strings.Contains(urlStr, pattern) {
			return false
		}
	}

	for _, h := range m.config.ExcludeHosts {
		if host == h {
			return false
		}
	}

	if len(m.config.IncludePatterns) > 0 {
		for _, pattern := range m.config.IncludePatterns {
			if strings.Contains(urlStr, pattern) {
				return true
			}
		}
		return false
	}

	return true
}

func (m *Middleware) isFresh(cachedResp *CachedResponse, req *http.Request) bool {
	if time.Now().After(cachedResp.ExpiresAt) {
		return false
	}

	if m.config.RespectCacheControl {
		if cacheControl := req.Header.Get("Cache-Control"); cacheControl != "" {
			directives := strings.Split(cacheControl, ",")
			for _, directive := range directives {
				if strings.TrimSpace(directive) == "no-cache" {
					return false
				}
			}
		}

		if pragma := req.Header.Get("Pragma"); pragma == "no-cache" {
			return false
		}
	}

	cachedResp.LastAccessed = time.Now()
	return true
}

func (m *Middleware) isCacheable(resp *http.Response) bool {
	if !CacheableStatus[resp.StatusCode] {
		return false
	}

	if m.config.RespectCacheControl {
		if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "" {
			directives := strings.Split(cacheControl, ",")
			for _, directive := range directives {
				directive = strings.TrimSpace(directive)
				if directive == "no-store" || directive == "no-cache" || directive == "private" {
					return false
				}
			}
		}
	}

	return true
}

func calculateExpiration(resp *http.Response, defaultTTL time.Duration) time.Time {
	if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "" {
		directives := strings.Split(cacheControl, ",")
		for _, directive := range directives {
			directive = strings.TrimSpace(directive)
			if strings.HasPrefix(directive, "max-age=") {
				seconds := strings.TrimPrefix(directive, "max-age=")
				if maxAge, err := time.ParseDuration(seconds + "s"); err == nil {
					return time.Now().Add(maxAge)
				}
			}
		}
	}

	if expires := resp.Header.Get("Expires"); expires != "" {
		if expiresTime, err := time.Parse(time.RFC1123, expires); err == nil {
			return expiresTime
		}
	}

	return time.Now().Add(defaultTTL)
}
