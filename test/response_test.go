package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anggasct/httpio/internal/client"
)

func TestResponseBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	bytes, err := response.Bytes()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if string(bytes) != "test response" {
		t.Errorf("Expected 'test response', got %s", string(bytes))
	}
}

func TestResponseString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	str, err := response.String()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if str != "test response" {
		t.Errorf("Expected 'test response', got %s", str)
	}
}

func TestResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "test"}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	var data map[string]string
	err = response.JSON(&data)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if data["message"] != "test" {
		t.Errorf("Expected message 'test', got %s", data["message"])
	}
}

func TestResponseWriteTo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	var buf strings.Builder
	n, err := response.WriteTo(&buf)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if n != int64(len("test response")) {
		t.Errorf("Expected %d bytes written, got %d", len("test response"), n)
	}

	if buf.String() != "test response" {
		t.Errorf("Expected 'test response', got %s", buf.String())
	}
}

func TestResponsePipe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
		w.Write([]byte(" response"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	ch := make(chan []byte, 10)

	go func() {
		response.Pipe(ch)
	}()

	var result []byte
	for chunk := range ch {
		result = append(result, chunk...)
	}

	if string(result) != "test response" {
		t.Errorf("Expected 'test response', got %s", string(result))
	}
}

func TestResponseClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	err = response.Close()
	if err != nil {
		t.Fatalf("Expected no error closing response, got %v", err)
	}

	// Try to read after close - should fail
	_, err = io.ReadAll(response.Body)
	if err == nil {
		t.Error("Expected error reading from closed body, got nil")
	}
}
