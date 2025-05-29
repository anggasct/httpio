package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anggasct/httpio/internal/client"
)

func TestStreamLines(t *testing.T) {
	data := "line 1\nline 2\nline 3"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(data))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	var lines []string
	err = client.StreamLines(response, func(line []byte) error {
		lines = append(lines, string(line))
		return nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []string{"line 1", "line 2", "line 3"}
	if len(lines) != len(expected) {
		t.Fatalf("Expected %d lines, got %d", len(expected), len(lines))
	}

	for i, line := range lines {
		if line != expected[i] {
			t.Errorf("Expected line %d to be %s, got %s", i, expected[i], line)
		}
	}
}

func TestStreamLinesWithCustomDelimiter(t *testing.T) {
	data := "chunk1|chunk2|chunk3"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(data))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	var chunks []string
	err = client.StreamLines(response, func(chunk []byte) error {
		chunks = append(chunks, string(chunk))
		return nil
	}, client.WithDelimiter("|"))

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []string{"chunk1", "chunk2", "chunk3"}
	if len(chunks) != len(expected) {
		t.Fatalf("Expected %d chunks, got %d", len(expected), len(chunks))
	}

	for i, chunk := range chunks {
		if chunk != expected[i] {
			t.Errorf("Expected chunk %d to be %s, got %s", i, expected[i], chunk)
		}
	}
}

func TestStreamLinesWithContentType(t *testing.T) {
	data := "line 1\nline 2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(data))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	var lines []string
	err = client.StreamLines(response, func(line []byte) error {
		lines = append(lines, string(line))
		return nil
	}, client.WithContentType("text/plain"))

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestStreamLinesWithWrongContentType(t *testing.T) {
	data := "line 1\nline 2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(data))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	err = client.StreamLines(response, func(line []byte) error {
		return nil
	}, client.WithContentType("text/plain"))

	if err == nil {
		t.Error("Expected error for wrong content type")
	}
}

func TestStreamJSON(t *testing.T) {
	jsonData := `{"name": "Alice"}
{"name": "Bob"}
{"name": "Charlie"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonData))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	var objects []json.RawMessage
	err = client.StreamJSON(response, func(raw json.RawMessage) error {
		objects = append(objects, raw)
		return nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(objects) != 3 {
		t.Errorf("Expected 3 JSON objects, got %d", len(objects))
	}

	// Verify first object
	var firstObj map[string]string
	err = json.Unmarshal(objects[0], &firstObj)
	if err != nil {
		t.Fatalf("Failed to unmarshal first object: %v", err)
	}

	if firstObj["name"] != "Alice" {
		t.Errorf("Expected first object name to be Alice, got %s", firstObj["name"])
	}
}

func TestStreamInto(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	jsonData := `{"name": "Alice", "age": 30}
{"name": "Bob", "age": 25}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonData))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	var people []Person
	err = client.StreamInto(response, func(person *Person) error {
		people = append(people, *person)
		return nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(people) != 2 {
		t.Errorf("Expected 2 people, got %d", len(people))
	}

	if people[0].Name != "Alice" || people[0].Age != 30 {
		t.Errorf("Expected first person to be Alice(30), got %s(%d)", people[0].Name, people[0].Age)
	}

	if people[1].Name != "Bob" || people[1].Age != 25 {
		t.Errorf("Expected second person to be Bob(25), got %s(%d)", people[1].Name, people[1].Age)
	}
}

func TestStreamIntoWithPointer(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
	}

	jsonData := `{"name": "Alice"}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonData))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	var people []*Person
	err = client.StreamInto(response, func(person *Person) error {
		people = append(people, person)
		return nil
	})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(people) != 1 {
		t.Errorf("Expected 1 person, got %d", len(people))
	}

	if people[0].Name != "Alice" {
		t.Errorf("Expected person name to be Alice, got %s", people[0].Name)
	}
}

func TestStreamIntoWithInvalidHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}

	// Invalid handler - not a function
	err = client.StreamInto(response, "not a function")
	if err == nil {
		t.Error("Expected error for invalid handler type")
	}

	// Reset response body
	resp, _ = http.Get(server.URL)
	response = &client.Response{Response: resp}

	// Invalid handler - wrong signature
	err = client.StreamInto(response, func() {})
	if err == nil {
		t.Error("Expected error for invalid handler signature")
	}
}

func TestStreamWithNilBody(t *testing.T) {
	response := &client.Response{
		Response: &http.Response{Body: nil},
	}

	err := client.StreamLines(response, func(line []byte) error {
		return nil
	})

	if err == nil {
		t.Error("Expected error for nil body")
	}

	err = client.StreamJSON(response, func(raw json.RawMessage) error {
		return nil
	})

	if err == nil {
		t.Error("Expected error for nil body")
	}

	err = client.StreamInto(response, func(obj interface{}) error {
		return nil
	})

	if err == nil {
		t.Error("Expected error for nil body")
	}
}

func TestWithBufferSize(t *testing.T) {
	option := client.WithBufferSize(8192)

	if option == nil {
		t.Error("Expected buffer size option to be created")
	}
}

func TestWithByteDelimiter(t *testing.T) {
	option := client.WithByteDelimiter('|')

	if option == nil {
		t.Error("Expected byte delimiter option to be created")
	}
}
