package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/davidroman0O/firm-go"
)

type Post struct {
	ID    int
	Title string
	Body  string
}

// Simulate API call with possible errors
func fetchPost(id int) (Post, error) {
	// Simulate network delay
	time.Sleep(100 * time.Millisecond)

	// Return error for certain IDs
	if id <= 0 {
		return Post{}, errors.New("invalid post ID")
	}

	if id > 100 {
		return Post{}, errors.New("post not found")
	}

	// Return mock data
	return Post{
		ID:    id,
		Title: fmt.Sprintf("Post %d", id),
		Body:  fmt.Sprintf("Content of post %d", id),
	}, nil
}

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		// Signal for post ID
		postId := firm.Signal(owner, 1)

		// Create post resource
		postResource := firm.Resource(owner, func() (Post, error) {
			id := postId.Get()
			return fetchPost(id)
		})

		// React to resource state changes
		firm.Effect(owner, func() firm.CleanUp {
			if postResource.Loading() {
				fmt.Println("⏳ Loading post...")
			} else if err := postResource.Error(); err != nil {
				fmt.Println("❌ Error:", err)
			} else {
				post := postResource.Data()
				fmt.Printf("✅ Post loaded: %s\n", post.Title)
				fmt.Printf("   Body: %s\n", post.Body)
			}
			return nil
		}, []firm.Reactive{postResource})

		// Test different scenarios
		time.Sleep(200 * time.Millisecond)

		fmt.Println("\nTrying valid post ID:")
		postId.Set(42)
		time.Sleep(200 * time.Millisecond)

		fmt.Println("\nTrying invalid post ID:")
		postId.Set(-1)
		time.Sleep(200 * time.Millisecond)

		fmt.Println("\nTrying non-existent post ID:")
		postId.Set(101)
		time.Sleep(200 * time.Millisecond)

		fmt.Println("\nGoing back to valid ID:")
		postId.Set(5)

		return nil
	})

	wait() // Wait for async operations to complete
	cleanup()
}
