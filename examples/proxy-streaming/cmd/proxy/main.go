package main

import (
	"fmt"

	"github.com/anggasct/httpio/examples/proxy-streaming/internal/proxy"
)

func main() {
	proxyService := proxy.NewService()
	proxyService.SetupRoutes()

	if err := proxyService.Start("8080"); err != nil {
		fmt.Println("Proxy service failed to start:", err)
	}
}
