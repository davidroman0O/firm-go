package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/davidroman0O/firm-go"
)

// User represents a user in our system
type User struct {
	ID    int
	Name  string
	Email string
	Age   int
}

// Weather represents weather data
type Weather struct {
	Temperature float64
	Conditions  string
	Location    string
}

// Mock API functions
func fetchUserById(id int) (User, error) {
	// Simulate API delay
	time.Sleep(time.Millisecond * 200)

	if id <= 0 {
		return User{}, fmt.Errorf("invalid user id: %d", id)
	}

	// Return mock user
	return User{
		ID:    id,
		Name:  fmt.Sprintf("User %d", id),
		Email: fmt.Sprintf("user%d@example.com", id),
		Age:   25 + (id % 20), // Some variety in ages
	}, nil
}

func fetchWeather(location string) (Weather, error) {
	// Simulate API delay
	time.Sleep(time.Millisecond * 250)

	if location == "" {
		return Weather{}, fmt.Errorf("location cannot be empty")
	}

	// Return mock weather data with some randomness
	conditions := []string{"Sunny", "Cloudy", "Rainy", "Windy"}
	return Weather{
		Temperature: 20.0 + rand.Float64()*10.0, // Random temperature between 20-30
		Conditions:  conditions[rand.Intn(len(conditions))],
		Location:    location,
	}, nil
}

