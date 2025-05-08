package main

import (
	"fmt"

	"github.com/4Noyis/system-stats-monitoring/internal/stats"
)

func main() {
	cpu := stats.GetCpuUsage()
	fmt.Println(cpu)
}
