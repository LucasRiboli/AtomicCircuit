# AtomicCircuit

[![Go Reference](https://pkg.go.dev/badge/github.com/LucasRiboli/atomiccircuit.svg)](https://pkg.go.dev/github.com/LucasRiboli/atomiccircuit)
[![Go Report Card](https://goreportcard.com/badge/github.com/LucasRiboli/atomiccircuit)](https://goreportcard.com/report/github.com/LucasRiboli/atomiccircuit)

A thread-safe Circuit Breaker implementation in Go using atomic operations.

## About

AtomicCircuit implements the Circuit Breaker pattern to prevent cascading failures in distributed systems. This component protects your system against recurring failures in external services, allowing time for recovery and preventing overload of requests destined to fail.

The implementation uses atomic operations from the `sync/atomic` package to ensure thread-safety in highly concurrent environments, without the overhead of traditional mutexes.

## Features

- **Thread-Safe**: Fully concurrent implementation using atomic operations
- **State Machine**: Implements the three standard states (Closed, Open, Half-Open)
- **Configurable**: Allows customizing failure thresholds, success thresholds, and timeout
- **Lightweight**: Efficient implementation with minimal overhead
- **Tested**: Test coverage to ensure proper behavior

## Installation

```bash
go get github.com/LucasRiboli/atomiccircuit
```

## Basic Usage

```go
package main

import (
    "fmt"
    "net/http"
    "time"
    
    "github.com/LucasRiboli/atomiccircuit"
)

func main() {
    // Create a new circuit breaker:
    // - 5 failures to open the circuit
    // - 2 successes to close after half-open
    // - 10 seconds timeout to test recovery
    cb := atomiccircuit.NewCircuitBreaker(5, 2, 10*time.Second)
    
    // Use the circuit breaker to protect an HTTP request
    err := cb.Execute(func() error {
        resp, err := http.Get("https://api.example.com/data")
        if err != nil {
            return err
        }
        defer resp.Body.Close()
        
        if resp.StatusCode >= 500 {
            return fmt.Errorf("server error: %d", resp.StatusCode)
        }
        
        return nil
    })
    
    if err == cb.ErrBreakerOpen {
        fmt.Println("Circuit open! Service possibly unavailable.")
        // Implement fallback or inform user
    } else if err != nil {
        fmt.Printf("Request error: %v\n", err)
    } else {
        fmt.Println("Request successful!")
    }
}
```

## Circuit Breaker States

The CircuitBreaker can be in one of three states:

1. **Closed**: Initial state. Requests are allowed normally. Failures are counted, and when they reach the threshold, the circuit opens.

2. **Open**: Requests are immediately rejected, returning `ErrBreakerOpen`. After the configured timeout period, the circuit transitions to Half-Open.

3. **Half-Open**: A limited number of requests are allowed to test service recovery. If they succeed, the circuit closes; otherwise, it opens again.

## API

### Creation

```go
cb := atomiccircuit.NewCircuitBreaker(
    failureThreshold int64,     // Number of failures to open the circuit
    successThreshold uint64,    // Number of successes in Half-Open to close
    resetTimeout time.Duration, // Time in Open state before recovery attempt
)
```

### Methods

```go
// Execute runs a function through the circuit breaker
// Returns ErrBreakerOpen if the circuit is open
err := cb.Execute(func() error {
    // Operation that might fail
    return nil
})
```

## HTTP Server Example

```go
func handler(w http.ResponseWriter, r *http.Request) {
    err := cb.Execute(func() error {
        // Call to external service
        return externalService()
    })
    
    if err == cb.ErrBreakerOpen {
        http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
        return
    }
    
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    fmt.Fprintf(w, "Operation completed successfully!")
}
```

## How It Works

AtomicCircuit uses atomic operations (`atomic.Int32`, `atomic.Uint64`, `atomic.Value`) to ensure thread-safety without the need for traditional mutexes. This provides better performance in high-concurrency environments.

The use of `CompareAndSwap` for state transitions ensures that state changes are atomic and consistent, even with multiple goroutines accessing the circuit breaker simultaneously.

## Contributing

Contributions are welcome! Feel free to open issues or pull requests to improve this library.

## License

[MIT](LICENSE)