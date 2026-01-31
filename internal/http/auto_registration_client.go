package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/selfhostly/internal/apipaths"
)

// attemptAutoRegistration tries to auto-register this secondary node with the primary
// Includes retry logic with exponential backoff
func (s *Server) attemptAutoRegistration() {
	// Wait a bit for server to fully initialize
	time.Sleep(2 * time.Second)

	// Check if we have a registration token
	if s.config.Node.RegistrationToken == "" {
		slog.Info("skipping auto-registration - no REGISTRATION_TOKEN configured")
		slog.Info("INFO: you can still register manually through the primary UI")
		return
	}

	slog.Info("attempting auto-registration with primary", "primary_url", s.config.Node.PrimaryNodeURL)

	// Retry configuration
	maxRetries := 5
	retryDelay := 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		slog.Info("auto-registration attempt", "attempt", attempt, "max", maxRetries)

		err := s.registerWithPrimary()
		if err == nil {
			slog.Info("auto-registration successful")
			// After successful registration, start sending heartbeats
			go s.sendStartupHeartbeat()
			return
		}

		slog.Warn("auto-registration failed", "attempt", attempt, "error", err)

		if attempt < maxRetries {
			slog.Info("retrying auto-registration", "retry_in", retryDelay)
			time.Sleep(retryDelay)
			// Exponential backoff: 5s, 10s, 20s, 40s
			retryDelay *= 2
		}
	}

	slog.Error("ERROR: auto-registration failed after all retries", "max_retries", maxRetries)
	slog.Info("INFO: you can manually register this node through the primary UI at: " + s.config.Node.PrimaryNodeURL + "/nodes")
}

// registerWithPrimary sends the auto-registration request to the primary node
func (s *Server) registerWithPrimary() error {
	primaryURL := s.config.Node.PrimaryNodeURL
	registerURL := primaryURL + apipaths.NodeRegister

	// Prepare registration request
	registrationReq := AutoRegisterRequest{
		ID:          s.config.Node.ID,
		Name:        s.config.Node.Name,
		APIEndpoint: s.config.Node.APIEndpoint,
		APIKey:      s.config.Node.APIKey,
		Token:       s.config.Node.RegistrationToken,
	}

	jsonData, err := json.Marshal(registrationReq)
	if err != nil {
		return fmt.Errorf("failed to marshal registration request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", registerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// Add node authentication headers
	req.Header.Set("X-Node-ID", s.config.Node.ID)
	req.Header.Set("X-Node-API-Key", s.config.Node.APIKey)

	// Send request
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
			slog.Info("auto-registration response", "message", result["message"], "status", result["status"])
		}
		return nil
	}

	// Handle error response
	var errorResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err == nil {
		return fmt.Errorf("registration failed (%d): %s - %s", resp.StatusCode, errorResp.Error, errorResp.Details)
	}

	return fmt.Errorf("registration failed with status code: %d", resp.StatusCode)
}
