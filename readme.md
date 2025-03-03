Here's the updated README.md with the StreamSignal feature added:

# Firm-Go

<div align="center">
  <h3>Fine-grained Reactive State Management for Go</h3>
  <p>A Solid.js-inspired reactive library for Go applications</p>

  ![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)
  ![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8.svg)
  ![Status](https://img.shields.io/badge/Status-Beta-yellow)
</div>

> Work in progress

## 📚 Table of Contents

- [Introduction](#introduction)
- [Installation](#installation)
- [Core Concepts](#core-concepts)
  - [Signals](#signals)
  - [Effects](#effects)
  - [Computed Values and Memos](#computed-values-and-memos)
  - [Contexts](#contexts)
  - [Resources](#resources)
  - [Streaming Data](#streaming-data)
  - [Batching](#batching)
  - [Async Operations](#async-operations)
  - [Polling](#polling)
- [Usage Examples](#usage-examples)
- [Concurrency & Safety](#concurrency--safety)
- [Advanced Features](#advanced-features)
- [API Reference](#api-reference)
- [Best Practices](#best-practices)

## Introduction

**Firm-Go** is a reactive state management library for Go applications inspired by [Solid.js](https://www.solidjs.com/). It enables building applications with fine-grained reactivity, automatic dependency tracking, and efficient update propagation - all while maintaining Go's type safety and concurrency model.

### Key Features

- **✅ Fine-grained reactivity**: Updates propagate efficiently through a dependency graph
- **✅ Type-safe**: Built with Go generics for compile-time type checking
- **✅ Automatic cleanup**: Resources are automatically cleaned up when no longer needed
- **✅ Batched updates**: Efficiently group related state changes
- **✅ Async support**: First-class support for asynchronous operations with WaitGroups
- **✅ Reactive primitives**: Signals, Effects, Computed, Context, Resources and more
- **✅ Streaming data**: Create signals from continuous data sources like CLI output, websockets, or events

## Installation

```bash
go get github.com/davidroman0O/firm-go
```

## Core Concepts

### Owner and Root

Firm-Go uses the concept of an "Owner" to manage the lifecycle of reactive primitives. The `Root` function creates a new owner and provides a way to safely wait for async operations:

```go
cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    // Create signals, effects, etc. owned by this owner
    
    // Optional cleanup to run when root is disposed
    return func() {
        fmt.Println("Root disposed")
    }
})

// Wait for all async operations to complete
wait()

// Later, clean up all resources
defer cleanup()
```

### Signals

Signals are the foundation of Firm-Go's reactivity. They hold values that can change over time:

```go
// Create a signal with an initial value
count := firm.Signal(owner, 0)

// Read the current value (tracks as dependency)
value := count.Get()

// Read without tracking
value := count.Peek()

// Update the value
count.Set(5)

// Update based on current value
count.Update(func(current int) int {
    return current + 1
})
```

### Effects

Effects run side effects when their dependencies change:

```go
// Effect with automatic dependency tracking
firm.Effect(owner, func() firm.CleanUp {
    fmt.Println("Count is now:", count.Get())
    
    // Return an optional cleanup function
    return func() {
        fmt.Println("Cleaning up after effect")
    }
}, nil) // nil means auto-track dependencies

// Effect with explicit dependencies
firm.Effect(owner, func() firm.CleanUp {
    fmt.Println("Count is now:", count.Get())
    return nil
}, []firm.Reactive{count})
```

### Computed Values and Memos

Computed values are derived from other reactive values:

```go
// Create a memo (computed value) from other signals
count := firm.Signal(owner, 5)

// Memo with automatic dependency tracking (nil)
doubled := firm.Memo(owner, func() int {
    return count.Get() * 2
}, nil) // nil means auto-track dependencies

// Read the computed value
fmt.Println("Doubled:", doubled.Get())
```

### Contexts

Contexts provide a way to pass values down through a reactive system:

```go
// Create a context with a default value
themeContext := firm.NewContext(owner, "light")

// In a child component:
firm.Effect(owner, func() firm.CleanUp {
    // Get the current theme
    theme := themeContext.Use()
    fmt.Println("Current theme:", theme)
    return nil
}, nil)

// Update the context
themeContext.Set("dark")

// Conditional rendering based on context
themeContext.Match(owner, "dark", func(childOwner *firm.Owner) firm.CleanUp {
    // This runs only when theme is "dark"
    return nil
})
```

### Resources

Resources handle asynchronous operations with built-in loading and error states:

```go
// Create a resource with an async fetcher
userResource := firm.Resource(owner, func() (User, error) {
    // Simulate API call
    time.Sleep(100 * time.Millisecond)
    return User{Name: "John", Age: 30}, nil
})

// Check loading state
firm.Effect(owner, func() firm.CleanUp {
    if userResource.Loading() {
        fmt.Println("Loading user...")
    } else if err := userResource.Error(); err != nil {
        fmt.Println("Error loading user:", err)
    } else {
        user := userResource.Data()
        fmt.Println("User loaded:", user.Name)
    }
    return nil
}, []firm.Reactive{userResource})

// Refresh data
userResource.Refetch()
```

### Streaming Data

Create signals that update from continuous data sources like CLI output, WebSockets, or events:

```go
// Create a signal from a continuous data source
output := firm.StreamSignal(owner, "", func(set func(string), done func()) {
    // Start a command or open a connection
    cmd := exec.Command("ping", "-c", "5", "example.com")
    stdout, _ := cmd.StdoutPipe()
    cmd.Start()
    
    // Read and update the signal with each line
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        set(scanner.Text())
    }
    
    // Mark as done when finished
    cmd.Wait()
    done()
})

// Use the streaming data like any other signal
firm.Effect(owner, func() firm.CleanUp {
    line := output.Get()
    fmt.Println("Output:", line)
    return nil
}, []firm.Reactive{output})
```

### Batching

Batch multiple updates to prevent cascading rerenders:

```go
firm.Batch(owner, func() {
    // These updates will be batched together
    firstName.Set("John")
    lastName.Set("Doe")
    age.Set(30)
    // Effects will only run once after the batch completes
})
```

### Async Operations

Firm-Go provides a robust way to track and wait for asynchronous operations:

```go
cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    // Track a pending operation
    owner.TrackPendingOp()
    
    go func() {
        // Do some async work
        time.Sleep(500 * time.Millisecond)
        
        // Signal completion
        owner.CompletePendingOp()
    }()
    
    return nil
})

// Wait for all async operations to complete
wait()

// Clean up
cleanup()
```

### Polling

Create values that automatically update on an interval:

```go
// Create a polling signal that updates every second
timePolling := firm.NewPolling(owner, func() time.Time {
    return time.Now()
}, time.Second)

// Use the polling value
firm.Effect(owner, func() firm.CleanUp {
    fmt.Println("Current time:", timePolling.Get().Format(time.RFC3339))
    return nil
}, []firm.Reactive{timePolling})

// Control the polling
timePolling.Pause()  // Stop polling
timePolling.Resume() // Resume polling
```

## Usage Examples

### Simple Counter

```go
cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    count := firm.Signal(owner, 0)
    doubled := firm.Memo(owner, func() int {
        return count.Get() * 2
    }, nil) // auto-tracking on ANY change - you should use `[]firm.Reactive{count}` 
    
    firm.Effect(owner, func() firm.CleanUp {
        fmt.Printf("Count: %d, Doubled: %d\n", count.Get(), doubled.Get())
        return nil
    }, nil)
    
    // Simulate updates
    count.Set(1)  // Logs: Count: 1, Doubled: 2
    count.Set(2)  // Logs: Count: 2, Doubled: 4
    
    return nil
})

wait()
defer cleanup()
```

### Data Fetching with Resources

```go
cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    userId := firm.Signal(owner, 1)
    
    userResource := firm.Resource(owner, func() (User, error) {
        id := userId.Get()
        return fetchUserById(id) // Your API function
    })
    
    firm.Effect(owner, func() firm.CleanUp {
        if userResource.Loading() {
            fmt.Println("Loading user...")
        } else if err := userResource.Error(); err != nil {
            fmt.Println("Error:", err)
        } else {
            user := userResource.Data()
            fmt.Println("User:", user.Name)
        }
        return nil
    }, []firm.Reactive{userResource})
    
    // Change user ID to trigger a new fetch
    userId.Set(2)
    
    return nil
})

// Wait for all async operations (including fetches)
wait()
defer cleanup()
```

### WebSocket Stream Example

```go
cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    // Create a signal from a WebSocket connection
    messages := firm.StreamSignal(owner, "", func(set func(string), done func()) {
        conn, _, err := websocket.DefaultDialer.Dial("ws://example.com/socket", nil)
        if err != nil {
            fmt.Println("Error connecting:", err)
            done()
            return
        }
        defer conn.Close()
        
        // Read messages until connection closes
        for {
            _, message, err := conn.ReadMessage()
            if err != nil {
                fmt.Println("Error reading:", err)
                break
            }
            
            // Update signal with each new message
            set(string(message))
        }
        
        done()
    })
    
    // Process incoming messages
    firm.Effect(owner, func() firm.CleanUp {
        msg := messages.Get()
        if msg != "" {
            fmt.Println("Received message:", msg)
            // Process message here
        }
        return nil
    }, []firm.Reactive{message})
    
    return nil
})

// Wait for stream to complete
wait()
defer cleanup()
```

### Debounced Search

```go
cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    search := firm.Signal(owner, "")
    
    // Debounced search query - updates 300ms after the source
    debouncedSearch := firm.Defer(owner, search, 300)
    
    firm.Effect(owner, func() firm.CleanUp {
        // Only runs when the debounced value changes
        query := debouncedSearch.Get()
        if query != "" {
            fmt.Println("Searching for:", query)
            // performSearch(query)
        }
        return nil
    }, nil)
    
    // These rapid updates only result in one search
    search.Set("a")
    search.Set("ap")
    search.Set("app")
    search.Set("appl")
    search.Set("apple")
    
    return nil
})

// Wait for debounced operations to complete
wait()
defer cleanup()
```

## Concurrency & Safety

Firm-Go is designed for concurrent Go applications with safety built-in:

```go
cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    count := firm.Signal(owner, 0)
    
    // Launch multiple goroutines updating the signal
    for i := 0; i < 10; i++ {
        owner.TrackPendingOp() // Track each goroutine
        
        go func(idx int) {
            defer owner.CompletePendingOp() // Signal completion
            
            // Atomic update of signal
            count.Update(func(v int) int {
                return v + 1
            })
            
            fmt.Printf("Goroutine %d updated count\n", idx)
        }(i)
    }
    
    return func() {
        fmt.Println("Final count:", count.Get()) // Should be 10
    }
})

// Wait for all goroutines to complete
wait()
defer cleanup()
```

## Advanced Features

### Derived Signals

Create signals that derive from others with two-way binding:

```go
cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    user := firm.Signal(owner, User{Name: "John", Age: 30})
    
    // Create a derived signal for the name field
    nameSignal := firm.DerivedSignal(
        owner, 
        user,
        // Getter
        func(u User) string {
            return u.Name
        },
        // Setter
        func(u User, name string) User {
            u.Name = name
            return u
        },
    )
    
    // Now nameSignal can be used as a regular signal
    fmt.Println("Name:", nameSignal.Get())
    
    // Update via the derived signal
    nameSignal.Set("Jane")
    
    // The original user signal is also updated
    fmt.Println("User:", user.Get().Name) // "Jane"
    
    return nil
})

wait()
defer cleanup()
```

### Untracking Dependencies

Sometimes you need to read signals without creating dependencies:

```go
firm.Effect(owner, func() firm.CleanUp {
    // This creates a dependency
    count := counter.Get()
    
    // This does not create a dependency
    config := firm.Untrack(owner, func() Config {
        return configSignal.Get()
    })
    
    fmt.Printf("Count: %d, Config: %v\n", count, config)
    return nil
}, nil)
```

## API Reference

### Signal

```go
Signal[T](owner, initialValue) -> *signalImpl[T]
  Methods:
    Get() -> T                     // Get with dependency tracking
    Peek() -> T                    // Get without tracking
    Set(value T)                   // Set a new value
    Update(fn func(T) T)           // Update functionally
```

### Effect

```go
Effect(owner, fn func() CleanUp, deps []Reactive)
```

### Memo

```go
Memo[T](owner, compute func() T, deps []Reactive) -> *signalImpl[T]
```

### Context

```go
NewContext[T](owner, defaultValue) -> *Context[T]
  Methods:
    Use() -> T                     // Get context value with tracking
    Set(value T)                   // Update context value
    Match(owner, value, fn) -> CleanUp // Run when value matches exactly
    When(owner, matcher, fn) -> CleanUp // Run when matcher returns true
```

### Resource

```go
Resource[T](owner, fetcher func() (T, error)) -> *resourceImpl[T]
  Methods:
    Loading() -> bool              // Check if loading
    Data() -> T                    // Get data
    Error() -> error               // Get error
    Refetch()                      // Fetch again
    OnLoad(fn func(T, error))      // Run when load completes
```

### StreamSignal

```go
StreamSignal[T](owner, initialValue T, setup func(set func(T), done func())) -> *signalImpl[T]
```

### Owner

```go
Root(fn func(owner *Owner) CleanUp) -> (cleanup CleanUp, wait func())

Owner Methods:
  TrackPendingOp()                 // Track an async operation
  CompletePendingOp()              // Signal completion of an async operation
  WaitForPending()                 // Wait for all tracked operations to complete
```

## Best Practices

### Wait for Async Operations

Always use the `wait()` function to ensure all async operations complete:

```go
cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    // Your reactive code with async operations
    return nil
})

// Wait for all async operations to complete
wait()

// Then clean up
defer cleanup()
```

### Balance Tracking and Completion

For every call to `TrackPendingOp()`, ensure there's a matching `CompletePendingOp()`:

```go
owner.TrackPendingOp()

go func() {
    defer owner.CompletePendingOp() // Always call this, even on error paths
    
    // Your async code
}()
```

### Use Mutexes for Shared Data

When using resources or sharing state across goroutines, use mutexes:

```go
var mu sync.Mutex
count := 0

owner.TrackPendingOp()
go func() {
    defer owner.CompletePendingOp()
    
    mu.Lock()
    count++
    mu.Unlock()
}()
```

### Clean Up Resources

Always return cleanup functions from effects that create resources:

```go
firm.Effect(owner, func() firm.CleanUp {
    connection := openConnection(url.Get())
    
    return func() {
        connection.Close() // Runs when effect reruns or owner is disposed
    }
}, nil)
```

### Use Explicit Dependencies When Possible

For performance and clarity, specify explicit dependencies when known:

```go
firm.Effect(owner, func() firm.CleanUp {
    fmt.Println("User:", firstName.Get(), lastName.Get())
    return nil
}, []firm.Reactive{firstName, lastName})
```

## License

MIT License