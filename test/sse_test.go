package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anggasct/httpio/internal/client"
)

type testSSEHandler struct {
	events []client.Event
	opened bool
	closed bool
}

func (h *testSSEHandler) OnEvent(event client.Event) error {
	h.events = append(h.events, event)
	return nil
}

func (h *testSSEHandler) OnOpen() error {
	h.opened = true
	return nil
}

func (h *testSSEHandler) OnClose() error {
	h.closed = true
	return nil
}

func TestStreamSSE(t *testing.T) {
	sseData := `event: message
data: Hello World

event: notification
data: First line
data: Second line

id: 123
event: update
data: {"status": "ok"}
retry: 3000

`

	reader := io.NopCloser(strings.NewReader(sseData))
	handler := &testSSEHandler{}

	err := client.StreamSSE(reader, handler)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !handler.opened {
		t.Error("Expected OnOpen to be called")
	}

	if !handler.closed {
		t.Error("Expected OnClose to be called")
	}

	if len(handler.events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(handler.events))
	}

	// Check first event
	event1 := handler.events[0]
	if event1.Event != "message" {
		t.Errorf("Expected first event type to be 'message', got %s", event1.Event)
	}
	if event1.Data != "Hello World" {
		t.Errorf("Expected first event data to be 'Hello World', got %s", event1.Data)
	}

	// Check second event (multi-line data)
	event2 := handler.events[1]
	if event2.Event != "notification" {
		t.Errorf("Expected second event type to be 'notification', got %s", event2.Event)
	}
	if event2.Data != "First line\nSecond line" {
		t.Errorf("Expected second event data to be multi-line, got %s", event2.Data)
	}

	// Check third event (with ID and retry)
	event3 := handler.events[2]
	if event3.ID != "123" {
		t.Errorf("Expected third event ID to be '123', got %s", event3.ID)
	}
	if event3.Event != "update" {
		t.Errorf("Expected third event type to be 'update', got %s", event3.Event)
	}
	if event3.Data != `{"status": "ok"}` {
		t.Errorf("Expected third event data to be JSON, got %s", event3.Data)
	}
	if event3.Retry != 3000 {
		t.Errorf("Expected third event retry to be 3000, got %d", event3.Retry)
	}
}

func TestStreamSSEWithComments(t *testing.T) {
	sseData := `: This is a comment
event: test
: Another comment
data: test data

`

	reader := io.NopCloser(strings.NewReader(sseData))
	handler := &testSSEHandler{}

	err := client.StreamSSE(reader, handler)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(handler.events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(handler.events))
	}

	event := handler.events[0]
	if event.Event != "test" {
		t.Errorf("Expected event type to be 'test', got %s", event.Event)
	}
	if event.Data != "test data" {
		t.Errorf("Expected event data to be 'test data', got %s", event.Data)
	}
}

func TestEventHandlerFunc(t *testing.T) {
	var receivedEvent client.Event

	handlerFunc := client.EventHandlerFunc(func(event client.Event) error {
		receivedEvent = event
		return nil
	})

	sseData := `event: test
data: test data

`

	reader := io.NopCloser(strings.NewReader(sseData))

	err := client.StreamSSE(reader, handlerFunc)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if receivedEvent.Event != "test" {
		t.Errorf("Expected event type to be 'test', got %s", receivedEvent.Event)
	}
	if receivedEvent.Data != "test data" {
		t.Errorf("Expected event data to be 'test data', got %s", receivedEvent.Data)
	}
}

func TestEventFullHandlerFunc(t *testing.T) {
	var events []client.Event
	opened := false
	closed := false

	handler := &client.EventFullHandlerFunc{
		OnEventFunc: func(event client.Event) error {
			events = append(events, event)
			return nil
		},
		OnOpenFunc: func() error {
			opened = true
			return nil
		},
		OnCloseFunc: func() error {
			closed = true
			return nil
		},
	}

	sseData := `data: test

`

	reader := io.NopCloser(strings.NewReader(sseData))

	err := client.StreamSSE(reader, handler)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !opened {
		t.Error("Expected OnOpen to be called")
	}

	if !closed {
		t.Error("Expected OnClose to be called")
	}

	if len(events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(events))
	}

	if events[0].Data != "test" {
		t.Errorf("Expected event data to be 'test', got %s", events[0].Data)
	}
}

func TestStreamSSEWithInvalidRetry(t *testing.T) {
	sseData := `retry: invalid
data: test

`

	reader := io.NopCloser(strings.NewReader(sseData))
	handler := &testSSEHandler{}

	err := client.StreamSSE(reader, handler)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(handler.events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(handler.events))
	}

	// Retry should remain 0 for invalid value
	if handler.events[0].Retry != 0 {
		t.Errorf("Expected retry to be 0 for invalid value, got %d", handler.events[0].Retry)
	}
}

func TestResponseStreamSSE(t *testing.T) {
	sseData := `event: test
data: Hello from server

`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Write([]byte(sseData))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	response := &client.Response{Response: resp}
	handler := &testSSEHandler{}

	err = response.StreamSSE(handler)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(handler.events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(handler.events))
	}

	event := handler.events[0]
	if event.Event != "test" {
		t.Errorf("Expected event type to be 'test', got %s", event.Event)
	}
	if event.Data != "Hello from server" {
		t.Errorf("Expected event data to be 'Hello from server', got %s", event.Data)
	}
}
