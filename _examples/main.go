package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidroman0O/firm-go"
)

// Define some example types
type User struct {
	ID   int
	Name string
}

type Todo struct {
	ID        int
	Text      string
	Completed bool
	UserID    int
}

func main() {
	// Create a reactive system
	dispose := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		fmt.Println("=== Firm-Go Example ===")

		// Basic signal example
		count := firm.Signal(owner, 0)
		message := firm.Signal(owner, "Hello")

		// Effect that runs when count changes
		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("Count changed to: %d\n", count.Get())
			return func() {
				fmt.Println("Count effect cleanup")
			}
		}, []firm.Reactive{count})

		// Computed value (memo) based on count
		doubled := firm.Memo(owner, func() int {
			return count.Get() * 2
		}, []firm.Reactive{count})

		// Effect for the computed value
		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("Doubled value: %d\n", doubled.Get())
			return nil
		}, []firm.Reactive{doubled})

		// Context example - now much more useful!
		themeContext := firm.NewContext(owner, "light")

		// Effect that uses the context
		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("Current theme: %s\n", themeContext.Use())
			return nil
		}, []firm.Reactive{themeContext})

		// Setup theme-specific code blocks
		themeContext.Match(owner, "dark", func(darkOwner *firm.Owner) firm.CleanUp {
			// This code only runs when theme is "dark"
			firm.Effect(darkOwner, func() firm.CleanUp {
				fmt.Println("Dark mode activated!")
				return nil
			}, []firm.Reactive{})

			return func() {
				fmt.Println("Dark mode deactivated!")
			}
		})

		themeContext.When(owner, func(theme string) bool {
			// This code only runs when theme contains "light"
			return theme == "light" || theme == "light-high-contrast"
		}, func(lightOwner *firm.Owner) firm.CleanUp {
			firm.Effect(lightOwner, func() firm.CleanUp {
				fmt.Println("Light mode active")
				return nil
			}, []firm.Reactive{})

			// Can use parent signals in this scope
			firm.Effect(lightOwner, func() firm.CleanUp {
				fmt.Printf("Light theme with count: %d\n", count.Get())
				return nil
			}, []firm.Reactive{count})

			return func() {
				fmt.Println("Light mode deactivated!")
				count.Set(count.Get() + 1)
			}
		})

		// Async resource example
		userResource := firm.Resource(owner, func() (User, error) {
			// Simulate API call
			time.Sleep(100 * time.Millisecond)
			return User{ID: 1, Name: "John Doe"}, nil
		})

		// Effect that reacts to resource state
		firm.Effect(owner, func() firm.CleanUp {
			if userResource.Loading() {
				fmt.Println("Loading user...")
			} else if err := userResource.Error(); err != nil {
				fmt.Printf("Error loading user: %v\n", err)
			} else {
				user := userResource.Data()
				fmt.Printf("User loaded: %s (ID: %d)\n", user.Name, user.ID)
			}
			return nil
		}, []firm.Reactive{userResource})

		// Use a simple signal for todos
		todos := firm.Signal(owner, []Todo{
			{ID: 1, Text: "Learn Go", Completed: false, UserID: 1},
			{ID: 2, Text: "Learn Firm-Go", Completed: false, UserID: 1},
		})

		// Effect to display todos
		firm.Effect(owner, func() firm.CleanUp {
			todoList := todos.Get()
			fmt.Println("Current todos:")
			for _, todo := range todoList {
				status := "[ ]"
				if todo.Completed {
					status = "[x]"
				}
				fmt.Printf("  %s %s (ID: %d)\n", status, todo.Text, todo.ID)
			}
			return nil
		}, []firm.Reactive{todos})

		// Computed for completed todos count
		completedCount := firm.Memo(owner, func() int {
			todoList := todos.Get()
			count := 0
			for _, todo := range todoList {
				if todo.Completed {
					count++
				}
			}
			return count
		}, []firm.Reactive{todos})

		// Effect for completed count
		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("Completed todos: %d/%d\n", completedCount.Get(), len(todos.Get()))
			return nil
		}, []firm.Reactive{completedCount, todos})

		// Deferred signal that updates after a delay
		delayedMessage := firm.Defer(owner, message, 500) // 500ms delay

		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("Delayed message: %s\n", delayedMessage.Get())
			return nil
		}, []firm.Reactive{delayedMessage})

		// Map example
		todoTexts := firm.Memo(owner, func() []string {
			todoList := todos.Get()
			texts := make([]string, len(todoList))
			for i, todo := range todoList {
				texts[i] = todo.Text
			}
			return texts
		}, []firm.Reactive{todos})

		todoLabels := firm.Map(
			owner,
			todoTexts,
			func(text string, index int) string {
				return fmt.Sprintf("%d: %s", index+1, text)
			},
		)

		firm.Effect(owner, func() firm.CleanUp {
			fmt.Println("Todo labels:")
			for _, label := range todoLabels.Get() {
				fmt.Printf("  %s\n", label)
			}
			return nil
		}, []firm.Reactive{todoLabels})

		// Derived signal example
		userName := firm.Signal(owner, "john.doe")
		userDisplayName := firm.DerivedSignal(
			owner,
			userName,
			func(name string) string { return "User: " + name },
			func(original string, display string) string {
				// This updates the original when display changes
				if len(display) > 6 {
					return display[6:] // Remove "User: " prefix
				}
				return original
			},
		)

		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("User display name: %s\n", userDisplayName.Get())
			return nil
		}, []firm.Reactive{userDisplayName})

		// Computed example - a value that can be manually recomputed
		randomNumber := firm.NewComputed(owner, func() int {
			return rand.Intn(100) // Generate random number between 0-99
		})

		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("Random number: %d\n", randomNumber.Get())
			return nil
		}, []firm.Reactive{randomNumber})

		// Computed example with file system data
		currentDir := firm.NewComputed(owner, func() []string {
			// Get current directory contents
			dir, err := os.Getwd()
			if err != nil {
				return []string{"Error getting directory"}
			}

			entries, err := os.ReadDir(dir)
			if err != nil {
				return []string{"Error reading directory"}
			}

			result := make([]string, 0, len(entries))
			for _, entry := range entries {
				// Only include non-hidden files and directories
				if !strings.HasPrefix(entry.Name(), ".") {
					if entry.IsDir() {
						result = append(result, "📁 "+entry.Name())
					} else {
						result = append(result, "📄 "+entry.Name())
					}
				}
			}

			return result
		})

		firm.Effect(owner, func() firm.CleanUp {
			files := currentDir.Get()
			if len(files) > 0 {
				fmt.Println("\nDirectory contents:")
				for i, file := range files {
					if i < 5 { // Show only first 5 items
						fmt.Printf("  %s\n", file)
					} else {
						fmt.Println("  ...")
						break
					}
				}
				fmt.Printf("  (%d items total)\n", len(files))
			}
			return nil
		}, []firm.Reactive{currentDir})

		// Polling example - automatically updates current time
		currentTime := firm.NewPolling(owner, func() string {
			return time.Now().Format("15:04:05")
		}, 1*time.Second) // Update every second

		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("Current time: %s\n", currentTime.Get())
			return nil
		}, []firm.Reactive{currentTime})

		// Polling example that watches a specific file
		tmpFilePath := filepath.Join(os.TempDir(), "firm-demo.txt")

		// Create the initial file
		if f, err := os.Create(tmpFilePath); err == nil {
			f.WriteString("Initial content")
			f.Close()
		}

		fileContent := firm.NewPolling(owner, func() string {
			content, err := os.ReadFile(tmpFilePath)
			if err != nil {
				return "Error reading file"
			}
			return string(content)
		}, 500*time.Millisecond) // Check every 500ms

		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("File content: %s\n", fileContent.Get())
			return func() {
				// Clean up temp file on exit
				os.Remove(tmpFilePath)
			}
		}, []firm.Reactive{fileContent})

		// Now let's make some changes to demonstrate reactivity
		fmt.Println("\n=== Making changes ===")

		// Batch multiple updates
		firm.Batch(owner, func() {
			// Update count which triggers effects and memos
			count.Set(5)

			// Mark a todo as completed - direct mutation with Get and Set
			todoList := todos.Get()
			todoList[0].Completed = true // Mark first todo as completed
			todos.Set(todoList)          // Set the entire list back to trigger reactivity

			// Add a new todo
			todoList = todos.Get() // Get latest version
			todoList = append(todoList, Todo{ID: 3, Text: "Master Firm-Go", Completed: false, UserID: 1})
			todos.Set(todoList) // Set the entire list back

			// Update message which will be reflected in delayed message after 500ms
			message.Set("Updated message")

			// Update user name which updates the derived display name
			userName.Set("jane.doe")
		})

		// Update the temp file
		go func() {
			time.Sleep(300 * time.Millisecond)
			if f, err := os.OpenFile(tmpFilePath, os.O_WRONLY|os.O_TRUNC, 0644); err == nil {
				f.WriteString("Updated content")
				f.Close()
				fmt.Println("\n=== Updated file content ===")
			}
		}()

		// Force recompute the random number
		go func() {
			time.Sleep(400 * time.Millisecond)
			fmt.Println("\n=== Recomputing random number ===")
			randomNumber.Recompute()
		}()

		// Wait a bit then change the theme to demonstrate conditional contexts
		time.Sleep(600 * time.Millisecond)
		fmt.Println("\n=== Changing theme ===")
		themeContext.Set("dark")

		// Change it back after a delay
		time.Sleep(600 * time.Millisecond)
		fmt.Println("\n=== Changing theme back ===")
		themeContext.Set("light")

		// Pause the time polling temporarily
		go func() {
			time.Sleep(200 * time.Millisecond)
			fmt.Println("\n=== Pausing time updates ===")
			currentTime.Pause()

			time.Sleep(2 * time.Second)
			fmt.Println("\n=== Resuming time updates ===")
			currentTime.Resume()
		}()

		// Recompute directory contents after some files might have changed
		go func() {
			time.Sleep(1200 * time.Millisecond)
			fmt.Println("\n=== Refreshing directory contents ===")
			currentDir.Recompute()
		}()

		// Set up a cleanup function
		return func() {
			fmt.Println("\n=== Cleaning up ===")
		}
	})

	// Wait a bit to see the delayed effects
	time.Sleep(4 * time.Second)

	// Dispose when done
	dispose()
}
