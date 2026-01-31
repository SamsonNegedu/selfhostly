package tunnel

import (
	"time"
)

// Tunnel represents a provider-agnostic tunnel response.
// This is used at the service layer and above - it does NOT correspond to
// a database table. Each provider manages its own database schema.
//
// The Tunnel type provides a common interface for all providers while allowing
// provider-specific data in the Metadata field.
type Tunnel struct {
	// ID is the internal database ID
	ID string `json:"id"`

	// AppID is the ID of the application this tunnel belongs to
	AppID string `json:"app_id"`

	// ProviderType identifies which provider manages this tunnel
	// (e.g., "cloudflare", "ngrok", "tailscale")
	ProviderType string `json:"provider_type"`

	// TunnelID is the provider's unique identifier for this tunnel
	TunnelID string `json:"tunnel_id"`

	// TunnelName is a human-readable name for the tunnel
	TunnelName string `json:"tunnel_name"`

	// TunnelToken is the authentication token/credential for the tunnel
	// This is typically used by container-based providers
	TunnelToken string `json:"tunnel_token,omitempty"`

	// PublicURL is the public-facing URL for accessing this tunnel
	PublicURL string `json:"public_url"`

	// Status represents the current state of the tunnel
	// Common values: "active", "inactive", "error", "deleted"
	Status string `json:"status"`

	// IsActive indicates whether the tunnel is currently active
	IsActive bool `json:"is_active"`

	// IngressRules contains provider-specific routing rules
	// Type is interface{} because different providers have different rule formats
	// For Cloudflare: []db.IngressRule
	// For other providers: might be nil or a different structure
	IngressRules interface{} `json:"ingress_rules,omitempty"`

	// Metadata contains any provider-specific data that doesn't fit the common fields
	// This keeps the Tunnel type extensible without breaking changes
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// ErrorDetails contains error information if the tunnel is in error state
	ErrorDetails string `json:"error_details,omitempty"`

	// LastSyncedAt is the timestamp of the last status sync (if supported)
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`

	// CreatedAt is when the tunnel was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the tunnel was last updated
	UpdatedAt time.Time `json:"updated_at"`
}

// ContainerConfig defines the Docker container configuration for a tunnel provider.
// This is used to inject the tunnel sidecar container into an application's
// docker-compose file.
//
// Not all providers need this - some are purely API-based.
type ContainerConfig struct {
	// Image is the Docker image to use (e.g., "cloudflare/cloudflared:latest")
	Image string

	// Command is the command to run in the container (e.g., ["tunnel", "run"])
	Command []string

	// Environment contains environment variables for the container
	// Typically includes authentication tokens and configuration
	Environment map[string]string

	// Networks are the Docker networks the container should join
	// Usually includes the application's network for internal routing
	Networks []string

	// Volumes are optional volume mounts for the container
	Volumes []string

	// Ports are optional port mappings (e.g., ["2000:2000"] for Quick Tunnel metrics)
	Ports []string
}
