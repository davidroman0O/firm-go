package main

import (
	"fmt"

	"github.com/davidroman0O/firm-go"
)

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		// Create a signal with initial value
		count := firm.Signal(owner, 0)

		// Read signal value (with dependency tracking)
		fmt.Println("Initial count:", count.Get())

		// Update signal value
		count.Set(5)

		fmt.Println("After set:", count.Get())

		// Update signal using a function
		count.Update(func(v int) int {
			return v * 2
		})

		fmt.Println("After update:", count.Get()) // 10

		// Read without tracking (no dependency)
		fmt.Println("Peek value:", count.Peek())

		return nil
	})

	wait()
	cleanup()
}
