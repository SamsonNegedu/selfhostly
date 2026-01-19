package domain

import (
	"context"
)

// AppRepository defines the interface for app data operations
type AppRepository interface {
	Create(ctx context.Context, app *App) error
	Get(ctx context.Context, id int64) (*App, error)
	List(ctx context.Context) ([]*App, error)
	Update(ctx context.Context, app *App) error
	Delete(ctx context.Context, id int64) error
}

// DockerManager defines the interface for Docker operations
type DockerManager interface {
	CreateAppDirectory(name, composeContent string) error
	StartApp(name string) error
	StopApp(name string) error
	UpdateApp(name string) error
	GetAppStatus(name string) (string, error)
	GetAppLogs(name string) ([]byte, error)
	DeleteAppDirectory(name string) error
	RestartCloudflared(name string) error
}

// CloudflareTunnel defines the interface for Cloudflare tunnel operations
type CloudflareTunnel interface {
	CreateTunnel(appName string) (tunnelID, token string, err error)
	DeleteTunnel(tunnelID string) error
	CreatePublicRoute(tunnelID, service string) (publicURL string, err error)
}

// CloudflareTunnelManager defines the interface for Cloudflare tunnel management operations
type CloudflareTunnelManager interface {
	CreateTunnelWithMetadata(appName string, appID int64) (*CloudflareTunnel, error)
	UpdateTunnelStatus(tunnelID string, status string, errorDetails string) error
	DeleteTunnelByAppID(appID int64) error
	GetAllActiveTunnels() ([]*CloudflareTunnel, error)
	SyncTunnelStatus(tunnelID string) error
}

// App represents a self-hosted application domain entity
type App struct {
	ID             int64
	Name           string
	Description    string
	ComposeContent string
	TunnelToken    string
	TunnelID       string
	TunnelDomain   string
	PublicURL      string
	Status         string
	ErrorMessage   string
}
