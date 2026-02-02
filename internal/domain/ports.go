package domain

import (
	"context"
	"time"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/system"
	"github.com/selfhostly/internal/tunnel"
)

// ============================================================================
// Primary Ports (Application Use Cases)
// ============================================================================

// AppService defines the primary port for application management use cases
type AppService interface {
	CreateApp(ctx context.Context, req CreateAppRequest) (*db.App, error)
	GetApp(ctx context.Context, appID string, nodeID string) (*db.App, error)
	ListApps(ctx context.Context, nodeIDs []string) ([]*db.App, error)
	UpdateApp(ctx context.Context, appID string, nodeID string, req UpdateAppRequest) (*db.App, error)
	DeleteApp(ctx context.Context, appID string, nodeID string) error
	StartApp(ctx context.Context, appID string, nodeID string) (*db.App, error)
	StopApp(ctx context.Context, appID string, nodeID string) (*db.App, error)
	UpdateAppContainers(ctx context.Context, appID string, nodeID string) (*db.App, error)
	RestartCloudflared(ctx context.Context, appID string, nodeID string) error

	// Async job-based operations (return job instead of waiting for completion)
	UpdateAppContainersAsync(ctx context.Context, appID string) (*db.Job, error)
	CreateAppAsync(ctx context.Context, req CreateAppRequest) (*db.Job, error)
	CreateTunnelForAppAsync(ctx context.Context, appID string, ingressRules []db.IngressRule) (*db.Job, error)
	CreateQuickTunnelForAppAsync(ctx context.Context, appID string, service string, port int) (*db.Job, error)
	SwitchAppToCustomTunnelAsync(ctx context.Context, appID string, ingressRules []db.IngressRule) (*db.Job, error)
	DeleteTunnelAsync(ctx context.Context, appID string) (*db.Job, error)

	// CreateTunnelForApp creates a named (custom domain) tunnel for an app that has none. When nodeID is remote, the request is forwarded to that node (all-or-nothing).
	// Returns (app, handledLocally, error). handledLocally is true when the work was done on this node so the HTTP layer may apply optional ingress_rules.
	CreateTunnelForApp(ctx context.Context, appID string, nodeID string, body interface{}) (*db.App, bool, error)
	// SwitchAppToCustomTunnel switches an app from Quick Tunnel to a named (custom domain) tunnel.
	// When nodeID is a remote node, the request is forwarded to that node (all-or-nothing). body is optional (ingress_rules).
	SwitchAppToCustomTunnel(ctx context.Context, appID string, nodeID string, body interface{}) (*db.App, error)
	// GetQuickTunnelURL runs Quick Tunnel URL extraction on the node that hosts the app and returns the URL.
	GetQuickTunnelURL(ctx context.Context, appID string, nodeID string) (string, error)
	// CreateQuickTunnelForApp adds a Quick Tunnel (temporary trycloudflare.com URL) to an app that has no tunnel.
	CreateQuickTunnelForApp(ctx context.Context, appID string, nodeID string, service string, port int) (*db.App, error)
}

// TunnelService defines the primary port for tunnel management use cases
type TunnelService interface {
	// Tunnel operations
	GetTunnelByAppID(ctx context.Context, appID string, nodeID string) (*db.CloudflareTunnel, error)
	ListActiveTunnels(ctx context.Context, nodeIDs []string) ([]*db.CloudflareTunnel, error)
	SyncTunnelStatus(ctx context.Context, appID string, nodeID string) error
	UpdateTunnelIngress(ctx context.Context, appID string, nodeID string, req UpdateIngressRequest) error
	CreateDNSRecord(ctx context.Context, appID string, nodeID string, req CreateDNSRequest) error
	DeleteTunnel(ctx context.Context, appID string, nodeID string) error

	// Quick Tunnel operations (provider-specific)
	// These delegate to QuickTunnelProvider if the active provider supports it
	ExtractQuickTunnelURL(ctx context.Context, appID string, nodeID string) (string, error)
	CreateQuickTunnelConfig(targetService string, targetPort int, metricsHostPort int) (*tunnel.ContainerConfig, error)
	NextFreeQuickTunnelMetricsPort() (int, error)

	// Provider discovery (NEW)
	ListProviders(ctx context.Context) ([]ProviderInfo, error)
	GetProviderFeatures(ctx context.Context, providerName string) (*ProviderFeatures, error)
}

// ProviderInfo contains metadata about an available tunnel provider
type ProviderInfo struct {
	Name         string `json:"name"`
	DisplayName  string `json:"display_name"`
	IsConfigured bool   `json:"is_configured"`
}

// ProviderFeatures describes what features a tunnel provider supports
type ProviderFeatures struct {
	Provider     string `json:"provider"`
	DisplayName  string `json:"display_name"`
	IsConfigured bool   `json:"is_configured"`
	Features     map[string]bool `json:"features"`
}

// SystemService defines the primary port for system monitoring use cases
type SystemService interface {
	GetSystemStats(ctx context.Context, nodeIDs []string) ([]*system.SystemStats, error)
	GetAppStats(ctx context.Context, appID string, nodeID string) (*AppStats, error)
	GetAppLogs(ctx context.Context, appID string, nodeID string, service string) ([]byte, error)
	GetAppServices(ctx context.Context, appID string, nodeID string) ([]string, error)
	RestartContainer(ctx context.Context, containerID, nodeID string) error
	StopContainer(ctx context.Context, containerID, nodeID string) error
	DeleteContainer(ctx context.Context, containerID, nodeID string) error
}

