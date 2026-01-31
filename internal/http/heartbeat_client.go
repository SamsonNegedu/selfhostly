package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/selfhostly/internal/apipaths"
)

// HeartbeatClient manages continuous heartbeat with exponential backoff
type HeartbeatClient struct {
	config          *Config
	client          *http.Client
	stopCh          chan struct{}
	running         bool
	mu              sync.Mutex
	wasDisconnected bool
	failureCount    int
	onReconnect     func(context.Context) error // Callback for reconnection events
}

// Config holds heartbeat configuration
type Config struct {
	PrimaryURL        string
	NodeID            string
	NodeAPIKey        string
	InitialInterval   time.Duration
	MaxInterval       time.Duration
	MaxRetries        int
	HeartbeatInterval time.Duration
	OnReconnect       func(context.Context) error // Callback for reconnection events
}

// NewHeartbeatClient creates a new heartbeat client
func NewHeartbeatClient(config *Config) *HeartbeatClient {
	if config.InitialInterval == 0 {
		config.InitialInterval = 2 * time.Second
	}
	if config.MaxInterval == 0 {
		config.MaxInterval = 5 * time.Minute
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 10
	}
	if config.HeartbeatInterval == 0 {
		config.HeartbeatInterval = 60 * time.Second
	}

	return &HeartbeatClient{
		config:      config,
		client:      &http.Client{Timeout: 10 * time.Second},
		stopCh:      make(chan struct{}),
		onReconnect: config.OnReconnect,
	}
}

// Start begins the heartbeat routine with exponential backoff
func (h *HeartbeatClient) Start(ctx context.Context) {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.mu.Unlock()

	slog.Info("Starting heartbeat client",
		"primary_url", h.config.PrimaryURL,
		"interval", h.config.HeartbeatInterval)

	// Initial delay before first heartbeat
	time.Sleep(2 * time.Second)

	// Trigger initial settings sync on startup
	if h.onReconnect != nil {
		go func() {
			slog.Info("Initial settings sync on startup")
			if err := h.onReconnect(ctx); err != nil {
				slog.Error("Failed initial settings sync", "error", err)
			} else {
				slog.Info("Initial settings synced successfully")
			}
		}()
	}

	go h.runHeartbeatLoop(ctx)
}

// Stop stops the heartbeat routine
func (h *HeartbeatClient) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	h.running = false
	close(h.stopCh)
	slog.Info("Heartbeat client stopped")
}

// runHeartbeatLoop manages the heartbeat loop with exponential backoff
func (h *HeartbeatClient) runHeartbeatLoop(ctx context.Context) {
	backoffInterval := h.config.InitialInterval
	consecutiveFailures := 0

	for {
		select {
		case <-ctx.Done():
			slog.Info("Heartbeat loop shutting down due to context cancellation")
			return
		case <-h.stopCh:
			return
		default:
			// Send heartbeat
			err := h.sendHeartbeat()

			if err != nil {
				consecutiveFailures++
				h.mu.Lock()
				h.wasDisconnected = true
				h.failureCount++
				h.mu.Unlock()

				slog.Warn("Heartbeat failed",
					"consecutive_failures", consecutiveFailures,
					"error", err)

				// Exponential backoff on failure
				if consecutiveFailures > h.config.MaxRetries {
					backoffInterval = h.config.MaxInterval
				} else {
					backoffInterval = h.calculateBackoff(consecutiveFailures)
				}

				slog.Info("Next heartbeat attempt",
					"retry_in", backoffInterval,
					"consecutive_failures", consecutiveFailures)
			} else {
				// Success - reset backoff and check if we need to reconcile
				wasDisconnected := false
				h.mu.Lock()
				wasDisconnected = h.wasDisconnected
				if h.wasDisconnected {
					h.wasDisconnected = false
					h.failureCount = 0
				}
				h.mu.Unlock()

				if consecutiveFailures > 0 {
					slog.Info("Heartbeat connection restored",
						"after_failures", consecutiveFailures)
				}

				consecutiveFailures = 0
				backoffInterval = h.config.HeartbeatInterval

				// If we were disconnected and now reconnected, trigger state reconciliation
				if wasDisconnected {
					slog.Info("Node reconnected after network partition - state reconciliation may be needed")
					// TODO: Trigger state reconciliation here
				}

				slog.Debug("Heartbeat successful", "next_in", backoffInterval)
			}

			// Wait before next attempt
			select {
			case <-ctx.Done():
				return
			case <-h.stopCh:
				return
			case <-time.After(backoffInterval):
				// Continue to next iteration
			}
		}
	}
}

// sendHeartbeat sends a heartbeat to the primary node
func (h *HeartbeatClient) sendHeartbeat() error {
	heartbeatURL := h.config.PrimaryURL + apipaths.NodeHeartbeat(h.config.NodeID)

	req, err := http.NewRequest("POST", heartbeatURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add node authentication headers
	req.Header.Set("X-Node-ID", h.config.NodeID)
	req.Header.Set("X-Node-API-Key", h.config.NodeAPIKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResp ErrorResponse
		if json.NewDecoder(resp.Body).Decode(&errorResp) == nil {
			return fmt.Errorf("heartbeat failed (%d): %s", resp.StatusCode, errorResp.Error)
		}
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	return nil
}

// calculateBackoff calculates exponential backoff interval
func (h *HeartbeatClient) calculateBackoff(failures int) time.Duration {
	// Exponential backoff: initialInterval * 2^failures
	backoff := float64(h.config.InitialInterval) * math.Pow(2, float64(failures))
	interval := time.Duration(backoff)

	// Cap at max interval
	if interval > h.config.MaxInterval {
		interval = h.config.MaxInterval
	}

	return interval
}

// GetStats returns current heartbeat statistics
func (h *HeartbeatClient) GetStats() HeartbeatStats {
	h.mu.Lock()
	defer h.mu.Unlock()

	return HeartbeatStats{
		Running:         h.running,
		WasDisconnected: h.wasDisconnected,
		FailureCount:    h.failureCount,
	}
}

// HeartbeatStats holds statistics about the heartbeat client
type HeartbeatStats struct {
	Running         bool
	WasDisconnected bool
	FailureCount    int
}

// sendPeriodicHeartbeats starts sending periodic heartbeats (for use in Server)
func (s *Server) sendPeriodicHeartbeats() {
	if s.config.Node.IsPrimary {
		return // Primary nodes don't send heartbeats
	}

	if s.config.Node.PrimaryNodeURL == "" {
		slog.Warn("PRIMARY_NODE_URL not configured - skipping heartbeat client")
		return
	}

	config := &Config{
		PrimaryURL:        s.config.Node.PrimaryNodeURL,
		NodeID:            s.config.Node.ID,
		NodeAPIKey:        s.config.Node.APIKey,
		InitialInterval:   2 * time.Second,
		MaxInterval:       5 * time.Minute,
		MaxRetries:        10,
		HeartbeatInterval: 60 * time.Second,
		OnReconnect: func(ctx context.Context) error {
			// Sync settings from primary node
			return s.nodeService.SyncSettingsFromPrimary(ctx)
		},
	}

	heartbeatClient := NewHeartbeatClient(config)
	heartbeatClient.Start(s.shutdownCtx)

	// Store client for potential shutdown
	// (You could add this to Server struct if needed)
}
