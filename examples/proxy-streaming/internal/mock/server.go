package mock

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type MockUser struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Status   string `json:"status"`
	JoinedAt string `json:"joined_at"`
}

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

func (ms *Server) StreamDataHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	log.Printf("Mock: Starting text stream for query: %s", query)

	for i := 1; i <= 10; i++ {
		data := fmt.Sprintf("Line %d: Data for query '%s' - %s", i, query, time.Now().Format("15:04:05"))

		_, err := w.Write([]byte(data + "\n"))
		if err != nil {
			log.Printf("Mock: Error writing data: %v", err)
			return
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		time.Sleep(500 * time.Millisecond)
	}

	log.Println("Mock: Text stream completed")
}

func (ms *Server) StreamJSONHandler(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	log.Printf("Mock: Starting JSON stream for category: %s", category)

	for i := 1; i <= 8; i++ {
		item := map[string]interface{}{
			"id":        i,
			"category":  category,
			"title":     fmt.Sprintf("Item %d in %s", i, category),
			"content":   fmt.Sprintf("This is the content for item %d", i),
			"timestamp": time.Now().Format(time.RFC3339),
			"score":     float64(i) * 1.5,
		}

		jsonData, err := json.Marshal(item)
		if err != nil {
			log.Printf("Mock: Error marshaling JSON: %v", err)
			continue
		}

		_, err = w.Write(jsonData)
		if err != nil {
			log.Printf("Mock: Error writing JSON: %v", err)
			return
		}

		_, err = w.Write([]byte("\n"))
		if err != nil {
			log.Printf("Mock: Error writing newline: %v", err)
			return
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		time.Sleep(750 * time.Millisecond)
	}

	log.Println("Mock: JSON stream completed")
}

func (ms *Server) SSEHandler(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	log.Printf("Mock: Starting SSE stream for topic: %s", topic)

	events := []struct {
		event string
		data  string
	}{
		{"start", fmt.Sprintf("Started listening to topic: %s", topic)},
		{"update", "First update message"},
		{"data", `{"type": "notification", "message": "New data available"}`},
		{"update", "Second update message"},
		{"data", `{"type": "alert", "message": "Important notification"}`},
		{"end", "Stream ending"},
	}

	for i, event := range events {
		if event.event != "message" {
			_, err := fmt.Fprintf(w, "event: %s\n", event.event)
			if err != nil {
				log.Printf("Mock: Error writing SSE event: %v", err)
				return
			}
		}

		_, err := fmt.Fprintf(w, "data: %s\n", event.data)
		if err != nil {
			log.Printf("Mock: Error writing SSE data: %v", err)
			return
		}

		_, err = fmt.Fprintf(w, "id: %d\n", i+1)
		if err != nil {
			log.Printf("Mock: Error writing SSE id: %v", err)
			return
		}

		_, err = w.Write([]byte("\n"))
		if err != nil {
			log.Printf("Mock: Error writing SSE separator: %v", err)
			return
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		time.Sleep(1 * time.Second)
	}

	log.Println("Mock: SSE stream completed")
}

func (ms *Server) StreamUsersHandler(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	log.Printf("Mock: Starting user stream for status: %s", status)

	users := []MockUser{
		{1, "Alice Johnson", "alice@example.com", "active", "2023-01-15T10:30:00Z"},
		{2, "Bob Smith", "bob@example.com", "active", "2023-02-20T14:15:00Z"},
		{3, "Charlie Brown", "charlie@example.com", "banned", "2023-03-10T09:45:00Z"},
		{4, "Diana Prince", "diana@example.com", "active", "2023-04-05T16:20:00Z"},
		{5, "Eve Wilson", "eve@example.com", "inactive", "2023-05-12T11:10:00Z"},
		{6, "Frank Miller", "frank@example.com", "active", "2023-06-18T13:25:00Z"},
	}

	for _, user := range users {
		if status != "" && user.Status != status {
			continue
		}

		jsonData, err := json.Marshal(user)
		if err != nil {
			log.Printf("Mock: Error marshaling user: %v", err)
			continue
		}

		_, err = w.Write(jsonData)
		if err != nil {
			log.Printf("Mock: Error writing user data: %v", err)
			return
		}

		_, err = w.Write([]byte("\n"))
		if err != nil {
			log.Printf("Mock: Error writing newline: %v", err)
			return
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		time.Sleep(600 * time.Millisecond)
	}

	log.Println("Mock: User stream completed")
}

func (ms *Server) StreamRawHandler(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	log.Printf("Mock: Starting raw stream for source: %s", source)

	for i := 1; i <= 6; i++ {
		rawData := map[string]interface{}{
			"id":            i,
			"value":         fmt.Sprintf("raw-value-%d", i),
			"metric":        i * 10,
			"source":        source,
			"raw_timestamp": time.Now().Unix(),
		}

		jsonData, err := json.Marshal(rawData)
		if err != nil {
			log.Printf("Mock: Error marshaling raw data: %v", err)
			continue
		}

		_, err = w.Write(jsonData)
		if err != nil {
			log.Printf("Mock: Error writing raw data: %v", err)
			return
		}

		_, err = w.Write([]byte("\n"))
		if err != nil {
			log.Printf("Mock: Error writing newline: %v", err)
			return
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		time.Sleep(800 * time.Millisecond)
	}

	log.Println("Mock: Raw stream completed")
}

func (ms *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	}
	json.NewEncoder(w).Encode(response)
}

func (ms *Server) InfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Mock Streaming API Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { margin: 15px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
        .method { color: #28a745; font-weight: bold; }
        .path { font-family: monospace; background: #f8f9fa; padding: 2px 4px; }
        h1 { color: #333; }
        h3 { color: #666; margin-bottom: 5px; }
        p { margin: 5px 0; }
    </style>
</head>
<body>
    <h1>Mock Streaming API Server</h1>
    <p>This server provides mock streaming endpoints for testing the proxy service.</p>
    
    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/stream/data</span></h3>
        <p>Mock text streaming endpoint</p>
        <p><strong>Parameters:</strong> query (string)</p>
        <p><strong>Example:</strong> <code>/stream/data?query=test</code></p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/stream/json</span></h3>
        <p>Mock NDJSON streaming endpoint</p>
        <p><strong>Parameters:</strong> category (string)</p>
        <p><strong>Example:</strong> <code>/stream/json?category=news</code></p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/sse/events</span></h3>
        <p>Mock Server-Sent Events endpoint</p>
        <p><strong>Parameters:</strong> topic (string)</p>
        <p><strong>Example:</strong> <code>/sse/events?topic=updates</code></p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/stream/users</span></h3>
        <p>Mock user streaming endpoint</p>
        <p><strong>Parameters:</strong> status (string, optional)</p>
        <p><strong>Example:</strong> <code>/stream/users?status=active</code></p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/stream/raw</span></h3>
        <p>Mock raw data streaming for transformation</p>
        <p><strong>Parameters:</strong> source (string)</p>
        <p><strong>Example:</strong> <code>/stream/raw?source=api1</code></p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/health</span></h3>
        <p>Health check endpoint</p>
    </div>

    <p><strong>Server running on:</strong> http://localhost:9090</p>
    <p><strong>Use with proxy service:</strong> Start the proxy service and configure it to use this mock server as the external API.</p>
</body>
</html>`

	w.Write([]byte(html))
}

func (ms *Server) SetupRoutes() {
	http.HandleFunc("/stream/data", ms.StreamDataHandler)
	http.HandleFunc("/stream/json", ms.StreamJSONHandler)
	http.HandleFunc("/sse/events", ms.SSEHandler)
	http.HandleFunc("/stream/users", ms.StreamUsersHandler)
	http.HandleFunc("/stream/raw", ms.StreamRawHandler)
	http.HandleFunc("/health", ms.HealthHandler)
	http.HandleFunc("/", ms.InfoHandler)
}

func (ms *Server) Start(port string) error {
	log.Printf("Starting Mock Streaming API Server on :%s", port)
	log.Printf("Visit http://localhost:%s for endpoint information", port)

	return http.ListenAndServe(":"+port, nil)
}