func main() {
	fmt.Println("🚀 Firm-Go Demo Application")

	disposeFirst, waitFirst := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		search := firm.Signal(owner, "")

		// Debounced search query - updates 300ms after the source
		debouncedSearch := firm.Defer(owner, search, 300)

		firm.Effect(owner, func() firm.CleanUp {
			// Only runs when the debounced value changes
			query := debouncedSearch.Get()
			if query != "" {
				fmt.Println("Searching for:", query)
				// performSearch(query)
			}
			return nil
		}, nil)

		// These rapid updates only result in one search
		search.Set("a")
		search.Set("ap")
		search.Set("app")
		search.Set("appl")
		search.Set("apple")

		return func() {
			fmt.Println("🧹 Root Cleanup executed")
		}
	})

	waitFirst()

	disposeFirst()

	fmt.Println("===========================")

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Create root reactive system
	cleanup, waiting := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		fmt.Println("\n--- 1. Signals Demonstration ---")

		// Create basic signals
		count := firm.Signal(owner, 0)
		message := firm.Signal(owner, "Hello, Firm-Go!")

		// Effect that depends on both signals
		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("Count: %d, Message: '%s'\n", count.Get(), message.Get())
			return nil
		}, nil)

		// Update signals to trigger the effect
		count.Set(1)
		message.Set("Signals are reactive!")

		// Update with functional update
		count.Update(func(v int) int {
			return v + 1
		})

		fmt.Println("\n--- 2. Computed/Memo Demonstration ---")

		// Create signals for computation
		price := firm.Signal(owner, 10.0)
		quantity := firm.Signal(owner, 2)

		// Memo that computes total price
		total := firm.Memo(owner, func() float64 {
			return price.Get() * float64(quantity.Get())
		}, nil)

		// Memo with multiple dependencies
		summary := firm.Memo(owner, func() string {
			return fmt.Sprintf("%d items at $%.2f each = $%.2f total",
				quantity.Get(), price.Get(), total.Get())
		}, nil)

		firm.Effect(owner, func() firm.CleanUp {
			fmt.Println("Order:", summary.Get())
			return nil
		}, nil)

		// Update dependencies to trigger recomputation
		price.Set(12.5)
		quantity.Set(4)

		fmt.Println("\n--- 3. Context Demonstration ---")

		// Create a context with default value
		themeContext := firm.NewContext(owner, "light")

		// Effect that depends on the context
		firm.Effect(owner, func() firm.CleanUp {
			theme := themeContext.Use()
			fmt.Println("Current theme:", theme)
			return nil
		}, nil)

		// Update context value
		themeContext.Set("dark")

		// Conditional rendering with Match
		matchCleanup := themeContext.Match(owner, "dark", func(childOwner *firm.Owner) firm.CleanUp {
			fmt.Println("🌙 Dark theme components initialized")
			return func() {
				fmt.Println("🌙 Dark theme components cleaned up")
			}
		})

		// When condition
		whenCleanup := themeContext.When(owner,
			func(theme string) bool {
				return theme == "dark" || theme == "system"
			},
			func(childOwner *firm.Owner) firm.CleanUp {
				fmt.Println("🎨 Theme is dark or system")
				return nil
			},
		)

		// Switch theme to trigger conditional cleanup
		themeContext.Set("light")

		fmt.Println("\n--- 4. Resource Demonstration ---")

		// Signal for current user ID
		userId := firm.Signal(owner, 1)

		// Resource for user data
		userResource := firm.Resource(owner, func() (User, error) {
			fmt.Println("Fetching user data for ID:", userId.Get())
			return fetchUserById(userId.Get())
		})

		// Effect to display user data
		firm.Effect(owner, func() firm.CleanUp {
			if userResource.Loading() {
				fmt.Println("👤 Loading user...")
			} else if err := userResource.Error(); err != nil {
				fmt.Println("❌ Error loading user:", err)
			} else {
				user := userResource.Data()
				fmt.Printf("👤 User: %s (%s), Age: %d\n", user.Name, user.Email, user.Age)
			}
			return nil
		}, []firm.Reactive{userResource})

		// Trigger a refetch with a different user
		time.Sleep(500 * time.Millisecond) // Wait for first fetch to complete
		userId.Set(2)

		fmt.Println("\n--- 5. Batching Demonstration ---")

		// Signals to track
		firstName := firm.Signal(owner, "John")
		lastName := firm.Signal(owner, "Doe")
		age := firm.Signal(owner, 30)

		// Computed full name
		fullName := firm.Memo(owner, func() string {
			return fmt.Sprintf("%s %s (age %d)", firstName.Get(), lastName.Get(), age.Get())
		}, nil)

		// Track effect runs
		updateCount := firm.Signal(owner, 0)

		firm.Effect(owner, func() firm.CleanUp {
			updateCount.Update(func(v int) int { return v + 1 })
			fmt.Printf("Person update #%d: %s\n", updateCount.Get(), fullName.Get())
			return nil
		}, []firm.Reactive{fullName})

		// Individual updates (causes multiple effect runs)
		firstName.Set("Jane")
		lastName.Set("Smith")
		age.Set(28)

		// Batched updates (causes single effect run)
		fmt.Println("\nNow updating in batch:")
		firm.Batch(owner, func() {
			firstName.Set("Bob")
			lastName.Set("Johnson")
			age.Set(42)
		})

		fmt.Println("\n--- 6. Untrack Demonstration ---")

		config := firm.Signal(owner, "default-config")

		firm.Effect(owner, func() firm.CleanUp {
			// This creates a dependency on count
			currentCount := count.Get()

			// This reads config without creating a dependency
			configValue := firm.Untrack(owner, func() string {
				return config.Get()
			})

			fmt.Printf("Count (tracked): %d, Config (untracked): %s\n", currentCount, configValue)
			return nil
		}, nil)

		// Update count (should trigger the effect)
		count.Set(10)

		// Update config (should NOT trigger the effect)
		config.Set("new-config")
		count.Set(11) // Proving the effect still works

		fmt.Println("\n--- 7. Derived Signal Demonstration ---")

		// Create a user signal
		user := firm.Signal(owner, User{
			ID:    1,
			Name:  "John Smith",
			Email: "john@example.com",
			Age:   30,
		})

		// Create a derived signal for the user's name
		userName := firm.DerivedSignal(
			owner,
			user,
			// Getter
			func(u User) string {
				return u.Name
			},
			// Setter
			func(u User, name string) User {
				u.Name = name
				return u
			},
		)

		// Effect to track name changes
		firm.Effect(owner, func() firm.CleanUp {
			fmt.Printf("User name (derived): %s\n", userName.Get())
			return nil
		}, []firm.Reactive{userName})

		// Update via derived signal
		userName.Set("John Doe")

		// Show that original signal was updated
		fmt.Printf("User object after derived update: %+v\n", user.Get())

		fmt.Println("\n--- 8. Polling Demonstration ---")

		// Create a polling signal for weather updates
		weatherLocation := firm.Signal(owner, "New York")

		weatherPolling := firm.NewPolling(owner, func() Weather {
			location := weatherLocation.Get()
			weather, err := fetchWeather(location)
			if err != nil {
				fmt.Println("Error fetching weather:", err)
				return Weather{Temperature: 0, Conditions: "Error", Location: location}
			}
			return weather
		}, 2*time.Second)

		// Track weather updates
		firm.Effect(owner, func() firm.CleanUp {
			weather := weatherPolling.Get()
			fmt.Printf("🌡️ Weather in %s: %.1f°C, %s\n",
				weather.Location, weather.Temperature, weather.Conditions)
			return nil
		}, []firm.Reactive{weatherPolling})

		// Let it run for a few polling cycles
		fmt.Println("\nWatching weather updates for 5 seconds...")
		time.Sleep(5 * time.Second)

		// Change location
		fmt.Println("\nChanging location to London...")
		weatherLocation.Set("London")

		// Wait for a couple more updates
		time.Sleep(4 * time.Second)

		// Pause polling
		fmt.Println("\nPausing weather updates...")
		weatherPolling.Pause()
		time.Sleep(3 * time.Second)

		// Resume with different interval
		fmt.Println("\nResuming with faster update interval...")
		weatherPolling.SetInterval(1 * time.Second)
		weatherPolling.Resume()
		time.Sleep(3 * time.Second)

		fmt.Println("\n✅ All demonstrations completed!")

		// Return root cleanup function
		return func() {
			// Clean up our matched context components
			matchCleanup()
			whenCleanup()
			fmt.Println("🧹 Root cleanup executed")
		}
	})

	waiting()

	// Finally, clean up the root
	cleanup()
	fmt.Println("🏁 Application terminated")
}
