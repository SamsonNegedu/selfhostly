package cloudflare

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/selfhostly/internal/cloudflare"
	"github.com/selfhostly/internal/constants"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/tunnel"
)

// Provider is the Cloudflare tunnel provider implementation.
// It wraps the existing cloudflare.TunnelManager and implements all tunnel interfaces
// since Cloudflare supports all features (ingress, DNS, status sync, containers).
type Provider struct {
	apiToken  string
	accountID string
	database  *db.DB
	manager   *cloudflare.TunnelManager
	logger    *slog.Logger
}

// Config contains the configuration required for Cloudflare provider.
type Config struct {
	APIToken  string       `json:"api_token"`
	AccountID string       `json:"account_id"`
	Database  *db.DB       `json:"-"` // Not serialized
	Logger    *slog.Logger `json:"-"` // Not serialized
}

// NewProvider creates a new Cloudflare provider instance.
// This is the factory function registered with the tunnel registry.
func NewProvider(config map[string]interface{}) (tunnel.Provider, error) {
	// Extract required fields
	apiToken, ok := config["api_token"].(string)
	if !ok || apiToken == "" {
		return nil, fmt.Errorf("%w: api_token is required", tunnel.ErrInvalidConfiguration)
	}

	accountID, ok := config["account_id"].(string)
	if !ok || accountID == "" {
		return nil, fmt.Errorf("%w: account_id is required", tunnel.ErrInvalidConfiguration)
	}

	// Database and logger should be injected separately since they're not serializable
	database, ok := config["database"].(*db.DB)
	if !ok || database == nil {
		return nil, fmt.Errorf("%w: database is required", tunnel.ErrInvalidConfiguration)
	}

	logger, ok := config["logger"].(*slog.Logger)
	if !ok {
		// Use a default logger if not provided
		logger = slog.Default()
	}

	return NewProviderWithConfig(Config{
		APIToken:  apiToken,
		AccountID: accountID,
		Database:  database,
		Logger:    logger,
	})
}

// NewProviderWithConfig creates a new Cloudflare provider with a structured config.
// This is useful for testing and when you have a Config struct already.
func NewProviderWithConfig(cfg Config) (*Provider, error) {
	if cfg.APIToken == "" {
		return nil, fmt.Errorf("%w: api_token is required", tunnel.ErrInvalidConfiguration)
	}
	if cfg.AccountID == "" {
		return nil, fmt.Errorf("%w: account_id is required", tunnel.ErrInvalidConfiguration)
	}
	if cfg.Database == nil {
		return nil, fmt.Errorf("%w: database is required", tunnel.ErrInvalidConfiguration)
	}

	manager := cloudflare.NewTunnelManager(cfg.APIToken, cfg.AccountID, cfg.Database)

	return &Provider{
		apiToken:  cfg.APIToken,
		accountID: cfg.AccountID,
		database:  cfg.Database,
		manager:   manager,
		logger:    cfg.Logger,
	}, nil
}

// ============================================================================
// Core Provider Interface
// ============================================================================

// CreateTunnel creates a new Cloudflare tunnel for an application.
func (p *Provider) CreateTunnel(ctx context.Context, opts tunnel.CreateOptions) (*tunnel.Tunnel, error) {
	p.logger.InfoContext(ctx, "creating cloudflare tunnel", "app_id", opts.AppID, "name", opts.Name)

	// Create tunnel via Cloudflare API
	tunnelID, tunnelToken, err := p.manager.ApiManager.CreateTunnel(opts.Name)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to create cloudflare tunnel", "error", err)
		return nil, fmt.Errorf("failed to create cloudflare tunnel: %w", err)
	}

	// Tunnel is source of truth for public URL (placeholder until custom domain is set via ingress)
	publicURL := fmt.Sprintf("https://%s.cfargotunnel.com", tunnelID)

	cfTunnel := db.NewCloudflareTunnel(opts.AppID, tunnelID, opts.Name, tunnelToken, p.accountID, publicURL)
	if err := p.database.CreateCloudflareTunnel(cfTunnel); err != nil {
		p.logger.ErrorContext(ctx, "failed to save tunnel to database", "tunnel_id", tunnelID, "error", err)
		// Cleanup: try to delete the tunnel from Cloudflare API
		if delErr := p.manager.ApiManager.DeleteTunnel(tunnelID); delErr != nil {
			p.logger.ErrorContext(ctx, "failed to cleanup tunnel after database error", "tunnel_id", tunnelID, "error", delErr)
		}
		return nil, fmt.Errorf("failed to save tunnel to database: %w", err)
	}

	p.logger.InfoContext(ctx, "cloudflare tunnel created successfully", "tunnel_id", tunnelID, "public_url", publicURL)

	return p.toGenericTunnel(cfTunnel, publicURL), nil
}

