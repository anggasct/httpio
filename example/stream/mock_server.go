package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func StartMockServer(port string) {
	// Line streaming endpoint
	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		for i := 1; i <= 10; i++ {
			fmt.Fprintf(w, "Line %d\n", i)
			flusher.Flush()
			time.Sleep(500 * time.Millisecond)
		}
	})

	// JSON streaming endpoint
	http.HandleFunc("/stream-json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		for i := 1; i <= 5; i++ {
			jsonData := fmt.Sprintf(`{"id": "msg-%d", "content": "This is message %d"}`, i, i)
			fmt.Fprintf(w, "%s\n", jsonData)
			flusher.Flush()
			time.Sleep(1000 * time.Millisecond)
		}
	})

	// SSE endpoint
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		eventTypes := []string{"update", "message", "alert"}

		for i := 1; i <= 5; i++ {
			eventType := eventTypes[i%len(eventTypes)]
			fmt.Fprintf(w, "id: %d\n", i)
			fmt.Fprintf(w, "event: %s\n", eventType)
			fmt.Fprintf(w, "data: Event data for event %d\n\n", i)
			flusher.Flush()
			time.Sleep(2000 * time.Millisecond)
		}
	})

	log.Printf("Mock server started on port %s\n", port)
	log.Printf("Available endpoints:\n")
	log.Printf("  - Line streaming: http://localhost:%s/stream\n", port)
	log.Printf("  - JSON streaming: http://localhost:%s/stream-json\n", port)
	log.Printf("  - SSE: http://localhost:%s/events\n", port)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
