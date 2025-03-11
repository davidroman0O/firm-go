package main

import (
	"fmt"

	"github.com/davidroman0O/firm-go"
)

type User struct {
	FirstName string
	LastName  string
	Age       int
}

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		// Parent signal
		user := firm.Signal(owner, User{
			FirstName: "John",
			LastName:  "Doe",
			Age:       30,
		})

		// Create derived signal for first name
		firstName := firm.DerivedSignal(
			owner,
			user,
			// Getter
			func(u User) string {
				return u.FirstName
			},
			// Setter
			func(u User, name string) User {
				u.FirstName = name
				return u
			},
		)

		fmt.Println("Initial name:", firstName.Get())

		// Update via derived signal
		firstName.Set("Jane")

		// Parent is automatically updated
		fmt.Println("Updated user:", user.Get())

		return nil
	})

	wait()
	cleanup()
}