// GetTunnelByAppID retrieves tunnel information for a specific application.
func (p *Provider) GetTunnelByAppID(ctx context.Context, appID string) (*tunnel.Tunnel, error) {
	cfTunnel, err := p.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, tunnel.ErrTunnelNotFound
		}
		return nil, fmt.Errorf("failed to get tunnel from database: %w", err)
	}

	// Get public URL from app if available
	app, err := p.database.GetApp(appID)
	if err == nil && app.PublicURL != "" {
		return p.toGenericTunnel(cfTunnel, app.PublicURL), nil
	}

	return p.toGenericTunnel(cfTunnel, ""), nil
}

// DeleteTunnel deletes a Cloudflare tunnel and all associated resources.
func (p *Provider) DeleteTunnel(ctx context.Context, appID string) error {
	p.logger.InfoContext(ctx, "deleting cloudflare tunnel", "app_id", appID)

	// Use existing manager method which handles both API and database cleanup
	if err := p.manager.DeleteTunnelByAppID(appID); err != nil {
		p.logger.ErrorContext(ctx, "failed to delete tunnel", "app_id", appID, "error", err)
		return fmt.Errorf("failed to delete tunnel: %w", err)
	}

	p.logger.InfoContext(ctx, "cloudflare tunnel deleted successfully", "app_id", appID)
	return nil
}

// CleanupOrphanedTunnels removes tunnels that exist in the database but not in Cloudflare's API.
func (p *Provider) CleanupOrphanedTunnels(ctx context.Context) error {
	p.logger.InfoContext(ctx, "cleaning up orphaned cloudflare tunnels")

	// Get all tunnels from database
	tunnels, err := p.database.ListActiveCloudflareTunnels()
	if err != nil {
		return fmt.Errorf("failed to list tunnels from database: %w", err)
	}

	cleanedCount := 0
	for _, cfTunnel := range tunnels {
		// Check if tunnel still exists in Cloudflare API by trying to get the tunnel token
		// If we get an error, we assume the tunnel doesn't exist (or API is unreachable)
		_, err := p.manager.ApiManager.GetTunnelToken(cfTunnel.TunnelID)
		if err != nil {
			// Tunnel likely doesn't exist in API, remove from database
			p.logger.InfoContext(ctx, "removing orphaned tunnel", "tunnel_id", cfTunnel.TunnelID, "app_id", cfTunnel.AppID, "reason", err)
			if err := p.database.DeleteCloudflareTunnel(cfTunnel.ID); err != nil {
				p.logger.ErrorContext(ctx, "failed to delete orphaned tunnel", "tunnel_id", cfTunnel.TunnelID, "error", err)
			} else {
				cleanedCount++
			}
		}
	}

	p.logger.InfoContext(ctx, "cleanup completed", "cleaned_count", cleanedCount)
	return nil
}

// Name returns the provider's unique identifier.
func (p *Provider) Name() string {
	return constants.ProviderCloudflare
}

// DisplayName returns the provider's human-readable name.
func (p *Provider) DisplayName() string {
	return "Cloudflare Tunnel"
}

// ============================================================================
// IngressProvider Interface
// ============================================================================

