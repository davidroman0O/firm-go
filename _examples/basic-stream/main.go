package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/davidroman0O/firm-go"
)

// PingData represents structured ping information
type PingData struct {
	Target        string
	IP            string
	Responses     []PingResponse
	PacketsSent   int
	PacketsRecv   int
	PacketLoss    float64
	MinLatency    float64
	AvgLatency    float64
	MaxLatency    float64
	StdDevLatency float64
	Complete      bool
}

// PingResponse represents a single ping response
type PingResponse struct {
	Sequence int
	TTL      int
	Time     float64 // ms
}

func main() {
	cleanup, wait := firm.Root(func(owner *firm.Owner) firm.CleanUp {
		// Create a structured ping data signal
		pingData := firm.StreamSignal(owner, PingData{Target: "example.com"}, func(set func(PingData), done func()) {
			// Initialize data structure
			data := PingData{Target: "example.com"}

			// Start ping command
			cmd := exec.Command("ping", "-c", "5", data.Target)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				fmt.Println("Error:", err)
				done()
				return
			}

			if err := cmd.Start(); err != nil {
				fmt.Println("Error starting command:", err)
				done()
				return
			}

			// Regex patterns for parsing
			ipPattern := regexp.MustCompile(`PING\s+\S+\s+\(([0-9.]+)\)`)
			responsePattern := regexp.MustCompile(`icmp_seq=(\d+)\s+ttl=(\d+)\s+time=(\d+\.\d+)`)
			statsPattern := regexp.MustCompile(`(\d+) packets transmitted, (\d+) packets received, ([0-9.]+)% packet loss`)
			latencyPattern := regexp.MustCompile(`min/avg/max/stddev = ([0-9.]+)/([0-9.]+)/([0-9.]+)/([0-9.]+)`)

			// Read output line by line
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				line := scanner.Text()
				fmt.Println("Raw:", line) // Show raw output for debugging

				// Extract IP
				if matches := ipPattern.FindStringSubmatch(line); len(matches) > 1 {
					data.IP = matches[1]
					set(data) // Update with IP
				}

				// Extract ping response
				if matches := responsePattern.FindStringSubmatch(line); len(matches) > 3 {
					seq, _ := strconv.Atoi(matches[1])
					ttl, _ := strconv.Atoi(matches[2])
					timeMs, _ := strconv.ParseFloat(matches[3], 64)

					response := PingResponse{
						Sequence: seq,
						TTL:      ttl,
						Time:     timeMs,
					}

					data.Responses = append(data.Responses, response)
					set(data) // Update with new response
				}

				// Extract packet stats
				if matches := statsPattern.FindStringSubmatch(line); len(matches) > 3 {
					data.PacketsSent, _ = strconv.Atoi(matches[1])
					data.PacketsRecv, _ = strconv.Atoi(matches[2])
					data.PacketLoss, _ = strconv.ParseFloat(matches[3], 64)
					set(data) // Update with stats
				}

				// Extract latency stats
				if matches := latencyPattern.FindStringSubmatch(line); len(matches) > 4 {
					data.MinLatency, _ = strconv.ParseFloat(matches[1], 64)
					data.AvgLatency, _ = strconv.ParseFloat(matches[2], 64)
					data.MaxLatency, _ = strconv.ParseFloat(matches[3], 64)
					data.StdDevLatency, _ = strconv.ParseFloat(matches[4], 64)
					data.Complete = true
					set(data) // Update with latency stats
				}
			}

			if err := cmd.Wait(); err != nil {
				fmt.Println("Command finished with error:", err)
			}

			// Final update with complete flag
			data.Complete = true
			set(data)
			done()
		})

		// Current latency signal derived from ping data
		currentLatency := firm.Memo(owner, func() float64 {
			data := pingData.Get()
			if len(data.Responses) == 0 {
				return 0
			}
			return data.Responses[len(data.Responses)-1].Time
		}, []firm.Reactive{pingData})

		// Track status changes
		firm.Effect(owner, func() firm.CleanUp {
			data := pingData.Get()

			if data.IP != "" && len(data.Responses) == 0 {
				fmt.Printf("Pinging %s (%s)...\n", data.Target, data.IP)
			}

			return nil
		}, []firm.Reactive{pingData})

		// Track individual responses
		firm.Effect(owner, func() firm.CleanUp {
			latency := currentLatency.Get()
			if latency > 0 {
				data := pingData.Get()
				latestResponse := data.Responses[len(data.Responses)-1]

				// Color code based on latency
				var indicator string
				switch {
				case latency < 50:
					indicator = "🟢" // Green - excellent
				case latency < 100:
					indicator = "🟡" // Yellow - good
				case latency < 200:
					indicator = "🟠" // Orange - fair
				default:
					indicator = "🔴" // Red - poor
				}

				fmt.Printf("Response %d: %s %.2f ms (ttl=%d)\n",
					latestResponse.Sequence, indicator, latestResponse.Time, latestResponse.TTL)
			}
			return nil
		}, []firm.Reactive{currentLatency})

		// Display summary when complete
		firm.Effect(owner, func() firm.CleanUp {
			data := pingData.Get()

			if data.Complete {
				fmt.Printf("\n--- %s ping summary ---\n", data.Target)
				fmt.Printf("IP address: %s\n", data.IP)
				fmt.Printf("Packets: %d sent, %d received, %.1f%% loss\n",
					data.PacketsSent, data.PacketsRecv, data.PacketLoss)
				fmt.Printf("RTT (ms): min=%.3f, avg=%.3f, max=%.3f, stddev=%.3f\n",
					data.MinLatency, data.AvgLatency, data.MaxLatency, data.StdDevLatency)

				// Calculate reliability rating
				var reliabilityRating string
				if data.PacketLoss == 0 && data.AvgLatency < 100 {
					reliabilityRating = "Excellent"
				} else if data.PacketLoss < 5 && data.AvgLatency < 150 {
					reliabilityRating = "Good"
				} else if data.PacketLoss < 10 && data.AvgLatency < 200 {
					reliabilityRating = "Fair"
				} else {
					reliabilityRating = "Poor"
				}

				fmt.Printf("Connection quality: %s\n", reliabilityRating)
			}

			return nil
		}, []firm.Reactive{pingData})

		return nil
	})

	wait()
	cleanup()
}
