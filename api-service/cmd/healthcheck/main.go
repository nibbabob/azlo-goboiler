package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Struct to match the API's "writeResponse" wrapper
type HealthResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Status string `json:"status"`
	} `json:"data"`
}

func main() {
	// Create a client with a short timeout
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	// Create request with context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/health", nil)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		os.Exit(1)
	}

	// Set headers
	req.Header.Set("User-Agent", "healthcheck/1.0")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Health check request failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Health check failed with status: %d\n", resp.StatusCode)
		os.Exit(1)
	}

	// Parse response
	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		fmt.Printf("Failed to parse health response: %v\n", err)
		os.Exit(1)
	}

	// FIX: Check the nested status inside Data
	if health.Data.Status != "healthy" {
		fmt.Printf("Service is not healthy: %s\n", health.Data.Status)
		os.Exit(1)
	}

	// All good
	fmt.Println("Health check passed")
	os.Exit(0)
}
