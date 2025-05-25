package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"time"
)

func createAuthServer() *httptest.Server {
	tokenCounter := 0
	refreshCounter := 0

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/token" {
			log.Printf("TOKEN SERVER: Invalid request to %s with method %s", r.URL.Path, r.Method)
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"invalid_request","error_description":"Invalid endpoint"}`))
			return
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			log.Printf("TOKEN SERVER: Failed to parse form data: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid_request","error_description":"Could not parse form data"}`))
			return
		}

		grantType := r.FormValue("grant_type")
		clientID := r.FormValue("client_id")
		clientSecret := r.FormValue("client_secret")

		// Validate client credentials
		if clientID != "test-client" || clientSecret != "test-secret" {
			log.Printf("TOKEN SERVER: Invalid client credentials: ID=%s", clientID)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid_client","error_description":"Invalid client credentials"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Grant types
		switch grantType {
		case "refresh_token":
			if r.FormValue("refresh_token") != "mock-refresh-token-456" {
				log.Printf("TOKEN SERVER: Invalid refresh token: %s", r.FormValue("refresh_token"))
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"invalid_grant","error_description":"Invalid refresh token"}`))
				return
			}

			refreshCounter++
			log.Printf("TOKEN SERVER: Refresh token request received (count: %d)", refreshCounter)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token":"refreshed-token-` + time.Now().Format("15:04:05") + `","token_type":"Bearer","expires_in":300,"refresh_token":"mock-refresh-token-456","scope":"read write"}`))

		case "client_credentials":
			tokenCounter++
			log.Printf("TOKEN SERVER: Client credentials token request (count: %d)", tokenCounter)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token":"client-creds-token-` + time.Now().Format("15:04:05") + `","token_type":"Bearer","expires_in":10,"refresh_token":"mock-refresh-token-456","scope":"read write"}`))

		case "password":
			username := r.FormValue("username")
			password := r.FormValue("password")

			if username != "testuser" || password != "testpass" {
				log.Printf("TOKEN SERVER: Invalid username/password: %s", username)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid_grant","error_description":"Invalid credentials"}`))
				return
			}

			tokenCounter++
			log.Printf("TOKEN SERVER: Password grant token request (count: %d)", tokenCounter)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"access_token":"password-grant-token-` + time.Now().Format("15:04:05") + `","token_type":"Bearer","expires_in":10,"refresh_token":"mock-refresh-token-456","scope":"read write"}`))

		default:
			log.Printf("TOKEN SERVER: Unsupported grant type: %s", grantType)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"unsupported_grant_type","error_description":"Unsupported grant type"}`))
		}
	}))
}
