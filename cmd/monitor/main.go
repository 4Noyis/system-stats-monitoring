package main

import (
	"github.com/4Noyis/system-stats-monitoring/internal/stats"
)

func main() {

	stats.GetCpuUsage()

	stats.CoreCounts()

	stats.GetSystemInfo()

	/*
	   err := logger.LogCPUStats(2*time.Second, 5, "json", "cpu_log.json")

	   	if err != nil {
	   		fmt.Println("Logging error:", err)
	   	}
	*/
}
