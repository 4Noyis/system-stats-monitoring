package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	appLogger "github.com/4Noyis/system-stats-monitoring/internal/logger"
	clientStats "github.com/4Noyis/system-stats-monitoring/internal/stats"
	"github.com/4Noyis/system-stats-monitoring/pkg/exporter"
	"github.com/shirou/gopsutil/v3/net"
)

type AllHostStats struct {
	CollectedAt time.Time                   `json:"collected_at"`
	System      clientStats.SystemInfoData  `json:"system_info"`
	CPU         clientStats.CPUInfoData     `json:"cpu_info"`
	Memory      clientStats.MemInfoData     `json:"memory_info"`
	Network     clientStats.NetworkData     `json:"network_info"`
	Processes   []clientStats.ProcessData   `json:"processes,omitempty"`
	Disks       []clientStats.DiskUsageData `json:"disk_usage,omitempty"`
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
	previousNetCounters, err = clientStats.GetCurrentIOCounters()
	if err != nil {
		appLogger.Fatal("Error getting initial network counters: %v. Exiting.", err)
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
		appLogger.Info("Shutdown signal received (%s), cancelling context.", sig)
		cancel() // signal all goroutines to stop
	}()

	ticker := time.NewTicker(collectionInterval)
	defer ticker.Stop()

	appLogger.Info("Collecting and sending stats to %s every %s.", serverURL, collectionInterval)

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
			appLogger.Info("Collector stopped due to context cancellation.")
			// Allow a brief moment for any final logging or cleanup if necessary
			time.Sleep(200 * time.Millisecond)
			fmt.Println("Client exited.")
			return
		}
	}
}

func collectAndSendStats(ctx context.Context) {
	appLogger.Info("Collecting stats...")

	var hostStats AllHostStats
	hostStats.CollectedAt = time.Now().UTC()

	var err error
	hostStats.System, err = clientStats.GetSystemInfo()
	if err != nil {
		appLogger.Error("Error getting system info: %v", err)
	}

	hostStats.CPU, err = clientStats.GetCPUInfo()
	if err != nil {
		appLogger.Error("Error getting CPU info: %v", err)
	}

	hostStats.Memory, err = clientStats.GetMemInfo()
	if err != nil {
		appLogger.Error("Error getting memory info: %v", err)
	}

	// Network
	currentNetCounters, err := clientStats.GetCurrentIOCounters()
	if err != nil {
		appLogger.Error("Error getting current network counters: %v", err)
	} else {
		currentTime := time.Now()
		if networkStatsInitialized {
			duration := currentTime.Sub(previousNetCollectionTime)
			hostStats.Network, err = clientStats.CalculateNetworkRates(currentNetCounters, previousNetCounters, duration)
			if err != nil {

				appLogger.Error("Error calculating network rates: %v", err)
				// Set to a default or empty struct if calculation fails
				hostStats.Network = clientStats.NetworkData{InterfaceName: "all"}

			}

		}
		// Update for next iteration
		previousNetCounters = currentNetCounters
		previousNetCollectionTime = currentTime
	}

	// process List
	hostStats.Processes, err = clientStats.GetProcessList(maxProcessesUsagePercent)
	if err != nil {
		appLogger.Error("Error getting process list: %v", err)
	}

	// disk
	hostStats.Disks, err = clientStats.GetDiskUsageInfo()
	if err != nil {
		appLogger.Error("Error getting disk usage %v", err)
	}

	// <-------- SEND THE DATA -------->
	err = exporter.SendStatsJSON(ctx, serverURL, hostStats) // Pass the populated hostStats struct
	if err != nil {

		appLogger.Error("Failed to send stats: %v", err)
	} else {
		appLogger.Info("Stats dispatch initiated successfully by exporter.")
		fmt.Println("-----------------------------------------------------")
	}

}
