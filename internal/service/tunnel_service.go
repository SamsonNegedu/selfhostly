package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/selfhostly/internal/cloudflare"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/node"
	"github.com/selfhostly/internal/routing"
	"github.com/selfhostly/internal/tunnel"
	cloudflareProvider "github.com/selfhostly/internal/tunnel/providers/cloudflare"
)

// tunnelService implements the TunnelService interface
type tunnelService struct {
	database         *db.DB
	nodeClient       *node.Client
	config           *config.Config
	logger           *slog.Logger
	tunnelManager    *cloudflare.TunnelManager // DEPRECATED: kept for backward compatibility
	providerRegistry *tunnel.Registry          // NEW: provider registry for multi-provider support
	router           *routing.NodeRouter
	tunnelsAgg       *routing.TunnelsAggregator
}

// NewTunnelService creates a new tunnel service with provider registry
func NewTunnelService(database *db.DB, cfg *config.Config, logger *slog.Logger) domain.TunnelService {
	nodeClient := node.NewClient()
	router := routing.NewNodeRouter(database, nodeClient, cfg.Node.ID, logger)
	tunnelsAgg := routing.NewTunnelsAggregator(router, logger)

	// Initialize provider registry
	registry := tunnel.NewRegistry()

	// Register Cloudflare provider
	registry.Register("cloudflare", func(config map[string]interface{}) (tunnel.Provider, error) {
		// Inject database and logger into config
		config["database"] = database
		config["logger"] = logger
		return cloudflareProvider.NewProvider(config)
	})

	// Future providers can be registered here:
	// registry.Register("ngrok", ngrokProviderFactory)
	// registry.Register("tailscale", tailscaleProviderFactory)

	return &tunnelService{
		database:         database,
		nodeClient:       nodeClient,
		config:           cfg,
		logger:           logger,
		providerRegistry: registry,
		router:           router,
		tunnelsAgg:       tunnelsAgg,
	}
}

// NewTunnelServiceWithManager creates a new tunnel service with a custom tunnel manager (for testing)
// DEPRECATED: Use NewTunnelService with provider registry instead
func NewTunnelServiceWithManager(database *db.DB, cfg *config.Config, logger *slog.Logger, tunnelManager *cloudflare.TunnelManager) domain.TunnelService {
	nodeClient := node.NewClient()
	router := routing.NewNodeRouter(database, nodeClient, cfg.Node.ID, logger)
	tunnelsAgg := routing.NewTunnelsAggregator(router, logger)

	return &tunnelService{
		database:      database,
		nodeClient:    nodeClient,
		config:        cfg,
		logger:        logger,
		tunnelManager: tunnelManager, // For backward compatibility with tests
		router:        router,
		tunnelsAgg:    tunnelsAgg,
	}
}

// getActiveProvider returns the active tunnel provider instance.
// This method handles backward compatibility with old cloudflare-specific settings
// and test setups that inject a tunnelManager directly.
func (s *tunnelService) getActiveProvider() (tunnel.Provider, error) {
	// BACKWARD COMPATIBILITY: Check if old tunnelManager was injected (for tests)
	if s.tunnelManager != nil {
		// Wrap the old tunnel manager in a provider adapter
		return newCloudflareProviderFromManager(s.tunnelManager, s.database, s.logger), nil
	}

	// Normal path: use provider registry
	if s.providerRegistry == nil {
		return nil, fmt.Errorf("provider registry not initialized")
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// Get active provider name (with backward compatibility fallback)
	providerName := settings.GetActiveProviderName()

	// Get provider configuration
	providerConfig, err := settings.GetProviderConfig(providerName)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", tunnel.ErrProviderNotConfigured, err)
	}

	// Get provider from registry
	provider, err := s.providerRegistry.GetProvider(providerName, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", providerName, err)
	}

	return provider, nil
}

// cloudflareManagerAdapter wraps an old cloudflare.TunnelManager to implement the tunnel.Provider interface
// This provides backward compatibility for tests that inject a mock tunnel manager.
type cloudflareManagerAdapter struct {
	manager  *cloudflare.TunnelManager
	database *db.DB
	logger   *slog.Logger
}

