package main

import (
	"fmt"
	"time"

	"github.com/davidroman0O/firm-go"
)

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		source := firm.Signal(owner, "initial")

		// Effect with resource acquisition and cleanup
		firm.Effect(owner, func() firm.CleanUp {
			value := source.Get()
			fmt.Println("Setting up for:", value)

			// Simulate resource acquisition
			startTime := time.Now()

			// Return cleanup function
			return func() {
				duration := time.Since(startTime)
				fmt.Printf("Cleaning up %s after %v\n", value, duration)
			}
		}, nil)

		// Each change causes previous cleanup to run
		time.Sleep(100 * time.Millisecond)
		source.Set("second value")

		time.Sleep(200 * time.Millisecond)
		source.Set("third value")

		return nil
	})

	wait()
	cleanup() // Final cleanup runs here
}
