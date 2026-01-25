package domain

import (
	"context"
	"time"

	"github.com/selfhostly/internal/db"
)

// ============================================================================
// Primary Ports (Application Use Cases)
// ============================================================================

// AppService defines the primary port for application management use cases
type AppService interface {
	CreateApp(ctx context.Context, req CreateAppRequest) (*db.App, error)
	GetApp(ctx context.Context, appID string) (*db.App, error)
	ListApps(ctx context.Context) ([]*db.App, error)
	UpdateApp(ctx context.Context, appID string, req UpdateAppRequest) (*db.App, error)
	DeleteApp(ctx context.Context, appID string) error
	StartApp(ctx context.Context, appID string) (*db.App, error)
	StopApp(ctx context.Context, appID string) (*db.App, error)
	UpdateAppContainers(ctx context.Context, appID string) (*db.App, error)
	RepairApp(ctx context.Context, appID string) (*db.App, error)
}

// TunnelService defines the primary port for tunnel management use cases
type TunnelService interface {
	GetTunnelByAppID(ctx context.Context, appID string) (*db.CloudflareTunnel, error)
	ListActiveTunnels(ctx context.Context) ([]*db.CloudflareTunnel, error)
	SyncTunnelStatus(ctx context.Context, appID string) error
	UpdateTunnelIngress(ctx context.Context, appID string, req UpdateIngressRequest) error
	CreateDNSRecord(ctx context.Context, appID string, req CreateDNSRequest) error
	DeleteTunnel(ctx context.Context, appID string) error
}

// SystemService defines the primary port for system monitoring use cases
type SystemService interface {
	GetSystemStats(ctx context.Context) (*SystemStats, error)
	GetAppStats(ctx context.Context, appID string) (*AppStats, error)
	GetAppLogs(ctx context.Context, appID string) ([]byte, error)
	RestartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	DeleteContainer(ctx context.Context, containerID string) error
}

// ComposeService defines the primary port for compose version management
type ComposeService interface {
	GetVersions(ctx context.Context, appID string) ([]*db.ComposeVersion, error)
	GetVersion(ctx context.Context, appID string, version int) (*db.ComposeVersion, error)
	RollbackToVersion(ctx context.Context, appID string, version int, reason *string, changedBy *string) (*db.ComposeVersion, error)
}

// ============================================================================
// Secondary Ports (Infrastructure Interfaces)
// ============================================================================

// AppRepository defines the port for app persistence
type AppRepository interface {
	Save(ctx context.Context, app *db.App) error
	FindByID(ctx context.Context, id string) (*db.App, error)
	FindByName(ctx context.Context, name string) (*db.App, error)
	FindAll(ctx context.Context) ([]*db.App, error)
	Update(ctx context.Context, app *db.App) error
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
}

// TunnelRepository defines the port for tunnel persistence
type TunnelRepository interface {
	Save(ctx context.Context, tunnel *db.CloudflareTunnel) error
	FindByAppID(ctx context.Context, appID string) (*db.CloudflareTunnel, error)
	FindByTunnelID(ctx context.Context, tunnelID string) (*db.CloudflareTunnel, error)
	FindAllActive(ctx context.Context) ([]*db.CloudflareTunnel, error)
	Update(ctx context.Context, tunnel *db.CloudflareTunnel) error
	DeleteByAppID(ctx context.Context, appID string) error
}

// ComposeVersionRepository defines the port for compose version persistence
type ComposeVersionRepository interface {
	Save(ctx context.Context, version *db.ComposeVersion) error
	FindByAppID(ctx context.Context, appID string) ([]*db.ComposeVersion, error)
	FindByAppIDAndVersion(ctx context.Context, appID string, version int) (*db.ComposeVersion, error)
	FindCurrentVersion(ctx context.Context, appID string) (*db.ComposeVersion, error)
	GetLatestVersionNumber(ctx context.Context, appID string) (int, error)
	MarkAllAsNotCurrent(ctx context.Context, appID string) error
	MarkAsCurrent(ctx context.Context, appID string, version int) error
	DeleteByAppID(ctx context.Context, appID string) error
}

// SettingsRepository defines the port for settings persistence
type SettingsRepository interface {
	Get(ctx context.Context) (*db.Settings, error)
	Update(ctx context.Context, settings *db.Settings) error
}

// ContainerOrchestrator defines the port for container management
type ContainerOrchestrator interface {
	CreateAppDirectory(ctx context.Context, name, composeContent string) error
	WriteComposeFile(ctx context.Context, name, content string) error
	StartApp(ctx context.Context, name string) error
	StopApp(ctx context.Context, name string) error
	UpdateApp(ctx context.Context, name string) error
	GetAppStatus(ctx context.Context, name string) (string, error)
	GetAppLogs(ctx context.Context, name string) ([]byte, error)
	GetAppStats(ctx context.Context, name string) (*AppStats, error)
	DeleteAppDirectory(ctx context.Context, name string) error
	RestartCloudflared(ctx context.Context, name string) error
	RestartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	DeleteContainer(ctx context.Context, containerID string) error
	GetSystemStats(ctx context.Context) (*SystemStats, error)
}

// TunnelProvider defines the port for Cloudflare tunnel operations
type TunnelProvider interface {
	CreateTunnel(ctx context.Context, name string) (tunnelID, token string, err error)
	DeleteTunnel(ctx context.Context, tunnelID string) error
	GetTunnelToken(ctx context.Context, tunnelID string) (string, error)
	CreatePublicRoute(ctx context.Context, tunnelID, service string) (publicURL string, err error)
	CreateIngressConfiguration(ctx context.Context, tunnelID string, rules []db.IngressRule) error
	GetZoneID(ctx context.Context, domain string) (string, error)
	CreateDNSRecord(ctx context.Context, zoneID, hostname, tunnelID string) (recordID string, err error)
	DeleteDNSRecordsForTunnel(ctx context.Context, tunnelID string) error
}

// ComposeParser defines the port for Docker Compose file operations
type ComposeParser interface {
	Parse(content []byte) (*ComposeFile, error)
	Marshal(compose *ComposeFile) ([]byte, error)
	ExtractNetworks(compose *ComposeFile) []string
	InjectCloudflared(compose *ComposeFile, appName, tunnelToken, network string) error
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

// SystemStats represents system-wide statistics
type SystemStats struct {
	Containers      []ContainerInfo `json:"containers"`
	TotalContainers int             `json:"total_containers"`
	RunningCount    int             `json:"running_count"`
	StoppedCount    int             `json:"stopped_count"`
	CPUUsage        float64         `json:"cpu_usage"`
	MemoryUsage     int64           `json:"memory_usage"`
	MemoryLimit     int64           `json:"memory_limit"`
	DiskUsage       *DiskUsage      `json:"disk_usage,omitempty"`
}

// ContainerInfo represents container information
type ContainerInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	State   string `json:"state"`
	Created int64  `json:"created"`
}

// DiskUsage represents disk usage information
type DiskUsage struct {
	Path        string  `json:"path"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
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
