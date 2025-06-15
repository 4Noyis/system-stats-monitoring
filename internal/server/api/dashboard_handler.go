package api

import (
	"net/http"
	"strings"
	"time"

	appLogger "github.com/4Noyis/system-stats-monitoring/internal/logger"
	"github.com/4Noyis/system-stats-monitoring/internal/server/database"
	"github.com/4Noyis/system-stats-monitoring/internal/server/models"

	"github.com/gin-gonic/gin"
)

// DashboardHandler holds dependencies for the dashboard API handlers.
type DashboardHandler struct {
	dbReader *database.InfluxDBReader
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(dbReader *database.InfluxDBReader) *DashboardHandler {
	return &DashboardHandler{
		dbReader: dbReader,
	}
}

// GetHostsOverview handles GET /api/dashboard/hosts/overview
func (h *DashboardHandler) GetHostsOverview(c *gin.Context) {
	overviews, err := h.dbReader.GetHostOverviewList(c.Request.Context())
	if err != nil {
		appLogger.Error("Failed to get hosts overview: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve hosts overview"})
		return
	}
	if overviews == nil { // Ensure we send an empty array instead of null if no hosts
		overviews = []models.HostOverviewData{}
	}
	c.JSON(http.StatusOK, overviews)
}

// GetHostDetailsByName handles GET /api/dashboard/host/:hostID/details
func (h *DashboardHandler) GetHostDetailsByID(c *gin.Context) {
	hostID := c.Param("hostID")
	if hostID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HostID parameter is required"})
		return
	}

	details, err := h.dbReader.GetHostDetails(c.Request.Context(), hostID)
	if err != nil {
		// dbReader.GetHostDetails might return a "not found" specific error if we implement it
		// For now, any error from there is treated as server error or potentially not found.
		if strings.Contains(err.Error(), "no system data found for host_id") {
			appLogger.Warn("Host details not found for hostID %s: %v", hostID, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Host details not found"})
		} else {
			appLogger.Error("Failed to get host details for hostID %s: %v", hostID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve host details"})
		}
		return
	}
	c.JSON(http.StatusOK, details)
}

// GetHostMetricHistory handles GET /api/dashboard/host/:hostID/metrics/:metricName
func (h *DashboardHandler) GetHostMetricHistory(c *gin.Context) {
	hostID := c.Param("hostID")
	metricName := c.Param("metricName") // e.g., "cpu_usage_percent", "mem_usage_percent"

	if hostID == "" || metricName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hostID and metricName parameters are required"})
		return
	}

	// Query parameters for time range and aggregation
	// Example: /api/dashboard/host/123/metrics/cpu_usage_percent?range=1h&aggregate=30s
	rangeStr := c.DefaultQuery("range", "1h")          // Default to 1 hour
	aggregateStr := c.DefaultQuery("aggregate", "30s") // Default to 30 second aggregates

	rangeDuration, err := time.ParseDuration(rangeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid range duration format"})
		return
	}
	aggregateInterval, err := time.ParseDuration(aggregateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid aggregate interval format"})
		return
	}

	// Basic validation for metricName (already done in dbReader, but good for early exit)
	// This could be more sophisticated, checking against a list of allowed metrics.
	allowedMetrics := map[string]bool{
		"cpu_usage_percent": true, "mem_usage_percent": true,
		"net_upload_bytes_sec": true, "net_download_bytes_sec": true,
	}
	if !allowedMetrics[metricName] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid metric name specified"})
		return
	}

	history, err := h.dbReader.GetHostMetricHistory(c.Request.Context(), hostID, metricName, rangeDuration, aggregateInterval)
	if err != nil {
		appLogger.Error("Failed to get metric history for host %s, metric %s: %v", hostID, metricName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metric history"})
		return
	}
	if history == nil { // Ensure empty array instead of null
		history = []models.MetricPoint{}
	}
	c.JSON(http.StatusOK, history)
}

// RegisterDashboardRoutes registers the API routes for dashboard data.
func (h *DashboardHandler) RegisterDashboardRoutes(router *gin.Engine) {
	// Prefixing with /api/dashboard to group dashboard related endpoints
	dashboardGroup := router.Group("/api/dashboard")
	{
		dashboardGroup.GET("/hosts/overview", h.GetHostsOverview)
		dashboardGroup.GET("/host/:hostID/details", h.GetHostDetailsByID)
		dashboardGroup.GET("/host/:hostID/metrics/:metricName", h.GetHostMetricHistory)

	}
}