// UpdateIngress updates the routing/ingress rules for a Cloudflare tunnel.
func (p *Provider) UpdateIngress(ctx context.Context, appID string, rules interface{}) error {
	p.logger.InfoContext(ctx, "updating tunnel ingress", "app_id", appID)

	// Type assert to Cloudflare-specific ingress rules
	ingressRules, ok := rules.([]db.IngressRule)
	if !ok {
		return fmt.Errorf("%w: expected []db.IngressRule, got %T", tunnel.ErrInvalidConfiguration, rules)
	}

	// Get the tunnel
	cfTunnel, err := p.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return tunnel.ErrTunnelNotFound
		}
		return fmt.Errorf("failed to get tunnel: %w", err)
	}

	// Convert to Cloudflare API format and ensure catch-all rule
	cfRules := cloudflare.ConvertToCloudflareRules(ingressRules)
	cfRules = cloudflare.EnsureCatchAllRule(cfRules)

	// Update via Cloudflare API
	if err := p.manager.UpdateTunnelIngress(cfTunnel.TunnelID, cfRules, "", ""); err != nil {
		p.logger.ErrorContext(ctx, "failed to update ingress", "tunnel_id", cfTunnel.TunnelID, "error", err)
		return err // Don't wrap - already has context from manager
	}

	// Update tunnel record: ingress rules and public_url from first hostname (tunnel is source of truth)
	cfTunnel.IngressRules = &ingressRules
	if len(ingressRules) > 0 && ingressRules[0].Hostname != nil && *ingressRules[0].Hostname != "" {
		cfTunnel.PublicURL = fmt.Sprintf("https://%s", *ingressRules[0].Hostname)
	}
	if err := p.database.UpdateCloudflareTunnel(cfTunnel); err != nil {
		p.logger.WarnContext(ctx, "failed to update tunnel in database", "tunnel_id", cfTunnel.TunnelID, "error", err)
		// Don't fail the request - API update succeeded
	}
	if cfTunnel.PublicURL != "" {
		if app, err := p.database.GetApp(cfTunnel.AppID); err == nil {
			app.PublicURL = cfTunnel.PublicURL
			_ = p.database.UpdateApp(app)
		}
	}

	p.logger.InfoContext(ctx, "tunnel ingress updated successfully", "tunnel_id", cfTunnel.TunnelID)
	return nil
}

// ============================================================================
// DNSProvider Interface
// ============================================================================

// CreateDNSRecord creates a DNS record pointing to the Cloudflare tunnel.
func (p *Provider) CreateDNSRecord(ctx context.Context, appID string, opts tunnel.DNSOptions) error {
	p.logger.InfoContext(ctx, "creating DNS record", "app_id", appID, "hostname", opts.Hostname, "domain", opts.Domain)

	// Get the tunnel
	cfTunnel, err := p.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return tunnel.ErrTunnelNotFound
		}
		return fmt.Errorf("failed to get tunnel: %w", err)
	}

	// Get zone ID for the domain
	zoneID, err := p.manager.ApiManager.GetZoneID(opts.Domain)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to get zone ID", "domain", opts.Domain, "error", err)
		return fmt.Errorf("failed to get zone ID for domain %s: %w", opts.Domain, err)
	}

	// Create DNS record
	_, err = p.manager.ApiManager.CreateDNSRecord(zoneID, opts.Hostname, cfTunnel.TunnelID)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to create DNS record", "hostname", opts.Hostname, "error", err)
		return fmt.Errorf("failed to create DNS record: %w", err)
	}

	p.logger.InfoContext(ctx, "DNS record created successfully", "hostname", opts.Hostname)
	return nil
}

// ============================================================================
// StatusSyncProvider Interface
// ============================================================================

// SyncStatus synchronizes tunnel status with Cloudflare's API.
func (p *Provider) SyncStatus(ctx context.Context, appID string) error {
	p.logger.InfoContext(ctx, "syncing tunnel status", "app_id", appID)

	cfTunnel, err := p.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return tunnel.ErrTunnelNotFound
		}
		return fmt.Errorf("failed to get tunnel: %w", err)
	}

	if err := p.manager.SyncTunnelStatus(cfTunnel.TunnelID); err != nil {
		p.logger.ErrorContext(ctx, "failed to sync tunnel status", "tunnel_id", cfTunnel.TunnelID, "error", err)
		return fmt.Errorf("failed to sync tunnel status: %w", err)
	}

	p.logger.InfoContext(ctx, "tunnel status synced successfully", "tunnel_id", cfTunnel.TunnelID)
	return nil
}

// ============================================================================
// ContainerProvider Interface
// ============================================================================

