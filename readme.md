# Firm-Go

A comprehensive library that brings Solid.js-style reactivity to Go, featuring signals, computed values, effects, resources, and stores.

> WORK IN PROGRESS: fr fr i'm just checking the concept and ideas for now

## Overview

Firm-Go is a Go library that implements a reactive programming model inspired by Solid.js. It provides primitives for creating and managing reactive state in Go applications in an idiomatic way.

The core concepts include:

- **Signals**: Reactive values that notify subscribers when they change
- **Computed**: Derived values that automatically update when their dependencies change
- **Memo**: Optimized computed values with custom equality functions
- **Effects**: Side effects that run when their dependencies change
- **Resources**: Async data fetching with loading/error states
- **Stores**: Reactive objects with nested updates
- **Contexts**: Scoped reactivity with cleanup
- **Derived Signals**: Two-way bindable derived state
- **Accessors**: Getter/setter combined functions (like in Solid.js)

## Features

- **Core Reactivity**
  - Fine-grained dependency tracking
  - Automatic subscription management
  - Immutable update patterns
  - Batch processing for efficient updates

- **Type Safety**
  - Generic type support for all reactive primitives
  - Compile-time type checking

- **Advanced Features**
  - Resource management for async operations
  - Context-based cleanup
  - Nested path updates for complex state
  - Middleware support for stores
  - Two-way binding for derived values
  - Custom equality functions

- **Performance**
  - Minimal re-computation through dependency tracking
  - Batching to reduce update cascades
  - Memoization with equality checks

## Installation

```bash
go get github.com/davidroman0O/firm-go
```

## Usage

### Signals

```go
import "github.com/davidroman0O/firm-go"

// Create a signal with an initial value
count := firm.NewSignal(0)

// Read the current value (tracks dependency)
value := count.Get()

// Read without tracking (does not create dependency)
untracked := count.Peek()

// Update the value
count.Set(5)

// Update based on current value
count.Update(func(current int) int {
    return current + 1
})

// Subscribe to changes
unsubscribe := count.Subscribe(func(newValue int) {
    fmt.Printf("Value changed to: %d\n", newValue)
})

// Stop receiving updates
unsubscribe()
```

### Computed Values

```go
import "github.com/davidroman0O/firm-go"

// Create signals
x := firm.NewSignal(5)
y := firm.NewSignal(10)

// Create a computed value that depends on both signals
sum := firm.NewComputed(func() int {
    return x.Get() + y.Get()
})

// The computed value automatically updates when dependencies change
fmt.Println(sum.Get()) // 15
x.Set(10)
fmt.Println(sum.Get()) // 20

// Peek at the value without tracking it as a dependency
peekedValue := sum.Peek()

// Force recomputation when needed (e.g., after untracked dependencies change)
sum.ForceComputation()
```

### Memo

```go
import "github.com/davidroman0O/firm-go"
import "math"

// Create a memo with custom equality function
memo := firm.CreateMemo(
    // Compute function
    func() string {
        return fmt.Sprintf("%s %s", firstName.Get(), lastName.Get())
    },
    // Equality function
    func(a, b string) bool {
        // Only update if the length difference is more than 2 characters
        return math.Abs(float64(len(a) - len(b))) <= 2
    },
)

// Memo behaves like a computed but skips updates when the equality function returns true
```

### Effects

```go
import "github.com/davidroman0O/firm-go"

// Create a signal
message := firm.NewSignal("Hello")

// Create an effect that runs when the signal changes
effect := firm.CreateEffect(func() {
    fmt.Println("Message changed:", message.Get())
    
    // Register cleanup function
    firm.OnCleanup(func() {
        fmt.Println("Cleaning up effect")
    })
})

// Updating the signal triggers the effect
message.Set("World")

// Dispose the effect when no longer needed
effect.Dispose()
```

### Batching Updates

```go
import "github.com/davidroman0O/firm-go"

firm.Batch(func() {
    // Multiple updates are batched into a single notification
    count.Set(1)
    count.Set(2)
    count.Set(3)
    // Only the final value (3) will trigger effects
})
```

