package test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/anggasct/httpio"
	"github.com/anggasct/httpio/middleware/logger"
)

// mockLogger implements the Logger interface for testing
type mockLogger struct {
	buf    *bytes.Buffer
	level  logger.LogLevel
	format logger.OutputFormat
}

func (m *mockLogger) Log(ctx context.Context, level logger.LogLevel, msg string, fields map[string]interface{}) {
	if level > m.level {
		return
	}

	m.buf.WriteString(msg)
	if len(fields) > 0 {
		m.buf.WriteString(" | ")
		for k := range fields {
			m.buf.WriteString(k + "=value ")
		}
	}
	m.buf.WriteString("\n")
}

func TestLoggerMiddleware(t *testing.T) {
	var buf bytes.Buffer
	mockLog := &mockLogger{
		buf:    &buf,
		level:  logger.LevelDebug,
		format: logger.FormatText,
	}

	config := &logger.Config{
		Logger: mockLog,
		Level:  logger.LevelDebug,
	}

	loggerMiddleware := logger.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
		}, nil
	}

	handler := loggerMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logOutput := buf.String()
	if len(logOutput) == 0 {
		t.Error("Expected some log output")
	}
}

func TestLoggerLevels(t *testing.T) {
	var buf bytes.Buffer
	mockLog := &mockLogger{
		buf:    &buf,
		level:  logger.LevelInfo,
		format: logger.FormatText,
	}

	// Test Info level
	config := &logger.Config{
		Logger: mockLog,
		Level:  logger.LevelInfo,
	}

	loggerMiddleware := logger.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
		}, nil
	}

	handler := loggerMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logOutput := buf.String()
	if len(logOutput) == 0 {
		t.Error("Expected some log output for Info level")
	}
}

func TestLoggerTrace(t *testing.T) {
	var buf bytes.Buffer
	mockLog := &mockLogger{
		buf:    &buf,
		level:  logger.LevelTrace,
		format: logger.FormatText,
	}

	config := &logger.Config{
		Logger: mockLog,
		Level:  logger.LevelTrace,
	}

	loggerMiddleware := logger.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
		}, nil
	}

	handler := loggerMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logOutput := buf.String()
	if len(logOutput) == 0 {
		t.Error("Expected some log output for Trace level")
	}
}

func TestLoggerError(t *testing.T) {
	var buf bytes.Buffer
	mockLog := &mockLogger{
		buf:    &buf,
		level:  logger.LevelError,
		format: logger.FormatText,
	}

	config := &logger.Config{
		Logger: mockLog,
		Level:  logger.LevelError,
	}

	loggerMiddleware := logger.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 500,
			Header:     make(http.Header),
		}, nil
	}

	handler := loggerMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logOutput := buf.String()
	if len(logOutput) == 0 {
		t.Error("Expected some log output for error response")
	}
}

func TestLoggerJSONFormat(t *testing.T) {
	var buf bytes.Buffer
	mockLog := &mockLogger{
		buf:    &buf,
		level:  logger.LevelDebug,
		format: logger.FormatJSON,
	}

	config := &logger.Config{
		Logger: mockLog,
		Level:  logger.LevelDebug,
		Format: logger.FormatJSON,
	}

	loggerMiddleware := logger.New(config)

	baseHandler := func(ctx context.Context, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Header:     make(http.Header),
		}, nil
	}

	handler := loggerMiddleware.Handle(baseHandler)

	req, _ := http.NewRequest("GET", "http://example.com/test", nil)

	_, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	logOutput := buf.String()
	if len(logOutput) == 0 {
		t.Error("Expected some log output")
	}
}

func TestLoggerConfiguration(t *testing.T) {
	config := &logger.Config{
		Logger: &logger.StandardLogger{},
		Level:  logger.LevelDebug,
		Format: logger.FormatText,
	}

	// Test that we can create middleware with config
	middleware := logger.New(config)
	if middleware == nil {
		t.Error("Expected middleware to be created")
	}
}

func TestLoggerIntegration(t *testing.T) {
	var buf bytes.Buffer
	mockLog := &mockLogger{
		buf:    &buf,
		level:  logger.LevelDebug,
		format: logger.FormatText,
	}

	loggerMiddleware := logger.New(&logger.Config{
		Logger: mockLog,
		Level:  logger.LevelDebug,
	})

	// Test with actual HTTP client
	client := httpio.New().WithMiddleware(loggerMiddleware)

	if client == nil {
		t.Error("Expected client to be created")
	}
}
