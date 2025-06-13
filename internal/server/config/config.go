package config

import (
	"os"
	"strconv"

	appLogger "github.com/4Noyis/system-stats-monitoring/internal/logger"
)

// For parsing numeric env vars if needed

// Assuming shared logger

// holds the configuration for connecting to InfluxDB
type InfluxDBConfig struct {
	URL    string
	Token  string
	Org    string
	Bucket string
}

// holds overall server config
type ServerConfig struct {
	ListenAddress  string
	InfluxDB       InfluxDBConfig
	EnableDebugLog bool
}

// Load loads configuration from environment variables.
func Load() (*ServerConfig, error) {
	cfg := &ServerConfig{
		ListenAddress: getEnv("SERVER_LISTEN_ADDRESS", ":8080"), //default port

		InfluxDB: InfluxDBConfig{
			URL:    getEnv("INFLUXDB_URL", "http://localhost:8086"),
			Token:  getEnv("INFLUXDB_TOKEN", "API-KEY"),      // Add API Key
			Org:    getEnv("INFLUXDB_ORG", "ORG-NAME"),       // Add organization name                                                                                   //
			Bucket: getEnv("INFLUXDB_BUCKET", "BUCKET-NAME"), // Add bucket                                                                            //
		},
		EnableDebugLog: getEnvAsBool("SERVER_ENABLE_DEBUG_LOG", false),
	}
	// Validate essential InfluxDB settings
	if cfg.InfluxDB.Token == "" {
		appLogger.Error("INFLUXDB_TOKEN environment variable is not set.")
	}
	if cfg.InfluxDB.Org == "" {
		appLogger.Error("INFLUXDB_ORG environment variable is not set.")
	}
	if cfg.InfluxDB.Bucket == "" {
		appLogger.Error("INFLUXDB_BUCKET environment variable is not set.")

	}

	return cfg, nil
}

// get an environment variable or return a default value.
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// Helper function to get an environment variable as a boolean.
func getEnvAsBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
		appLogger.Warn("Failed to parse env var %s as bool: %v. Using fallback: %t", key, err, fallback)
	}
	return fallback
}
