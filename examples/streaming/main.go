package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/anggasct/goclient/pkg/goclient"
)

func main() {
	client := goclient.New().
		WithBaseURL("http://localhost:8080").
		WithHeader("User-Agent", "goclient-streaming-example").
		WithTimeout(30 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	serverReady := make(chan struct{})

	go startStreamingServer(serverReady)

	<-serverReady

	fmt.Println("Running streaming examples...")
	fmt.Println("=============================")

	fmt.Println("\n1. Basic Streaming - Raw Chunks")
	fmt.Println("-------------------------------")
	if err := runBasicStreamExample(ctx, client); err != nil {
		log.Fatalf("Basic stream example failed: %v", err)
	}

	fmt.Println("\n2. Line-by-Line Streaming")
	fmt.Println("------------------------")
	if err := runLineStreamExample(ctx, client); err != nil {
		log.Fatalf("Line stream example failed: %v", err)
	}

	fmt.Println("\n3. JSON Streaming")
	fmt.Println("----------------")
	if err := runJSONStreamExample(ctx, client); err != nil {
		log.Fatalf("JSON stream example failed: %v", err)
	}

	fmt.Println("\n4. Typed Object Streaming")
	fmt.Println("------------------------")
	if err := runTypedStreamExample(ctx, client); err != nil {
		log.Fatalf("Typed stream example failed: %v", err)
	}

	fmt.Println("\n5. Server-Sent Events (SSE)")
	fmt.Println("---------------------------")
	if err := runSSEExample(ctx, client); err != nil {
		log.Fatalf("SSE example failed: %v", err)
	}

	fmt.Println("\n6. Context Cancellation")
	fmt.Println("----------------------")
	if err := runCancellationExample(client); err != nil {
		log.Fatalf("Cancellation example failed: %v", err)
	}

	fmt.Println("\nAll examples completed successfully!")
}
