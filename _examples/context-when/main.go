package main

import (
	"fmt"

	"github.com/davidroman0O/firm-go"
)

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		// Create a user role context
		userRole := firm.NewContext(owner, "guest")

		// Create admin panel when user is admin or moderator
		adminPanelCleanup := userRole.When(
			owner,
			// Matcher function
			func(role string) bool {
				return role == "admin" || role == "moderator"
			},
			// Component initialization
			func(childOwner *firm.Owner) firm.CleanUp {
				fmt.Println("👑 Admin panel initialized")

				// Admin panel UI here...

				return func() {
					fmt.Println("👑 Admin panel cleaned up")
				}
			},
		)

		// Change roles to test condition
		fmt.Println("Setting role to moderator...")
		userRole.Set("moderator") // Admin panel initializes

		fmt.Println("Setting role to user...")
		userRole.Set("user") // Admin panel cleans up

		fmt.Println("Setting role to admin...")
		userRole.Set("admin") // Admin panel initializes again

		// Clean up the when handler
		adminPanelCleanup()

		return nil
	})

	wait()
	cleanup()
}
