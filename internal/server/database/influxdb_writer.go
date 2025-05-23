package database

import (
	"context"
	"fmt"
	"time"

	appLogger "github.com/4Noyis/system-stats-monitoring/internal/logger"
	"github.com/4Noyis/system-stats-monitoring/internal/server/config"
	"github.com/4Noyis/system-stats-monitoring/internal/server/models"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// handles writing data to InfluxDB
type InfluxDBWriter struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	org      string
	bucket   string
}

// Create a new InfluxDBWriter
func NewInfluxDBWriter(cfg config.InfluxDBConfig) (*InfluxDBWriter, error) {
	client := influxdb2.NewClient(cfg.URL, cfg.Token)

	// Check connectivity (optional, but good for startup)
	// Use a timeout for the health check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	health, err := client.Health(ctx)
	if err != nil {
		appLogger.Error("InfluxDB health check failed: %v", err)
		return nil, fmt.Errorf("influxdb health check failed: %w", err)
	}
	if health.Status != "pass" {
		appLogger.Error("InfluxDB is not healthy: status %s, message %s", health.Status, *health.Message)
		return nil, fmt.Errorf("influxdb not healthy: status %s", health.Status)
	}
	appLogger.Info("Successfully connected to InfluxDB at %s", cfg.URL)

	writeAPI := client.WriteAPIBlocking(cfg.Org, cfg.Bucket)

	return &InfluxDBWriter{
		client:   client,
		writeAPI: writeAPI,
		org:      cfg.Org,
		bucket:   cfg.Bucket,
	}, nil
}

// converts the client payload into InfluxDB points and writes them.
func (w *InfluxDBWriter) WriteStats(ctx context.Context, payload *models.ClientPayload) error {

	// --- Create common tags for all points from this payload ---
	tags := map[string]string{
		"host_id":     payload.System.HostID,
		"hostname":    payload.System.Hostname,
		"os":          payload.System.OS,
		"kernel_arch": payload.System.KernelVersion,
	}

	// --- Create point for general system, CPU, and Memory stats ---
	measurement := "system_metrics"

	fields := map[string]interface{}{
		"uptime_seconds":         payload.System.Uptime,
		"cpu_model_name":         payload.CPU.ModelName, // String field
		"cpu_cores":              payload.CPU.Cores,
		"cpu_usage_percent":      payload.CPU.Usage,
		"mem_total_gb":           payload.Memory.TotalGB,
		"mem_used_gb":            payload.Memory.UsagePercent,
		"mem_available_gb":       payload.Memory.FreeGB,
		"mem_usage_percent":      payload.Memory.UsagePercent,
		"net_bytes_sent_period":  payload.Network.BytesSentPeriod, // Assuming aggregate network stats
		"net_bytes_recv_period":  payload.Network.BytesRecvPeriod,
		"net_upload_bytes_sec":   payload.Network.UploadBytesPerSec,
		"net_download_bytes_sec": payload.Network.DownloadBytesPerSec,
	}

	// Add network interface if available and not "all" or empty
	if payload.Network.InterfaceName != "" && payload.Network.InterfaceName != "all" {
		tags["net_interface"] = payload.Network.InterfaceName
	}

	// Create the point
	p := write.NewPoint(measurement, tags, fields, payload.CollectedAt)

	// write the point
	if err := w.writeAPI.WritePoint(ctx, p); err != nil {
		appLogger.Error("Failed to write system_metrics point to InfluxDB for host %s: %v", payload.System.HostID, err)
		return fmt.Errorf("influxdb write point error for system_metrics: %w", err)
	}
	appLogger.Debug("Successfully wrote system_metrics point for host %s at %s", payload.System.HostID, payload.CollectedAt)

	// --- Create separate points for each disk ---
	diskMeasurement := "disk_metrics"
	for _, disk := range payload.Disks {
		diskTags := make(map[string]string) // Create a new map for disk tags
		for k, v := range tags {            // Copy common tags
			diskTags[k] = v
		}
		diskTags["path"] = disk.Path // Add disk-specific tag

		diskFields := map[string]interface{}{
			"total_gb":      disk.TotalGB,
			"used_gb":       disk.UsedGB,
			"free_gb":       disk.FreeGB,
			"usage_percent": disk.UsagePercent,
		}
		diskPoint := write.NewPoint(diskMeasurement, diskTags, diskFields, payload.CollectedAt)
		if err := w.writeAPI.WritePoint(ctx, diskPoint); err != nil {
			appLogger.Error("Failed to write disk_metrics point for host %s, disk %s: %v", payload.System.HostID, disk.Path, err)
			// Continue to try writing other disk points
		} else {
			appLogger.Debug("Successfully wrote disk_metrics point for host %s, disk %s", payload.System.HostID, disk.Path)
		}
	}

	// ----- HANDLING PROCESSES ------
	processMeasurement := "process_metrics"
	for _, proc := range payload.Processes {
		processTags := make(map[string]string)
		for k, v := range tags {
			processTags[k] = v
		}
		processTags["pid"] = string(proc.PID)
		processTags["name"] = proc.Name

		processFields := map[string]interface{}{
			"cpu_percent": proc.CPUPercent,
			"mem_percent": proc.MemoryPercent,
			"user":        proc.Username,
		}
		processPoint := write.NewPoint(processMeasurement, processTags, processFields, payload.CollectedAt)
		if err := w.writeAPI.WritePoint(ctx, processPoint); err != nil {
			appLogger.Error("Failed to write process_metrics point for host %s, process %s (PID %d): %v", payload.System.HostID, proc.Name, proc.PID, err)
			// Continue writing other processes
		} else {
			appLogger.Debug("Successfully wrote process_metrics point for host %s, process %s (PID %d)", payload.System.HostID, proc.Name, proc.PID)
		}
	}

	return nil
}

// Close ensures the InfluxDB client is closed gracefully.
func (w *InfluxDBWriter) Close() {
	if w.client != nil {
		w.client.Close()
		appLogger.Info("InfluxDB client closed.")
	}
}
