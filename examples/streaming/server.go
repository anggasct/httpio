package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Location string `json:"location"`
}

func startStreamingServer(ready chan struct{}) {
	mux := http.NewServeMux()

	mux.HandleFunc("/stream", handleRawStream)
	mux.HandleFunc("/streamlines", handleLineStream)
	mux.HandleFunc("/streamjson", handleJSONStream)
	mux.HandleFunc("/streamusers", handleUsersStream)
	mux.HandleFunc("/sse", handleSSE)
	mux.HandleFunc("/slowendless", handleSlowEndlessStream)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	close(ready)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
	}
}

func handleRawStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	for i := 1; i <= 10; i++ {
		data := make([]byte, i*100)
		for j := range data {
			data[j] = byte(65 + (i+j)%26)
		}

		w.Write(data)
		flusher.Flush()
		time.Sleep(200 * time.Millisecond)
	}
}

func handleLineStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	lines := []string{
		"This is the first line of streaming text",
		"Here's another line with some data",
		"Streaming makes it easy to process large datasets",
		"Each line can be processed independently",
		"This allows for efficient memory usage",
		"And responsive user interfaces",
		"Even with very large amounts of data",
	}

	for _, line := range lines {
		fmt.Fprintln(w, line)
		flusher.Flush()
		time.Sleep(300 * time.Millisecond)
	}
}

func handleJSONStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	objects := []map[string]interface{}{
		{"id": 1, "type": "message", "content": "Hello, streaming world!"},
		{"id": 2, "type": "status", "status": "active", "timestamp": time.Now().Unix()},
		{"id": 3, "type": "data", "values": []int{10, 20, 30, 40, 50}},
		{"id": 4, "type": "message", "content": "Streaming is fun!"},
		{"id": 5, "type": "status", "status": "inactive", "timestamp": time.Now().Unix()},
	}

	for _, obj := range objects {
		data, _ := json.Marshal(obj)
		w.Write(data)
		w.Write([]byte("\n"))
		flusher.Flush()
		time.Sleep(500 * time.Millisecond)
	}
}

func handleUsersStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	users := []User{
		{ID: 1, Name: "Alice Johnson", Email: "alice@example.com", Location: "New York"},
		{ID: 2, Name: "Bob Smith", Email: "bob@example.com", Location: "Los Angeles"},
		{ID: 3, Name: "Carol Williams", Email: "carol@example.com", Location: "Chicago"},
		{ID: 4, Name: "Dave Brown", Email: "dave@example.com", Location: "Houston"},
		{ID: 5, Name: "Eve Davis", Email: "eve@example.com", Location: "Phoenix"},
	}

	for _, user := range users {
		data, _ := json.Marshal(user)
		w.Write(data)
		w.Write([]byte("\n"))
		flusher.Flush()
		time.Sleep(600 * time.Millisecond)
	}
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	events := []struct {
		eventType string
		data      string
	}{
		{"", "Default message event type"},
		{"update", "System update scheduled for tomorrow"},
		{"alert", "High CPU usage detected"},
		{"notification", "You have 3 new messages"},
		{"update", "Update completed successfully"},
	}

	for _, evt := range events {
		if evt.eventType != "" {
			fmt.Fprintf(w, "event: %s\n", evt.eventType)
		}
		fmt.Fprintf(w, "data: %s\n\n", evt.data)
		flusher.Flush()
		time.Sleep(700 * time.Millisecond)
	}
}

func handleSlowEndlessStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	counter := 1
	for {
		select {
		case <-r.Context().Done():
			return
		default:
			fmt.Fprintf(w, "This is slow data item #%d\n", counter)
			flusher.Flush()
			counter++
			time.Sleep(500 * time.Millisecond)
		}
	}
}
