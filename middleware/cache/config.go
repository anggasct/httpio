package cache

import (
	"time"
)

// Config holds configuration options for the cache middleware
type Config struct {
	// Enabled determines whether caching is enabled
	Enabled bool
	// DefaultTTL is the default time-to-live for cached responses
	DefaultTTL time.Duration
	// RespectCacheControl determines whether to respect Cache-Control headers
	RespectCacheControl bool
	// IncludePatterns is a list of URL patterns to cache (if empty, all URLs are cached)
	IncludePatterns []string
	// ExcludePatterns is a list of URL patterns to exclude from caching
	ExcludePatterns []string
	// ExcludeHosts is a list of hosts to exclude from caching
	ExcludeHosts []string
	// CleanupInterval is the interval at which expired cache entries are cleaned up
	CleanupInterval time.Duration
	// KeyStrategy defines how cache keys are generated
	KeyStrategy KeyStrategyType
	// DomainTTLRules allows specifying different TTLs for different domains
	DomainTTLRules map[string]time.Duration
	// PathTTLRules allows specifying different TTLs for different URL path patterns
	PathTTLRules map[string]time.Duration
}

// DefaultConfig returns a default configuration for the cache middleware
func DefaultConfig() *Config {
	return &Config{
		Enabled:             true,
		DefaultTTL:          10 * time.Minute,
		RespectCacheControl: true,
		KeyStrategy:         KeyByURLAndMethod,
		CleanupInterval:     30 * time.Minute,
		IncludePatterns:     []string{},
		ExcludePatterns:     []string{},
		ExcludeHosts:        []string{},
		DomainTTLRules:      make(map[string]time.Duration),
		PathTTLRules:        make(map[string]time.Duration),
	}
}

// WithEnabled sets whether caching is enabled
func (c *Config) WithEnabled(enabled bool) *Config {
	c.Enabled = enabled
	return c
}

// WithDefaultTTL sets the default time-to-live for cached responses
func (c *Config) WithDefaultTTL(ttl time.Duration) *Config {
	c.DefaultTTL = ttl
	return c
}

// WithRespectCacheControl sets whether to respect Cache-Control headers
func (c *Config) WithRespectCacheControl(respect bool) *Config {
	c.RespectCacheControl = respect
	return c
}

// WithIncludePatterns sets URL patterns to include in caching
func (c *Config) WithIncludePatterns(patterns ...string) *Config {
	c.IncludePatterns = patterns
	return c
}

// WithExcludePatterns sets URL patterns to exclude from caching
func (c *Config) WithExcludePatterns(patterns ...string) *Config {
	c.ExcludePatterns = patterns
	return c
}

// WithExcludeHosts sets hosts to exclude from caching
func (c *Config) WithExcludeHosts(hosts ...string) *Config {
	c.ExcludeHosts = hosts
	return c
}

// WithKeyStrategy sets the key generation strategy
func (c *Config) WithKeyStrategy(strategy KeyStrategyType) *Config {
	c.KeyStrategy = strategy
	return c
}

// WithDomainTTL sets a specific TTL for a domain
func (c *Config) WithDomainTTL(domain string, ttl time.Duration) *Config {
	c.DomainTTLRules[domain] = ttl
	return c
}

// WithPathTTL sets a specific TTL for a URL path pattern
func (c *Config) WithPathTTL(pathPattern string, ttl time.Duration) *Config {
	c.PathTTLRules[pathPattern] = ttl
	return c
}

// WithCleanupInterval sets the interval for cleaning up expired cache entries
func (c *Config) WithCleanupInterval(interval time.Duration) *Config {
	c.CleanupInterval = interval
	return c
}
