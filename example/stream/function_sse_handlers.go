package main

import (
	"context"
	"fmt"
	"log"

	"github.com/anggasct/httpio"
)

func demonstrateFunctionHandlers() {
	client := httpio.New().WithTimeout(30)
	req := client.NewRequest("GET", "http://localhost:8080/events")

	var directFunc httpio.SSEEventHandlerFunc = func(event httpio.SSEEvent) error {
		fmt.Printf("Direct: %s - %s\n", event.Event, event.Data)
		return nil
	}

	partialLifecycleHandler := &httpio.SSEEventFullHandlerFunc{
		OnEventFunc: func(event httpio.SSEEvent) error {
			fmt.Printf("Partial: %s - %s\n", event.Event, event.Data)
			return nil
		},
		OnOpenFunc: func() error {
			fmt.Println("Partial: Connection opened")
			return nil
		},
		// OnCloseFunc is nil - will be ignored
	}

	fmt.Println("Using direct function type...")
	if err := req.StreamSSE(context.Background(), directFunc); err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println("Using partial lifecycle handler...")
	if err := req.StreamSSE(context.Background(), partialLifecycleHandler); err != nil {
		log.Printf("Error: %v", err)
	}
}
