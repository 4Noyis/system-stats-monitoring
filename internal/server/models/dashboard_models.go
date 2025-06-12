package models

import "time"

type HostOverviewData struct {
	ID              string  `json:"id"` //HostID
	Hostname        string  `json:"hostname"`
	Status          string  `json:"status"` // online, offline, warning
	CPUUsage        float64 `json:"cpuUsage"`
	RAMUsage        float64 `json:"ramUsage"`
	DiskUsage       float64 `json:"diskUsage"`
	NetworkUpload   float64 `json:"networkUpload"`   // Bytes/sec
	NetworkDownload float64 `json:"networkDownload"` // Bytes/sec
	// UptimeSeconds   string    `json:"uptimeSeconds"`   // Client send seconds
	LastSeen time.Time `json:"lastSeen"`
}

// For timeseries chart data
type MetricPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type CPUDetails struct {
	Cores     int32  `json:"cores"`
	ModelName string `json:"model_name"`
}

type MemoryDetails struct {
	TotalGB      float64 `json:"total_gb"`      // Total memory in GB
	AvailableGB  float64 `json:"free_gb"`       // Available memory in GB (maps to 'free' in mock)
	UsagePercent float64 `json:"usage_percent"` // not Used GB, Percent of Usage
}

type RootDiskDetails struct {
	Path         string  `json:"path"`
	TotalGB      float64 `json:"total_gb"`
	UsedGB       float64 `json:"used_gb"`
	FreeGB       float64 `json:"free_gb"`
	UsagePercent float64 `json:"usage_percent"`
}

type OSLiteralDetails struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Kernel     string `json:"kernel"`
	KernelArch string `json:"kernelArch"`
}

type ProcessDetail struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
	Username      string  `json:"username"`
}

type HostDetailsData struct {
	ID       string `json:"id"` // HostID
	Hostname string `json:"hostname"`
	Status   string `json:"status"` // online, offline, warning
	//	UptimeSeconds   string           `json:"uptimeSeconds"`
	LastSeen        time.Time        `json:"lastSeen"`
	CPU             CPUDetails       `json:"cpu"`
	Memory          MemoryDetails    `json:"memory"`
	Disk            RootDiskDetails  `json:"disk"`
	OS              OSLiteralDetails `json:"os"`
	Processes       []ProcessDetail  `json:"processes,omitempty"`
	CPUUsage        float64          `json:"cpuUsage"`
	RAMUsage        float64          `json:"ramUsage"`      // Memory usage percent
	NetworkUpload   float64          `json:"networkUpload"` // Bytes/sec
	NetworkDownload float64          `json:"networkDownload"`
}
