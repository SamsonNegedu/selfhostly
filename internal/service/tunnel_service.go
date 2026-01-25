package service

import (
	"context"
	"log/slog"

	"github.com/selfhostly/internal/cloudflare"
	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/domain"
)

// tunnelService implements the TunnelService interface
type tunnelService struct {
	database *db.DB
	logger   *slog.Logger
}

// NewTunnelService creates a new tunnel service
func NewTunnelService(database *db.DB, logger *slog.Logger) domain.TunnelService {
	return &tunnelService{
		database: database,
		logger:   logger,
	}
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

// ListActiveTunnels retrieves all active tunnels
func (s *tunnelService) ListActiveTunnels(ctx context.Context) ([]*db.CloudflareTunnel, error) {
	s.logger.DebugContext(ctx, "listing active tunnels")

	tunnels, err := s.database.ListActiveCloudflareTunnels()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list tunnels", "error", err)
		return nil, domain.WrapDatabaseOperation("list tunnels", err)
	}

	return tunnels, nil
}

// SyncTunnelStatus synchronizes tunnel status with Cloudflare
func (s *tunnelService) SyncTunnelStatus(ctx context.Context, appID string) error {
	s.logger.InfoContext(ctx, "syncing tunnel status", "appID", appID)

	tunnel, err := s.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		return domain.NewDomainError("TUNNEL_NOT_FOUND", "tunnel not found", err)
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		return domain.WrapDatabaseOperation("get settings", err)
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil {
		return domain.ErrTunnelNotConfigured
	}

	tunnelManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)
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

	settings, err := s.database.GetSettings()
	if err != nil {
		return domain.WrapDatabaseOperation("get settings", err)
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil {
		return domain.ErrTunnelNotConfigured
	}

	// Convert domain IngressRules to cloudflare IngressRules
	cfRules := make([]cloudflare.IngressRule, len(req.IngressRules))
	for i, rule := range req.IngressRules {
		cfRule := cloudflare.IngressRule{
			Service:       rule.Service,
			OriginRequest: rule.OriginRequest,
		}
		if rule.Hostname != nil {
			cfRule.Hostname = *rule.Hostname
		}
		if rule.Path != nil {
			cfRule.Path = *rule.Path
		}
		cfRules[i] = cfRule
	}

	tunnelManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)
	if err := tunnelManager.UpdateTunnelIngress(tunnel.TunnelID, cfRules, req.Hostname, req.TargetDomain); err != nil {
		s.logger.ErrorContext(ctx, "failed to update ingress", "tunnelID", tunnel.TunnelID, "error", err)
		return err
	}

	return nil
}

// CreateDNSRecord creates a DNS record for a tunnel
func (s *tunnelService) CreateDNSRecord(ctx context.Context, appID string, req domain.CreateDNSRequest) error {
	s.logger.InfoContext(ctx, "creating DNS record", "appID", appID, "hostname", req.Hostname)

	tunnel, err := s.database.GetCloudflareTunnelByAppID(appID)
	if err != nil {
		return domain.NewDomainError("TUNNEL_NOT_FOUND", "tunnel not found", err)
	}

	settings, err := s.database.GetSettings()
	if err != nil {
		return domain.WrapDatabaseOperation("get settings", err)
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil {
		return domain.ErrTunnelNotConfigured
	}

	tunnelManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)
	
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

	settings, err := s.database.GetSettings()
	if err != nil {
		return domain.WrapDatabaseOperation("get settings", err)
	}

	if settings.CloudflareAPIToken == nil || settings.CloudflareAccountID == nil {
		return domain.ErrTunnelNotConfigured
	}

	tunnelManager := cloudflare.NewTunnelManager(*settings.CloudflareAPIToken, *settings.CloudflareAccountID, s.database)
	if err := tunnelManager.DeleteTunnelByAppID(appID); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete tunnel", "appID", appID, "error", err)
		return err
	}

	return nil
}
