package main

import (
	"fmt"

	"github.com/davidroman0O/firm-go"
)

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		firstName := firm.Signal(owner, "John")
		lastName := firm.Signal(owner, "Doe")
		title := firm.Signal(owner, "Mr.")

		// Memo with explicit dependencies - only firstName and lastName
		fullName := firm.Memo(owner, func() string {
			// We explicitly list dependencies, so title won't trigger updates
			// even though we access it with Get()
			return fmt.Sprintf("%s %s %s",
				title.Get(), firstName.Get(), lastName.Get())
		}, []firm.Reactive{firstName, lastName}) // Explicit dependencies

		fmt.Println("Initial name:", fullName.Get()) // "Mr. John Doe"

		// These trigger a recomputation
		firstName.Set("Jane")
		fmt.Println("Updated name:", fullName.Get()) // "Mr. Jane Doe"

		// This doesn't trigger recomputation even though we use title.Get()
		// because title isn't in the dependency list
		title.Set("Ms.")
		fmt.Println("After title change:", fullName.Get()) // Still "Mr. Jane Doe"

		// Force a re-read to see the actual current value
		// This would show "Ms. Jane Doe" if we manually get it again

		return nil
	})

	wait()
	cleanup()
}