// GetContainerConfig returns the Docker container configuration for Cloudflare named tunnel.
func (p *Provider) GetContainerConfig(tunnelToken string, appName string) *tunnel.ContainerConfig {
	return &tunnel.ContainerConfig{
		Image:   "cloudflare/cloudflared:latest",
		Command: []string{"tunnel", "run"},
		Environment: map[string]string{
			"TUNNEL_TOKEN": tunnelToken,
		},
	}
}

// ============================================================================
// QuickTunnelProvider Interface
// ============================================================================

// CreateQuickTunnelConfig implements QuickTunnelProvider interface.
// Creates a container configuration for a Cloudflare Quick Tunnel.
func (p *Provider) CreateQuickTunnelConfig(targetService string, targetPort int, metricsHostPort int) *tunnel.ContainerConfig {
	return QuickTunnelContainerConfig(targetService, targetPort, metricsHostPort)
}

// ExtractQuickTunnelURL implements QuickTunnelProvider interface.
// Extracts the public URL from a running Quick Tunnel by querying the metrics endpoint.
func (p *Provider) ExtractQuickTunnelURL(ctx context.Context, appName string, composeContent string, appsDir string, commandExecutor tunnel.CommandExecutor) (string, error) {
	// Wait a bit for tunnel to start and metrics endpoint to be ready
	// This helps avoid immediate failures when tunnel container was just started
	p.logger.DebugContext(ctx, "waiting for tunnel to be ready before extracting URL", "app", appName, "delay", constants.QuickTunnelStartupDelay)
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("context cancelled during startup delay: %w", ctx.Err())
	case <-time.After(constants.QuickTunnelStartupDelay):
		// Continue
	}
	
	// Extract URL via HTTP method (accessing metrics endpoint via network)
	// Note: Exec method is not available because cloudflared containers don't include curl/wget
	// Build list of endpoints to try, prioritizing based on environment
	endpoints := p.buildQuickTunnelMetricsEndpoints(composeContent)
	p.logger.InfoContext(ctx, "extracting Quick Tunnel URL from metrics (HTTP method)", "app", appName, "endpoints", endpoints)
	
	// Try each endpoint with retries (fewer retries per endpoint since we try multiple)
	// Use shorter retry interval and fewer retries to avoid long waits
	for i, endpoint := range endpoints {
		p.logger.InfoContext(ctx, "trying endpoint to extract Quick Tunnel URL", "app", appName, "endpoint", endpoint, "attempt", i+1, "total_endpoints", len(endpoints))
		url, err := cloudflare.ExtractQuickTunnelURL(ctx, endpoint, 5, 1*time.Second)
		if err == nil && url != "" {
			p.logger.InfoContext(ctx, "Quick Tunnel URL extracted via HTTP method", "app", appName, "public_url", url, "endpoint", endpoint)
			return url, nil
		}
		// Check if context was cancelled - don't try next endpoint if so
		if ctx.Err() != nil {
			p.logger.WarnContext(ctx, "context cancelled while extracting URL", "app", appName, "error", ctx.Err())
			return "", fmt.Errorf("context cancelled while extracting URL: %w", ctx.Err())
		}
		p.logger.WarnContext(ctx, "failed to extract URL from endpoint, trying next", "app", appName, "endpoint", endpoint, "error", err)
	}
	
	return "", fmt.Errorf("failed to extract Quick Tunnel URL from any endpoint (tried_endpoints: %v)", endpoints)
}

