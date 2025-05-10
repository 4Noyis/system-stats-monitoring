package stats

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/net"
)

// Converts bytes to gigabytes
func BytesToGB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024)
}

// Converts bytes to megabytes
func BytesToMB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024)
}

/* <---------------- SYSTEM INFO -----------------> */
func GetSystemInfo() error {

	info, err := host.Info()
	if err != nil {
		return err
	}
	// info.HostID
	hostname := info.Hostname
	hostID := info.HostID
	// os := info.Platform // Geri açılcak
	os := "Sequoia" // silincek
	osVersion := info.PlatformVersion
	kernel := info.KernelArch
	kernelVersion := info.KernelVersion

	uptime := time.Duration(info.Uptime) * time.Second
	uptime = uptime.Round(time.Second)

	fmt.Println("System Information:")
	fmt.Printf("  Hostname: %s\n", hostname)
	fmt.Printf("  HostID: %s\n", hostID)
	fmt.Printf("  OS: %s %s\n", os, osVersion)
	fmt.Printf("  Kernel: %s %s\n", kernel, kernelVersion)
	fmt.Printf("  Uptime: %s\n\n", uptime)

	return nil
}

/* <---------------- CPU INFO -----------------> */

func GetCpuUsage() error {
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return err
	}

	fmt.Printf("  Total CPU Usage: %.2f%%\n\n", percent[0])

	return nil
}

func GetCPUInfo() error {
	cpuInfo, err := cpu.Info()
	if err != nil {
		return err
	}
	modelName := cpuInfo[0].ModelName
	cpuCores := cpuInfo[0].Cores

	fmt.Printf("CPU Info\n")
	fmt.Printf("  CPU Model: %s\n", modelName)
	fmt.Printf("  CPU Cores: %d\n", cpuCores) //8 performance (P) + 2 efficiency (E) cores
	GetCpuUsage()

	return nil
}

/* <---------------- MEMORY INFO -----------------> */

func GetMemInfo() error {
	memoryInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	memoryTotal := BytesToGB(memoryInfo.Total)
	memoryUsed := BytesToGB(memoryInfo.Used)
	memoryFree := BytesToGB(memoryInfo.Available)

	fmt.Printf("Memory Info\n")
	fmt.Printf("  Total: %.2f GB\n", memoryTotal)
	fmt.Printf("  Used: %.2f GB\n", memoryUsed)
	fmt.Printf("  Free: %.2f GB\n", memoryFree)
	GetMemUsage()
	return nil
}

func GetMemUsage() error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	memPercent := memInfo.UsedPercent

	fmt.Printf("  Usage: %.2f%%\n\n", memPercent)

	return nil
}

/* <---------------- NETWORK INFO -----------------> */

func GetDownloadInfo() error {
	// Get initial network stats
	initial, err := net.IOCounters(false)
	if err != nil {
		panic(err)
	}

	// Wait for some time (e.g., 5 seconds)
	fmt.Println("Measuring network usage for 5 seconds...")
	time.Sleep(5 * time.Second)

	// Get stats again
	final, err := net.IOCounters(false)
	if err != nil {
		panic(err)
	}

	if len(initial) > 0 && len(final) > 0 {
		sent := final[0].BytesSent - initial[0].BytesSent
		recv := final[0].BytesRecv - initial[0].BytesRecv

		fmt.Printf("Network Info \n")
		fmt.Printf("  Upload:   %.2f MB\n", BytesToMB(sent))
		fmt.Printf("  Download: %.2f MB\n", BytesToMB(recv))
	}

	return nil
}
