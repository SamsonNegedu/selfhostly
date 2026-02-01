package gateway

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/selfhostly/internal/constants"
)

func TestNodeRegistry_Refresh(t *testing.T) {
	tests := []struct {
		name           string
		nodes          []NodeEntry
		statusCode     int
		wantErr        bool
		wantNodeCount  int
		wantPrimaryID  string
		gatewayAPIKey  string
	}{
		{
			name: "successful refresh with multiple nodes",
			nodes: []NodeEntry{
				{ID: "primary-1", APIEndpoint: "http://primary:8082", IsPrimary: true, Status: constants.NodeStatusOnline},
				{ID: "secondary-1", APIEndpoint: "http://secondary-1:8083", IsPrimary: false, Status: constants.NodeStatusOnline},
				{ID: "secondary-2", APIEndpoint: "http://secondary-2:8084", IsPrimary: false, Status: constants.NodeStatusOnline},
			},
			statusCode:    http.StatusOK,
			wantErr:       false,
			wantNodeCount: 3,
			wantPrimaryID: "primary-1",
			gatewayAPIKey: "test-api-key",
		},
		{
			name: "refresh with offline nodes",
			nodes: []NodeEntry{
				{ID: "primary-1", APIEndpoint: "http://primary:8082", IsPrimary: true, Status: constants.NodeStatusOnline},
				{ID: "secondary-1", APIEndpoint: "http://secondary-1:8083", IsPrimary: false, Status: constants.NodeStatusOffline},
				{ID: "secondary-2", APIEndpoint: "http://secondary-2:8084", IsPrimary: false, Status: constants.NodeStatusUnreachable},
			},
			statusCode:    http.StatusOK,
			wantErr:       false,
			wantNodeCount: 3,
			wantPrimaryID: "primary-1",
			gatewayAPIKey: "test-api-key",
		},
		{
			name: "no primary marked, uses first node",
			nodes: []NodeEntry{
				{ID: "node-1", APIEndpoint: "http://node-1:8082", IsPrimary: false, Status: constants.NodeStatusOnline},
				{ID: "node-2", APIEndpoint: "http://node-2:8083", IsPrimary: false, Status: constants.NodeStatusOnline},
			},
			statusCode:    http.StatusOK,
			wantErr:       false,
			wantNodeCount: 2,
			wantPrimaryID: "node-1",
			gatewayAPIKey: "test-api-key",
		},
		{
			name:          "HTTP error",
			statusCode:    http.StatusInternalServerError,
			wantErr:       true,
			wantNodeCount: 0,
			gatewayAPIKey: "test-api-key",
		},
		{
			name:          "invalid JSON",
			statusCode:    http.StatusOK,
			wantErr:       true,
			wantNodeCount: 0,
			gatewayAPIKey: "test-api-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify API key header
				if r.Header.Get("X-Gateway-API-Key") != tt.gatewayAPIKey {
					t.Errorf("expected API key %q, got %q", tt.gatewayAPIKey, r.Header.Get("X-Gateway-API-Key"))
				}

				if tt.statusCode != http.StatusOK {
					w.WriteHeader(tt.statusCode)
					return
				}

				// For invalid JSON test
				if tt.name == "invalid JSON" {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("invalid json"))
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(tt.nodes)
			}))
			defer server.Close()

			// Create registry
			logger := slog.Default()
			registry := NewNodeRegistry(server.URL, tt.gatewayAPIKey, 60*time.Second, logger)

			// Refresh
			err := registry.refresh()

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("refresh() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Check node count
			registry.mu.RLock()
			nodeCount := len(registry.nodes)
			primaryID := registry.primary
			registry.mu.RUnlock()

			if nodeCount != tt.wantNodeCount {
				t.Errorf("expected %d nodes, got %d", tt.wantNodeCount, nodeCount)
			}

			if primaryID != tt.wantPrimaryID {
				t.Errorf("expected primary ID %q, got %q", tt.wantPrimaryID, primaryID)
			}
		})
	}
}

