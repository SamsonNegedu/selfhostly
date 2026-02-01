package gateway

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/selfhostly/internal/constants"
)

// NodeEntry is a node known to the gateway (id, endpoint, is_primary, status)
type NodeEntry struct {
	ID          string `json:"id"`
	APIEndpoint string `json:"api_endpoint"`
	IsPrimary   bool   `json:"is_primary"`
	Status      string `json:"status"`
}

// 	NodeRegistry caches node list from primary and refreshes periodically
type NodeRegistry struct {
	primaryBackendURL string
	gatewayAPIKey     string
	httpClient        *http.Client
	logger            *slog.Logger
	ttl               time.Duration

	mu      sync.RWMutex
	nodes   map[string]NodeEntry // nodeID -> NodeEntry (includes endpoint and status)
	primary string               // primary node ID for "global" routes
}

// NewNodeRegistry creates a registry that fetches from primary
func NewNodeRegistry(primaryBackendURL, gatewayAPIKey string, ttl time.Duration, logger *slog.Logger) *NodeRegistry {
	return &NodeRegistry{
		primaryBackendURL: primaryBackendURL,
		gatewayAPIKey: gatewayAPIKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger: logger,
		ttl:    ttl,
		nodes:  make(map[string]NodeEntry),
	}
}

// Start begins periodic refresh; call once after creation
func (r *NodeRegistry) Start() {
	if err := r.refresh(); err != nil {
		r.logger.Warn("initial node registry refresh failed", "error", err)
	}
	go func() {
		ticker := time.NewTicker(r.ttl)
		defer ticker.Stop()
		for range ticker.C {
			if err := r.refresh(); err != nil {
				r.logger.Warn("node registry refresh failed", "error", err)
			}
		}
	}()
}

func (r *NodeRegistry) refresh() error {
	r.logger.Debug("node registry: refreshing", "primary_backend_url", r.primaryBackendURL)

	req, err := http.NewRequest(http.MethodGet, r.primaryBackendURL+"/api/nodes", nil)
	if err != nil {
		r.logger.Error("node registry: failed to create request", "error", err)
		return err
	}
	req.Header.Set("X-Gateway-API-Key", r.gatewayAPIKey)
	resp, err := r.httpClient.Do(req)
	if err != nil {
		r.logger.Error("node registry: request failed", "error", err, "primary_backend_url", r.primaryBackendURL)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		r.logger.Warn("node registry: unexpected status", "status", resp.StatusCode)
		return errStatusCode(resp.StatusCode)
	}
	var list []NodeEntry
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		r.logger.Error("node registry: failed to decode response", "error", err)
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodes = make(map[string]NodeEntry)
	for _, n := range list {
		r.nodes[n.ID] = n
		if n.IsPrimary {
			r.primary = n.ID
		}
		r.logger.Debug("node registry: registered node",
			"id", n.ID,
			"endpoint", n.APIEndpoint,
			"is_primary", n.IsPrimary,
			"status", n.Status,
		)
	}
	if r.primary == "" && len(list) > 0 {
		r.primary = list[0].ID
	}
	r.logger.Info("node registry refreshed",
		"count", len(r.nodes),
		"primary", r.primary,
	)
	return nil
}

// Get returns the API endpoint for the node, or empty if not found or offline/unreachable
func (r *NodeRegistry) Get(nodeID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.nodes[nodeID]
	if !ok {
		return ""
	}
	// Don't route to offline or unreachable nodes
	if entry.Status == constants.NodeStatusOffline || entry.Status == constants.NodeStatusUnreachable {
		r.logger.Debug("node registry: skipping offline/unreachable node",
			"node_id", nodeID,
			"status", entry.Status,
		)
		return ""
	}
	return entry.APIEndpoint
}

// GetEntry returns the full node entry, or nil if not found
func (r *NodeRegistry) GetEntry(nodeID string) *NodeEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.nodes[nodeID]
	if !ok {
		return nil
	}
	return &entry
}

// PrimaryID returns the primary node ID
func (r *NodeRegistry) PrimaryID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.primary
}

// PrimaryBaseURL returns the primary node base URL (for forwarding).
// Uses the configured PRIMARY_BACKEND_URL so the gateway always forwards
// primary traffic to the same URL it uses to fetch /api/nodes, avoiding mismatches
// when the primary's DB has a different self-reported api_endpoint (e.g. from an old seed).
func (r *NodeRegistry) PrimaryBaseURL() string {
	return r.primaryBackendURL
}

type errStatusCode int

func (e errStatusCode) Error() string {
	return fmt.Sprintf("primary returned status %d", int(e))
}
