package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/selfhostly/internal/db"
	"github.com/selfhostly/internal/docker"
	"github.com/selfhostly/internal/domain"
)

// TunnelCreateHandler handles tunnel_create jobs
// Handles both creating a new custom tunnel and switching from Quick Tunnel to custom tunnel
type TunnelCreateHandler struct {
	db            *db.DB
	dockerManager *docker.Manager
	appService    domain.AppService
	tunnelService domain.TunnelService
	logger        *slog.Logger
}

// NewTunnelCreateHandler creates a new tunnel create handler
func NewTunnelCreateHandler(
	database *db.DB,
	dockerMgr *docker.Manager,
	appSvc domain.AppService,
	tunnelSvc domain.TunnelService,
	logger *slog.Logger,
) *TunnelCreateHandler {
	return &TunnelCreateHandler{
		db:            database,
		dockerManager: dockerMgr,
		appService:    appSvc,
		tunnelService: tunnelSvc,
		logger:        logger,
	}
}

// Handle processes a tunnel_create job
func (h *TunnelCreateHandler) Handle(ctx context.Context, job *db.Job, progress *ProgressTracker) error {
	// Parse payload for ingress rules
	var payload TunnelCreatePayload
	if job.Payload != nil {
		if err := json.Unmarshal([]byte(*job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to parse tunnel_create payload: %w", err)
		}
	}

	progress.Update(10, "Getting app details...")

	// Get app
	app, err := h.db.GetApp(job.AppID)
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	progress.Update(20, "Creating tunnel with provider...")

	// Delegate to app service which has access to provider registry
	// The sync method CreateTunnelForApp does all the heavy lifting
	// We call it here in the background job context
	// Note: We pass app.NodeID since we're on the same node
	updatedApp, _, err := h.appService.CreateTunnelForApp(ctx, app.ID, app.NodeID, nil)
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}

	// If ingress rules were provided, apply them now
	if len(payload.IngressRules) > 0 {
		progress.Update(80, "Applying ingress rules...")

		// Convert payload ingress rules to domain ingress rules
		domainRules := make([]db.IngressRule, len(payload.IngressRules))
		for i, rule := range payload.IngressRules {
			domainRules[i] = db.IngressRule{
				Hostname:      rule.Hostname,
				Service:       rule.Service,
				Path:          rule.Path,
				OriginRequest: rule.OriginRequest,
			}
		}

		ingressReq := domain.UpdateIngressRequest{IngressRules: domainRules}
		if err := h.tunnelService.UpdateTunnelIngress(ctx, app.ID, app.NodeID, ingressReq); err != nil {
			// Don't fail the entire job if ingress update fails - tunnel is still created
			h.logger.Warn("failed to apply ingress rules, but tunnel created successfully", "app_id", app.ID, "error", err)
		} else {
			// Restart tunnel container to ensure ingress rules are picked up
			if err := h.dockerManager.RestartTunnelService(app.Name); err != nil {
				h.logger.Warn("failed to restart tunnel after ingress update", "app", app.Name, "error", err)
			}

			// Fetch updated app to get new public_url after ingress rules applied
			updatedApp, err = h.db.GetApp(job.AppID)
			if err != nil {
				h.logger.Warn("failed to fetch updated app after ingress", "app_id", app.ID, "error", err)
			}
		}
	}

	progress.Update(90, "Tunnel created successfully")

	// Log success
	h.logger.Info("tunnel created via background job", "app_id", updatedApp.ID, "tunnel_id", updatedApp.TunnelID, "has_ingress", len(payload.IngressRules) > 0)

	progress.Update(100, "Complete")
	return nil
}