func TestNodeRegistry_Get(t *testing.T) {
	logger := slog.Default()
	registry := NewNodeRegistry("http://primary:8082", "test-api-key", 60*time.Second, logger)

	// Manually populate nodes for testing
	registry.mu.Lock()
	registry.nodes = map[string]NodeEntry{
		"online-node": {
			ID:          "online-node",
			APIEndpoint: "http://online:8082",
			Status:      constants.NodeStatusOnline,
		},
		"offline-node": {
			ID:          "offline-node",
			APIEndpoint: "http://offline:8082",
			Status:      constants.NodeStatusOffline,
		},
		"unreachable-node": {
			ID:          "unreachable-node",
			APIEndpoint: "http://unreachable:8082",
			Status:      constants.NodeStatusUnreachable,
		},
	}
	registry.mu.Unlock()

	tests := []struct {
		name     string
		nodeID   string
		want     string
		wantEmpty bool
	}{
		{
			name:      "online node returns endpoint",
			nodeID:    "online-node",
			want:      "http://online:8082",
			wantEmpty: false,
		},
		{
			name:      "offline node returns empty",
			nodeID:    "offline-node",
			want:      "",
			wantEmpty: true,
		},
		{
			name:      "unreachable node returns empty",
			nodeID:    "unreachable-node",
			want:      "",
			wantEmpty: true,
		},
		{
			name:      "non-existent node returns empty",
			nodeID:    "non-existent",
			want:      "",
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := registry.Get(tt.nodeID)
			if (got == "") != tt.wantEmpty {
				t.Errorf("Get() = %q, wantEmpty %v", got, tt.wantEmpty)
			}
			if !tt.wantEmpty && got != tt.want {
				t.Errorf("Get() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNodeRegistry_GetEntry(t *testing.T) {
	logger := slog.Default()
	registry := NewNodeRegistry("http://primary:8082", "test-api-key", 60*time.Second, logger)

	// Manually populate nodes for testing
	registry.mu.Lock()
	registry.nodes = map[string]NodeEntry{
		"test-node": {
			ID:          "test-node",
			APIEndpoint: "http://test:8082",
			Status:      constants.NodeStatusOnline,
			IsPrimary:   true,
		},
	}
	registry.mu.Unlock()

	tests := []struct {
		name   string
		nodeID string
		want   *NodeEntry
	}{
		{
			name:   "existing node returns entry",
			nodeID: "test-node",
			want: &NodeEntry{
				ID:          "test-node",
				APIEndpoint: "http://test:8082",
				Status:      constants.NodeStatusOnline,
				IsPrimary:   true,
			},
		},
		{
			name:   "non-existent node returns nil",
			nodeID: "non-existent",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := registry.GetEntry(tt.nodeID)
			if tt.want == nil {
				if got != nil {
					t.Errorf("GetEntry() = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Errorf("GetEntry() = nil, want %v", tt.want)
				return
			}
			if got.ID != tt.want.ID || got.APIEndpoint != tt.want.APIEndpoint ||
				got.Status != tt.want.Status || got.IsPrimary != tt.want.IsPrimary {
				t.Errorf("GetEntry() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestNodeRegistry_PrimaryID(t *testing.T) {
	logger := slog.Default()
	registry := NewNodeRegistry("http://primary:8082", "test-api-key", 60*time.Second, logger)

	registry.mu.Lock()
	registry.primary = "primary-node-id"
	registry.mu.Unlock()

	if got := registry.PrimaryID(); got != "primary-node-id" {
		t.Errorf("PrimaryID() = %q, want %q", got, "primary-node-id")
	}
}

func TestNodeRegistry_PrimaryBaseURL(t *testing.T) {
	primaryURL := "http://primary:8082"
	logger := slog.Default()
	registry := NewNodeRegistry(primaryURL, "test-api-key", 60*time.Second, logger)

	if got := registry.PrimaryBaseURL(); got != primaryURL {
		t.Errorf("PrimaryBaseURL() = %q, want %q", got, primaryURL)
	}
}