func newCloudflareProviderFromManager(manager *cloudflare.TunnelManager, database *db.DB, logger *slog.Logger) tunnel.Provider {
	return &cloudflareManagerAdapter{
		manager:  manager,
		database: database,
		logger:   logger,
	}
}

// Core Provider interface implementation
func (a *cloudflareManagerAdapter) CreateTunnel(ctx context.Context, opts tunnel.CreateOptions) (*tunnel.Tunnel, error) {
	tunnelID, token, err := a.manager.ApiManager.CreateTunnel(opts.Name)
	if err != nil {
		return nil, err
	}

	// Get account ID from settings
	settings, err := a.database.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	accountID := ""
	if settings.CloudflareAccountID != nil {
		accountID = *settings.CloudflareAccountID
	}

	cfTunnel := db.NewCloudflareTunnel(opts.AppID, tunnelID, opts.Name, token, accountID, "")
	if err := a.database.CreateCloudflareTunnel(cfTunnel); err != nil {
		return nil, err
	}

	return a.toGenericTunnel(cfTunnel), nil
}

func (a *cloudflareManagerAdapter) GetTunnelByAppID(ctx context.Context, appID string) (*tunnel.Tunnel, error) {
	cfTunnel, err := a.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, tunnel.ErrTunnelNotFound
		}
		return nil, err
	}
	return a.toGenericTunnel(cfTunnel), nil
}

func (a *cloudflareManagerAdapter) DeleteTunnel(ctx context.Context, appID string) error {
	return a.manager.DeleteTunnelByAppID(appID)
}

func (a *cloudflareManagerAdapter) CleanupOrphanedTunnels(ctx context.Context) error {
	return nil // Not implemented for adapter
}

func (a *cloudflareManagerAdapter) Name() string {
	return "cloudflare"
}

func (a *cloudflareManagerAdapter) DisplayName() string {
	return "Cloudflare Tunnel"
}

// IngressProvider interface
func (a *cloudflareManagerAdapter) UpdateIngress(ctx context.Context, appID string, rules interface{}) error {
	cfRules := rules.([]db.IngressRule)
	cfTunnel, err := a.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return tunnel.ErrTunnelNotFound
		}
		return err
	}

	convertedRules := cloudflare.ConvertToCloudflareRules(cfRules)
	if err := a.manager.UpdateTunnelIngress(cfTunnel.TunnelID, convertedRules, "", ""); err != nil {
		return err
	}

	cfTunnel.IngressRules = &cfRules
	return a.database.UpdateCloudflareTunnel(cfTunnel)
}

// DNSProvider interface
func (a *cloudflareManagerAdapter) CreateDNSRecord(ctx context.Context, appID string, opts tunnel.DNSOptions) error {
	cfTunnel, err := a.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return tunnel.ErrTunnelNotFound
		}
		return err
	}

	zoneID, err := a.manager.ApiManager.GetZoneID(opts.Domain)
	if err != nil {
		return err
	}

	_, err = a.manager.ApiManager.CreateDNSRecord(zoneID, opts.Hostname, cfTunnel.TunnelID)
	return err
}

// StatusSyncProvider interface
func (a *cloudflareManagerAdapter) SyncStatus(ctx context.Context, appID string) error {
	cfTunnel, err := a.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			return tunnel.ErrTunnelNotFound
		}
		return err
	}
	return a.manager.SyncTunnelStatus(cfTunnel.TunnelID)
}

// ContainerProvider interface
func (a *cloudflareManagerAdapter) GetContainerConfig(tunnelToken string, appName string) *tunnel.ContainerConfig {
	return &tunnel.ContainerConfig{
		Image:   "cloudflare/cloudflared:latest",
		Command: []string{"tunnel", "run"},
		Environment: map[string]string{
			"TUNNEL_TOKEN": tunnelToken,
		},
	}
}

