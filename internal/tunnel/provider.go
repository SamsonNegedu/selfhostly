package tunnel

import (
	"context"
)

// Provider defines the core interface that ALL tunnel providers must implement.
// This is intentionally minimal - providers only implement what they actually support.
//
// Following Interface Segregation Principle (ISP), optional features are defined
// as separate interfaces below. Providers implement only the interfaces for features
// they support.
type Provider interface {
	// Core lifecycle operations - required for all providers

	// CreateTunnel creates a new tunnel for an application.
	// The provider is responsible for:
	// - Calling its external API to create the tunnel
	// - Storing tunnel metadata in its provider-specific database table
	// - Returning a generic Tunnel response
	CreateTunnel(ctx context.Context, opts CreateOptions) (*Tunnel, error)

	// GetTunnelByAppID retrieves tunnel information for a specific application.
	// Returns ErrTunnelNotFound if no tunnel exists for this app.
	GetTunnelByAppID(ctx context.Context, appID string) (*Tunnel, error)

	// DeleteTunnel deletes a tunnel and all associated resources.
	// This should:
	// - Remove the tunnel from the provider's external API
	// - Clean up the provider-specific database record
	DeleteTunnel(ctx context.Context, appID string) error

	// CleanupOrphanedTunnels removes tunnels that exist in the database but
	// not in the provider's API, or vice versa.
	CleanupOrphanedTunnels(ctx context.Context) error

	// Metadata

	// Name returns the provider's unique identifier (e.g., "cloudflare", "ngrok")
	Name() string

	// DisplayName returns the provider's human-readable name (e.g., "Cloudflare Tunnel", "Ngrok")
	DisplayName() string
}

// IngressProvider defines the interface for providers that support configurable
// ingress/routing rules. Not all providers support this (e.g., ngrok doesn't).
//
// Example: Cloudflare Tunnels allow you to configure routing based on hostname/path.
type IngressProvider interface {
	Provider

	// UpdateIngress updates the routing/ingress rules for a tunnel.
	// The rules parameter is intentionally interface{} since different providers
	// have different rule formats. The provider is responsible for type assertion
	// and validation.
	UpdateIngress(ctx context.Context, appID string, rules interface{}) error
}

// DNSProvider defines the interface for providers that can manage DNS records.
// Not all providers have DNS management capabilities.
//
// Example: Cloudflare can create CNAME records pointing to tunnels.
type DNSProvider interface {
	Provider

	// CreateDNSRecord creates a DNS record pointing to the tunnel.
	// The provider determines the record type (CNAME, A, etc.) based on its
	// DNS infrastructure.
	CreateDNSRecord(ctx context.Context, appID string, opts DNSOptions) error
}

// StatusSyncProvider defines the interface for providers that can sync tunnel
// status from their external API.
//
// Example: Cloudflare Tunnels can be queried for connection status.
type StatusSyncProvider interface {
	Provider

	// SyncStatus queries the provider's API to get the latest tunnel status
	// and updates the database accordingly.
	SyncStatus(ctx context.Context, appID string) error
}

// ContainerProvider defines the interface for providers that require a
// sidecar container to be injected into the docker-compose file.
//
// Not all providers need this:
// - Cloudflare: needs cloudflared container
// - Ngrok: needs ngrok agent container
// - Some hosted services: might not need any container
type ContainerProvider interface {
	Provider

	// GetContainerConfig returns the Docker container configuration for this provider.
	// Returns nil if the provider doesn't require a container.
	//
	// The returned config will be injected into the application's docker-compose file.
	GetContainerConfig(tunnelToken string, appName string) *ContainerConfig
}

// ListProvider defines the interface for providers that can list all tunnels.
// This is optional because some providers might not support efficient listing.
type ListProvider interface {
	Provider

	// ListTunnels returns all tunnels managed by this provider.
	// The nodeIDs parameter can be used to filter tunnels by node in multi-node setups.
	ListTunnels(ctx context.Context, nodeIDs []string) ([]*Tunnel, error)
}
