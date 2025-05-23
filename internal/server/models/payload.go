package models

import "time"

// --- These structs should mirror what the client sends ---

type SystemInfoPayload struct {
	Hostname      string `json:"hostname"`
	HostID        string `json:"host_id"`
	OS            string `json:"os"`
	OSVersion     string `json:"os_version"`
	Kernel        string `json:"kernel"`
	KernelVersion string `json:"kernel_version"`
	Uptime        string `json:"uptime"`
}

type CPUInfoPayload struct {
	ModelName string  `json:"model_name"`
	Cores     int32   `json:"cores"`
	Usage     float64 `json:"usage_percent"` // Combined from GetCpuUsage
}

type MemInfoPayload struct {
	TotalGB      float64 `json:"total_gb"`
	FreeGB       float64 `json:"free_gb"` // From memoryInfo.Available
	UsagePercent float64 `json:"usage_percent"`
}

type NetworkPayload struct {
	InterfaceName       string  `json:"interface_name,omitempty"` // "all" for aggregate
	BytesSentPeriod     uint64  `json:"bytes_sent_period"`
	BytesRecvPeriod     uint64  `json:"bytes_recv_period"`
	PacketsSentPeriod   uint64  `json:"packets_sent_period"`
	PacketsRecvPeriod   uint64  `json:"packets_recv_period"`
	UploadBytesPerSec   float64 `json:"upload_bytes_per_sec"`
	DownloadBytesPerSec float64 `json:"download_bytes_per_sec"`
}
type ProcessPayload struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
	Username      string  `json:"username"`
	// Add more fields as needed, e.g., status, command line
}

type DiskUsagePayload struct {
	Path         string  `json:"path"`
	TotalGB      float64 `json:"total_gb"`
	UsedGB       float64 `json:"used_gb"`
	FreeGB       float64 `json:"free_gb"`
	UsagePercent float64 `json:"usage_percent"`
}

// ClientPayload is the top-level struct expected from the client.
// This must match the AllHostStats struct sent by your client.
type ClientPayload struct {
	CollectedAt time.Time          `json:"collected_at"` // Crucial for InfluxDB timestamp
	System      SystemInfoPayload  `json:"system_info"`
	CPU         CPUInfoPayload     `json:"cpu_info"`
	Memory      MemInfoPayload     `json:"memory_info"`
	Network     NetworkPayload     `json:"network_info"`
	Processes   []ProcessPayload   `json:"processes,omitempty"`
	Disks       []DiskUsagePayload `json:"disk_usage,omitempty"`
}
