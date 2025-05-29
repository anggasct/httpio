package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [lines|json|sse|func-handlers|sse-examples|comprehensive|quick|server]")
		os.Exit(1)
	}

	exampleType := os.Args[1]

	switch exampleType {
	case "lines":
		fmt.Println("Running line streaming example...")
		StreamLines()
	case "json":
		fmt.Println("Running JSON streaming example...")
		StreamJSON()
	case "sse":
		fmt.Println("Running SSE streaming example...")
		StreamSSE()
	case "func-handlers":
		fmt.Println("Running function handlers example...")
		demonstrateFunctionHandlers()
	case "server":
		port := "8080"
		if len(os.Args) > 2 {
			port = os.Args[2]
		}
		fmt.Printf("Starting mock server on port %s...\n", port)
		StartMockServer(port)
	default:
		fmt.Println("Unknown example type. Use 'lines', 'json', 'sse', 'func-handlers', 'sse-examples', 'comprehensive', 'quick', or 'server'")
		os.Exit(1)
	}
}
