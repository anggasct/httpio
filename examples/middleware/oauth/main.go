// OAuth middleware example demonstrating:
// - Client Credentials grant (service-to-service)
// - Password grant (user authentication)
// - Token refresh
// - Error handling

package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/anggasct/httpio/internal/client"
	"github.com/anggasct/httpio/internal/middleware/oauth"
)

func main() {
	// Create mock servers
	oauthServer := createAuthServer()
	defer oauthServer.Close()

	apiServer := createResourceServer()
	defer apiServer.Close()

	// OAuth middleware with client_credentials grant
	oauthMiddleware := oauth.New(&oauth.Config{
		TokenURL:         oauthServer.URL + "/token",
		ClientID:         "test-client",
		ClientSecret:     "test-secret",
		Scopes:           []string{"read", "write"},
		GrantType:        "client_credentials",
		RefreshThreshold: 5 * time.Second,
		OnNewToken: func(token *oauth.TokenResponse) {
			log.Printf("EVENT: New token obtained (%s), expires in: %d seconds", token.AccessToken, token.ExpiresIn)
		},
		OnTokenError: func(err error) {
			log.Printf("EVENT: Error getting token: %v", err)
		},
		AdditionalParams: map[string]string{
			"audience": "api://default",
		},
	})

	// HTTP client with OAuth middleware
	httpClient := client.New().
		WithBaseURL(apiServer.URL).
		WithMiddleware(oauthMiddleware)

	ctx := context.Background()

	// Example 1: Token Caching and Different Endpoints
	log.Println("\n=== Example 1: Token Caching & Multiple Endpoints ===")
	log.Println("Making multiple requests to different endpoints using the same cached token")

	endpoints := []string{"/api/resource", "/api/profile", "/api/data"}

	for i, endpoint := range endpoints {
		log.Printf("\n--- Request %d to %s ---", i+1, endpoint)
		resp, err := httpClient.GET(ctx, endpoint)
		if err != nil {
			log.Printf("ERROR: Request failed: %v", err)
			continue
		}

		log.Printf("Response status: %s", resp.Status)

		if resp.StatusCode == http.StatusOK {
			respBody, _ := resp.String()
			if len(respBody) > 150 {
				respBody = respBody[:150] + "..."
			}
			log.Printf("Response body: %s", respBody)
		}
		if i < len(endpoints)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	// Example 2: Token Refresh
	log.Println("\n=== Example 2: Automatic Token Refresh ===")
	log.Println("Our token expires in 10 seconds. Waiting 12 seconds to trigger refresh...")

	time.Sleep(12 * time.Second)

	log.Println("\n--- Request after token expiration ---")
	resp, err := httpClient.GET(ctx, "/api/resource")
	if err != nil {
		log.Printf("ERROR: Request failed: %v", err)
	} else {
		log.Printf("Response status: %s", resp.Status)
		respBody, _ := resp.String()
		if len(respBody) > 150 {
			respBody = respBody[:150] + "..."
		}
		log.Printf("Response body: %s", respBody)
	}

	// Example 3: Password Grant Type
	log.Println("\n=== Example 3: Password Grant Type ===")
	log.Println("Demonstrating the Resource Owner Password Credentials flow...")

	// OAuth with password grant
	passwordOAuthMiddleware := oauth.New(&oauth.Config{
		TokenURL:     oauthServer.URL + "/token",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		GrantType:    "password",
		Username:     "testuser",
		Password:     "testpass",
		Scopes:       []string{"read", "write", "profile"},
		OnNewToken: func(token *oauth.TokenResponse) {
			log.Printf("EVENT: Password grant token obtained, expires in: %d seconds", token.ExpiresIn)
		},
		OnTokenError: func(err error) {
			log.Printf("EVENT: Password grant error: %v", err)
		},
	})

	// Client with password grant OAuth
	passwordClient := client.New().
		WithBaseURL(apiServer.URL).
		WithMiddleware(passwordOAuthMiddleware)

	log.Println("\n--- Request using password grant type ---")
	resp, err = passwordClient.GET(ctx, "/api/profile")
	if err != nil {
		log.Printf("ERROR: Request failed: %v", err)
	} else {
		log.Printf("Response status: %s", resp.Status)
		respBody, _ := resp.String()
		if len(respBody) > 150 {
			respBody = respBody[:150] + "..."
		}
		log.Printf("Response body: %s", respBody)
	}

	// Example 4: Error Handling
	log.Println("\n=== Example 4: Error Handling ===")
	log.Println("Creating a client with invalid credentials to demonstrate error handling...")

	// OAuth with invalid credentials
	invalidOAuthMiddleware := oauth.New(&oauth.Config{
		TokenURL:     oauthServer.URL + "/token",
		ClientID:     "invalid-client",
		ClientSecret: "invalid-secret",
		GrantType:    "client_credentials",
		OnNewToken: func(token *oauth.TokenResponse) {
			log.Printf("EVENT: New token obtained (should not happen)")
		},
		OnTokenError: func(err error) {
			log.Printf("EVENT: Authentication error (expected): %v", err)
		},
	})

	// Client with invalid OAuth
	invalidClient := client.New().
		WithBaseURL(apiServer.URL).
		WithMiddleware(invalidOAuthMiddleware)

	log.Println("\n--- Request with invalid credentials ---")
	_, err = invalidClient.GET(ctx, "/api/resource")
	if err != nil {
		log.Printf("Request failed (expected): %v", err)
	}
}
