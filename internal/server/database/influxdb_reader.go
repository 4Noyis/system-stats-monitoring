package database

import (
	"context"
	"fmt"
	"sort"
	"time"

	appLogger "github.com/4Noyis/system-stats-monitoring/internal/logger"
	"github.com/4Noyis/system-stats-monitoring/internal/server/config"
	"github.com/4Noyis/system-stats-monitoring/internal/server/models"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

const (
	defaultLookbackWindow = 15 * time.Second // last seen
	activeHostLookback    = 30 * time.Second // for determining online status
)

type InfluxDBReader struct {
	client   influxdb2.Client
	queryAPI api.QueryAPI
	org      string
	bucket   string
}

// NewInfluxDBReader creates a new InfluxDBReader.
func NewInfluxDBReader(cfg config.InfluxDBConfig) (*InfluxDBReader, error) {
	// Client setup is similar to InfluxDBWriter
	// Consider sharing the client if both reader and writer are heavily used,
	// but for now, separate clients are fine and simpler.
	client := influxdb2.NewClient(cfg.URL, cfg.Token)
	// Health check (optional but good)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("influxdb health check failed for reader: %w", err)
	}
	if health.Status != "pass" {
		return nil, fmt.Errorf("influxdb not healthy for reader: status %s", health.Status)
	}
	appLogger.Info("InfluxDBReader successfully connected to InfluxDB at %s", cfg.URL)

	queryAPI := client.QueryAPI(cfg.Org)
	return &InfluxDBReader{
		client:   client,
		queryAPI: queryAPI,
		org:      cfg.Org,
		bucket:   cfg.Bucket,
	}, nil
}

func (r *InfluxDBReader) GetHostOverviewList(ctx context.Context) ([]models.HostOverviewData, error) {
	query := fmt.Sprintf(`
		import "influxdata/influxdb/schema"
		import "join"

		systemData = from(bucket: "%s")
			|> range(start: -%s)
			|> filter(fn: (r) => r._measurement == "system_metrics")
			|> last()
			|> pivot(rowKey:["_time", "host_id", "hostname"], columnKey: ["_field"], valueColumn: "_value")
			|> map(fn: (r) => { // Using explicit map structure
				return {
					_time: r._time,
					host_id: r.host_id,
					hostname: r.hostname,
					cpu_usage_percent: if exists r.cpu_usage_percent then r.cpu_usage_percent else 0.0,
					mem_usage_percent: if exists r.mem_usage_percent then r.mem_usage_percent else 0.0,
					// uptime_seconds: REMOVED FOR TESTING
					net_upload_bytes_sec: if exists r.net_upload_bytes_sec then r.net_upload_bytes_sec else 0.0,
					net_download_bytes_sec: if exists r.net_download_bytes_sec then r.net_download_bytes_sec else 0.0
				}
			})

		rootDiskUsage = from(bucket: "%s")
			|> range(start: -%s)
			|> filter(fn: (r) => 
				r._measurement == "disk_metrics" and 
				r._field == "usage_percent" and 
				r.path == "/"
			)
			|> group(columns: ["host_id"])
			|> last()
			|> rename(columns: {_value: "root_disk_usage_percent"})
			|> keep(columns: ["host_id", "root_disk_usage_percent"])

		join.left(
			left: systemData,
			right: rootDiskUsage,
			on: (l, r) => l.host_id == r.host_id,
			as: (l, r) => ({
				_time: l._time,
				host_id: l.host_id,
				hostname: l.hostname,
				cpu_usage_percent: l.cpu_usage_percent,
				mem_usage_percent: l.mem_usage_percent,
				// uptime_seconds: REMOVED FOR TESTING
				net_upload_bytes_sec: l.net_upload_bytes_sec,
				net_download_bytes_sec: l.net_download_bytes_sec,
				disk_usage_percent: if exists r.root_disk_usage_percent then r.root_disk_usage_percent else 0.0
			})
		)
		|> yield(name: "overview")
	`, r.bucket, activeHostLookback.String(), /* for systemData */
		r.bucket, activeHostLookback.String() /* for rootDiskUsage */)

	appLogger.Debug("GetHostOverviewList Query:\n%s", query) // Log the query
	results, err := r.queryAPI.Query(ctx, query)
	if err != nil {
		appLogger.Error("InfluxDB query failed for GetHostOverviewList: %v", err)
		return nil, fmt.Errorf("query influxdb for host overview: %w", err)
	}

	var overviews []models.HostOverviewData
	now := time.Now()

	for results.Next() {
		record := results.Record()
		getFloat := func(field string) float64 {
			val, ok := record.ValueByKey(field).(float64)
			if !ok {
				return 0.0
			}
			return val
		}

		overview := models.HostOverviewData{
			ID:              record.ValueByKey("host_id").(string),
			Hostname:        record.ValueByKey("hostname").(string),
			CPUUsage:        getFloat("cpu_usage_percent"),
			RAMUsage:        getFloat("mem_usage_percent"),
			DiskUsage:       getFloat("disk_usage_percent"), // This now directly comes from 'root_disk_usage_percent'
			NetworkUpload:   getFloat("net_upload_bytes_sec"),
			NetworkDownload: getFloat("net_download_bytes_sec"),
			//UptimeSeconds:   record.ValueByKey("uptime_seconds").(string),
			LastSeen: record.Time(),
		}

		if now.Sub(overview.LastSeen) <= activeHostLookback+(5*time.Second) {
			overview.Status = "online"
			if overview.CPUUsage > 85 || overview.RAMUsage > 85 || overview.DiskUsage > 90 {
				overview.Status = "warning"
			}
		} else {
			overview.Status = "offline"
		}
		overviews = append(overviews, overview)
	}

	if results.Err() != nil {
		appLogger.Error("Error processing results for GetHostOverviewList: %v", results.Err())
		return nil, fmt.Errorf("process query results for host overview: %w", results.Err())
	}

	sort.Slice(overviews, func(i, j int) bool {
		return overviews[i].Hostname < overviews[j].Hostname
	})

	return overviews, nil
}

