package main

import (
	"fmt"

	"github.com/davidroman0O/firm-go"
)

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		firstName := firm.Signal(owner, "John")
		lastName := firm.Signal(owner, "Doe")
		age := firm.Signal(owner, 30)

		// Count effect runs
		effectRuns := 0

		// Effect that uses all three signals
		firm.Effect(owner, func() firm.CleanUp {
			effectRuns++
			fmt.Printf("Effect run #%d: %s %s, age %d\n",
				effectRuns, firstName.Get(), lastName.Get(), age.Get())
			return nil
		}, nil)

		// Individual updates - will trigger effect 3 times
		fmt.Println("\nIndividual updates:")
		firstName.Set("Jane")
		lastName.Set("Smith")
		age.Set(28)

		// Batched updates - will trigger effect only once
		fmt.Println("\nBatched updates:")
		firm.Batch(owner, func() {
			firstName.Set("Bob")
			lastName.Set("Johnson")
			age.Set(42)
		})

		fmt.Printf("\nTotal effect runs: %d\n", effectRuns)

		return nil
	})

	wait()
	cleanup()
}
