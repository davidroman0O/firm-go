package main

import (
	"fmt"
	"time"

	"github.com/davidroman0O/firm-go"
)

type User struct {
	ID   int
	Name string
	Age  int
}

// Simulate API call
func fetchUser(id int) (User, error) {
	// Simulate network delay
	time.Sleep(200 * time.Millisecond)

	// Return mock data
	return User{
		ID:   id,
		Name: fmt.Sprintf("User %d", id),
		Age:  20 + id,
	}, nil
}

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		// Signal for user ID
		userId := firm.Signal(owner, 1)

		// Create user resource
		userResource := firm.Resource(owner, func() (User, error) {
			// This runs when resource is created or refetched
			id := userId.Get()
			fmt.Printf("Fetching user with ID: %d\n", id)
			return fetchUser(id)
		})

		// Track resource state
		firm.Effect(owner, func() firm.CleanUp {
			if userResource.Loading() {
				fmt.Println("⏳ Loading user...")
			} else if err := userResource.Error(); err != nil {
				fmt.Println("❌ Error:", err)
			} else {
				user := userResource.Data()
				fmt.Printf("✅ User loaded: %s (age %d)\n", user.Name, user.Age)
			}
			return nil
		}, []firm.Reactive{userResource})

		// After some time, change the user ID to trigger a refetch
		time.Sleep(300 * time.Millisecond)
		fmt.Println("\nChanging user ID...")
		userId.Set(2)

		// Manually trigger a refetch for the same user
		time.Sleep(300 * time.Millisecond)
		fmt.Println("\nRefetching current user...")
		userResource.Refetch()

		return nil
	})

	wait() // Wait for async operations to complete
	cleanup()
}
