package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/anggasct/httpio/internal/middleware"
	"github.com/google/uuid"
)

// ContextKey type for context value storage
type ContextKey string

const (
	// RequestIDKey is the context key for storing request IDs
	RequestIDKey ContextKey = "request_id"
	// MaxBodyLogSize limits the body size in logs
	MaxBodyLogSize = 10 * 1024 // 10KB
)

// LogLevel defines logging verbosity
type LogLevel int

const (
	// LevelNone disables logging
	LevelNone LogLevel = iota
	// LevelError logs only errors
	LevelError
	// LevelInfo logs basic request/response info without headers/body
	LevelInfo
	// LevelDebug logs with headers but no bodies
	LevelDebug
	// LevelTrace logs everything including bodies
	LevelTrace
)

// String returns a string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LevelNone:
		return "NONE"
	case LevelError:
		return "ERROR"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	case LevelTrace:
		return "TRACE"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", l)
	}
}

// OutputFormat defines how logs are formatted
type OutputFormat int

const (
	// FormatText outputs logs in human-readable text format
	FormatText OutputFormat = iota
	// FormatJSON outputs logs in JSON format
	FormatJSON
)

// Logger interface allows for integration with any logging library
type Logger interface {
	Log(ctx context.Context, level LogLevel, msg string, fields map[string]interface{})
}

// StandardLogger is a simple logger that writes to stdout
type StandardLogger struct {
	Level  LogLevel
	Format OutputFormat
}

// Log implements the Logger interface
func (l *StandardLogger) Log(ctx context.Context, level LogLevel, msg string, fields map[string]interface{}) {
	if level > l.Level {
		return
	}

	if l.Format == FormatJSON {
		data := map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
			"level":     level.String(),
			"message":   msg,
		}
		// Add all fields to the JSON output
		for k, v := range fields {
			data[k] = v
		}

		// Add request ID if available
		if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
			data["request_id"] = reqID
		}

		jsonData, _ := json.Marshal(data)
		fmt.Println(string(jsonData))
	} else {
		reqID := "unknown"
		if id, ok := ctx.Value(RequestIDKey).(string); ok {
			reqID = id
		}

		fmt.Printf("[%s] [%s] [%s] %s",
			time.Now().Format(time.RFC3339),
			level.String(),
			reqID,
			msg)

		if len(fields) > 0 {
			fmt.Printf(" | %v\n", fields)
		} else {
			fmt.Println()
		}
	}
}

// Config holds the configuration for the logger middleware
type Config struct {
	// Logger is the logger implementation to use
	Logger Logger
	// Level determines the logging verbosity
	Level LogLevel
	// Format defines the output format (text or JSON)
	Format OutputFormat
	// RequestIDGenerator creates unique request identifiers
	RequestIDGenerator func() string
	// RequestIDHeader is the header name for propagating request IDs
	RequestIDHeader string
	// SensitiveHeaders are headers that should be redacted
	SensitiveHeaders []string
	// SkipPaths are URL paths that should not be logged
	SkipPaths []string
	// SensitiveFields are JSON fields that should be redacted in bodies
	SensitiveFields []string
	// EnableSampling enables log sampling to reduce volume
	EnableSampling bool
	// SampleRate defines the log sampling rate (1.0 = 100%)
	SampleRate float64
	// PropagateRequestID controls whether to propagate request ID
	PropagateRequestID bool
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Logger:             &StandardLogger{Level: LevelInfo, Format: FormatText},
		Level:              LevelInfo,
		Format:             FormatText,
		RequestIDGenerator: uuid.NewString,
		RequestIDHeader:    "X-Request-ID",
		SensitiveHeaders:   []string{"Authorization", "Cookie", "X-Api-Key", "Proxy-Authorization"},
		SkipPaths:          []string{"/healthz", "/ready", "/metrics"},
		SensitiveFields:    []string{"password", "token", "secret", "key", "credential", "auth"},
		EnableSampling:     false,
		SampleRate:         1.0,
		PropagateRequestID: true,
	}
}

// Middleware implements HTTP client logging
type Middleware struct {
	config *Config
}

// New creates a new logger middleware
func New(config *Config) *Middleware {
	cfg := DefaultConfig()
	if config != nil {
		if config.Logger != nil {
			cfg.Logger = config.Logger
		}
		if config.Level != 0 {
			cfg.Level = config.Level
		}
		if config.Format != 0 {
			cfg.Format = config.Format
		}
		if config.RequestIDGenerator != nil {
			cfg.RequestIDGenerator = config.RequestIDGenerator
		}
		if config.RequestIDHeader != "" {
			cfg.RequestIDHeader = config.RequestIDHeader
		}
		if len(config.SensitiveHeaders) > 0 {
			cfg.SensitiveHeaders = config.SensitiveHeaders
		}
		if len(config.SkipPaths) > 0 {
			cfg.SkipPaths = config.SkipPaths
		}
		if len(config.SensitiveFields) > 0 {
			cfg.SensitiveFields = config.SensitiveFields
		}
		if config.EnableSampling {
			cfg.EnableSampling = config.EnableSampling
			cfg.SampleRate = config.SampleRate
		}
		cfg.PropagateRequestID = config.PropagateRequestID
	}
	return &Middleware{config: cfg}
}

// WithLevel returns a middleware with the specified log level
func WithLevel(level LogLevel) *Middleware {
	return New(&Config{Level: level})
}

// WithJSON returns a middleware with JSON output format
func WithJSON() *Middleware {
	return New(&Config{Format: FormatJSON})
}

