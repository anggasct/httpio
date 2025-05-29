package main

import (
	"context"
	"fmt"
	"log"

	"github.com/anggasct/httpio"
)

type SSEHandler struct{}

// OnOpen is called when the SSE connection is established
func (h *SSEHandler) OnOpen() error {
	fmt.Println("SSE connection opened")
	return nil
}

// OnEvent is called for each received SSE event
func (h *SSEHandler) OnEvent(event httpio.SSEEvent) error {
	fmt.Printf("Received event - Event: %s, ID: %s, Data: %s\n",
		event.Event, event.ID, event.Data)
	return nil
}

// OnClose is called when the SSE connection is closed
func (h *SSEHandler) OnClose() error {
	fmt.Println("SSE connection closed")
	return nil
}

func StreamSSE() {
	client := httpio.New().WithTimeout(30)

	req := client.NewRequest("GET", "http://localhost:8080/events")

	err := req.StreamSSE(context.Background(), &SSEHandler{})
	if err != nil {
		log.Fatal("SSE streaming error:", err)
	}
}
