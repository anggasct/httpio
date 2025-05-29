package main

import (
	"context"
	"fmt"
	"log"

	"github.com/anggasct/httpio"
)

func StreamLines() {
	client := httpio.New().WithTimeout(30)

	resp, err := client.GET(context.Background(), "http://localhost:8080/stream")
	if err != nil {
		log.Fatal("Request failed:", err)
	}
	defer resp.Close()

	err = resp.StreamLines(func(line []byte) error {
		fmt.Println("Received line:", string(line))
		return nil
	}, httpio.WithBufferSize(4096), httpio.WithDelimiter("\n"))

	if err != nil {
		log.Fatal("Streaming error:", err)
	}
}
