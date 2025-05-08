package main

import (
	"fmt"
	"time"

	"github.com/4Noyis/system-stats-monitoring/internal/logger"
	"github.com/4Noyis/system-stats-monitoring/internal/stats"
)

func main() {

	stats.GetCpuUsage()

	stats.CoreCounts()

	err := logger.LogCPUStats(2*time.Second, 5, "json", "cpu_log.json")
	if err != nil {
		fmt.Println("Logging error:", err)
	}
}
