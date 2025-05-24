# Circuit Breaker Example

This example demonstrates the circuit breaker functionality in goclient. The circuit breaker pattern helps prevent cascading failures by stopping requests to a failing service and allowing it time to recover.

## What is a Circuit Breaker?

A circuit breaker has three states:

1. **CLOSED** - Normal operation, requests are allowed through
2. **OPEN** - Service is failing, requests are immediately rejected  
3. **HALF_OPEN** - Testing if service has recovered, limited requests allowed

## Circuit Breaker Configuration

```go
client := goclient.New().
    WithCircuitBreaker(
        3,                    // Failure threshold - open after 3 failures
        2*time.Second,        // Recovery timeout - try recovery after 2 seconds  
        2,                    // Half-open max calls - allow 2 test calls
    )
```

## Features Demonstrated

1. **Failure Detection** - Circuit opens after threshold failures
2. **Request Rejection** - Fast-fail when circuit is open
3. **Automatic Recovery** - Circuit tries to recover after timeout
4. **Half-Open Testing** - Limited requests to test service health
5. **State Change Callbacks** - Monitor circuit breaker state changes
6. **Manual Reset** - Force circuit back to closed state
7. **Statistics** - Get current circuit breaker state and metrics

## Running the Example

```bash
cd example/circuit_breaker
go run main.go
```

## Expected Output

The example will show:
- Initial requests failing and opening the circuit
- Requests being rejected while circuit is open
- Automatic recovery attempts after timeout
- Circuit closing after successful recovery
- Statistics and state information throughout the process

## Key Benefits

- **Fault Tolerance** - Prevents cascading failures
- **Fast Failure** - Immediate rejection instead of waiting for timeouts
- **Automatic Recovery** - Self-healing behavior
- **Monitoring** - Visibility into service health via state changes
- **Resource Protection** - Reduces load on failing services