// buildQuickTunnelMetricsEndpoints returns a list of metrics endpoints to try, ordered by priority.
// Works in both local and Docker environments by trying multiple host options.
func (p *Provider) buildQuickTunnelMetricsEndpoints(composeContent string) []string {
	// Parse compose to extract container name and ports
	compose, err := docker.ParseCompose([]byte(composeContent))
	if err != nil {
		// Fallback to defaults if parsing fails
		return p.buildDefaultEndpoints(constants.QuickTunnelMetricsPort)
	}

	// Find tunnel service and extract the host port (for published ports)
	var hostPort int = constants.QuickTunnelMetricsPort

	for _, svc := range compose.Services {
		if strings.Contains(svc.Image, "cloudflared") {
			// Extract host port from port mapping
			if extractedHostPort, ok := docker.ExtractQuickTunnelMetricsHostPort(composeContent); ok {
				hostPort = extractedHostPort
			}
			break
		}
	}

	var endpoints []string
	isInDocker := p.isRunningInDocker()

	if isInDocker {
		// Backend is running in Docker, tunnel is on a different network with published ports.
		// We need to access the tunnel via the Docker HOST's published port, not the container IP.
		// Priority order:
		// 1. host.docker.internal (Docker Desktop on Mac/Windows)
		// 2. Default Docker bridge gateway (172.17.0.1) - routes to host on Linux
		// 3. Network gateway (might work in some setups)
		// 4. Other common gateway IPs
		// Note: Container name/IP won't work across different Docker networks
		
		endpoints = append(endpoints,
			fmt.Sprintf("http://host.docker.internal:%d/metrics", hostPort),
			fmt.Sprintf("http://172.17.0.1:%d/metrics", hostPort), // Default Docker bridge - often routes to host
		)
		
		// Try current network gateway (might work for published ports)
		if gatewayIP := p.getNetworkGatewayIP(); gatewayIP != "" && gatewayIP != "172.17.0.1" {
			endpoints = append(endpoints, fmt.Sprintf("http://%s:%d/metrics", gatewayIP, hostPort))
		}
		
		endpoints = append(endpoints,
			fmt.Sprintf("http://172.18.0.1:%d/metrics", hostPort), // Common Docker Compose network gateway
			fmt.Sprintf("http://172.19.0.1:%d/metrics", hostPort), // Another common gateway
			fmt.Sprintf("http://172.20.0.1:%d/metrics", hostPort), // Yet another gateway
			fmt.Sprintf("http://localhost:%d/metrics", hostPort),  // Fallback (usually doesn't work in containers)
		)
	} else {
		// Backend is running locally - use host port
		endpoints = []string{
			fmt.Sprintf("http://localhost:%d/metrics", hostPort),
		}
	}

	return endpoints
}

// buildDefaultEndpoints builds default endpoints when compose parsing fails
func (p *Provider) buildDefaultEndpoints(port int) []string {
	isInDocker := p.isRunningInDocker()
	if isInDocker {
		return []string{
			fmt.Sprintf("http://host.docker.internal:%d/metrics", port),
			fmt.Sprintf("http://172.17.0.1:%d/metrics", port),
			fmt.Sprintf("http://localhost:%d/metrics", port),
		}
	}
	return []string{
		fmt.Sprintf("http://localhost:%d/metrics", port),
	}
}

// isRunningInDocker checks if the current process is running inside a Docker container.
func (p *Provider) isRunningInDocker() bool {
	// Check for /.dockerenv file (Docker sets this)
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	
	// Check /proc/self/cgroup for docker (Linux)
	if _, err := os.ReadFile("/proc/self/cgroup"); err == nil {
		cgroup, err := os.ReadFile("/proc/self/cgroup")
		if err == nil && strings.Contains(string(cgroup), "docker") {
			return true
		}
	}
	
	return false
}

// getNetworkGatewayIP gets the default gateway IP for the Docker network.
// Reads from /proc/net/route to find the gateway IP.
func (p *Provider) getNetworkGatewayIP() string {
	// Read /proc/net/route to find default gateway
	routeFile := "/proc/net/route"
	data, err := os.ReadFile(routeFile)
	if err != nil {
		return ""
	}
	
	// Parse route file to find default gateway (destination 00000000)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Iface") {
			continue // Skip header
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			// Check if this is the default route (destination is 00000000)
			if fields[1] == "00000000" && fields[2] != "00000000" {
				// Gateway is in hex format, convert to IP
				gatewayHex := fields[2]
				if len(gatewayHex) == 8 {
					// Convert hex to IP (little-endian)
					ip := p.hexToIP(gatewayHex)
					if ip != "" {
						return ip
					}
				}
			}
		}
	}
	return ""
}

// hexToIP converts a hex string (little-endian) to IP address string.
func (p *Provider) hexToIP(hexStr string) string {
	if len(hexStr) != 8 {
		return ""
	}
	// Parse hex bytes in reverse order (little-endian)
	var bytes [4]int
	for i := 0; i < 4; i++ {
		byteStr := hexStr[i*2:(i+1)*2]
		val, err := strconv.ParseInt(byteStr, 16, 64)
		if err != nil {
			return ""
		}
		bytes[3-i] = int(val) // Reverse for little-endian
	}
	return fmt.Sprintf("%d.%d.%d.%d", bytes[0], bytes[1], bytes[2], bytes[3])
}

