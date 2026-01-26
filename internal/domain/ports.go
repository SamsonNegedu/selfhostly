package domain

import (
	"context"
	"time"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/system"
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
	RepairApp(ctx context.Context, appID string) (*db.App, error)
	RestartCloudflared(ctx context.Context, appID string, nodeID string) error
}

// TunnelService defines the primary port for tunnel management use cases
type TunnelService interface {
	GetTunnelByAppID(ctx context.Context, appID string) (*db.CloudflareTunnel, error)
	ListActiveTunnels(ctx context.Context, nodeIDs []string) ([]*db.CloudflareTunnel, error)
	SyncTunnelStatus(ctx context.Context, appID string) error
	UpdateTunnelIngress(ctx context.Context, appID string, req UpdateIngressRequest) error
	CreateDNSRecord(ctx context.Context, appID string, req CreateDNSRequest) error
	DeleteTunnel(ctx context.Context, appID string) error
}

// SystemService defines the primary port for system monitoring use cases
type SystemService interface {
	GetSystemStats(ctx context.Context, nodeIDs []string) ([]*system.SystemStats, error)
	GetAppStats(ctx context.Context, appID string) (*AppStats, error)
	GetAppLogs(ctx context.Context, appID string) ([]byte, error)
	RestartContainer(ctx context.Context, containerID, nodeID string) error
	StopContainer(ctx context.Context, containerID, nodeID string) error
	DeleteContainer(ctx context.Context, containerID, nodeID string) error
}

// ComposeService defines the primary port for compose version management
type ComposeService interface {
	GetVersions(ctx context.Context, appID string) ([]*db.ComposeVersion, error)
	GetVersion(ctx context.Context, appID string, version int) (*db.ComposeVersion, error)
	RollbackToVersion(ctx context.Context, appID string, version int, reason *string, changedBy *string) (*db.ComposeVersion, error)
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
	Name           string            `json:"name" binding:"required"`
	Description    string            `json:"description"`
	ComposeContent string            `json:"compose_content" binding:"required"`
	IngressRules   []db.IngressRule  `json:"ingress_rules,omitempty"`
	NodeID         string            `json:"node_id,omitempty"` // Target node for app deployment
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