// GetHostDetails fetches detailed information for a single host.
func (r *InfluxDBReader) GetHostDetails(ctx context.Context, hostID string) (*models.HostDetailsData, error) {

	// --- Query for System Data ---
	systemQuery := fmt.Sprintf(`
    from(bucket: "%s")
        |> range(start: -%s)
        |> filter(fn: (r) => r._measurement == "system_metrics" and r.host_id == "%s")
        |> last()
        |> pivot(rowKey:["_time", "host_id"], columnKey: ["_field"], valueColumn: "_value")
        |> map(fn: (r) => ({
            _time: r._time,
            host_id: r.host_id,
            // Ensure all fields from the pivot that you need are here
            hostname: if exists r.hostname then r.hostname else "",
            cpu_cores: if exists r.cpu_cores then int(v: r.cpu_cores) else 0,
            cpu_model_name: if exists r.cpu_model_name then r.cpu_model_name else "",
            cpu_usage_percent: if exists r.cpu_usage_percent then r.cpu_usage_percent else 0.0,
            mem_available_gb: if exists r.mem_available_gb then r.mem_available_gb else 0.0,
            mem_total_gb: if exists r.mem_total_gb then r.mem_total_gb else 0.0,
            mem_used_gb: if exists r.mem_used_gb then r.mem_used_gb else 0.0,
            mem_usage_percent: if exists r.mem_usage_percent then r.mem_usage_percent else 0.0,
            net_download_bytes_sec: if exists r.net_download_bytes_sec then r.net_download_bytes_sec else 0.0,
            net_upload_bytes_sec: if exists r.net_upload_bytes_sec then r.net_upload_bytes_sec else 0.0,
            os: if exists r.os then r.os else "",
            os_version: if exists r.os_version then r.os_version else "",
			kernel: if exists r.kernel then r.kernel else "",
            kernel_arch: if exists r.kernel_arch then r.kernel_arch else "",
            // uptime_seconds: if exists r.uptime_seconds then uint(v: r.uptime_seconds) else uint(v: 0) // if you re-add it
        })) // <<<< THIS IS THE END OF THE map() call.
           // There is no findRecord after this.
`, r.bucket, defaultLookbackWindow, hostID)

	appLogger.Debug("GetHostDetails System Query for host %s:\n%s", hostID, systemQuery)
	sysResults, err := r.queryAPI.Query(ctx, systemQuery)
	if err != nil {
		appLogger.Error("InfluxDB query failed for GetHostDetails (system) for host %s: %v", hostID, err)
		return nil, fmt.Errorf("query influxdb for host details (system): %w", err)
	}

	if !sysResults.Next() {
		if sysResults.Err() != nil {
			appLogger.Error("Error processing system results for GetHostDetails host %s: %v", hostID, sysResults.Err())
			return nil, fmt.Errorf("no data found for host %s or query error: %w", hostID, sysResults.Err())
		}
		appLogger.Warn("No system data found for host_id: %s", hostID)
		return nil, fmt.Errorf("no system data found for host_id: %s", hostID) // Or return a specific "not found" error
	}
	record := sysResults.Record()
	if sysResults.Err() != nil { // Check error after Next()
		appLogger.Error("Error after Next() for system results, host %s: %v", hostID, sysResults.Err())
		return nil, fmt.Errorf("error processing system record for host %s: %w", hostID, sysResults.Err())
	}

	// Helper to get float, defaulting to 0.0 if not found or wrong type
	getF := func(key string) float64 {
		v, ok := record.ValueByKey(key).(float64)
		if !ok {
			return 0.0
		}
		return v
	}

	// Helper to get int32, defaulting to 0 if not found or wrong type
	getI32 := func(key string) int32 {
		val, ok := record.ValueByKey(key).(int64) // Flux typically returns integers as int64
		if !ok {
			fVal, fOk := record.ValueByKey(key).(float64) // Or float for some reason
			if fOk {
				return int32(fVal)
			}
			return 0
		}
		return int32(val)
	}
	// Helper to get string, defaulting to ""
	getS := func(key string) string {
		v, ok := record.ValueByKey(key).(string)
		if !ok {
			return ""
		}
		return v
	}

	details := &models.HostDetailsData{
		ID:       hostID,
		Hostname: getS("hostname"),
		//UptimeSeconds: getS("uptime_seconds"),
		LastSeen: record.Time(),
		CPU: models.CPUDetails{
			Cores:     getI32("cpu_cores"),
			ModelName: getS("cpu_model_name"),
		},
		Memory: models.MemoryDetails{
			TotalGB:      getF("mem_total_gb"),
			AvailableGB:  getF("mem_available_gb"),
			UsagePercent: getF("mem_used_gb"),
		},
		OS: models.OSLiteralDetails{
			Name:       getS("os"), // Assuming 'os' field in system_metrics stores this
			Version:    getS("os_version"),
			Kernel:     getS("kernel"),
			KernelArch: getS("kernel_arch"),
		},
		CPUUsage:        getF("cpu_usage_percent"),
		RAMUsage:        getF("mem_usage_percent"),
		NetworkUpload:   getF("net_upload_bytes_sec"),
		NetworkDownload: getF("net_download_bytes_sec"),
	}

	// --- Query for Root Disk Data ---
	diskQuery := fmt.Sprintf(`
    from(bucket: "%s")
        |> range(start: -%s)
        |> filter(fn: (r) => 
            r._measurement == "disk_metrics" and 
            r.host_id == "%s" and 
            r.path == "/"
        )
        |> last()
        |> pivot(rowKey:["_time", "host_id", "path"], columnKey: ["_field"], valueColumn: "_value")

	`, r.bucket, defaultLookbackWindow, hostID)

	appLogger.Debug("GetHostDetails Disk Query for host %s:\n%s", hostID, diskQuery)
	diskResults, err := r.queryAPI.Query(ctx, diskQuery)
	if err != nil {
		appLogger.Error("InfluxDB query failed for GetHostDetails (root disk) for host %s: %v", hostID, err)
		// Set default empty disk details or handle error as appropriate
		details.Disk = models.RootDiskDetails{Path: "/"} // Indicate path even if data is missing
	} else {
		if diskResults.Next() {
			dRec := diskResults.Record()
			getDF := func(key string) float64 {
				v, ok := dRec.ValueByKey(key).(float64)
				if !ok {
					return 0.0
				}
				return v
			}

			details.Disk = models.RootDiskDetails{
				Path:         dRec.ValueByKey("path").(string), // Should be "/"
				TotalGB:      getDF("total_gb"),
				UsedGB:       getDF("used_gb"),
				FreeGB:       getDF("free_gb"),
				UsagePercent: getDF("usage_percent"),
			}
		} else {
			appLogger.Warn("No root disk data found for host_id: %s", hostID)
			details.Disk = models.RootDiskDetails{Path: "/"} // Default if no record found
		}
		if diskResults.Err() != nil {
			appLogger.Error("Error processing root disk results for host %s: %v", hostID, diskResults.Err())
			// Disk details might be partially populated or default
		}
	}

	// --- Query for Process Metrics ---
	// --- Query for Process Metrics (Username field excluded for testing) ---
	processMap := make(map[string]*models.ProcessDetail) // Pointer to modify in place

	// Query 1: Get mem_percent and base process info (pid, name)
	processQuery_mem_and_tags := fmt.Sprintf(`
		targetFields = ["mem_percent"] 
		from(bucket: "%s")
			|> range(start: -%s)
			|> filter(fn: (r) => r._measurement == "process_metrics" and r.host_id == "%s" and contains(value: r._field, set: targetFields))
			|> group(columns: ["host_id", "pid", "name"]) 
			|> last() 
			|> pivot(rowKey:["_time", "host_id", "pid", "name"], columnKey: ["_field"], valueColumn: "_value")
	`, r.bucket, defaultLookbackWindow, hostID)

	appLogger.Debug("GetHostDetails Process Query (Mem & Tags) for host %s:\n%s", hostID, processQuery_mem_and_tags)
	memResults, memErr := r.queryAPI.Query(ctx, processQuery_mem_and_tags)
	if memErr != nil {
		appLogger.Error("InfluxDB query failed for GetHostDetails (processes mem_and_tags) for host %s: %v", hostID, memErr)
	} else {
		for memResults.Next() {
			pRec := memResults.Record()
			getPF := func(key string) float64 { /* ... same as before ... */
				val, ok := pRec.ValueByKey(key).(float64)
				if !ok {
					appLogger.Warn("[MemQuery] Field '%s' expected float64, got %T for process PID '%s', Name '%s'", key, pRec.ValueByKey(key), pRec.ValueByKey("pid"), pRec.ValueByKey("name"))
					return 0.0
				}
				return val
			}

			pidStr, _ := pRec.ValueByKey("pid").(string)
			nameStr, _ := pRec.ValueByKey("name").(string)
			var pidVal int32
			_, scanErr := fmt.Sscan(pidStr, &pidVal)
			if scanErr != nil { /* ... log error ... */
			}

			processKey := fmt.Sprintf("%s_%s", pidStr, nameStr) // Unique key for the map
			procDetail := &models.ProcessDetail{
				PID:           pidVal,
				Name:          nameStr,
				MemoryPercent: float32(getPF("mem_percent")),
				CPUPercent:    0, // Default, will be updated by CPU query
				// Username: "", // If you bring it back
			}
			processMap[processKey] = procDetail
		}
		if memResults.Err() != nil {
			appLogger.Error("Error processing process mem_and_tags results for host %s: %v", hostID, memResults.Err())
		}
	}

	// Query 2: Get cpu_percent
	processQuery_cpu := fmt.Sprintf(`
		targetFields = ["cpu_percent"]
		from(bucket: "%s")
			|> range(start: -%s)
			|> filter(fn: (r) => r._measurement == "process_metrics" and r.host_id == "%s" and contains(value: r._field, set: targetFields))
			|> group(columns: ["host_id", "pid", "name"])
			|> last()
			|> pivot(rowKey:["_time", "host_id", "pid", "name"], columnKey: ["_field"], valueColumn: "_value")
	`, r.bucket, defaultLookbackWindow, hostID)

	appLogger.Debug("GetHostDetails Process Query (CPU) for host %s:\n%s", hostID, processQuery_cpu)
	cpuResults, cpuErr := r.queryAPI.Query(ctx, processQuery_cpu)
	if cpuErr != nil {
		appLogger.Error("InfluxDB query failed for GetHostDetails (processes cpu) for host %s: %v", hostID, cpuErr)
	} else {
		for cpuResults.Next() {
			pRec := cpuResults.Record()
			getPF := func(key string) float64 { /* ... same as before ... */
				val, ok := pRec.ValueByKey(key).(float64)
				if !ok {
					appLogger.Warn("[CPUQuery] Field '%s' expected float64, got %T for process PID '%s', Name '%s'", key, pRec.ValueByKey(key), pRec.ValueByKey("pid"), pRec.ValueByKey("name"))
					return 0.0
				}
				return val
			}

			pidStr, _ := pRec.ValueByKey("pid").(string)
			nameStr, _ := pRec.ValueByKey("name").(string)

			processKey := fmt.Sprintf("%s_%s", pidStr, nameStr)
			if procDetail, exists := processMap[processKey]; exists {
				procDetail.CPUPercent = getPF("cpu_percent")
			} else {
				// This case means a process had CPU usage but no memory usage reported in the first query
				// or there's a timing mismatch. You might want to create a new entry or log it.
				appLogger.Warn("Found CPU data for process PID '%s', Name '%s' but no prior mem data. Creating new entry.", pidStr, nameStr)
				var pidVal int32 // Need to parse pidStr again if creating new
				_, scanErr := fmt.Sscan(pidStr, &pidVal)
				if scanErr != nil { /* ... log error ... */
				}

				newProcDetail := &models.ProcessDetail{
					PID:           pidVal,
					Name:          nameStr,
					CPUPercent:    getPF("cpu_percent"),
					MemoryPercent: 0, // No memory data from first query
				}
				processMap[processKey] = newProcDetail
			}
		}
		if cpuResults.Err() != nil {
			appLogger.Error("Error processing process cpu results for host %s: %v", hostID, cpuResults.Err())
		}
	}

	// Convert map to slice for the final details struct
	var finalProcesses []models.ProcessDetail
	for _, procDetail := range processMap {
		finalProcesses = append(finalProcesses, *procDetail)
	}
	// Optionally sort finalProcesses, e.g., by PID or Name
	sort.Slice(finalProcesses, func(i, j int) bool {
		return finalProcesses[i].PID < finalProcesses[j].PID
	})
	details.Processes = finalProcesses

	// Determine status
	if time.Since(details.LastSeen) <= activeHostLookback+(5*time.Second) {
		details.Status = "online"
		if details.CPUUsage > 85 || details.RAMUsage > 85 { // Add disk warning later
			details.Status = "warning"
		}
	} else {
		details.Status = "offline"
	}

	return details, nil
}

