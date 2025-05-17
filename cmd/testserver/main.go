package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// For incoming statistics data
func statsReceiveHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	// Check if it's a POST rquest
	if r.Method != http.MethodPost {
		log.Printf("Invalid method: %s, expected POST", r.Method)
		http.Error(w, "Only POST method is accepted", http.StatusMethodNotAllowed)
		return
	}

	// Check content-type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Printf("Invalid Content-Type: %s, expected application/json", contentType)
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// 4. Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// log raw body
	log.Printf("Raw JSON received:\n%s\n", string(body))

	var prettyJSON interface{} // use interface{} to accept any valit JSON structure
	err = json.Unmarshal(body, &prettyJSON)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		log.Printf("Problematic raw JSON data:\n%s\n", string(body)) // Log the raw data if unmarshal fails
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	prettyPrintedJSON, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		log.Printf("Error re-marshaling JSON for pretty printing: %v", err)
		// Fallback to logging raw body if pretty printing fails for some reason
		log.Printf("Raw JSON received (pretty print failed):\n%s\n", string(body))
	} else {
		log.Printf("Formatted JSON received:\n%s\n", string(prettyPrintedJSON))
	}

	// 7. Respond with success
	w.WriteHeader(http.StatusOK)                       // 200 OK
	w.Header().Set("Content-Type", "application/json") // Set response content type
	responseMessage := map[string]string{"status": "success", "message": "Data received"}
	json.NewEncoder(w).Encode(responseMessage) // Send a simple JSON response

	log.Println("Successfully processed stats and sent OK response.")
	fmt.Println("-----------------------------------------------------")
}

func main() {
	port := "8080"
	apiPath := "/api/stats"

	// Register the handler function for the "/api/stats" path
	http.HandleFunc(apiPath, statsReceiveHandler)

	log.Printf("Starting basic test server on port %s...", port)
	log.Printf("Listening for POST requests on http://localhost:%s%s", port, apiPath)

	// Configure the server
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second, // Max time to read the entire request
		WriteTimeout: 10 * time.Second, // Max time to write the response
		IdleTimeout:  15 * time.Second, // Max time for connections to remain idle
	}

	// Start the HTTP server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", port, err)
	}
	log.Println("Server stopped.")
}
