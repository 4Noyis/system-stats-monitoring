package main

import (
	"github.com/4Noyis/system-stats-monitoring/internal/stats"
)

func main() {

	// stats.GetCpuUsage()

	stats.GetSystemInfo()

	stats.GetCPUInfo()

	stats.GetMemInfo()

	stats.GetDownloadInfo()
	/*
	   err := logger.LogCPUStats(2*time.Second, 5, "json", "cpu_log.json")

	   	if err != nil {
	   		fmt.Println("Logging error:", err)
	   	}
	*/
}
