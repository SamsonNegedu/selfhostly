package gateway

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
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

// NodeRegistry caches node list from primary and refreshes periodically
type NodeRegistry struct {
	primaryBackendURL string
	gatewayAPIKey     string
	httpClient        *http.Client
	logger            *slog.Logger
	ttl               time.Duration

	mu          sync.RWMutex
	nodes       map[string]NodeEntry // nodeID -> NodeEntry (includes endpoint and status)
	primary     string               // primary node ID for "global" routes
	initialized bool                 // true after first successful refresh

	// The cached list can be a whole TTL old, so a node that just joined or reconnected would be
	// unroutable for up to a minute. When a lookup fails, the list is re-read at once, at most every
	// minOnDemandRefresh so a stream of bad requests cannot hammer the primary. Enabled by Start.
	refreshOnMiss bool
	refreshMu     sync.Mutex
	lastOnDemand  time.Time
}

const minOnDemandRefresh = 3 * time.Second

// NewNodeRegistry creates a registry that fetches from primary
func NewNodeRegistry(primaryBackendURL, gatewayAPIKey string, ttl time.Duration, logger *slog.Logger) *NodeRegistry {
	return &NodeRegistry{
		primaryBackendURL: primaryBackendURL,
		gatewayAPIKey:     gatewayAPIKey,
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
	r.mu.Lock()
	r.refreshOnMiss = true
	r.mu.Unlock()
	// Do initial refresh in background to not block gateway startup
	// Gateway can serve health checks immediately while registry initializes
	go func() {
		if err := r.refresh(); err != nil {
			r.logger.Warn("initial node registry refresh failed", "error", err)
		}
	}()

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
	r.initialized = true // Mark as initialized after first successful refresh
	r.logger.Info("node registry refreshed",
		"count", len(r.nodes),
		"primary", r.primary,
	)
	return nil
}

// Get returns the API endpoint for the node, or empty if not found or offline/unreachable
func (r *NodeRegistry) Get(nodeID string) string {
	if url, ok := r.lookup(nodeID); ok {
		return url
	}
	// Unknown or not usable per the cached list: it may simply be stale (a node that just joined or
	// came back), so look again once with fresh data
	if r.refreshNow() {
		if url, ok := r.lookup(nodeID); ok {
			return url
		}
	}
	return ""
}

// refreshNow re-reads the node list if on-demand refresh is enabled and not done too recently
func (r *NodeRegistry) refreshNow() bool {
	r.mu.RLock()
	enabled := r.refreshOnMiss
	r.mu.RUnlock()
	if !enabled {
		return false
	}
	r.refreshMu.Lock()
	if time.Since(r.lastOnDemand) < minOnDemandRefresh {
		r.refreshMu.Unlock()
		return false
	}
	r.lastOnDemand = time.Now()
	r.refreshMu.Unlock()
	return r.refresh() == nil
}

// lookup reports the endpoint to route to and whether the node is usable right now
func (r *NodeRegistry) lookup(nodeID string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.nodes[nodeID]
	if !ok {
		return "", false
	}
	// Don't route to offline or unreachable nodes
	if entry.Status == constants.NodeStatusOffline || entry.Status == constants.NodeStatusUnreachable {
		r.logger.Debug("node registry: skipping offline/unreachable node",
			"node_id", nodeID,
			"status", entry.Status,
		)
		return "", false
	}
	// A node that dialled out to the primary has no address the gateway can reach: its requests go to
	// the primary, which forwards them down the node's link.
	if strings.HasPrefix(entry.APIEndpoint, constants.LinkEndpointScheme+"://") {
		return r.primaryBackendURL, true
	}
	return entry.APIEndpoint, true
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

// IsReady returns true if the registry has been initialized with at least one successful refresh.
// This indicates the gateway can actually route requests to the primary backend.
func (r *NodeRegistry) IsReady() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.initialized
}

type errStatusCode int

func (e errStatusCode) Error() string {
	return fmt.Sprintf("primary returned status %d", int(e))
}
