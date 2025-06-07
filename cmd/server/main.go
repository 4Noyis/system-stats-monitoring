package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appLogger "github.com/4Noyis/system-stats-monitoring/internal/logger"
	"github.com/4Noyis/system-stats-monitoring/internal/server/api"
	"github.com/4Noyis/system-stats-monitoring/internal/server/config"
	"github.com/4Noyis/system-stats-monitoring/internal/server/database"
	"github.com/gin-gonic/gin"
)

// For incoming statistics data

func main() {
	// -------- load config ---------
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err) // Use fmt here as logger might not be fully up
		os.Exit(1)
	}

	// --------- initialize logger ----------
	if cfg.EnableDebugLog {
		appLogger.SetDebug(true)
		appLogger.Info("Debug logging enabled")
	}
	appLogger.Info("Server configuration loaded.")
	appLogger.Debug("Full configuration: %+v", cfg)

	// --------- initialize influxDB writer ------------
	dbWriter, err := database.NewInfluxDBWriter(cfg.InfluxDB)
	if err != nil {
		appLogger.Fatal("Gailed to initialize InfluxDB writer: %v", err)
	}
	defer dbWriter.Close() // ensure client is closed on exit
	appLogger.Info("InfluxDB writer initialized.")

	// ------- Initialize Gin ------------
	if !cfg.EnableDebugLog {
		gin.SetMode(gin.ReleaseMode)
		appLogger.Info("Gin set to ReleaseMode.")
	} else {
		gin.SetMode(gin.DebugMode)
		appLogger.Info("Gin set to DebugMode.")
	}

	router := gin.New() // Using gin.New() for more control over middleware

	// Middleware
	router.Use(ginLoggerMiddleware()) // Recover from any panics and return a 500
	appLogger.Info("Gin engine initialized.")

	// ------ Setup API Handlers and Routes -------
	statsAPIHandler := api.NewStatsHandler(dbWriter)
	statsAPIHandler.RegisterRoutes(router)
	appLogger.Info("API routes registered.")

	// ------- Start http Server --------
	srv := &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: router,

		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine so that it doesn't block.
	go func() {
		appLogger.Info("Starting server on %s", cfg.ListenAddress)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Fatal("Could not listen on %s: %v\n", cfg.ListenAddress, err)
		}
	}()

	// 7. Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	receivedSignal := <-quit
	appLogger.Info("Shutdown signal (%s) received. Shutting down server gracefully...", receivedSignal)

	// The context is used to inform the server it has 5 seconds to finish
	// the requests it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown: %v", err)
	}

	appLogger.Info("Server exiting.")
}

func ginLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next() // Process request
		latency := time.Since(startTime)

		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		// userAgent := c.Request.UserAgent() // Optional
		// errors := c.Errors.ByType(gin.ErrorTypePrivate).String() // Optional for logging Gin errors

		logFunc := appLogger.Info // Default to Info
		if status >= 400 && status < 500 {
			logFunc = appLogger.Warn
		} else if status >= 500 {
			logFunc = appLogger.Error
		}

		logFunc("GIN | %3d | %13v | %15s | %-7s %s",
			status,
			latency,
			clientIP,
			method,
			path,
		)
		// if errors != "" {
		//  appLogger.Error("GIN ERRORS | %s", errors)
		// }
	}
}
