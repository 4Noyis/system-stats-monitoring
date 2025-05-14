package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/net"
)

type MemInfoData struct {
	TotalGB      float64 `json:"total_gb"`
	UsedGB       float64 `json:"used_gb"`
	FreeGB       float64 `json:"free_gb"` // From memoryInfo.Available
	UsagePercent float64 `json:"usage_percent"`
}

type NetworkData struct {
	UploadMB   float64 `json:"upload_mb_period"`   // MB over the measurement period
	DownloadMB float64 `json:"download_mb_period"` // MB over the measurement period
	// Consider adding rates: UploadBytesPerSec, DownloadBytesPerSec
}

// Converts bytes to gigabytes
func BytesToGB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024 * 1024)
}

// Converts bytes to megabytes
func BytesToMB(bytes uint64) float64 {
	return float64(bytes) / (1024 * 1024)
}

/* <---------------- SYSTEM INFO -----------------> */

type SystemInfoData struct {
	Hostname      string `json:"hostname"`
	HostID        string `json:"host_id"`
	OS            string `json:"os"`
	OSVersion     string `json:"os_version"`
	Kernel        string `json:"kernel"`
	KernelVersion string `json:"kernel_version"`
	Uptime        string `json:"uptime"`
}

func GetSystemInfo() (SystemInfoData, error) {
	var data SystemInfoData

	SystemInfo, err := host.Info()
	if err != nil {
		return data, fmt.Errorf("error getting System info: %w", err)
	}

	data.Hostname = SystemInfo.Hostname
	data.HostID = SystemInfo.HostID
	data.OS = SystemInfo.OS

	data.OSVersion = SystemInfo.PlatformVersion
	data.Kernel = SystemInfo.KernelArch
	data.KernelVersion = SystemInfo.KernelVersion

	uptime := time.Duration(SystemInfo.Uptime) * time.Second
	uptime = uptime.Round(time.Second)
	data.Uptime = uptime.String()

	return data, nil
}

/* <---------------- CPU INFO -----------------> */

type CPUInfoData struct {
	ModelName string  `json:"model_name"`
	Cores     int32   `json:"cores"`
	Usage     float64 `json:"usage_percent"` // Combined from GetCpuUsage
}

func GetCPUInfo() (CPUInfoData, error) {

	var data CPUInfoData

	cpuInfos, err := cpu.Info()
	if err != nil {
		return data, fmt.Errorf("error getting CPU info: %w", err)
	}
	if len(cpuInfos) > 0 {
		data.ModelName = cpuInfos[0].ModelName
		data.Cores = cpuInfos[0].Cores // This is physical cores * sockets * threads per core usually. Or logical processors.
	} else {
		return data, fmt.Errorf("no CPU info found")
	}

	// Get CPU Usage
	percent, err := cpu.Percent(time.Second, false) // false -> overall percentage
	if err != nil {
		return data, fmt.Errorf("error getting CPU usage %w", err)
	}
	if len(percent) > 0 {
		data.Usage = percent[0]
	} else {
		return data, fmt.Errorf("could not retrieve CPU usage percentage")
	}
	return data, nil
}

// StartCPUMonitor continuously monitors CPU usage
func StartCPUMonitor(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Println("Starting CPU monitor...")
	for {
		select {
		case <-ticker.C:
			percent, err := cpu.Percent(time.Second, false) // Use a short interval for measurement
			if err != nil {
				fmt.Printf("Error getting CPU usage: %v\n", err)
				continue
			}
			if len(percent) > 0 {
				fmt.Printf("[Live CPU Usage]: %.2f%%\n", percent[0]) // direkt bunu return ederek veriyi elde ederiz
			}
		case <-ctx.Done():
			fmt.Println("Stopping CPU monitor.")
			return // Exit goroutine
		}
	}
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

// GetMemUsage is kept for one-time snapshot
func GetMemUsage() error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	memPercent := memInfo.UsedPercent
	fmt.Printf("  Usage (snapshot): %.2f%%\n\n", memPercent)
	return nil
}

// StartMemoryMonitor continuously monitors memory usage
func StartMemoryMonitor(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Println("Starting Memory monitor...")
	for {
		select {
		case <-ticker.C:
			memInfo, err := mem.VirtualMemory()
			if err != nil {
				fmt.Printf("Error getting Memory usage: %v\n", err)
				continue
			}
			fmt.Printf("[Live Memory Usage]: %.2f%%\n", memInfo.UsedPercent) // direkt bunu return ederek veriyi elde ederiz
		case <-ctx.Done():
			fmt.Println("Stopping Memory monitor.")
			return // Exit goroutine
		}
	}
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
