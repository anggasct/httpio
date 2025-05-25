package main

import (
	"context"
	"time"

	"github.com/anggasct/httpio/internal/client"
)

func main() {
	client := client.New().
		WithBaseURL("http://localhost:8080").
		WithHeader("User-Agent", "client-streaming-example").
		WithTimeout(30 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	serverReady := make(chan struct{})
	go startStreamingServer(serverReady)
	<-serverReady

	runBasicStreamExample(ctx, client)
	runLineStreamExample(ctx, client)
	runJSONStreamExample(ctx, client)
	runTypedStreamExample(ctx, client)
	runSSEExample(ctx, client)
	runCancellationExample(client)
}