// Helper
func (a *cloudflareManagerAdapter) toGenericTunnel(cfTunnel *db.CloudflareTunnel) *tunnel.Tunnel {
	return &tunnel.Tunnel{
		ID:           cfTunnel.ID,
		AppID:        cfTunnel.AppID,
		ProviderType: "cloudflare",
		TunnelID:     cfTunnel.TunnelID,
		TunnelName:   cfTunnel.TunnelName,
		TunnelToken:  cfTunnel.TunnelToken,
		PublicURL:    cfTunnel.PublicURL,
		Status:       cfTunnel.Status,
		IsActive:     cfTunnel.IsActive,
		IngressRules: cfTunnel.IngressRules,
		CreatedAt:    cfTunnel.CreatedAt,
		UpdatedAt:    cfTunnel.UpdatedAt,
	}
}

// GetTunnelByAppID retrieves a tunnel by app ID using the active provider
func (s *tunnelService) GetTunnelByAppID(ctx context.Context, appID string, nodeID string) (*db.CloudflareTunnel, error) {
	s.logger.DebugContext(ctx, "getting tunnel by app ID", "appID", appID, "nodeID", nodeID)

	result, err := s.router.RouteToNode(
		ctx,
		nodeID,
		func() (interface{}, error) {
			// Try to get the active provider
			provider, err := s.getActiveProvider()
			if err != nil {
				// Fallback to direct database query for backward compatibility
				s.logger.WarnContext(ctx, "failed to get provider, falling back to direct query", "error", err)
				tunnel, err := s.database.GetCloudflareTunnelByAppID(appID)
				if err != nil {
					return nil, domain.NewDomainError("TUNNEL_NOT_FOUND", "tunnel not found", err)
				}
				return tunnel, nil
			}

			// Use provider to get tunnel
			genericTunnel, err := provider.GetTunnelByAppID(ctx, appID)
			if err != nil {
				if err == tunnel.ErrTunnelNotFound {
					return nil, domain.ErrTunnelNotFound
				}
				return nil, fmt.Errorf("failed to get tunnel: %w", err)
			}

			// Convert generic tunnel back to CloudflareTunnel for backward compatibility
			// This is a temporary measure until we update all consumers to use generic Tunnel type
			cfTunnel, err := s.database.GetCloudflareTunnelByAppID(appID)
			if err != nil {
				return nil, domain.ErrTunnelNotFound
			}

			// Update with provider data if available
			if genericTunnel.PublicURL != "" {
				cfTunnel.Status = genericTunnel.Status
			}

			return cfTunnel, nil
		},
		func(n *db.Node) (interface{}, error) {
			return s.nodeClient.GetTunnelByAppID(n, appID)
		},
	)

	if err != nil {
		return nil, err
	}

	return result.(*db.CloudflareTunnel), nil
}

// ListActiveTunnels retrieves all active tunnels from specified nodes.
// Always uses the aggregator so that node_ids are respected (local DB + remote node client);
// the provider's ListTunnels only queries the local DB and would ignore remote nodes.
func (s *tunnelService) ListActiveTunnels(ctx context.Context, nodeIDs []string) ([]*db.CloudflareTunnel, error) {
	s.logger.DebugContext(ctx, "listing active tunnels", "nodeIDs", nodeIDs)

	targetNodes, err := s.router.DetermineTargetNodes(ctx, nodeIDs)
	if err != nil {
		return nil, err
	}

	allTunnels, err := s.tunnelsAgg.AggregateTunnels(
		ctx,
		targetNodes,
		func() ([]*db.CloudflareTunnel, error) {
			return s.database.ListActiveCloudflareTunnels()
		},
		func(n *db.Node) ([]*db.CloudflareTunnel, error) {
			return s.nodeClient.GetTunnels(n)
		},
	)

	return allTunnels, err
}

