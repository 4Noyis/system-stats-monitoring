package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/4Noyis/system-stats-monitoring/internal/stats"
	"github.com/4Noyis/system-stats-monitoring/pkg/exporter"
	"github.com/shirou/gopsutil/v3/net"
)

type AllHostStats struct {
	CollectedAt time.Time             `json:"collected_at"`
	System      stats.SystemInfoData  `json:"system_info"`
	CPU         stats.CPUInfoData     `json:"cpu_info"`
	Memory      stats.MemInfoData     `json:"memory_info"`
	Network     stats.NetworkData     `json:"network_info"`
	Processes   []stats.ProcessData   `json:"processes,omitempty"`
	Disks       []stats.DiskUsageData `json:"disk_usage,omitempty"`
}

var (
	previousNetCounters       net.IOCountersStat
	previousNetCollectionTime time.Time
	networkStatsInitialized   bool
)

const (
	serverURL                = "http://localhost:8080/api/stats" // Replace with your actual server URL
	collectionInterval       = 5 * time.Second
	maxProcessesUsagePercent = 10.0 // Limit the usage percent for procesess memory & CPU
)

func main() {
	fmt.Printf("Starting System Statistics Monitor Client (PID: %d)...\n", os.Getpid())

	// Initialize network stats baseline
	var err error
	previousNetCounters, err = stats.GetCurrentIOCounters()
	if err != nil {
		log.Fatalf("Error getting initial network counters: %v. Exiting.", err)
	}
	previousNetCollectionTime = time.Now()
	networkStatsInitialized = true

	// ---- Setup for periodic collection and sending -----
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %s. Shutting down...\n", sig)
		cancel() // signal all goroutines to stop
	}()

	ticker := time.NewTicker(collectionInterval)
	defer ticker.Stop()

	fmt.Printf("Collecting and sending stats to %s every %s.\n", serverURL, collectionInterval)

	fmt.Println("Press Ctrl+C to stop.")

	// Initial collection and send, then tick
	collectAndSendStats(ctx)

	for {
		select {
		case <-ticker.C:
			if ctx.Err() == nil { // Only collect if context is not already cancelled
				collectAndSendStats(ctx)
			}
		case <-ctx.Done():
			log.Println("Collector stopped due to context cancellation.")
			// Allow a brief moment for any final logging or cleanup if necessary
			time.Sleep(200 * time.Millisecond)
			fmt.Println("Client exited.")
			return
		}
	}
}

func collectAndSendStats(ctx context.Context) {
	log.Println("Collecting stats...")

	var hostStats AllHostStats
	hostStats.CollectedAt = time.Now().UTC()

	var err error
	hostStats.System, err = stats.GetSystemInfo()
	if err != nil {
		log.Printf("Error getting system info: %v", err)
	}

	hostStats.CPU, err = stats.GetCPUInfo()
	if err != nil {
		log.Printf("Error getting CPU info: %v", err)
	}

	hostStats.Memory, err = stats.GetMemInfo()
	if err != nil {
		log.Printf("Error getting memory info: %v", err)
	}

	// Network
	currentNetCounters, err := stats.GetCurrentIOCounters()
	if err != nil {
		log.Printf("Error getting current network counters: %v", err)
	} else {
		currentTime := time.Now()
		if networkStatsInitialized {
			duration := currentTime.Sub(previousNetCollectionTime)
			hostStats.Network, err = stats.CalculateNetworkRates(currentNetCounters, previousNetCounters, duration)
			if err != nil {

				log.Printf("Error calculating network rates: %v", err)
				// Set to a default or empty struct if calculation fails
				hostStats.Network = stats.NetworkData{InterfaceName: "all"}

			}

		}
		// Update for next iteration
		previousNetCounters = currentNetCounters
		previousNetCollectionTime = currentTime
	}

	// process List
	hostStats.Processes, err = stats.GetProcessList(maxProcessesUsagePercent)
	if err != nil {
		log.Printf("Error getting process list %v", err)
	}

	// disk
	hostStats.Disks, err = stats.GetDiskUsageInfo()
	if err != nil {
		log.Printf("Error getting disk usage info: %v", err)
	}

	// <-------- SEND THE DATA -------->
	err = exporter.SendStatsJSON(ctx, serverURL, hostStats) // Pass the populated hostStats struct
	if err != nil {

		log.Printf("Failed to send stats: %v", err)
	} else {
		log.Println("Stats dispatch initiated successfully.")
		fmt.Println("-----------------------------------------------------")
	}

}
