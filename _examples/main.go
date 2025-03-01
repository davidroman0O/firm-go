package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/davidroman0O/firm-go"
)

type User struct {
	Name     string
	Age      int
	Address  Address
	Contacts []string
}

type Address struct {
	Street string
	City   string
	Zip    string
}

func main() {
	fmt.Println("=== Basic Signals ===")
	basicSignalsExample()

	fmt.Println("\n=== Computed Values ===")
	computedExample()

	fmt.Println("\n=== Effects ===")
	effectsExample()

	fmt.Println("\n=== Batch Updates ===")
	batchingExample()

	fmt.Println("\n=== Resources ===")
	resourceExample()

	fmt.Println("\n=== Store API ===")
	storeExample()

	fmt.Println("\n=== Accessor API ===")
	accessorExample()

	fmt.Println("\n=== Derived Signals ===")
	derivedSignalExample()

	fmt.Println("\n=== Root Context ===")
	rootContextExample()
}

func basicSignalsExample() {
	// Create a signal with initial value
	count := firm.NewSignal(0)

	// Subscribe to changes
	unsub := count.Subscribe(func(newValue int) {
		fmt.Printf("Count changed to: %d\n", newValue)
	})

	// Update the signal
	count.Set(1)
	count.Update(func(current int) int {
		return current + 1
	})

	// Unsubscribe
	unsub()

	// This update won't trigger the callback
	count.Set(10)
	fmt.Printf("Final count (using Peek): %d\n", count.Peek())
}

func computedExample() {
	// Create signals
	firstName := firm.NewSignal("John")
	lastName := firm.NewSignal("Doe")

	// Create a computed value
	fullName := firm.NewComputed(func() string {
		return firstName.Get() + " " + lastName.Get()
	})

	// Create a memo (optimized computed with custom equality)
	nameLength := firm.CreateMemo(func() int {
		return len(fullName.Get())
	}, func(a, b int) bool {
		// Only update if the difference is more than 1 character
		return a-b > -2 && a-b < 2
	})

	// Display initial values
	fmt.Printf("Full name: %s, length: %d\n", fullName.Get(), nameLength.Get())

	// Update signals
	firstName.Set("Jane")
	fmt.Printf("After update - Full name: %s, length: %d\n", fullName.Get(), nameLength.Get())

	// This should not update the nameLength due to our custom equality function
	// (difference is only 1 character)
	lastName.Set("Doe ")
	fmt.Printf("After small change - Full name: %s, length: %d\n", fullName.Get(), nameLength.Get())

	// This should update the nameLength (difference is more than 1 character)
	lastName.Set("Johnson")
	fmt.Printf("After big change - Full name: %s, length: %d\n", fullName.Get(), nameLength.Get())
}

func effectsExample() {
	// Create signals
	text := firm.NewSignal("Hello")
	count := firm.NewSignal(0)

	// Create an effect with cleanup
	effect := firm.CreateEffect(func() {
		value := text.Get()
		fmt.Printf("Text effect: %s\n", value)

		// Register cleanup function
		firm.OnCleanup(func() {
			fmt.Printf("Cleaning up effect for: %s\n", value)
		})
	})

	// Create an effect with untracked read
	firm.CreateEffect(func() {
		// This dependency is tracked
		fmt.Printf("Count changed to: %d\n", count.Get())

		// This is read without tracking
		untrackedValue := firm.Untrack(func() string {
			return text.Get()
		})
		fmt.Printf("Untracked value: %s\n", untrackedValue)
	})

	// Update signals
	text.Set("World")
	count.Set(1)

	// Changes to text won't affect the second effect
	text.Set("Signals")
	count.Set(2)

	// Dispose an effect
	effect.Dispose()
	text.Set("After Dispose") // Won't trigger the first effect

	fmt.Println("Effects test complete")
}

func batchingExample() {
	counter := firm.NewSignal(0)
	effectCount := 0

	// Create an effect to track how many times it runs
	firm.CreateEffect(func() {
		// Just access the counter to create dependency
		_ = counter.Get()
		effectCount++
	})

	// Reset the count after the initial effect run
	effectCount = 0

	// Individual updates
	fmt.Println("Individual updates:")
	counter.Set(1)
	counter.Set(2)
	counter.Set(3)
	fmt.Printf("Effect ran %d times\n", effectCount)

	// Reset count
	effectCount = 0

	// Batched updates
	fmt.Println("Batched updates:")
	firm.Batch(func() {
		counter.Set(10)
		counter.Set(20)
		counter.Set(30)
		// Manually check the value inside batch
		fmt.Printf("Counter inside batch: %d\n", counter.Get())
	})

	fmt.Printf("Counter after batch: %d\n", counter.Get())
	fmt.Printf("Effect ran %d time\n", effectCount)
}