// SyncTunnelStatus synchronizes tunnel status with the provider (if supported)
func (s *tunnelService) SyncTunnelStatus(ctx context.Context, appID string, nodeID string) error {
	s.logger.InfoContext(ctx, "syncing tunnel status", "appID", appID, "nodeID", nodeID)

	result, err := s.router.RouteToNode(
		ctx,
		nodeID,
		func() (interface{}, error) {
			provider, err := s.getActiveProvider()
			if err != nil {
				return nil, fmt.Errorf("failed to get provider: %w", err)
			}

			// Check if provider supports status sync
			syncProvider, ok := provider.(tunnel.StatusSyncProvider)
			if !ok {
				// Provider doesn't support status sync - this is not an error, just log it
				s.logger.DebugContext(ctx, "provider does not support status sync", "provider", provider.Name())
				return nil, nil
			}

			if err := syncProvider.SyncStatus(ctx, appID); err != nil {
				return nil, fmt.Errorf("failed to sync tunnel status: %w", err)
			}

			return nil, nil
		},
		func(n *db.Node) (interface{}, error) {
			return nil, s.nodeClient.SyncTunnelStatus(n, appID)
		},
	)

	if err != nil {
		return err
	}

	// Result should be nil on success
	_ = result
	return nil
}

// UpdateTunnelIngress updates the ingress configuration for a tunnel (if supported)
func (s *tunnelService) UpdateTunnelIngress(ctx context.Context, appID string, nodeID string, req domain.UpdateIngressRequest) error {
	s.logger.InfoContext(ctx, "updating tunnel ingress", "appID", appID, "nodeID", nodeID)

	result, err := s.router.RouteToNode(
		ctx,
		nodeID,
		func() (interface{}, error) {
			provider, err := s.getActiveProvider()
			if err != nil {
				return nil, fmt.Errorf("failed to get provider: %w", err)
			}

			// Check if provider supports ingress configuration
			ingressProvider, ok := provider.(tunnel.IngressProvider)
			if !ok {
				return nil, tunnel.NewFeatureNotSupportedError(provider.DisplayName(), tunnel.FeatureIngress)
			}

			// Pass the ingress rules to the provider (provider handles conversion to its format)
			if err := ingressProvider.UpdateIngress(ctx, appID, req.IngressRules); err != nil {
				return nil, fmt.Errorf("failed to update ingress: %w", err)
			}

			s.logger.InfoContext(ctx, "tunnel ingress updated successfully", "appID", appID)
			return nil, nil
		},
		func(n *db.Node) (interface{}, error) {
			return nil, s.nodeClient.UpdateTunnelIngress(n, appID, req)
		},
	)

	if err != nil {
		return err
	}

	// Result should be nil on success
	_ = result
	return nil
}

// CreateDNSRecord creates a DNS record for a tunnel (if supported)
func (s *tunnelService) CreateDNSRecord(ctx context.Context, appID string, nodeID string, req domain.CreateDNSRequest) error {
	s.logger.InfoContext(ctx, "creating DNS record", "appID", appID, "hostname", req.Hostname, "nodeID", nodeID)

	result, err := s.router.RouteToNode(
		ctx,
		nodeID,
		func() (interface{}, error) {
			provider, err := s.getActiveProvider()
			if err != nil {
				return nil, fmt.Errorf("failed to get provider: %w", err)
			}

			// Check if provider supports DNS management
			dnsProvider, ok := provider.(tunnel.DNSProvider)
			if !ok {
				return nil, tunnel.NewFeatureNotSupportedError(provider.DisplayName(), tunnel.FeatureDNS)
			}

			// Create DNS record via provider
			opts := tunnel.DNSOptions{
				Hostname: req.Hostname,
				Domain:   req.Domain,
			}

			if err := dnsProvider.CreateDNSRecord(ctx, appID, opts); err != nil {
				return nil, fmt.Errorf("failed to create DNS record: %w", err)
			}

			s.logger.InfoContext(ctx, "DNS record created successfully", "hostname", req.Hostname)
			return nil, nil
		},
		func(n *db.Node) (interface{}, error) {
			return nil, s.nodeClient.CreateTunnelDNSRecord(n, appID, req)
		},
	)

	if err != nil {
		return err
	}

	// Result should be nil on success
	_ = result
	return nil
}

