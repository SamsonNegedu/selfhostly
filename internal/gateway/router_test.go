package gateway

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/selfhostly/internal/constants"
)

func setupTestRouter(t *testing.T) (*Router, *NodeRegistry) {
	logger := slog.Default()
	registry := NewNodeRegistry("http://primary:8082", "test-api-key", 60*time.Second, logger)

	// Populate registry with test nodes
	registry.mu.Lock()
	registry.nodes = map[string]NodeEntry{
		"primary-node": {
			ID:          "primary-node",
			APIEndpoint: "http://primary:8082",
			IsPrimary:   true,
			Status:      constants.NodeStatusOnline,
		},
		"online-node": {
			ID:          "online-node",
			APIEndpoint: "http://online:8083",
			IsPrimary:   false,
			Status:      constants.NodeStatusOnline,
		},
		"offline-node": {
			ID:          "offline-node",
			APIEndpoint: "http://offline:8084",
			IsPrimary:   false,
			Status:      constants.NodeStatusOffline,
		},
		"unreachable-node": {
			ID:          "unreachable-node",
			APIEndpoint: "http://unreachable:8085",
			IsPrimary:   false,
			Status:      constants.NodeStatusUnreachable,
		},
	}
	registry.primary = "primary-node"
	registry.initialized = true
	registry.mu.Unlock()

	router := NewRouter(registry, logger)
	return router, registry
}

