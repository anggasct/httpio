# Streaming Examples

This directory contains examples demonstrating various streaming capabilities of the goclient library.

## File Structure

- **`main.go`** - Main entry point that orchestrates all streaming examples
- **`types.go`** - Data type definitions (User struct)
- **`examples.go`** - Client-side streaming example functions
- **`server.go`** - HTTP server implementation with various streaming endpoints

## Examples Included

1. **Basic Streaming** - Raw byte chunks
2. **Line-by-Line Streaming** - Text data streamed line by line
3. **JSON Streaming** - JSON objects streamed one at a time
4. **Typed Object Streaming** - Streaming with automatic type conversion
5. **Server-Sent Events (SSE)** - Event-driven streaming
6. **Context Cancellation** - Demonstrates proper cancellation handling

## Running the Examples

```bash
go run .
```

This will:
1. Start a local HTTP server on port 8080
2. Run all streaming examples in sequence
3. Display the results of each example

## Server Endpoints

The test server provides the following endpoints:

- `/stream` - Raw binary data streaming
- `/streamlines` - Line-by-line text streaming
- `/streamjson` - JSON objects streaming
- `/streamusers` - User objects streaming
- `/sse` - Server-Sent Events
- `/slowendless` - Slow endless stream for cancellation testing
