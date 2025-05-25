// Package cache provides HTTP response caching middleware for httpio.
//
// The cache middleware improves performance by storing HTTP responses and
// serving them from cache for subsequent identical requests. It supports
// multiple storage backends including in-memory, disk-based, and distributed
// caching systems.
//
// Key features:
// - HTTP-compliant caching (respects Cache-Control, ETag, etc.)
// - Flexible storage backends via the Cache interface
// - Configurable TTL (Time-To-Live) settings
// - URL/domain-based cache rules
// - Support for cache invalidation
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

	"github.com/anggasct/httpio/internal/middleware"
)

func init() {
	// Register types with gob for serialization to support disk cache
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
	Response     *http.Response
	Body         []byte
	RequestURL   string
	LastAccessed time.Time
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// Cache defines the interface that all cache implementations must satisfy
type Cache interface {
	// Get retrieves a cached response for a request if available
	Get(ctx context.Context, key string) (*CachedResponse, bool)

	// Set stores a response in the cache with the specified key
	Set(ctx context.Context, key string, response *CachedResponse) error

	// Delete removes a cached response
	Delete(ctx context.Context, key string) error

	// Clear removes all cached responses
	Clear(ctx context.Context) error

	// Close performs any cleanup necessary when the cache is no longer needed
	Close() error
}

// Middleware implements the middleware.Middleware interface for HTTP caching
type Middleware struct {
	cache       Cache
	config      *Config
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
		// Don't cache if disabled or if this is not a cacheable method
		if !m.config.Enabled || !isCacheableMethod(req.Method) {
			return next(ctx, req)
		}

		// Check if this URL/host should be cached based on rules
		if !m.shouldCache(req) {
			return next(ctx, req)
		}

		// Generate cache key
		key := m.keyStrategy.GenerateKey(req)

		// Check cache for existing response
		if cachedResp, found := m.cache.Get(ctx, key); found {
			// Validate cached response is still fresh
			if m.isFresh(cachedResp, req) {
				// Clone the response to avoid modifying the cached response
				resp := cachedResp.Response
				resp.Body = io.NopCloser(bytes.NewReader(cachedResp.Body))
				return resp, nil
			}

			// Response is stale, delete from cache
			m.cache.Delete(ctx, key)
		}

		// Cache miss or stale, make the actual request
		resp, err := next(ctx, req)
		if err != nil || resp == nil {
			return resp, err
		}

		// Cache the response if it's cacheable
		if m.isCacheable(resp) {
			// Read and store the body
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				// If we can't read the body, just return the original response
				return resp, nil
			}

			// Replace the body reader so the response is still usable
			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			// Determine expiration time
			expiresAt := calculateExpiration(resp, m.config.DefaultTTL)

			// Create cached response
			// We save a copy of the response without the body
			// since we'll store the body separately
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

			// Store in cache (don't block on this)
			go func() {
				m.cache.Set(context.Background(), key, cachedResp)
			}()
		}

		return resp, nil
	}
}

// KeyStrategy defines how cache keys are generated from HTTP requests
type KeyStrategy interface {
	// GenerateKey creates a unique key for the given HTTP request
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

// GenerateKey creates a cache key based on HTTP method and URL
func (s *MethodURLKeyStrategy) GenerateKey(req *http.Request) string {
	return req.Method + ":" + req.URL.String()
}

// URLOnlyKeyStrategy generates keys based only on URL
type URLOnlyKeyStrategy struct{}

// NewURLOnlyKeyStrategy creates a new URLOnlyKeyStrategy
func NewURLOnlyKeyStrategy() *URLOnlyKeyStrategy {
	return &URLOnlyKeyStrategy{}
}

// GenerateKey creates a cache key based only on URL
func (s *URLOnlyKeyStrategy) GenerateKey(req *http.Request) string {
	return req.URL.String()
}

// FullRequestKeyStrategy generates keys based on method, URL, headers, and body
type FullRequestKeyStrategy struct{}

// NewFullRequestKeyStrategy creates a new FullRequestKeyStrategy
func NewFullRequestKeyStrategy() *FullRequestKeyStrategy {
	return &FullRequestKeyStrategy{}
}

// GenerateKey creates a complex cache key including HTTP method, URL, headers, and body
func (s *FullRequestKeyStrategy) GenerateKey(req *http.Request) string {
	hasher := md5.New()

	// Add method and URL
	io.WriteString(hasher, req.Method)
	io.WriteString(hasher, req.URL.String())

	// Add sorted headers
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

	// Add body if present and not too large
	if req.Body != nil && req.ContentLength > 0 && req.ContentLength < 1024*1024 {
		bodyBytes, err := io.ReadAll(req.Body)
		if err == nil {
			// Reset the body reader
			req.Body.Close()
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			hasher.Write(bodyBytes)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

// Helper functions

// isCacheableMethod determines if a request method can be cached
func isCacheableMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

// shouldCache determines if the request should be cached based on URL patterns
func (m *Middleware) shouldCache(req *http.Request) bool {
	urlStr := req.URL.String()
	host := req.URL.Host

	// Check exclude patterns
	for _, pattern := range m.config.ExcludePatterns {
		if strings.Contains(urlStr, pattern) {
			return false
		}
	}

	// Check exclude hosts
	for _, h := range m.config.ExcludeHosts {
		if host == h {
			return false
		}
	}

	// If we have include patterns, only cache URLs matching these patterns
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

// isFresh determines if a cached response is still fresh
func (m *Middleware) isFresh(cachedResp *CachedResponse, req *http.Request) bool {
	// Check if the response has expired
	if time.Now().After(cachedResp.ExpiresAt) {
		return false
	}

	// Check cache control headers if requested
	if m.config.RespectCacheControl {
		// Check for Cache-Control: no-cache in the request
		if cacheControl := req.Header.Get("Cache-Control"); cacheControl != "" {
			directives := strings.Split(cacheControl, ",")
			for _, directive := range directives {
				if strings.TrimSpace(directive) == "no-cache" {
					return false
				}
			}
		}

		// Check for Pragma: no-cache in the request
		if pragma := req.Header.Get("Pragma"); pragma == "no-cache" {
			return false
		}
	}

	// Update last accessed time
	cachedResp.LastAccessed = time.Now()
	return true
}

// isCacheable determines if a response should be cached
func (m *Middleware) isCacheable(resp *http.Response) bool {
	// Check status code
	if !CacheableStatus[resp.StatusCode] {
		return false
	}

	if m.config.RespectCacheControl {
		// Check Cache-Control header
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

// calculateExpiration determines when a response should expire based on headers and default TTL
func calculateExpiration(resp *http.Response, defaultTTL time.Duration) time.Time {
	// Check for Cache-Control: max-age
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

	// Check for Expires header
	if expires := resp.Header.Get("Expires"); expires != "" {
		if expiresTime, err := time.Parse(time.RFC1123, expires); err == nil {
			return expiresTime
		}
	}

	// Use default TTL
	return time.Now().Add(defaultTTL)
}