func TestRouter_Target_PrimaryOnlyRoutes(t *testing.T) {
	router, _ := setupTestRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
		want   string
		wantOK bool
	}{
		{
			name:   "auth route",
			method: http.MethodGet,
			path:   "/auth/github/login",
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name:   "avatar route",
			method: http.MethodGet,
			path:   "/avatar/user123",
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name:   "health check",
			method: http.MethodGet,
			path:   "/api/health",
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name:   "list nodes",
			method: http.MethodGet,
			path:   "/api/nodes",
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name:   "get node",
			method: http.MethodGet,
			path:   "/api/nodes/123",
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name:   "settings",
			method: http.MethodGet,
			path:   "/api/settings",
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name:   "me endpoint",
			method: http.MethodGet,
			path:   "/api/me",
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name:   "list apps",
			method: http.MethodGet,
			path:   "/api/apps",
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name:   "list tunnels",
			method: http.MethodGet,
			path:   "/api/tunnels",
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name:   "tunnel providers",
			method: http.MethodGet,
			path:   "/api/tunnels/providers",
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name:   "system stats",
			method: http.MethodGet,
			path:   "/api/system/stats",
			want:   "http://primary:8082",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			got, ok := router.Target(req)
			if ok != tt.wantOK {
				t.Errorf("Target() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("Target() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRouter_Target_ResourceByID(t *testing.T) {
	router, _ := setupTestRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
		nodeID string
		want   string
		wantOK bool
	}{
		{
			name:   "online node with node_id",
			method: http.MethodGet,
			path:   "/api/apps/app-123",
			nodeID: "online-node",
			want:   "http://online:8083",
			wantOK: true,
		},
		{
			name:   "offline node returns empty",
			method: http.MethodGet,
			path:   "/api/apps/app-123",
			nodeID: "offline-node",
			want:   "",
			wantOK: false,
		},
		{
			name:   "unreachable node returns empty",
			method: http.MethodGet,
			path:   "/api/apps/app-123",
			nodeID: "unreachable-node",
			want:   "",
			wantOK: false,
		},
		{
			name:   "missing node_id",
			method: http.MethodGet,
			path:   "/api/apps/app-123",
			nodeID: "",
			want:   "",
			wantOK: false,
		},
		{
			name:   "non-existent node_id",
			method: http.MethodGet,
			path:   "/api/apps/app-123",
			nodeID: "non-existent",
			want:   "",
			wantOK: false,
		},
		{
			name:   "tunnel route with node_id",
			method: http.MethodGet,
			path:   "/api/tunnels/apps/app-123",
			nodeID: "online-node",
			want:   "http://online:8083",
			wantOK: true,
		},
		{
			name:   "container route with node_id",
			method: http.MethodGet,
			path:   "/api/system/containers/container-123",
			nodeID: "online-node",
			want:   "http://online:8083",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := tt.path
			if tt.nodeID != "" {
				url += "?node_id=" + tt.nodeID
			}
			req := httptest.NewRequest(tt.method, url, nil)
			got, ok := router.Target(req)
			if ok != tt.wantOK {
				t.Errorf("Target() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("Target() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRouter_Target_CreateApp(t *testing.T) {
	router, _ := setupTestRouter(t)

	tests := []struct {
		name   string
		body   map[string]interface{}
		want   string
		wantOK bool
	}{
		{
			name: "create app with online node_id",
			body: map[string]interface{}{
				"name":    "test-app",
				"node_id": "online-node",
			},
			want:   "http://online:8083",
			wantOK: true,
		},
		{
			name: "create app with offline node_id falls back to primary",
			body: map[string]interface{}{
				"name":    "test-app",
				"node_id": "offline-node",
			},
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name: "create app with unreachable node_id falls back to primary",
			body: map[string]interface{}{
				"name":    "test-app",
				"node_id": "unreachable-node",
			},
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name: "create app without node_id defaults to primary",
			body: map[string]interface{}{
				"name": "test-app",
			},
			want:   "http://primary:8082",
			wantOK: true,
		},
		{
			name: "create app with non-existent node_id defaults to primary",
			body: map[string]interface{}{
				"name":    "test-app",
				"node_id": "non-existent",
			},
			want:   "http://primary:8082",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/apps", bytes.NewReader(bodyBytes))
			got, ok := router.Target(req)
			if ok != tt.wantOK {
				t.Errorf("Target() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("Target() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRouter_isPrimaryOnly(t *testing.T) {
	router, _ := setupTestRouter(t)

	tests := []struct {
		name   string
		path   string
		method string
		want   bool
	}{
		{"auth prefix", "/auth/login", http.MethodGet, true},
		{"avatar prefix", "/avatar/user", http.MethodGet, true},
		{"health", "/api/health", http.MethodGet, true},
		{"nodes list", "/api/nodes", http.MethodGet, true},
		{"node detail", "/api/nodes/123", http.MethodGet, true},
		{"settings", "/api/settings", http.MethodGet, true},
		{"me", "/api/me", http.MethodGet, true},
		{"node info", "/api/node/info", http.MethodGet, true},
		{"apps list GET", "/api/apps", http.MethodGet, true},
		{"apps list POST", "/api/apps", http.MethodPost, false},
		{"tunnels list GET", "/api/tunnels", http.MethodGet, true},
		{"tunnels list POST", "/api/tunnels", http.MethodPost, false},
		{"tunnel providers", "/api/tunnels/providers", http.MethodGet, true},
		{"system stats GET", "/api/system/stats", http.MethodGet, true},
		{"system stats POST", "/api/system/stats", http.MethodPost, false},
		{"app detail", "/api/apps/123", http.MethodGet, false},
		{"other path", "/api/other", http.MethodGet, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.isPrimaryOnly(tt.path, tt.method)
			if got != tt.want {
				t.Errorf("isPrimaryOnly(%q, %q) = %v, want %v", tt.path, tt.method, got, tt.want)
			}
		})
	}
}

func TestRouter_requiresNodeID(t *testing.T) {
	router, _ := setupTestRouter(t)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"app detail", "/api/apps/123", true},
		{"app detail with subpath", "/api/apps/123/logs", true},
		{"app list", "/api/apps", false},
		{"tunnel app", "/api/tunnels/apps/123", true},
		{"container", "/api/system/containers/123", true},
		{"other path", "/api/other", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := router.requiresNodeID(tt.path)
			if got != tt.want {
				t.Errorf("requiresNodeID(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRouter_nodeIDFromCreateAppBody(t *testing.T) {
	router, _ := setupTestRouter(t)

	tests := []struct {
		name    string
		body    map[string]interface{}
		want    string
		wantErr bool
	}{
		{
			name: "valid body with node_id",
			body: map[string]interface{}{
				"name":    "test-app",
				"node_id": "online-node",
			},
			want:    "online-node",
			wantErr: false,
		},
		{
			name: "body without node_id",
			body: map[string]interface{}{
				"name": "test-app",
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "body with empty node_id",
			body: map[string]interface{}{
				"name":    "test-app",
				"node_id": "",
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "body with whitespace node_id",
			body: map[string]interface{}{
				"name":    "test-app",
				"node_id": "  online-node  ",
			},
			want:    "online-node",
			wantErr: false,
		},
		{
			name:    "nil body",
			body:    nil,
			want:    "",
			wantErr: true, // Reading nil body returns error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/apps", bytes.NewReader(bodyBytes))
			got, err := router.nodeIDFromCreateAppBody(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("nodeIDFromCreateAppBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("nodeIDFromCreateAppBody() = %q, want %q", got, tt.want)
			}
		})
	}
}
