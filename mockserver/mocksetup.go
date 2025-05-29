// Package main provides a ready-to-use mock server with predefined routes and data
package mockserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/anggasct/httpio"
)

// MockData contains all predefined mock data for testing
type MockData struct {
	Users       []User
	Posts       []Post
	Comments    []Comment
	Products    []Product
	Events      []httpio.SSEEvent
	StreamItems []interface{}
	NDJSONItems []interface{}
}

// User represents a user entity for testing
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// Post represents a post entity for testing
type Post struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

// Comment represents a comment entity for testing
type Comment struct {
	ID     int    `json:"id"`
	PostID int    `json:"postId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Body   string `json:"body"`
}

// Product represents a product entity for testing
type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	InStock     bool    `json:"inStock"`
}

// MockSetup represents a preconfigured mock server
type MockSetup struct {
	Server   *MockServer
	Data     MockData
	RestData map[string]interface{}
}

// NewMockSetup creates a new preconfigured mock server
func NewMockSetup(address string) *MockSetup {
	if address == "" {
		address = "localhost:8080"
	}

	mockData := generateMockData()
	restData := make(map[string]interface{})

	for _, user := range mockData.Users {
		restData[fmt.Sprintf("%d", user.ID)] = user
	}

	ms := &MockSetup{
		Server:   NewMockServer(address),
		Data:     mockData,
		RestData: restData,
	}

	ms.setupDefaultRoutes()
	return ms
}

// Start starts the mock server
func (ms *MockSetup) Start() error {
	return ms.Server.Start()
}

// Stop stops the mock server
func (ms *MockSetup) Stop() error {
	return ms.Server.Stop()
}

// URL returns the full URL for the given path
func (ms *MockSetup) URL(path string) string {
	return ms.Server.URL(path)
}

// setupDefaultRoutes configures all the default routes for the mock server
func (ms *MockSetup) setupDefaultRoutes() {
	// Standard JSON routes
	ms.Server.AddJSONRoute("/api/users", ResponseConfig{
		Data: ms.Data.Users,
	})

	ms.Server.AddJSONRoute("/api/users/1", ResponseConfig{
		Data: ms.Data.Users[0],
	})

	ms.Server.AddJSONRoute("/api/posts", ResponseConfig{
		Data: ms.Data.Posts,
	})

	ms.Server.AddJSONRoute("/api/comments", ResponseConfig{
		Data: ms.Data.Comments,
	})

	ms.Server.AddJSONRoute("/api/products", ResponseConfig{
		Data: ms.Data.Products,
	})

	// Delayed response
	ms.Server.AddJSONRoute("/api/slow", ResponseConfig{
		Data:  map[string]string{"message": "This response was delayed"},
		Delay: 2 * time.Second,
	})

	// Custom status codes
	ms.Server.AddJSONRoute("/api/error", ResponseConfig{
		StatusCode: http.StatusInternalServerError,
		Data:       map[string]string{"error": "Internal Server Error"},
	})

	ms.Server.AddJSONRoute("/api/unauthorized", ResponseConfig{
		StatusCode: http.StatusUnauthorized,
		Data:       map[string]string{"error": "Unauthorized"},
	})

	// Streaming routes
	ms.Server.AddStreamingJSONRoute("/api/stream", ms.Data.StreamItems, 500)
	ms.Server.AddNDJSONRoute("/api/ndjson", ms.Data.NDJSONItems, 500)
	ms.Server.AddSSERoute("/api/events", ms.Data.Events, 500)

	// REST API with CRUD operations
	ms.Server.AddRESTRoute("/api/resources", struct{}{}, ms.RestData)
}

// generateMockData creates all the mock data for testing
func generateMockData() MockData {
	users := []User{
		{ID: 1, Name: "John Doe", Email: "john@example.com", Username: "johndoe"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com", Username: "janesmith"},
		{ID: 3, Name: "Mike Johnson", Email: "mike@example.com", Username: "mikej"},
		{ID: 4, Name: "Sarah Williams", Email: "sarah@example.com", Username: "sarahw"},
		{ID: 5, Name: "Robert Brown", Email: "robert@example.com", Username: "robertb"},
	}

	posts := []Post{
		{ID: 1, UserID: 1, Title: "First post", Body: "This is the first post content"},
		{ID: 2, UserID: 1, Title: "Second post", Body: "This is the second post content"},
		{ID: 3, UserID: 2, Title: "Hello World", Body: "Introduction post from Jane"},
		{ID: 4, UserID: 3, Title: "Technology News", Body: "Latest updates in tech"},
		{ID: 5, UserID: 4, Title: "Travel Diaries", Body: "My recent trip to Europe"},
	}

	comments := []Comment{
		{ID: 1, PostID: 1, Name: "Jane Smith", Email: "jane@example.com", Body: "Great post!"},
		{ID: 2, PostID: 1, Name: "Mike Johnson", Email: "mike@example.com", Body: "Thanks for sharing"},
		{ID: 3, PostID: 2, Name: "Sarah Williams", Email: "sarah@example.com", Body: "Interesting topic"},
		{ID: 4, PostID: 3, Name: "John Doe", Email: "john@example.com", Body: "Welcome to the platform"},
		{ID: 5, PostID: 4, Name: "Robert Brown", Email: "robert@example.com", Body: "Keep me updated"},
	}

	products := []Product{
		{ID: 1, Name: "Laptop", Price: 999.99, Description: "High-performance laptop", Category: "Electronics", InStock: true},
		{ID: 2, Name: "Smartphone", Price: 699.99, Description: "Latest model", Category: "Electronics", InStock: true},
		{ID: 3, Name: "Headphones", Price: 199.99, Description: "Noise-cancelling", Category: "Audio", InStock: false},
		{ID: 4, Name: "Coffee Maker", Price: 89.99, Description: "Automatic drip", Category: "Kitchen", InStock: true},
		{ID: 5, Name: "Desk Chair", Price: 249.99, Description: "Ergonomic design", Category: "Furniture", InStock: true},
	}

	// Create streaming items
	streamItems := make([]interface{}, 0)
	for i := 1; i <= 10; i++ {
		streamItems = append(streamItems, map[string]interface{}{
			"id":      i,
			"message": fmt.Sprintf("Stream message %d", i),
			"time":    time.Now().Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
		})
	}

	// Create NDJSON items
	ndjsonItems := make([]interface{}, 0)
	for i := 1; i <= 10; i++ {
		ndjsonItems = append(ndjsonItems, map[string]interface{}{
			"id":      i,
			"data":    fmt.Sprintf("NDJSON item %d", i),
			"created": time.Now().Unix(),
		})
	}

	// Create SSE events
	events := []httpio.SSEEvent{
		{ID: "1", Event: "message", Data: `{"id":1,"message":"First event"}`},
		{ID: "2", Event: "update", Data: `{"id":2,"message":"Status update"}`},
		{ID: "3", Event: "message", Data: `{"id":3,"message":"Another message"}`},
		{ID: "4", Event: "alert", Data: `{"id":4,"message":"Important alert"}`},
		{ID: "5", Event: "message", Data: `{"id":5,"message":"Final message"}`},
	}

	return MockData{
		Users:       users,
		Posts:       posts,
		Comments:    comments,
		Products:    products,
		Events:      events,
		StreamItems: streamItems,
		NDJSONItems: ndjsonItems,
	}
}