// redactHeaders returns a copy of headers with sensitive values redacted
func (m *Middleware) redactHeaders(headers http.Header) http.Header {
	result := make(http.Header)
	for name, values := range headers {
		isSensitive := false
		for _, sensitive := range m.config.SensitiveHeaders {
			if strings.EqualFold(name, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			result[name] = []string{"[REDACTED]"}
		} else {
			result[name] = values
		}
	}
	return result
}

// truncateBody returns a truncated copy of the body if it's too large
func truncateBody(body []byte) []byte {
	if len(body) <= MaxBodyLogSize {
		return body
	}
	return append(body[:MaxBodyLogSize], []byte("... (truncated)")...)
}

// redactJSONFields redacts sensitive fields in JSON bodies
func (m *Middleware) redactJSONFields(body []byte) []byte {
	if len(body) == 0 {
		return body
	}

	// Only try to parse if it seems like JSON
	if !bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) && !bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
		return body
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		// If it's not valid JSON, return as is
		return body
	}

	// Recursive function to redact fields
	var redact func(v interface{}) interface{}
	redact = func(v interface{}) interface{} {
		switch val := v.(type) {
		case map[string]interface{}:
			result := make(map[string]interface{})
			for k, v := range val {
				isSensitive := false
				for _, field := range m.config.SensitiveFields {
					if regexp.MustCompile(fmt.Sprintf("(?i)%s", field)).MatchString(k) {
						isSensitive = true
						break
					}
				}
				if isSensitive {
					result[k] = "[REDACTED]"
				} else {
					result[k] = redact(v)
				}
			}
			return result
		case []interface{}:
			result := make([]interface{}, len(val))
			for i, v := range val {
				result[i] = redact(v)
			}
			return result
		default:
			return v
		}
	}

	redacted := redact(data)
	redactedJSON, err := json.Marshal(redacted)
	if err != nil {
		return body
	}

	return redactedJSON
}

// shouldSkipLogging determines if the request should be logged
func (m *Middleware) shouldSkipLogging(path string) bool {
	for _, p := range m.config.SkipPaths {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(RequestIDKey).(string)
	return id, ok
}

// Handle implements the middleware.Handler interface
func (m *Middleware) Handle(next middleware.Handler) middleware.Handler {
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		// Skip logging if path is in the skip list
		if m.shouldSkipLogging(req.URL.Path) {
			return next(ctx, req)
		}

		// Sample logging if enabled
		if m.config.EnableSampling {
			if m.config.SampleRate < 1.0 && float64(uuid.New().ID()) > m.config.SampleRate*float64(^uint64(0)) {
				return next(ctx, req)
			}
		}

		// Generate or retrieve request ID
		requestID := ""
		if existingID := req.Header.Get(m.config.RequestIDHeader); existingID != "" {
			requestID = existingID
		} else {
			requestID = m.config.RequestIDGenerator()
			if m.config.PropagateRequestID {
				req.Header.Set(m.config.RequestIDHeader, requestID)
			}
		}

		// Store request ID in context
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		// Pre-request logging
		if m.config.Level >= LevelInfo {
			fields := map[string]interface{}{
				"method": req.Method,
				"url":    req.URL.String(),
			}

			// Add headers for debug level
			if m.config.Level >= LevelDebug {
				fields["headers"] = m.redactHeaders(req.Header)
			}

			// Add body for trace level
			if m.config.Level >= LevelTrace && req.Body != nil {
				var bodyBuffer bytes.Buffer
				req.Body, _ = duplicateBody(req.Body, &bodyBuffer)
				bodyBytes := m.redactJSONFields(bodyBuffer.Bytes())
				fields["body"] = string(truncateBody(bodyBytes))
			}

			m.config.Logger.Log(ctx, LevelInfo, "Outgoing request", fields)
		}

		start := time.Now()
		resp, err := next(ctx, req)
		duration := time.Since(start)

		// Post-request logging
		level := LevelInfo
		if err != nil || (resp != nil && resp.StatusCode >= 400) {
			level = LevelError
		}

		// Skip further logging if level is too high
		if m.config.Level < level {
			return resp, err
		}

		fields := map[string]interface{}{
			"method":   req.Method,
			"url":      req.URL.String(),
			"duration": duration.Milliseconds(),
		}

		if err != nil {
			fields["error"] = err.Error()
		}

		if resp != nil {
			fields["status"] = resp.StatusCode

			// Add headers for debug level
			if m.config.Level >= LevelDebug {
				fields["response_headers"] = m.redactHeaders(resp.Header)
			}

			// Add body for trace level
			if m.config.Level >= LevelTrace && resp.Body != nil {
				var bodyBuffer bytes.Buffer
				resp.Body, _ = duplicateBody(resp.Body, &bodyBuffer)
				bodyBytes := m.redactJSONFields(bodyBuffer.Bytes())
				fields["response_body"] = string(truncateBody(bodyBytes))
			}
		}

		logMessage := fmt.Sprintf("%s %s completed in %dms",
			req.Method, req.URL.Path, duration.Milliseconds())

		if err != nil {
			logMessage += fmt.Sprintf(" with error: %v", err)
		} else if resp != nil {
			logMessage += fmt.Sprintf(" with status: %d", resp.StatusCode)
		}

		m.config.Logger.Log(ctx, level, logMessage, fields)

		return resp, err
	}
}

// duplicateBody reads the body and restores it with a copy for further reading
func duplicateBody(body io.ReadCloser, buffer *bytes.Buffer) (io.ReadCloser, error) {
	if body == nil {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	_ = body.Close()

	buffer.Write(bodyBytes)
	return io.NopCloser(bytes.NewBuffer(bodyBytes)), nil
}

// WithContext returns a new context with the request ID
func WithContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}