func resourceExample() {
	// Simulate an async API call
	fetchUser := func() (User, error) {
		// Simulate network delay
		time.Sleep(100 * time.Millisecond)

		// Randomly succeed or fail to demonstrate error handling
		if time.Now().UnixNano()%2 == 0 {
			return User{
				Name: "John Doe",
				Age:  30,
				Address: Address{
					Street: "123 Main St",
					City:   "Anytown",
					Zip:    "12345",
				},
				Contacts: []string{"john@example.com", "555-1234"},
			}, nil
		}

		return User{}, errors.New("failed to fetch user")
	}

	// Create a resource
	userResource := firm.CreateResource(fetchUser)

	// Initial state (should be loading)
	state := userResource.Read()
	fmt.Printf("Initial state - Loading: %v, Error: %v\n", state.Loading, state.Error)

	// Wait for the resource to complete loading
	time.Sleep(200 * time.Millisecond)

	// Check the result
	state = userResource.Read()
	if state.Error != nil {
		fmt.Printf("Resource error: %v\n", state.Error)
	} else {
		fmt.Printf("Resource loaded - User: %s, Age: %d\n", state.Data.Name, state.Data.Age)
	}

	// Manually trigger a refetch
	fmt.Println("Refetching resource...")
	userResource.Refetch()

	// Wait for the refetch to complete
	time.Sleep(200 * time.Millisecond)

	// Check the result again
	state = userResource.Read()
	fmt.Printf("After refetch - Loading: %v, Has error: %v\n", state.Loading, state.Error != nil)
	if state.Error == nil {
		fmt.Printf("User data: %+v\n", state.Data)
	}
}

func storeExample() {
	// Create a store with a complex object
	userStore := firm.CreateStore(User{
		Name: "John Doe",
		Age:  30,
		Address: Address{
			Street: "123 Main St",
			City:   "Anytown",
			Zip:    "12345",
		},
		Contacts: []string{"john@example.com", "555-1234"},
	})

	// Create effect to watch for changes
	firm.CreateEffect(func() {
		user := userStore.Get()
		fmt.Printf("User updated: %s, %d, %s\n", user.Name, user.Age, user.Address.City)
	})

	// Update individual properties using the path API
	fmt.Println("Updating the user's name...")
	userStore.SetPath([]string{"Name"}, "Jane Doe")

	// Update nested property
	fmt.Println("Updating the user's city...")
	userStore.SetPath([]string{"Address", "City"}, "New City")

	// Update array element
	fmt.Println("Updating contact email...")
	userStore.SetPath([]string{"Contacts", "0"}, "jane@example.com")

	// Create middleware for logging
	userStore.Use(func(path []string, newValue any, oldValue any) any {
		fmt.Printf("Store middleware - Path: %v, Old: %v, New: %v\n", path, oldValue, newValue)
		return newValue // Return the value unchanged
	})

	// This update will trigger the middleware
	fmt.Println("Updating with middleware...")
	userStore.SetPath([]string{"Age"}, 31)

	// Get the final state
	finalUser := userStore.Get()
	fmt.Printf("Final store state: %+v\n", finalUser)
	fmt.Printf("Address: %+v\n", finalUser.Address)
	fmt.Printf("Contacts: %v\n", finalUser.Contacts)
}

func accessorExample() {
	// Create an accessor (similar to Solid's createSignal but with single function API)
	count := firm.NewAccessor(0)

	// Use the accessor as a getter
	value := count.Call().(int)
	fmt.Printf("Initial accessor value: %d\n", value)

	// Use the accessor as a setter
	count.Call(10)

	// Get the updated value
	value = count.Call().(int)
	fmt.Printf("Updated accessor value: %d\n", value)

	// Create a computed value that depends on the accessor
	doubled := firm.NewComputed(func() int {
		return count.Call().(int) * 2
	})

	fmt.Printf("Computed from accessor: %d\n", doubled.Get())

	// Update through the accessor again
	count.Call(20)
	fmt.Printf("Computed after update: %d\n", doubled.Get())
}

func derivedSignalExample() {
	// Create a base signal
	user := firm.NewSignal(User{
		Name: "John Doe",
		Age:  30,
		Address: Address{
			Street: "123 Main St",
			City:   "Anytown",
			Zip:    "12345",
		},
	})

	// Create a selector for a specific property
	nameSignal := firm.Selector(user, func(u User) string {
		return u.Name
	})

	fmt.Printf("Selected name: %s\n", nameSignal.Get())

	// Create a derived signal with custom getter and setter
	ageSignal := firm.NewDerivedSignal(
		user,
		// Getter extracts age from user
		func(u User) int {
			return u.Age
		},
		// Setter updates age in user
		func(u User, newAge int) User {
			u.Age = newAge
			return u
		},
	)

	fmt.Printf("Derived age: %d\n", ageSignal.Get())

	// Update through the derived signal
	ageSignal.Set(40)

	// Check that the update affected the original signal
	updatedUser := user.Get()
	fmt.Printf("User after derived update - Name: %s, Age: %d\n",
		updatedUser.Name, updatedUser.Age)

	// Update the original signal
	updatedUser.Name = "Jane Doe"
	user.Set(updatedUser)

	// Check that the selector was updated
	fmt.Printf("Selected name after update: %s\n", nameSignal.Get())
}

func rootContextExample() {
	// Create a root context
	dispose := firm.CreateRoot(func() {
		// Signals created here will be disposed when the root is disposed
		counter := firm.NewSignal(0)

		// Create an effect in this context
		firm.CreateEffect(func() {
			value := counter.Get()
			fmt.Printf("Root context effect: %d\n", value)
		})

		// Update within the context
		counter.Set(1)
		counter.Set(2)

		// Child context
		childDispose := firm.CreateRoot(func() {
			childCounter := firm.NewSignal(100)

			firm.CreateEffect(func() {
				parentValue := counter.Get()
				childValue := childCounter.Get()
				fmt.Printf("Child context effect: parent=%d, child=%d\n",
					parentValue, childValue)
			})

			childCounter.Set(101)
		})

		// Dispose the child context
		fmt.Println("Disposing child context...")
		childDispose()

		// Updates still work in parent context
		counter.Set(3)
	})

	// Dispose the root context
	fmt.Println("Disposing root context...")
	dispose()
	fmt.Println("Root context disposed.")
}
