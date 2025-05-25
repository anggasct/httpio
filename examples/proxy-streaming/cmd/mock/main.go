package main

import (
	"fmt"

	"github.com/anggasct/httpio/examples/proxy-streaming/internal/mock"
)

func main() {
	mockServer := mock.NewServer()
	mockServer.SetupRoutes()

	if err := mockServer.Start("9090"); err != nil {
		fmt.Println("Mock server failed to start:", err)
	}
}
