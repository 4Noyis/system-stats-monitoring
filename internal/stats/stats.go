package stats

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

func GetCpuUsage() {
	percent, _ := cpu.Percent(time.Second, false)

	fmt.Printf("Total CPU Usage: %.2f%%\n", percent[0])
}

func CoreCounts() {
	c, _ := cpu.Counts(true)
	fmt.Println(c) //8 performance (P) + 2 efficiency (E) cores
}

func VirtualMemory() {
	v, _ := mem.VirtualMemory()
	// almost every return value is a struct
	fmt.Printf("Total: %v, Free:%v, UsedPercent:%f%%\n", v.Total, v.Free, v.UsedPercent)

	// convert to JSON. String() is also implemented
	fmt.Println(v)
}
