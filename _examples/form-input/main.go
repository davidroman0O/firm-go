package main

import (
	"fmt"

	"github.com/davidroman0O/firm-go"
)

type FormData struct {
	Username   string
	Email      string
	Age        int
	Newsletter bool
}

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		// Individual form fields
		username := firm.Signal(owner, "")
		email := firm.Signal(owner, "")
		age := firm.Signal(owner, 0)
		newsletter := firm.Signal(owner, false)

		// Effect to detect form changes
		validationRuns := 0
		firm.Effect(owner, func() firm.CleanUp {
			validationRuns++

			// Get all values
			u := username.Get()
			e := email.Get()
			a := age.Get()
			n := newsletter.Get()

			fmt.Printf("Form validation #%d:\n", validationRuns)
			fmt.Printf("  Username: %s\n", u)
			fmt.Printf("  Email: %s\n", e)
			fmt.Printf("  Age: %d\n", a)
			fmt.Printf("  Newsletter: %v\n", n)

			// Validate form...

			return nil
		}, nil)

		// Simulate user filling out form field by field
		fmt.Println("User filling form (individual updates):")
		username.Set("johndoe")
		email.Set("john@example.com")
		age.Set(30)
		newsletter.Set(true)

		// Reset form
		fmt.Println("\nResetting form...")
		firm.Batch(owner, func() {
			username.Set("")
			email.Set("")
			age.Set(0)
			newsletter.Set(false)
		})

		// Simulate restoring form from saved data (all at once)
		fmt.Println("\nRestoring saved form (batched update):")
		savedData := FormData{
			Username:   "janedoe",
			Email:      "jane@example.com",
			Age:        28,
			Newsletter: true,
		}

		firm.Batch(owner, func() {
			username.Set(savedData.Username)
			email.Set(savedData.Email)
			age.Set(savedData.Age)
			newsletter.Set(savedData.Newsletter)
		})

		fmt.Printf("\nTotal validation runs: %d\n", validationRuns)

		return nil
	})

	wait()
	cleanup()
}
