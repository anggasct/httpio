package main

import (
	"context"
	"fmt"
	"log"

	"github.com/anggasct/httpio"
)

type Message struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

func StreamJSON() {
	client := httpio.New().WithTimeout(30)

	resp, err := client.GET(context.Background(), "http://localhost:8080/stream-json")
	if err != nil {
		log.Fatal("Request failed:", err)
	}
	defer resp.Close()

	handler := func(msg *Message) error {
		fmt.Printf("Received message - ID: %s, Content: %s\n", msg.ID, msg.Content)
		return nil
	}

	err = resp.StreamInto(handler, httpio.WithBufferSize(4096), httpio.WithDelimiter("\n"))
	if err != nil {
		log.Fatal("JSON streaming error:", err)
	}
}
