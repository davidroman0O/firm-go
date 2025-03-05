package main

import (
	"fmt"

	"github.com/davidroman0O/firm-go"
)

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		// Create a theme context
		theme := firm.NewContext(owner, "light")

		// Component that uses the theme
		createComponent := func(owner *firm.Owner) {
			firm.Effect(owner, func() firm.CleanUp {
				currentTheme := theme.Use() // Get context value with tracking
				fmt.Println("Rendering with theme:", currentTheme)
				return nil
			}, nil)
		}

		// Create initial component
		createComponent(owner)

		// Change theme - will re-render component
		theme.Set("dark")

		return nil
	})

	wait()
	cleanup()
}