// DeleteTunnel deletes a tunnel
func (s *tunnelService) DeleteTunnel(ctx context.Context, appID string, nodeID string) error {
	s.logger.InfoContext(ctx, "deleting tunnel", "appID", appID, "nodeID", nodeID)

	result, err := s.router.RouteToNode(
		ctx,
		nodeID,
		func() (interface{}, error) {
			provider, err := s.getActiveProvider()
			if err != nil {
				return nil, fmt.Errorf("failed to get provider: %w", err)
			}

			if err := provider.DeleteTunnel(ctx, appID); err != nil {
				return nil, fmt.Errorf("failed to delete tunnel: %w", err)
			}

			s.logger.InfoContext(ctx, "tunnel deleted successfully", "appID", appID)
			return nil, nil
		},
		func(n *db.Node) (interface{}, error) {
			return nil, s.nodeClient.DeleteTunnel(n, appID)
		},
	)

	if err != nil {
		return err
	}

	// Result should be nil on success
	_ = result
	return nil
}

// ListProviders returns information about all available tunnel providers
func (s *tunnelService) ListProviders(ctx context.Context) ([]domain.ProviderInfo, error) {
	s.logger.DebugContext(ctx, "listing available tunnel providers")

	if s.providerRegistry == nil {
		// Fallback for tests or old setup: return just cloudflare
		settings, err := s.database.GetSettings()
		if err != nil {
			return nil, fmt.Errorf("failed to get settings: %w", err)
		}

		isConfigured := settings.CloudflareAPIToken != nil && *settings.CloudflareAPIToken != ""
		return []domain.ProviderInfo{
			{
				Name:         "cloudflare",
				DisplayName:  "Cloudflare Tunnel",
				IsConfigured: isConfigured,
			},
		}, nil
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	providerNames := s.providerRegistry.ListProviders()
	providers := make([]domain.ProviderInfo, 0, len(providerNames))

	for _, name := range providerNames {
		// Check if provider is configured
		providerConfig, err := settings.GetProviderConfig(name)
		isConfigured := err == nil && providerConfig != nil

		displayName := name
		if isConfigured {
			// Try to get display name from provider
			provider, err := s.providerRegistry.GetProvider(name, providerConfig)
			if err == nil {
				displayName = provider.DisplayName()
			}
		}

		providers = append(providers, domain.ProviderInfo{
			Name:         name,
			DisplayName:  displayName,
			IsConfigured: isConfigured,
		})
	}

	return providers, nil
}

// GetProviderFeatures returns the features supported by a specific provider
func (s *tunnelService) GetProviderFeatures(ctx context.Context, providerName string) (*domain.ProviderFeatures, error) {
	s.logger.DebugContext(ctx, "getting provider features", "provider", providerName)

	if s.providerRegistry == nil {
		return nil, fmt.Errorf("provider registry not initialized")
	}

	// Check if provider is registered
	if !s.providerRegistry.IsRegistered(providerName) {
		return nil, fmt.Errorf("provider %s not registered", providerName)
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	providerConfig, err := settings.GetProviderConfig(providerName)
	if err != nil {
		// Provider not configured
		return &domain.ProviderFeatures{
			Provider:     providerName,
			DisplayName:  providerName,
			IsConfigured: false,
			Features:     make(map[string]bool),
		}, nil
	}

	// Create provider instance
	provider, err := s.providerRegistry.GetProvider(providerName, providerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Get supported features
	features := tunnel.GetSupportedFeatures(provider)

	// Convert Feature type to string for domain layer
	featuresMap := map[string]bool{
		"ingress":     features[tunnel.FeatureIngress],
		"dns":         features[tunnel.FeatureDNS],
		"status_sync": features[tunnel.FeatureStatusSync],
		"container":   features[tunnel.FeatureContainer],
		"list":        features[tunnel.FeatureList],
	}

	return &domain.ProviderFeatures{
		Provider:     providerName,
		DisplayName:  provider.DisplayName(),
		IsConfigured: true,
		Features:     featuresMap,
	}, nil
}
