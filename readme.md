# Firm-Go

<div align="center">
  <h3>Fine-grained Reactive State Management for Go</h3>
  <p>A thread-safe, Solid.js-inspired reactive library for Go applications</p>

  ![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)
  ![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8.svg)
  ![Status](https://img.shields.io/badge/Status-Beta-yellow)
</div>

## 📚 Table of Contents

- [Introduction](#introduction)
- [Installation](#installation)
- [Core Concepts](#core-concepts)
  - [Signals](#signals)
  - [Effects](#effects)
  - [Computed Values and Memos](#computed-values-and-memos)
  - [Contexts](#contexts)
  - [Resources](#resources)
  - [Batching](#batching)
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
- **✅ Thread-safe**: All operations are protected by mutexes for concurrent Go applications
- **✅ Type-safe**: Built with Go generics for compile-time type checking
- **✅ Automatic cleanup**: Resources are automatically cleaned up when no longer needed
- **✅ Batched updates**: Efficiently group related state changes
- **✅ Async support**: First-class support for asynchronous operations with Resources
- **✅ Reactive primitives**: Signals, Effects, Computed, Context, Resources and more

## Installation

```bash
go get github.com/davidroman0O/firm-go
```

## Core Concepts

### Owner and Root

Firm-Go uses the concept of an "Owner" to manage the lifecycle of reactive primitives. The `Root` function creates a new owner:

```go
cleanup := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    // Create signals, effects, etc. owned by this owner
    
    // Optional cleanup to run when root is disposed
    return func() {
        fmt.Println("Root disposed")
    }
})

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
doubled := firm.Memo(owner, func() int {
    return count.Get() * 2
}, nil) // nil means auto-track dependencies

// Read the computed value
fmt.Println("Doubled:", doubled.Get())

// A more complex example with multiple dependencies
count := firm.Signal(owner, 5)
multiplier := firm.Signal(owner, 2)

// This memo tracks both count and multiplier
product := firm.Memo(owner, func() int {
    return count.Get() * multiplier.Get()
}, nil)

firm.Effect(owner, func() firm.CleanUp {
    fmt.Printf("Product is now: %d\n", product.Get())
    return nil
}, nil)

// When either signal changes, the memo updates
count.Set(10)      // Logs: "Product is now: 20"
multiplier.Set(3)  // Logs: "Product is now: 30"
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

// Conditional with custom matcher
themeContext.When(owner, func(theme string) bool {
    return theme == "light" || theme == "system"
}, func(childOwner *firm.Owner) firm.CleanUp {
    // This runs when theme is "light" or "system"
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

// Run function when data loads
userResource.OnLoad(func(data User, err error) {
    if err != nil {
        fmt.Println("Failed to load:", err)
    } else {
        fmt.Println("User loaded:", data.Name)
    }
})
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
timePolling.SetInterval(time.Minute) // Change interval
```

## Usage Examples

### Simple Counter

```go
cleanup := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    count := firm.Signal(owner, 0)
    doubled := firm.Memo(owner, func() int {
        return count.Get() * 2
    }, nil)
    
    firm.Effect(owner, func() firm.CleanUp {
        fmt.Printf("Count: %d, Doubled: %d\n", count.Get(), doubled.Get())
        return nil
    }, nil)
    
    // Simulate updates
    count.Set(1)  // Logs: Count: 1, Doubled: 2
    count.Set(2)  // Logs: Count: 2, Doubled: 4
    
    return nil
})
defer cleanup()
```

### Data Fetching with Resources

```go
cleanup := firm.Root(func(owner *firm.Owner) firm.CleanUp {
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
defer cleanup()
```

### Theme Context Example

```go
cleanup := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    theme := firm.NewContext(owner, "light")
    
    // Create UI with theme context
    createUI := func(childOwner *firm.Owner) firm.CleanUp {
        firm.Effect(childOwner, func() firm.CleanUp {
            currentTheme := theme.Use()
            fmt.Println("Rendering UI with theme:", currentTheme)
            return nil
        }, nil)
        
        return nil
    }
    
    // Create UI initially
    createUI(owner)
    
    // Change theme later
    theme.Set("dark")
    
    return nil
})
defer cleanup()
```

## Concurrency & Safety

Firm-Go is designed for concurrent Go applications. All operations are protected by mutexes:

```go
cleanup := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    count := firm.Signal(owner, 0)
    
    // Launch multiple goroutines updating the signal
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            // Atomic update of signal
            count.Update(func(v int) int {
                return v + 1
            })
        }()
    }
    
    wg.Wait()
    fmt.Println("Final count:", count.Get()) // Should be 10
    
    return nil
})
defer cleanup()
```

## Advanced Features

### Derived Signals

Create signals that derive from others with two-way binding:

```go
cleanup := firm.Root(func(owner *firm.Owner) firm.CleanUp {
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

### Defer

Create signals with delayed updates:

```go
cleanup := firm.Root(func(owner *firm.Owner) firm.CleanUp {
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
defer cleanup()
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

### Computed

```go
NewComputed[T](owner, compute func() T) -> *Computed[T]
  Methods:
    Get() -> T                     // Get the computed value
    Recompute() -> bool            // Force recomputation
```

### Batch

```go
Batch(owner, fn func())
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

### Polling

```go
NewPolling[T](owner, compute func() T, interval) -> *Polling[T]
  Methods:
    Get() -> T                     // Get current value
    SetInterval(interval)          // Change polling interval
    Pause()                        // Pause polling
    Resume()                       // Resume polling
```

## Best Practices

### Memory Management

Always call cleanup functions to prevent memory leaks:

```go
cleanup := firm.Root(func(owner *firm.Owner) firm.CleanUp {
    // Your reactive code
    return nil
})
defer cleanup() // Important!
```

### Batch Updates for Performance

Use batching for multiple related updates:

```go
firm.Batch(owner, func() {
    firstName.Set("John")
    lastName.Set("Doe")
    email.Set("john.doe@example.com")
})
```

### Minimize Dependency Tracking

Use `Peek()` when you don't need reactivity:

```go
// This creates a dependency - effect will rerun when configSignal changes
config := configSignal.Get()

// This doesn't create a dependency
config := configSignal.Peek()
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
