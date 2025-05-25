package stream

import (
	"io"
	"strings"
	"testing"
)

// TestSSEStandardCompliance tests SSE implementation against W3C standards
func TestSSEStandardCompliance(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []SSEEvent
	}{
		{
			name: "Basic data field only",
			input: `data: Hello World

`,
			expected: []SSEEvent{
				{Event: "message", Data: "Hello World", ID: "", Retry: 0},
			},
		},
		{
			name: "Event type with data",
			input: `event: chat
data: Hello from chat

`,
			expected: []SSEEvent{
				{Event: "chat", Data: "Hello from chat", ID: "", Retry: 0},
			},
		},
		{
			name: "Multi-line data",
			input: `event: multiline
data: First line
data: Second line
data: Third line

`,
			expected: []SSEEvent{
				{Event: "multiline", Data: "First line\nSecond line\nThird line", ID: "", Retry: 0},
			},
		},
		{
			name: "Event with ID",
			input: `event: update
data: System updated
id: 42

`,
			expected: []SSEEvent{
				{Event: "update", Data: "System updated", ID: "42", Retry: 0},
			},
		},
		{
			name: "Event with retry",
			input: `event: heartbeat
data: ping
retry: 5000

`,
			expected: []SSEEvent{
				{Event: "heartbeat", Data: "ping", ID: "", Retry: 5000},
			},
		},
		{
			name: "Comments should be ignored",
			input: `: This is a comment
data: Hello World

`,
			expected: []SSEEvent{
				{Event: "message", Data: "Hello World", ID: "", Retry: 0},
			},
		},
		{
			name: "Empty data field",
			input: `data:

`,
			expected: []SSEEvent{
				{Event: "message", Data: "", ID: "", Retry: 0},
			},
		},
		{
			name: "Multiple events",
			input: `event: first
data: First event

event: second
data: Second event

`,
			expected: []SSEEvent{
				{Event: "first", Data: "First event", ID: "", Retry: 0},
				{Event: "second", Data: "Second event", ID: "", Retry: 0},
			},
		},
		{
			name: "Field order independence",
			input: `id: 123
data: Hello
event: test

`,
			expected: []SSEEvent{
				{Event: "test", Data: "Hello", ID: "123", Retry: 0},
			},
		},
		{
			name: "Invalid retry value should be ignored",
			input: `event: test
data: Hello
retry: invalid

`,
			expected: []SSEEvent{
				{Event: "test", Data: "Hello", ID: "", Retry: 0},
			},
		},
		{
			name: "Negative retry value should be ignored",
			input: `event: test
data: Hello
retry: -100

`,
			expected: []SSEEvent{
				{Event: "test", Data: "Hello", ID: "", Retry: 0},
			},
		},
		{
			name: "Unknown fields should be ignored",
			input: `unknown: value
event: test
data: Hello
custom: field

`,
			expected: []SSEEvent{
				{Event: "test", Data: "Hello", ID: "", Retry: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			var events []SSEEvent

			err := streamSSE(io.NopCloser(reader), EventSourceHandler(func(event SSEEvent) error {
				events = append(events, event)
				return nil
			}))

			if err != nil {
				t.Fatalf("streamSSEExtended returned error: %v", err)
			}

			if len(events) != len(tt.expected) {
				t.Fatalf("Expected %d events, got %d", len(tt.expected), len(events))
			}

			for i, expected := range tt.expected {
				actual := events[i]
				if actual.Event != expected.Event {
					t.Errorf("Event[%d].Event: expected %q, got %q", i, expected.Event, actual.Event)
				}
				if actual.Data != expected.Data {
					t.Errorf("Event[%d].Data: expected %q, got %q", i, expected.Data, actual.Data)
				}
				if actual.ID != expected.ID {
					t.Errorf("Event[%d].ID: expected %q, got %q", i, expected.ID, actual.ID)
				}
				if actual.Retry != expected.Retry {
					t.Errorf("Event[%d].Retry: expected %d, got %d", i, expected.Retry, actual.Retry)
				}
			}
		})
	}
}

// TestSSEEdgeCases tests edge cases and error conditions
func TestSSEEdgeCases(t *testing.T) {
	t.Run("Empty stream", func(t *testing.T) {
		reader := strings.NewReader("")
		var events []SSEEvent

		err := streamSSE(io.NopCloser(reader), EventSourceHandler(func(event SSEEvent) error {
			events = append(events, event)
			return nil
		}))

		if err != nil {
			t.Fatalf("Empty stream should not return error: %v", err)
		}

		if len(events) != 0 {
			t.Fatalf("Empty stream should produce no events, got %d", len(events))
		}
	})

	t.Run("Stream without final newline", func(t *testing.T) {
		reader := strings.NewReader("data: Hello World")
		var events []SSEEvent

		err := streamSSE(io.NopCloser(reader), EventSourceHandler(func(event SSEEvent) error {
			events = append(events, event)
			return nil
		}))

		if err != nil {
			t.Fatalf("Stream without final newline should not return error: %v", err)
		}

		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}

		if events[0].Data != "Hello World" {
			t.Errorf("Expected data 'Hello World', got %q", events[0].Data)
		}
	})

	t.Run("Only empty lines", func(t *testing.T) {
		reader := strings.NewReader("\n\n\n")
		var events []SSEEvent

		err := streamSSE(io.NopCloser(reader), EventSourceHandler(func(event SSEEvent) error {
			events = append(events, event)
			return nil
		}))

		if err != nil {
			t.Fatalf("Empty lines should not return error: %v", err)
		}

		if len(events) != 0 {
			t.Fatalf("Only empty lines should produce no events, got %d", len(events))
		}
	})

	t.Run("Data with colon separator", func(t *testing.T) {
		reader := strings.NewReader(`data: key:value

`)
		var events []SSEEvent

		err := streamSSE(io.NopCloser(reader), EventSourceHandler(func(event SSEEvent) error {
			events = append(events, event)
			return nil
		}))

		if err != nil {
			t.Fatalf("Data with colon should not return error: %v", err)
		}

		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}

		if events[0].Data != "key:value" {
			t.Errorf("Expected data 'key:value', got %q", events[0].Data)
		}
	})
}
