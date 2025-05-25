package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/anggasct/httpio/internal/client"
	"github.com/anggasct/httpio/internal/stream"
)

func runBasicStreamExample(ctx context.Context, client *client.Client) error {
	var totalBytes int
	chunks := 0

	err := stream.GetStream(client, ctx, "/stream", func(chunk []byte) error {
		chunks++
		totalBytes += len(chunk)
		fmt.Println("Received chunk:", len(chunk), "bytes")
		return nil
	})

	return err
}

func runLineStreamExample(ctx context.Context, client *client.Client) error {
	lines := 0
	err := stream.GetStreamLines(client, ctx, "/streamlines", func(line []byte) error {
		lines++
		fmt.Println("Line data:", string(line))
		return nil
	})

	return err
}

func runJSONStreamExample(ctx context.Context, client *client.Client) error {
	count := 0
	err := stream.GetStreamJSON(client, ctx, "/streamjson", func(raw json.RawMessage) error {
		count++
		var data map[string]interface{}
		if err := json.Unmarshal(raw, &data); err != nil {
			return err
		}
		fmt.Println("JSON object:", data)
		return nil
	})

	return err
}

func runTypedStreamExample(ctx context.Context, client *client.Client) error {
	count := 0
	err := stream.GetStreamInto(client, ctx, "/streamusers", func(user User) error {
		count++
		fmt.Println("User:", user.Name, "from", user.Location)
		return nil
	})

	return err
}

func runSSEExample(ctx context.Context, client *client.Client) error {
	events := 0
	err := stream.GetSSE(client, ctx, "/sse", func(event stream.SSEEvent) error {
		events++
		fmt.Println("Event:", event.Event, "-", event.Data)
		return nil
	})

	return err
}

func runCancellationExample(client *client.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := stream.GetStreamLines(client, ctx, "/slowendless", func(line []byte) error {
		fmt.Println("Received line:", string(line))
		return nil
	})

	return err
}
