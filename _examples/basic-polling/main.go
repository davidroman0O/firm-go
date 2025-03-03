package main

import (
	"fmt"
	"time"

	"github.com/davidroman0O/firm-go"
)

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		fmt.Println("Starting time polling example")

		// Create a polling signal for current time (updates every second)
		timePolling := firm.NewPolling(owner, func() time.Time {
			return time.Now()
		}, time.Second)

		// Track the time updates
		firm.Effect(owner, func() firm.CleanUp {
			currentTime := timePolling.Get()
			fmt.Printf("Current time: %s\n", currentTime.Format("15:04:05"))
			return nil
		}, []firm.Reactive{timePolling})

		// Let it run for a few seconds
		fmt.Println("Watching time for 3 seconds...")
		time.Sleep(3 * time.Second)

		// Pause polling
		fmt.Println("\nPausing time updates...")
		timePolling.Pause()
		time.Sleep(2 * time.Second)

		// Resume with a different interval
		fmt.Println("\nResuming with faster updates (500ms)...")
		timePolling.SetInterval(500 * time.Millisecond)
		timePolling.Resume()
		time.Sleep(2 * time.Second)

		return nil
	})

	wait()
	cleanup()
}
