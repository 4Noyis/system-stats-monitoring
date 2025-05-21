package stats

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/net"
)

type SystemInfoData struct {
	Hostname      string `json:"hostname"`
	HostID        string `json:"host_id"`
	OS            string `json:"os"`
	OSVersion     string `json:"os_version"`
	Kernel        string `json:"kernel"`
	KernelVersion string `json:"kernel_version"`
	Uptime        string `json:"uptime"`
}

type CPUInfoData struct {
	ModelName string  `json:"model_name"`
	Cores     int32   `json:"cores"`
	Usage     float64 `json:"usage_percent"` // Combined from GetCpuUsage
}

type MemInfoData struct {
	TotalGB      float64 `json:"total_gb"`
	FreeGB       float64 `json:"free_gb"` // From memoryInfo.Available
	UsagePercent float64 `json:"usage_percent"`
}

type NetworkData struct {
	InterfaceName       string  `json:"interface_name,omitempty"` // "all" for aggregate
	BytesSentPeriod     uint64  `json:"bytes_sent_period"`
	BytesRecvPeriod     uint64  `json:"bytes_recv_period"`
	PacketsSentPeriod   uint64  `json:"packets_sent_period"`
	PacketsRecvPeriod   uint64  `json:"packets_recv_period"`
	UploadBytesPerSec   float64 `json:"upload_bytes_per_sec"`
	DownloadBytesPerSec float64 `json:"download_bytes_per_sec"`
}
type ProcessData struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
	Username      string  `json:"username"`
	// Add more fields as needed, e.g., status, command line
}

type DiskUsageData struct {
	Path         string  `json:"path"`
	TotalGB      float64 `json:"total_gb"`
	UsedGB       float64 `json:"used_gb"`
	FreeGB       float64 `json:"free_gb"`
	UsagePercent float64 `json:"usage_percent"`
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
		usage := math.Round(percent[0]*100) / 100
		data.Usage = usage
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

func GetMemInfo() (MemInfoData, error) {
	var data MemInfoData

	memoryInfo, err := mem.VirtualMemory()
	if err != nil {
		return data, fmt.Errorf("error getting Memory info: %w", err)
	}
	if memoryInfo != nil {
		data.TotalGB = BytesToGB(memoryInfo.Total)
		data.FreeGB = BytesToGB(memoryInfo.Available)
	} else {
		return data, fmt.Errorf("no Memory info found")
	}

	// Get memory usage Percent
	memoryPercent := math.Round(memoryInfo.UsedPercent*100) / 100
	data.UsagePercent = memoryPercent

	return data, nil

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

func GetCurrentIOCounters() (net.IOCountersStat, error) {
	ioCounters, err := net.IOCounters(false) // false for aggregate (sum of all interfaces)
	if err != nil {
		return net.IOCountersStat{}, fmt.Errorf("failed to get I/O counters: %w", err)
	}
	if len(ioCounters) == 0 {
		return net.IOCountersStat{}, fmt.Errorf("no I/O counters returned")
	}
	return ioCounters[0], nil // Return the first (and only) element for aggregate stats
}

func CalculateNetworkRates(current, previous net.IOCountersStat, duration time.Duration) (NetworkData, error) {
	var data NetworkData
	data.InterfaceName = "all"

	if duration.Seconds() <= 0 {
		return data, fmt.Errorf("duration must be positive, got %v", duration)
	}

	data.BytesSentPeriod = current.BytesSent - previous.BytesSent
	data.BytesRecvPeriod = current.BytesRecv - previous.BytesRecv
	data.PacketsSentPeriod = current.PacketsSent - previous.PacketsSent
	data.PacketsRecvPeriod = current.PacketsRecv - previous.PacketsRecv

	// Calculate rates per second
	durationSeconds := duration.Seconds()
	data.UploadBytesPerSec = float64(data.BytesSentPeriod) / durationSeconds
	data.DownloadBytesPerSec = float64(data.BytesRecvPeriod) / durationSeconds

	return data, nil
}

/* <----------------  PROCESSES INFO -----------------> */
func GetProcessList(count float64) ([]ProcessData, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, err
	}

	var processes []ProcessData

	for _, pid := range pids {
		proc, err := process.NewProcess(pid)
		if err != nil {
			continue
		}
		cpuPercent, _ := proc.CPUPercent()
		memPercent, _ := proc.MemoryPercent()

		if cpuPercent > count || memPercent > float32(count) {
			name, _ := proc.Name()
			username, _ := proc.Username()

			processes = append(processes, ProcessData{
				PID:           pid,
				Name:          name,
				CPUPercent:    cpuPercent,
				MemoryPercent: memPercent,
				Username:      username,
			})

		}

		// for limited number process
		// if count > 0 && len(processes) >= count {

		// }
	}
	return processes, nil
}

/* <----------------  DISK INFO -----------------> */
func GetDiskUsageInfo() ([]DiskUsageData, error) {
	// partitions, err := disk.Partitions(false) // false for physical devices only
	// if err != nil {
	// 	return nil, err
	// }

	var usages []DiskUsageData

	usage, err := disk.Usage("/")
	if err != nil {
		fmt.Printf("Could not get usage: %v\n", err)

	}

	usages = append(usages, DiskUsageData{
		Path:         usage.Path,
		TotalGB:      BytesToGB(usage.Total),
		UsedGB:       BytesToGB(usage.Used),
		FreeGB:       BytesToGB(usage.Free),
		UsagePercent: usage.UsedPercent,
	})

	return usages, nil

}
