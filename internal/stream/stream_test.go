package stream

import (
	"testing"
)

func TestStreamOptions(t *testing.T) {
	opts := defaultStreamOptions()

	// Test default values
	if opts.buffSize != 4096 {
		t.Errorf("Expected default buffer size 4096, got %d", opts.buffSize)
	}

	if opts.delimiterStr != "\n" {
		t.Errorf("Expected default delimiter \\n, got %s", opts.delimiterStr)
	}

	if opts.contentType != "" {
		t.Errorf("Expected default content type empty string, got %s", opts.contentType)
	}

	// Test option setters
	WithBufferSize(8192)(opts)
	if opts.buffSize != 8192 {
		t.Errorf("Expected buffer size 8192, got %d", opts.buffSize)
	}

	WithDelimiter(",")(opts)
	if opts.delimiterStr != "," {
		t.Errorf("Expected delimiter ,, got %s", opts.delimiterStr)
	}

	WithContentType("application/json")(opts)
	if opts.contentType != "application/json" {
		t.Errorf("Expected content type application/json, got %s", opts.contentType)
	}
}

func TestByteDelimiter(t *testing.T) {
	opts := defaultStreamOptions()
	WithByteDelimiter(byte('|'))(opts)

	if opts.delimiterStr != "|" {
		t.Errorf("Expected delimiter |, got %s", opts.delimiterStr)
	}
}