// GetQuickTunnelContainerConfig returns the Docker container configuration for a Quick Tunnel
// (temporary trycloudflare.com URL). No tunnel token; cloudflared runs with --url and exposes metrics.
// metricsHostPort is the host port for the metrics endpoint (container port is always 2000); use a unique port per app to avoid conflicts.
// DEPRECATED: Use CreateQuickTunnelConfig instead (implements QuickTunnelProvider interface).
func (p *Provider) GetQuickTunnelContainerConfig(targetService string, targetPort int, metricsHostPort int) *tunnel.ContainerConfig {
	return QuickTunnelContainerConfig(targetService, targetPort, metricsHostPort)
}

// QuickTunnelContainerConfig returns the Docker container configuration for a Quick Tunnel
// without needing a Provider instance (no API credentials required). Use when tunnel_mode is "quick".
// metricsHostPort is the host port for the metrics endpoint (container port is always 2000); use a unique port per app to avoid conflicts.
func QuickTunnelContainerConfig(targetService string, targetPort int, metricsHostPort int) *tunnel.ContainerConfig {
	targetURL := fmt.Sprintf("http://%s:%d", targetService, targetPort)
	if metricsHostPort < constants.MinPort || metricsHostPort > constants.MaxPort {
		metricsHostPort = constants.QuickTunnelMetricsPort
	}
	metricsEndpoint := fmt.Sprintf(constants.QuickTunnelMetricsEndpointFormat, constants.QuickTunnelMetricsPort)
	return &tunnel.ContainerConfig{
		Image:   "cloudflare/cloudflared:latest",
		Command: []string{"tunnel", "--url", targetURL, "--metrics", metricsEndpoint},
		Ports:   []string{fmt.Sprintf("%d:%d", metricsHostPort, constants.QuickTunnelMetricsPort)},
	}
}

// ============================================================================
// ListProvider Interface
// ============================================================================

// ListTunnels returns all Cloudflare tunnels.
func (p *Provider) ListTunnels(ctx context.Context, nodeIDs []string) ([]*tunnel.Tunnel, error) {
	cfTunnels, err := p.database.ListActiveCloudflareTunnels()
	if err != nil {
		return nil, fmt.Errorf("failed to list tunnels: %w", err)
	}

	// Convert to generic tunnel format
	genericTunnels := make([]*tunnel.Tunnel, 0, len(cfTunnels))
	for _, cfTunnel := range cfTunnels {
		// Get public URL from app if available
		app, err := p.database.GetApp(cfTunnel.AppID)
		publicURL := ""
		if err == nil {
			publicURL = app.PublicURL
		}

		genericTunnels = append(genericTunnels, p.toGenericTunnel(cfTunnel, publicURL))
	}

	return genericTunnels, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// toGenericTunnel converts a Cloudflare-specific tunnel to the generic tunnel type.
func (p *Provider) toGenericTunnel(cfTunnel *db.CloudflareTunnel, publicURL string) *tunnel.Tunnel {
	if publicURL == "" && cfTunnel.TunnelID != "" {
		// Generate public URL from tunnel ID if not provided
		publicURL = fmt.Sprintf("https://%s.cfargotunnel.com", cfTunnel.TunnelID)
	}

	return &tunnel.Tunnel{
		ID:           cfTunnel.ID,
		AppID:        cfTunnel.AppID,
		ProviderType: p.Name(),
		TunnelID:     cfTunnel.TunnelID,
		TunnelName:   cfTunnel.TunnelName,
		TunnelToken:  cfTunnel.TunnelToken,
		PublicURL:    publicURL,
		Status:       cfTunnel.Status,
		IsActive:     cfTunnel.IsActive,
		IngressRules: cfTunnel.IngressRules,
		Metadata: map[string]interface{}{
			"account_id": cfTunnel.AccountID,
		},
		ErrorDetails: func() string {
			if cfTunnel.ErrorDetails != nil {
				return *cfTunnel.ErrorDetails
			}
			return ""
		}(),
		LastSyncedAt: cfTunnel.LastSyncedAt,
		CreatedAt:    cfTunnel.CreatedAt,
		UpdatedAt:    cfTunnel.UpdatedAt,
	}
}
