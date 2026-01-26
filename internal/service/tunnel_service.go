package service

import (
	"context"
	"log/slog"

	"github.com/selfhostly/internal/cloudflare"
	"github.com/selfhostly/internal/config"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
	"github.com/selfhostly/internal/node"
	"github.com/selfhostly/internal/routing"
)

// tunnelService implements the TunnelService interface
type tunnelService struct {
	database      *db.DB
	nodeClient    *node.Client
	config        *config.Config
	logger        *slog.Logger
	tunnelManager *cloudflare.TunnelManager // Optional, for dependency injection in tests
	router        *routing.NodeRouter
	tunnelsAgg    *routing.TunnelsAggregator
}

// NewTunnelService creates a new tunnel service
func NewTunnelService(database *db.DB, cfg *config.Config, logger *slog.Logger) domain.TunnelService {
	nodeClient := node.NewClient()
	router := routing.NewNodeRouter(database, nodeClient, cfg.Node.ID, logger)
	tunnelsAgg := routing.NewTunnelsAggregator(router, logger)
	
	return &tunnelService{
		database:   database,
		nodeClient: nodeClient,
		config:     cfg,
		logger:     logger,
		router:     router,
		tunnelsAgg: tunnelsAgg,
	}
}

// NewTunnelServiceWithManager creates a new tunnel service with a custom tunnel manager (for testing)
func NewTunnelServiceWithManager(database *db.DB, cfg *config.Config, logger *slog.Logger, tunnelManager *cloudflare.TunnelManager) domain.TunnelService {
	nodeClient := node.NewClient()
	router := routing.NewNodeRouter(database, nodeClient, cfg.Node.ID, logger)
	tunnelsAgg := routing.NewTunnelsAggregator(router, logger)
	
	return &tunnelService{
		database:      database,
		nodeClient:    nodeClient,
		config:        cfg,
		logger:        logger,
		tunnelManager: tunnelManager,
		router:        router,
		tunnelsAgg:    tunnelsAgg,
	}
}

// getTunnelManager returns the tunnel manager, creating one if not provided
func (s *tunnelService) getTunnelManager() (*cloudflare.TunnelManager, error) {
	if s.tunnelManager != nil {
		return s.tunnelManager, nil
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		return nil, err
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil {
		return nil, domain.ErrTunnelNotConfigured
	}

	return cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database), nil
}

// GetTunnelByAppID retrieves a tunnel by app ID
func (s *tunnelService) GetTunnelByAppID(ctx context.Context, appID string) (*db.CloudflareTunnel, error) {
	s.logger.DebugContext(ctx, "getting tunnel by app ID", "appID", appID)

	tunnel, err := s.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		s.logger.DebugContext(ctx, "tunnel not found", "appID", appID)
		return nil, domain.NewDomainError("TUNNEL_NOT_FOUND", "tunnel not found", err)
	}

	return tunnel, nil
}

// ListActiveTunnels retrieves all active tunnels from specified nodes
func (s *tunnelService) ListActiveTunnels(ctx context.Context, nodeIDs []string) ([]*db.CloudflareTunnel, error) {
	s.logger.DebugContext(ctx, "listing active tunnels", "nodeIDs", nodeIDs)

	// Determine which nodes to fetch from
	targetNodes, err := s.router.DetermineTargetNodes(ctx, nodeIDs)
	if err != nil {
		return nil, err
	}

	// Aggregate tunnels from all target nodes in parallel
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

// SyncTunnelStatus synchronizes tunnel status with Cloudflare
func (s *tunnelService) SyncTunnelStatus(ctx context.Context, appID string) error {
	s.logger.InfoContext(ctx, "syncing tunnel status", "appID", appID)

	tunnel, err := s.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		return domain.NewDomainError("TUNNEL_NOT_FOUND", "tunnel not found", err)
	}

	tunnelManager, err := s.getTunnelManager()
	if err != nil {
		return err
	}

	if err := tunnelManager.SyncTunnelStatus(tunnel.TunnelID); err != nil {
		s.logger.ErrorContext(ctx, "failed to sync tunnel status", "tunnelID", tunnel.TunnelID, "error", err)
		return err
	}

	return nil
}

// UpdateTunnelIngress updates the ingress configuration for a tunnel
func (s *tunnelService) UpdateTunnelIngress(ctx context.Context, appID string, req domain.UpdateIngressRequest) error {
	s.logger.InfoContext(ctx, "updating tunnel ingress", "appID", appID)

	tunnel, err := s.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		return domain.NewDomainError("TUNNEL_NOT_FOUND", "tunnel not found", err)
	}

	tunnelManager, err := s.getTunnelManager()
	if err != nil {
		return err
	}

	// Convert domain IngressRules to cloudflare IngressRules
	cfRules := cloudflare.ConvertToCloudflareRules(req.IngressRules)

	if err := tunnelManager.UpdateTunnelIngress(tunnel.TunnelID, cfRules, req.Hostname, req.TargetDomain); err != nil {
		s.logger.ErrorContext(ctx, "failed to update ingress", "tunnelID", tunnel.TunnelID, "error", err)
		return err
	}

	// Update the database record with the new ingress rules
	tunnel.IngressRules = &req.IngressRules
	if err := s.database.UpdateCloudflareTunnel(tunnel); err != nil {
		s.logger.ErrorContext(ctx, "failed to update tunnel in database", "tunnelID", tunnel.TunnelID, "error", err)
		// Don't fail the request if database update fails - Cloudflare API update succeeded
		// but log the error for debugging
	}

	s.logger.InfoContext(ctx, "tunnel ingress updated successfully", "tunnelID", tunnel.TunnelID, "appID", appID)
	return nil
}

// CreateDNSRecord creates a DNS record for a tunnel
func (s *tunnelService) CreateDNSRecord(ctx context.Context, appID string, req domain.CreateDNSRequest) error {
	s.logger.InfoContext(ctx, "creating DNS record", "appID", appID, "hostname", req.Hostname)

	tunnel, err := s.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		return domain.NewDomainError("TUNNEL_NOT_FOUND", "tunnel not found", err)
	}

	tunnelManager, err := s.getTunnelManager()
	if err != nil {
		return err
	}

	// Get zone ID for the domain
	zoneID, err := tunnelManager.ApiManager.GetZoneID(req.Domain)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get zone ID", "domain", req.Domain, "error", err)
		return err
	}

	// Create DNS record
	_, err = tunnelManager.ApiManager.CreateDNSRecord(zoneID, req.Hostname, tunnel.TunnelID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create DNS record", "hostname", req.Hostname, "error", err)
		return err
	}

	s.logger.InfoContext(ctx, "DNS record created successfully", "hostname", req.Hostname)
	return nil
}

// DeleteTunnel deletes a tunnel
func (s *tunnelService) DeleteTunnel(ctx context.Context, appID string) error {
	s.logger.InfoContext(ctx, "deleting tunnel", "appID", appID)

	tunnelManager, err := s.getTunnelManager()
	if err != nil {
		return err
	}

	if err := tunnelManager.DeleteTunnelByAppID(appID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete tunnel", "appID", appID, "error", err)
		return err
	}

	return nil
}
