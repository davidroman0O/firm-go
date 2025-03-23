package main

import (
	"fmt"

	"github.com/davidroman0O/firm-go"
)

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		// Create a theme context
		theme := firm.NewContext(owner, "light")

		// Only render dark mode components when theme is "dark"
		darkModeCleanup := theme.Match(owner, "dark", func(childOwner *firm.Owner) firm.CleanUp {
			fmt.Println("🌙 Dark mode features initialized")

			// Create dark-theme-specific UI here...

			return func() {
				fmt.Println("🌙 Dark mode features cleaned up")
			}
		})

		// Change theme to trigger the match
		fmt.Println("Switching to dark theme...")
		theme.Set("dark")

		fmt.Println("Switching to light theme...")
		theme.Set("light")

		// Clean up the match handler
		fmt.Println("Cleaning up match handler...")
		darkModeCleanup()

		return nil
	})

	wait()
	cleanup()
}