// ComposeService defines the primary port for compose version management
type ComposeService interface {
	GetVersions(ctx context.Context, appID string, nodeID string) ([]*db.ComposeVersion, error)
	GetVersion(ctx context.Context, appID string, version int, nodeID string) (*db.ComposeVersion, error)
	RollbackToVersion(ctx context.Context, appID string, version int, nodeID string, reason *string, changedBy *string) (*db.ComposeVersion, error)
}

// NodeService defines the primary port for node management use cases
type NodeService interface {
	RegisterNode(ctx context.Context, req RegisterNodeRequest) (*db.Node, error)
	GetNode(ctx context.Context, nodeID string) (*db.Node, error)
	ListNodes(ctx context.Context) ([]*db.Node, error)
	UpdateNode(ctx context.Context, nodeID string, req UpdateNodeRequest) (*db.Node, error)
	DeleteNode(ctx context.Context, nodeID string) error
	HealthCheckNode(ctx context.Context, nodeID string) error
	HealthCheckAllNodes(ctx context.Context) error
	NodeHeartbeat(ctx context.Context, nodeID string) error
	SyncSettingsFromPrimary(ctx context.Context) error
	GetCurrentNodeInfo(ctx context.Context) (*db.Node, error)
}

// ============================================================================
// Request/Response Types
// ============================================================================

// CreateAppRequest represents the request to create a new app
type CreateAppRequest struct {
	Name               string           `json:"name" binding:"required"`
	Description        string           `json:"description"`
	ComposeContent    string           `json:"compose_content" binding:"required"`
	IngressRules      []db.IngressRule `json:"ingress_rules,omitempty"`
	NodeID            string           `json:"node_id,omitempty"`             // Target node for app deployment
	TunnelMode        string           `json:"tunnel_mode,omitempty"`        // "custom" | "quick" | "" (empty = no tunnel)
	QuickTunnelService string          `json:"quick_tunnel_service,omitempty"` // Required when tunnel_mode="quick"
	QuickTunnelPort   int              `json:"quick_tunnel_port,omitempty"`   // Required when tunnel_mode="quick"
}

// UpdateAppRequest represents the request to update an app
type UpdateAppRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	ComposeContent string `json:"compose_content"`
}

// UpdateIngressRequest represents the request to update tunnel ingress
type UpdateIngressRequest struct {
	IngressRules []db.IngressRule `json:"ingress_rules" binding:"required"`
	Hostname     string           `json:"hostname"`
	TargetDomain string           `json:"target_domain"`
}

// CreateDNSRequest represents the request to create a DNS record
type CreateDNSRequest struct {
	Hostname string `json:"hostname" binding:"required"`
	Domain   string `json:"domain" binding:"required"`
}

// AppStats represents application resource statistics
type AppStats struct {
	AppName           string              `json:"app_name"`
	TotalCPUPercent   float64             `json:"total_cpu_percent"`
	TotalMemoryBytes  int64               `json:"total_memory_bytes"`
	MemoryLimitBytes  int64               `json:"memory_limit_bytes"`
	Containers        []ContainerStats    `json:"containers"`
	Timestamp         time.Time           `json:"timestamp"`
	Status            string              `json:"status"`
	Message           string              `json:"message,omitempty"`
}

// ContainerStats represents individual container statistics
type ContainerStats struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryBytes   int64   `json:"memory_bytes"`
	MemoryLimit   int64   `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
	NetInput      int64   `json:"net_input"`
	NetOutput     int64   `json:"net_output"`
	BlockInput    int64   `json:"block_input"`
	BlockOutput   int64   `json:"block_output"`
}

// ComposeFile represents a Docker Compose file structure
type ComposeFile struct {
	Version  string                          `yaml:"version,omitempty"`
	Services map[string]DockerComposeService `yaml:"services"`
	Networks map[string]ComposeNetwork       `yaml:"networks,omitempty"`
	Volumes  map[string]ComposeVolume        `yaml:"volumes,omitempty"`
}

// DockerComposeService represents a service in a Docker Compose file
type DockerComposeService struct {
	Image       string                 `yaml:"image,omitempty"`
	Build       interface{}            `yaml:"build,omitempty"`
	Command     interface{}            `yaml:"command,omitempty"`
	Environment map[string]interface{} `yaml:"environment,omitempty"`
	Ports       []string               `yaml:"ports,omitempty"`
	Volumes     []string               `yaml:"volumes,omitempty"`
	Networks    []string               `yaml:"networks,omitempty"`
	DependsOn   []string               `yaml:"depends_on,omitempty"`
	Restart     string                 `yaml:"restart,omitempty"`
}

// ComposeNetwork represents a network in a Docker Compose file
type ComposeNetwork struct {
	Driver string                 `yaml:"driver,omitempty"`
	Config map[string]interface{} `yaml:",inline"`
}

// ComposeVolume represents a volume in a Docker Compose file
type ComposeVolume struct {
	Driver string                 `yaml:"driver,omitempty"`
	Config map[string]interface{} `yaml:",inline"`
}

// RegisterNodeRequest represents the request to register a new node
type RegisterNodeRequest struct {
	ID          string `json:"id" binding:"required"`            // Required: Secondary's existing node ID for heartbeat auth
	Name        string `json:"name" binding:"required"`
	APIEndpoint string `json:"api_endpoint" binding:"required"`
	APIKey      string `json:"api_key" binding:"required"`
}

// UpdateNodeRequest represents the request to update a node
type UpdateNodeRequest struct {
	Name        string `json:"name"`
	APIEndpoint string `json:"api_endpoint"`
	APIKey      string `json:"api_key"`
}
