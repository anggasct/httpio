package main

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

func createResourceServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		auth := r.Header.Get("Authorization")

		if auth == "" {
			log.Printf("API SERVER: Request received without Authorization header")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized","error_description":"Missing Authorization header"}`))
			return
		}

		if !strings.HasPrefix(auth, "Bearer ") {
			log.Printf("API SERVER: Invalid Authorization format: %s", auth)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized","error_description":"Invalid Authorization format, expected Bearer token"}`))
			return
		}

		tokenValue := strings.TrimPrefix(auth, "Bearer ")
		endpointPath := r.URL.Path

		log.Printf("API SERVER: Successful request to %s with token: %s", endpointPath, tokenValue)
		switch endpointPath {
		case "/api/profile":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"Successfully authenticated","endpoint":"profile","token":"` + tokenValue + `","user":{"id":12345,"username":"test_user","email":"user@example.com","role":"admin"}}`))
		case "/api/data":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"Successfully authenticated","endpoint":"data","token":"` + tokenValue + `","items":[{"id":1,"name":"Item 1"},{"id":2,"name":"Item 2"},{"id":3,"name":"Item 3"}]}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"Successfully authenticated with token","endpoint":"` + endpointPath + `","token":"` + tokenValue + `","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
		}
	}))
}
