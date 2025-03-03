package main

import (
	"fmt"

	"github.com/davidroman0O/firm-go"
)

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		width := firm.Signal(owner, 5)
		height := firm.Signal(owner, 10)

		// Memo computes area, auto-tracking width and height
		area := firm.Memo(owner, func() int {
			// Both signals are auto-tracked as dependencies
			return width.Get() * height.Get()
		}, nil) // Auto-track dependencies

		fmt.Println("Initial area:", area.Get()) // 50

		// Update dependency
		width.Set(7)
		fmt.Println("Updated area:", area.Get()) // 70

		// Track computed value with an effect
		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("Area changed: %d (w=%d, h=%d)\n",
				area.Get(), width.Get(), height.Get())
			return nil
		}, nil)

		height.Set(12) // Triggers effect: Area changed: 84

		return nil
	})

	wait()
	cleanup()
}
