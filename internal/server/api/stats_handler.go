package api

import (
	"net/http"

	appLogger "github.com/4Noyis/system-stats-monitoring/internal/logger"
	"github.com/4Noyis/system-stats-monitoring/internal/server/database"
	"github.com/4Noyis/system-stats-monitoring/internal/server/models"
	"github.com/gin-gonic/gin"
)

// holds depebndencies for the stats API handlers
type StatsHandler struct {
	dbWriter *database.InfluxDBWriter
}

// creates a new StatsHandler
func NewStatsHandler(dbWriter *database.InfluxDBWriter) *StatsHandler {
	return &StatsHandler{
		dbWriter: dbWriter,
	}
}

// Gin handler for receiving stats from clients
func (h *StatsHandler) PostStats(c *gin.Context) {
	var payload models.ClientPayload

	// 1. Bind JSON payload to the struct
	if err := c.ShouldBindJSON(&payload); err != nil {
		appLogger.Error("Failed to bind JSON payload: %v. Client IP: %s", err, c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload", "details": err.Error()})
		return
	}
	// 2. Basic validation (ensure HostID is present)
	if payload.System.HostID == "" {
		appLogger.Warn("Received payload with empty HostID from %s. Payload Hostname: %s", c.ClientIP(), payload.System.Hostname)
		c.JSON(http.StatusBadRequest, gin.H{"error": "HostID is missing in system_info"})
		return
	}
	if payload.CollectedAt.IsZero() {
		appLogger.Warn("Received payload with zero CollectedAt timestamp from HostID %s", payload.System.HostID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "CollectedAt timestamp is missing or zero"})
		return
	}

	appLogger.Info("Received stats from HostID: %s, Hostname: %s", payload.System.HostID, payload.System.Hostname)
	appLogger.Debug("Payload received: %+v", payload) // Log full payload only in debug mode

	// 3. Write stats to the database
	// The context from Gin (c.Request.Context()) can be used for cancellation propagation
	// if the client disconnects or the request times out.
	if err := h.dbWriter.WriteStats(c.Request.Context(), &payload); err != nil {
		// dbWriter already logs detailed errors
		appLogger.Error("Failed to write stats to database for HostID %s: %v", payload.System.HostID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store statistics"})
		return
	}

	// 4. Respond with success
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Statistics received and processed"})
	appLogger.Info("Successfully processed and stored stats for HostID: %s", payload.System.HostID)

}

// RegisterRoutes registers the API routes for stats handling.
func (h *StatsHandler) RegisterRoutes(router *gin.Engine) {
	apiGroup := router.Group("/api")
	{
		apiGroup.POST("/stats", h.PostStats)
	}
}