// GetHostMetricHistory fetches time-series data for a specific metric of a host.
func (r *InfluxDBReader) GetHostMetricHistory(ctx context.Context, hostID, metricField string, rangeStart time.Duration, aggregateInterval time.Duration) ([]models.MetricPoint, error) {
	// Validate metricField to prevent injection and ensure it's a known numeric field
	validNumericFields := map[string]bool{
		"cpu_usage_percent":      true,
		"mem_usage_percent":      true,
		"net_upload_bytes_sec":   true,
		"net_download_bytes_sec": true,
		// Add disk usage later if needed, requires specifying path
	}
	if !validNumericFields[metricField] {
		return nil, fmt.Errorf("invalid or non-numeric metric field for history: %s", metricField)
	}

	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -%s)
			|> filter(fn: (r) => r._measurement == "system_metrics" and r.host_id == "%s" and r._field == "%s")
			|> aggregateWindow(every: %s, fn: mean, createEmpty: false) // Use mean for aggregation
			|> yield(name: "mean")
	`, r.bucket, rangeStart.String(), hostID, metricField, aggregateInterval.String())

	appLogger.Debug("GetHostMetricHistory Query for host %s, metric %s:\n%s", hostID, metricField, query)
	results, err := r.queryAPI.Query(ctx, query)
	if err != nil {
		appLogger.Error("InfluxDB query failed for GetHostMetricHistory (host %s, metric %s): %v", hostID, metricField, err)
		return nil, fmt.Errorf("query influxdb for host metric history: %w", err)
	}

	var points []models.MetricPoint
	for results.Next() {
		record := results.Record()
		value, ok := record.Value().(float64) // Assuming aggregated values are float64
		if !ok {
			// Try int64 then cast, sometimes it might be integer if original data was integer and aggregateWindow didn't change type
			ival, iok := record.Value().(int64)
			if iok {
				value = float64(ival)
				ok = true
			} else {
				appLogger.Warn("Unexpected value type for metric %s, host %s: %T, value: %v", metricField, hostID, record.Value(), record.Value())
				continue // Skip if not a float or convertible int
			}
		}

		points = append(points, models.MetricPoint{
			// Format timestamp as "HH:MM" as in your mock data
			Timestamp: record.Time().In(time.Local).Format("15:04"), // Use local time for display
			Value:     value,
		})
	}

	if results.Err() != nil {
		appLogger.Error("Error processing results for GetHostMetricHistory (host %s, metric %s): %v", hostID, metricField, results.Err())
		return nil, fmt.Errorf("process query results for host metric history: %w", results.Err())
	}

	// Data from InfluxDB is typically time-sorted, but ensure if needed
	// sort.Slice(points, func(i, j int) bool { return points[i].Timestamp < points[j].Timestamp })

	return points, nil
}

// Close cleans up resources.
func (r *InfluxDBReader) Close() {
	if r.client != nil {
		r.client.Close()
		appLogger.Info("InfluxDBReader client closed.")
	}
}