### Untracked Reads

```go
import "github.com/davidroman0O/firm-go"

// Create a computed value with mixed tracking
mixed := firm.NewComputed(func() int {
    // This is tracked - changes will trigger recomputation
    aValue := a.Get()
    
    // This is not tracked - changes won't trigger recomputation
    bValue := firm.Untrack(func() int {
        return b.Get()
    })
    
    return aValue + bValue
})

// After an untracked dependency changes, you might need to force recomputation
b.Set(20) // This won't trigger recomputation
mixed.ForceComputation() // Manually trigger recomputation to see latest b value
```

### Resources

```go
import "github.com/davidroman0O/firm-go"
import "time"

// Create a resource with an async fetcher
userResource := firm.CreateResource(func() (User, error) {
    // Simulate an API call
    time.Sleep(100 * time.Millisecond)
    return User{Name: "John", Age: 30}, nil
})

// Check if the resource is loading
if userResource.Loading() {
    fmt.Println("Loading user data...")
}

// Check for errors
if err := userResource.Error(); err != nil {
    fmt.Println("Error loading user:", err)
}

// Access the data
user := userResource.Data()
fmt.Println("User:", user.Name)

// Manually trigger a refetch
userResource.Refetch()
```

### Reactive Store

```go
import "github.com/davidroman0O/firm-go"

// Create a store with initial state
userStore := firm.CreateStore(User{
    Name: "John",
    Age:  30,
    Address: Address{
        Street: "123 Main St",
        City:   "Anytown",
    },
})

// Get the current state
user := userStore.Get()

// Set the entire state
userStore.Set(User{Name: "Jane", Age: 25})

// Update a specific path
userStore.SetPath([]string{"Name"}, "Bob")

// Update a nested path
userStore.SetPath([]string{"Address", "City"}, "Newtown")

// Add middleware
userStore.Use(func(path []string, value any, oldValue any) any {
    fmt.Printf("Updating path %v: %v -> %v\n", path, oldValue, value)
    return value // Return the value (possibly modified)
})
```

### Context Management

```go
import "github.com/davidroman0O/firm-go"

// Create a root context
dispose := firm.CreateRoot(func() {
    // Signals and effects created here will be automatically disposed
    count := firm.NewSignal(0)
    
    firm.CreateEffect(func() {
        fmt.Println("Count:", count.Get())
    })
    
    // Register cleanup
    firm.OnCleanup(func() {
        fmt.Println("Cleaning up resources")
    })
})

// Dispose the context and clean up all resources
dispose()
```

### Accessor API

```go
import "github.com/davidroman0O/firm-go"

// Create an accessor (similar to Solid's createSignal)
count := firm.NewAccessor(0)

// Use as getter
value := count.Call().(int)

// Use as setter
count.Call(10)

// Can be used in computed values too
doubled := firm.NewComputed(func() int {
    return count.Call().(int) * 2
})
```

### Derived Signals

```go
import "github.com/davidroman0O/firm-go"

// Create a base signal
user := firm.NewSignal(User{Name: "John", Age: 30})

// Create a selector for a specific property
nameSignal := firm.Selector(user, func(u User) string {
    return u.Name
})

// Create a two-way derived signal
ageSignal := firm.NewDerivedSignal(
    user,
    // Getter
    func(u User) int {
        return u.Age
    },
    // Setter
    func(u User, newAge int) User {
        u.Age = newAge
        return u
    },
)

// Use the derived signal
fmt.Println("Age:", ageSignal.Get())
ageSignal.Set(40) // Updates the original signal
```

## Real-World Example

Here's a more complete example showing how these concepts work together:

```go
package main

import (
	"fmt"
	"time"
	"github.com/davidroman0O/firm-go"
)

type TodoItem struct {
	ID        int
	Text      string
	Completed bool
}

func main() {
	// Create a store for our todo list
	todoStore := firm.CreateStore([]TodoItem{
		{ID: 1, Text: "Learn Go", Completed: true},
		{ID: 2, Text: "Learn Firm-Go", Completed: false},
	})
	
	// Computed for active todos
	activeTodos := firm.NewComputed(func() []TodoItem {
		todos := todoStore.Get()
		result := make([]TodoItem, 0)
		for _, todo := range todos {
			if !todo.Completed {
				result = append(result, todo)
			}
		}
		return result
	})
	
	// Computed for completed todos
	completedTodos := firm.NewComputed(func() []TodoItem {
		todos := todoStore.Get()
		result := make([]TodoItem, 0)
		for _, todo := range todos {
			if todo.Completed {
				result = append(result, todo)
			}
		}
		return result
	})
	
	// Effect to log changes
	firm.CreateEffect(func() {
		todos := todoStore.Get()
		active := len(activeTodos.Get())
		completed := len(completedTodos.Get())
		fmt.Printf("Todos: %d total, %d active, %d completed\n", 
			len(todos), active, completed)
	})
	
	// Function to add a todo
	addTodo := func(text string) {
		todos := todoStore.Get()
		newID := len(todos) + 1
		todoStore.Set(append(todos, TodoItem{
			ID:        newID,
			Text:      text,
			Completed: false,
		}))
	}
	
	// Function to toggle a todo
	toggleTodo := func(id int) {
		todos := todoStore.Get()
		for i, todo := range todos {
			if todo.ID == id {
				// Create a new array with the updated item
				newTodos := make([]TodoItem, len(todos))
				copy(newTodos, todos)
				newTodos[i].Completed = !newTodos[i].Completed
				todoStore.Set(newTodos)
				break
			}
		}
	}
	
	// Add some todos
	addTodo("Build an app")
	
	// Toggle a todo
	toggleTodo(2)
	
	// Access the computed values
	fmt.Println("\nActive todos:")
	for _, todo := range activeTodos.Get() {
		fmt.Printf("- %s\n", todo.Text)
	}
	
	fmt.Println("\nCompleted todos:")
	for _, todo := range completedTodos.Get() {
		fmt.Printf("- %s\n", todo.Text)
	}
	
	// Create a resource that fetches todos from a server
	todoResource := firm.CreateResource(func() ([]TodoItem, error) {
		// Simulating an API call
		time.Sleep(100 * time.Millisecond)
		return []TodoItem{
			{ID: 100, Text: "Remote todo 1", Completed: false},
			{ID: 101, Text: "Remote todo 2", Completed: true},
		}, nil
	})
	
	// Wait for the resource to load
	time.Sleep(200 * time.Millisecond)
	
	// Check the resource
	if todoResource.Loading() {
		fmt.Println("\nStill loading remote todos...")
	} else if err := todoResource.Error(); err != nil {
		fmt.Println("\nError loading remote todos:", err)
	} else {
		fmt.Println("\nRemote todos loaded:")
		for _, todo := range todoResource.Data() {
			fmt.Printf("- %s (Completed: %v)\n", todo.Text, todo.Completed)
		}
	}
}
```

## Important Concepts

### Dependency Tracking

Firm-Go automatically tracks dependencies when you call `.Get()` on a signal. Any computed value or effect that reads a signal will automatically re-run when that signal changes.

### Untracked Dependencies

Sometimes you need to read a signal without creating a dependency. Use the `Untrack` function or `Peek()` method for this:

```go
// Read a value without creating a dependency
untracked := firm.Untrack(func() {
    return signal.Get()
})

// Or use Peek
value := signal.Peek()
```

**Important Note**: When using untracked dependencies, changes to those dependencies won't automatically trigger recomputation. If you need to force a recomputation to see the latest values, use `ForceComputation()`:

```go
// Force a computed value to recompute using the latest values
myComputed.ForceComputation()
```

## Advanced Patterns

### Composing Reactive Logic

You can create custom reactive primitives by composing the basic building blocks:

