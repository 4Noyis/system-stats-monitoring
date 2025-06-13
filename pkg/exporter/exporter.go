package exporter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt" // Used for potential error wrapping
	"io"

	"net/http"
	"time"

	appLogger "github.com/4Noyis/system-stats-monitoring/internal/logger"
)

// SendStatsJSON marshals the provided data to JSON and sends it via HTTP POST to the specified serverURL.

// The 'data' parameter is an interface{} to allow sending various data structures.
func SendStatsJSON(ctx context.Context, serverURL string, data interface{}) error {
	// 1. Marshal data to JSON
	// Using MarshalIndent for readability during debugging, can switch to Marshal for production.
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		appLogger.Error("Error marshaling stats to JSON: %v", err)
		return fmt.Errorf("error marshaling data to JSON: %w", err)
	}

	// 2. Log for debugging (optional, can be removed or made conditional)
	appLogger.Info("Sending data (size %d bytes) to %s", len(jsonData), serverURL)

	// 3. Create HTTP request with context for timeout and cancellation
	reqCtx, reqCancel := context.WithTimeout(ctx, 15*time.Second) // 15-second timeout for the HTTP request
	defer reqCancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST", serverURL, bytes.NewBuffer(jsonData))
	if err != nil {
		appLogger.Error("Error creating HTTP request: %v", err)
		return fmt.Errorf("error creating HTTP request to %s: %w", serverURL, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 4. Execute the HTTP request
	httpClient := &http.Client{} // default client
	resp, err := httpClient.Do(req)
	if err != nil {
		// Check for context errors (timeout or cancellation)
		if reqCtx.Err() == context.DeadlineExceeded {
			appLogger.Error("HTTP request to %s timed out.", serverURL)
			return fmt.Errorf("http request to %s timed out: %w", serverURL, err)
		} else if ctx.Err() != nil { // Check original context passed to SendStatsJSON
			appLogger.Error("HTTP request to %s cancelled by parent context: %v", serverURL, ctx.Err())
			return fmt.Errorf("http request to %s cancelled by parent context: %w", serverURL, ctx.Err())
		}
		appLogger.Error("Error sending stats to server %s: %v", serverURL, err)
		return fmt.Errorf("error sending stats to server %s: %w", serverURL, err)
	}
	defer resp.Body.Close()

	// 5. Process the response
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		appLogger.Info("Stats sent successfully to %s. Server responded with %s", serverURL, resp.Status)
	} else {
		appLogger.Warn("Server at %s responded with non-OK status: %s", serverURL, resp.Status)
		responseBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			appLogger.Error("Error reading error response body from %s: %v", serverURL, readErr)
			return fmt.Errorf("server at %s responded with %s (and error reading response body: %v)", serverURL, resp.Status, readErr)
		}
		appLogger.Error("Server error response from %s: %s", serverURL, string(responseBody))
		return fmt.Errorf("server at %s responded with %s: %s", serverURL, resp.Status, string(responseBody))
	}

	return nil // Success
}
