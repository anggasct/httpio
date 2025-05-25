package proxy

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/anggasct/httpio/internal/client"
	"github.com/anggasct/httpio/internal/stream"
)

type Service struct {
	httpClient *client.Client
}

func NewService() *Service {
	httpClient := client.New().
		WithTimeout(30 * time.Second).
		WithBaseURL("http://localhost:9090")

	return &Service{
		httpClient: httpClient,
	}
}

func (ps *Service) StreamDataHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	externalPath := fmt.Sprintf("/stream/data?query=%s", query)

	log.Printf("Proxying stream request to: %s", externalPath)

	headersSent := false

	err := stream.GetStreamLines(ps.httpClient, r.Context(), externalPath, func(line []byte) error {
		headersSent = true

		_, writeErr := w.Write(line)
		if writeErr != nil {
			return writeErr
		}

		_, writeErr = w.Write([]byte("\n"))
		if writeErr != nil {
			return writeErr
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		return nil
	})

	if err != nil {
		log.Printf("Error stream data: %v", err)
		if !headersSent {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	log.Println("Stream completed successfully")
}

func (ps *Service) StreamJSONHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	category := r.URL.Query().Get("category")
	if category == "" {
		category = "default"
	}

	externalPath := fmt.Sprintf("/stream/json?category=%s", category)
	log.Printf("Proxying JSON stream request to: %s", externalPath)

	headersSent := false

	err := stream.GetStreamJSON(ps.httpClient, r.Context(), externalPath, func(rawData json.RawMessage) error {
		headersSent = true

		var data map[string]interface{}
		if err := json.Unmarshal(rawData, &data); err != nil {
			return err
		}

		data["proxy_timestamp"] = time.Now().Format(time.RFC3339)

		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		_, writeErr := w.Write(jsonData)
		if writeErr != nil {
			return writeErr
		}

		_, writeErr = w.Write([]byte("\n"))
		if writeErr != nil {
			return writeErr
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		return nil
	})

	if err != nil {
		log.Printf("Error stream JSON: %v", err)
		if !headersSent {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	log.Println("JSON stream completed successfully")
}

func (ps *Service) SSEHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	topic := r.URL.Query().Get("topic")
	if topic == "" {
		topic = "general"
	}

	externalPath := fmt.Sprintf("/sse/events?topic=%s", topic)
	log.Printf("Proxying SSE request to: %s", externalPath)

	headersSent := false

	err := stream.GetSSE(ps.httpClient, r.Context(), externalPath, func(event stream.SSEEvent) error {
		headersSent = true

		if event.Event != "" {
			_, writeErr := fmt.Fprintf(w, "event: %s\n", event.Event)
			if writeErr != nil {
				return writeErr
			}
		}

		_, writeErr := fmt.Fprintf(w, "data: %s\n", event.Data)
		if writeErr != nil {
			return writeErr
		}

		if event.ID != "" {
			_, writeErr := fmt.Fprintf(w, "id: %s\n", event.ID)
			if writeErr != nil {
				return writeErr
			}
		}

		if event.Retry > 0 {
			_, writeErr := fmt.Fprintf(w, "retry: %d\n", event.Retry)
			if writeErr != nil {
				return writeErr
			}
		}

		_, writeErr = w.Write([]byte("\n"))
		if writeErr != nil {
			return writeErr
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		return nil
	})

	if err != nil {
		log.Printf("Error stream SSE: %v", err)
		if !headersSent {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Internal Server Error",
				"message": "An error occurred while stream SSE data",
			})
		}
		return
	}

	log.Println("SSE stream completed successfully")
}

func (ps *Service) UserStreamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	status := r.URL.Query().Get("status")
	externalPath := fmt.Sprintf("/stream/users?status=%s", status)

	log.Printf("Proxying user stream request to: %s", externalPath)

	headersSent := false
	userCount := 0

	err := stream.GetStreamJSON(ps.httpClient, r.Context(), externalPath, func(rawData json.RawMessage) error {
		headersSent = true
		userCount++

		var data map[string]interface{}
		if err := json.Unmarshal(rawData, &data); err != nil {
			return err
		}

		data["proxy_processed_at"] = time.Now().Format(time.RFC3339)
		data["proxy_user_number"] = userCount

		if email, ok := data["email"].(string); ok {
			if len(email) > 3 {
				maskedEmail := email[:3] + "***@" + email[strings.LastIndex(email, "@")+1:]
				data["masked_email"] = maskedEmail
				delete(data, "email")
			}
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}

		_, writeErr := w.Write(jsonData)
		if writeErr != nil {
			return writeErr
		}

		_, writeErr = w.Write([]byte("\n"))
		if writeErr != nil {
			return writeErr
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		return nil
	})

	if err != nil {
		log.Printf("Error stream users: %v", err)
		if !headersSent {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("User stream completed successfully, processed %d users", userCount)
}

func (ps *Service) RawDataTransformHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	source := r.URL.Query().Get("source")
	externalPath := fmt.Sprintf("/stream/raw?source=%s", source)

	log.Printf("Proxying raw data stream request to: %s", externalPath)

	headersSent := false

	err := stream.GetStreamJSON(ps.httpClient, r.Context(), externalPath, func(rawData json.RawMessage) error {
		headersSent = true

		var data map[string]interface{}
		if err := json.Unmarshal(rawData, &data); err != nil {
			return err
		}

		transformed := map[string]interface{}{
			"original_id": data["id"],
			"source":      data["source"],
			"processed": map[string]interface{}{
				"value":          data["value"],
				"metric":         data["metric"],
				"metric_doubled": data["metric"].(float64) * 2,
				"timestamp":      time.Now().Format(time.RFC3339),
			},
			"raw_data": data,
		}

		jsonData, err := json.Marshal(transformed)
		if err != nil {
			return err
		}

		_, writeErr := w.Write(jsonData)
		if writeErr != nil {
			return writeErr
		}

		_, writeErr = w.Write([]byte("\n"))
		if writeErr != nil {
			return writeErr
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		return nil
	})

	if err != nil {
		log.Printf("Error stream transformed data: %v", err)
		if !headersSent {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	log.Println("Raw data transformation stream completed successfully")
}

func (ps *Service) HealthHandler(w http.ResponseWriter, r *http.Request) {
	resp, err := ps.httpClient.GET(r.Context(), "/health")
	if err != nil {
		http.Error(w, "External API unavailable", http.StatusServiceUnavailable)
		return
	}
	resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"status":       "healthy",
		"timestamp":    time.Now().Format(time.RFC3339),
		"version":      "1.0.0",
		"external_api": "healthy",
	}
	json.NewEncoder(w).Encode(response)
}

func (ps *Service) InfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Stream Proxy Service</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { margin: 15px 0; padding: 10px; border: 1px solid #ddd; border-radius: 5px; }
        .method { color: #007bff; font-weight: bold; }
        .path { font-family: monospace; background: #f8f9fa; padding: 2px 4px; }
        h1 { color: #333; }
        h3 { color: #666; margin-bottom: 5px; }
        p { margin: 5px 0; }
        .note { background: #fff3cd; padding: 10px; border-radius: 5px; margin: 15px 0; }
    </style>
</head>
<body>
    <h1>Stream Proxy Service</h1>
    <p>This service proxies stream requests to external APIs with transformation capabilities.</p>
    
    <div class="note">
        <strong>Note:</strong> This proxy service forwards requests to the mock server running on localhost:9090
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/proxy/stream/data</span></h3>
        <p>Proxy text stream with pass-through</p>
        <p><strong>Parameters:</strong> q (string)</p>
        <p><strong>Example:</strong> <code>/proxy/stream/data?q=test</code></p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/proxy/stream/json</span></h3>
        <p>Proxy NDJSON stream with timestamp injection</p>
        <p><strong>Parameters:</strong> category (string)</p>
        <p><strong>Example:</strong> <code>/proxy/stream/json?category=news</code></p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/proxy/sse/events</span></h3>
        <p>Proxy Server-Sent Events with forwarding</p>
        <p><strong>Parameters:</strong> topic (string)</p>
        <p><strong>Example:</strong> <code>/proxy/sse/events?topic=updates</code></p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/proxy/stream/users</span></h3>
        <p>Proxy user stream with email masking</p>
        <p><strong>Parameters:</strong> status (string, optional)</p>
        <p><strong>Example:</strong> <code>/proxy/stream/users?status=active</code></p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/proxy/stream/transform</span></h3>
        <p>Proxy raw data stream with transformation</p>
        <p><strong>Parameters:</strong> source (string)</p>
        <p><strong>Example:</strong> <code>/proxy/stream/transform?source=api1</code></p>
    </div>

    <div class="endpoint">
        <h3><span class="method">GET</span> <span class="path">/health</span></h3>
        <p>Health check for proxy service and external API</p>
    </div>

    <p><strong>Proxy service running on:</strong> http://localhost:8080</p>
    <p><strong>External API:</strong> http://localhost:9090</p>
</body>
</html>`

	w.Write([]byte(html))
}

func (ps *Service) SetupRoutes() {
	http.HandleFunc("/proxy/stream/data", ps.StreamDataHandler)
	http.HandleFunc("/proxy/stream/json", ps.StreamJSONHandler)
	http.HandleFunc("/proxy/sse/events", ps.SSEHandler)
	http.HandleFunc("/proxy/stream/users", ps.UserStreamHandler)
	http.HandleFunc("/proxy/stream/transform", ps.RawDataTransformHandler)
	http.HandleFunc("/health", ps.HealthHandler)
	http.HandleFunc("/", ps.InfoHandler)
}

func (ps *Service) Start(port string) error {
	log.Printf("Starting Stream Proxy Service on :%s", port)
	log.Printf("Visit http://localhost:%s for endpoint information", port)
	log.Printf("External API configured for: http://localhost:9090")

	return http.ListenAndServe(":"+port, nil)
}
