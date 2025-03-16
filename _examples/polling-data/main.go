package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/davidroman0O/firm-go"
)

type StockPrice struct {
	Symbol string
	Price  float64
	Change float64
}

// Simulate API call for stock price
func fetchStockPrice(symbol string) StockPrice {
	// Simulate base price
	basePrice := 100.0
	if symbol == "AAPL" {
		basePrice = 150.0
	} else if symbol == "MSFT" {
		basePrice = 250.0
	}

	// Random fluctuation
	change := (rand.Float64() * 6) - 3 // Between -3 and +3

	return StockPrice{
		Symbol: symbol,
		Price:  basePrice + change,
		Change: change,
	}
}

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		// Stock symbol to track
		symbol := firm.Signal(owner, "AAPL")

		fmt.Println("Starting stock price monitor...")

		// Create polling for stock prices (every 2 seconds)
		stockPrices := firm.NewPolling(owner, func() StockPrice {
			// Use current symbol
			currentSymbol := symbol.Get()
			return fetchStockPrice(currentSymbol)
		}, 2*time.Second)

		// Display price updates
		firm.Effect(owner, func() firm.CleanUp {
			price := stockPrices.Get()
			changeSymbol := "="
			if price.Change > 0 {
				changeSymbol = "↑"
			} else if price.Change < 0 {
				changeSymbol = "↓"
			}

			fmt.Printf("%s: $%.2f %s (%.2f)\n",
				price.Symbol, price.Price, changeSymbol, price.Change)

			return nil
		}, []firm.Reactive{stockPrices})

		// Switch stock symbol after 6 seconds
		time.Sleep(6 * time.Second)
		fmt.Println("\nSwitching to MSFT...")
		symbol.Set("MSFT")

		// Watch for a few more updates
		time.Sleep(6 * time.Second)

		return nil
	})

	wait()
	cleanup()
}
