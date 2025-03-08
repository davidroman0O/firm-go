package main

import (
	"fmt"

	"github.com/davidroman0O/firm-go"
)

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		count := firm.Signal(owner, 0)
		message := firm.Signal(owner, "Hello")

		// Effect with explicit dependencies - only watches count
		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("Effect running: count=%d, message=%s\n",
				count.Get(), message.Peek()) // note Peek() for message

			return nil
		}, []firm.Reactive{count}) // Only count is a dependency

		// Only this triggers the effect
		count.Set(1)         // Effect runs
		message.Set("World") // Effect does NOT run

		return nil
	})

	wait()
	cleanup()
}