```go
import "github.com/davidroman0O/firm-go"

// Create a custom reactive counter with increment/decrement methods
type Counter struct {
	value   *firm.Signal[int]
	doubled *firm.Computed[int]
}

func NewCounter(initial int) *Counter {
	value := firm.NewSignal(initial)
	
	doubled := firm.NewComputed(func() int {
		return value.Get() * 2
	})
	
	return &Counter{
		value: value,
		doubled: doubled,
	}
}

func (c *Counter) Get() int {
	return c.value.Get()
}

func (c *Counter) Doubled() int {
	return c.doubled.Get()
}

func (c *Counter) Increment() {
	c.value.Update(func(v int) int {
		return v + 1
	})
}

func (c *Counter) Decrement() {
	c.value.Update(func(v int) int {
		return v - 1
	})
}
```

### Reactive Forms

```go
import "github.com/davidroman0O/firm-go"

type FormState struct {
	Values     map[string]string
	Errors     map[string]string
	Touched    map[string]bool
	Submitting bool
}

func CreateForm(initialValues map[string]string) *firm.Store[FormState] {
	return firm.CreateStore(FormState{
		Values:     initialValues,
		Errors:     make(map[string]string),
		Touched:    make(map[string]bool),
		Submitting: false,
	})
}

// Usage
form := CreateForm(map[string]string{"email": "", "password": ""})

// Set a field value
form.SetPath([]string{"Values", "email"}, "user@example.com")

// Mark as touched
form.SetPath([]string{"Touched", "email"}, true)

// Set validation error
form.SetPath([]string{"Errors", "email"}, "Invalid email format")

// Create a computed for form validity
isValid := firm.NewComputed(func() bool {
	state := form.Get()
	return len(state.Errors) == 0
})
```

## Performance Considerations

- **Use Peek/GetUntracked** when you need to read a signal without creating a dependency
- **Use Batch** when making multiple updates to avoid cascading reactions
- **Use Memo** with custom equality functions to prevent unnecessary updates
- **Use ForceComputation** when you need to refresh computed values after untracked dependencies change
- **Dispose** effects and contexts when they're no longer needed
- **Use Resources** for asynchronous operations to manage loading states

## Thread Safety

Firm-Go uses mutexes to provide thread safety for basic operations. However, complex state updates should be batched to avoid race conditions and ensure consistency.

## Differences from Solid.js

While Firm-Go aims to provide a similar API to Solid.js, there are some differences due to the nature of Go:

1. **Static Typing**: Firm-Go uses generics for type safety
2. **Garbage Collection**: Go's garbage collection means we need explicit disposal for cleanup
3. **Concurrency Model**: Go's goroutines require thread-safe reactivity primitives
4. **Error Handling**: Resources in Firm-Go explicitly handle errors
5. **Manual Recomputation**: Sometimes you need to manually force recomputation with `ForceComputation()`

## Limitations

- This library is primarily designed for state management
- Complex nested updates in stores may not be as ergonomic as in JavaScript
- Go's type system sometimes requires more verbose code than Solid.js

## Best Practices

1. **Use signals for primitive values** that change frequently
2. **Use computed values for derived state** rather than duplicating logic
3. **Use effects for side effects** like logging or API calls
4. **Use stores for complex objects** with nested properties
5. **Use resources for async operations** to track loading and error states
6. **Use batching for multiple updates** to avoid cascading updates
7. **Use untracking judiciously** for optimization
8. **Use ForceComputation when needed** for untracked dependencies
9. **Clean up effects and contexts** to avoid memory leaks

## Common Pitfalls

1. **Circular dependencies** between computed values can cause stack overflows
2. **Forgetting to dispose effects** can lead to memory leaks
3. **Mutating objects directly** instead of using signals or stores
4. **Not forcing computation** when using untracked dependencies
5. **Overusing effects** for derived state instead of computed values

## Future Improvements

- **Optimized Dependency Tracking**: More efficient tracking of dependencies
- **Better Context API**: More powerful context propagation
- **Debugging Tools**: Better visualization of reactive dependencies

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.