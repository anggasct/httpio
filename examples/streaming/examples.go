package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anggasct/goclient/pkg/goclient"
	"github.com/anggasct/goclient/pkg/streaming"
)

func runBasicStreamExample(ctx context.Context, client *goclient.Client) error {
	fmt.Println("Streaming raw data chunks...")

	var totalBytes int
	chunks := 0

	err := streaming.GETSTREAM(client, ctx, "/stream", func(chunk []byte) error {
		chunks++
		totalBytes += len(chunk)
		fmt.Printf("  Chunk %d: %d bytes received\n", chunks, len(chunk))
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("  Received %d total bytes in %d chunks\n", totalBytes, chunks)
	return nil
}

func runLineStreamExample(ctx context.Context, client *goclient.Client) error {
	fmt.Println("Streaming data line by line...")

	lines := 0
	err := streaming.GETStreamLines(client, ctx, "/streamlines", func(line []byte) error {
		lines++
		fmt.Printf("  Line %d: %s\n", lines, string(line))
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("  Processed %d lines\n", lines)
	return nil
}

func runJSONStreamExample(ctx context.Context, client *goclient.Client) error {
	fmt.Println("Streaming JSON objects...")

	count := 0
	err := streaming.GETStreamJSON(client, ctx, "/streamjson", func(raw json.RawMessage) error {
		count++
		var data map[string]interface{}
		if err := json.Unmarshal(raw, &data); err != nil {
			return err
		}
		fmt.Printf("  Object %d: %v\n", count, data)
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("  Processed %d JSON objects\n", count)
	return nil
}

func runTypedStreamExample(ctx context.Context, client *goclient.Client) error {
	fmt.Println("Streaming typed objects...")

	count := 0
	err := streaming.GETStreamInto(client, ctx, "/streamusers", func(user User) error {
		count++
		fmt.Printf("  User %d: %s (%s) from %s\n",
			user.ID, user.Name, user.Email, user.Location)
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("  Processed %d users\n", count)
	return nil
}

func runSSEExample(ctx context.Context, client *goclient.Client) error {
	fmt.Println("Processing Server-Sent Events...")

	events := 0
	err := streaming.GETSSE(client, ctx, "/sse", func(event, data string) error {
		events++
		fmt.Printf("  Event #%d: [%s] - %s\n", events, event, data)
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("  Processed %d SSE events\n", events)
	return nil
}

func runCancellationExample(client *goclient.Client) error {
	fmt.Println("Demonstrating context cancellation...")
	fmt.Println("  Will cancel after 2 seconds...")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := streaming.GETStreamLines(client, ctx, "/slowendless", func(line []byte) error {
		fmt.Printf("  Received: %s\n", string(line))
		return nil
	})

	if err == nil {
		return fmt.Errorf("expected a context timeout error, but got nil")
	}

	fmt.Printf("  Stream was cancelled: %v\n", err)
	return nil
}
