package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/selfhostly/internal/constants"
)

// Router resolves the target node base URL for a request
type Router struct {
	registry *NodeRegistry
	logger   *slog.Logger
}

// NewRouter creates a router that uses the node registry
func NewRouter(registry *NodeRegistry, logger *slog.Logger) *Router {
	return &Router{
		registry: registry,
		logger:   logger,
	}
}

// Target returns the base URL (e.g. http://primary:8082) for the request and whether it was resolved
func (r *Router) Target(req *http.Request) (baseURL string, ok bool) {
	path := req.URL.Path
	query := req.URL.Query()

	r.logger.Debug("router: resolving target",
		"path", path,
		"method", req.Method,
		"node_id", query.Get("node_id"),
	)

	// Global / list / aggregate -> primary
	if r.isPrimaryOnly(path, req.Method) {
		target := r.registry.PrimaryBaseURL()
		r.logger.Debug("router: primary-only route", "path", path, "target", target)
		return target, true
	}

	// Resource-by-id: need node_id in query
	if r.requiresNodeID(path) {
		nodeID := query.Get("node_id")
		if nodeID == "" {
			r.logger.Warn("router: node_id required but missing", "path", path)
			return "", false
		}
		base := r.registry.Get(nodeID)
		if base == "" {
			// Check if node exists but is offline/unreachable
			if entry := r.registry.GetEntry(nodeID); entry != nil {
				r.logger.Warn("router: node is offline/unreachable",
					"node_id", nodeID,
					"status", entry.Status,
					"path", path)
			} else {
				r.logger.Warn("router: node not found", "node_id", nodeID, "path", path)
			}
			return "", false
		}
		r.logger.Debug("router: resolved by node_id", "node_id", nodeID, "target", base)
		return base, true
	}

	// POST /api/apps: node_id in body
	if req.Method == http.MethodPost && path == "/api/apps" {
		nodeID, err := r.nodeIDFromCreateAppBody(req)
		if err != nil || nodeID == "" {
			// Default to primary if no node_id in body
			return r.registry.PrimaryBaseURL(), true
		}
		base := r.registry.Get(nodeID)
		if base == "" {
			// If target node is offline, fallback to primary
			if entry := r.registry.GetEntry(nodeID); entry != nil && (entry.Status == constants.NodeStatusOffline || entry.Status == constants.NodeStatusUnreachable) {
				r.logger.Warn("router: target node is offline, falling back to primary",
					"node_id", nodeID,
					"status", entry.Status)
			}
			return r.registry.PrimaryBaseURL(), true
		}
		return base, true
	}

	return r.registry.PrimaryBaseURL(), true
}

func (r *Router) isPrimaryOnly(path, method string) bool {
	switch {
	case strings.HasPrefix(path, "/auth/"):
		return true
	case strings.HasPrefix(path, "/avatar/"):
		return true
	// Note: /api/health is handled directly by the proxy, not routed
	case path == "/api/health":
		return true
	case path == "/api/nodes" || strings.HasPrefix(path, "/api/nodes/"):
		return true
	case path == "/api/settings":
		return true
	case path == "/api/me":
		return true
	case path == "/api/node/info":
		return true
	case method == http.MethodGet && path == "/api/apps":
		return true
	case method == http.MethodGet && path == "/api/tunnels":
		return true
	case strings.HasPrefix(path, "/api/tunnels/providers"):
		return true
	case method == http.MethodGet && path == "/api/system/stats":
		return true
	default:
		return false
	}
}

func (r *Router) requiresNodeID(path string) bool {
	// /api/apps/:id and below (except GET /api/apps which is list)
	if strings.HasPrefix(path, "/api/apps/") {
		rest := strings.TrimPrefix(path, "/api/apps/")
		return rest != ""
	}
	if strings.HasPrefix(path, "/api/tunnels/apps/") {
		return true
	}
	if strings.HasPrefix(path, "/api/system/containers/") {
		return true
	}
	// Job endpoints require node_id to route to the correct node's local DB
	if strings.HasPrefix(path, "/api/jobs/") {
		return true
	}
	return false
}

// createAppBody is a minimal struct to read node_id from POST /api/apps
type createAppBody struct {
	NodeID string `json:"node_id"`
}

func (r *Router) nodeIDFromCreateAppBody(req *http.Request) (string, error) {
	if req.Body == nil {
		return "", nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	var c createAppBody
	if err := json.Unmarshal(body, &c); err != nil {
		return "", err
	}
	return strings.TrimSpace(c.NodeID), nil
}
