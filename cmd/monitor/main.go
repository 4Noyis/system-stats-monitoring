package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/4Noyis/system-stats-monitoring/internal/stats"
)

func main() {

	fmt.Println("Starting System Statistics Monitor...")

	// --- Static Information (runs once) ---
	err := stats.GetSystemInfo()
	if err != nil {
		fmt.Printf("Error getting system info: %v\n", err)
	}

	err = stats.GetCPUInfo() // This will print static CPU info and a CPU usage snapshot
	if err != nil {
		fmt.Printf("Error getting CPU info: %v\n", err)
	}

	err = stats.GetMemInfo() // This will print static Mem info and a Mem usage snapshot
	if err != nil {
		fmt.Printf("Error getting memory info: %v\n", err)
	}

	// --- Network Info (runs once, blocks for 5s during its measurement) ---
	// You can run this before or after starting monitors, or also in a goroutine if needed.
	// err = stats.GetDownloadInfo()
	// if err != nil {
	// 	fmt.Printf("Error getting download info: %v\n", err)
	// }

	// <------ Setup for continuous monitoring ----->
	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancel is called when main exits, to clean up goroutines

	// Define the update interval
	updateInterval := 5 * time.Second

	// Start CPU monitor in a new goroutine
	go stats.StartCPUMonitor(ctx, updateInterval)

	// Start Memory monitor in a new goroutine
	go stats.StartMemoryMonitor(ctx, updateInterval)

	fmt.Println("\nCPU and Memory usage will update every 5 seconds.")
	fmt.Println("Press Ctrl+C to stop.")

	// <----- Keep the main program running and wait for a shutdown signal ---> Gonna change later. Just form termianl testing!!!!
	sigChan := make(chan os.Signal, 1)
	// Notify sigChan on Interrupt (Ctrl+C) or Terminate signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	<-sigChan

	fmt.Println("\nShutdown signal received. Stopping monitors...")
	cancel() // Signal the goroutines to stop by canceling the context

	// Give goroutines a moment to print their "Stopping..." message and exit cleanly
	time.Sleep(1 * time.Second)
	fmt.Println("Exited.")

}
