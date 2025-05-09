package stats

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/v3/cpu"
)

func GetCpuUsage() {
	percent, _ := cpu.Percent(time.Second, false)

	fmt.Printf("Total CPU Usage: %.2f%%\n", percent[0])
}

func CoreCounts() error {
	coreCount, err := cpu.Counts(true)
	if err != nil {
		return err
	}

	fmt.Println(coreCount) //8 performance (P) + 2 efficiency (E) cores

	return nil
}

func GetSystemInfo() error {

	info, err := host.Info()
	if err != nil {
		return err
	}

	os := info.Platform
	osVersion := info.PlatformVersion

	kernel := info.KernelArch
	kernelVersion := info.KernelVersion

	uptime := time.Duration(info.Uptime) * time.Second
	uptime = uptime.Round(time.Second)

	fmt.Println("System Information:")
	fmt.Printf("  OS: %s %s\n", os, osVersion)
	fmt.Printf("  Kernel: %s %s\n", kernel, kernelVersion)
	fmt.Printf("  Uptime: %s\n", uptime)

	return nil
}
